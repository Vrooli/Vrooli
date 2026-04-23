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
	"strings"
	"testing"
	"time"
)

func newMockStripeServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/checkout/sessions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"cs_mock","url":"https://stripe.test/cs_mock","status":"open","customer_email":"handler@example.com","customer":"cus_mock","subscription":"sub_mock","amount_total":5000,"mode":"subscription","currency":"usd"}`)
			return
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/subscriptions/"):
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"id":"%s","status":"canceled"}`, strings.TrimPrefix(r.URL.Path, "/v1/subscriptions/"))
			return
		case r.Method == http.MethodPost && r.URL.Path == "/v1/billing_portal/sessions":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"url":"https://stripe.test/portal"}`)
			return
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func signStripePayload(t *testing.T, payload []byte, timestamp string, secret string) string {
	t.Helper()
	// Use current timestamp if empty or zero (for timestamp validation compatibility)
	if timestamp == "" || timestamp == "0" {
		timestamp = fmt.Sprintf("%d", time.Now().Unix())
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "." + string(payload)))
	return "t=" + timestamp + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}

func TestHandleCheckoutCreateValidation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	resetStripeTestData(t, db)
	upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_validation", "production", 1000000, 0.001, "credits")
	service := ConfigureStripeServiceSimple(t, db)
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
	stripeServer := newMockStripeServer(t)

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
	cfg := DefaultStripeTestConfig().WithKeys("pk_test_handlers", "sk_test_handlers", "whsec_handlers")
	stripeService := ConfigureStripeService(t, db, cfg, stripeServer)

	session, err := stripeService.CreateCheckoutSession("price_handlers_sub", "/ok", "/cancel", "handler@example.com")
	if err != nil {
		t.Fatalf("checkout create failed: %v", err)
	}

	sessionID := session.SessionId
	body := map[string]interface{}{
		"id":   "evt_handler_checkout_123",
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
	req.Header.Set("Stripe-Signature", signStripePayload(t, payload, "", "whsec_handlers"))
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
	stripeServer := newMockStripeServer(t)

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
	cfg := DefaultStripeTestConfig().WithKeys("pk_test_handlers", "sk_test_handlers", "whsec_handlers")
	stripeService := ConfigureStripeService(t, db, cfg, stripeServer)
	session, err := stripeService.CreateCheckoutSession("price_credits_topup", "/ok", "/cancel", "credits@example.com")
	if err != nil {
		t.Fatalf("checkout creation failed: %v", err)
	}
	sessionID := session.SessionId

	body := map[string]interface{}{
		"id":   "evt_credits_topup_123",
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
	req.Header.Set("Stripe-Signature", signStripePayload(t, payload, "", "whsec_handlers"))
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

func TestHandleStripeWebhookInvoiceEvents(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	resetStripeTestData(t, db)

	// Seed subscription to be updated by invoice events.
	if _, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_email, status, price_id, bundle_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (subscription_id) DO NOTHING
	`, "sub_invoice", "invoice@example.com", "past_due", "price_invoice_prev", "business_suite"); err != nil {
		t.Fatalf("failed to seed subscription: %v", err)
	}

	cfg := DefaultStripeTestConfig().WithKeys("pk_test_handlers", "sk_test_handlers", "whsec_handlers")
	stripeService := ConfigureStripeService(t, db, cfg, nil)

	paidPayload := map[string]interface{}{
		"id":   "evt_invoice_paid_handler",
		"type": "invoice.paid",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"subscription":   "sub_invoice",
				"customer_email": "invoice@example.com",
				"customer":       "cus_invoice",
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
	rawPaid, _ := json.Marshal(paidPayload)
	reqPaid := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(rawPaid))
	reqPaid.Header.Set("Stripe-Signature", signStripePayload(t, rawPaid, "", "whsec_handlers"))
	recPaid := httptest.NewRecorder()
	handleStripeWebhook(stripeService).ServeHTTP(recPaid, reqPaid)
	if recPaid.Code != http.StatusOK {
		t.Fatalf("invoice.paid handler returned %d: %s", recPaid.Code, recPaid.Body.String())
	}

	var status, priceID string
	if err := db.QueryRow(`SELECT status, price_id FROM subscriptions WHERE subscription_id = $1`, "sub_invoice").Scan(&status, &priceID); err != nil {
		t.Fatalf("failed to load subscription after invoice.paid: %v", err)
	}
	if status != "active" {
		t.Fatalf("expected status active after invoice.paid, got %s", status)
	}
	if priceID != "price_invoice_new" {
		t.Fatalf("expected price updated from invoice lines, got %s", priceID)
	}

	// Payment failed should flip to past_due.
	failedPayload := map[string]interface{}{
		"id":   "evt_invoice_failed_handler",
		"type": "invoice.payment_failed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"subscription":   "sub_invoice",
				"customer_email": "invoice@example.com",
			},
		},
	}
	rawFailed, _ := json.Marshal(failedPayload)
	reqFailed := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(rawFailed))
	reqFailed.Header.Set("Stripe-Signature", signStripePayload(t, rawFailed, "", "whsec_handlers"))
	recFailed := httptest.NewRecorder()
	handleStripeWebhook(stripeService).ServeHTTP(recFailed, reqFailed)
	if recFailed.Code != http.StatusOK {
		t.Fatalf("invoice.payment_failed handler returned %d: %s", recFailed.Code, recFailed.Body.String())
	}

	if err := db.QueryRow(`SELECT status FROM subscriptions WHERE subscription_id = $1`, "sub_invoice").Scan(&status); err != nil {
		t.Fatalf("failed to reload subscription after invoice.payment_failed: %v", err)
	}
	if status != "past_due" {
		t.Fatalf("expected past_due after payment_failed, got %s", status)
	}
}

