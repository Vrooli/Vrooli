package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// --- StripeCreditService Interface Implementation ---
// This file contains all credit wallet operations: AddCredits, ConsumeCredits, GetBalance.

// AddCredits adds credits to a user's wallet with idempotency protection.
// This is the public interface method that delegates to the internal addCredits.
func (s *StripeService) AddCredits(email string, amount int64, txnType, eventID string, metadata map[string]interface{}) error {
	return s.addCredits(email, amount, txnType, eventID, metadata)
}

// ConsumeCredits deducts credits from a user's wallet for a given reason.
// For idempotent credit consumption (e.g., webhook handlers), use ConsumeCreditsIdempotent instead.
func (s *StripeService) ConsumeCredits(ctx context.Context, email string, amount int64, reason string, metadata map[string]interface{}) error {
	return s.ConsumeCreditsIdempotent(ctx, email, amount, reason, "", metadata)
}

// ConsumeCreditsIdempotent deducts credits from a user's wallet with optional idempotency protection.
// If idempotencyKey is provided and a transaction with that key already exists, the operation
// returns success without deducting credits again (idempotent behavior).
// This should be used for webhook handlers to prevent double-deductions on retries.
func (s *StripeService) ConsumeCreditsIdempotent(ctx context.Context, email string, amount int64, reason, idempotencyKey string, metadata map[string]interface{}) error {
	if email == "" || amount <= 0 {
		return errors.New("email and positive amount are required")
	}

	email = NormalizeEmail(email)
	metaBytes, _ := json.Marshal(metadata)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin consume credits transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// If idempotency key is provided, check for existing transaction first
	if idempotencyKey != "" {
		var exists bool
		err = tx.QueryRowContext(ctx, `
			SELECT EXISTS(SELECT 1 FROM credit_transactions WHERE stripe_event_id = $1)
		`, idempotencyKey).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check idempotency key: %w", err)
		}
		if exists {
			logStructured("credit_consumption_already_processed", map[string]interface{}{
				"level":           "info",
				"idempotency_key": idempotencyKey,
				"customer_email":  email,
			})
			return nil
		}
	}

	// Check current balance with row-level lock
	var balance int64
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(balance_credits, 0) FROM credit_wallets WHERE customer_email = $1 FOR UPDATE
	`, email).Scan(&balance)
	if err == sql.ErrNoRows {
		return errors.New("no credit wallet found for user")
	}
	if err != nil {
		return fmt.Errorf("check credit balance: %w", err)
	}

	if balance < amount {
		return fmt.Errorf("insufficient credits: have %d, need %d", balance, amount)
	}

	// Record the consumption transaction (negative amount)
	// Include idempotency key if provided to prevent duplicate processing
	var eventIDParam interface{}
	if idempotencyKey != "" {
		eventIDParam = idempotencyKey
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO credit_transactions (customer_email, amount_credits, transaction_type, stripe_event_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, email, -amount, reason, eventIDParam, string(metaBytes))
	if err != nil {
		return fmt.Errorf("insert consumption transaction: %w", err)
	}

	// Update wallet balance
	_, err = tx.ExecContext(ctx, `
		UPDATE credit_wallets SET balance_credits = balance_credits - $1, updated_at = NOW()
		WHERE customer_email = $2
	`, amount, email)
	if err != nil {
		return fmt.Errorf("update credit wallet: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit consume credits transaction: %w", err)
	}

	logStructured("credits_consumed", map[string]interface{}{
		"level":           "info",
		"customer_email":  email,
		"amount":          amount,
		"reason":          reason,
		"idempotency_key": idempotencyKey,
	})

	return nil
}

// GetBalance returns the current credit balance for a user.
func (s *StripeService) GetBalance(email string) (int64, error) {
	if email == "" {
		return 0, errors.New("email is required")
	}

	email = NormalizeEmail(email)

	var balance int64
	err := s.db.QueryRow(`
		SELECT COALESCE(balance_credits, 0) FROM credit_wallets WHERE customer_email = $1
	`, email).Scan(&balance)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get credit balance: %w", err)
	}

	return balance, nil
}

// addCredits is the internal implementation for adding credits with idempotency protection.
func (s *StripeService) addCredits(customerEmail string, amount int64, txnType string, stripeEventID string, metadata map[string]interface{}) error {
	if customerEmail == "" || amount <= 0 {
		return nil
	}

	// Normalize email for consistency
	customerEmail = NormalizeEmail(customerEmail)

	metaBytes, _ := json.Marshal(metadata)

	// Use transaction to prevent race conditions
	// Insert transaction record FIRST - unique index on stripe_event_id prevents duplicates
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin credit transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Store stripe_event_id for idempotency (nullable)
	var eventIDParam interface{}
	if stripeEventID != "" {
		eventIDParam = stripeEventID
	}

	// Insert transaction record first - ON CONFLICT DO NOTHING makes this idempotent
	// The unique index on stripe_event_id ensures only one insert succeeds for concurrent requests
	result, err := tx.Exec(`
		INSERT INTO credit_transactions (customer_email, amount_credits, transaction_type, stripe_event_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (stripe_event_id) DO NOTHING
	`, customerEmail, amount, txnType, eventIDParam, string(metaBytes))
	if err != nil {
		return fmt.Errorf("insert credit transaction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check credit transaction rows: %w", err)
	}

	// If no rows were affected, this event was already processed (idempotent success)
	if rowsAffected == 0 {
		logStructured("credit_topup_already_processed", map[string]interface{}{
			"level":           "info",
			"stripe_event_id": stripeEventID,
			"customer_email":  customerEmail,
		})
		return nil
	}

	// Only update wallet if transaction record was inserted successfully
	_, err = tx.Exec(`
		INSERT INTO credit_wallets (customer_email, balance_credits, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (customer_email) DO UPDATE
		SET balance_credits = credit_wallets.balance_credits + $2, updated_at = NOW()
	`, customerEmail, amount)
	if err != nil {
		return fmt.Errorf("update credit wallet: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit credit transaction: %w", err)
	}

	logStructured("credits_added", map[string]interface{}{
		"level":            "info",
		"customer_email":   customerEmail,
		"amount":           amount,
		"transaction_type": txnType,
		"stripe_event_id":  stripeEventID,
	})

	return nil
}

// handleCreditTopup processes a credit top-up from a checkout session.
func (s *StripeService) handleCreditTopup(customerEmail string, amountCents int64, plan *PlanOption, stripeEventID string, metadata map[string]interface{}) error {
	if customerEmail == "" {
		return errors.New("customer email required for credit top-up")
	}

	// Normalize email for consistency
	customerEmail = NormalizeEmail(customerEmail)

	if amountCents == 0 {
		amountCents = plan.AmountCents
	}

	bundle, err := s.planService.GetBundleProduct()
	if err != nil {
		return err
	}

	if bundle == nil {
		logStructuredError("bundle_product_not_configured", map[string]interface{}{
			"customer_email":  customerEmail,
			"amount_cents":    amountCents,
			"stripe_event_id": stripeEventID,
		})
		return errors.New("bundle product not configured - cannot process credit topup")
	}

	if amountCents == 0 {
		return errors.New("amount is zero - cannot process credit topup")
	}

	credits := (bundle.CreditsPerUsd * amountCents) / 100
	if credits <= 0 {
		return nil
	}

	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["price_id"] = plan.StripePriceId
	metadata["session_type"] = sessionTypeCreditsTopup

	return s.addCredits(customerEmail, credits, "credit_topup", stripeEventID, metadata)
}
