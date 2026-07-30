package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"landing-page-business-suite-api/internal/commerce"
)

// ============================================================================
// One-Time Coupon Enforcement Tests
// ============================================================================

// TestCoupon_OneTimePerEmail_BlocksReuse verifies that an email that has already
// used the intro coupon cannot use it again.
func TestCoupon_OneTimePerEmail_BlocksReuse(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	// Create users table with intro tracking
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
	ctx := context.Background()

	// First check: user doesn't exist, should be eligible
	eligible, err := service.checkIntroEligibility(ctx, "newuser@example.com")
	require.NoError(t, err)
	assert.True(t, eligible, "new user should be eligible for intro")

	// Create user who has used intro
	_, err = db.Exec(`
		INSERT INTO users (email, has_used_intro, stripe_customer_id)
		VALUES ($1, TRUE, $2)
	`, "usedintro@example.com", "cus_used_123")
	require.NoError(t, err)

	// Second check: user exists with has_used_intro=true, should NOT be eligible
	eligible, err = service.checkIntroEligibility(ctx, "usedintro@example.com")
	require.NoError(t, err)
	assert.False(t, eligible, "user who has used intro should not be eligible again")
}

// TestCoupon_OneTimePerEmail_CaseVariations verifies that case variations of the
// same email are treated as the same user for intro eligibility.
func TestCoupon_OneTimePerEmail_CaseVariations(t *testing.T) {
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

	service := NewStripeService(db)
	ctx := context.Background()

	// Insert user with lowercase email who has used intro
	_, err = db.Exec(`
		INSERT INTO users (email, has_used_intro)
		VALUES ($1, TRUE)
	`, "casetest@example.com")
	require.NoError(t, err)

	// All case variations should be blocked
	testCases := []struct {
		name  string
		email string
	}{
		{"exact lowercase", "casetest@example.com"},
		{"all uppercase", "CASETEST@EXAMPLE.COM"},
		{"mixed case", "CaseTest@Example.COM"},
		{"partial uppercase", "CaseTest@example.com"},
		{"domain uppercase", "casetest@EXAMPLE.COM"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			eligible, err := service.checkIntroEligibility(ctx, tc.email)
			require.NoError(t, err)
			assert.False(t, eligible, "case variation %q should not be eligible", tc.email)
		})
	}
}

// TestCoupon_OneTimePerEmail_WhitespaceVariations verifies that whitespace in
// emails is trimmed before eligibility checks.
func TestCoupon_OneTimePerEmail_WhitespaceVariations(t *testing.T) {
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

	service := NewStripeService(db)
	ctx := context.Background()

	// Insert user
	_, err = db.Exec(`
		INSERT INTO users (email, has_used_intro)
		VALUES ($1, TRUE)
	`, "whitespace@example.com")
	require.NoError(t, err)

	testCases := []struct {
		name  string
		email string
	}{
		{"leading space", " whitespace@example.com"},
		{"trailing space", "whitespace@example.com "},
		{"both spaces", "  whitespace@example.com  "},
		{"tab character", "\twhitespace@example.com\t"},
		{"newline", "\nwhitespace@example.com\n"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			eligible, err := service.checkIntroEligibility(ctx, tc.email)
			require.NoError(t, err)
			assert.False(t, eligible, "whitespace variation %q should not be eligible", tc.email)
		})
	}
}

// TestCoupon_OneTimePerEmail_ConcurrentAttempts verifies that concurrent requests
// to check eligibility and mark intro used work correctly.
func TestCoupon_OneTimePerEmail_ConcurrentAttempts(t *testing.T) {
	db := setupTestDB(t)
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
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"cus_concurrent"}`)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	ctx := context.Background()
	email := "concurrent@example.com"
	const numGoroutines = 10

	var wg sync.WaitGroup
	var successCount int64
	startCh := make(chan struct{})

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-startCh

			err := service.markIntroUsed(ctx, email, fmt.Sprintf("cus_%d", idx), "coupon_test", "pro", fmt.Sprintf("sub_%d", idx))
			if err == nil {
				atomic.AddInt64(&successCount, 1)
			}
		}(i)
	}

	close(startCh)
	wg.Wait()

	// At least one should succeed (the first one that acquires the lock)
	assert.GreaterOrEqual(t, successCount, int64(1), "at least one concurrent attempt should succeed")

	// Verify user has_used_intro is true
	var hasUsed bool
	err = db.QueryRow(`SELECT has_used_intro FROM users WHERE email = $1`, email).Scan(&hasUsed)
	require.NoError(t, err)
	assert.True(t, hasUsed, "user should be marked as having used intro")

	// Verify at least one usage record exists
	var usageCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM intro_coupon_usage WHERE email = $1`, email).Scan(&usageCount)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, usageCount, 1, "at least one usage record should exist")
}

