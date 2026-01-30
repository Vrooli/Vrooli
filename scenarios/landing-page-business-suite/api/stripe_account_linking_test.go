package main

import (
	"context"
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

// ============================================================================
// Account Linking Tests
// ============================================================================

// TestLinkUserToStripeCustomer_NewUser verifies linking a new user to a Stripe customer.
func TestLinkUserToStripeCustomer_NewUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
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

	service := NewStripeService(db)

	// Link a new user
	err = service.linkUserToStripeCustomer("newuser@example.com", "cus_new_123")
	require.NoError(t, err)

	// Verify the user was created with the correct customer ID
	var email, customerID string
	err = db.QueryRow(`SELECT email, stripe_customer_id FROM users WHERE email = $1`, "newuser@example.com").Scan(&email, &customerID)
	require.NoError(t, err)
	assert.Equal(t, "newuser@example.com", email)
	assert.Equal(t, "cus_new_123", customerID)
}

// TestLinkUserToStripeCustomer_ExistingUser_UpdatesCustomerID verifies that
// linking an existing user updates their Stripe customer ID.
func TestLinkUserToStripeCustomer_ExistingUser_UpdatesCustomerID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
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

	// Insert existing user with old customer ID
	_, err = db.Exec(`
		INSERT INTO users (email, stripe_customer_id)
		VALUES ($1, $2)
	`, "existing@example.com", "cus_old_123")
	require.NoError(t, err)

	service := NewStripeService(db)

	// Link with new customer ID
	err = service.linkUserToStripeCustomer("existing@example.com", "cus_new_456")
	require.NoError(t, err)

	// Verify the customer ID was updated
	var customerID string
	err = db.QueryRow(`SELECT stripe_customer_id FROM users WHERE email = $1`, "existing@example.com").Scan(&customerID)
	require.NoError(t, err)
	assert.Equal(t, "cus_new_456", customerID)
}

// TestLinkUserToStripeCustomer_EmailNormalized verifies that the email is
// normalized when linking users.
func TestLinkUserToStripeCustomer_EmailNormalized(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
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

	service := NewStripeService(db)

	// Link with uppercase email
	err = service.linkUserToStripeCustomer("UPPERCASE@EXAMPLE.COM", "cus_upper_123")
	require.NoError(t, err)

	// Verify it was stored as lowercase
	var email string
	err = db.QueryRow(`SELECT email FROM users WHERE stripe_customer_id = $1`, "cus_upper_123").Scan(&email)
	require.NoError(t, err)
	assert.Equal(t, "uppercase@example.com", email)
}

// TestLinkUserToStripeCustomer_RequiresEmailAndCustomerID verifies that both
// email and customer ID are required.
func TestLinkUserToStripeCustomer_RequiresEmailAndCustomerID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewStripeService(db)

	// Test empty email
	err := service.linkUserToStripeCustomer("", "cus_123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "email and customer ID are required")

	// Test empty customer ID
	err = service.linkUserToStripeCustomer("test@example.com", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "email and customer ID are required")

	// Test both empty
	err = service.linkUserToStripeCustomer("", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "email and customer ID are required")

	// Test whitespace only
	err = service.linkUserToStripeCustomer("  ", "   ")
	require.Error(t, err)
}

// TestAccountLink_CheckoutCompleted_LinksAccount verifies that checkout completion
// links the user account.
func TestAccountLink_CheckoutCompleted_LinksAccount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Create required tables
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

	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_link", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_link_test", "Link Plan", "pro", "month", "usd", 4900, false, "", 0, 0, "", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	// Pre-create checkout session
	_, err = db.Exec(`
		INSERT INTO checkout_sessions (session_id, customer_email, price_id, status, session_type)
		VALUES ($1, $2, $3, $4, $5)
	`, "cs_link_test", "checkout@example.com", "price_link_test", "open", sessionTypeSubscription)
	require.NoError(t, err)

	cfg := DefaultStripeTestConfig()
	service := ConfigureStripeService(t, db, cfg, nil)

	// Simulate checkout.session.completed webhook
	event := map[string]interface{}{
		"id":   "evt_checkout_link_123",
		"type": "checkout.session.completed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":             "cs_link_test",
				"customer_email": "checkout@example.com",
				"customer":       "cus_checkout_123",
				"subscription":   "sub_link_123",
				"amount_total":   4900,
			},
		},
	}

	payload, _ := json.Marshal(event)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	signedPayload := timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(cfg.WebhookSecret))
	mac.Write([]byte(signedPayload))
	signature := hex.EncodeToString(mac.Sum(nil))
	signatureHeader := "t=" + timestamp + ",v1=" + signature

	err = service.HandleWebhook(payload, signatureHeader)
	require.NoError(t, err)

	// Verify user was linked
	var customerID sql.NullString
	err = db.QueryRow(`SELECT stripe_customer_id FROM users WHERE email = $1`, "checkout@example.com").Scan(&customerID)
	require.NoError(t, err)
	assert.True(t, customerID.Valid)
	assert.Equal(t, "cus_checkout_123", customerID.String)
}

