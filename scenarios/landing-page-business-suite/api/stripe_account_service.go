package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// --- StripeAccountLinkService Interface Implementation ---
// This file contains all account linking operations: LinkUserToStripeCustomer,
// LookupCustomerID, MigrateCustomerEmail.

// LinkUserToStripeCustomer associates a user email with a Stripe customer ID.
// This is the public interface method that delegates to the internal linkUserToStripeCustomer.
func (s *StripeService) LinkUserToStripeCustomer(email, customerID string) error {
	return s.linkUserToStripeCustomer(email, customerID)
}

// LookupCustomerID finds the Stripe customer ID for a user (by email or customer ID).
// This is the public interface method that delegates to the internal lookupCustomerID.
func (s *StripeService) LookupCustomerID(userIdentity string) string {
	return s.lookupCustomerID(userIdentity)
}

// MigrateCustomerEmail updates all tables when a customer's email changes.
func (s *StripeService) MigrateCustomerEmail(ctx context.Context, oldEmail, newEmail, customerID string) error {
	normalizedOld := NormalizeEmail(oldEmail)
	normalizedNew := NormalizeEmail(newEmail)

	if normalizedOld == "" || normalizedNew == "" {
		return errors.New("both old and new emails are required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin email migration transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Get the old email's has_used_intro flag and customer ID
	var oldHasUsedIntro bool
	var oldStripeCustomerID sql.NullString
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(has_used_intro, FALSE), stripe_customer_id FROM users WHERE email = $1
	`, normalizedOld).Scan(&oldHasUsedIntro, &oldStripeCustomerID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("check old email intro status: %w", err)
	}

	// Check if a user with the new email already exists
	var newEmailExists bool
	err = tx.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)
	`, normalizedNew).Scan(&newEmailExists)
	if err != nil {
		return fmt.Errorf("check new email exists: %w", err)
	}

	if newEmailExists {
		// New email user already exists - merge intro flags and delete old user
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

		// Delete the old email user to avoid duplicates
		_, err = tx.ExecContext(ctx, `
			DELETE FROM users WHERE email = $1
		`, normalizedOld)
		if err != nil {
			return fmt.Errorf("delete old email user: %w", err)
		}
	} else {
		// New email doesn't exist - update old email to new email
		_, err = tx.ExecContext(ctx, `
			UPDATE users SET
				email = $1,
				updated_at = NOW()
			WHERE email = $2 OR stripe_customer_id = $3
		`, normalizedNew, normalizedOld, customerID)
		if err != nil {
			return fmt.Errorf("migrate users table: %w", err)
		}
	}

	// Update subscriptions table
	_, err = tx.ExecContext(ctx, `
		UPDATE subscriptions SET customer_email = $1, updated_at = NOW()
		WHERE customer_email = $2 OR customer_id = $3
	`, normalizedNew, normalizedOld, customerID)
	if err != nil {
		return fmt.Errorf("migrate subscriptions table: %w", err)
	}

	// Update credit_wallets table
	_, err = tx.ExecContext(ctx, `
		UPDATE credit_wallets SET customer_email = $1, updated_at = NOW()
		WHERE customer_email = $2
	`, normalizedNew, normalizedOld)
	if err != nil {
		return fmt.Errorf("migrate credit_wallets table: %w", err)
	}

	// Update credit_transactions table
	_, err = tx.ExecContext(ctx, `
		UPDATE credit_transactions SET customer_email = $1
		WHERE customer_email = $2
	`, normalizedNew, normalizedOld)
	if err != nil {
		return fmt.Errorf("migrate credit_transactions table: %w", err)
	}

	// Update intro_coupon_usage table
	_, err = tx.ExecContext(ctx, `
		UPDATE intro_coupon_usage SET email = $1
		WHERE email = $2 OR stripe_customer_id = $3
	`, normalizedNew, normalizedOld, customerID)
	if err != nil {
		return fmt.Errorf("migrate intro_coupon_usage table: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit email migration: %w", err)
	}

	logStructured("customer_email_migrated", map[string]interface{}{
		"level":       "info",
		"old_email":   normalizedOld,
		"new_email":   normalizedNew,
		"customer_id": customerID,
	})

	return nil
}

// linkUserToStripeCustomer creates or updates a user record with the Stripe customer ID.
// This is called when a checkout completes to link the user account to their Stripe customer.
func (s *StripeService) linkUserToStripeCustomer(email, customerID string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	customerID = strings.TrimSpace(customerID)

	if email == "" || customerID == "" {
		return errors.New("email and customer ID are required")
	}

	// Upsert user with stripe_customer_id
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

	logStructured("stripe_customer_linked", map[string]interface{}{
		"level":       "info",
		"email":       email,
		"customer_id": customerID,
	})

	return nil
}

// lookupCustomerID finds the Stripe customer ID for a user from the subscriptions table.
func (s *StripeService) lookupCustomerID(user string) string {
	if strings.TrimSpace(user) == "" {
		return ""
	}
	// Normalize email for case-insensitive lookup
	normalizedUser := NormalizeEmail(user)
	var customerID sql.NullString
	err := s.db.QueryRow(`
		SELECT customer_id
		FROM subscriptions
		WHERE customer_email = $1 OR customer_id = $1
		ORDER BY updated_at DESC
		LIMIT 1
	`, normalizedUser).Scan(&customerID)
	if err != nil {
		return ""
	}
	if customerID.Valid {
		return customerID.String
	}
	return ""
}
