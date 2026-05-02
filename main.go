package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
)

func main() {
	// The Architect: We connect via localhost because Go is running on your Windows host right now,
	// communicating through the port (5432) we explicitly exposed in docker-compose.yml.
	dbURL := "postgres://ledger_admin:super_secret_password_123@localhost:5432/ledgercore"

	// The SRE: Never let a database dictate how long your API waits.
	// We enforce a strict 3-second timeout for the connection attempt.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Attempt the connection
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	// Ensure the connection is cleanly closed when the program exits
	defer conn.Close(context.Background())

	fmt.Println("LedgerCore Go Engine connected to PostgreSQL successfully! The engine is alive.")
}