// TestAccountLink_CreditsTopup_LinksAccount verifies that credit topup links
// the user account.
func TestAccountLink_CreditsTopup_LinksAccount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
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

	productID := upsertTestBundleProduct(t, db, "credits_bundle", "Credits", "prod_credits_link", "production", 100, 1, "credits")
	insertBundlePrice(t, db, productID, "price_credits_link", "Credits Pack", "credits", "one_time", "usd", 1000, false, "", 0, 0, "", 0, 100, 1, 0, "none", sessionTypeCreditsTopup, map[string]interface{}{})

	// Pre-create checkout session
	_, err = db.Exec(`
		INSERT INTO checkout_sessions (session_id, customer_email, price_id, status, session_type, amount_cents)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, "cs_credits_link", "credits@example.com", "price_credits_link", "open", sessionTypeCreditsTopup, 1000)
	require.NoError(t, err)

	cfg := DefaultStripeTestConfig()
	service := ConfigureStripeService(t, db, cfg, nil)

	// Simulate checkout.session.completed webhook for credits
	event := map[string]interface{}{
		"id":   "evt_credits_link_123",
		"type": "checkout.session.completed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":             "cs_credits_link",
				"customer_email": "credits@example.com",
				"customer":       "cus_credits_link",
				"amount_total":   1000,
			},
		},
	}

	payload, _ := json.Marshal(event)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	signedPayload := timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(cfg.WebhookSecret))
	mac.Write([]byte(signedPayload))
	signature := hex.EncodeToString(mac.Sum(nil))
	signatureHeader := "t=" + timestamp + ",v1=" + signature

	err = service.HandleWebhook(payload, signatureHeader)
	require.NoError(t, err)

	// Verify user was linked
	var customerID sql.NullString
	err = db.QueryRow(`SELECT stripe_customer_id FROM users WHERE email = $1`, "credits@example.com").Scan(&customerID)
	require.NoError(t, err)
	assert.True(t, customerID.Valid)
	assert.Equal(t, "cus_credits_link", customerID.String)

	// Verify credits were added
	var balance int64
	err = db.QueryRow(`SELECT balance_credits FROM credit_wallets WHERE customer_email = $1`, "credits@example.com").Scan(&balance)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), balance)
}

// ============================================================================
// Customer Lookup Tests
// ============================================================================

// TestLookupCustomerID_ByEmail verifies looking up a customer ID by email.
func TestLookupCustomerID_ByEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
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

	// Insert subscription with email
	_, err = db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status)
		VALUES ($1, $2, $3, $4)
	`, "sub_lookup_email", "cus_lookup_email", "lookup@example.com", "active")
	require.NoError(t, err)

	service := NewStripeService(db)

	// Lookup by email
	customerID := service.lookupCustomerID("lookup@example.com")
	assert.Equal(t, "cus_lookup_email", customerID)

	// Lookup by uppercase email (case insensitive)
	customerID = service.lookupCustomerID("LOOKUP@EXAMPLE.COM")
	assert.Equal(t, "cus_lookup_email", customerID)
}

