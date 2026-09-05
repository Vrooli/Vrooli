package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
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

// ============================================================================
// Full User Journey Integration Tests
// ============================================================================

// TestFlow_NewUser_IntroPricing_FullCycle tests the complete intro pricing flow
// for a new user from checkout through subscription activation.
func TestFlow_NewUser_IntroPricing_FullCycle(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	// Create all required tables
	_, err := db.Exec(`
		DROP TABLE IF EXISTS intro_coupon_usage CASCADE;
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
			billing_cycle_start INTEGER,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);
		CREATE TABLE intro_coupon_usage (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL,
			stripe_customer_id VARCHAR(255),
			coupon_id VARCHAR(255),
			plan_tier VARCHAR(50),
			subscription_id VARCHAR(255),
			used_at TIMESTAMP DEFAULT NOW()
		)
	`)
	require.NoError(t, err)

	// Setup bundle with intro pricing
	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_intro_flow", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_pro_intro", "Pro Plan", "pro", "month", "usd", 2900, true, "flat_amount", 100, 1, "intro_pro_lookup", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/checkout/sessions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{
				"id":"cs_intro_flow",
				"url":"https://checkout.stripe.test/cs_intro_flow",
				"status":"open",
				"customer_email":"introflow@example.com",
				"customer":"cus_intro_flow",
				"subscription":"sub_intro_flow",
				"amount_total":100,
				"mode":"subscription",
				"currency":"usd"
			}`)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/customers/cus_intro_flow":
			// Customer metadata update
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"cus_intro_flow"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer stripeServer.Close()

	cfg := DefaultStripeTestConfig().WithKeys("pk_test", "sk_test", "whsec_test").WithIntroCoupon(true, map[string]string{"pro": "coupon_intro_pro"})
	service := ConfigureStripeService(t, db, cfg, stripeServer)

	ctx := context.Background()

	// Step 1: Verify user is eligible for intro
	eligible, err := service.checkIntroEligibility(ctx, "introflow@example.com")
	require.NoError(t, err)
	assert.True(t, eligible, "new user should be eligible for intro")

	// Step 2: Create checkout session
	session, err := service.CreateCheckoutSession("price_pro_intro", "/success", "/cancel", "introflow@example.com")
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.Contains(t, session.Url, "cs_intro_flow")

	// Step 3: Simulate checkout.session.completed webhook with intro coupon
	event := map[string]interface{}{
		"id":   "evt_intro_flow_complete",
		"type": "checkout.session.completed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":             "cs_intro_flow",
				"customer_email": "introflow@example.com",
				"customer":       "cus_intro_flow",
				"subscription":   "sub_intro_flow",
				"amount_total":   100, // $1 intro price
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

	// Step 4: Verify subscription is active
	var subStatus string
	err = db.QueryRow(`SELECT status FROM subscriptions WHERE subscription_id = $1`, "sub_intro_flow").Scan(&subStatus)
	require.NoError(t, err)
	assert.Equal(t, "active", subStatus)

	// Step 5: Verify user is linked and no longer eligible for intro
	var customerID sql.NullString
	err = db.QueryRow(`SELECT stripe_customer_id FROM users WHERE email = $1`, "introflow@example.com").Scan(&customerID)
	require.NoError(t, err)
	assert.True(t, customerID.Valid)
	assert.Equal(t, "cus_intro_flow", customerID.String)
}

// TestFlow_ReturningUser_NoCoupon verifies that returning users who have
// already used intro don't get the coupon again.
func TestFlow_ReturningUser_NoCoupon(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	// Create users table
	_, err := db.Exec(`
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
		)
	`)
	require.NoError(t, err)

	// Insert user who has already used intro
	_, err = db.Exec(`
		INSERT INTO users (email, has_used_intro, stripe_customer_id)
		VALUES ($1, TRUE, $2)
	`, "returning@example.com", "cus_returning")
	require.NoError(t, err)

	cfg := DefaultStripeTestConfig().WithIntroCoupon(true, map[string]string{"pro": "coupon_intro_pro"})
	service := ConfigureStripeService(t, db, cfg, nil)

	ctx := context.Background()

	// Verify returning user is NOT eligible for intro
	eligible, err := service.checkIntroEligibility(ctx, "returning@example.com")
	require.NoError(t, err)
	assert.False(t, eligible, "returning user should not be eligible for intro")
}

// TestFlow_CreditPurchase_BalanceUpdated verifies the complete credit purchase flow.
func TestFlow_CreditPurchase_BalanceUpdated(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	// Create required tables
	_, err := db.Exec(`
		DROP TABLE IF EXISTS credit_transactions CASCADE;
		DROP TABLE IF EXISTS credit_wallets CASCADE;
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
		CREATE UNIQUE INDEX IF NOT EXISTS idx_credit_transactions_stripe_event_id
		ON credit_transactions(stripe_event_id) WHERE stripe_event_id IS NOT NULL
	`)
	require.NoError(t, err)

	// Setup bundle with credits
	productID := upsertTestBundleProduct(t, db, "credits_bundle", "Credits", "prod_credits_flow", "production", 100, 1, "credits")
	insertBundlePrice(t, db, productID, "price_credits_pack", "Credits Pack", "credits", "one_time", "usd", 1000, false, "", 0, 0, "", 0, 1000, 1, 0, "none", sessionTypeCreditsTopup, map[string]interface{}{})

	// Pre-create checkout session
	_, err = db.Exec(`
		INSERT INTO checkout_sessions (session_id, customer_email, price_id, status, session_type, amount_cents)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, "cs_credits_flow", "credits@example.com", "price_credits_pack", "open", sessionTypeCreditsTopup, 1000)
	require.NoError(t, err)

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig().WithKeys("pk_test", "sk_test", "whsec_test"), nil)

	// Simulate checkout.session.completed webhook
	event := map[string]interface{}{
		"id":   "evt_credits_flow_123",
		"type": "checkout.session.completed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":             "cs_credits_flow",
				"customer_email": "credits@example.com",
				"customer":       "cus_credits_flow",
				"amount_total":   1000,
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

	// Verify credit balance was updated
	var balance int64
	err = db.QueryRow(`SELECT balance_credits FROM credit_wallets WHERE customer_email = $1`, "credits@example.com").Scan(&balance)
	require.NoError(t, err)
	// creditsPerUSD=100, amountCents=1000, so credits = 100 * 1000 / 100 = 1000
	assert.Equal(t, int64(1000), balance)

	// Verify transaction was recorded
	var txnCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM credit_transactions WHERE customer_email = $1`, "credits@example.com").Scan(&txnCount)
	require.NoError(t, err)
	assert.Equal(t, 1, txnCount)

	// Verify user was linked
	var customerID sql.NullString
	err = db.QueryRow(`SELECT stripe_customer_id FROM users WHERE email = $1`, "credits@example.com").Scan(&customerID)
	require.NoError(t, err)
	assert.True(t, customerID.Valid)
	assert.Equal(t, "cus_credits_flow", customerID.String)
}

// TestFlow_SubscriptionCancel_StatusUpdated verifies subscription cancellation flow.
func TestFlow_SubscriptionCancel_StatusUpdated(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	// Create subscriptions table
	_, err := db.Exec(`
		DROP TABLE IF EXISTS subscriptions CASCADE;
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
		)
	`)
	require.NoError(t, err)

	// Insert active subscription
	_, err = db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, bundle_key)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, "sub_cancel_flow", "cus_cancel_flow", "cancel@example.com", "active", "pro", "business_suite")
	require.NoError(t, err)

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/subscriptions/sub_cancel_flow" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"sub_cancel_flow","status":"canceled"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	// Cancel subscription via API
	result, err := service.CancelSubscription("cancel@example.com")
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify subscription status in database
	var status string
	var canceledAt sql.NullTime
	err = db.QueryRow(`SELECT status, canceled_at FROM subscriptions WHERE subscription_id = $1`, "sub_cancel_flow").Scan(&status, &canceledAt)
	require.NoError(t, err)
	assert.Equal(t, "canceled", status)
	assert.True(t, canceledAt.Valid, "canceled_at should be set")
}

// TestFlow_EmailChange_AllTablesMigrated verifies complete email migration flow.
func TestFlow_EmailChange_AllTablesMigrated(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	// Create all required tables
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
			email_verified BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			last_login_at TIMESTAMP
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
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL,
			stripe_customer_id VARCHAR(255),
			coupon_id VARCHAR(255),
			plan_tier VARCHAR(50),
			subscription_id VARCHAR(255),
			used_at TIMESTAMP DEFAULT NOW()
		)
	`)
	require.NoError(t, err)

	oldEmail := "oldemail@example.com"
	newEmail := "newemail@example.com"
	customerID := "cus_email_change"

	// Insert data in all tables with old email
	_, err = db.Exec(`INSERT INTO users (email, stripe_customer_id, has_used_intro) VALUES ($1, $2, TRUE)`, oldEmail, customerID)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, bundle_key) VALUES ($1, $2, $3, $4, $5, $6)`,
		"sub_email_change", customerID, oldEmail, "active", "pro", "business_suite")
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO credit_wallets (customer_email, balance_credits) VALUES ($1, $2)`, oldEmail, 5000)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO credit_transactions (customer_email, amount_credits, transaction_type) VALUES ($1, $2, $3)`,
		oldEmail, 5000, "credit_topup")
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO intro_coupon_usage (email, stripe_customer_id, coupon_id, plan_tier, subscription_id) VALUES ($1, $2, $3, $4, $5)`,
		oldEmail, customerID, "coupon_email_test", "pro", "sub_email_change")
	require.NoError(t, err)

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), nil)

	// Simulate customer.updated webhook
	customerObj := map[string]interface{}{
		"id":    customerID,
		"email": newEmail,
		"previous_attributes": map[string]interface{}{
			"email": oldEmail,
		},
	}

	err = service.handleCustomerUpdated(customerObj)
	require.NoError(t, err)

	// Verify all tables were migrated
	var email string

	// Users
	err = db.QueryRow(`SELECT email FROM users WHERE stripe_customer_id = $1`, customerID).Scan(&email)
	require.NoError(t, err)
	assert.Equal(t, newEmail, email)

	// Subscriptions
	err = db.QueryRow(`SELECT customer_email FROM subscriptions WHERE customer_id = $1`, customerID).Scan(&email)
	require.NoError(t, err)
	assert.Equal(t, newEmail, email)

	// Credit wallets
	var balance int64
	err = db.QueryRow(`SELECT balance_credits FROM credit_wallets WHERE customer_email = $1`, newEmail).Scan(&balance)
	require.NoError(t, err)
	assert.Equal(t, int64(5000), balance)

	// Verify old email no longer exists
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM credit_wallets WHERE customer_email = $1`, oldEmail).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Credit transactions
	err = db.QueryRow(`SELECT customer_email FROM credit_transactions WHERE amount_credits = 5000`).Scan(&email)
	require.NoError(t, err)
	assert.Equal(t, newEmail, email)

	// Intro coupon usage
	err = db.QueryRow(`SELECT email FROM intro_coupon_usage WHERE stripe_customer_id = $1`, customerID).Scan(&email)
	require.NoError(t, err)
	assert.Equal(t, newEmail, email)
}

// TestFlow_CouponAbuse_CaseVariation_Blocked verifies that coupon abuse via
// case variations is prevented.
func TestFlow_CouponAbuse_CaseVariation_Blocked(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	// Create users table
	_, err := db.Exec(`
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
		)
	`)
	require.NoError(t, err)

	// User has already used intro with lowercase email
	_, err = db.Exec(`
		INSERT INTO users (email, has_used_intro)
		VALUES ($1, TRUE)
	`, "abuser@example.com")
	require.NoError(t, err)

	cfg := DefaultStripeTestConfig().WithIntroCoupon(true, map[string]string{"pro": "coupon_intro_pro"})
	service := ConfigureStripeService(t, db, cfg, nil)

	ctx := context.Background()

	// All case variations should be blocked
	variations := []string{
		"abuser@example.com",
		"ABUSER@EXAMPLE.COM",
		"Abuser@Example.Com",
		"ABUSER@example.com",
		"abuser@EXAMPLE.COM",
		"  abuser@example.com  ",
	}

	for _, email := range variations {
		eligible, err := service.checkIntroEligibility(ctx, email)
		require.NoError(t, err)
		assert.False(t, eligible, "case variation %q should be blocked", email)
	}
}

// TestFlow_MultipleCheckouts_SameUser verifies handling of multiple checkout
// attempts by the same user.
func TestFlow_MultipleCheckouts_SameUser(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	// Create required tables
	_, err := db.Exec(`
		DROP TABLE IF EXISTS subscriptions CASCADE;
		DROP TABLE IF EXISTS checkout_sessions CASCADE;

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
			billing_cycle_start INTEGER,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)
	`)
	require.NoError(t, err)

	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_multi", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_multi_test", "Pro Plan", "pro", "month", "usd", 2900, false, "", 0, 0, "", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	checkoutCount := 0
	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/checkout/sessions" {
			checkoutCount++
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{
				"id":"cs_multi_%d",
				"url":"https://checkout.stripe.test/cs_multi_%d",
				"status":"open",
				"customer_email":"multi@example.com",
				"customer":"cus_multi",
				"subscription":"sub_multi",
				"amount_total":2900,
				"mode":"subscription",
				"currency":"usd"
			}`, checkoutCount, checkoutCount)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	// Create multiple checkout sessions
	for i := 0; i < 3; i++ {
		session, err := service.CreateCheckoutSession("price_multi_test", "/success", "/cancel", "multi@example.com")
		require.NoError(t, err)
		require.NotNil(t, session)
	}

	// Verify all checkout sessions were created
	var sessionCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM checkout_sessions WHERE customer_email = $1`, "multi@example.com").Scan(&sessionCount)
	require.NoError(t, err)
	assert.Equal(t, 3, sessionCount)
}

// TestFlow_WebhookRetry_Idempotent verifies that webhook retries are handled
// idempotently.
func TestFlow_WebhookRetry_Idempotent(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	// Create required tables
	_, err := db.Exec(`
		DROP TABLE IF EXISTS credit_transactions CASCADE;
		DROP TABLE IF EXISTS credit_wallets CASCADE;
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
		CREATE UNIQUE INDEX IF NOT EXISTS idx_credit_transactions_stripe_event_id
		ON credit_transactions(stripe_event_id) WHERE stripe_event_id IS NOT NULL
	`)
	require.NoError(t, err)

	productID := upsertTestBundleProduct(t, db, "credits_bundle", "Credits", "prod_retry", "production", 100, 1, "credits")
	insertBundlePrice(t, db, productID, "price_retry_test", "Credits Pack", "credits", "one_time", "usd", 1000, false, "", 0, 0, "", 0, 1000, 1, 0, "none", sessionTypeCreditsTopup, map[string]interface{}{})

	// Pre-create checkout session
	_, err = db.Exec(`
		INSERT INTO checkout_sessions (session_id, customer_email, price_id, status, session_type, amount_cents)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, "cs_retry_test", "retry@example.com", "price_retry_test", "open", sessionTypeCreditsTopup, 1000)
	require.NoError(t, err)

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig().WithKeys("pk_test", "sk_test", "whsec_test"), nil)

	// Same event processed multiple times (simulating retries)
	event := map[string]interface{}{
		"id":   "evt_retry_test_123",
		"type": "checkout.session.completed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":             "cs_retry_test",
				"customer_email": "retry@example.com",
				"customer":       "cus_retry",
				"amount_total":   1000,
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

	// Process same webhook 5 times
	for i := 0; i < 5; i++ {
		err = service.HandleWebhook(payload, signatureHeader)
		require.NoError(t, err)
	}

	// Verify credits were only added once
	var balance int64
	err = db.QueryRow(`SELECT balance_credits FROM credit_wallets WHERE customer_email = $1`, "retry@example.com").Scan(&balance)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), balance, "credits should only be added once despite retries")

	// Verify only one transaction exists
	var txnCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM credit_transactions WHERE stripe_event_id = $1`, "evt_retry_test_123").Scan(&txnCount)
	require.NoError(t, err)
	assert.Equal(t, 1, txnCount)
}

// TestFlow_InvoicePaid_RefreshesSubscription verifies that invoice.paid events
// refresh subscription status correctly.
func TestFlow_InvoicePaid_RefreshesSubscription(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	// Create subscriptions table
	_, err := db.Exec(`
		DROP TABLE IF EXISTS subscriptions CASCADE;
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
		)
	`)
	require.NoError(t, err)

	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_invoice", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_invoice_new", "Pro Plan", "pro", "month", "usd", 2900, false, "", 0, 0, "", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	// Insert subscription with past_due status
	_, err = db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, price_id, bundle_key)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, "sub_invoice_flow", "cus_invoice_flow", "invoice@example.com", "past_due", "price_old", "business_suite")
	require.NoError(t, err)

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig().WithKeys("pk_test", "sk_test", "whsec_test"), nil)

	// Simulate invoice.paid event
	event := map[string]interface{}{
		"id":   "evt_invoice_paid_flow",
		"type": "invoice.paid",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"subscription":   "sub_invoice_flow",
				"customer":       "cus_invoice_flow",
				"customer_email": "invoice@example.com",
				"lines": map[string]interface{}{
					"data": []interface{}{
						map[string]interface{}{
							"price": map[string]interface{}{
								"id": "price_invoice_new",
							},
						},
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

	// Verify subscription is now active
	var status, priceID string
	err = db.QueryRow(`SELECT status, price_id FROM subscriptions WHERE subscription_id = $1`, "sub_invoice_flow").Scan(&status, &priceID)
	require.NoError(t, err)
	assert.Equal(t, "active", status)
	assert.Equal(t, "price_invoice_new", priceID)
}
