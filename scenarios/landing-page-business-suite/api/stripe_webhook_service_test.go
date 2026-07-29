package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupCheckoutAtomicitySchema provisions a clean schema containing the tables
// touched by handleCheckoutCompleted's subscription branch. Intentionally
// recreated per test so that added CHECK constraints do not leak across cases.
func setupCheckoutAtomicitySchema(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`
		DROP TABLE IF EXISTS subscription_schedules CASCADE;
		DROP TABLE IF EXISTS subscriptions CASCADE;
		DROP TABLE IF EXISTS checkout_sessions CASCADE;
		DROP TABLE IF EXISTS users CASCADE;

		CREATE TABLE users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			stripe_customer_id VARCHAR(255),
			has_used_intro BOOLEAN DEFAULT FALSE,
			email_verified BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			last_login_at TIMESTAMP
		);
		CREATE TABLE checkout_sessions (
			id SERIAL PRIMARY KEY,
			session_id VARCHAR(255) UNIQUE NOT NULL,
			customer_email VARCHAR(255),
			customer_id VARCHAR(255),
			price_id VARCHAR(255),
			subscription_id VARCHAR(255),
			status VARCHAR(50) NOT NULL,
			session_type VARCHAR(50) DEFAULT 'subscription',
			amount_cents INTEGER,
			schedule_id VARCHAR(255),
			metadata JSONB DEFAULT '{}'::jsonb,
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
			billing_cycle_start INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);
		CREATE TABLE subscription_schedules (
			id SERIAL PRIMARY KEY,
			schedule_id VARCHAR(255) UNIQUE NOT NULL,
			subscription_id VARCHAR(255),
			price_id VARCHAR(255),
			billing_interval VARCHAR(50),
			intro_enabled BOOLEAN DEFAULT FALSE,
			intro_amount_cents INTEGER DEFAULT 0,
			intro_periods INTEGER DEFAULT 0,
			normal_amount_cents INTEGER DEFAULT 0,
			next_billing_at TIMESTAMP,
			status VARCHAR(50) DEFAULT 'active',
			metadata JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)
	`)
	require.NoError(t, err)
}

// buildCheckoutCompletedWebhook returns a signed checkout.session.completed
// payload together with the stripe signature header for HandleWebhook.
func buildCheckoutCompletedWebhook(t *testing.T, secret, eventID, sessionID, subscriptionID, customerEmail string, amountTotal int64) ([]byte, string) {
	t.Helper()
	body := map[string]interface{}{
		"id":   eventID,
		"type": "checkout.session.completed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":             sessionID,
				"customer":       "cus_atomicity",
				"customer_email": customerEmail,
				"subscription":   subscriptionID,
				"amount_total":   amountTotal,
			},
		},
	}
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "." + string(payload)))
	signature := hex.EncodeToString(mac.Sum(nil))
	return payload, "t=" + timestamp + ",v1=" + signature
}