// TestLookupCustomerID_ByCustomerID verifies looking up by customer ID directly.
func TestLookupCustomerID_ByCustomerID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
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

	// Insert subscription
	_, err = db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status)
		VALUES ($1, $2, $3, $4)
	`, "sub_lookup_cus", "cus_direct_lookup", "direct@example.com", "active")
	require.NoError(t, err)

	service := NewStripeService(db)

	// Lookup by customer ID directly
	customerID := service.lookupCustomerID("cus_direct_lookup")
	assert.Equal(t, "cus_direct_lookup", customerID)
}

// TestLookupCustomerID_NotFound verifies behavior when customer is not found.
func TestLookupCustomerID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
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

	service := NewStripeService(db)

	// Lookup non-existent email
	customerID := service.lookupCustomerID("notfound@example.com")
	assert.Equal(t, "", customerID)

	// Lookup non-existent customer ID
	customerID = service.lookupCustomerID("cus_nonexistent")
	assert.Equal(t, "", customerID)

	// Lookup empty string
	customerID = service.lookupCustomerID("")
	assert.Equal(t, "", customerID)
}

// TestLookupCustomerID_MostRecentSubscription verifies that the most recent
// subscription is used for lookup.
func TestLookupCustomerID_MostRecentSubscription(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
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

	now := time.Now()

	// Insert older subscription
	_, err = db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`, "sub_old", "cus_old", "multisubscription@example.com", "canceled", now.Add(-24*time.Hour))
	require.NoError(t, err)

	// Insert newer subscription
	_, err = db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`, "sub_new", "cus_new", "multisubscription@example.com", "active", now)
	require.NoError(t, err)

	service := NewStripeService(db)

	// Should return the most recent (cus_new)
	customerID := service.lookupCustomerID("multisubscription@example.com")
	assert.Equal(t, "cus_new", customerID)
}

// ============================================================================
// Email Migration Tests
// ============================================================================

// TestCustomerUpdated_EmailMigration_AllTables verifies that email changes are
// propagated to all relevant tables.
func TestCustomerUpdated_EmailMigration_AllTables(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
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
	customerID := "cus_migrate_all"

	// Insert data in all tables with old email
	_, err = db.Exec(`INSERT INTO users (email, stripe_customer_id) VALUES ($1, $2)`, oldEmail, customerID)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status) VALUES ($1, $2, $3, $4)`,
		"sub_migrate_all", customerID, oldEmail, "active")
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO credit_wallets (customer_email, balance_credits) VALUES ($1, $2)`, oldEmail, 1000)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO credit_transactions (customer_email, amount_credits, transaction_type) VALUES ($1, $2, $3)`,
		oldEmail, 1000, "credit_topup")
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO intro_coupon_usage (email, stripe_customer_id, coupon_id) VALUES ($1, $2, $3)`,
		oldEmail, customerID, "coupon_migrate")
	require.NoError(t, err)

	service := NewStripeService(db)

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

	// Verify all tables were updated
	var email string

	// Check users table
	err = db.QueryRow(`SELECT email FROM users WHERE stripe_customer_id = $1`, customerID).Scan(&email)
	require.NoError(t, err)
	assert.Equal(t, newEmail, email)

	// Check subscriptions table
	err = db.QueryRow(`SELECT customer_email FROM subscriptions WHERE customer_id = $1`, customerID).Scan(&email)
	require.NoError(t, err)
	assert.Equal(t, newEmail, email)

	// Check credit_wallets table
	var balance int64
	err = db.QueryRow(`SELECT balance_credits FROM credit_wallets WHERE customer_email = $1`, newEmail).Scan(&balance)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), balance)

	// Verify old email no longer exists in credit_wallets
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM credit_wallets WHERE customer_email = $1`, oldEmail).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// Check credit_transactions table
	err = db.QueryRow(`SELECT customer_email FROM credit_transactions WHERE amount_credits = 1000`).Scan(&email)
	require.NoError(t, err)
	assert.Equal(t, newEmail, email)

	// Check intro_coupon_usage table
	err = db.QueryRow(`SELECT email FROM intro_coupon_usage WHERE stripe_customer_id = $1`, customerID).Scan(&email)
	require.NoError(t, err)
	assert.Equal(t, newEmail, email)
}

// TestCustomerUpdated_EmailMigration_NoPreviousAttributes verifies behavior
// when previous_attributes is not provided in the webhook.
func TestCustomerUpdated_EmailMigration_NoPreviousAttributes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
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

	oldEmail := "lookup_old@example.com"
	newEmail := "lookup_new@example.com"
	customerID := "cus_lookup_migrate"

	// Insert subscription with old email
	_, err = db.Exec(`INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status) VALUES ($1, $2, $3, $4)`,
		"sub_lookup_migrate", customerID, oldEmail, "active")
	require.NoError(t, err)

	service := NewStripeService(db)

	// Simulate customer.updated webhook without previous_attributes
	customerObj := map[string]interface{}{
		"id":    customerID,
		"email": newEmail,
	}

	err = service.handleCustomerUpdated(customerObj)
	require.NoError(t, err)

	// Verify subscription was updated (old email was looked up from DB)
	var email string
	err = db.QueryRow(`SELECT customer_email FROM subscriptions WHERE customer_id = $1`, customerID).Scan(&email)
	require.NoError(t, err)
	assert.Equal(t, newEmail, email)
}

// TestCustomerUpdated_SameEmail_NoOp verifies that no changes are made when
// old and new emails are the same.
func TestCustomerUpdated_SameEmail_NoOp(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
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

	email := "same@example.com"
	customerID := "cus_same_email"

	// Insert subscription
	originalTime := time.Now().UTC().Add(-1 * time.Hour)
	_, err = db.Exec(`INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, updated_at) VALUES ($1, $2, $3, $4, $5)`,
		"sub_same_email", customerID, email, "active", originalTime)
	require.NoError(t, err)

	service := NewStripeService(db)

	// Simulate customer.updated webhook with same email
	customerObj := map[string]interface{}{
		"id":    customerID,
		"email": email,
		"previous_attributes": map[string]interface{}{
			"email": email,
		},
	}

	err = service.handleCustomerUpdated(customerObj)
	require.NoError(t, err)

	// Verify no changes were made (updated_at should be unchanged)
	var updatedAt time.Time
	err = db.QueryRow(`SELECT updated_at FROM subscriptions WHERE customer_id = $1`, customerID).Scan(&updatedAt)
	require.NoError(t, err)
	// The timestamps should be within a second of each other (no update happened)
	// Compare in UTC to avoid timezone issues
	assert.WithinDuration(t, originalTime, updatedAt.UTC(), 2*time.Second)
}

