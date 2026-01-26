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
	"os"
	"strings"
	"testing"
	"time"

	landing_page_react_vite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
)

// [REQ:STRIPE-CONFIG] Test Stripe environment configuration
func TestNewStripeService(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Set environment variables
	os.Setenv("STRIPE_PUBLISHABLE_KEY", "pk_test_123")
	os.Setenv("STRIPE_SECRET_KEY", "sk_test_123")
	os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_123")
	defer func() {
		os.Unsetenv("STRIPE_PUBLISHABLE_KEY")
		os.Unsetenv("STRIPE_SECRET_KEY")
		os.Unsetenv("STRIPE_WEBHOOK_SECRET")
	}()

	service := NewStripeService(db)
	if service == nil {
		t.Fatal("NewStripeService returned nil")
	}

	snapshot := service.ConfigSnapshot()
	if !snapshot.PublishableKeySet {
		t.Fatalf("expected publishable key to be detected")
	}
	if snapshot.PublishableKeyPreview == "" {
		t.Fatalf("expected publishable key preview to be populated")
	}
	if !snapshot.SecretKeySet {
		t.Fatalf("expected secret key to be detected")
	}
	if !snapshot.WebhookSecretSet {
		t.Fatalf("expected webhook secret to be detected")
	}
}

func TestStripeService_ConfigLoaderOverride(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Ensure ambient env does not leak into the custom loader.
	envKeys := []string{"STRIPE_PUBLISHABLE_KEY", "STRIPE_SECRET_KEY", "STRIPE_WEBHOOK_SECRET", "STRIPE_API_BASE"}
	originals := map[string]string{}
	for _, key := range envKeys {
		originals[key] = os.Getenv(key)
		os.Unsetenv(key)
	}
	t.Cleanup(func() {
		for _, key := range envKeys {
			if val := originals[key]; val != "" {
				os.Setenv(key, val)
			} else {
				os.Unsetenv(key)
			}
		}
	})

	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_loader", "production", 1000000, 0.001, "credits")
	insertBundlePrice(
		t,
		db,
		productID,
		"price_loader",
		"Loader Plan",
		"pro",
		"month",
		"usd",
		4900,
		false,
		"",
		0,
		0,
		"",
		0,
		0,
		1,
		10,
		"none",
		sessionTypeSubscription,
		map[string]interface{}{},
	)

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/checkout/sessions" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"cs_loader","url":"https://stripe.test/cs_loader","status":"open","customer_email":"loader@example.com","subscription":"sub_loader","amount_total":4900,"mode":"subscription","currency":"usd"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := requireTestStripeService(t, db)
	service.UseHTTPClient(stripeServer.Client())
	service.UseConfigLoader(func(ctx context.Context) (stripeRuntimeConfig, error) {
		return stripeRuntimeConfig{
			publishableKey: "pk_loader",
			secretKey:      "rk_loader",
			webhookSecret:  "whsec_loader",
			hasPublishable: true,
			hasSecret:      true,
			hasWebhook:     true,
			apiBase:        stripeServer.URL,
			source:         "test_loader",
		}, nil
	})
	if err := service.RefreshConfig(context.Background()); err != nil {
		t.Fatalf("refresh with custom loader failed: %v", err)
	}

	session, err := service.CreateCheckoutSession(
		"price_loader",
		"/ok",
		"/cancel",
		"loader@example.com",
	)
	if err != nil {
		t.Fatalf("CreateCheckoutSession failed with custom loader: %v", err)
	}
	if session.Url != "https://stripe.test/cs_loader" {
		t.Fatalf("expected session URL from mock stripe, got %s", session.Url)
	}
	if session.PublishableKey != "pk_loader" {
		t.Fatalf("expected publishable key from custom loader, got %s", session.PublishableKey)
	}
}

