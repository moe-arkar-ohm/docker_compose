-- 1. Enable UUID generation inside PostgreSQL
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 2. The Accounts Table (e.g., User A Wallet, System Revenue, Escrow)
CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 3. The Transactions Table (The atomic event, e.g., "User A pays User B")
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    reference VARCHAR(255) UNIQUE NOT NULL, -- An idempotency key from your Go API
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 4. The Entries Table (The actual money movement)
CREATE TABLE entries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    transaction_id UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounts(id),
    amount DECIMAL(19, 4) NOT NULL, -- Use DECIMAL for money, NEVER use FLOAT
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);