// insertPendingSubscriptionCheckout seeds an open subscription checkout_session
// so that handleCheckoutCompleted can transition it to 'complete'.
func insertPendingSubscriptionCheckout(t *testing.T, db *sql.DB, sessionID, customerEmail, priceID string, amountCents int) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO checkout_sessions (session_id, customer_email, price_id, status, session_type, amount_cents)
		VALUES ($1, $2, $3, 'pending', $4, $5)
	`, sessionID, customerEmail, priceID, sessionTypeSubscription, amountCents)
	require.NoError(t, err)
}

// TestHandleCheckoutCompleted_SubscriptionInsertFailure_RollsBackStatus
// Injects a schema-level CHECK constraint that blocks the subscription UPSERT
// (it INSERTs with status='active'). Asserts that the checkout_sessions row
// remains 'pending' and no subscription row materializes — proving the
// UPDATE + INSERT are wrapped in a single transaction.
//
// Failure mechanism: CHECK (status <> 'active') on subscriptions.
// If the UPSERT in handleSubscriptionCompletion is ever relaxed so that the
// SET clause no longer writes status='active', update this constraint.
func TestHandleCheckoutCompleted_SubscriptionInsertFailure_RollsBackStatus(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)
	setupCheckoutAtomicitySchema(t, db)

	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_atomicity_sub_fail", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_atomicity_sub_fail", "Pro Plan", "pro", "month", "usd", 2900, false, "", 0, 0, "", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	insertPendingSubscriptionCheckout(t, db, "cs_atomicity_sub_fail", "atomicity1@example.com", "price_atomicity_sub_fail", 2900)

	_, err := db.Exec(`ALTER TABLE subscriptions ADD CONSTRAINT test_block_active_sub CHECK (status <> 'active')`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS test_block_active_sub`)
	})

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig().WithKeys("pk_test", "sk_test", "whsec_atomicity"), nil)

	payload, sig := buildCheckoutCompletedWebhook(t, "whsec_atomicity", "evt_sub_fail", "cs_atomicity_sub_fail", "sub_atomicity_fail", "atomicity1@example.com", 2900)

	err = service.HandleWebhook(payload, sig)
	require.Error(t, err, "webhook should surface the DB error for Stripe to retry")

	var sessionStatus string
	require.NoError(t, db.QueryRow(`SELECT status FROM checkout_sessions WHERE session_id = $1`, "cs_atomicity_sub_fail").Scan(&sessionStatus))
	assert.Equal(t, "pending", sessionStatus, "checkout_sessions.status must be rolled back when subscription INSERT fails")

	var subCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM subscriptions WHERE subscription_id = $1`, "sub_atomicity_fail").Scan(&subCount))
	assert.Equal(t, 0, subCount, "no subscription row should exist after rollback")
}

// TestHandleCheckoutCompleted_ScheduleInsertFailure_RollsBackAll
// Uses an intro-enabled monthly plan so handleSubscriptionCompletion also
// calls createSubscriptionSchedule. Blocks the schedule UPSERT via a CHECK
// constraint and asserts that BOTH the subscription INSERT and the
// checkout_sessions UPDATE roll back.
func TestHandleCheckoutCompleted_ScheduleInsertFailure_RollsBackAll(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)
	setupCheckoutAtomicitySchema(t, db)

	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_atomicity_sched_fail", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_atomicity_sched_fail", "Pro Plan", "pro", "month", "usd", 2900, true, "flat_amount", 100, 1, "intro_atomicity_lookup", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	insertPendingSubscriptionCheckout(t, db, "cs_atomicity_sched_fail", "atomicity2@example.com", "price_atomicity_sched_fail", 2900)

	_, err := db.Exec(`ALTER TABLE subscription_schedules ADD CONSTRAINT test_block_active_sched CHECK (status <> 'active')`)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`ALTER TABLE subscription_schedules DROP CONSTRAINT IF EXISTS test_block_active_sched`)
	})

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig().WithKeys("pk_test", "sk_test", "whsec_atomicity"), nil)

	payload, sig := buildCheckoutCompletedWebhook(t, "whsec_atomicity", "evt_sched_fail", "cs_atomicity_sched_fail", "sub_atomicity_sched", "atomicity2@example.com", 2900)

	err = service.HandleWebhook(payload, sig)
	require.Error(t, err, "schedule failure must propagate so Stripe retries")

	var sessionStatus string
	require.NoError(t, db.QueryRow(`SELECT status FROM checkout_sessions WHERE session_id = $1`, "cs_atomicity_sched_fail").Scan(&sessionStatus))
	assert.Equal(t, "pending", sessionStatus, "checkout_sessions.status must be rolled back when schedule INSERT fails")

	var subCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM subscriptions WHERE subscription_id = $1`, "sub_atomicity_sched").Scan(&subCount))
	assert.Equal(t, 0, subCount, "subscription INSERT must roll back with the schedule INSERT")

	var schedCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM subscription_schedules WHERE subscription_id = $1`, "sub_atomicity_sched").Scan(&schedCount))
	assert.Equal(t, 0, schedCount, "no schedule row should exist after rollback")
}

// TestHandleCheckoutCompleted_SubscriptionBranch_HappyPath_UsesSingleTx
// Covers both the non-scheduled (flat monthly) and scheduled (intro) plans,
// asserting that all expected rows materialise when the transaction commits.
func TestHandleCheckoutCompleted_SubscriptionBranch_HappyPath_UsesSingleTx(t *testing.T) {
	t.Run("non-scheduled plan", func(t *testing.T) {
		db := setupTestDB(t)
		resetStripeTestData(t, db)
		setupCheckoutAtomicitySchema(t, db)

		productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_happy_flat", "production", 1000000, 0.001, "credits")
		insertBundlePrice(t, db, productID, "price_happy_flat", "Pro Plan", "pro", "month", "usd", 2900, false, "", 0, 0, "", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

		insertPendingSubscriptionCheckout(t, db, "cs_happy_flat", "happy1@example.com", "price_happy_flat", 2900)

		service := ConfigureStripeService(t, db, DefaultStripeTestConfig().WithKeys("pk_test", "sk_test", "whsec_happy"), nil)
		payload, sig := buildCheckoutCompletedWebhook(t, "whsec_happy", "evt_happy_flat", "cs_happy_flat", "sub_happy_flat", "happy1@example.com", 2900)
		require.NoError(t, service.HandleWebhook(payload, sig))

		var sessionStatus string
		require.NoError(t, db.QueryRow(`SELECT status FROM checkout_sessions WHERE session_id = $1`, "cs_happy_flat").Scan(&sessionStatus))
		assert.Equal(t, "complete", sessionStatus)

		var subStatus, planTier string
		require.NoError(t, db.QueryRow(`SELECT status, plan_tier FROM subscriptions WHERE subscription_id = $1`, "sub_happy_flat").Scan(&subStatus, &planTier))
		assert.Equal(t, "active", subStatus)
		assert.Equal(t, "pro", planTier)

		var schedCount int
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM subscription_schedules WHERE subscription_id = $1`, "sub_happy_flat").Scan(&schedCount))
		assert.Equal(t, 0, schedCount, "flat plan should not create a schedule row")
	})

	t.Run("scheduled intro plan", func(t *testing.T) {
		db := setupTestDB(t)
		resetStripeTestData(t, db)
		setupCheckoutAtomicitySchema(t, db)

		productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_happy_intro", "production", 1000000, 0.001, "credits")
		insertBundlePrice(t, db, productID, "price_happy_intro", "Pro Plan", "pro", "month", "usd", 2900, true, "flat_amount", 100, 1, "intro_happy_lookup", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

		insertPendingSubscriptionCheckout(t, db, "cs_happy_intro", "happy2@example.com", "price_happy_intro", 2900)

		service := ConfigureStripeService(t, db, DefaultStripeTestConfig().WithKeys("pk_test", "sk_test", "whsec_happy"), nil)
		payload, sig := buildCheckoutCompletedWebhook(t, "whsec_happy", "evt_happy_intro", "cs_happy_intro", "sub_happy_intro", "happy2@example.com", 2900)
		require.NoError(t, service.HandleWebhook(payload, sig))

		var sessionStatus, scheduleID string
		require.NoError(t, db.QueryRow(`SELECT status, COALESCE(schedule_id,'') FROM checkout_sessions WHERE session_id = $1`, "cs_happy_intro").Scan(&sessionStatus, &scheduleID))
		assert.Equal(t, "complete", sessionStatus)
		assert.NotEmpty(t, scheduleID, "checkout_sessions.schedule_id must be written inside the tx")

		var subStatus string
		require.NoError(t, db.QueryRow(`SELECT status FROM subscriptions WHERE subscription_id = $1`, "sub_happy_intro").Scan(&subStatus))
		assert.Equal(t, "active", subStatus)

		var schedStatus string
		require.NoError(t, db.QueryRow(`SELECT status FROM subscription_schedules WHERE subscription_id = $1`, "sub_happy_intro").Scan(&schedStatus))
		assert.Equal(t, "active", schedStatus)
	})
}