// [REQ:STRIPE-ROUTES] Test checkout session creation
func TestCreateCheckoutSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create tables
	_, err := db.Exec(`
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
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create checkout_sessions table: %v", err)
	}

	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_business_suite", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_123", "Test Plan", "pro", "month", "usd", 5000, true, "flat_amount", 100, 1, "test_intro_lookup", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	os.Setenv("STRIPE_PUBLISHABLE_KEY", "pk_test_valid")
	os.Setenv("STRIPE_SECRET_KEY", "sk_test_valid")
	os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test_valid")
	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/checkout/sessions" && r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"cs_test_created","url":"https://checkout.stripe.test/cs_test_created","status":"open","customer_email":"test@example.com","customer":"cus_123","subscription":"sub_123","amount_total":5000,"mode":"subscription","currency":"usd"}`)
			return
		}
		http.NotFound(w, r)
	}))
	os.Setenv("STRIPE_API_BASE", stripeServer.URL)
	defer func() {
		os.Unsetenv("STRIPE_PUBLISHABLE_KEY")
		os.Unsetenv("STRIPE_SECRET_KEY")
		os.Unsetenv("STRIPE_WEBHOOK_SECRET")
		os.Unsetenv("STRIPE_API_BASE")
		stripeServer.Close()
	}()

	service := requireTestStripeService(t, db)
	service.UseHTTPClient(stripeServer.Client())

	session, err := service.CreateCheckoutSession(
		"price_123",
		"/success",
		"/cancel",
		"test@example.com",
	)
	if err != nil {
		t.Fatalf("CreateCheckoutSession failed: %v", err)
	}

	if session.CustomerEmail != "test@example.com" {
		t.Errorf("Expected customer test@example.com, got %v", session.CustomerEmail)
	}

	if session.Status != landing_page_react_vite_v1.CheckoutSessionStatus_CHECKOUT_SESSION_STATUS_OPEN {
		t.Errorf("Expected status open, got %v", session.Status)
	}

	// Verify session was stored in database
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM checkout_sessions WHERE customer_email = $1", "test@example.com").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 checkout session in database, got %d", count)
	}
}

func TestCreateCheckoutSessionRequiresSecret(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	os.Unsetenv("STRIPE_SECRET_KEY")
	os.Setenv("STRIPE_PUBLISHABLE_KEY", "pk_test_missing_secret")
	defer func() {
		os.Unsetenv("STRIPE_PUBLISHABLE_KEY")
	}()

	service := NewStripeService(db)
	_, err := service.CreateCheckoutSession("price_missing_secret", "/ok", "/cancel", "no-secret@example.com")
	if err == nil {
		t.Fatalf("expected error when secret key is missing")
	}
}

// [REQ:STRIPE-SIG] Test webhook signature verification
func TestVerifyWebhookSignature(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	os.Setenv("STRIPE_PUBLISHABLE_KEY", "pk_test_valid")
	os.Setenv("STRIPE_SECRET_KEY", "sk_test_valid")
	os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test_secret")
	defer func() {
		os.Unsetenv("STRIPE_PUBLISHABLE_KEY")
		os.Unsetenv("STRIPE_SECRET_KEY")
		os.Unsetenv("STRIPE_WEBHOOK_SECRET")
	}()

	service := NewStripeService(db)

	payload := []byte(`{"type":"checkout.session.completed","data":{}}`)
	timestamp := time.Now().Unix()
	timestampStr := fmt.Sprintf("%d", timestamp)

	// Generate valid signature
	signedPayload := timestampStr + "." + string(payload)
	mac := hmac.New(sha256.New, []byte("whsec_test_secret"))
	mac.Write([]byte(signedPayload))
	signature := hex.EncodeToString(mac.Sum(nil))

	signatureHeader := "t=" + timestampStr + ",v1=" + signature

	// Test valid signature
	if !service.VerifyWebhookSignature(payload, signatureHeader) {
		t.Error("Valid signature was rejected")
	}

	// Test invalid signature
	invalidHeader := "t=" + timestampStr + ",v1=invalid_signature"
	if service.VerifyWebhookSignature(payload, invalidHeader) {
		t.Error("Invalid signature was accepted")
	}

	// Test missing signature
	if service.VerifyWebhookSignature(payload, "") {
		t.Error("Missing signature was accepted")
	}
}

