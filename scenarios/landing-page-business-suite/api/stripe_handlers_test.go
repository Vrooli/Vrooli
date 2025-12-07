package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func setStripeEnv(t *testing.T) func() {
	t.Helper()
	os.Setenv("STRIPE_PUBLISHABLE_KEY", "pk_test_handlers")
	os.Setenv("STRIPE_SECRET_KEY", "sk_test_handlers")
	os.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_handlers")
	return func() {
		os.Unsetenv("STRIPE_PUBLISHABLE_KEY")
		os.Unsetenv("STRIPE_SECRET_KEY")
		os.Unsetenv("STRIPE_WEBHOOK_SECRET")
	}
}

func signStripePayload(t *testing.T, payload []byte, timestamp string, secret string) string {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "." + string(payload)))
	return "t=" + timestamp + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}

func TestHandleCheckoutCreateValidation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanup := setStripeEnv(t)
	defer cleanup()

	service := NewStripeService(db)
	handler := handleCheckoutCreate(service)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/checkout/create", bytes.NewBufferString(`{"price_id":""}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing fields, got %d", rec.Code)
	}
}

func TestHandleCheckoutCreateAndWebhookEndToEnd(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanup := setStripeEnv(t)
	defer cleanup()

	resetStripeTestData(t, db)

	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_handlers", "production", 1000000, 0.001, "credits")
	insertBundlePrice(
		t,
		db,
		productID,
		"price_handlers_sub",
		"Handlers Plan",
		"pro",
		"month",
		"usd",
		5000,
		true,
		"flat_amount",
		100,
		1,
		"intro_lookup",
		1000000,
		0,
		1,
		10,
		"none",
		sessionTypeSubscription,
		map[string]interface{}{"features": []string{"Handlers coverage"}},
	)
	stripeService := NewStripeServiceWithSettings(db, NewPlanService(db), NewPaymentSettingsService(db))

	session, err := stripeService.CreateCheckoutSession("price_handlers_sub", "/ok", "/cancel", "handler@example.com")
	if err != nil {
		t.Fatalf("checkout create failed: %v", err)
	}

	sessionID := session.SessionId
	body := map[string]interface{}{
		"type": "checkout.session.completed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":             sessionID,
				"customer_email": "handler@example.com",
				"subscription":   "sub_handlers_123",
				"amount_total":   5000,
			},
		},
	}
	payload, _ := json.Marshal(body)

	handler := handleStripeWebhook(stripeService)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", signStripePayload(t, payload, "1700000000", "whsec_handlers"))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("webhook handler returned %d: %s", rec.Code, rec.Body.String())
	}

	var status string
	if err := db.QueryRow(`SELECT status FROM subscriptions WHERE subscription_id = $1`, "sub_handlers_123").Scan(&status); err != nil {
		t.Fatalf("expected subscription row: %v", err)
	}
	if status != "active" {
		t.Fatalf("expected active subscription, got %s", status)
	}

	verifyHandler := handleSubscriptionVerify(stripeService)
	verifyReq := httptest.NewRequest(http.MethodGet, "/api/v1/subscription/verify?user=handler@example.com", nil)
	verifyRec := httptest.NewRecorder()
	verifyHandler.ServeHTTP(verifyRec, verifyReq)
	if verifyRec.Code != http.StatusOK {
		t.Fatalf("verify handler returned %d", verifyRec.Code)
	}
}

func TestHandleStripeWebhookCreditTopup(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanup := setStripeEnv(t)
	defer cleanup()

	resetStripeTestData(t, db)

	productID := upsertTestBundleProduct(t, db, "credits_bundle", "Credits Bundle", "prod_credits", "production", 1000, 1, "credits")
	insertBundlePrice(
		t,
		db,
		productID,
		"price_credits_topup",
		"Credits Topup",
		"credits",
		"one_time",
		"usd",
		9900,
		false,
		"",
		0,
		0,
		"",
		0,
		0,
		1,
		0,
		"",
		sessionTypeCreditsTopup,
		map[string]interface{}{},
	)
	stripeService := NewStripeServiceWithSettings(db, NewPlanService(db), NewPaymentSettingsService(db))
	session, err := stripeService.CreateCheckoutSession("price_credits_topup", "/ok", "/cancel", "credits@example.com")
	if err != nil {
		t.Fatalf("checkout creation failed: %v", err)
	}
	sessionID := session.SessionId

	body := map[string]interface{}{
		"type": "checkout.session.completed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":             sessionID,
				"customer_email": "credits@example.com",
				"subscription":   "",
				"amount_total":   9900,
			},
		},
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", signStripePayload(t, payload, "1700000001", "whsec_handlers"))
	rec := httptest.NewRecorder()

	handleStripeWebhook(stripeService).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("webhook handler failed: %d %s", rec.Code, rec.Body.String())
	}

	var balance int64
	if err := db.QueryRow(`SELECT balance_credits FROM credit_wallets WHERE customer_email = $1`, "credits@example.com").Scan(&balance); err != nil {
		t.Fatalf("expected credit wallet: %v", err)
	}
	if balance == 0 {
		t.Fatalf("expected non-zero credit balance after top-up")
	}
}