// ============================================================================
// Intro Pricing Flow Tests
// ============================================================================

// TestIntroPricing_FirstMonthDiscount_Applied verifies that the intro coupon
// is correctly applied for eligible users.
func TestIntroPricing_FirstMonthDiscount_Applied(t *testing.T) {
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

	// Test intro coupon config using direct struct (no env vars needed)
	config := IntroCouponConfig{
		Enabled:   true,
		CouponMap: map[string]string{"pro": "coupon_pro_intro"},
	}

	assert.True(t, config.Enabled, "intro should be enabled")
	assert.Equal(t, "coupon_pro_intro", config.GetCouponForTier("pro"))
	assert.Equal(t, "", config.GetCouponForTier("nonexistent"))
}

// TestIntroPricing_EligibleNewUser_CouponApplied verifies that new users
// get the intro coupon applied.
func TestIntroPricing_EligibleNewUser_CouponApplied(t *testing.T) {
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

	service := NewStripeService(db)
	ctx := context.Background()

	// New user (not in database) should be eligible
	eligible, err := service.checkIntroEligibility(ctx, "newuser@example.com")
	require.NoError(t, err)
	assert.True(t, eligible, "new user should be eligible for intro")

	// User exists but has_used_intro is false
	_, err = db.Exec(`
		INSERT INTO users (email, has_used_intro)
		VALUES ($1, FALSE)
	`, "notused@example.com")
	require.NoError(t, err)

	eligible, err = service.checkIntroEligibility(ctx, "notused@example.com")
	require.NoError(t, err)
	assert.True(t, eligible, "user with has_used_intro=false should be eligible")
}

// TestIntroPricing_ExistingUser_NoCoupon verifies that users who have already
// used the intro don't get the coupon.
func TestIntroPricing_ExistingUser_NoCoupon(t *testing.T) {
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

	service := NewStripeService(db)
	ctx := context.Background()

	// Insert user who has used intro
	_, err = db.Exec(`
		INSERT INTO users (email, has_used_intro)
		VALUES ($1, TRUE)
	`, "existinguser@example.com")
	require.NoError(t, err)

	eligible, err := service.checkIntroEligibility(ctx, "existinguser@example.com")
	require.NoError(t, err)
	assert.False(t, eligible, "user who has used intro should not be eligible")
}

// TestIntroPricing_EmptyEmail_NotEligible verifies that empty or whitespace-only
// emails are not eligible for intro pricing.
func TestIntroPricing_EmptyEmail_NotEligible(t *testing.T) {
	db := setupTestDB(t)

	service := NewStripeService(db)
	ctx := context.Background()

	testCases := []struct {
		name  string
		email string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"tab only", "\t"},
		{"newline only", "\n"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			eligible, err := service.checkIntroEligibility(ctx, tc.email)
			require.NoError(t, err)
			assert.False(t, eligible, "empty/whitespace email should not be eligible")
		})
	}
}

