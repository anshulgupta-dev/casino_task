
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE wallets (
    wallet_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    player_id UUID NOT NULL,
    wallet_type VARCHAR(20) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    balance NUMERIC(20, 2) NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT positive_balance CHECK (balance >= 0),
    UNIQUE(player_id, wallet_type, currency)
);

CREATE INDEX idx_wallets_player ON wallets(player_id);

CREATE TABLE transactions (
    transaction_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    wallet_id UUID NOT NULL REFERENCES wallets(wallet_id),
    player_id UUID NOT NULL,
    transaction_type VARCHAR(20) NOT NULL,
    amount NUMERIC(20, 2) NOT NULL,
    balance_before NUMERIC(20, 2) NOT NULL,
    balance_after NUMERIC(20, 2) NOT NULL,
    reference_id VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,
    UNIQUE(reference_id, transaction_type)
);

CREATE INDEX idx_transactions_wallet ON transactions(wallet_id);
CREATE INDEX idx_transactions_player ON transactions(player_id);
CREATE INDEX idx_transactions_ref ON transactions(reference_id);