// [REQ:STRIPE-ROUTES] Test webhook handling
func TestHandleWebhook_CheckoutCompleted(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create tables (drop first to ensure clean state)
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
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_business_suite", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_123", "Test Plan", "pro", "month", "usd", 5000, true, "flat_amount", 100, 1, "test_intro_lookup", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	// Insert initial checkout session
	_, err = db.Exec(`
		INSERT INTO checkout_sessions (session_id, customer_email, price_id, status)
		VALUES ($1, $2, $3, $4)
	`, "cs_test_123", "test@example.com", "price_123", "open")
	if err != nil {
		t.Fatalf("Failed to insert checkout session: %v", err)
	}

	os.Setenv("STRIPE_PUBLISHABLE_KEY", "pk_test_valid")
	os.Setenv("STRIPE_SECRET_KEY", "sk_test_valid")
	os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test_secret")
	defer func() {
		os.Unsetenv("STRIPE_PUBLISHABLE_KEY")
		os.Unsetenv("STRIPE_SECRET_KEY")
		os.Unsetenv("STRIPE_WEBHOOK_SECRET")
	}()

	service := requireTestStripeService(t, db)

	event := map[string]interface{}{
		"type": "checkout.session.completed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":             "cs_test_123",
				"customer_email": "test@example.com",
				"subscription":   "sub_123",
			},
		},
	}

	payload, _ := json.Marshal(event)
	timestamp := "1234567890"
	signedPayload := timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte("whsec_test_secret"))
	mac.Write([]byte(signedPayload))
	signature := hex.EncodeToString(mac.Sum(nil))
	signatureHeader := "t=" + timestamp + ",v1=" + signature

	err = service.HandleWebhook(payload, signatureHeader)
	if err != nil {
		t.Fatalf("HandleWebhook failed: %v", err)
	}

	// Verify checkout session was updated
	var status, subscriptionID string
	err = db.QueryRow(`
		SELECT status, subscription_id FROM checkout_sessions WHERE session_id = $1
	`, "cs_test_123").Scan(&status, &subscriptionID)
	if err != nil {
		t.Fatalf("Failed to query checkout session: %v", err)
	}

	if status != "complete" {
		t.Errorf("Expected status complete, got %s", status)
	}

	if subscriptionID != "sub_123" {
		t.Errorf("Expected subscription_id sub_123, got %s", subscriptionID)
	}

	// Verify subscription was created
	var subStatus string
	err = db.QueryRow(`
		SELECT status FROM subscriptions WHERE subscription_id = $1
	`, "sub_123").Scan(&subStatus)
	if err != nil {
		t.Fatalf("Failed to query subscription: %v", err)
	}

	if subStatus != "active" {
		t.Errorf("Expected subscription status active, got %s", subStatus)
	}
}

// [REQ:SUB-VERIFY] Test subscription verification
func TestVerifySubscription(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create subscriptions table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS subscriptions (
			id SERIAL PRIMARY KEY,
			subscription_id VARCHAR(255) UNIQUE NOT NULL,
			customer_id VARCHAR(255),
			customer_email VARCHAR(255),
			status VARCHAR(50) NOT NULL,
			canceled_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create subscriptions table: %v", err)
	}

	// Insert test subscription
	_, err = db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_email, status)
		VALUES ($1, $2, $3)
	`, "sub_test_123", "active@example.com", "active")
	if err != nil {
		t.Fatalf("Failed to insert subscription: %v", err)
	}

	os.Setenv("STRIPE_PUBLISHABLE_KEY", "pk_test_valid")
	os.Setenv("STRIPE_SECRET_KEY", "sk_test_valid")
	os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test_valid")
	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/subscriptions/sub_cancel_test") {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"sub_cancel_test","status":"canceled"}`)
			return
		}
		http.NotFound(w, r)
	}))
	os.Setenv("STRIPE_API_BASE", stripeServer.URL)
	defer func() {
		os.Unsetenv("STRIPE_PUBLISHABLE_KEY")
		os.Unsetenv("STRIPE_SECRET_KEY")
		os.Unsetenv("STRIPE_WEBHOOK_SECRET")
		os.Unsetenv("STRIPE_API_BASE")
		stripeServer.Close()
	}()

	service := NewStripeService(db)
	service.UseHTTPClient(stripeServer.Client())

	// Test active subscription
	result, err := service.VerifySubscription("active@example.com")
	if err != nil {
		t.Fatalf("VerifySubscription failed: %v", err)
	}

	if result.State != landing_page_react_vite_v1.SubscriptionState_SUBSCRIPTION_STATE_ACTIVE {
		t.Errorf("Expected status active, got %v", result.State)
	}

	// [REQ:SUB-CACHE] Verify cache metadata is present
	if result.CachedAt == nil {
		t.Error("Expected cached_at in result")
	}

	if result.CacheAgeMs < 0 {
		t.Error("Expected non-negative cache_age_ms in result")
	}

	// Test non-existent subscription
	result, err = service.VerifySubscription("nonexistent@example.com")
	if err != nil {
		t.Fatalf("VerifySubscription failed: %v", err)
	}

	if result.State != landing_page_react_vite_v1.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE {
		t.Errorf("Expected status inactive, got %v", result.State)
	}
}

