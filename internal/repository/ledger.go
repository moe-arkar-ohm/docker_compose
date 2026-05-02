package repository

import (
	"context"

	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// LedgerRepo manages all database interactions for the ledger.
type LedgerRepo struct {
	pool *pgxpool.Pool
}

// NewLedgerRepo is the constructor.
func NewLedgerRepo(pool *pgxpool.Pool) *LedgerRepo {
	return &LedgerRepo{pool: pool}
}

// GetBalance calculates the total balance for a specific account.
func (r *LedgerRepo) GetBalance(ctx context.Context, accountID string) (float64, error) {
	var balance float64

	// The Architect: The SQL lives here, and ONLY here.
	query := `SELECT COALESCE(SUM(amount), 0) FROM entries WHERE account_id = $1`

	err := r.pool.QueryRow(ctx, query, accountID).Scan(&balance)
	if err != nil {
		return 0, err
	}

	return balance, nil
}

// ExecuteTransfer handles the ACID transaction for moving funds safely.
func (r *LedgerRepo) ExecuteTransfer(ctx context.Context, senderID, receiverID string, amount float64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. PESSIMISTIC LOCK: Lock the Sender's Account Row
	_, err = tx.Exec(ctx, "SELECT id FROM accounts WHERE id = $1 FOR UPDATE", senderID)
	if err != nil {
		return fmt.Errorf("database locked, try again: %w", err)
	}

	// 2. CALCULATE BALANCE
	var currentBalance float64
	err = tx.QueryRow(ctx, "SELECT COALESCE(SUM(amount), 0) FROM entries WHERE account_id = $1", senderID).Scan(&currentBalance)
	if err != nil {
		return fmt.Errorf("failed to calculate balance: %w", err)
	}

	// 3. BUSINESS LOGIC: Prevent the Infinite Money Glitch
	if currentBalance < amount {
		return fmt.Errorf("insufficient funds") // We return an error, stopping the execution
	}

	// 4. EXECUTE TRANSFER
	txRef := fmt.Sprintf("API_TX_%d", time.Now().UnixNano())

	_, err = tx.Exec(ctx, "INSERT INTO transactions (reference) VALUES ($1)", txRef)
	if err != nil {
		return fmt.Errorf("failed to write transaction: %w", err)
	}

	_, err = tx.Exec(ctx, "INSERT INTO entries (transaction_id, account_id, amount) VALUES ((SELECT id FROM transactions WHERE reference = $1), $2, $3)", txRef, senderID, -amount)
	if err != nil {
		return fmt.Errorf("failed to write debit: %w", err)
	}

	_, err = tx.Exec(ctx, "INSERT INTO entries (transaction_id, account_id, amount) VALUES ((SELECT id FROM transactions WHERE reference = $1), $2, $3)", txRef, receiverID, amount)
	if err != nil {
		return fmt.Errorf("failed to write credit: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
