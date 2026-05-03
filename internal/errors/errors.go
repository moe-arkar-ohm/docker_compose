package errors

import "errors"

// ErrInsufficientFunds is our Sentinel Error.
// It is explicitly defined here and exported (capital E).
var ErrInsufficientFunds = errors.New("insufficient funds")
