
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

-- Bonus tables
CREATE TABLE player_bonus (
    player_bonus_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    player_id UUID NOT NULL,
    bonus_id UUID NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    bonus_amount NUMERIC(20, 2) NOT NULL,
    wagering_required NUMERIC(20, 2) NOT NULL,
    wagering_completed NUMERIC(20, 2) NOT NULL DEFAULT 0,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_bonus_status CHECK (status IN ('active', 'completed', 'forfeited', 'expired'))
);

CREATE INDEX idx_player_bonus_player ON player_bonus(player_id);
CREATE INDEX idx_player_bonus_status ON player_bonus(status);

CREATE TABLE games (
    game_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    game_name VARCHAR(100) NOT NULL,
    game_type VARCHAR(50) NOT NULL,
    contribution NUMERIC(5, 4) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_contribution CHECK (contribution >= 0 AND contribution <= 1)
);

CREATE INDEX idx_games_type ON games(game_type);

CREATE TABLE wagering_events (
    event_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    player_bonus_id UUID NOT NULL REFERENCES player_bonus(player_bonus_id),
    bet_id VARCHAR(255) NOT NULL UNIQUE,
    game_id UUID NOT NULL REFERENCES games(game_id),
    bet_amount NUMERIC(20, 2) NOT NULL,
    contribution_percentage NUMERIC(5, 4) NOT NULL,
    wagering_contribution NUMERIC(20, 2) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_wagering_events_bonus ON wagering_events(player_bonus_id);
CREATE INDEX idx_wagering_events_bet ON wagering_events(bet_id);

-- Seed data for games (for testing)
INSERT INTO games (game_id, game_name, game_type, contribution) VALUES
    ('11111111-1111-1111-1111-111111111111', 'Slots Game', 'slots', 1.0000),
    ('22222222-2222-2222-2222-222222222222', 'Blackjack', 'table_games', 0.1000),
    ('33333333-3333-3333-3333-333333333333', 'Live Roulette', 'live_casino', 0.5000);
