package commerce

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// seam: CreditWallet keeps credit ledger mutation independent of Stripe webhook
// parsing and API composition.
type CreditWallet interface {
	AddCredits(customerEmail string, amount int64, txnType, stripeEventID string, metadata map[string]interface{}) error
	ConsumeCredits(ctx context.Context, email string, amount int64, reason string, metadata map[string]interface{}) error
	ConsumeCreditsIdempotent(ctx context.Context, email string, amount int64, reason, idempotencyKey string, metadata map[string]interface{}) error
	Balance(email string) (int64, error)
}

// CreditWalletService owns credit-wallet balances and their immutable ledger.
// A Stripe event ID is the replay key for provider-driven mutations; a caller
// supplied idempotency key is the replay key for credit consumption.
type CreditWalletService struct {
	db StripeStore
}

func NewCreditWalletService(db StripeStore) *CreditWalletService {
	return &CreditWalletService{db: db}
}

var _ CreditWallet = (*CreditWalletService)(nil)

// AddCredits records an additive ledger entry and updates the wallet in the
// same transaction. Replaying a non-empty event ID is a successful no-op.
func (s *CreditWalletService) AddCredits(customerEmail string, amount int64, txnType, stripeEventID string, metadata map[string]interface{}) error {
	if customerEmail == "" || amount <= 0 {
		return nil
	}

	metadataJSON, _ := json.Marshal(metadata)
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin credit transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var eventID any
	if stripeEventID != "" {
		eventID = stripeEventID
	}
	result, err := tx.Exec(`
		INSERT INTO credit_transactions (customer_email, amount_credits, transaction_type, stripe_event_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (stripe_event_id) DO NOTHING
	`, normalizeEmail(customerEmail), amount, txnType, eventID, string(metadataJSON))
	if err != nil {
		return fmt.Errorf("insert credit transaction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check credit transaction rows: %w", err)
	}
	if rowsAffected == 0 {
		return nil
	}

	_, err = tx.Exec(`
		INSERT INTO credit_wallets (customer_email, balance_credits, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (customer_email) DO UPDATE
		SET balance_credits = credit_wallets.balance_credits + $2, updated_at = NOW()
	`, normalizeEmail(customerEmail), amount)
	if err != nil {
		return fmt.Errorf("update credit wallet: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit credit transaction: %w", err)
	}
	return nil
}

// ConsumeCredits deducts a wallet balance for an ordinary request.
func (s *CreditWalletService) ConsumeCredits(ctx context.Context, email string, amount int64, reason string, metadata map[string]interface{}) error {
	return s.ConsumeCreditsIdempotent(ctx, email, amount, reason, "", metadata)
}

// ConsumeCreditsIdempotent deducts a wallet balance. The optional idempotency
// key makes retries a successful no-op after the first committed deduction.
func (s *CreditWalletService) ConsumeCreditsIdempotent(ctx context.Context, email string, amount int64, reason, idempotencyKey string, metadata map[string]interface{}) error {
	if email == "" || amount <= 0 {
		return errors.New("email and positive amount are required")
	}

	metadataJSON, _ := json.Marshal(metadata)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin consume credits transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if idempotencyKey != "" {
		var exists bool
		err = tx.QueryRowContext(ctx, `
			SELECT EXISTS(SELECT 1 FROM credit_transactions WHERE stripe_event_id = $1)
		`, idempotencyKey).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check idempotency key: %w", err)
		}
		if exists {
			return nil
		}
	}

	var balance int64
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(balance_credits, 0) FROM credit_wallets WHERE customer_email = $1 FOR UPDATE
	`, normalizeEmail(email)).Scan(&balance)
	if err == sql.ErrNoRows {
		return errors.New("no credit wallet found for user")
	}
	if err != nil {
		return fmt.Errorf("check credit balance: %w", err)
	}
	if balance < amount {
		return fmt.Errorf("insufficient credits: have %d, need %d", balance, amount)
	}

	var eventID any
	if idempotencyKey != "" {
		eventID = idempotencyKey
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO credit_transactions (customer_email, amount_credits, transaction_type, stripe_event_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, normalizeEmail(email), -amount, reason, eventID, string(metadataJSON))
	if err != nil {
		return fmt.Errorf("insert consumption transaction: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE credit_wallets SET balance_credits = balance_credits - $1, updated_at = NOW()
		WHERE customer_email = $2
	`, amount, normalizeEmail(email))
	if err != nil {
		return fmt.Errorf("update credit wallet: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit consume credits transaction: %w", err)
	}
	return nil
}

func (s *CreditWalletService) Balance(email string) (int64, error) {
	if email == "" {
		return 0, errors.New("email is required")
	}

	var balance int64
	err := s.db.QueryRow(`
		SELECT COALESCE(balance_credits, 0) FROM credit_wallets WHERE customer_email = $1
	`, normalizeEmail(email)).Scan(&balance)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get credit balance: %w", err)
	}
	return balance, nil
}
