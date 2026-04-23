package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCheckoutSession_NormalizesEmail verifies that CreateCheckoutSession normalizes
// email addresses before making Stripe API calls.
func TestCheckoutSession_NormalizesEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Create checkout_sessions table
	_, err := db.Exec(`
		DROP TABLE IF EXISTS checkout_sessions CASCADE;
		DROP TABLE IF EXISTS subscriptions CASCADE;
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

	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_test", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_email_test", "Test Plan", "pro", "month", "usd", 5000, false, "", 0, 0, "", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	// Track what email was sent to Stripe
	var receivedEmail string
	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/checkout/sessions" && r.Method == http.MethodPost {
			if err := r.ParseForm(); err == nil {
				receivedEmail = r.FormValue("customer_email")
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"id":"cs_email_test","url":"https://stripe.test/cs","status":"open","customer_email":"%s","amount_total":5000,"mode":"subscription","currency":"usd"}`, receivedEmail)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	tests := []struct {
		name          string
		inputEmail    string
		expectedEmail string
	}{
		{
			name:          "uppercase email normalized",
			inputEmail:    "TEST@EXAMPLE.COM",
			expectedEmail: "test@example.com",
		},
		{
			name:          "mixed case email normalized",
			inputEmail:    "Test.User@Example.COM",
			expectedEmail: "test.user@example.com",
		},
		{
			name:          "email with leading/trailing spaces trimmed",
			inputEmail:    "  user@example.com  ",
			expectedEmail: "user@example.com",
		},
		{
			name:          "already lowercase unchanged",
			inputEmail:    "user@example.com",
			expectedEmail: "user@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receivedEmail = "" // Reset

			session, err := service.CreateCheckoutSession(
				"price_email_test",
				"/success",
				"/cancel",
				tt.inputEmail,
			)
			require.NoError(t, err)
			require.NotNil(t, session)

			// Verify Stripe received normalized email
			assert.Equal(t, tt.expectedEmail, receivedEmail, "email sent to Stripe should be normalized")
		})
	}
}

// TestIntroEligibility_CaseInsensitive verifies that users cannot use the intro
// coupon twice by using different email case variations.
func TestIntroEligibility_CaseInsensitive(t *testing.T) {
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

	service := ConfigureStripeServiceSimple(t, db)

	// Insert user with lowercase email who has used intro
	_, err = db.Exec(`
		INSERT INTO users (email, has_used_intro, stripe_customer_id)
		VALUES ($1, TRUE, $2)
	`, "used@example.com", "cus_used_intro")
	require.NoError(t, err)

	ctx := context.Background()

	tests := []struct {
		name             string
		email            string
		expectedEligible bool
	}{
		{
			name:             "lowercase matches existing",
			email:            "used@example.com",
			expectedEligible: false,
		},
		{
			name:             "uppercase matches existing (case insensitive)",
			email:            "USED@EXAMPLE.COM",
			expectedEligible: false,
		},
		{
			name:             "mixed case matches existing",
			email:            "Used@Example.COM",
			expectedEligible: false,
		},
		{
			name:             "different user is eligible",
			email:            "new@example.com",
			expectedEligible: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eligible, err := service.checkIntroEligibility(ctx, tt.email)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedEligible, eligible)
		})
	}
}

// TestLookupCustomerID_CaseInsensitive verifies that customer lookup works
// regardless of email case.
func TestLookupCustomerID_CaseInsensitive(t *testing.T) {
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

	// Insert subscription with lowercase email
	_, err = db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status)
		VALUES ($1, $2, $3, $4)
	`, "sub_lookup_test", "cus_lookup_123", "lookup@example.com", "active")
	require.NoError(t, err)

	service := ConfigureStripeServiceSimple(t, db)

	tests := []struct {
		name               string
		lookupEmail        string
		expectedCustomerID string
	}{
		{
			name:               "exact match",
			lookupEmail:        "lookup@example.com",
			expectedCustomerID: "cus_lookup_123",
		},
		{
			name:               "uppercase lookup",
			lookupEmail:        "LOOKUP@EXAMPLE.COM",
			expectedCustomerID: "cus_lookup_123",
		},
		{
			name:               "mixed case lookup",
			lookupEmail:        "Lookup@Example.COM",
			expectedCustomerID: "cus_lookup_123",
		},
		{
			name:               "non-existent email",
			lookupEmail:        "nonexistent@example.com",
			expectedCustomerID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customerID := service.lookupCustomerID(tt.lookupEmail)
			assert.Equal(t, tt.expectedCustomerID, customerID)
		})
	}
}

// TestVerifySubscription_MixedCaseMatch verifies that subscription verification
// works with any email case.
func TestVerifySubscription_MixedCaseMatch(t *testing.T) {
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

	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_test", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_verify_test", "Pro Plan", "pro", "month", "usd", 5000, false, "", 0, 0, "", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	// Insert active subscription with lowercase email
	_, err = db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status, plan_tier, price_id, bundle_key, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, "sub_verify_test", "cus_verify_123", "verify@example.com", "active", "pro", "price_verify_test", "business_suite", time.Now())
	require.NoError(t, err)

	// Set up Stripe mock (for potential refresh calls)
	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)
	service.checkoutCacheTTL = 1 * time.Hour // Prevent refresh attempts

	tests := []struct {
		name         string
		userIdentity string
		expectActive bool
	}{
		{
			name:         "lowercase email finds subscription",
			userIdentity: "verify@example.com",
			expectActive: true,
		},
		{
			name:         "uppercase email finds subscription",
			userIdentity: "VERIFY@EXAMPLE.COM",
			expectActive: true,
		},
		{
			name:         "mixed case email finds subscription",
			userIdentity: "Verify@Example.COM",
			expectActive: true,
		},
		{
			name:         "customer ID also works",
			userIdentity: "cus_verify_123",
			expectActive: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := service.VerifySubscription(tt.userIdentity)
			require.NoError(t, err)
			require.NotNil(t, status)

			if tt.expectActive {
				assert.Equal(t, "SUBSCRIPTION_STATE_ACTIVE", status.State.String())
			}
		})
	}
}

// TestMarkIntroUsed_EmailNormalization verifies that markIntroUsed normalizes
// emails when recording intro usage.
func TestMarkIntroUsed_EmailNormalization(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Create required tables
	_, err := db.Exec(`
		DROP TABLE IF EXISTS intro_coupon_usage CASCADE;
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

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Accept customer metadata updates
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"cus_intro_test"}`)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	ctx := context.Background()

	// Mark intro used with mixed case email
	err = service.markIntroUsed(ctx, "MixedCase@Example.COM", "cus_intro_test", "coupon_test", "pro", "sub_intro_test")
	require.NoError(t, err)

	// Verify it was stored with normalized (lowercase) email
	var storedEmail string
	var hasUsedIntro bool
	err = db.QueryRow(`SELECT email, has_used_intro FROM users WHERE email = $1`, "mixedcase@example.com").Scan(&storedEmail, &hasUsedIntro)
	require.NoError(t, err)
	assert.Equal(t, "mixedcase@example.com", storedEmail)
	assert.True(t, hasUsedIntro)

	// Verify intro_coupon_usage also has normalized email
	var usageEmail string
	err = db.QueryRow(`SELECT email FROM intro_coupon_usage WHERE stripe_customer_id = $1`, "cus_intro_test").Scan(&usageEmail)
	require.NoError(t, err)
	assert.Equal(t, "mixedcase@example.com", usageEmail)
}
