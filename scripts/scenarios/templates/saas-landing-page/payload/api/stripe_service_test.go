package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
	"time"
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

	if service.publishableKey != "pk_test_123" {
		t.Errorf("Expected publishableKey pk_test_123, got %s", service.publishableKey)
	}

	if service.secretKey != "sk_test_123" {
		t.Errorf("Expected secretKey sk_test_123, got %s", service.secretKey)
	}

	if service.webhookSecret != "whsec_123" {
		t.Errorf("Expected webhookSecret whsec_123, got %s", service.webhookSecret)
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
			price_id VARCHAR(255),
			subscription_id VARCHAR(255),
			status VARCHAR(50) NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create checkout_sessions table: %v", err)
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

	session, err := service.CreateCheckoutSession(
		"price_123",
		"/success",
		"/cancel",
		"test@example.com",
	)

	if err != nil {
		t.Fatalf("CreateCheckoutSession failed: %v", err)
	}

	if session["customer"] != "test@example.com" {
		t.Errorf("Expected customer test@example.com, got %v", session["customer"])
	}

	if session["status"] != "open" {
		t.Errorf("Expected status open, got %v", session["status"])
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

	// Generate valid signature
	signedPayload := string(rune(timestamp)) + "." + string(payload)
	mac := hmac.New(sha256.New, []byte("whsec_test_secret"))
	mac.Write([]byte(signedPayload))
	signature := hex.EncodeToString(mac.Sum(nil))

	signatureHeader := "t=" + string(rune(timestamp)) + ",v1=" + signature

	// Test valid signature
	if !service.VerifyWebhookSignature(payload, signatureHeader) {
		t.Error("Valid signature was rejected")
	}

	// Test invalid signature
	invalidHeader := "t=" + string(rune(timestamp)) + ",v1=invalid_signature"
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
			price_id VARCHAR(255),
			subscription_id VARCHAR(255),
			status VARCHAR(50) NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);
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
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

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

	service := NewStripeService(db)

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
	defer func() {
		os.Unsetenv("STRIPE_PUBLISHABLE_KEY")
		os.Unsetenv("STRIPE_SECRET_KEY")
		os.Unsetenv("STRIPE_WEBHOOK_SECRET")
	}()

	service := NewStripeService(db)

	// Test active subscription
	result, err := service.VerifySubscription("active@example.com")
	if err != nil {
		t.Fatalf("VerifySubscription failed: %v", err)
	}

	if result["status"] != "active" {
		t.Errorf("Expected status active, got %v", result["status"])
	}

	// [REQ:SUB-CACHE] Verify cache metadata is present
	if _, ok := result["cached_at"]; !ok {
		t.Error("Expected cached_at in result")
	}

	if _, ok := result["cache_age_ms"]; !ok {
		t.Error("Expected cache_age_ms in result")
	}

	// Test non-existent subscription
	result, err = service.VerifySubscription("nonexistent@example.com")
	if err != nil {
		t.Fatalf("VerifySubscription failed: %v", err)
	}

	if result["status"] != "inactive" {
		t.Errorf("Expected status inactive, got %v", result["status"])
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
	defer func() {
		os.Unsetenv("STRIPE_PUBLISHABLE_KEY")
		os.Unsetenv("STRIPE_SECRET_KEY")
		os.Unsetenv("STRIPE_WEBHOOK_SECRET")
	}()

	service := NewStripeService(db)

	// Cancel subscription
	result, err := service.CancelSubscription("cancel@example.com")
	if err != nil {
		t.Fatalf("CancelSubscription failed: %v", err)
	}

	if result["subscription_id"] != "sub_cancel_test" {
		t.Errorf("Expected subscription_id sub_cancel_test, got %v", result["subscription_id"])
	}

	if result["status"] != "canceled" {
		t.Errorf("Expected status canceled, got %v", result["status"])
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

	cacheAgeMs := result["cache_age_ms"].(int64)
	if cacheAgeMs < 60000 {
		t.Errorf("Expected cache_age_ms > 60000, got %d", cacheAgeMs)
	}
}