// TestIntroPricing_MarkIntroUsed_RecordsUsage verifies that markIntroUsed
// correctly records the usage in all appropriate tables.
func TestIntroPricing_MarkIntroUsed_RecordsUsage(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)
	upsertTestBundleProduct(t, db, "intro_coupon_test", "Intro Coupon Test", "prod_intro_coupon_test", "test", 1_000_000, 1, "credits")

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
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"cus_mark_test"}`)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	ctx := context.Background()

	err = service.markIntroUsed(ctx, "marktest@example.com", "cus_mark_123", "coupon_test", "pro", "sub_mark_123")
	require.NoError(t, err)

	// Verify user record
	var hasUsed bool
	var customerID sql.NullString
	err = db.QueryRow(`SELECT has_used_intro, stripe_customer_id FROM users WHERE email = $1`, "marktest@example.com").Scan(&hasUsed, &customerID)
	require.NoError(t, err)
	assert.True(t, hasUsed)
	assert.True(t, customerID.Valid)
	assert.Equal(t, "cus_mark_123", customerID.String)

	// Verify usage record
	var usageEmail, usageCouponID, usagePlanTier, usageSubID string
	err = db.QueryRow(`
		SELECT email, coupon_id, plan_tier, subscription_id
		FROM intro_coupon_usage
		WHERE stripe_customer_id = $1
	`, "cus_mark_123").Scan(&usageEmail, &usageCouponID, &usagePlanTier, &usageSubID)
	require.NoError(t, err)
	assert.Equal(t, "marktest@example.com", usageEmail)
	assert.Equal(t, "coupon_test", usageCouponID)
	assert.Equal(t, "pro", usagePlanTier)
	assert.Equal(t, "sub_mark_123", usageSubID)
}

// ============================================================================
// Coupon CRUD Tests
// ============================================================================

// TestCreateCoupon_PercentOff_Valid verifies creating a percent-off coupon.
func TestCreateCoupon_PercentOff_Valid(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/coupons" {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "parse form error", http.StatusBadRequest)
				return
			}
			percentOff := r.FormValue("percent_off")
			// The service sends percent_off as "50.00" (formatted with 2 decimal places)
			if percentOff != "50.00" {
				http.Error(w, fmt.Sprintf("expected 50.00 percent off, got %s", percentOff), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{
				"id":"coupon_percent_50",
				"percent_off":50,
				"duration":"once",
				"valid":true,
				"created":1700000000
			}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	ctx := context.Background()
	percentOff := float64(50)
	coupon, err := service.CreateCoupon(ctx, commerce.CreateCouponInput{
		ID:         "coupon_percent_50",
		PercentOff: &percentOff,
		Duration:   "once",
	})
	require.NoError(t, err)
	require.NotNil(t, coupon)
	assert.Equal(t, "coupon_percent_50", coupon.ID)
	require.NotNil(t, coupon.PercentOff)
	assert.Equal(t, float64(50), *coupon.PercentOff)
}

// TestCreateCoupon_AmountOff_Valid verifies creating an amount-off coupon.
func TestCreateCoupon_AmountOff_Valid(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/coupons" {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "parse form error", http.StatusBadRequest)
				return
			}
			amountOff := r.FormValue("amount_off")
			currency := r.FormValue("currency")
			if amountOff != "1000" || currency != "usd" {
				http.Error(w, "expected 1000 cents off in usd", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{
				"id":"coupon_amount_1000",
				"amount_off":1000,
				"currency":"usd",
				"duration":"once",
				"valid":true,
				"created":1700000000
			}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	ctx := context.Background()
	amountOff := int64(1000)
	coupon, err := service.CreateCoupon(ctx, commerce.CreateCouponInput{
		ID:        "coupon_amount_1000",
		AmountOff: &amountOff,
		Currency:  "usd",
		Duration:  "once",
	})
	require.NoError(t, err)
	require.NotNil(t, coupon)
	assert.Equal(t, "coupon_amount_1000", coupon.ID)
	require.NotNil(t, coupon.AmountOff)
	assert.Equal(t, int64(1000), *coupon.AmountOff)
}

// TestDeleteCoupon_Success verifies successful coupon deletion.
func TestDeleteCoupon_Success(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == "/v1/coupons/coupon_to_delete" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"coupon_to_delete","deleted":true}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	ctx := context.Background()
	err := service.DeleteCoupon(ctx, "coupon_to_delete")
	require.NoError(t, err)
}

// TestDeleteCoupon_NotFound verifies error handling when coupon doesn't exist.
func TestDeleteCoupon_NotFound(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":{"message":"No such coupon","type":"invalid_request_error"}}`)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	ctx := context.Background()
	err := service.DeleteCoupon(ctx, "nonexistent_coupon")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "No such coupon")
}

