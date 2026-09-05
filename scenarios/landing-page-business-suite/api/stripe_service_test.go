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
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	landing_page_business_suite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"landing-page-business-suite-api/internal/commerce"
)

// [REQ:STRIPE-CONFIG] Test Stripe credential-authority configuration
func TestNewStripeService(t *testing.T) {
	db := setupTestDB(t)

	payment := NewPaymentSettingsService(db)
	if _, err := payment.SaveStripeSettings(context.Background(), commerce.StripeSettingsInput{
		PublishableKey: ptrStripe("pk_test_123"),
		SecretKey:      ptrStripe("sk_test_123"),
		WebhookSecret:  ptrStripe("whsec_123"),
	}); err != nil {
		t.Fatalf("failed to seed authority-backed test credentials: %v", err)
	}

	service := NewStripeServiceWithSettings(db, NewPlanService(db), payment)
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
	resetStripeTestData(t, db)

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

	// Use ConfigureStripeService with custom keys
	cfg := DefaultStripeTestConfig().WithKeys("pk_loader", "rk_loader", "whsec_loader")
	service := ConfigureStripeService(t, db, cfg, stripeServer)

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

	// setupTestDB applies the embedded checkout_sessions schema.
	var err error

	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_business_suite", "production", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_123", "Test Plan", "pro", "month", "usd", 5000, true, "flat_amount", 100, 1, "test_intro_lookup", 1000000, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/checkout/sessions" && r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"cs_test_created","url":"https://checkout.stripe.test/cs_test_created","status":"open","customer_email":"test@example.com","customer":"cus_123","subscription":"sub_123","amount_total":5000,"mode":"subscription","currency":"usd"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

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

	if session.Status != landing_page_business_suite_v1.CheckoutSessionStatus_CHECKOUT_SESSION_STATUS_OPEN {
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

	// Configure service with no secret key to test error handling
	service := requireTestStripeService(t, db)
	service.UseConfigLoader(func(ctx context.Context) (stripeRuntimeConfig, error) {
		return stripeRuntimeConfig{
			publishableKey: "pk_test_missing_secret",
			secretKey:      "", // Missing secret key
			webhookSecret:  "",
			hasPublishable: true,
			hasSecret:      false,
			hasWebhook:     false,
			source:         "test",
		}, nil
	})
	if err := service.RefreshConfig(context.Background()); err != nil {
		t.Fatalf("refresh failed: %v", err)
	}

	_, err := service.CreateCheckoutSession("price_missing_secret", "/ok", "/cancel", "no-secret@example.com")
	if err == nil {
		t.Fatalf("expected error when secret key is missing")
	}
}

// [REQ:STRIPE-SIG] Test webhook signature verification
func TestVerifyWebhookSignature(t *testing.T) {
	db := setupTestDB(t)

	cfg := DefaultStripeTestConfig().WithKeys("pk_test_valid", "sk_test_valid", "whsec_test_secret")
	service := ConfigureStripeService(t, db, cfg, nil)

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

	// setupTestDB applies the embedded checkout_sessions and subscriptions schemas.
	var err error

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

	cfg := DefaultStripeTestConfig().WithKeys("pk_test_valid", "sk_test_valid", "whsec_test_secret")
	service := ConfigureStripeService(t, db, cfg, nil)

	event := map[string]interface{}{
		"id":   "evt_checkout_123",
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
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
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

	// setupTestDB applies the embedded subscriptions schema.
	var err error

	// Insert test subscription
	_, err = db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_email, status)
		VALUES ($1, $2, $3)
	`, "sub_test_123", "active@example.com", "active")
	if err != nil {
		t.Fatalf("Failed to insert subscription: %v", err)
	}

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/subscriptions/sub_cancel_test") {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"sub_cancel_test","status":"canceled"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer stripeServer.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

	// Test active subscription
	result, err := service.VerifySubscription("active@example.com")
	if err != nil {
		t.Fatalf("VerifySubscription failed: %v", err)
	}

	if result.State != shared.SubscriptionState_SUBSCRIPTION_STATE_ACTIVE {
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

	if result.State != shared.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE {
		t.Errorf("Expected status inactive, got %v", result.State)
	}
}

// [REQ:SUB-CANCEL] Test subscription cancellation
func TestCancelSubscription(t *testing.T) {
	db := setupTestDB(t)

	// setupTestDB applies the embedded subscriptions schema.
	var err error

	// Insert active subscription
	_, err = db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_email, status)
		VALUES ($1, $2, $3)
	`, "sub_cancel_test", "cancel@example.com", "active")
	if err != nil {
		t.Fatalf("Failed to insert subscription: %v", err)
	}

	mockStripe := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/subscriptions/") {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"id":"%s","status":"canceled"}`, strings.TrimPrefix(r.URL.Path, "/v1/subscriptions/"))
			return
		}
		http.NotFound(w, r)
	}))
	defer mockStripe.Close()

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), mockStripe)

	// Cancel subscription
	result, err := service.CancelSubscription("cancel@example.com")
	if err != nil {
		t.Fatalf("CancelSubscription failed: %v", err)
	}

	if result.GetSubscriptionId() != "sub_cancel_test" {
		t.Errorf("Expected subscription_id sub_cancel_test, got %v", result.GetSubscriptionId())
	}

	if result.State != shared.SubscriptionState_SUBSCRIPTION_STATE_CANCELED {
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

	// setupTestDB applies the embedded subscriptions schema.
	var err error

	// Insert stale subscription (updated_at > 60s ago)
	staleTime := time.Now().Add(-120 * time.Second)
	_, err = db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_email, status, updated_at)
		VALUES ($1, $2, $3, $4)
	`, "sub_stale", "stale@example.com", "active", staleTime)
	if err != nil {
		t.Fatalf("Failed to insert subscription: %v", err)
	}

	service := ConfigureStripeService(t, db, DefaultStripeTestConfig(), nil)

	result, err := service.VerifySubscription("stale@example.com")
	if err != nil {
		t.Fatalf("VerifySubscription failed: %v", err)
	}

	if result == nil || result.State == shared.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE {
		t.Errorf("Expected a subscription status, got %v", result)
	}
}

func TestPersistSubscriptionFromStripe_PreservesPlanTier(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_test", "production", 1_000_000, 0.001, "credits")
	service := requireTestStripeService(t, db)

	_, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_email, status, plan_tier, bundle_key, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,NOW(),NOW())
	`, "sub_keep", "keep@example.com", "active", "pro", "business_suite")
	require.NoError(t, err)

	sub := &commerce.StripeSubscription{
		ID:            "sub_keep",
		Status:        "active",
		Customer:      "cus_keep",
		CustomerEmail: "keep@example.com",
		Metadata:      map[string]interface{}{},
	}

	_, err = service.persistSubscriptionFromStripe("", sub)
	require.NoError(t, err)

	var planTier string
	err = db.QueryRow(`SELECT plan_tier FROM subscriptions WHERE subscription_id = $1`, "sub_keep").Scan(&planTier)
	require.NoError(t, err)
	assert.Equal(t, "pro", planTier)
}

