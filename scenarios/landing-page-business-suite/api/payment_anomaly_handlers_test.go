package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	landing_page_business_suite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
)

func TestUpdate_RejectsEnabledWithoutURL(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	paymentService := NewPaymentSettingsService(db)
	stripeService := NewStripeServiceWithSettings(db, NewPlanService(db), paymentService)
	anomalyService := NewPaymentAnomalyService(context.Background(), db, context.Background())

	h := stripeSettingsConnectHandler{payment: paymentService, stripe: stripeService, anomaly: anomalyService}
	_, err := h.UpdateStripeSettings(context.Background(), connect.NewRequest(&landing_page_business_suite_v1.UpdateStripeSettingsRequest{AnomalyWebhookEnabled: protoBool(true)}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func TestUpdate_RejectsNonHTTPSWebhookURL(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	paymentService := NewPaymentSettingsService(db)
	stripeService := NewStripeServiceWithSettings(db, NewPlanService(db), paymentService)
	anomalyService := NewPaymentAnomalyService(context.Background(), db, context.Background())

	h := stripeSettingsConnectHandler{payment: paymentService, stripe: stripeService, anomaly: anomalyService}
	_, err := h.UpdateStripeSettings(context.Background(), connect.NewRequest(&landing_page_business_suite_v1.UpdateStripeSettingsRequest{AnomalyWebhookUrl: protoString("http://insecure.example.com/hook")}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func TestUpdate_RefreshesAnomalyConfig(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	var received atomic.Int32
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer stub.Close()

	paymentService := NewPaymentSettingsService(db)
	stripeService := NewStripeServiceWithSettings(db, NewPlanService(db), paymentService)
	anomalyService := NewPaymentAnomalyService(context.Background(), db, context.Background())

	if cfg := anomalyService.currentConfig(); cfg.enabled || cfg.webhookURL != "" {
		t.Fatalf("baseline config non-empty: %+v", cfg)
	}

	h := stripeSettingsConnectHandler{payment: paymentService, stripe: stripeService, anomaly: anomalyService}
	// Convert stub.URL (http://) into an https-looking URL for validation,
	// but we need the stub to receive. So we swap to a non-https check by
	// testing the push-on-PATCH behaviour with a URL we know was accepted.
	// Easier: preload the settings row directly to skip the https validation,
	// then PATCH with just anomaly_webhook_enabled.
	if _, err := db.Exec(`
		INSERT INTO payment_settings (id, publishable_key, secret_key, webhook_secret, anomaly_webhook_url, anomaly_webhook_enabled, anomaly_rate_limits, updated_at)
		VALUES (1, '', '', '', $1, FALSE, '{}'::jsonb, NOW())
		ON CONFLICT (id) DO UPDATE SET anomaly_webhook_url = EXCLUDED.anomaly_webhook_url
	`, stub.URL); err != nil {
		t.Fatal(err)
	}

	if _, err := h.UpdateStripeSettings(context.Background(), connect.NewRequest(&landing_page_business_suite_v1.UpdateStripeSettingsRequest{AnomalyWebhookEnabled: protoBool(true)})); err != nil {
		t.Fatalf("update: %v", err)
	}

	// The config snapshot must have picked up the change without restarting.
	cfg := anomalyService.currentConfig()
	if !cfg.enabled || cfg.webhookURL != stub.URL {
		t.Fatalf("config not refreshed after PATCH: %+v", cfg)
	}

	// Issue a Log to prove end-to-end dispatch on the refreshed config.
	id, err := anomalyService.Log(context.Background(), PaymentAnomaly{Type: "post_refresh"})
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	status, err := anomalyService.WaitForDispatch(ctx, id)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if status != anomalyDispatchSent {
		t.Fatalf("expected sent, got %q", status)
	}
	if received.Load() != 1 {
		t.Fatalf("expected 1 POST, got %d", received.Load())
	}
}

func TestUpdate_AcceptsRateLimitsObject(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	paymentService := NewPaymentSettingsService(db)
	stripeService := NewStripeServiceWithSettings(db, NewPlanService(db), paymentService)
	anomalyService := NewPaymentAnomalyService(context.Background(), db, context.Background())

	h := stripeSettingsConnectHandler{payment: paymentService, stripe: stripeService, anomaly: anomalyService}
	if _, err := h.UpdateStripeSettings(context.Background(), connect.NewRequest(&landing_page_business_suite_v1.UpdateStripeSettingsRequest{AnomalyRateLimits: protoString(`{"checkout_subscription_missing":{"burst":3,"refill_seconds":300}}`)})); err != nil {
		t.Fatalf("update: %v", err)
	}

	cfg := anomalyService.currentConfig()
	override, ok := cfg.rateLimits["checkout_subscription_missing"]
	if !ok {
		t.Fatalf("rate limit override not loaded: %+v", cfg.rateLimits)
	}
	if override.Burst != 3 || override.RefillSeconds != 300 {
		t.Fatalf("override mismatch: %+v", override)
	}
}

func TestReveal_AnomalyWebhookURL(t *testing.T) {
	db := setupTestDB(t)
	resetStripeTestData(t, db)

	paymentService := NewPaymentSettingsService(db)
	stripeService := NewStripeServiceWithSettings(db, NewPlanService(db), paymentService)

	if _, err := db.Exec(`
		INSERT INTO payment_settings (id, publishable_key, secret_key, webhook_secret, anomaly_webhook_url, anomaly_webhook_enabled, anomaly_rate_limits, updated_at)
		VALUES (1, '', '', '', 'https://hooks.example.com/anomaly', TRUE, '{}'::jsonb, NOW())
		ON CONFLICT (id) DO UPDATE SET anomaly_webhook_url = EXCLUDED.anomaly_webhook_url, anomaly_webhook_enabled = EXCLUDED.anomaly_webhook_enabled
	`); err != nil {
		t.Fatal(err)
	}
	// GetSecretValue reads from stripeService runtime config which merges
	// payment_settings; ensure it is refreshed.
	if err := stripeService.RefreshConfig(context.Background()); err != nil {
		t.Fatal(err)
	}

	h := stripeSettingsConnectHandler{payment: paymentService, stripe: stripeService}
	response, err := h.RevealStripeSecret(context.Background(), connect.NewRequest(&landing_page_business_suite_v1.RevealStripeSecretRequest{Field: "anomaly_webhook_url"}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetField() != "anomaly_webhook_url" || response.Msg.GetValue() != "https://hooks.example.com/anomaly" {
		t.Fatalf("unexpected reveal response: %+v", response.Msg)
	}
}