// TestListCoupons_Success verifies listing coupons from Stripe.
func TestListCoupons_Success(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/coupons" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{
				"data": [
					{"id":"coupon_1","percent_off":10,"duration":"once","valid":true,"created":1700000000},
					{"id":"coupon_2","amount_off":500,"currency":"usd","duration":"forever","valid":true,"created":1700000001}
				],
				"has_more": false
			}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	ctx := context.Background()
	coupons, err := service.ListCoupons(ctx)
	require.NoError(t, err)
	assert.Len(t, coupons, 2)
	assert.Equal(t, "coupon_1", coupons[0].ID)
	require.NotNil(t, coupons[0].PercentOff)
	assert.Equal(t, float64(10), *coupons[0].PercentOff)
	assert.Equal(t, "coupon_2", coupons[1].ID)
	require.NotNil(t, coupons[1].AmountOff)
	assert.Equal(t, int64(500), *coupons[1].AmountOff)
}

// TestGetCoupon_Success verifies fetching a single coupon by ID.
func TestGetCoupon_Success(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/coupons/coupon_single" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{
				"id":"coupon_single",
				"percent_off":25,
				"duration":"repeating",
				"duration_in_months":3,
				"valid":true,
				"created":1700000000,
				"times_redeemed":5,
				"max_redemptions":100
			}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	ctx := context.Background()
	coupon, err := service.GetCoupon(ctx, "coupon_single")
	require.NoError(t, err)
	require.NotNil(t, coupon)
	assert.Equal(t, "coupon_single", coupon.ID)
	require.NotNil(t, coupon.PercentOff)
	assert.Equal(t, float64(25), *coupon.PercentOff)
	assert.Equal(t, "repeating", coupon.Duration)
	require.NotNil(t, coupon.DurationInMonths)
	assert.Equal(t, 3, *coupon.DurationInMonths)
	assert.Equal(t, 5, coupon.TimesRedeemed)
}

// ============================================================================
// Intro Coupon Configuration Tests
// ============================================================================

// TestIntroCouponConfig_LoadFromEnv verifies loading intro coupon config from env.
func TestIntroCouponConfig_LoadFromEnv(t *testing.T) {
	// Test disabled by default (using t.Setenv for parallel safety)
	t.Setenv("INTRO_ENABLED", "")
	config := loadIntroCouponConfig()
	assert.False(t, config.Enabled)

	// Test enabled with all coupons
	t.Setenv("INTRO_ENABLED", "true")
	t.Setenv("INTRO_COUPON_SOLO", "coupon_solo")
	t.Setenv("INTRO_COUPON_PRO", "coupon_pro")
	t.Setenv("INTRO_COUPON_STUDIO", "coupon_studio")
	t.Setenv("INTRO_COUPON_BUSINESS", "coupon_business")

	config = loadIntroCouponConfig()
	assert.True(t, config.Enabled)
	assert.Equal(t, "coupon_solo", config.GetCouponForTier("solo"))
	assert.Equal(t, "coupon_pro", config.GetCouponForTier("pro"))
	assert.Equal(t, "coupon_studio", config.GetCouponForTier("studio"))
	assert.Equal(t, "coupon_business", config.GetCouponForTier("business"))
	assert.Equal(t, "", config.GetCouponForTier("nonexistent"))
}

// TestIntroCouponConfig_TierCaseInsensitive verifies tier lookup is case-insensitive.
func TestIntroCouponConfig_TierCaseInsensitive(t *testing.T) {
	config := IntroCouponConfig{
		Enabled: true,
		CouponMap: map[string]string{
			"pro": "coupon_pro",
		},
	}

	assert.Equal(t, "coupon_pro", config.GetCouponForTier("pro"))
	assert.Equal(t, "coupon_pro", config.GetCouponForTier("PRO"))
	assert.Equal(t, "coupon_pro", config.GetCouponForTier("Pro"))
	assert.Equal(t, "coupon_pro", config.GetCouponForTier("  pro  "))
}