// [REQ:SUB-CANCEL] Test subscription cancellation
func TestCancelSubscription(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create subscriptions table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS subscriptions (
			id SERIAL PRIMARY KEY,
			subscription_id VARCHAR(255) UNIQUE NOT NULL,
			customer_id VARCHAR(255),
			customer_email VARCHAR(255),
			status VARCHAR(50) NOT NULL,
			canceled_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create subscriptions table: %v", err)
	}

	// Insert active subscription
	_, err = db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_email, status)
		VALUES ($1, $2, $3)
	`, "sub_cancel_test", "cancel@example.com", "active")
	if err != nil {
		t.Fatalf("Failed to insert subscription: %v", err)
	}

	os.Setenv("STRIPE_PUBLISHABLE_KEY", "pk_test_valid")
	os.Setenv("STRIPE_SECRET_KEY", "sk_test_valid")
	os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test_valid")
	mockStripe := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/subscriptions/") {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"id":"%s","status":"canceled"}`, strings.TrimPrefix(r.URL.Path, "/v1/subscriptions/"))
			return
		}
		http.NotFound(w, r)
	}))
	os.Setenv("STRIPE_API_BASE", mockStripe.URL)
	defer func() {
		os.Unsetenv("STRIPE_PUBLISHABLE_KEY")
		os.Unsetenv("STRIPE_SECRET_KEY")
		os.Unsetenv("STRIPE_WEBHOOK_SECRET")
		os.Unsetenv("STRIPE_API_BASE")
		mockStripe.Close()
	}()

	service := NewStripeService(db)
	service.UseHTTPClient(mockStripe.Client())

	// Cancel subscription
	result, err := service.CancelSubscription("cancel@example.com")
	if err != nil {
		t.Fatalf("CancelSubscription failed: %v", err)
	}

	if result.GetSubscriptionId() != "sub_cancel_test" {
		t.Errorf("Expected subscription_id sub_cancel_test, got %v", result.GetSubscriptionId())
	}

	if result.State != landing_page_react_vite_v1.SubscriptionState_SUBSCRIPTION_STATE_CANCELED {
		t.Errorf("Expected status canceled, got %v", result.State)
	}

	// Verify database was updated
	var status string
	var canceledAt *time.Time
	err = db.QueryRow(`
		SELECT status, canceled_at FROM subscriptions WHERE subscription_id = $1
	`, "sub_cancel_test").Scan(&status, &canceledAt)
	if err != nil {
		t.Fatalf("Failed to query subscription: %v", err)
	}

	if status != "canceled" {
		t.Errorf("Expected database status canceled, got %s", status)
	}

	if canceledAt == nil {
		t.Error("Expected canceled_at to be set")
	}

	// Test canceling non-existent subscription
	_, err = service.CancelSubscription("nonexistent@example.com")
	if err == nil {
		t.Error("Expected error when canceling non-existent subscription")
	}
}

