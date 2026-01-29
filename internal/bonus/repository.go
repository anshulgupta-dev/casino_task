package bonus

import (
	"context"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrBonusNotFound         = errors.New("bonus not found")
	ErrBonusNotActive        = errors.New("bonus is not active")
	ErrBonusExpired          = errors.New("bonus has expired")
	ErrGameNotFound          = errors.New("game not found")
	ErrWageringEventExists   = errors.New("wagering event already exists for this bet")
	ErrWageringEventNotFound = errors.New("wagering event not found")
)

type BonusRepository interface {
	GetActiveBonus(ctx context.Context, playerID string) (*PlayerBonus, error)
	GetGame(ctx context.Context, gameID string) (*Game, error)
	GetEventByBetID(ctx context.Context, betID string) (*WageringEvent, error)
	GetBonusForUpdate(ctx context.Context, tx *gorm.DB, playerBonusID string) (*PlayerBonus, error)
	UpdateWageringProgress(ctx context.Context, tx *gorm.DB, playerBonusID string, newProgress decimal.Decimal) error
	CreateWageringEvent(ctx context.Context, tx *gorm.DB, wageringEvent *WageringEvent) error
	UpdateBonusStatus(ctx context.Context, tx *gorm.DB, playerBonusID string, status string) error
	GetBonus(ctx context.Context, playerBonusID string) (*PlayerBonus, error)
	CreatePlayerBonus(ctx context.Context, playerBonus *PlayerBonus) error
}

type BonusRepositoryImpl struct {
	db *gorm.DB
}

func NewBonusRepository(db *gorm.DB) *BonusRepositoryImpl {
	return &BonusRepositoryImpl{db: db}
}

func (r *BonusRepositoryImpl) GetActiveBonus(ctx context.Context, playerID string) (*PlayerBonus, error) {
	var bonus PlayerBonus
	err := r.db.WithContext(ctx).
		Where("player_id = ? AND status = ?", playerID, BonusStatusActive).
		First(&bonus).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBonusNotFound
		}
		return nil, fmt.Errorf("failed to get active bonus: %w", err)
	}

	return &bonus, nil
}
func (r *BonusRepositoryImpl) GetGame(ctx context.Context, gameID string) (*Game, error) {
	var game Game
	err := r.db.WithContext(ctx).
		Where("game_id = ?", gameID).
		First(&game).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrGameNotFound
		}
		return nil, fmt.Errorf("failed to get game: %w", err)
	}

	return &game, nil
}

func (r *BonusRepositoryImpl) GetBonusForUpdate(ctx context.Context, tx *gorm.DB, playerBonusID string) (*PlayerBonus, error) {
	var bonus PlayerBonus

	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("player_bonus_id = ?", playerBonusID).
		First(&bonus).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBonusNotFound
		}
		return nil, fmt.Errorf("failed to lock bonus: %w", err)
	}

	return &bonus, nil
}

func (r *BonusRepositoryImpl) GetEventByBetID(ctx context.Context, betID string) (*WageringEvent, error) {
	var event WageringEvent
	err := r.db.WithContext(ctx).
		Where("bet_id = ?", betID).
		First(&event).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWageringEventNotFound
		}
		return nil, fmt.Errorf("failed to get wagering event: %w", err)
	}

	return &event, nil
}

func (r *BonusRepositoryImpl) UpdateWageringProgress(ctx context.Context, tx *gorm.DB, playerBonusID string, newProgress decimal.Decimal) error {
	result := tx.WithContext(ctx).
		Model(&PlayerBonus{}).
		Where("player_bonus_id = ?", playerBonusID).
		Updates(map[string]interface{}{
			"wagering_completed": newProgress,
			"updated_at":         gorm.Expr("NOW()"),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update wagering progress: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrBonusNotFound
	}

	return nil
}

func (r *BonusRepositoryImpl) CreateWageringEvent(ctx context.Context, tx *gorm.DB, event *WageringEvent) error {
	err := tx.WithContext(ctx).Create(event).Error
	if err != nil {
		return fmt.Errorf("failed to create wagering event: %w", err)
	}
	return nil
}

func (r *BonusRepositoryImpl) UpdateBonusStatus(ctx context.Context, tx *gorm.DB, playerBonusID string, status string) error {
	result := tx.WithContext(ctx).
		Model(&PlayerBonus{}).
		Where("player_bonus_id = ?", playerBonusID).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": gorm.Expr("NOW()"),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update bonus status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrBonusNotFound
	}

	return nil
}

func (r *BonusRepositoryImpl) GetBonus(ctx context.Context, playerBonusID string) (*PlayerBonus, error) {
	var bonus PlayerBonus
	err := r.db.WithContext(ctx).
		Where("player_bonus_id = ?", playerBonusID).
		First(&bonus).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBonusNotFound
		}
		return nil, fmt.Errorf("failed to get bonus: %w", err)
	}

	return &bonus, nil
}

func (r *BonusRepositoryImpl) CreatePlayerBonus(ctx context.Context, bonus *PlayerBonus) error {
	err := r.db.WithContext(ctx).Create(bonus).Error
	if err != nil {
		return fmt.Errorf("failed to create player bonus: %w", err)
	}
	return nil
}