func TestHandleStripeWebhookSubscriptionLifecycle(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanup := setStripeEnv(t)
	defer cleanup()

	resetStripeTestData(t, db)

	stripeService := NewStripeService(db)
	// Seed subscription row to be updated by lifecycle events
	if _, err := db.Exec(`INSERT INTO subscriptions (subscription_id, status, created_at, updated_at) VALUES ($1,$2,NOW(),NOW()) ON CONFLICT (subscription_id) DO UPDATE SET status = EXCLUDED.status`, "sub_lifecycle", "active"); err != nil {
		t.Fatalf("failed to seed subscription: %v", err)
	}

	events := []struct {
		eventType string
		payload   map[string]interface{}
	}{
		{
			eventType: "customer.subscription.updated",
			payload: map[string]interface{}{
				"id":     "sub_lifecycle",
				"status": "past_due",
			},
		},
		{
			eventType: "customer.subscription.deleted",
			payload: map[string]interface{}{
				"id":     "sub_lifecycle",
				"status": "canceled",
			},
		},
	}

	for i, evt := range events {
		body := map[string]interface{}{
			"type": evt.eventType,
			"data": map[string]interface{}{
				"object": evt.payload,
			},
		}
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(raw))
		req.Header.Set("Stripe-Signature", signStripePayload(t, raw, time.Now().Format("150405"), "whsec_handlers"))
		rec := httptest.NewRecorder()

		handleStripeWebhook(stripeService).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("event %d (%s) failed: %d %s", i, evt.eventType, rec.Code, rec.Body.String())
		}
	}

	var status string
	var canceledAt *time.Time
	if err := db.QueryRow(`SELECT status, canceled_at FROM subscriptions WHERE subscription_id = $1`, "sub_lifecycle").Scan(&status, &canceledAt); err != nil {
		t.Fatalf("query lifecycle subscription failed: %v", err)
	}
	if status != "canceled" {
		t.Fatalf("expected final status canceled, got %s", status)
	}
	if canceledAt == nil {
		t.Fatalf("expected canceled_at to be set")
	}
}

func TestStripeSettingsHandlers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanup := setStripeEnv(t)
	defer cleanup()

	resetStripeTestData(t, db)

	paymentService := NewPaymentSettingsService(db)
	stripeService := NewStripeServiceWithSettings(db, NewPlanService(db), paymentService)

	update := handleUpdateStripeSettings(paymentService, stripeService)
	body := bytes.NewBufferString(`{"publishable_key":"pk_live_handlers","secret_key":"sk_live_handlers","webhook_secret":"whsec_live_handlers","dashboard_url":"https://dashboard.stripe.com/test"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/settings/stripe", body)
	rec := httptest.NewRecorder()
	update.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from update handler, got %d (%s)", rec.Code, rec.Body.String())
	}

	get := handleGetStripeSettings(paymentService, stripeService)
	getRec := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings/stripe", nil)
	get.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200 from get handler, got %d", getRec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(getRec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	snapshot, _ := resp["snapshot"].(map[string]any)
	settings, _ := resp["settings"].(map[string]any)

	sourceVal := snapshot["source"]
	sourceStr, _ := sourceVal.(string)
	if sourceStr == "" {
		if num, ok := sourceVal.(float64); ok {
			sourceStr = fmt.Sprintf("%v", num)
		}
	}
	if sourceStr == "" {
		t.Fatalf("expected source to be present in snapshot")
	}
	if !snapshot["publishable_key_set"].(bool) || !snapshot["secret_key_set"].(bool) || !snapshot["webhook_secret_set"].(bool) {
		t.Fatalf("expected all stripe keys to be marked as set")
	}
	if settings["dashboard_url"] == nil || settings["dashboard_url"] == "" {
		t.Fatalf("expected dashboard url returned")
	}
}

func TestSubscriptionHandlersVerifyAndCancel(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanup := setStripeEnv(t)
	defer cleanup()

	resetStripeTestData(t, db)

	now := time.Now()
	if _, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_email, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
		ON CONFLICT (subscription_id) DO UPDATE SET status = EXCLUDED.status, updated_at = EXCLUDED.updated_at
	`, "sub_cancelme", "cancelme@example.com", "active", now); err != nil {
		t.Fatalf("failed to seed subscription: %v", err)
	}

	stripeService := NewStripeService(db)

	verifyRec := httptest.NewRecorder()
	verifyReq := httptest.NewRequest(http.MethodGet, "/api/v1/subscription/verify?user=cancelme@example.com", nil)
	handleSubscriptionVerify(stripeService).ServeHTTP(verifyRec, verifyReq)
	if verifyRec.Code != http.StatusOK {
		t.Fatalf("verify handler returned %d", verifyRec.Code)
	}

	cancelBody := bytes.NewBufferString(`{"user_identity":"cancelme@example.com"}`)
	cancelReq := httptest.NewRequest(http.MethodPost, "/api/v1/subscription/cancel", cancelBody)
	cancelRec := httptest.NewRecorder()
	handleSubscriptionCancel(stripeService).ServeHTTP(cancelRec, cancelReq)
	if cancelRec.Code != http.StatusOK {
		t.Fatalf("cancel handler returned %d: %s", cancelRec.Code, cancelRec.Body.String())
	}

	var status string
	if err := db.QueryRow(`SELECT status FROM subscriptions WHERE subscription_id = $1`, "sub_cancelme").Scan(&status); err != nil {
		t.Fatalf("failed to reload subscription: %v", err)
	}
	if status != "canceled" {
		t.Fatalf("expected canceled status, got %s", status)
	}
}