// [REQ:SUB-CACHE] Test subscription cache behavior
func TestVerifySubscription_CacheWarning(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create subscriptions table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS subscriptions (
			id SERIAL PRIMARY KEY,
			subscription_id VARCHAR(255) UNIQUE NOT NULL,
			customer_id VARCHAR(255),
			customer_email VARCHAR(255),
			status VARCHAR(50) NOT NULL,
			canceled_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create subscriptions table: %v", err)
	}

	// Insert stale subscription (updated_at > 60s ago)
	staleTime := time.Now().Add(-120 * time.Second)
	_, err = db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_email, status, updated_at)
		VALUES ($1, $2, $3, $4)
	`, "sub_stale", "stale@example.com", "active", staleTime)
	if err != nil {
		t.Fatalf("Failed to insert subscription: %v", err)
	}

	os.Setenv("STRIPE_PUBLISHABLE_KEY", "pk_test_valid")
	os.Setenv("STRIPE_SECRET_KEY", "sk_test_valid")
	os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test_valid")
	defer func() {
		os.Unsetenv("STRIPE_PUBLISHABLE_KEY")
		os.Unsetenv("STRIPE_SECRET_KEY")
		os.Unsetenv("STRIPE_WEBHOOK_SECRET")
	}()

	service := NewStripeService(db)

	result, err := service.VerifySubscription("stale@example.com")
	if err != nil {
		t.Fatalf("VerifySubscription failed: %v", err)
	}

	if result == nil || result.State == landing_page_react_vite_v1.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE {
		t.Errorf("Expected a subscription status, got %v", result)
	}
}

// ============================================================================
// extractBillingCycleDay Tests (GAP-005)
// ============================================================================

func TestExtractBillingCycleDay(t *testing.T) {
	tests := []struct {
		name      string
		timestamp int64
		expected  int
	}{
		{"zero", 0, 0},
		{"negative", -1, 0},
		{"day 15", time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC).Unix(), 15},
		{"day 1", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix(), 1},
		{"day 28", time.Date(2026, 1, 28, 23, 59, 0, 0, time.UTC).Unix(), 28},
		{"day 29 capped", time.Date(2026, 1, 29, 12, 0, 0, 0, time.UTC).Unix(), 28},
		{"day 31 capped", time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC).Unix(), 28},
		{"day 30 capped", time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC).Unix(), 28},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractBillingCycleDay(tc.timestamp); got != tc.expected {
				t.Errorf("expected %d, got %d", tc.expected, got)
			}
		})
	}
}

// ============================================================================
// chooseUserIdentity Tests (Pure Function)
// ============================================================================

func TestChooseUserIdentity_UserHintProvided(t *testing.T) {
	sub := &stripeSubscription{
		CustomerEmail: "sub@example.com",
		Customer:      "cus_123",
	}

	result := chooseUserIdentity("user@example.com", sub)
	if result != "user@example.com" {
		t.Errorf("expected user hint to be returned, got %s", result)
	}
}

func TestChooseUserIdentity_NilSubscription(t *testing.T) {
	result := chooseUserIdentity("", nil)
	if result != "" {
		t.Errorf("expected empty string for nil subscription, got %s", result)
	}
}

func TestChooseUserIdentity_SubscriptionWithEmail(t *testing.T) {
	sub := &stripeSubscription{
		CustomerEmail: "sub@example.com",
		Customer:      "cus_123",
	}

	result := chooseUserIdentity("", sub)
	if result != "sub@example.com" {
		t.Errorf("expected subscription email, got %s", result)
	}
}

func TestChooseUserIdentity_SubscriptionWithCustomerOnly(t *testing.T) {
	sub := &stripeSubscription{
		CustomerEmail: "",
		Customer:      "cus_123",
	}

	result := chooseUserIdentity("", sub)
	if result != "cus_123" {
		t.Errorf("expected customer ID, got %s", result)
	}
}

func TestChooseUserIdentity_EmptyUserHintAndEmail(t *testing.T) {
	sub := &stripeSubscription{
		CustomerEmail: "",
		Customer:      "",
	}

	result := chooseUserIdentity("", sub)
	if result != "" {
		t.Errorf("expected empty string, got %s", result)
	}
}

func TestChooseUserIdentity_WhitespaceHandling(t *testing.T) {
	sub := &stripeSubscription{
		CustomerEmail: "sub@example.com",
		Customer:      "cus_123",
	}

	// Whitespace-only user hint should fall back to subscription
	result := chooseUserIdentity("   ", sub)
	if result != "sub@example.com" {
		t.Errorf("expected subscription email for whitespace hint, got %s", result)
	}
}

// ============================================================================
// VerifyStripePrice Tests (HTTP Mock)
// ============================================================================

func TestVerifyStripePrice_ValidPriceID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/prices/price_valid") {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{
				"id": "price_valid",
				"lookup_key": "pro_monthly",
				"currency": "usd",
				"unit_amount": 4900,
				"active": true,
				"recurring": {"interval": "month"},
				"product": {"name": "Pro Plan"}
			}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := NewStripeService(db)
	service.UseHTTPClient(stripeServer.Client())
	service.UseConfigLoader(func(ctx context.Context) (stripeRuntimeConfig, error) {
		return stripeRuntimeConfig{
			publishableKey: "pk_test",
			secretKey:      "sk_test",
			hasPublishable: true,
			hasSecret:      true,
			apiBase:        stripeServer.URL,
		}, nil
	})
	if err := service.RefreshConfig(context.Background()); err != nil {
		t.Fatalf("Failed to refresh stripe config: %v", err)
	}

	result, err := service.VerifyStripePrice("price_valid")
	if err != nil {
		t.Fatalf("VerifyStripePrice failed: %v", err)
	}

	if result["id"] != "price_valid" {
		t.Errorf("expected id 'price_valid', got %v", result["id"])
	}
	if result["currency"] != "usd" {
		t.Errorf("expected currency 'usd', got %v", result["currency"])
	}
	if result["active"] != true {
		t.Errorf("expected active true, got %v", result["active"])
	}
}

func TestVerifyStripePrice_ValidLookupKey(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/v1/prices") {
			// First call: lookup by key
			if strings.Contains(r.URL.RawQuery, "lookup_keys") {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"data": [{"id": "price_from_lookup"}]}`)
				return
			}
			// Second call: get price by ID
			if strings.Contains(r.URL.Path, "price_from_lookup") {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{
					"id": "price_from_lookup",
					"lookup_key": "pro_monthly",
					"currency": "usd",
					"unit_amount": 4900,
					"active": true,
					"recurring": {"interval": "month"},
					"product": {"name": "Pro Plan"}
				}`)
				return
			}
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := NewStripeService(db)
	service.UseHTTPClient(stripeServer.Client())
	service.UseConfigLoader(func(ctx context.Context) (stripeRuntimeConfig, error) {
		return stripeRuntimeConfig{
			publishableKey: "pk_test",
			secretKey:      "sk_test",
			hasPublishable: true,
			hasSecret:      true,
			apiBase:        stripeServer.URL,
		}, nil
	})
	if err := service.RefreshConfig(context.Background()); err != nil {
		t.Fatalf("Failed to refresh stripe config: %v", err)
	}

	result, err := service.VerifyStripePrice("pro_monthly")
	if err != nil {
		t.Fatalf("VerifyStripePrice failed: %v", err)
	}

	if result["id"] != "price_from_lookup" {
		t.Errorf("expected id 'price_from_lookup', got %v", result["id"])
	}
}

func TestVerifyStripePrice_EmptyKey(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewStripeService(db)

	_, err := service.VerifyStripePrice("")
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestVerifyStripePrice_PriceNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.RawQuery, "lookup_keys") {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data": []}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error": {"message": "No such price"}}`)
	}))
	defer stripeServer.Close()

	service := NewStripeService(db)
	service.UseHTTPClient(stripeServer.Client())
	service.UseConfigLoader(func(ctx context.Context) (stripeRuntimeConfig, error) {
		return stripeRuntimeConfig{
			publishableKey: "pk_test",
			secretKey:      "sk_test",
			hasPublishable: true,
			hasSecret:      true,
			apiBase:        stripeServer.URL,
		}, nil
	})
	if err := service.RefreshConfig(context.Background()); err != nil {
		t.Fatalf("Failed to refresh stripe config: %v", err)
	}

	_, err := service.VerifyStripePrice("nonexistent_key")
	if err == nil {
		t.Error("expected error for nonexistent price")
	}
}

