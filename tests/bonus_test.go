package tests

import (
	"context"
	"sync"
	"testing"
	"time"
	"wallet_service/internal/wallet/bonus"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestConcurrentWagering(t *testing.T) {
	service := bonus.NewBonusService()
	playerId := "player1"
	bonusID := "bonus1"

	activeBonus := &bonus.PlayerBonus{
		PlayerBonusID:     bonusID,
		PlayerID:          playerId,
		BonusID:           bonusID,
		Status:            "active",
		BonusAmount:       decimal.NewFromFloat(100.0),
		WageringRequired:  decimal.NewFromFloat(1000.0),
		WageringCompleted: decimal.Zero,
		ExpiresAt:         time.Now().Add(24 * time.Hour),
	}
	service.AddActiveBonus(activeBonus)

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			bet := bonus.BetEvent{
				BetID:     uuid.New().String(),
				PlayerID:  playerId,
				GameID:    "slots",
				BetAmount: decimal.NewFromFloat(10.0),
				Timestamp: time.Now(),
			}
			err := service.ProcessBetWagering(context.Background(), bet)
			assert.NoError(t, err)
		}()

	}
	wg.Wait()

	progress, err := service.GetWageringProgress(context.Background(), bonusID, playerId)
	assert.NoError(t, err)
	assert.True(t, progress.Completed, "completed should be true")
	assert.True(t, progress.WageringCompleted.Equal(decimal.NewFromFloat(1000.0)), "wageringCompleted should be 1000")
	assert.Equal(t, 100.0, progress.PercentageComplete, "percentageComplete should be 100")

}
