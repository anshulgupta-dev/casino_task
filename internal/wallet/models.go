package wallet

import (
	"time"

	"github.com/shopspring/decimal"
)

type Wallet struct {
	WalletID   string          `db:"wallet_id"`
	PlayerID   string          `db:"player_id"`
	WalletType string          `db:"wallet_type"` // "main", "bonus"
	Currency   string          `db:"currency"`
	Balance    decimal.Decimal `db:"balance"`
	Version    int             `db:"version"`
	CreatedAt  time.Time       `db:"created_at"`
	UpdatedAt  time.Time       `db:"updated_at"`
}

type Transaction struct {
	TransactionID   string          `db:"transaction_id"`
	WalletID        string          `db:"wallet_id"`
	PlayerID        string          `db:"player_id"`
	TransactionType string          `db:"transaction_type"` // "deposit", "withdrawal", "bet", "win"
	Amount          decimal.Decimal `db:"amount"`
	BalanceBefore   decimal.Decimal `db:"balance_before"`
	BalanceAfter    decimal.Decimal `db:"balance_after"`
	ReferenceID     string          `db:"reference_id"` // external reference (game round, payment ID)
	Status          string          `db:"status"`       // "pending", "completed", "failed"
	CreatedAt       time.Time       `db:"created_at"`
	CompletedAt     *time.Time      `db:"completed_at"`
}

type TransactionRequest struct {
	PlayerID        string          `json:"player_id"`
	WalletType      string          `json:"wallet_type"`
	TransactionType string          `json:"transaction_type"`
	Amount          decimal.Decimal `json:"amount"`
	ReferenceID     string          `json:"reference_id"`
	Currency        string          `json:"currency"`
}

type TransactionResponse struct {
	TransactionID string          `json:"transaction_id"`
	Balance       decimal.Decimal `json:"balance"`
	Status        string          `json:"status"`
}
