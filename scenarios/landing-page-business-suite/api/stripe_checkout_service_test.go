package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateTestOutcomeCheckoutSessionUsesTheRealCheckoutPathWithScopedMetadata(t *testing.T) {
	t.Setenv("VROOLI_ENVIRONMENT", "test")
	t.Setenv("LPBS_ENVIRONMENT", "development")
	t.Setenv("STRIPE_MODE", "test")
	db := setupTestDB(t)
	productID := upsertTestBundleProduct(t, db, "business_suite", "Business Suite", "prod_py1_checkout", "test", 1000000, 0.001, "credits")
	insertBundlePrice(t, db, productID, "price_py1_checkout", "PY1 Test Checkout", "solo", "month", "usd", 2900, false, "", 0, 0, "", 0, 0, 1, 10, "none", sessionTypeSubscription, map[string]interface{}{})

	stripeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/checkout/sessions" {
			http.NotFound(w, r)
			return
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse Checkout request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if got := r.Form.Get("metadata[lpbs_py1_test_run_id]"); got != "py1-run-checkout" {
			t.Errorf("checkout run metadata = %q", got)
		}
		if got := r.Form.Get("metadata[lpbs_py1_test_outcome]"); got != "checkout_success" {
			t.Errorf("checkout outcome metadata = %q", got)
		}
		if got := r.Form.Get("subscription_data[metadata][lpbs_py1_test_run_id]"); got != "py1-run-checkout" {
			t.Errorf("subscription run metadata = %q", got)
		}
		_, _ = fmt.Fprint(w, `{"id":"cs_py1_checkout","url":"https://checkout.stripe.test/cs_py1_checkout","status":"open","customer_email":"py1@example.com","customer":"cus_py1","subscription":"sub_py1","amount_total":2900}`)
	}))
	defer stripeServer.Close()
	service := ConfigureStripeService(t, db, DefaultStripeTestConfig().WithKeys("pk_test_py1", "sk_test_py1", "whsec_py1"), stripeServer)
	session, err := service.CreateTestOutcomeCheckoutSession(context.Background(), "price_py1_checkout", "/success", "/cancel", "py1@example.com", "py1-run-checkout", "checkout_success")
	if err != nil {
		t.Fatalf("CreateTestOutcomeCheckoutSession(): %v", err)
	}
	if session.GetSessionId() != "cs_py1_checkout" {
		t.Fatalf("checkout session ID = %q", session.GetSessionId())
	}
	var raw string
	if err := db.QueryRow(`SELECT metadata FROM checkout_sessions WHERE session_id='cs_py1_checkout'`).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
		t.Fatalf("decode local checkout metadata: %v", err)
	}
	if metadata["lpbs_py1_test_run_id"] != "py1-run-checkout" || metadata["lpbs_py1_test_outcome"] != "checkout_success" {
		t.Fatalf("stored checkout metadata = %+v", metadata)
	}
}

func TestCreateTestOutcomeCheckoutSessionRefusesUnknownOutcome(t *testing.T) {
	t.Setenv("VROOLI_ENVIRONMENT", "test")
	t.Setenv("STRIPE_MODE", "test")
	db := setupTestDB(t)
	service := ConfigureStripeService(t, db, DefaultStripeTestConfig().WithKeys("pk_test_py1", "sk_test_py1", "whsec_py1"), nil)
	if _, err := service.CreateTestOutcomeCheckoutSession(context.Background(), "price_unused", "/success", "/cancel", "py1@example.com", "py1-run", "unrecognized"); err == nil {
		t.Fatal("unknown test outcome checkout was accepted")
	}
}
