package wallet

import (
	"context"
	"errors"
	"time"
)

const (
	MaxRetries = 3
	RetryDelay = 10 * time.Millisecond
)

type WalletService interface {
	ProcessTransaction()
	GetBalance(ctx context.Context, playerId string, game string, currency string) (*Wallet, error)
}

type Service struct {
	repo WalletRepository
}

func NewService(repo WalletRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetBalance(ctx context.Context, playerId string, game string, currency string) (*Wallet, error) {
	return s.repo.GetBalance(ctx, playerId, game, currency)

}

func (s *Service) ProcessTransaction(ctx context.Context, req TransactionRequest) (*TransactionResponse, error) {
	//idempotency check
	existingTx, err := s.repo.GetTransactionByReference(ctx, req.ReferenceID, req.TransactionType)
	if err != nil {
		return nil, err
	}
	if existingTx != nil {
		return &TransactionResponse{
			TransactionID: existingTx.TransactionID,
			Balance:       existingTx.BalanceAfter,
			Status:        existingTx.Status,
		}, nil
	}

	wallet, err := s.repo.GetBalance(ctx, req.PlayerID, req.WalletType, req.Currency)
	if err != nil {
		if err == ErrWalletNotFound {
			if req.TransactionType == "withdrawal" || req.TransactionType == "bet" {
				return nil, ErrInsufficientFunds
			}
			wallet, err = s.repo.CreateWallet(ctx, req.PlayerID, req.WalletType, req.Currency)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	tx := &Transaction{
		WalletID:        wallet.WalletID,
		PlayerID:        req.PlayerID,
		TransactionType: req.TransactionType,
		Amount:          req.Amount,
		ReferenceID:     req.ReferenceID,
	}

	for i := 0; i < MaxRetries; i++ {
		if req.TransactionType == "deposit" || req.TransactionType == "win" {
			err = s.repo.Credit(ctx, tx)
		} else if req.TransactionType == "withdrawal" || req.TransactionType == "bet" {
			err = s.repo.Debit(ctx, tx)
		} else {
			return nil, errors.New("Invalid Transaction Type")
		}
		if err == nil {
			return &TransactionResponse{
				TransactionID: tx.TransactionID,
				Balance:       tx.BalanceAfter,
				Status:        tx.Status,
			}, nil
		}
		if err == ErrOptimisticLock {
			time.Sleep(RetryDelay)
			continue
		}
		return nil, err

	}
	return nil, err
}
