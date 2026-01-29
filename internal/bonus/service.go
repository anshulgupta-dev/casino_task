package bonus

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type BonusWageringService interface {
	ProcessBetWagering(ctx context.Context, bet BetEvent) error
	GetWageringProgress(ctx context.Context, bonusID string, playerID string) (*WageringProgress, error)
	SubscribeToWageringUpdates(playerID string) (<-chan WageringUpdate, error)
	CreatePlayerBonus(ctx context.Context, playerID string, bonusID string, bonusAmount decimal.Decimal, wageringMultiplier decimal.Decimal, expiresAt time.Time) error
}

type BonusService struct {
	db        *gorm.DB
	repo      BonusRepository
	notifyHub *NotificationHub
}
type NotificationHub struct {
	mu          sync.RWMutex
	subscribers map[string][]chan WageringUpdate
}

func NewNotificationHub() *NotificationHub {
	return &NotificationHub{
		subscribers: make(map[string][]chan WageringUpdate),
	}
}
func (h *NotificationHub) Subscribe(playerID string) <-chan WageringUpdate {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan WageringUpdate, 10)
	h.subscribers[playerID] = append(h.subscribers[playerID], ch)
	return ch
}
func (h *NotificationHub) Notify(playerID string, update WageringUpdate) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, ch := range h.subscribers[playerID] {
		select {
		case ch <- update:
		default:
			// Channel full, skip (don't block)
		}
	}
}

func NewBonusService(db *gorm.DB, repo BonusRepository) *BonusService {
	return &BonusService{
		db:        db,
		repo:      repo,
		notifyHub: NewNotificationHub(),
	}
}

func (s *BonusService) ProcessBetWagering(ctx context.Context, bet BetEvent) error {
	_, err := s.repo.GetEventByBetID(ctx, bet.BetID)
	if err == nil {
		log.Printf("Event already exists for bet ID: %s", bet.BetID)
		return nil
	}
	if !errors.Is(err, ErrWageringEventNotFound) {
		log.Printf("idempotency check is failed")
		return err
	}

	activeBonus, err := s.repo.GetActiveBonus(ctx, bet.PlayerID)
	if err != nil {
		if errors.Is(err, ErrBonusNotFound) {
			log.Printf("No active bonus found for player ID: %s", bet.PlayerID)
			return nil
		}
		log.Printf("Error getting active bonus for player ID: %s", bet.PlayerID)
		return fmt.Errorf("error getting active bonus for player ID: %s", bet.PlayerID)
	}
	if time.Now().After(activeBonus.ExpiresAt) {
		log.Printf("Bonus expired: bonus_id=%s player=%s", activeBonus.PlayerBonusID, bet.PlayerID)
		return ErrBonusExpired
	}
	contribution, err := s.getGameContribution(ctx, bet.GameID)
	if err != nil {
		return fmt.Errorf("failed to get game contribution: %w", err)
	}
	wageringContribution := bet.BetAmount.Mul(contribution)
	var bonusCompleted bool
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		bonus, lockErr := s.repo.GetBonusForUpdate(ctx, tx, activeBonus.PlayerBonusID)
		if lockErr != nil {
			return lockErr
		}

		if bonus.Status != BonusStatusActive {
			return ErrBonusNotActive
		}
		newProgress := bonus.WageringCompleted.Add(wageringContribution)
		if newProgress.GreaterThan(bonus.WageringRequired) {
			newProgress = bonus.WageringRequired
		}
		if updateErr := s.repo.UpdateWageringProgress(ctx, tx, bonus.PlayerBonusID, newProgress); updateErr != nil {
			return updateErr
		}
		event := &WageringEvent{
			EventID:                uuid.New().String(),
			PlayerBonusID:          bonus.PlayerBonusID,
			BetID:                  bet.BetID,
			GameID:                 bet.GameID,
			BetAmount:              bet.BetAmount,
			ContributionPercentage: contribution,
			WageringContribution:   wageringContribution,
			CreatedAt:              time.Now(),
		}
		if createErr := s.repo.CreateWageringEvent(ctx, tx, event); createErr != nil {
			return createErr
		}
		if newProgress.GreaterThanOrEqual(bonus.WageringRequired) {
			if statusErr := s.repo.UpdateBonusStatus(ctx, tx, bonus.PlayerBonusID, BonusStatusCompleted); statusErr != nil {
				return statusErr
			}
			bonusCompleted = true
			log.Printf("Bonus wagering completed! bonus_id=%s player=%s", bonus.PlayerBonusID, bet.PlayerID)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to process wagering: %w", err)
	}
	s.sendWageringUpdate(ctx, bet.PlayerID, activeBonus.PlayerBonusID, bonusCompleted)

	log.Printf("Wagering processed: bet_id=%s player=%s contribution=%s completed=%t",
		bet.BetID, bet.PlayerID, wageringContribution.String(), bonusCompleted)

	return nil

}

func (s *BonusService) GetWageringProgress(ctx context.Context, playerID string, bonusID string) (*WageringProgress, error) {
	var bonus *PlayerBonus
	var err error

	// If bonusID is provided, get specific bonus, otherwise get active bonus
	if bonusID != "" {
		bonus, err = s.repo.GetBonus(ctx, bonusID)
	} else {
		bonus, err = s.repo.GetActiveBonus(ctx, playerID)
	}

	if err != nil {
		return nil, err
	}
	percentComplete := float64(0)
	if !bonus.WageringRequired.IsZero() {
		percentComplete = bonus.WageringCompleted.Div(bonus.WageringRequired).
			Mul(decimal.NewFromInt(100)).
			InexactFloat64()
	}

	return &WageringProgress{
		PlayerBonusID:      bonus.PlayerBonusID,
		WageringRequired:   bonus.WageringRequired,
		WageringCompleted:  bonus.WageringCompleted,
		PercentageComplete: percentComplete,
		Completed:          bonus.Status == BonusStatusCompleted,
	}, nil
}