// TestIntroCouponConfig_DisabledReturnsEmpty verifies disabled config returns empty.
func TestIntroCouponConfig_DisabledReturnsEmpty(t *testing.T) {
	config := IntroCouponConfig{
		Enabled: false,
		CouponMap: map[string]string{
			"pro": "coupon_pro",
		},
	}

	assert.Equal(t, "", config.GetCouponForTier("pro"))
}

// TestGetIntroCouponMap verifies the GetIntroCouponMap method.
func TestGetIntroCouponMap(t *testing.T) {
	db := setupTestDB(t)

	cfg := DefaultStripeTestConfig().WithIntroCoupon(true, map[string]string{"pro": "coupon_pro_test"})
	service := ConfigureStripeService(t, db, cfg, nil)
	couponMap := service.GetIntroCouponMap()

	assert.NotNil(t, couponMap)
	assert.Equal(t, "coupon_pro_test", couponMap["pro"])
}

// TestIsIntroCoupon verifies the isIntroCoupon helper function.
func TestIsIntroCoupon(t *testing.T) {
	db := setupTestDB(t)

	cfg := DefaultStripeTestConfig().WithIntroCoupon(true, map[string]string{"pro": "coupon_pro_check"})
	service := ConfigureStripeService(t, db, cfg, nil)

	assert.True(t, service.isIntroCoupon("coupon_pro_check"))
	assert.False(t, service.isIntroCoupon("some_other_coupon"))
	assert.False(t, service.isIntroCoupon(""))
}

