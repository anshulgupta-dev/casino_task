package bonus

import (
	"time"

	"github.com/shopspring/decimal"
)

type PlayerBonus struct {
	PlayerBonusID     string          `db:"player_bonus_id"`
	PlayerID          string          `db:"player_id"`
	BonusID           string          `db:"bonus_id"`
	Status            string          `db:"status"` // "active", "completed", "forfeited"
	BonusAmount       decimal.Decimal `db:"bonus_amount"`
	WageringRequired  decimal.Decimal `db:"wagering_required"`
	WageringCompleted decimal.Decimal `db:"wagering_completed"`
	ExpiresAt         time.Time       `db:"expires_at"`
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
	PlayerID     string  `json:"player_id"`
	Percentage   float64 `json:"percentage"`
	WageringLeft float64 `json:"wagering_left"`
	IsCompleted  bool    `json:"is_completed"`
}
