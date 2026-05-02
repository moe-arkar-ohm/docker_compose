package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// We define a struct to represent the JSON response we will send back.
type BalanceResponse struct {
	AccountID string  `json:"account_id"`
	Balance   float64 `json:"balance"`
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

	// 3. Start the Server
	fmt.Println("LedgerCore API is listening on port 8080...")
	// This function blocks forever, keeping the server alive.
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Fprintf(os.Stderr, "Server crashed: %v\n", err)
	}
}
