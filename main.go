package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// We define a struct to represent the JSON response we will send back.
type BalanceResponse struct {
	AccountID string  `json:"account_id"`
	Balance   float64 `json:"balance"`
}

// 1. Define the input structure matching the JSON payload
type TransferRequest struct {
	SenderID   string  `json:"sender_id"`
	ReceiverID string  `json:"receiver_id"`
	Amount     float64 `json:"amount"`
}

func main() {
	// 1. Initialize the Connection Pool
	dbURL := "postgres://ledger_admin:super_secret_password_123@localhost:5432/ledgercore"
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: Unable to create connection pool: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()
	fmt.Println("LedgerCore Pool initialized successfully.")

	// 2. Define the Drive-Thru Window (The Endpoint)
	http.HandleFunc("/balance", func(w http.ResponseWriter, r *http.Request) {
		// For now, we hardcode the ID. Later, we will extract this from the URL.
		accountID := "a1111111-1111-1111-1111-111111111111"
		var currentBalance float64

		// Execute the query
		sqlQuery := `SELECT SUM(amount) FROM entries WHERE account_id = $1`
		err := pool.QueryRow(r.Context(), sqlQuery, accountID).Scan(&currentBalance)

		if err != nil {
			// If the query fails, return an HTTP 500 status code
			fmt.Printf("DEBUG: JSON Decode Error: %v\n", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Prepare the JSON response
		response := BalanceResponse{
			AccountID: accountID,
			Balance:   currentBalance,
		}

		// Set the header to tell the client we are sending JSON, then send it.
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	// Inside your main() function, add this new endpoint:
	http.HandleFunc("/transfer", func(w http.ResponseWriter, r *http.Request) {
		// Only allow POST requests
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Decode the incoming JSON payload into our Go struct
		var req TransferRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			fmt.Printf("DEBUG: JSON Decode Error: %v\n", err) // <--- ADD THIS
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		// Begin the ACID Transaction using the request context
		tx, err := pool.Begin(r.Context())
		if err != nil {
			fmt.Printf("DEBUG: JSON Decode Error: %v\n", err)
			http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
			return
		}

		// THE SRE SAFETY NET:
		// If the function returns before calling tx.Commit(), this defer will automatically rollback.
		// If tx.Commit() is successful, tx.Rollback() does nothing. It is bulletproof.
		defer tx.Rollback(r.Context())

		// Generate a pseudo-random transaction reference (in production, use Google's uuid package)
		txRef := fmt.Sprintf("API_TX_%d", time.Now().UnixNano())

		// 1. Create the parent transaction record
		_, err = tx.Exec(r.Context(), "INSERT INTO transactions (reference) VALUES ($1)", txRef)
		if err != nil {
			fmt.Printf("DEBUG: JSON Decode Error: %v\n", err)
			http.Error(w, "Failed to write transaction", http.StatusInternalServerError)
			return
		}

		// 2. Debit the Sender (notice we multiply amount by -1)
		_, err = tx.Exec(r.Context(), "INSERT INTO entries (transaction_id, account_id, amount) VALUES ((SELECT id FROM transactions WHERE reference = $1), $2, $3)", txRef, req.SenderID, -req.Amount)
		if err != nil {
			fmt.Printf("DEBUG: JSON Decode Error: %v\n", err)
			http.Error(w, "Failed to write debit", http.StatusInternalServerError)
			return
		}

		// 3. Credit the Receiver
		_, err = tx.Exec(r.Context(), "INSERT INTO entries (transaction_id, account_id, amount) VALUES ((SELECT id FROM transactions WHERE reference = $1), $2, $3)", txRef, req.ReceiverID, req.Amount)
		if err != nil {
			fmt.Printf("DEBUG: JSON Decode Error: %v\n", err)
			http.Error(w, "Failed to write credit", http.StatusInternalServerError)
			return
		}

		// Commit the transaction to the database
		if err := tx.Commit(r.Context()); err != nil {
			http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("Transfer completed successfully.\n"))
	})

	// 3. Start the Server
	fmt.Println("LedgerCore API is listening on port 8080...")
	// This function blocks forever, keeping the server alive.
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Fprintf(os.Stderr, "Server crashed: %v\n", err)
	}
}
