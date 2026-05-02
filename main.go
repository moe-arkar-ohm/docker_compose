package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// 1. The Connection String
	dbURL := "postgres://ledger_admin:super_secret_password_123@localhost:5432/ledgercore"

	// 2. Create a Connection Pool (Instead of a single connection)
	// pgxpool manages dozens of connections for high concurrency.
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: Unable to create connection pool: %v\n", err)
		os.Exit(1)
	}
	// Defer the closing of the ENTIRE pool when the program shuts down
	defer pool.Close()

	fmt.Println("LedgerCore Pool initialized successfully.")

	// 3. The Target Data
	// This is User A's UUID that we seeded earlier.
	userA_ID := "a1111111-1111-1111-1111-111111111111"

	// 4. The Raw SQL Query
	sqlQuery := `
		SELECT SUM(amount) 
		FROM entries 
		WHERE account_id = $1
	`

	// 5. Execute the Query
	var currentBalance float64 // Variable to store the result

	// QueryRow asks for exactly ONE row of data.
	// Scan takes the database output and injects it into our currentBalance variable.
	err = pool.QueryRow(context.Background(), sqlQuery, userA_ID).Scan(&currentBalance)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Query failed: %v\n", err)
		return
	}

	// 6. Output the result
	fmt.Printf("User A's Current Balance: $%.4f\n", currentBalance)
}
