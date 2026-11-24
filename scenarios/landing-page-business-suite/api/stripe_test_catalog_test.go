package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateStripeTestPriceIsExplicitlyTestModeAndIdempotent(t *testing.T) {
	t.Setenv("STRIPE_MODE", "test")
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if r.Method != http.MethodPost || r.URL.Path != "/v1/prices" {
			t.Fatalf("request = %s %s, want POST /v1/prices", r.Method, r.URL.Path)
		}
		if r.Header.Get("Idempotency-Key") != "py1-test-reconcile-pro-year" {
			t.Fatalf("idempotency key = %q", r.Header.Get("Idempotency-Key"))
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		if r.Form.Get("product") != "prod_test" || r.Form.Get("unit_amount") != "64800" || r.Form.Get("currency") != "usd" || r.Form.Get("recurring[interval]") != "year" {
			t.Fatalf("form = %#v", r.Form)
		}
		if r.Form.Has("active") {
			t.Fatal("request tried to set Stripe's read-only active field")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"price_created","active":true,"currency":"usd","unit_amount":64800}`))
	}))
	defer server.Close()

	service := NewStripeService(setupTestDB(t))
	service.UseHTTPClient(server.Client())
	service.UseConfigLoader(func(context.Context) (stripeRuntimeConfig, error) {
		return stripeRuntimeConfig{secretKey: "sk_test_fixture", hasSecret: true, apiBase: server.URL}, nil
	})
	if err := service.RefreshConfig(context.Background()); err != nil {
		t.Fatalf("RefreshConfig() error = %v", err)
	}
	got, err := service.createStripeTestPrice(context.Background(), "prod_test", "Pro year", 64800, "year", "py1-test-reconcile-pro-year")
	if err != nil {
		t.Fatalf("createStripeTestPrice() error = %v", err)
	}
	if got != "price_created" || requestCount != 1 {
		t.Fatalf("id = %q, requests = %d", got, requestCount)
	}
}

func TestCreateStripeTestPriceRefusesLiveMode(t *testing.T) {
	t.Setenv("STRIPE_MODE", "live")
	service := NewStripeService(setupTestDB(t))
	service.UseConfigLoader(func(context.Context) (stripeRuntimeConfig, error) {
		return stripeRuntimeConfig{secretKey: "sk_test_fixture", hasSecret: true}, nil
	})
	if err := service.RefreshConfig(context.Background()); err != nil {
		t.Fatalf("RefreshConfig() error = %v", err)
	}
	if _, err := service.createStripeTestPrice(context.Background(), "prod_test", "Pro year", 64800, "year", "py1-test-reconcile-pro-year"); err == nil || !strings.Contains(err.Error(), "test-mode key") {
		t.Fatalf("createStripeTestPrice() error = %v, want test-mode refusal", err)
	}
}
