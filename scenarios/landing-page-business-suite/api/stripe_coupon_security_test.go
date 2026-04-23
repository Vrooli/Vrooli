package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCoupon_PaymentTimeEligibilityRecheck verifies that a second eligibility check
// is performed at invoice.paid time for subscription_create events with intro coupons.
func TestCoupon_PaymentTimeEligibilityRecheck(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Set up bundle product for ConfigureStripeService
	upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_test", "production", 100, 0.001, "credits")

	// Recreate the minimum tables the scenario needs; payment_anomaly_log is
	// created by setupTestDB via ensureSchema.
	_, err := db.Exec(`
		DROP TABLE IF EXISTS intro_coupon_usage CASCADE;
		DROP TABLE IF EXISTS subscriptions CASCADE;
		DROP TABLE IF EXISTS users CASCADE;
		DELETE FROM payment_anomaly_log;

		CREATE TABLE users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			stripe_customer_id VARCHAR(255),
			has_used_intro BOOLEAN DEFAULT FALSE,
			email_verified BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);
		CREATE TABLE subscriptions (
			id SERIAL PRIMARY KEY,
			subscription_id VARCHAR(255) UNIQUE NOT NULL,
			customer_id VARCHAR(255),
			customer_email VARCHAR(255),
			status VARCHAR(50) NOT NULL,
			plan_tier VARCHAR(50),
			price_id VARCHAR(255),
			bundle_key VARCHAR(100),
			canceled_at TIMESTAMP,
			billing_cycle_start INTEGER,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);
		CREATE TABLE intro_coupon_usage (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) NOT NULL,
			stripe_customer_id VARCHAR(255),
			coupon_id VARCHAR(255) NOT NULL,
			plan_tier VARCHAR(50),
			subscription_id VARCHAR(255),
			created_at TIMESTAMP DEFAULT NOW()
		);
	`)
	require.NoError(t, err)

	// Create a user who has already used their intro coupon
	_, err = db.Exec(`INSERT INTO users (email, has_used_intro) VALUES ($1, TRUE)`, "ineligible@example.com")
	require.NoError(t, err)

	// Configure service with intro coupon enabled
	cfg := DefaultStripeTestConfig().
		WithKeys("pk_test", "sk_test", "whsec_test").
		WithIntroCoupon(true, map[string]string{"pro": "coupon_intro_pro"})
	service := ConfigureStripeService(t, db, cfg, nil)

	// Simulate invoice.paid webhook with billing_reason = subscription_create
	event := map[string]interface{}{
		"id":   "evt_invoice_intro_check",
		"type": "invoice.paid",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"subscription":   "sub_intro_123",
				"customer":       "cus_intro_123",
				"customer_email": "ineligible@example.com",
				"billing_reason": "subscription_create",
				"discount": map[string]interface{}{
					"coupon": map[string]interface{}{
						"id": "coupon_intro_pro",
					},
				},
			},
		},
	}

	payload, _ := json.Marshal(event)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	signedPayload := timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte("whsec_test"))
	mac.Write([]byte(signedPayload))
	signature := hex.EncodeToString(mac.Sum(nil))
	signatureHeader := "t=" + timestamp + ",v1=" + signature

	err = service.HandleWebhook(payload, signatureHeader)
	require.NoError(t, err)

	// Verify that an anomaly was logged for ineligible user at payment time
	var anomalyCount int
	var anomalyType string
	err = db.QueryRow(`SELECT COUNT(*), MAX(anomaly_type) FROM payment_anomaly_log WHERE email = $1 AND subject_kind = 'intro_coupon'`, "ineligible@example.com").Scan(&anomalyCount, &anomalyType)
	require.NoError(t, err)
	assert.Equal(t, 1, anomalyCount, "should log exactly one anomaly")
	assert.Equal(t, "ineligible_at_payment", anomalyType, "anomaly type should be 'ineligible_at_payment'")
}

