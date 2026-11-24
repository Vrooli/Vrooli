package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPY1StripeTestClockCreateAdvanceAndWaitUseTestOnlyRequests(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	t.Setenv("VROOLI_ENVIRONMENT", "")
	t.Setenv("STRIPE_MODE", "test")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk_test_fixture" {
			t.Fatalf("authorization header missing test key")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "POST /v1/test_helpers/test_clocks":
			if err := r.ParseForm(); err != nil || r.Form.Get("frozen_time") != "1700000000" || !strings.HasPrefix(r.Form.Get("name"), "PY1 py1-run-") || r.Header.Get("Idempotency-Key") != "py1-run-1-test-clock-create" {
				t.Fatalf("create request form=%v idempotency=%q err=%v", r.Form, r.Header.Get("Idempotency-Key"), err)
			}
			_, _ = w.Write([]byte(`{"id":"clock_fixture","frozen_time":1700000000,"status":"ready","livemode":false}`))
		case "POST /v1/test_helpers/test_clocks/clock_fixture/advance":
			if err := r.ParseForm(); err != nil || r.Form.Get("frozen_time") != "1700086400" || r.Header.Get("Idempotency-Key") != "py1-run-1-clock-advance-1700086400" {
				t.Fatalf("advance request form=%v idempotency=%q err=%v", r.Form, r.Header.Get("Idempotency-Key"), err)
			}
			_, _ = w.Write([]byte(`{"id":"clock_fixture","frozen_time":1700086400,"status":"advancing","livemode":false}`))
		case "GET /v1/test_helpers/test_clocks/clock_fixture":
			_, _ = w.Write([]byte(`{"id":"clock_fixture","frozen_time":1700086400,"status":"ready","livemode":false}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	service := NewStripeService(setupTestDB(t))
	service.UseHTTPClient(server.Client())
	service.UseConfigLoader(func(context.Context) (stripeRuntimeConfig, error) {
		return stripeRuntimeConfig{secretKey: "sk_test_fixture", hasSecret: true, apiBase: server.URL}, nil
	})
	if err := service.RefreshConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	clock, err := service.createPY1TestClock(context.Background(), "py1-run-1", 1700000000)
	if err != nil || clock.ID != "clock_fixture" {
		t.Fatalf("createPY1TestClock() = %+v, %v", clock, err)
	}
	clock, err = service.advancePY1TestClock(context.Background(), "py1-run-1", clock.ID, 1700086400)
	if err != nil || clock.Status != "advancing" {
		t.Fatalf("advancePY1TestClock() = %+v, %v", clock, err)
	}
	clock, err = service.waitForPY1TestClock(context.Background(), "py1-run-1", clock.ID, 1)
	if err != nil || clock.Status != "ready" || clock.FrozenAt != 1700086400 {
		t.Fatalf("waitForPY1TestClock() = %+v, %v", clock, err)
	}
}

func TestPY1StripeTestClockRefusesNonTestAndUnscopedRequests(t *testing.T) {
	service := NewStripeService(setupTestDB(t))
	service.UseConfigLoader(func(context.Context) (stripeRuntimeConfig, error) {
		return stripeRuntimeConfig{secretKey: "sk_test_fixture", hasSecret: true}, nil
	})
	if err := service.RefreshConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LPBS_ENVIRONMENT", "development")
	t.Setenv("VROOLI_ENVIRONMENT", "")
	t.Setenv("STRIPE_MODE", "live")
	if _, err := service.createPY1TestClock(context.Background(), "py1-run-1", 1700000000); err == nil {
		t.Fatal("test clock accepted STRIPE_MODE=live")
	}
	t.Setenv("STRIPE_MODE", "test")
	if _, err := service.createPY1TestClock(context.Background(), "other-run", 1700000000); err == nil {
		t.Fatal("test clock accepted an unscoped run ID")
	}
	if _, err := service.advancePY1TestClock(context.Background(), "py1-run-1", "../clock_fixture", 1700086400); err == nil {
		t.Fatal("test clock accepted a non-clock path")
	}
}

func TestPY1StripeTestClockSubscriptionUsesARealTestPriceAndScopedMetadata(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	t.Setenv("VROOLI_ENVIRONMENT", "")
	t.Setenv("STRIPE_MODE", "test")
	var steps int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk_test_fixture" {
			t.Fatalf("authorization header missing test key")
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "POST /v1/customers":
			steps++
			if r.Form.Get("test_clock") != "clock_fixture" || r.Form.Get("email") != "py1-run-1@example.test" || r.Form.Get("metadata[lpbs_py1_test_run_id]") != "py1-run-1" || r.Form.Get("metadata[lpbs_py1_test_outcome]") != "renewal" || r.Header.Get("Idempotency-Key") != "py1-run-1-renewal-customer-create" {
				t.Fatalf("customer request form=%v idempotency=%q", r.Form, r.Header.Get("Idempotency-Key"))
			}
			_, _ = w.Write([]byte(`{"id":"cus_fixture"}`))
		case "POST /v1/payment_methods":
			steps++
			if r.Form.Get("type") != "card" || r.Form.Get("card[number]") != "4242424242424242" || r.Header.Get("Idempotency-Key") != "py1-run-1-renewal-payment-method-create" {
				t.Fatalf("payment method request form=%v idempotency=%q", r.Form, r.Header.Get("Idempotency-Key"))
			}
			_, _ = w.Write([]byte(`{"id":"pm_fixture"}`))
		case "POST /v1/payment_methods/pm_fixture/attach":
			steps++
			if r.Form.Get("customer") != "cus_fixture" || r.Header.Get("Idempotency-Key") != "py1-run-1-renewal-payment-method-attach" {
				t.Fatalf("attach request form=%v idempotency=%q", r.Form, r.Header.Get("Idempotency-Key"))
			}
			_, _ = w.Write([]byte(`{"id":"pm_fixture","customer":"cus_fixture"}`))
		case "POST /v1/subscriptions":
			steps++
			if r.Form.Get("customer") != "cus_fixture" || r.Form.Get("items[0][price]") != "price_fixture" || r.Form.Get("default_payment_method") != "pm_fixture" || r.Form.Get("payment_behavior") != "error_if_incomplete" || r.Form.Get("metadata[lpbs_py1_test_run_id]") != "py1-run-1" || r.Form.Get("metadata[lpbs_py1_test_outcome]") != "renewal" || r.Header.Get("Idempotency-Key") != "py1-run-1-renewal-subscription-create" {
				t.Fatalf("subscription request form=%v idempotency=%q", r.Form, r.Header.Get("Idempotency-Key"))
			}
			_, _ = w.Write([]byte(`{"id":"sub_fixture","status":"active","current_period_end":1700086400,"items":{"data":[{"id":"si_fixture"}]}}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	service := NewStripeService(setupTestDB(t))
	service.UseHTTPClient(server.Client())
	service.UseConfigLoader(func(context.Context) (stripeRuntimeConfig, error) {
		return stripeRuntimeConfig{secretKey: "sk_test_fixture", hasSecret: true, apiBase: server.URL}, nil
	})
	if err := service.RefreshConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := service.createPY1TestClockSubscription(context.Background(), "py1-run-1", "clock_fixture", "price_fixture", "renewal")
	if err != nil || steps != 4 || got.CustomerID != "cus_fixture" || got.PaymentMethodID != "pm_fixture" || got.SubscriptionID != "sub_fixture" || got.SubscriptionItemID != "si_fixture" || got.CurrentPeriodEnd != 1700086400 || got.SubscriptionState != "active" {
		t.Fatalf("createPY1TestClockSubscription() = %+v, %v; steps=%d", got, err, steps)
	}
}

func TestPY1ClockLifecycleControlsUseScopedCancelAndProratedPriceChanges(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	t.Setenv("VROOLI_ENVIRONMENT", "")
	t.Setenv("STRIPE_MODE", "test")
	var seen int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk_test_fixture" {
			t.Fatalf("authorization header missing test key")
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "POST /v1/subscriptions/sub_fixture":
			seen++
			if paymentMethod := r.Form.Get("default_payment_method"); paymentMethod != "" {
				if paymentMethod != "pm_decline_fixture" || r.Form.Get("metadata[lpbs_py1_test_run_id]") != "py1-run-1" || r.Form.Get("metadata[lpbs_py1_test_outcome]") != "past_due" || r.Header.Get("Idempotency-Key") != "py1-run-1-default-payment-past_due" {
					t.Fatalf("payment method form=%v idempotency=%q", r.Form, r.Header.Get("Idempotency-Key"))
				}
				_, _ = w.Write([]byte(`{"id":"sub_fixture","default_payment_method":"pm_decline_fixture"}`))
				return
			}
			if r.Form.Get("cancel_at_period_end") == "true" {
				if r.Form.Get("metadata[lpbs_py1_test_run_id]") != "py1-run-1" || r.Form.Get("metadata[lpbs_py1_test_outcome]") != "cancel" || r.Header.Get("Idempotency-Key") != "py1-run-1-schedule-cancel" {
					t.Fatalf("cancel form=%v idempotency=%q", r.Form, r.Header.Get("Idempotency-Key"))
				}
				_, _ = w.Write([]byte(`{"id":"sub_fixture","status":"active","cancel_at_period_end":true}`))
				return
			}
			outcome := r.Form.Get("metadata[lpbs_py1_test_outcome]")
			price := r.Form.Get("items[0][price]")
			if r.Form.Get("items[0][id]") != "si_fixture" || r.Form.Get("proration_behavior") != "always_invoice" || r.Form.Get("metadata[lpbs_py1_test_run_id]") != "py1-run-1" || (outcome != "upgrade" && outcome != "downgrade") || (price != "price_upgrade" && price != "price_downgrade") || r.Header.Get("Idempotency-Key") != "py1-run-1-"+outcome {
				t.Fatalf("price change form=%v idempotency=%q", r.Form, r.Header.Get("Idempotency-Key"))
			}
			_, _ = w.Write([]byte(`{"id":"sub_fixture","status":"active"}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	service := NewStripeService(setupTestDB(t))
	service.UseHTTPClient(server.Client())
	service.UseConfigLoader(func(context.Context) (stripeRuntimeConfig, error) {
		return stripeRuntimeConfig{secretKey: "sk_test_fixture", hasSecret: true, apiBase: server.URL}, nil
	})
	if err := service.RefreshConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := service.setPY1TestClockDefaultPaymentMethod(ctx, "py1-run-1", "sub_fixture", "pm_decline_fixture", "past_due"); err != nil {
		t.Fatalf("set decline payment method: %v", err)
	}
	canceled, err := service.schedulePY1TestClockCancellation(ctx, "py1-run-1", "sub_fixture")
	if err != nil || !canceled.CancelAtPeriodEnd || canceled.SubscriptionState != "active" {
		t.Fatalf("schedule cancellation = %+v, %v", canceled, err)
	}
	for _, item := range []struct{ outcome, price string }{{"upgrade", "price_upgrade"}, {"downgrade", "price_downgrade"}} {
		updated, err := service.changePY1TestClockSubscriptionPrice(ctx, "py1-run-1", "sub_fixture", "si_fixture", item.price, item.outcome)
		if err != nil || updated.SubscriptionState != "active" {
			t.Fatalf("%s result = %+v, %v", item.outcome, updated, err)
		}
	}
	if seen != 4 {
		t.Fatalf("received %d Stripe mutation requests, want 4", seen)
	}
}

func TestPY1DeclinePaymentMethodUsesOnlyTheScopedFailureCard(t *testing.T) {
	t.Setenv("LPBS_ENVIRONMENT", "development")
	t.Setenv("VROOLI_ENVIRONMENT", "")
	t.Setenv("STRIPE_MODE", "test")
	var seen int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk_test_fixture" {
			t.Fatalf("authorization header missing test key")
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "POST /v1/payment_methods":
			seen++
			if r.Form.Get("card[number]") != "4000000000000341" || r.Form.Get("type") != "card" || r.Header.Get("Idempotency-Key") != "py1-run-1-past-due-failure-payment-method-create" {
				t.Fatalf("failure card request form=%v key=%q", r.Form, r.Header.Get("Idempotency-Key"))
			}
			_, _ = w.Write([]byte(`{"id":"pm_failure_fixture"}`))
		case "POST /v1/payment_methods/pm_failure_fixture/attach":
			seen++
			if r.Form.Get("customer") != "cus_fixture" || r.Header.Get("Idempotency-Key") != "py1-run-1-past-due-failure-payment-method-attach" {
				t.Fatalf("failure card attach form=%v key=%q", r.Form, r.Header.Get("Idempotency-Key"))
			}
			_, _ = w.Write([]byte(`{"id":"pm_failure_fixture","customer":"cus_fixture"}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	service := NewStripeService(setupTestDB(t))
	service.UseHTTPClient(server.Client())
	service.UseConfigLoader(func(context.Context) (stripeRuntimeConfig, error) {
		return stripeRuntimeConfig{secretKey: "sk_test_fixture", hasSecret: true, apiBase: server.URL}, nil
	})
	if err := service.RefreshConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := service.createPY1DecliningRenewalPaymentMethod(context.Background(), "py1-run-1", "cus_fixture")
	if err != nil || got != "pm_failure_fixture" || seen != 2 {
		t.Fatalf("decline payment method = %q, requests=%d, err=%v", got, seen, err)
	}
}