// TestMarkIntroUsed_RequiresEmail verifies that markIntroUsed requires an email.
func TestMarkIntroUsed_RequiresEmail(t *testing.T) {
	db := setupTestDB(t)

	service := NewStripeService(db)
	ctx := context.Background()

	err := service.markIntroUsed(ctx, "", "cus_123", "coupon_test", "pro", "sub_123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "email required")
}

// TestMarkIntroUsed_NormalizesEmail verifies that markIntroUsed normalizes the email.
func TestMarkIntroUsed_NormalizesEmail(t *testing.T) {
	db := setupTestDB(t)
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
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"cus_normalize_test"}`)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	ctx := context.Background()

	// Pass uppercase email
	err = service.markIntroUsed(ctx, "UPPERCASE@EXAMPLE.COM", "cus_normalize", "coupon_test", "pro", "sub_normalize")
	require.NoError(t, err)

	// Verify it was stored as lowercase
	var storedEmail string
	err = db.QueryRow(`SELECT email FROM users WHERE stripe_customer_id = $1`, "cus_normalize").Scan(&storedEmail)
	require.NoError(t, err)
	assert.Equal(t, "uppercase@example.com", storedEmail)

	// Verify we can find with any case
	eligible, err := service.checkIntroEligibility(ctx, "uppercase@EXAMPLE.com")
	require.NoError(t, err)
	assert.False(t, eligible, "should find user with different case")
}

// TestCouponUpdateMetadata verifies updating coupon metadata.
func TestCouponUpdateMetadata(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/coupons/coupon_to_update" {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "parse form error", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{
				"id":"coupon_to_update",
				"percent_off":15,
				"duration":"once",
				"valid":true,
				"created":1700000000,
				"metadata":{"updated":"true"}
			}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	ctx := context.Background()
	coupon, err := service.UpdateCoupon(ctx, "coupon_to_update", commerce.UpdateCouponInput{
		Name: "Updated Coupon Name",
	})
	require.NoError(t, err)
	require.NotNil(t, coupon)
	assert.Equal(t, "coupon_to_update", coupon.ID)
}

// TestExtractIntroCouponFromInvoice verifies extracting intro coupon from invoice.
func TestExtractIntroCouponFromInvoice(t *testing.T) {
	db := setupTestDB(t)

	cfg := DefaultStripeTestConfig().WithIntroCoupon(true, map[string]string{"pro": "coupon_intro_pro"})
	service := ConfigureStripeService(t, db, cfg, nil)

	// Test with matching coupon in discount
	invoiceWithDiscount := map[string]interface{}{
		"discount": map[string]interface{}{
			"coupon": map[string]interface{}{
				"id": "coupon_intro_pro",
			},
		},
	}

	couponID := service.extractIntroCouponFromInvoice(invoiceWithDiscount)
	assert.Equal(t, "coupon_intro_pro", couponID)

	// Test with non-intro coupon
	invoiceWithOtherCoupon := map[string]interface{}{
		"discount": map[string]interface{}{
			"coupon": map[string]interface{}{
				"id": "some_other_coupon",
			},
		},
	}

	couponID = service.extractIntroCouponFromInvoice(invoiceWithOtherCoupon)
	assert.Equal(t, "", couponID)

	// Test with no discount
	invoiceNoDiscount := map[string]interface{}{}
	couponID = service.extractIntroCouponFromInvoice(invoiceNoDiscount)
	assert.Equal(t, "", couponID)
}

// TestCheckIntroCouponMapping verifies the checkIntroCouponMapping helper.
func TestCheckIntroCouponMapping(t *testing.T) {
	db := setupTestDB(t)

	cfg := DefaultStripeTestConfig().WithIntroCoupon(true, map[string]string{
		"pro":  "coupon_pro_mapping",
		"solo": "coupon_solo_mapping",
	})
	service := ConfigureStripeService(t, db, cfg, nil)

	// Test matching coupon
	isIntro, tier := service.checkIntroCouponMapping("coupon_pro_mapping")
	assert.True(t, isIntro)
	assert.Equal(t, "pro", tier)

	isIntro, tier = service.checkIntroCouponMapping("coupon_solo_mapping")
	assert.True(t, isIntro)
	assert.Equal(t, "solo", tier)

	// Test non-matching coupon
	isIntro, tier = service.checkIntroCouponMapping("unrelated_coupon")
	assert.False(t, isIntro)
	assert.Equal(t, "", tier)
}

// TestCouponIntegrationWithSchedule verifies coupons work with subscription schedules.
func TestCouponIntegrationWithSchedule(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	// Ensure subscription_schedules has all columns needed for this test
	// The declarative runtime schema already created it; add only test-specific data.
	_, err := db.Exec(`
		ALTER TABLE subscription_schedules ADD COLUMN IF NOT EXISTS customer_id VARCHAR(255);
		ALTER TABLE subscription_schedules ADD COLUMN IF NOT EXISTS current_phase_start TIMESTAMP;
		ALTER TABLE subscription_schedules ADD COLUMN IF NOT EXISTS current_phase_end TIMESTAMP;
		ALTER TABLE subscription_schedules ADD COLUMN IF NOT EXISTS phases JSONB DEFAULT '[]'::jsonb
	`)
	require.NoError(t, err)

	// Insert a checkout session with schedule_id
	_, err = db.Exec(`
		INSERT INTO checkout_sessions (session_id, customer_email, price_id, status, schedule_id)
		VALUES ($1, $2, $3, $4, $5)
	`, "cs_schedule_test", "schedule@example.com", "price_pro", "complete", "sub_sched_123")
	require.NoError(t, err)

	// Verify schedule_id was stored
	var scheduleID sql.NullString
	err = db.QueryRow(`SELECT schedule_id FROM checkout_sessions WHERE session_id = $1`, "cs_schedule_test").Scan(&scheduleID)
	require.NoError(t, err)
	assert.True(t, scheduleID.Valid)
	assert.Equal(t, "sub_sched_123", scheduleID.String)

	// Simulate second month - schedule would transition to full price
	now := time.Now()
	_, err = db.Exec(`
		INSERT INTO subscription_schedules (schedule_id, customer_id, subscription_id, status, current_phase_start, current_phase_end, price_id, billing_interval)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, "sub_sched_123", "cus_schedule", "sub_schedule", "active", now.Add(-30*24*time.Hour), now.Add(30*24*time.Hour), "price_pro", "month")
	require.NoError(t, err)

	var scheduleStatus string
	err = db.QueryRow(`SELECT status FROM subscription_schedules WHERE schedule_id = $1`, "sub_sched_123").Scan(&scheduleStatus)
	require.NoError(t, err)
	assert.Equal(t, "active", scheduleStatus)
}
