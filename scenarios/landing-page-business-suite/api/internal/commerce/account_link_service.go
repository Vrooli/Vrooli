package commerce

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// AccountLinkService owns the customer-to-user identity mapping created by
// Stripe checkout and maintained across customer email changes.
type AccountLinkService struct {
	db StripeStore
}

// NewAccountLinkService creates the commerce service responsible for Stripe
// customer identity persistence.
func NewAccountLinkService(db StripeStore) *AccountLinkService {
	return &AccountLinkService{db: db}
}

// LinkUserToStripeCustomer creates or updates a user record with the Stripe
// customer ID supplied by a completed checkout.
func (s *AccountLinkService) LinkUserToStripeCustomer(email, customerID string) error {
	email = normalizeEmail(email)
	customerID = strings.TrimSpace(customerID)
	if email == "" || customerID == "" {
		return errors.New("email and customer ID are required")
	}

	_, err := s.db.Exec(`
		INSERT INTO users (email, stripe_customer_id, email_verified)
		VALUES ($1, $2, FALSE)
		ON CONFLICT (email) DO UPDATE SET
			stripe_customer_id = $2,
			updated_at = NOW()
	`, email, customerID)
	if err != nil {
		return fmt.Errorf("link user to stripe customer: %w", err)
	}
	return nil
}

// LookupCustomerID finds the most recently updated Stripe customer for an
// email address or customer ID.
func (s *AccountLinkService) LookupCustomerID(userIdentity string) string {
	if strings.TrimSpace(userIdentity) == "" {
		return ""
	}

	var customerID sql.NullString
	err := s.db.QueryRow(`
		SELECT customer_id
		FROM subscriptions
		WHERE customer_email = $1 OR customer_id = $1
		ORDER BY updated_at DESC
		LIMIT 1
	`, normalizeEmail(userIdentity)).Scan(&customerID)
	if err != nil || !customerID.Valid {
		return ""
	}
	return customerID.String
}

// MigrateCustomerEmail updates all commerce records atomically. When an
// account already exists at the new address, it preserves the used-intro flag
// and avoids creating duplicate user records.
func (s *AccountLinkService) MigrateCustomerEmail(ctx context.Context, oldEmail, newEmail, customerID string) error {
	normalizedOld := normalizeEmail(oldEmail)
	normalizedNew := normalizeEmail(newEmail)
	if normalizedOld == "" || normalizedNew == "" {
		return errors.New("both old and new emails are required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin email migration transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var oldHasUsedIntro bool
	var oldStripeCustomerID sql.NullString
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(has_used_intro, FALSE), stripe_customer_id FROM users WHERE email = $1
	`, normalizedOld).Scan(&oldHasUsedIntro, &oldStripeCustomerID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check old email intro status: %w", err)
	}

	var newEmailExists bool
	err = tx.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)
	`, normalizedNew).Scan(&newEmailExists)
	if err != nil {
		return fmt.Errorf("check new email exists: %w", err)
	}

	if newEmailExists {
		_, err = tx.ExecContext(ctx, `
			UPDATE users SET
				has_used_intro = COALESCE(has_used_intro, FALSE) OR $2,
				stripe_customer_id = COALESCE(stripe_customer_id, $3),
				updated_at = NOW()
			WHERE email = $1
		`, normalizedNew, oldHasUsedIntro, customerID)
		if err != nil {
			return fmt.Errorf("update new email user intro flag: %w", err)
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM users WHERE email = $1`, normalizedOld)
		if err != nil {
			return fmt.Errorf("delete old email user: %w", err)
		}
	} else {
		_, err = tx.ExecContext(ctx, `
			UPDATE users SET email = $1, updated_at = NOW()
			WHERE email = $2 OR stripe_customer_id = $3
		`, normalizedNew, normalizedOld, customerID)
		if err != nil {
			return fmt.Errorf("migrate users table: %w", err)
		}
	}

	updates := []struct {
		query string
		args  []any
		label string
	}{
		{`UPDATE subscriptions SET customer_email = $1, updated_at = NOW() WHERE customer_email = $2 OR customer_id = $3`, []any{normalizedNew, normalizedOld, customerID}, "migrate subscriptions table"},
		{`UPDATE credit_wallets SET customer_email = $1, updated_at = NOW() WHERE customer_email = $2`, []any{normalizedNew, normalizedOld}, "migrate credit_wallets table"},
		{`UPDATE credit_transactions SET customer_email = $1 WHERE customer_email = $2`, []any{normalizedNew, normalizedOld}, "migrate credit_transactions table"},
		{`UPDATE intro_coupon_usage SET email = $1 WHERE email = $2 OR stripe_customer_id = $3`, []any{normalizedNew, normalizedOld, customerID}, "migrate intro_coupon_usage table"},
	}
	for _, update := range updates {
		if _, err := tx.ExecContext(ctx, update.query, update.args...); err != nil {
			return fmt.Errorf("%s: %w", update.label, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit email migration: %w", err)
	}
	return nil
}