func TestHandleStripeWebhookSubscriptionLifecycle(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	resetStripeTestData(t, db)

	cfg := DefaultStripeTestConfig().WithKeys("pk_test_handlers", "sk_test_handlers", "whsec_handlers")
	stripeService := ConfigureStripeService(t, db, cfg, nil)
	// Seed subscription row to be updated by lifecycle events
	if _, err := db.Exec(`INSERT INTO subscriptions (subscription_id, status, created_at, updated_at) VALUES ($1,$2,NOW(),NOW()) ON CONFLICT (subscription_id) DO UPDATE SET status = EXCLUDED.status`, "sub_lifecycle", "active"); err != nil {
		t.Fatalf("failed to seed subscription: %v", err)
	}

	events := []struct {
		eventID   string
		eventType string
		payload   map[string]interface{}
	}{
		{
			eventID:   "evt_lifecycle_updated",
			eventType: "customer.subscription.updated",
			payload: map[string]interface{}{
				"id":     "sub_lifecycle",
				"status": "past_due",
			},
		},
		{
			eventID:   "evt_lifecycle_deleted",
			eventType: "customer.subscription.deleted",
			payload: map[string]interface{}{
				"id":     "sub_lifecycle",
				"status": "canceled",
			},
		},
	}

	for i, evt := range events {
		body := map[string]interface{}{
			"id":   evt.eventID,
			"type": evt.eventType,
			"data": map[string]interface{}{
				"object": evt.payload,
			},
		}
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewReader(raw))
		req.Header.Set("Stripe-Signature", signStripePayload(t, raw, "", "whsec_handlers"))
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

	resetStripeTestData(t, db)

	paymentService := NewPaymentSettingsService(db)
	stripeService := NewStripeServiceWithSettings(db, NewPlanService(db), paymentService)

	update := handleUpdateStripeSettings(paymentService, stripeService, nil)
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
	stripeServer := newMockStripeServer(t)

	resetStripeTestData(t, db)

	now := time.Now()
	if _, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_email, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
		ON CONFLICT (subscription_id) DO UPDATE SET status = EXCLUDED.status, updated_at = EXCLUDED.updated_at
	`, "sub_cancelme", "cancelme@example.com", "active", now); err != nil {
		t.Fatalf("failed to seed subscription: %v", err)
	}

	stripeService := ConfigureStripeService(t, db, DefaultStripeTestConfig(), stripeServer)

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

func TestHandleStripeWebhookRequiresSignature(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	resetStripeTestData(t, db)

	stripeService := ConfigureStripeServiceSimple(t, db)
	handler := handleStripeWebhook(stripeService)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe", bytes.NewBufferString(`{"type":"test.event"}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing signature, got %d", rec.Code)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM subscriptions`).Scan(&count); err != nil {
		t.Fatalf("failed to count subscriptions: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no subscriptions created when signature missing, found %d", count)
	}
}
