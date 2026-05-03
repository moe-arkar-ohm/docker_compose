package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// 1. THE INTERFACE (The Electrical Outlet)
// The handler says: "I will accept ANY object that implements these two functions exactly."
type LedgerService interface {
	GetBalance(ctx context.Context, accountID string) (float64, error)
	ExecuteTransfer(ctx context.Context, senderID, receiverID string, amount float64) error
}

// 2. The Handler Struct holds the interface
type LedgerHandler struct {
	service LedgerService
}

// 3. The Constructor requires something that plugs into the interface
func NewLedgerHandler(s LedgerService) *LedgerHandler {
	return &LedgerHandler{service: s}
}

// ... (We will move the HTTP functions here next)
// Create a struct for the JSON payload
type TransferRequest struct {
	SenderID   string  `json:"sender_id"`
	ReceiverID string  `json:"receiver_id"`
	Amount     float64 `json:"amount"`
}

// HandleTransfer is the Waiter method for POST /transfer
func (h *LedgerHandler) HandleTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// 4. THE MAGIC: We call ExecuteTransfer on the INTERFACE, not the Postgres repo directly.
	err := h.service.ExecuteTransfer(ctx, req.SenderID, req.ReceiverID, req.Amount)
	if err != nil {
		if err.Error() == "insufficient funds" {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		fmt.Printf("DEBUG: Transfer failed: %v\n", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Transfer completed securely.\n"))
}
