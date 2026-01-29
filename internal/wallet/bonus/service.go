package bonus

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

type BonusWageringService interface {
	ProcessBetWagering()
}

type BonusService struct {
	bonuses        map[string]*PlayerBonus
	gameRules      map[string]GameContribution
	mu             sync.RWMutex
	updateChannels map[string]chan WageringUpdate
	channelsMu     sync.RWMutex
}

func NewBonusService() *BonusService {
	s := &BonusService{
		bonuses:        make(map[string]*PlayerBonus),
		gameRules:      make(map[string]GameContribution),
		updateChannels: make(map[string]chan WageringUpdate),
	}
	s.gameRules = map[string]GameContribution{
		"slots":      {GameID: "slots", GameType: "slots", Contribution: decimal.NewFromFloat(1.0)},
		"black_jack": {GameID: "black_jack", GameType: "table_games", Contribution: decimal.NewFromFloat(0.1)},
	}
	return s
}

func (s *BonusService) AddActiveBonus(bonus *PlayerBonus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bonuses[bonus.PlayerID] = bonus

}

func (s *BonusService) ProcessBetWagering(ctx context.Context, bet BetEvent) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	bonus, exists := s.bonuses[bet.PlayerID]
	if !exists || bonus.Status != "active" {
		return nil
	}

	if time.Now().After(bonus.ExpiresAt) {
		bonus.Status = "expired"
		return nil
	}

	rule, ok := s.gameRules[bet.GameID]
	contributionpct := decimal.Zero
	if ok {
		contributionpct = rule.Contribution
	}
	contributionAmount := bet.BetAmount.Mul(contributionpct)
	bonus.WageringCompleted = bonus.WageringCompleted.Add(contributionAmount)
	if bonus.WageringCompleted.GreaterThanOrEqual(bonus.WageringRequired) {
		bonus.Status = "completed"
		s.notifyPlayer(bet.PlayerID, bonus, true)
	}
	return nil

}

func (s *BonusService) notifyPlayer(playerID string, bonus *PlayerBonus, isCompleted bool) {
	s.channelsMu.Lock()
	defer s.channelsMu.Unlock()
	ch, ok := s.updateChannels[playerID]
	if ok {
		pct, _ := bonus.WageringCompleted.Div(bonus.WageringRequired).Float64()
		egleft, _ := bonus.WageringRequired.Sub(bonus.WageringCompleted).Float64()
		update := WageringUpdate{
			PlayerID:     playerID,
			Percentage:   pct * 100,
			WageringLeft: egleft,
			IsCompleted:  isCompleted,
		}
		ch <- update
	}

}

func (s *BonusService) GetWageringProgress(ctx context.Context, bonusID string, playerID string) (*WageringProgress, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bonus, ok := s.bonuses[playerID]
	if !ok {
		return nil, errors.New("bonus not found")
	}
	// fmt.Println("bonus ", bonus)
	pct, _ := bonus.WageringCompleted.Div(bonus.WageringRequired).Float64()
	return &WageringProgress{
		PlayerBonusID:      bonus.PlayerBonusID,
		WageringRequired:   bonus.WageringRequired,
		WageringCompleted:  bonus.WageringCompleted,
		PercentageComplete: pct * 100,
		Completed:          bonus.Status == "completed",
	}, nil

}

func (s *BonusService) SubscribeToWageringUpdates(playerID string) (<-chan WageringUpdate, error) {
	s.channelsMu.Lock()
	defer s.channelsMu.Unlock()

	ch := make(chan WageringUpdate, 10)
	s.updateChannels[playerID] = ch
	return ch, nil
}