func TestPersistInvoiceStatus_InfersPlanTierFromPriceID(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_test", "production", 1_000_000, 0.001, "credits")
	service := requireTestStripeService(t, db)

	err := service.persistInvoiceStatus("sub_infer", "cus_infer", "infer@example.com", "price_solo_monthly", "active")
	require.NoError(t, err)

	var planTier string
	err = db.QueryRow(`SELECT plan_tier FROM subscriptions WHERE subscription_id = $1`, "sub_infer").Scan(&planTier)
	require.NoError(t, err)
	assert.Equal(t, "solo", planTier)
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
			if got := commerce.ExtractBillingCycleDay(tc.timestamp); got != tc.expected {
				t.Errorf("expected %d, got %d", tc.expected, got)
			}
		})
	}
}

// ============================================================================
// chooseUserIdentity Tests (Pure Function)
// ============================================================================

func TestChooseUserIdentity_UserHintProvided(t *testing.T) {
	sub := &commerce.StripeSubscription{
		CustomerEmail: "sub@example.com",
		Customer:      "cus_123",
	}

	result := commerce.ChooseSubscriptionUserIdentity("user@example.com", sub)
	if result != "user@example.com" {
		t.Errorf("expected user hint to be returned, got %s", result)
	}
}

func TestChooseUserIdentity_NilSubscription(t *testing.T) {
	result := commerce.ChooseSubscriptionUserIdentity("", nil)
	if result != "" {
		t.Errorf("expected empty string for nil subscription, got %s", result)
	}
}