// TestCoupon_EligibleUser_NoAnomalyLogged verifies that eligible users don't get anomaly logged.
func TestCoupon_EligibleUser_NoAnomalyLogged(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Set up bundle product for ConfigureStripeService
	upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_test", "production", 100, 0.001, "credits")

	// Create tables (payment_anomaly_log is provided by ensureSchema in setupTestDB).
	_, err := db.Exec(`
		DROP TABLE IF EXISTS intro_coupon_usage CASCADE;
		DROP TABLE IF EXISTS subscriptions CASCADE;
		DROP TABLE IF EXISTS users CASCADE;
		DELETE FROM payment_anomaly_log;

		CREATE TABLE users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			stripe_customer_id VARCHAR(255),
			has_used_intro BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);
		CREATE TABLE subscriptions (
			id SERIAL PRIMARY KEY,
			subscription_id VARCHAR(255) UNIQUE NOT NULL,
			customer_id VARCHAR(255),
			customer_email VARCHAR(255),
			status VARCHAR(50) NOT NULL,
			plan_tier VARCHAR(50),
			price_id VARCHAR(255),
			bundle_key VARCHAR(100),
			canceled_at TIMESTAMP,
			billing_cycle_start INTEGER,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);
		CREATE TABLE intro_coupon_usage (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) NOT NULL,
			stripe_customer_id VARCHAR(255),
			coupon_id VARCHAR(255) NOT NULL,
			plan_tier VARCHAR(50),
			subscription_id VARCHAR(255),
			created_at TIMESTAMP DEFAULT NOW()
		);
	`)
	require.NoError(t, err)

	// Create an eligible user (has_used_intro = FALSE)
	_, err = db.Exec(`INSERT INTO users (email, has_used_intro) VALUES ($1, FALSE)`, "eligible@example.com")
	require.NoError(t, err)

	cfg := DefaultStripeTestConfig().
		WithKeys("pk_test", "sk_test", "whsec_test").
		WithIntroCoupon(true, map[string]string{"pro": "coupon_intro_pro"})
	service := ConfigureStripeService(t, db, cfg, nil)

	event := map[string]interface{}{
		"id":   "evt_invoice_eligible",
		"type": "invoice.paid",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"subscription":   "sub_eligible_123",
				"customer":       "cus_eligible_123",
				"customer_email": "eligible@example.com",
				"billing_reason": "subscription_create",
				"discount": map[string]interface{}{
					"coupon": map[string]interface{}{
						"id": "coupon_intro_pro",
					},
				},
			},
		},
	}

	payload, _ := json.Marshal(event)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	signedPayload := timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte("whsec_test"))
	mac.Write([]byte(signedPayload))
	signature := hex.EncodeToString(mac.Sum(nil))
	signatureHeader := "t=" + timestamp + ",v1=" + signature

	err = service.HandleWebhook(payload, signatureHeader)
	require.NoError(t, err)

	// Verify no anomaly was logged
	var anomalyCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM payment_anomaly_log WHERE email = $1 AND subject_kind = 'intro_coupon'`, "eligible@example.com").Scan(&anomalyCount)
	require.NoError(t, err)
	assert.Equal(t, 0, anomalyCount, "should not log anomaly for eligible user")
}

// TestCoupon_EmailMigration_IntroFlagCarriesOver verifies that the has_used_intro
// flag is properly carried over when a customer changes their email.
func TestCoupon_EmailMigration_IntroFlagCarriesOver(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Create tables
	_, err := db.Exec(`
		DROP TABLE IF EXISTS intro_coupon_usage CASCADE;
		DROP TABLE IF EXISTS credit_transactions CASCADE;
		DROP TABLE IF EXISTS credit_wallets CASCADE;
		DROP TABLE IF EXISTS subscriptions CASCADE;
		DROP TABLE IF EXISTS users CASCADE;

		CREATE TABLE users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			stripe_customer_id VARCHAR(255),
			has_used_intro BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);
		CREATE TABLE subscriptions (
			id SERIAL PRIMARY KEY,
			subscription_id VARCHAR(255) UNIQUE NOT NULL,
			customer_id VARCHAR(255),
			customer_email VARCHAR(255),
			status VARCHAR(50) NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);
		CREATE TABLE credit_wallets (
			id SERIAL PRIMARY KEY,
			customer_email VARCHAR(255) UNIQUE NOT NULL,
			balance_credits BIGINT DEFAULT 0,
			updated_at TIMESTAMP DEFAULT NOW()
		);
		CREATE TABLE credit_transactions (
			id SERIAL PRIMARY KEY,
			customer_email VARCHAR(255) NOT NULL,
			amount_credits BIGINT NOT NULL,
			transaction_type VARCHAR(50) NOT NULL,
			stripe_event_id VARCHAR(255) UNIQUE,
			metadata JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP DEFAULT NOW()
		);
		CREATE TABLE intro_coupon_usage (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) NOT NULL,
			stripe_customer_id VARCHAR(255),
			coupon_id VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		);
	`)
	require.NoError(t, err)

	oldEmail := "old-intro@example.com"
	newEmail := "new-intro@example.com"
	customerID := "cus_migrate_intro"

	// Create user with old email who has used intro
	_, err = db.Exec(`INSERT INTO users (email, stripe_customer_id, has_used_intro) VALUES ($1, $2, TRUE)`, oldEmail, customerID)
	require.NoError(t, err)

	// Create user with new email who has NOT used intro
	_, err = db.Exec(`INSERT INTO users (email, has_used_intro) VALUES ($1, FALSE)`, newEmail)
	require.NoError(t, err)

	service := NewStripeService(db)

	// Migrate email
	err = service.MigrateCustomerEmail(context.Background(), oldEmail, newEmail, customerID)
	require.NoError(t, err)

	// Verify the new email now has has_used_intro = TRUE (carried over from old)
	var hasUsedIntro bool
	err = db.QueryRow(`SELECT has_used_intro FROM users WHERE email = $1`, newEmail).Scan(&hasUsedIntro)
	require.NoError(t, err)
	assert.True(t, hasUsedIntro, "has_used_intro should be TRUE after migration (OR'd with old email's value)")
}

// TestCoupon_EmailMigration_BothUsedIntro verifies that the flag stays true
// when both emails have used intro.
func TestCoupon_EmailMigration_BothUsedIntro(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Create tables
	_, err := db.Exec(`
		DROP TABLE IF EXISTS intro_coupon_usage CASCADE;
		DROP TABLE IF EXISTS credit_transactions CASCADE;
		DROP TABLE IF EXISTS credit_wallets CASCADE;
		DROP TABLE IF EXISTS subscriptions CASCADE;
		DROP TABLE IF EXISTS users CASCADE;

		CREATE TABLE users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			stripe_customer_id VARCHAR(255),
			has_used_intro BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);
		CREATE TABLE subscriptions (
			id SERIAL PRIMARY KEY,
			subscription_id VARCHAR(255) UNIQUE NOT NULL,
			customer_id VARCHAR(255),
			customer_email VARCHAR(255),
			status VARCHAR(50) NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);
		CREATE TABLE credit_wallets (
			id SERIAL PRIMARY KEY,
			customer_email VARCHAR(255) UNIQUE NOT NULL,
			balance_credits BIGINT DEFAULT 0,
			updated_at TIMESTAMP DEFAULT NOW()
		);
		CREATE TABLE credit_transactions (
			id SERIAL PRIMARY KEY,
			customer_email VARCHAR(255) NOT NULL,
			amount_credits BIGINT NOT NULL,
			transaction_type VARCHAR(50) NOT NULL,
			stripe_event_id VARCHAR(255) UNIQUE,
			metadata JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP DEFAULT NOW()
		);
		CREATE TABLE intro_coupon_usage (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) NOT NULL,
			stripe_customer_id VARCHAR(255),
			coupon_id VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		);
	`)
	require.NoError(t, err)

	oldEmail := "old-both@example.com"
	newEmail := "new-both@example.com"
	customerID := "cus_both_intro"

	// Both users have used intro
	_, err = db.Exec(`INSERT INTO users (email, stripe_customer_id, has_used_intro) VALUES ($1, $2, TRUE)`, oldEmail, customerID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO users (email, has_used_intro) VALUES ($1, TRUE)`, newEmail)
	require.NoError(t, err)

	service := NewStripeService(db)

	err = service.MigrateCustomerEmail(context.Background(), oldEmail, newEmail, customerID)
	require.NoError(t, err)

	var hasUsedIntro bool
	err = db.QueryRow(`SELECT has_used_intro FROM users WHERE email = $1`, newEmail).Scan(&hasUsedIntro)
	require.NoError(t, err)
	assert.True(t, hasUsedIntro, "has_used_intro should remain TRUE")
}