func (s *BonusService) SubscribeToWageringUpdates(playerID string) (<-chan WageringUpdate, error) {
	ch := s.notifyHub.Subscribe(playerID)
	return ch, nil
}

func (s *BonusService) CreatePlayerBonus(ctx context.Context, playerID string, bonusID string, bonusAmount decimal.Decimal, wageringMultiplier decimal.Decimal, expiresAt time.Time) (*PlayerBonus, error) {
	bonus := &PlayerBonus{
		PlayerBonusID:     uuid.New().String(),
		PlayerID:          playerID,
		BonusID:           bonusID,
		Status:            BonusStatusActive,
		BonusAmount:       bonusAmount,
		WageringRequired:  bonusAmount.Mul(wageringMultiplier), // e.g., 35x wagering requirement
		WageringCompleted: decimal.Zero,
		ExpiresAt:         expiresAt,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := s.repo.CreatePlayerBonus(ctx, bonus); err != nil {
		return nil, fmt.Errorf("failed to create player bonus: %w", err)
	}

	log.Printf("Player bonus created: bonus_id=%s player=%s amount=%s wagering_required=%s",
		bonus.PlayerBonusID, playerID, bonusAmount.String(), bonus.WageringRequired.String())

	return bonus, nil
}

func (s *BonusService) getGameContribution(ctx context.Context, gameID string) (decimal.Decimal, error) {
	game, err := s.repo.GetGame(ctx, gameID)
	if err != nil {
		return decimal.Zero, err
	}

	return game.Contribution, nil
}

func (s *BonusService) sendWageringUpdate(ctx context.Context, playerID string, bonusID string, completed bool) {
	progress, err := s.GetWageringProgress(ctx, playerID, bonusID)
	if err != nil {
		log.Printf("Failed to get progress for notification: %v", err)
		return
	}

	update := WageringUpdate{
		PlayerBonusID:      progress.PlayerBonusID,
		PlayerID:           playerID,
		WageringCompleted:  progress.WageringCompleted,
		WageringRequired:   progress.WageringRequired,
		PercentageComplete: progress.PercentageComplete,
		Completed:          completed,
		Timestamp:          time.Now(),
	}

	s.notifyHub.Notify(playerID, update)
}

// func (s *BonusService) AddActiveBonus(bonus *PlayerBonus) {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()
// 	s.bonuses[bonus.PlayerID] = bonus

// }

// func (s *BonusService) ProcessBetWagering(ctx context.Context, bet BetEvent) error {

// 	s.mu.Lock()
// 	defer s.mu.Unlock()

// 	bonus, exists := s.bonuses[bet.PlayerID]
// 	if !exists || bonus.Status != "active" {
// 		return nil
// 	}

// 	if time.Now().After(bonus.ExpiresAt) {
// 		bonus.Status = "expired"
// 		return nil
// 	}

// 	rule, ok := s.gameRules[bet.GameID]
// 	contributionpct := decimal.Zero
// 	if ok {
// 		contributionpct = rule.Contribution
// 	}
// 	contributionAmount := bet.BetAmount.Mul(contributionpct)
// 	bonus.WageringCompleted = bonus.WageringCompleted.Add(contributionAmount)
// 	if bonus.WageringCompleted.GreaterThanOrEqual(bonus.WageringRequired) {
// 		bonus.Status = "completed"
// 		s.notifyPlayer(bet.PlayerID, bonus, true)
// 	}
// 	return nil

// }

// func (s *BonusService) notifyPlayer(playerID string, bonus *PlayerBonus, isCompleted bool) {
// 	s.channelsMu.Lock()
// 	defer s.channelsMu.Unlock()
// 	ch, ok := s.updateChannels[playerID]
// 	if ok {
// 		pct, _ := bonus.WageringCompleted.Div(bonus.WageringRequired).Float64()
// 		egleft, _ := bonus.WageringRequired.Sub(bonus.WageringCompleted).Float64()
// 		update := WageringUpdate{
// 			PlayerID:     playerID,
// 			Percentage:   pct * 100,
// 			WageringLeft: egleft,
// 			IsCompleted:  isCompleted,
// 		}
// 		ch <- update
// 	}

// }

// func (s *BonusService) GetWageringProgress(ctx context.Context, bonusID string, playerID string) (*WageringProgress, error) {
// 	s.mu.RLock()
// 	defer s.mu.RUnlock()
// 	bonus, ok := s.bonuses[playerID]
// 	if !ok {
// 		return nil, errors.New("bonus not found")
// 	}
// 	// fmt.Println("bonus ", bonus)
// 	pct, _ := bonus.WageringCompleted.Div(bonus.WageringRequired).Float64()
// 	return &WageringProgress{
// 		PlayerBonusID:      bonus.PlayerBonusID,
// 		WageringRequired:   bonus.WageringRequired,
// 		WageringCompleted:  bonus.WageringCompleted,
// 		PercentageComplete: pct * 100,
// 		Completed:          bonus.Status == "completed",
// 	}, nil

// }

// func (s *BonusService) SubscribeToWageringUpdates(playerID string) (<-chan WageringUpdate, error) {
// 	s.channelsMu.Lock()
// 	defer s.channelsMu.Unlock()

// 	ch := make(chan WageringUpdate, 10)
// 	s.updateChannels[playerID] = ch
// 	return ch, nil
// }
