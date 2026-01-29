// package tests

// import (
// 	"context"
// 	"sync"
// 	"testing"
// 	"time"
// 	"wallet_service/internal/wallet/bonus"

// 	"github.com/google/uuid"
// 	"github.com/shopspring/decimal"
// 	"github.com/stretchr/testify/assert"
// )

// func TestConcurrentWagering(t *testing.T) {
// 	service := bonus.NewBonusService()
// 	playerId := "player1"
// 	bonusID := "bonus1"

// 	activeBonus := &bonus.PlayerBonus{
// 		PlayerBonusID:     bonusID,
// 		PlayerID:          playerId,
// 		BonusID:           bonusID,
// 		Status:            "active",
// 		BonusAmount:       decimal.NewFromFloat(100.0),
// 		WageringRequired:  decimal.NewFromFloat(1000.0),
// 		WageringCompleted: decimal.Zero,
// 		ExpiresAt:         time.Now().Add(24 * time.Hour),
// 	}
// 	service.AddActiveBonus(activeBonus)

// 	var wg sync.WaitGroup

// 	for i := 0; i < 100; i++ {
// 		wg.Add(1)
// 		go func() {
// 			defer wg.Done()

// 			bet := bonus.BetEvent{
// 				BetID:     uuid.New().String(),
// 				PlayerID:  playerId,
// 				GameID:    "slots",
// 				BetAmount: decimal.NewFromFloat(10.0),
// 				Timestamp: time.Now(),
// 			}
// 			err := service.ProcessBetWagering(context.Background(), bet)
// 			assert.NoError(t, err)
// 		}()

// 	}
// 	wg.Wait()

// 	progress, err := service.GetWageringProgress(context.Background(), bonusID, playerId)
// 	assert.NoError(t, err)
// 	assert.True(t, progress.Completed, "completed should be true")
// 	assert.True(t, progress.WageringCompleted.Equal(decimal.NewFromFloat(1000.0)), "wageringCompleted should be 1000")
// 	assert.Equal(t, 100.0, progress.PercentageComplete, "percentageComplete should be 100")

// }

// Package tests provides integration tests for the bonus wagering service
package tests

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"wallet_service/internal/bonus"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupBonusTest creates test dependencies
func setupBonusTest(t *testing.T) (*bonus.BonusRepositoryImpl, *bonus.BonusService, error) {

	db, err := gorm.Open(postgres.Open(dbConnStr), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		fmt.Println("Failed to connect to database")
		return nil, nil, err
	}
	// err = db.AutoMigrate(&bonus.PlayerBonus{}, &bonus.BetEvent{})
	// if err != nil {
	// 	fmt.Println("Failed to migrate database")
	// 	return nil, nil, err
	// }

	repo := bonus.NewBonusRepository(db)
	service := bonus.NewBonusService(db, repo)
	return repo, service, nil
}

// TestConcurrentWageringUpdates tests that concurrent wagering updates are handled correctly
// Multiple bets updating wagering progress simultaneously
// Expected: Accurate final wagering amount
func TestConcurrentWageringUpdates(t *testing.T) {
	_, service, err := setupBonusTest(t)
	if err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}
	// defer cleanup()

	ctx := context.Background()
	playerID := uuid.New().String()

	// Create bonus with $100 amount and 10x wagering requirement = $1000 required
	playerBonus, err := service.CreatePlayerBonus(
		ctx,
		playerID,
		uuid.New().String(),
		decimal.NewFromInt(100),         // $100 bonus
		decimal.NewFromInt(10),          // 10x wagering
		time.Now().Add(30*24*time.Hour), // Expires in 30 days
	)
	if err != nil {
		t.Fatalf("Failed to create bonus: %v", err)
	}

	// Slots game ID (100% contribution)
	slotsGameID := "11111111-1111-1111-1111-111111111111"

	// Launch 10 goroutines placing $50 bets each = $500 total wagering
	numBets := 10
	betAmount := decimal.NewFromInt(50)

	var wg sync.WaitGroup
	var successCount int32

	for i := 0; i < numBets; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			betEvent := bonus.BetEvent{
				BetID:     "concurrent-wagering-" + uuid.New().String(),
				PlayerID:  playerID,
				GameID:    slotsGameID,
				BetAmount: betAmount,
				Timestamp: time.Now(),
			}

			err := service.ProcessBetWagering(ctx, betEvent)
			if err != nil {
				t.Errorf("Bet %d failed: %v", index, err)
			} else {
				atomic.AddInt32(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()

	// All bets should succeed
	if successCount != int32(numBets) {
		t.Errorf("Expected %d successful bets, got %d", numBets, successCount)
	}

	// Verify wagering progress: 10 bets * $50 * 100% = $500
	progress, err := service.GetWageringProgress(ctx, playerID, playerBonus.PlayerBonusID)
	if err != nil {
		t.Fatalf("Failed to get progress: %v", err)
	}

	expectedWagering := decimal.NewFromInt(500)
	if !progress.WageringCompleted.Equal(expectedWagering) {
		t.Errorf("Expected wagering $%s, got $%s", expectedWagering.String(), progress.WageringCompleted.String())
	}

	t.Logf("Concurrent wagering test passed: $%s wagered", progress.WageringCompleted.String())
}

// TestWageringCompletion tests that bonus is marked complete when wagering requirement is met
// TestWageringIdempotency tests that same bet is not counted twice
func TestWageringIdempotency(t *testing.T) {
	_, service, err := setupBonusTest(t)
	if err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}

	ctx := context.Background()
	playerID := uuid.New().String()

	// Create bonus
	playerBonus, err := service.CreatePlayerBonus(
		ctx,
		playerID,
		uuid.New().String(),
		decimal.NewFromInt(100),
		decimal.NewFromInt(10),
		time.Now().Add(30*24*time.Hour),
	)
	if err != nil {
		t.Fatalf("Failed to create bonus: %v", err)
	}

	// Use same bet ID for multiple attempts
	betID := "idempotent-bet-" + uuid.New().String()
	slotsGameID := "11111111-1111-1111-1111-111111111111"

	// Process same bet 3 times
	for i := 0; i < 3; i++ {
		betEvent := bonus.BetEvent{
			BetID:     betID, // Same bet ID
			PlayerID:  playerID,
			GameID:    slotsGameID,
			BetAmount: decimal.NewFromInt(50),
			Timestamp: time.Now(),
		}
		// First attempt should succeed, subsequent should be silently skipped
		_ = service.ProcessBetWagering(ctx, betEvent)
	}

	// Verify wagering is only $50 (not $150)
	progress, err := service.GetWageringProgress(ctx, playerID, playerBonus.PlayerBonusID)
	if err != nil {
		t.Fatalf("Failed to get progress: %v", err)
	}

	expectedWagering := decimal.NewFromInt(50)
	if !progress.WageringCompleted.Equal(expectedWagering) {
		t.Errorf("Expected wagering $%s, got $%s (idempotency failed)",
			expectedWagering.String(), progress.WageringCompleted.String())
	}

	t.Logf("Wagering idempotency test passed: $%s wagered", progress.WageringCompleted.String())
}
