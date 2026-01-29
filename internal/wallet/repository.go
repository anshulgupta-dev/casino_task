package wallet

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrWalletNotFound    = errors.New("wallet not found")
	ErrOptimisticLock    = errors.New("optimistic lock error")
)

type WalletRepository interface {
	GetBalance(ctx context.Context, playerId string, walletType string, currency string) (*Wallet, error)
	GetTransactionByReference(ctx context.Context, referenceId string, transactionType string) (*Transaction, error)
	CreateWallet(ctx context.Context, playerId string, walletType string, currency string) (*Wallet, error)
	Credit(ctx context.Context, transaction *Transaction) error
	Debit(ctx context.Context, transaction *Transaction) error
}

type WalletRepositoryImpl struct {
	db *gorm.DB
}

func NewWalletRepositoryImpl(db *gorm.DB) WalletRepository {
	return &WalletRepositoryImpl{db: db}
}

func (r *WalletRepositoryImpl) GetBalance(ctx context.Context, playerId string, walletType string, currency string) (*Wallet, error) {

	var w Wallet
	err := r.db.WithContext(ctx).Where("player_id = ? AND wallet_type = ? AND currency = ?", playerId, walletType, currency).First(&w).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWalletNotFound
		}
		return nil, err
	}
	return &w, nil
}

func (r *WalletRepositoryImpl) GetTransactionByReference(ctx context.Context, referenceId string, transactionType string) (*Transaction, error) {
	var t Transaction
	err := r.db.WithContext(ctx).Where("reference_id = ? AND transaction_type = ?", referenceId, transactionType).First(&t).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *WalletRepositoryImpl) CreateWallet(ctx context.Context, playerId string, walletType string, currency string) (*Wallet, error) {
	w := Wallet{
		WalletID:   uuid.New().String(),
		PlayerID:   playerId,
		WalletType: walletType,
		Currency:   currency,
	}

	err := r.db.WithContext(ctx).Create(&w).Error
	if err != nil {
		return nil, err
	}
	return &w, nil

}

func (r *WalletRepositoryImpl) Debit(ctx context.Context, tx *Transaction) error {
	return r.db.WithContext(ctx).Transaction(func(dbtx *gorm.DB) error {
		var w Wallet
		if err := dbtx.Where("wallet_id = ?", tx.WalletID).First(&w).Error; err != nil {
			return err
		}

		if w.Balance.LessThan(tx.Amount) {
			return ErrInsufficientFunds
		}

		newBalance := w.Balance.Sub(tx.Amount)

		result := dbtx.Model(&Wallet{}).Where("wallet_id = ? AND version = ?", w.WalletID, w.Version).
			Updates(map[string]interface{}{
				"balance":    newBalance,
				"version":    gorm.Expr("version + 1"),
				"updated_at": time.Now(),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrOptimisticLock
		}

		tx.TransactionID = uuid.New().String()
		tx.BalanceBefore = w.Balance
		tx.BalanceAfter = newBalance
		tx.Status = "completed"
		now := time.Now()
		tx.CompletedAt = &now

		if err := dbtx.Create(tx).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *WalletRepositoryImpl) Credit(ctx context.Context, tx *Transaction) error {
	return r.db.WithContext(ctx).Transaction(func(dbtx *gorm.DB) error {
		var w Wallet
		if err := dbtx.Where("wallet_id = ?", tx.WalletID).First(&w).Error; err != nil {
			return err
		}
		newBalance := w.Balance.Add(tx.Amount)

		result := dbtx.Model(&Wallet{}).Where("wallet_id = ? AND version = ?", w.WalletID, w.Version).
			Updates(map[string]interface{}{
				"balance":    newBalance,
				"version":    gorm.Expr("version + 1"),
				"updated_at": time.Now(),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrOptimisticLock
		}

		tx.TransactionID = uuid.New().String()
		tx.BalanceBefore = w.Balance
		tx.BalanceAfter = newBalance
		tx.Status = "completed"
		now := time.Now()
		tx.CompletedAt = &now

		if err := dbtx.Create(tx).Error; err != nil {
			return err
		}

		return nil
	})

}
