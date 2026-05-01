INSERT INTO accounts (id, name) VALUES 
('a1111111-1111-1111-1111-111111111111', 'User A Wallet'),
('b2222222-2222-2222-2222-222222222222', 'User B Wallet');

BEGIN;

-- 1. Create the parent transaction record
INSERT INTO transactions (id, reference) 
VALUES ('t3333333-3333-3333-3333-333333333333', 'PAYMENT_REQ_99812');

-- 2. Debit User A (Withdrawal)
INSERT INTO entries (transaction_id, account_id, amount) 
VALUES ('t3333333-3333-3333-3333-333333333333', 'a1111111-1111-1111-1111-111111111111', -50.0000);

-- 3. Credit User B (Deposit)
INSERT INTO entries (transaction_id, account_id, amount) 
VALUES ('t3333333-3333-3333-3333-333333333333', 'b2222222-2222-2222-2222-222222222222', 50.0000);

COMMIT;