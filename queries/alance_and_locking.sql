-- ==============================================================================
-- LEDGERCORE RAW SQL REFERENCE
-- Target Backend: Go (pgx or database/sql)
-- Architecture: Double-Entry Ledger Microservice
-- ==============================================================================

-- ------------------------------------------------------------------------------
-- 1. Calculate Account Balance (Aggregation)
-- Context: Double-entry means balance is calculated on the fly, never stored statically.
-- Go Implementation: Run this when a user queries their wallet balance via FastAPI/Go API.
-- ------------------------------------------------------------------------------
SELECT 
    account_id, 
    SUM(amount) AS current_balance
FROM entries
WHERE account_id = $1  -- $1 is the parameter injected by your Go backend
GROUP BY account_id;


-- ------------------------------------------------------------------------------
-- 2. Pessimistic Row-Level Locking (Concurrency Control)
-- Context: Prevents the "Double-Spend" race condition.
-- Go Implementation: MUST be executed inside an explicit transaction block (tx.Begin()).
-- Run this BEFORE calculating if the user has enough funds to make a transfer.
-- ------------------------------------------------------------------------------
SELECT 
    id, 
    name, 
    created_at 
FROM accounts 
WHERE id = $1          -- $1 is the parameter injected by your Go backend
FOR UPDATE;

-- Example Go Transaction Flow for the above lock:
-- 1. tx, err := db.Begin()
-- 2. tx.Exec("SELECT ... FOR UPDATE", senderID)
-- 3. tx.Exec("SELECT ... FOR UPDATE", receiverID)
-- 4. Check if sender balance >= transfer amount
-- 5. tx.Exec("INSERT INTO entries ... (Debit)")
-- 6. tx.Exec("INSERT INTO entries ... (Credit)")
-- 7. tx.Commit()
DO $$
DECLARE
    user_a_id UUID := 'a1111111-1111-1111-1111-111111111111';
    dummy_tx UUID;
BEGIN
    FOR i IN 1..100000 LOOP
        dummy_tx := uuid_generate_v4();
        INSERT INTO transactions (id, reference) VALUES (dummy_tx, 'DUMMY_' || i);
        INSERT INTO entries (transaction_id, account_id, amount) VALUES (dummy_tx, user_a_id, 1.0000);
    END LOOP;
END $$;