// TestConsumeCredits_Idempotent_SameKey verifies that the same idempotency key
// only deducts credits once.
func TestConsumeCredits_Idempotent_SameKey(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Create tables
	_, err := db.Exec(`
		DROP TABLE IF EXISTS credit_transactions CASCADE;
		DROP TABLE IF EXISTS credit_wallets CASCADE;
		CREATE TABLE credit_wallets (
			id SERIAL PRIMARY KEY,
			customer_email VARCHAR(255) UNIQUE NOT NULL,
			balance_credits BIGINT DEFAULT 0,
			updated_at TIMESTAMP DEFAULT NOW()
		);
		CREATE TABLE credit_transactions (
			id SERIAL PRIMARY KEY,
			customer_email VARCHAR(255) NOT NULL,
			amount_credits BIGINT NOT NULL,
			transaction_type VARCHAR(50) NOT NULL,
			stripe_event_id VARCHAR(255) UNIQUE,
			metadata JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP DEFAULT NOW()
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_credit_transactions_stripe_event
		ON credit_transactions(stripe_event_id) WHERE stripe_event_id IS NOT NULL;
	`)
	require.NoError(t, err)

	// Create wallet with credits
	_, err = db.Exec(`INSERT INTO credit_wallets (customer_email, balance_credits) VALUES ($1, $2)`, "consume@example.com", 1000)
	require.NoError(t, err)

	service := NewStripeService(db)

	// First consumption
	err = service.ConsumeCreditsIdempotent(context.Background(), "consume@example.com", 100, "test_consume", "evt_consume_123", nil)
	require.NoError(t, err)

	// Check balance
	var balance int64
	err = db.QueryRow(`SELECT balance_credits FROM credit_wallets WHERE customer_email = $1`, "consume@example.com").Scan(&balance)
	require.NoError(t, err)
	assert.Equal(t, int64(900), balance, "balance should be 900 after first deduction")

	// Second consumption with same key - should be idempotent
	err = service.ConsumeCreditsIdempotent(context.Background(), "consume@example.com", 100, "test_consume", "evt_consume_123", nil)
	require.NoError(t, err)

	// Balance should still be 900
	err = db.QueryRow(`SELECT balance_credits FROM credit_wallets WHERE customer_email = $1`, "consume@example.com").Scan(&balance)
	require.NoError(t, err)
	assert.Equal(t, int64(900), balance, "balance should still be 900 (idempotent)")

	// Verify only one transaction
	var txnCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM credit_transactions WHERE stripe_event_id = $1`, "evt_consume_123").Scan(&txnCount)
	require.NoError(t, err)
	assert.Equal(t, 1, txnCount, "should only have one transaction")
}

// TestConsumeCredits_DifferentKeys_BothDeduct verifies that different idempotency
// keys result in separate deductions.
func TestConsumeCredits_DifferentKeys_BothDeduct(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Create tables
	_, err := db.Exec(`
		DROP TABLE IF EXISTS credit_transactions CASCADE;
		DROP TABLE IF EXISTS credit_wallets CASCADE;
		CREATE TABLE credit_wallets (
			id SERIAL PRIMARY KEY,
			customer_email VARCHAR(255) UNIQUE NOT NULL,
			balance_credits BIGINT DEFAULT 0,
			updated_at TIMESTAMP DEFAULT NOW()
		);
		CREATE TABLE credit_transactions (
			id SERIAL PRIMARY KEY,
			customer_email VARCHAR(255) NOT NULL,
			amount_credits BIGINT NOT NULL,
			transaction_type VARCHAR(50) NOT NULL,
			stripe_event_id VARCHAR(255) UNIQUE,
			metadata JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP DEFAULT NOW()
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_credit_transactions_stripe_event
		ON credit_transactions(stripe_event_id) WHERE stripe_event_id IS NOT NULL;
	`)
	require.NoError(t, err)

	// Create wallet with credits
	_, err = db.Exec(`INSERT INTO credit_wallets (customer_email, balance_credits) VALUES ($1, $2)`, "multi@example.com", 1000)
	require.NoError(t, err)

	service := NewStripeService(db)

	// First consumption
	err = service.ConsumeCreditsIdempotent(context.Background(), "multi@example.com", 100, "test_consume", "evt_multi_1", nil)
	require.NoError(t, err)

	// Second consumption with different key
	err = service.ConsumeCreditsIdempotent(context.Background(), "multi@example.com", 100, "test_consume", "evt_multi_2", nil)
	require.NoError(t, err)

	// Balance should be 800
	var balance int64
	err = db.QueryRow(`SELECT balance_credits FROM credit_wallets WHERE customer_email = $1`, "multi@example.com").Scan(&balance)
	require.NoError(t, err)
	assert.Equal(t, int64(800), balance, "balance should be 800 after two deductions")
}

// TestCancelSubscription_StripeFailure_NoLocalUpdate verifies that the local DB
// is not updated when Stripe API fails.
func TestCancelSubscription_StripeFailure_NoLocalUpdate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Set up bundle product for ConfigureStripeService
	upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_test", "production", 100, 0.001, "credits")

	// Create tables
	_, err := db.Exec(`
		DROP TABLE IF EXISTS subscriptions CASCADE;
		CREATE TABLE subscriptions (
			id SERIAL PRIMARY KEY,
			subscription_id VARCHAR(255) UNIQUE NOT NULL,
			customer_id VARCHAR(255),
			customer_email VARCHAR(255),
			status VARCHAR(50) NOT NULL,
			canceled_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);
	`)
	require.NoError(t, err)

	// Insert an active subscription
	_, err = db.Exec(`INSERT INTO subscriptions (subscription_id, customer_email, status) VALUES ($1, $2, $3)`,
		"sub_fail_cancel", "fail@example.com", "active")
	require.NoError(t, err)

	// Create a mock Stripe server that returns an error
	mockStripe := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"error": {"message": "Stripe service unavailable"}}`)
	}))
	defer mockStripe.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), mockStripe)

	// Attempt to cancel - should fail
	_, err = service.CancelSubscription("fail@example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to cancel subscription with Stripe")

	// Verify DB was NOT updated (status should still be active)
	var status string
	var canceledAt *time.Time
	err = db.QueryRow(`SELECT status, canceled_at FROM subscriptions WHERE subscription_id = $1`, "sub_fail_cancel").Scan(&status, &canceledAt)
	require.NoError(t, err)
	assert.Equal(t, "active", status, "status should still be active after Stripe failure")
	assert.Nil(t, canceledAt, "canceled_at should still be NULL after Stripe failure")
}