// TestHandleCheckoutCompleted_RetryAfterRollback_Succeeds simulates Stripe's
// automatic webhook retry after a rolled-back failure: inject failure, fire,
// observe rollback, clear the constraint, fire again, observe success. This
// exercises the ON CONFLICT safety net the fix relies on for healing.
func TestHandleCheckoutCompleted_RetryAfterRollback_Succeeds(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)
	setupCheckoutAtomicitySchema(t, db)

	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_retry_after_rb", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_retry_after_rb", "Pro Plan", "pro", "month", "usd", 2900, false, "", 0, 0, "", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	insertPendingSubscriptionCheckout(t, db, "cs_retry_after_rb", "retry-rb@example.com", "price_retry_after_rb", 2900)

	_, err := db.Exec(`ALTER TABLE subscriptions ADD CONSTRAINT test_block_active_retry CHECK (status <> 'active')`)
	require.NoError(t, err)

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig().WithKeys("pk_test", "sk_test", "whsec_retry"), nil)
	payload, sig := buildCheckoutCompletedWebhook(t, "whsec_retry", "evt_retry_after_rb", "cs_retry_after_rb", "sub_retry_after_rb", "retry-rb@example.com", 2900)

	require.Error(t, service.HandleWebhook(payload, sig), "first attempt must fail")

	var sessionStatus string
	require.NoError(t, db.QueryRow(`SELECT status FROM checkout_sessions WHERE session_id = $1`, "cs_retry_after_rb").Scan(&sessionStatus))
	require.Equal(t, "pending", sessionStatus)

	_, err = db.Exec(`ALTER TABLE subscriptions DROP CONSTRAINT test_block_active_retry`)
	require.NoError(t, err)

	// Stripe replay of the same webhook: same event id, same payload, same signature window.
	payload2, sig2 := buildCheckoutCompletedWebhook(t, "whsec_retry", "evt_retry_after_rb", "cs_retry_after_rb", "sub_retry_after_rb", "retry-rb@example.com", 2900)
	require.NoError(t, service.HandleWebhook(payload2, sig2), "retry after clearing the failure must succeed")

	require.NoError(t, db.QueryRow(`SELECT status FROM checkout_sessions WHERE session_id = $1`, "cs_retry_after_rb").Scan(&sessionStatus))
	assert.Equal(t, "complete", sessionStatus)

	var subStatus string
	require.NoError(t, db.QueryRow(`SELECT status FROM subscriptions WHERE subscription_id = $1`, "sub_retry_after_rb").Scan(&subStatus))
	assert.Equal(t, "active", subStatus)
}