func TestChooseUserIdentity_SubscriptionWithEmail(t *testing.T) {
	sub := &commerce.StripeSubscription{
		CustomerEmail: "sub@example.com",
		Customer:      "cus_123",
	}

	result := commerce.ChooseSubscriptionUserIdentity("", sub)
	if result != "sub@example.com" {
		t.Errorf("expected subscription email, got %s", result)
	}
}

func TestChooseUserIdentity_SubscriptionWithCustomerOnly(t *testing.T) {
	sub := &commerce.StripeSubscription{
		CustomerEmail: "",
		Customer:      "cus_123",
	}

	result := commerce.ChooseSubscriptionUserIdentity("", sub)
	if result != "cus_123" {
		t.Errorf("expected customer ID, got %s", result)
	}
}

func TestChooseUserIdentity_EmptyUserHintAndEmail(t *testing.T) {
	sub := &commerce.StripeSubscription{
		CustomerEmail: "",
		Customer:      "",
	}

	result := commerce.ChooseSubscriptionUserIdentity("", sub)
	if result != "" {
		t.Errorf("expected empty string, got %s", result)
	}
}

func TestChooseUserIdentity_WhitespaceHandling(t *testing.T) {
	sub := &commerce.StripeSubscription{
		CustomerEmail: "sub@example.com",
		Customer:      "cus_123",
	}

	// Whitespace-only user hint should fall back to subscription
	result := commerce.ChooseSubscriptionUserIdentity("   ", sub)
	if result != "sub@example.com" {
		t.Errorf("expected subscription email for whitespace hint, got %s", result)
	}
}

// ============================================================================
// VerifyStripePrice Tests (HTTP Mock)
// ============================================================================