// TestCustomerUpdated_MissingEmail_NoOp verifies no crash when email is missing.
func TestCustomerUpdated_MissingEmail_NoOp(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewStripeService(db)

	// Simulate customer.updated webhook without email
	customerObj := map[string]interface{}{
		"id": "cus_no_email",
	}

	err := service.handleCustomerUpdated(customerObj)
	require.NoError(t, err) // Should be a no-op, not an error
}

// ============================================================================
// Repository Layer Tests
// ============================================================================

// TestStripeRepository_LinkUserToStripeCustomer verifies the repository method.
func TestStripeRepository_LinkUserToStripeCustomer(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
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

	repo := NewStripeRepository(db)

	// Test new user
	err = repo.LinkUserToStripeCustomer("repo@example.com", "cus_repo_123")
	require.NoError(t, err)

	var customerID string
	err = db.QueryRow(`SELECT stripe_customer_id FROM users WHERE email = $1`, "repo@example.com").Scan(&customerID)
	require.NoError(t, err)
	assert.Equal(t, "cus_repo_123", customerID)

	// Test update existing user
	err = repo.LinkUserToStripeCustomer("repo@example.com", "cus_repo_456")
	require.NoError(t, err)

	err = db.QueryRow(`SELECT stripe_customer_id FROM users WHERE email = $1`, "repo@example.com").Scan(&customerID)
	require.NoError(t, err)
	assert.Equal(t, "cus_repo_456", customerID)
}

// TestStripeRepository_MigrateCustomerEmail verifies the repository migration method.
func TestStripeRepository_MigrateCustomerEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
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

	oldEmail := "repo_old@example.com"
	newEmail := "repo_new@example.com"
	customerID := "cus_repo_migrate"

	// Insert test data
	_, err = db.Exec(`INSERT INTO users (email, stripe_customer_id) VALUES ($1, $2)`, oldEmail, customerID)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status) VALUES ($1, $2, $3, $4)`,
		"sub_repo", customerID, oldEmail, "active")
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO credit_wallets (customer_email, balance_credits) VALUES ($1, $2)`, oldEmail, 500)
	require.NoError(t, err)

	repo := NewStripeRepository(db)

	err = repo.MigrateCustomerEmail(context.Background(), oldEmail, newEmail, customerID)
	require.NoError(t, err)

	// Verify all tables were updated
	var email string
	err = db.QueryRow(`SELECT email FROM users WHERE stripe_customer_id = $1`, customerID).Scan(&email)
	require.NoError(t, err)
	assert.Equal(t, newEmail, email)

	err = db.QueryRow(`SELECT customer_email FROM subscriptions WHERE customer_id = $1`, customerID).Scan(&email)
	require.NoError(t, err)
	assert.Equal(t, newEmail, email)

	var balance int64
	err = db.QueryRow(`SELECT balance_credits FROM credit_wallets WHERE customer_email = $1`, newEmail).Scan(&balance)
	require.NoError(t, err)
	assert.Equal(t, int64(500), balance)
}

// TestStripeRepository_LookupCustomerID verifies the repository lookup method.
func TestStripeRepository_LookupCustomerID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
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

	// Insert subscription
	_, err = db.Exec(`INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status) VALUES ($1, $2, $3, $4)`,
		"sub_repo_lookup", "cus_repo_lookup", "repo_lookup@example.com", "active")
	require.NoError(t, err)

	repo := NewStripeRepository(db)

	// Lookup by email
	customerID := repo.LookupCustomerID("repo_lookup@example.com")
	assert.Equal(t, "cus_repo_lookup", customerID)

	// Lookup by customer ID
	customerID = repo.LookupCustomerID("cus_repo_lookup")
	assert.Equal(t, "cus_repo_lookup", customerID)

	// Case insensitive lookup
	customerID = repo.LookupCustomerID("REPO_LOOKUP@EXAMPLE.COM")
	assert.Equal(t, "cus_repo_lookup", customerID)

	// Not found
	customerID = repo.LookupCustomerID("notfound@example.com")
	assert.Equal(t, "", customerID)

	// Empty input
	customerID = repo.LookupCustomerID("")
	assert.Equal(t, "", customerID)
}
