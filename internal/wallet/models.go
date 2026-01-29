package wallet

import (
	"time"

	"github.com/shopspring/decimal"
)

type Wallet struct {
	WalletID   string          `gorm:"column:wallet_id;primaryKey;type:uuid;default:uuid_generate_v4()"`
	PlayerID   string          `gorm:"column:player_id;type:uuid;not null"`
	WalletType string          `gorm:"column:wallet_type;type:varchar(20);not null"` // "main", "bonus"
	Currency   string          `gorm:"column:currency;type:varchar(3);not null"`
	Balance    decimal.Decimal `gorm:"column:balance;type:numeric(20,2);not null;default:0"`
	Version    int             `gorm:"column:version;not null;default:1"`
	CreatedAt  time.Time       `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt  time.Time       `gorm:"column:updated_at;not null;default:now()"`
}

type Transaction struct {
	TransactionID   string          `gorm:"column:transaction_id;primaryKey;type:uuid;default:uuid_generate_v4()"`
	WalletID        string          `gorm:"column:wallet_id;type:uuid;not null"`
	PlayerID        string          `gorm:"column:player_id;type:uuid;not null"`
	TransactionType string          `gorm:"column:transaction_type;type:varchar(20);not null"` // "deposit", "withdrawal", "bet", "win"
	Amount          decimal.Decimal `gorm:"column:amount;type:numeric(20,2);not null"`
	BalanceBefore   decimal.Decimal `gorm:"column:balance_before;type:numeric(20,2);not null"`
	BalanceAfter    decimal.Decimal `gorm:"column:balance_after;type:numeric(20,2);not null"`
	ReferenceID     string          `gorm:"column:reference_id;type:varchar(255);not null"` // external reference (game round, payment ID)
	Status          string          `gorm:"column:status;type:varchar(20);not null"`        // "pending", "completed", "failed"
	CreatedAt       time.Time       `gorm:"column:created_at;not null;default:now()"`
	CompletedAt     *time.Time      `gorm:"column:completed_at"`
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
