package bonus

import (
	"time"

	"github.com/shopspring/decimal"
)

type PlayerBonus struct {
	PlayerBonusID     string          `gorm:"column:player_bonus_id;primaryKey;type:uuid;default:uuid_generate_v4()"`
	PlayerID          string          `gorm:"column:player_id;type:uuid;not null"`
	BonusID           string          `gorm:"column:bonus_id;type:uuid;not null"`
	Status            string          `gorm:"column:status;type:varchar(20);not null;default:'active'"` // "active", "completed", "forfeited", "expired"
	BonusAmount       decimal.Decimal `gorm:"column:bonus_amount;type:numeric(20,2);not null"`
	WageringRequired  decimal.Decimal `gorm:"column:wagering_required;type:numeric(20,2);not null"`
	WageringCompleted decimal.Decimal `gorm:"column:wagering_completed;type:numeric(20,2);not null;default:0"`
	ExpiresAt         time.Time       `gorm:"column:expires_at;not null"`
	CreatedAt         time.Time       `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt         time.Time       `gorm:"column:updated_at;not null;default:now()"`
}

type Game struct {
	GameID       string          `gorm:"column:game_id;primaryKey;type:uuid;default:uuid_generate_v4()"`
	GameName     string          `gorm:"column:game_name;type:varchar(100);not null"`
	GameType     string          `gorm:"column:game_type;type:varchar(50);not null"`     // "slots", "table_games", "live_casino"
	Contribution decimal.Decimal `gorm:"column:contribution;type:numeric(5,4);not null"` // 0.0000 to 1.0000 (100%)
	CreatedAt    time.Time       `gorm:"column:created_at;not null;default:now()"`
	UpdatedAt    time.Time       `gorm:"column:updated_at;not null;default:now()"`
}

type WageringEvent struct {
	EventID                string          `gorm:"column:event_id;primaryKey;type:uuid;default:uuid_generate_v4()"`
	PlayerBonusID          string          `gorm:"column:player_bonus_id;type:uuid;not null"`
	BetID                  string          `gorm:"column:bet_id;type:varchar(255);not null;unique"` // for idempotency
	GameID                 string          `gorm:"column:game_id;type:uuid;not null"`
	BetAmount              decimal.Decimal `gorm:"column:bet_amount;type:numeric(20,2);not null"`
	ContributionPercentage decimal.Decimal `gorm:"column:contribution_percentage;type:numeric(5,4);not null"`
	WageringContribution   decimal.Decimal `gorm:"column:wagering_contribution;type:numeric(20,2);not null"`
	CreatedAt              time.Time       `gorm:"column:created_at;not null;default:now()"`
}
type BetEvent struct {
	BetID     string          `json:"bet_id"`
	PlayerID  string          `json:"player_id"`
	GameID    string          `json:"game_id"`
	BetAmount decimal.Decimal `json:"bet_amount"`
	Timestamp time.Time       `json:"timestamp"`
}

type GameContribution struct {
	GameID       string          `json:"game_id"`
	GameType     string          `json:"game_type"`    // "slots", "table_games", "live_casino"
	Contribution decimal.Decimal `json:"contribution"` // 0.0 to 1.0 (100%)
}

type WageringProgress struct {
	PlayerBonusID      string          `json:"player_bonus_id"`
	WageringRequired   decimal.Decimal `json:"wagering_required"`
	WageringCompleted  decimal.Decimal `json:"wagering_completed"`
	PercentageComplete float64         `json:"percentage_complete"`
	Completed          bool            `json:"completed"`
}

type WageringUpdate struct {
	PlayerBonusID      string          `json:"player_bonus_id"`
	PlayerID           string          `json:"player_id"`
	WageringCompleted  decimal.Decimal `json:"wagering_completed"`
	WageringRequired   decimal.Decimal `json:"wagering_required"`
	PercentageComplete float64         `json:"percentage_complete"`
	Completed          bool            `json:"completed"`
	Timestamp          time.Time       `json:"timestamp"`
}

const (
	BonusStatusActive    = "active"
	BonusStatusCompleted = "completed"
	BonusStatusForfeited = "forfeited"
	BonusStatusExpired   = "expired"
)

const (
	GameTypeSlots      = "slots"
	GameTypeTableGames = "table_games"
	GameTypeLiveCasino = "live_casino"
)