func TestVerifyStripePrice_NetworkError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	service := NewStripeService(db)
	service.UseConfigLoader(func(ctx context.Context) (stripeRuntimeConfig, error) {
		return stripeRuntimeConfig{
			publishableKey: "pk_test",
			secretKey:      "sk_test",
			hasPublishable: true,
			hasSecret:      true,
			apiBase:        "http://localhost:99999", // Invalid port
		}, nil
	})
	if err := service.RefreshConfig(context.Background()); err != nil {
		t.Fatalf("Failed to refresh stripe config: %v", err)
	}

	_, err := service.VerifyStripePrice("price_test")
	if err == nil {
		t.Error("expected network error")
	}
}

func TestVerifyStripePriceTyped_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/prices/price_typed") {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{
				"id": "price_typed",
				"lookup_key": "pro_monthly",
				"currency": "usd",
				"unit_amount": 4900,
				"active": true,
				"recurring": {"interval": "month"},
				"product": {"name": "Pro Plan"}
			}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := NewStripeService(db)
	service.UseHTTPClient(stripeServer.Client())
	service.UseConfigLoader(func(ctx context.Context) (stripeRuntimeConfig, error) {
		return stripeRuntimeConfig{
			publishableKey: "pk_test",
			secretKey:      "sk_test",
			hasPublishable: true,
			hasSecret:      true,
			apiBase:        stripeServer.URL,
		}, nil
	})
	if err := service.RefreshConfig(context.Background()); err != nil {
		t.Fatalf("Failed to refresh stripe config: %v", err)
	}

	info, err := service.VerifyStripePriceTyped("price_typed")
	if err != nil {
		t.Fatalf("VerifyStripePriceTyped failed: %v", err)
	}

	if info.ID != "price_typed" {
		t.Errorf("expected id 'price_typed', got %s", info.ID)
	}
	if info.Currency != "usd" {
		t.Errorf("expected currency 'usd', got %s", info.Currency)
	}
	if !info.Active {
		t.Error("expected active to be true")
	}
}

// ============================================================================
// Webhook Handler Tests
// ============================================================================

