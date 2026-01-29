package tests

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"wallet_service/internal/wallet"

	"github.com/go-jose/go-jose/v4/testutils/assert"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	dbConnStr = "postgres://pam_user:pam_pass@localhost:5433/pam_db?sslmode=disable"
)

var db *gorm.DB

func init() {
	var err error
	db, err = gorm.Open(postgres.Open(dbConnStr), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		fmt.Println("Failed to connect to database")
		return
	}
	err = db.AutoMigrate(&wallet.Wallet{}, &wallet.Transaction{})
	if err != nil {
		fmt.Println("Failed to migrate database")
		return
	}
}

func setUpWallet(t *testing.T, balance decimal.Decimal) *wallet.Wallet {
	if db == nil {
		t.Skip("Database connection not initialized")
	}

	repo := wallet.NewWalletRepositoryImpl(db)
	playerID := uuid.NewString()
	w, err := repo.CreateWallet(context.Background(), playerID, "main", "USD")
	require.NoError(t, err)
	if balance.GreaterThan(decimal.Zero) {
		transaction := &wallet.Transaction{
			WalletID:        w.WalletID,
			Amount:          balance,
			TransactionType: "credit",
			ReferenceID:     uuid.NewString(),
		}
		err = repo.Credit(context.Background(), transaction)
		assert.NoError(t, err)
		w.Balance = balance
	}

	return w

}

func TestConcurrentDebits(t *testing.T) {

	intialBalance := decimal.NewFromInt(50)
	w := setUpWallet(t, intialBalance)
	repo := wallet.NewWalletRepositoryImpl(db)
	service := wallet.NewService(repo)

	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	failCount := 0

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tx := wallet.TransactionRequest{
				PlayerID:        w.PlayerID,
				WalletType:      "main",
				TransactionType: "withdrawal",
				Amount:          decimal.NewFromInt(10),
				ReferenceID:     uuid.NewString(),
				Currency:        "USD",
			}
			_, err := service.ProcessTransaction(context.Background(), tx)
			mu.Lock()
			if err != nil {
				failCount++
			} else {
				successCount++
			}
			mu.Unlock()
		}()

	}
	wg.Wait()
	require.Equal(t, 5, successCount, "successCount")
	require.Equal(t, 5, failCount, "failCount")

	finalWallet, err := service.GetBalance(context.Background(), w.PlayerID, "main", "USD")
	require.NoError(t, err)
	require.Equal(t, decimal.NewFromInt(0), finalWallet.Balance, "finalBalance")

}

func TestIdempotentTransaction(t *testing.T) {
	w := setUpWallet(t, decimal.NewFromInt(50))
	repo := wallet.NewWalletRepositoryImpl(db)
	service := wallet.NewService(repo)
	refId := uuid.NewString()
	tx := wallet.TransactionRequest{
		PlayerID:        w.PlayerID,
		WalletType:      "main",
		TransactionType: "withdrawal",
		Amount:          decimal.NewFromInt(10),
		ReferenceID:     refId,
		Currency:        "USD",
	}
	res1, err := service.ProcessTransaction(context.Background(), tx)
	assert.NoError(t, err)

	res2, err := service.ProcessTransaction(context.Background(), tx)
	assert.NoError(t, err)
	res3, err := service.ProcessTransaction(context.Background(), tx)
	assert.NoError(t, err)

	require.Equal(t, res1.TransactionID, res2.TransactionID)
	require.Equal(t, res2.TransactionID, res3.TransactionID)

	finalWallet, err := service.GetBalance(context.Background(), w.PlayerID, "main", "USD")
	require.NoError(t, err)
	require.Equal(t, decimal.NewFromInt(40), finalWallet.Balance, "finalBalance")

}

func TestRaceCondition(t *testing.T) {

	w := setUpWallet(t, decimal.NewFromInt(50))
	repo := wallet.NewWalletRepositoryImpl(db)
	service := wallet.NewService(repo)

	var wg sync.WaitGroup
	var mu sync.Mutex
	successDebits := 0
	succCredits := 0

	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			tx := wallet.TransactionRequest{
				PlayerID:        w.PlayerID,
				WalletType:      "main",
				TransactionType: "withdrawal",
				Amount:          decimal.NewFromInt(1),
				ReferenceID:     uuid.NewString(),
				Currency:        "USD",
			}
			_, err := service.ProcessTransaction(context.Background(), tx)
			mu.Lock()
			if err == nil {
				successDebits++
			}
			mu.Unlock()
		}()
		go func() {
			defer wg.Done()
			tx := wallet.TransactionRequest{
				PlayerID:        w.PlayerID,
				WalletType:      "main",
				TransactionType: "deposit",
				Amount:          decimal.NewFromInt(1),
				ReferenceID:     uuid.NewString(),
				Currency:        "USD",
			}
			_, err := service.ProcessTransaction(context.Background(), tx)
			mu.Lock()
			if err == nil {
				succCredits++
			}
			mu.Unlock()

		}()
	}

	wg.Wait()
	finalWallet, err := service.GetBalance(context.Background(), w.PlayerID, "main", "USD")
	require.NoError(t, err)
	exactBalance := decimal.NewFromInt(50).Add(decimal.NewFromInt(int64(succCredits - successDebits)))
	require.Equal(t, exactBalance, finalWallet.Balance, "finalBalance")

}
