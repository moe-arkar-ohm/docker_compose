package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

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
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req TransferRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			fmt.Printf("DEBUG: JSON Decode Error: %v\n", err)
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		// =====================================================================
		// THE TIMEOUT SAFETY NET
		// We wrap the HTTP request context with a strict 5-second limit.
		// Every database call below will use 'ctx' instead of 'r.Context()'.
		// =====================================================================
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		tx, err := pool.Begin(ctx)
		if err != nil {
			fmt.Printf("DEBUG: Failed to begin tx: %v\n", err)
			http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback(ctx)

		// 1. PESSIMISTIC LOCK: Lock the Sender's Account Row
		// This freezes any other API requests trying to touch User A's money.
		_, err = tx.Exec(ctx, "SELECT id FROM accounts WHERE id = $1 FOR UPDATE", req.SenderID)
		if err != nil {
			fmt.Printf("DEBUG: Failed to lock account: %v\n", err)
			http.Error(w, "Database locked, try again", http.StatusGatewayTimeout) // 504 Error!
			return
		}

		// 2. CALCULATE BALANCE: COALESCE turns a NULL (no entries yet) into a 0.
		var currentBalance float64
		err = tx.QueryRow(ctx, "SELECT COALESCE(SUM(amount), 0) FROM entries WHERE account_id = $1", req.SenderID).Scan(&currentBalance)
		if err != nil {
			fmt.Printf("DEBUG: Failed to calculate balance: %v\n", err)
			http.Error(w, "Failed to verify funds", http.StatusInternalServerError)
			return
		}

		// 3. THE BUSINESS LOGIC: Prevent the Infinite Money Glitch
		if currentBalance < req.Amount {
			// Do not process the transfer. The defer tx.Rollback() will clean up.
			http.Error(w, "Insufficient funds", http.StatusBadRequest)
			return
		}

		// 4. EXECUTE TRANSFER (Now that we know it is mathematically safe)
		txRef := fmt.Sprintf("API_TX_%d", time.Now().UnixNano())

		_, err = tx.Exec(ctx, "INSERT INTO transactions (reference) VALUES ($1)", txRef)
		if err != nil {
			http.Error(w, "Failed to write transaction", http.StatusInternalServerError)
			return
		}

		_, err = tx.Exec(ctx, "INSERT INTO entries (transaction_id, account_id, amount) VALUES ((SELECT id FROM transactions WHERE reference = $1), $2, $3)", txRef, req.SenderID, -req.Amount)
		if err != nil {
			http.Error(w, "Failed to write debit", http.StatusInternalServerError)
			return
		}

		_, err = tx.Exec(ctx, "INSERT INTO entries (transaction_id, account_id, amount) VALUES ((SELECT id FROM transactions WHERE reference = $1), $2, $3)", txRef, req.ReceiverID, req.Amount)
		if err != nil {
			http.Error(w, "Failed to write credit", http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(ctx); err != nil {
			http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("Transfer completed securely.\n"))
	})

	// 3. Start the Server
	fmt.Println("LedgerCore API is listening on port 8080...")
	// This function blocks forever, keeping the server alive.
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Fprintf(os.Stderr, "Server crashed: %v\n", err)
	}
}