func TestHandleWebhook_SubscriptionCreated_NewSubscription(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test")
	defer os.Unsetenv("STRIPE_WEBHOOK_SECRET")

	service := NewStripeService(db)

	event := map[string]interface{}{
		"type": "customer.subscription.created",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":       "sub_new_123",
				"status":   "active",
				"customer": "cus_456",
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

	err := service.HandleWebhook(payload, signatureHeader)
	if err != nil {
		t.Fatalf("HandleWebhook failed: %v", err)
	}

	// Verify subscription was created
	var status string
	err = db.QueryRow("SELECT status FROM subscriptions WHERE subscription_id = $1", "sub_new_123").Scan(&status)
	if err != nil {
		t.Fatalf("subscription not created: %v", err)
	}
	if status != "active" {
		t.Errorf("expected status 'active', got %s", status)
	}
}

func TestHandleWebhook_SubscriptionCreated_MissingID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test")
	defer os.Unsetenv("STRIPE_WEBHOOK_SECRET")

	service := NewStripeService(db)

	event := map[string]interface{}{
		"type": "customer.subscription.created",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"status": "active",
				// Missing "id"
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

	err := service.HandleWebhook(payload, signatureHeader)
	if err == nil {
		t.Error("expected error for missing subscription ID")
	}
}

func TestHandleWebhook_InvoicePaid_RefreshesSubscription(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Pre-create a subscription
	_, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, status, created_at, updated_at)
		VALUES ('sub_invoice', 'cus_invoice', 'active', NOW(), NOW())
	`)
	if err != nil {
		t.Fatalf("failed to insert subscription: %v", err)
	}

	os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test")
	defer os.Unsetenv("STRIPE_WEBHOOK_SECRET")

	service := NewStripeService(db)

	event := map[string]interface{}{
		"type": "invoice.paid",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"subscription":   "sub_invoice",
				"customer":       "cus_invoice",
				"customer_email": "invoice@example.com",
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
	if err != nil {
		t.Fatalf("HandleWebhook failed: %v", err)
	}

	// Verify subscription status is still active
	var status string
	err = db.QueryRow("SELECT status FROM subscriptions WHERE subscription_id = $1", "sub_invoice").Scan(&status)
	if err != nil {
		t.Fatalf("failed to query subscription: %v", err)
	}
	if status != "active" {
		t.Errorf("expected status 'active', got %s", status)
	}
}

func TestHandleWebhook_SubscriptionUpdated_UpdatesStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Pre-create a subscription
	_, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, status, created_at, updated_at)
		VALUES ('sub_update', 'cus_update', 'active', NOW(), NOW())
	`)
	if err != nil {
		t.Fatalf("failed to insert subscription: %v", err)
	}

	os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test")
	defer os.Unsetenv("STRIPE_WEBHOOK_SECRET")

	service := NewStripeService(db)

	event := map[string]interface{}{
		"type": "customer.subscription.updated",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":       "sub_update",
				"status":   "past_due",
				"customer": "cus_update",
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
	if err != nil {
		t.Fatalf("HandleWebhook failed: %v", err)
	}

	var status string
	err = db.QueryRow("SELECT status FROM subscriptions WHERE subscription_id = $1", "sub_update").Scan(&status)
	if err != nil {
		t.Fatalf("failed to query subscription: %v", err)
	}
	if status != "past_due" {
		t.Errorf("expected status 'past_due', got %s", status)
	}
}

func TestHandleWebhook_SubscriptionDeleted_CancelsSubscription(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Pre-create a subscription
	_, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, status, created_at, updated_at)
		VALUES ('sub_delete', 'cus_delete', 'active', NOW(), NOW())
	`)
	if err != nil {
		t.Fatalf("failed to insert subscription: %v", err)
	}

	os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test")
	defer os.Unsetenv("STRIPE_WEBHOOK_SECRET")

	service := NewStripeService(db)

	event := map[string]interface{}{
		"type": "customer.subscription.deleted",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":       "sub_delete",
				"status":   "canceled",
				"customer": "cus_delete",
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
	if err != nil {
		t.Fatalf("HandleWebhook failed: %v", err)
	}

	var status string
	var canceledAt *time.Time
	err = db.QueryRow("SELECT status, canceled_at FROM subscriptions WHERE subscription_id = $1", "sub_delete").Scan(&status, &canceledAt)
	if err != nil {
		t.Fatalf("failed to query subscription: %v", err)
	}
	if status != "canceled" {
		t.Errorf("expected status 'canceled', got %s", status)
	}
	if canceledAt == nil {
		t.Error("expected canceled_at to be set")
	}
}

func TestHandleWebhook_InvoicePaymentFailed_UpdatesStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetStripeTestData(t, db)

	// Pre-create a subscription
	_, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, status, created_at, updated_at)
		VALUES ('sub_failed', 'cus_failed', 'active', NOW(), NOW())
	`)
	if err != nil {
		t.Fatalf("failed to insert subscription: %v", err)
	}

	os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test")
	defer os.Unsetenv("STRIPE_WEBHOOK_SECRET")

	service := NewStripeService(db)

	event := map[string]interface{}{
		"type": "invoice.payment_failed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"subscription":   "sub_failed",
				"customer":       "cus_failed",
				"customer_email": "failed@example.com",
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
	if err != nil {
		t.Fatalf("HandleWebhook failed: %v", err)
	}

	var status string
	err = db.QueryRow("SELECT status FROM subscriptions WHERE subscription_id = $1", "sub_failed").Scan(&status)
	if err != nil {
		t.Fatalf("failed to query subscription: %v", err)
	}
	if status != "past_due" {
		t.Errorf("expected status 'past_due', got %s", status)
	}
}