func TestVerifyStripePrice_ValidPriceID(t *testing.T) {
	db := setupTestDB(t)
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

	service := NewStripeService(db)

	_, err := service.VerifyStripePrice("")
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestVerifyStripePrice_PriceNotFound(t *testing.T) {
	db := setupTestDB(t)
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
	resetStripeTestData(t, db)

	cfg := DefaultStripeTestConfig().WithKeys("pk_test_default", "sk_test_default", "whsec_test")
	service := ConfigureStripeService(t, db, cfg, nil)

	event := map[string]interface{}{
		"id":   "evt_sub_created_123",
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

	cfg := DefaultStripeTestConfig().WithKeys("pk_test_default", "sk_test_default", "whsec_test")
	service := ConfigureStripeService(t, db, cfg, nil)

	event := map[string]interface{}{
		"id":   "evt_sub_missing_id",
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
	resetStripeTestData(t, db)

	// Pre-create a subscription
	_, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, status, created_at, updated_at)
		VALUES ('sub_invoice', 'cus_invoice', 'active', NOW(), NOW())
	`)
	if err != nil {
		t.Fatalf("failed to insert subscription: %v", err)
	}

	cfg := DefaultStripeTestConfig().WithKeys("pk_test_default", "sk_test_default", "whsec_test")
	service := ConfigureStripeService(t, db, cfg, nil)

	event := map[string]interface{}{
		"id":   "evt_invoice_paid_123",
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
	resetStripeTestData(t, db)

	// Pre-create a subscription
	_, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, status, created_at, updated_at)
		VALUES ('sub_update', 'cus_update', 'active', NOW(), NOW())
	`)
	if err != nil {
		t.Fatalf("failed to insert subscription: %v", err)
	}

	cfg := DefaultStripeTestConfig().WithKeys("pk_test_default", "sk_test_default", "whsec_test")
	service := ConfigureStripeService(t, db, cfg, nil)

	event := map[string]interface{}{
		"id":   "evt_sub_updated_123",
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
	resetStripeTestData(t, db)

	// Pre-create a subscription
	_, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, status, created_at, updated_at)
		VALUES ('sub_delete', 'cus_delete', 'active', NOW(), NOW())
	`)
	if err != nil {
		t.Fatalf("failed to insert subscription: %v", err)
	}

	cfg := DefaultStripeTestConfig().WithKeys("pk_test_default", "sk_test_default", "whsec_test")
	service := ConfigureStripeService(t, db, cfg, nil)

	event := map[string]interface{}{
		"id":   "evt_sub_deleted_123",
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
	resetStripeTestData(t, db)

	// Pre-create a subscription
	_, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_id, status, created_at, updated_at)
		VALUES ('sub_failed', 'cus_failed', 'active', NOW(), NOW())
	`)
	if err != nil {
		t.Fatalf("failed to insert subscription: %v", err)
	}

	cfg := DefaultStripeTestConfig().WithKeys("pk_test_default", "sk_test_default", "whsec_test")
	service := ConfigureStripeService(t, db, cfg, nil)

	event := map[string]interface{}{
		"id":   "evt_invoice_failed_123",
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

	cfg := DefaultStripeTestConfig().WithKeys("pk_test_default", "sk_test_default", "whsec_test")
	service := ConfigureStripeService(t, db, cfg, nil)

	event := map[string]interface{}{
		"id":   "evt_unknown_123",
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

	service := NewStripeService(db)

	result := service.parseStripeAmount(float64(4900))
	if result != 4900 {
		t.Errorf("expected 4900, got %d", result)
	}
}

func TestParseStripeAmount_Int64(t *testing.T) {
	db := setupTestDB(t)

	service := NewStripeService(db)

	result := service.parseStripeAmount(int64(4900))
	if result != 4900 {
		t.Errorf("expected 4900, got %d", result)
	}
}

func TestParseStripeAmount_JSONNumber(t *testing.T) {
	db := setupTestDB(t)

	service := NewStripeService(db)

	result := service.parseStripeAmount(json.Number("4900"))
	if result != 4900 {
		t.Errorf("expected 4900, got %d", result)
	}
}

func TestParseStripeAmount_String(t *testing.T) {
	db := setupTestDB(t)

	service := NewStripeService(db)

	result := service.parseStripeAmount("4900")
	if result != 4900 {
		t.Errorf("expected 4900, got %d", result)
	}
}

func TestParseStripeAmount_InvalidType(t *testing.T) {
	db := setupTestDB(t)

	service := NewStripeService(db)

	result := service.parseStripeAmount(struct{}{})
	if result != 0 {
		t.Errorf("expected 0 for invalid type, got %d", result)
	}
}

func TestBillingIntervalDuration_Year(t *testing.T) {
	db := setupTestDB(t)

	service := NewStripeService(db)

	result := service.billingIntervalDuration(shared.BillingInterval_BILLING_INTERVAL_YEAR)
	expected := 365 * 24 * time.Hour
	if result != expected {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestBillingIntervalDuration_Month(t *testing.T) {
	db := setupTestDB(t)

	service := NewStripeService(db)

	result := service.billingIntervalDuration(shared.BillingInterval_BILLING_INTERVAL_MONTH)
	expected := 30 * 24 * time.Hour
	if result != expected {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestBillingIntervalDuration_Default(t *testing.T) {
	db := setupTestDB(t)

	service := NewStripeService(db)

	result := service.billingIntervalDuration(shared.BillingInterval_BILLING_INTERVAL_UNSPECIFIED)
	expected := 30 * 24 * time.Hour
	if result != expected {
		t.Errorf("expected default of %v, got %v", expected, result)
	}
}

func TestExtractAmount_FromAmountTotal(t *testing.T) {
	db := setupTestDB(t)

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
	result := maskValue("stripe-test-value-1234567890")
	if !strings.HasPrefix(result, "stri") || !strings.HasSuffix(result, "90") {
		t.Errorf("expected masked value, got '%s'", result)
	}
}

func TestMaskValue_EmptyValue(t *testing.T) {
	result := maskValue("")
	if result != "" {
		t.Errorf("expected empty string, got '%s'", result)
	}
}

// TestCreditTopup_NilBundle_ReturnsError verifies that credit topup returns an
// explicit error when the bundle product is not configured.
func TestCreditTopup_NilBundle_ReturnsError(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	// Create an empty plan store (no bundle configured) - GetBundleProduct returns nil
	emptyStore := NewPlanStoreWithOptions(commerce.PlanStoreOptions{
		PlansPath:  "", // No file - empty store
		BundleKey:  "nonexistent",
		DisplayEnv: "production",
	})

	planService := NewPlanServiceWithPlanStore(emptyStore)
	service := NewStripeServiceWithSettings(db, planService, NewPaymentSettingsService(db))

	// Create a mock plan for credit topup
	plan := &commerce.PlanOption{
		StripePriceId: "price_test",
		AmountCents:   1000,
		Kind:          shared.PlanKind_PLAN_KIND_CREDITS_TOPUP,
	}

	err := service.handleCreditTopup("test@example.com", 1000, plan, "evt_test", nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "bundle product not configured")
}

// TestHandleCustomerUpdated_EmailMigration verifies that customer email changes
// are properly propagated to all local tables.
func TestHandleCustomerUpdated_EmailMigration(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	// setupTestDB applies the embedded operations and financial schemas. Keep
	// this migration assertion on the same production tables as the runtime.
	var err error

	oldEmail := "old@example.com"
	newEmail := "new@example.com"
	customerID := "cus_migrate_123"

	// Insert test data with old email
	_, err = db.Exec(`INSERT INTO users (email, stripe_customer_id) VALUES ($1, $2)`, oldEmail, customerID)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status) VALUES ($1, $2, $3, $4)`,
		"sub_migrate_123", customerID, oldEmail, "active")
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO checkout_sessions (session_id, customer_id, customer_email, status) VALUES ($1, $2, $3, $4)`,
		"cs_migrate_123", customerID, oldEmail, "complete")
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO credit_wallets (customer_email, balance_credits) VALUES ($1, $2)`, oldEmail, 500)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO credit_transactions (customer_email, amount_credits, transaction_type) VALUES ($1, $2, $3)`,
		oldEmail, 500, "credit_topup")
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO intro_coupon_usage (email, stripe_customer_id, coupon_id) VALUES ($1, $2, $3)`,
		oldEmail, customerID, "coupon_test")
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
	assert.Equal(t, newEmail, email, "users.email should be updated")

	// Check subscriptions table
	err = db.QueryRow(`SELECT customer_email FROM subscriptions WHERE customer_id = $1`, customerID).Scan(&email)
	require.NoError(t, err)
	assert.Equal(t, newEmail, email, "subscriptions.customer_email should be updated")

	// Check checkout_sessions table
	err = db.QueryRow(`SELECT customer_email FROM checkout_sessions WHERE customer_id = $1`, customerID).Scan(&email)
	require.NoError(t, err)
	assert.Equal(t, newEmail, email, "checkout_sessions.customer_email should be updated")

	// Check credit_wallets table
	var balance int64
	err = db.QueryRow(`SELECT balance_credits FROM credit_wallets WHERE customer_email = $1`, newEmail).Scan(&balance)
	require.NoError(t, err)
	assert.Equal(t, int64(500), balance, "credit_wallets should be migrated")

	// Verify old email no longer exists in credit_wallets
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM credit_wallets WHERE customer_email = $1`, oldEmail).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "old email should not exist in credit_wallets")

	// Check credit_transactions table
	err = db.QueryRow(`SELECT customer_email FROM credit_transactions WHERE amount_credits = 500`).Scan(&email)
	require.NoError(t, err)
	assert.Equal(t, newEmail, email, "credit_transactions.customer_email should be updated")

	// Check intro_coupon_usage table
	err = db.QueryRow(`SELECT email FROM intro_coupon_usage WHERE stripe_customer_id = $1`, customerID).Scan(&email)
	require.NoError(t, err)
	assert.Equal(t, newEmail, email, "intro_coupon_usage.email should be updated")
}

// TestCreditTopup_TransactionRecorded verifies that credit transactions are
// properly recorded for auditing.
func TestCreditTopup_TransactionRecorded(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	// setupTestDB applies the embedded credit schema.
	var err error

	service := NewStripeService(db)

	metadata := map[string]interface{}{
		"price_id":   "price_test_123",
		"session_id": "cs_test_123",
	}

	err = service.creditWallet.AddCredits("audit@example.com", 250, "credit_topup", "evt_audit_123", metadata)
	require.NoError(t, err)

	// Verify transaction was recorded with all details
	var (
		customerEmail   string
		amountCredits   int64
		transactionType string
		stripeEventID   string
		metadataJSON    string
	)

	err = db.QueryRow(`
		SELECT customer_email, amount_credits, transaction_type, stripe_event_id, metadata::text
		FROM credit_transactions
		WHERE stripe_event_id = $1
	`, "evt_audit_123").Scan(&customerEmail, &amountCredits, &transactionType, &stripeEventID, &metadataJSON)
	require.NoError(t, err)

	assert.Equal(t, "audit@example.com", customerEmail)
	assert.Equal(t, int64(250), amountCredits)
	assert.Equal(t, "credit_topup", transactionType)
	assert.Equal(t, "evt_audit_123", stripeEventID)
	assert.Contains(t, metadataJSON, "price_test_123")
	assert.Contains(t, metadataJSON, "cs_test_123")
}

// TestHandleCustomerUpdated_NoOldEmail verifies that the handler works when
// previous_attributes doesn't include the old email.
func TestHandleCustomerUpdated_NoOldEmail(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	// setupTestDB applies the embedded subscriptions schema.
	var err error

	oldEmail := "lookup@example.com"
	newEmail := "updated@example.com"
	customerID := "cus_lookup_update"

	// Insert subscription with old email
	_, err = db.Exec(`INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status) VALUES ($1, $2, $3, $4)`,
		"sub_lookup_123", customerID, oldEmail, "active")
	require.NoError(t, err)

	service := NewStripeService(db)

	// Simulate customer.updated without previous_attributes
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

// TestHandleCustomerUpdated_SameEmail verifies that no-op when emails match.
func TestHandleCustomerUpdated_SameEmail(t *testing.T) {
	db := setupTestDB(t)

	// setupTestDB applies the embedded subscriptions schema.
	var err error

	email := "same@example.com"
	customerID := "cus_same_123"

	// Insert subscription
	_, err = db.Exec(`INSERT INTO subscriptions (subscription_id, customer_id, customer_email, status) VALUES ($1, $2, $3, $4)`,
		"sub_same_123", customerID, email, "active")
	require.NoError(t, err)

	service := NewStripeService(db)

	// Simulate customer.updated with same email
	customerObj := map[string]interface{}{
		"id":    customerID,
		"email": email,
		"previous_attributes": map[string]interface{}{
			"email": email,
		},
	}

	err = service.handleCustomerUpdated(customerObj)
	require.NoError(t, err) // Should be no-op, no error
}