// TestCancelSubscription_StripeSuccess_BothUpdated verifies successful cancellation
// updates both Stripe and local DB.
func TestCancelSubscription_StripeSuccess_BothUpdated(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Set up bundle product for ConfigureStripeService
	upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_test", "production", 100, 0.001, "credits")

	// Create tables
	_, err := db.Exec(`
		DROP TABLE IF EXISTS subscriptions CASCADE;
		CREATE TABLE subscriptions (
			id SERIAL PRIMARY KEY,
			subscription_id VARCHAR(255) UNIQUE NOT NULL,
			customer_id VARCHAR(255),
			customer_email VARCHAR(255),
			status VARCHAR(50) NOT NULL,
			canceled_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);
	`)
	require.NoError(t, err)

	// Insert an active subscription
	_, err = db.Exec(`INSERT INTO subscriptions (subscription_id, customer_email, status) VALUES ($1, $2, $3)`,
		"sub_success_cancel", "success@example.com", "active")
	require.NoError(t, err)

	// Create a mock Stripe server that returns success
	mockStripe := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"sub_success_cancel","status":"canceled","cancel_at_period_end":true}`)
	}))
	defer mockStripe.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), mockStripe)

	// Cancel subscription
	result, err := service.CancelSubscription("success@example.com")
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify DB was updated
	var status string
	var canceledAt *time.Time
	err = db.QueryRow(`SELECT status, canceled_at FROM subscriptions WHERE subscription_id = $1`, "sub_success_cancel").Scan(&status, &canceledAt)
	require.NoError(t, err)
	assert.Equal(t, "canceled", status, "status should be canceled")
	assert.NotNil(t, canceledAt, "canceled_at should be set")
}

// TestWebhookTimestampValidation_OldTimestamp verifies that old timestamps are rejected.
func TestWebhookTimestampValidation_OldTimestamp(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Set up bundle product for ConfigureStripeService
	upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_test", "production", 100, 0.001, "credits")

	cfg := DefaultStripeTestConfig().WithKeys("pk_test", "sk_test", "whsec_test")
	service := ConfigureStripeService(t, db, cfg, nil)

	payload := []byte(`{"type":"test.event","data":{}}`)

	// Create signature with old timestamp (10 minutes ago)
	oldTimestamp := fmt.Sprintf("%d", time.Now().Add(-10*time.Minute).Unix())
	signedPayload := oldTimestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte("whsec_test"))
	mac.Write([]byte(signedPayload))
	signature := hex.EncodeToString(mac.Sum(nil))
	signatureHeader := "t=" + oldTimestamp + ",v1=" + signature

	// Should fail timestamp validation
	valid := service.VerifyWebhookSignature(payload, signatureHeader)
	assert.False(t, valid, "old timestamp should be rejected")
}

// TestWebhookTimestampValidation_CurrentTimestamp verifies that current timestamps pass.
func TestWebhookTimestampValidation_CurrentTimestamp(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Set up bundle product for ConfigureStripeService
	upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_test", "production", 100, 0.001, "credits")

	cfg := DefaultStripeTestConfig().WithKeys("pk_test", "sk_test", "whsec_test")
	service := ConfigureStripeService(t, db, cfg, nil)

	payload := []byte(`{"type":"test.event","data":{}}`)

	// Create signature with current timestamp
	currentTimestamp := fmt.Sprintf("%d", time.Now().Unix())
	signedPayload := currentTimestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte("whsec_test"))
	mac.Write([]byte(signedPayload))
	signature := hex.EncodeToString(mac.Sum(nil))
	signatureHeader := "t=" + currentTimestamp + ",v1=" + signature

	// Should pass timestamp validation
	valid := service.VerifyWebhookSignature(payload, signatureHeader)
	assert.True(t, valid, "current timestamp should be accepted")
}

// TestWebhookTimestampValidation_FutureTimestamp verifies that future timestamps are rejected.
func TestWebhookTimestampValidation_FutureTimestamp(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Set up bundle product for ConfigureStripeService
	upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_test", "production", 100, 0.001, "credits")

	cfg := DefaultStripeTestConfig().WithKeys("pk_test", "sk_test", "whsec_test")
	service := ConfigureStripeService(t, db, cfg, nil)

	payload := []byte(`{"type":"test.event","data":{}}`)

	// Create signature with future timestamp (10 minutes from now)
	futureTimestamp := fmt.Sprintf("%d", time.Now().Add(10*time.Minute).Unix())
	signedPayload := futureTimestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte("whsec_test"))
	mac.Write([]byte(signedPayload))
	signature := hex.EncodeToString(mac.Sum(nil))
	signatureHeader := "t=" + futureTimestamp + ",v1=" + signature

	// Should fail timestamp validation
	valid := service.VerifyWebhookSignature(payload, signatureHeader)
	assert.False(t, valid, "future timestamp should be rejected")
}

// TestLogIntroAnomaly_RecordsCorrectly verifies that anomalies are forwarded
// through PaymentAnomalyService into payment_anomaly_log.
func TestLogIntroAnomaly_RecordsCorrectly(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	if _, err := db.Exec(`DELETE FROM payment_anomaly_log`); err != nil {
		t.Fatalf("reset payment_anomaly_log: %v", err)
	}

	service := NewStripeService(db)
	service.SetPaymentAnomaly(NewPaymentAnomalyService(context.Background(), db, context.Background()))

	// Log an anomaly
	details := map[string]interface{}{
		"subscription_id": "sub_test_123",
		"reason":          "test_reason",
	}
	service.logIntroAnomaly("anomaly@example.com", "cus_anomaly_123", "coupon_test", "test_anomaly_type", details)

	// Verify it was recorded under the unified pipeline.
	var email, customerID, subjectID, anomalyType, detailsJSON string
	err := db.QueryRow(`
		SELECT email, customer_id, subject_id, anomaly_type, details::text
		FROM payment_anomaly_log
		WHERE email = $1 AND subject_kind = 'intro_coupon'
	`, "anomaly@example.com").Scan(&email, &customerID, &subjectID, &anomalyType, &detailsJSON)
	require.NoError(t, err)
	assert.Equal(t, "anomaly@example.com", email)
	assert.Equal(t, "cus_anomaly_123", customerID)
	assert.Equal(t, "coupon_test", subjectID)
	assert.Equal(t, "test_anomaly_type", anomalyType)
	assert.Contains(t, detailsJSON, "sub_test_123")
}