func TestHandleWebhook_UnknownEventType_Succeeds(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_test")
	defer os.Unsetenv("STRIPE_WEBHOOK_SECRET")

	service := NewStripeService(db)

	event := map[string]interface{}{
		"type": "unknown.event.type",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id": "unknown_123",
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

	err := service.HandleWebhook(payload, signatureHeader)
	if err != nil {
		t.Errorf("expected unknown event to succeed, got error: %v", err)
	}
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestParseStripeAmount_Float64(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewStripeService(db)

	result := service.parseStripeAmount(float64(4900))
	if result != 4900 {
		t.Errorf("expected 4900, got %d", result)
	}
}

func TestParseStripeAmount_Int64(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewStripeService(db)

	result := service.parseStripeAmount(int64(4900))
	if result != 4900 {
		t.Errorf("expected 4900, got %d", result)
	}
}

func TestParseStripeAmount_JSONNumber(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewStripeService(db)

	result := service.parseStripeAmount(json.Number("4900"))
	if result != 4900 {
		t.Errorf("expected 4900, got %d", result)
	}
}

func TestParseStripeAmount_String(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewStripeService(db)

	result := service.parseStripeAmount("4900")
	if result != 4900 {
		t.Errorf("expected 4900, got %d", result)
	}
}

func TestParseStripeAmount_InvalidType(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewStripeService(db)

	result := service.parseStripeAmount(struct{}{})
	if result != 0 {
		t.Errorf("expected 0 for invalid type, got %d", result)
	}
}

func TestBillingIntervalDuration_Year(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewStripeService(db)

	result := service.billingIntervalDuration(landing_page_react_vite_v1.BillingInterval_BILLING_INTERVAL_YEAR)
	expected := 365 * 24 * time.Hour
	if result != expected {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestBillingIntervalDuration_Month(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewStripeService(db)

	result := service.billingIntervalDuration(landing_page_react_vite_v1.BillingInterval_BILLING_INTERVAL_MONTH)
	expected := 30 * 24 * time.Hour
	if result != expected {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestBillingIntervalDuration_Default(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewStripeService(db)

	result := service.billingIntervalDuration(landing_page_react_vite_v1.BillingInterval_BILLING_INTERVAL_UNSPECIFIED)
	expected := 30 * 24 * time.Hour
	if result != expected {
		t.Errorf("expected default of %v, got %v", expected, result)
	}
}

func TestExtractAmount_FromAmountTotal(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewStripeService(db)

	obj := map[string]interface{}{
		"amount_total": float64(4900),
	}

	result := service.extractAmount(obj, nil)
	if result != 4900 {
		t.Errorf("expected 4900, got %d", result)
	}
}

func TestExtractAmount_FallbackToSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewStripeService(db)

	obj := map[string]interface{}{}
	session := &checkoutSessionRecord{
		AmountCents: struct {
			Int64 int64
			Valid bool
		}{Int64: 9900, Valid: true},
	}

	result := service.extractAmount(obj, session)
	if result != 9900 {
		t.Errorf("expected 9900, got %d", result)
	}
}

func TestMaskValue_ShortValue(t *testing.T) {
	result := maskValue("abc")
	if result != "abc" {
		t.Errorf("expected 'abc' for short value, got '%s'", result)
	}
}

func TestMaskValue_LongValue(t *testing.T) {
	result := maskValue("pk_test_1234567890")
	if !strings.HasPrefix(result, "pk_t") || !strings.HasSuffix(result, "90") {
		t.Errorf("expected masked value, got '%s'", result)
	}
}

func TestMaskValue_EmptyValue(t *testing.T) {
	result := maskValue("")
	if result != "" {
		t.Errorf("expected empty string, got '%s'", result)
	}
}
