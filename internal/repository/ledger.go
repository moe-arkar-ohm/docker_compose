package repository

import (
	"context"

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
