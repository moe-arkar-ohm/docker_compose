package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"ledgercore/internal/config"
	"ledgercore/internal/handlers"
	"ledgercore/internal/repository"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

// 1. Define the Upgrader (The Switchboard Operator)
// This upgrades the Walkie-Talkie connection to a Phone Call connection.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Security Lead: In production, strictly verify the origin here.
	},
}

// We define a struct to represent the JSON response we will send back.
type BalanceResponse struct {
	AccountID string  `json:"account_id"`
	Balance   float64 `json:"balance"`
}

func main() {
	// 1. Initialize the Connection Pool
	// 1. Load Configuration
	cfg := config.Load()

	// 2. Initialize the Connection Pool using the injected URL
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: Unable to create connection pool: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()
	fmt.Println("LedgerCore Pool initialized successfully.")
	// 3. Initialize the Repository Layer
	ledgerRepo := repository.NewLedgerRepo(pool)
	// 3. Initialize the Repository Layer (The Concrete Postgres Farmer)

	// 4. Initialize the Handler Layer (The Waiter)
	// Because `ledgerRepo` has the exact functions required by the `LedgerService` interface,
	// Go automatically allows it to be plugged in here!
	ledgerHandler := handlers.NewLedgerHandler(ledgerRepo)

	// 5. Define the Routes
	http.HandleFunc("/transfer", ledgerHandler.HandleTransfer)
	// ... (Keep your old /balance and /ws routes here for now)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// 2. Upgrade the connection
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Printf("DEBUG: Failed to upgrade WebSocket: %v\n", err)
			return
		}
		defer conn.Close()

		fmt.Println("New WebSocket client connected!")

		// 3. Keep the line open and listen
		for {
			// Read messages from the Flutter client (if they send any)
			messageType, p, err := conn.ReadMessage()
			if err != nil {
				fmt.Println("Client disconnected.")
				break // Exit the loop and close the connection if the client drops
			}

			// Just an echo for testing: send the exact message back
			if err := conn.WriteMessage(messageType, p); err != nil {
				fmt.Println("Failed to send message.")
				break
			}
		}
	})
	// 2. Define the Drive-Thru Window (The Endpoint)
	http.HandleFunc("/balance", func(w http.ResponseWriter, r *http.Request) {
		// For now, we hardcode the ID. Later, we will extract this from the URL.
		accountID := "a1111111-1111-1111-1111-111111111111"
		var currentBalance float64

		// Execute the query
		// sqlQuery := `SELECT SUM(amount) FROM entries WHERE account_id = $1`
		// err := pool.QueryRow(r.Context(), sqlQuery, accountID).Scan(&currentBalance)

		// if err != nil {
		// 	// If the query fails, return an HTTP 500 status code
		// 	http.Error(w, "Database error", http.StatusInternalServerError)
		// 	return
		// }
		// The Waiter (Handler) asks the Farmer (Repository) for the data.
		currentBalance, err := ledgerRepo.GetBalance(r.Context(), accountID)
		if err != nil {
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

	// 3. Start the Server
	fmt.Printf("LedgerCore API is listening on port %s...\n", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		fmt.Fprintf(os.Stderr, "Server crashed: %v\n", err)
	}
	// fmt.Println("LedgerCore API is listening on port 8080...")
	// // This function blocks forever, keeping the server alive.
	// if err := http.ListenAndServe(":8080", nil); err != nil {
	// 	fmt.Fprintf(os.Stderr, "Server crashed: %v\n", err)
	// }
}
