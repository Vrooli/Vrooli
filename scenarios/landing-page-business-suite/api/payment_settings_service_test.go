package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"landing-page-business-suite-api/internal/commerce"
)

func TestPaymentSettingsServiceUpsert(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	t.Cleanup(func() {
		if _, err := db.Exec("DELETE FROM payment_settings"); err != nil {
			t.Fatalf("failed to clean payment_settings: %v", err)
		}
	})

	service := NewPaymentSettingsService(db)
	ctx := context.Background()

	record, err := service.SaveStripeSettings(ctx, commerce.StripeSettingsInput{
		PublishableKey: ptrStripe("pk_live_123"),
		SecretKey:      ptrStripe("sk_live_123"),
		WebhookSecret:  ptrStripe("whsec_live_456"),
		DashboardURL:   ptrStripe("https://dashboard.stripe.com/test"),
	})
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	if record.GetPublishableKey() != "pk_live_123" {
		t.Fatalf("unexpected publishable key %s", record.GetPublishableKey())
	}

	reloaded, err := service.GetStripeSettings(ctx)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if reloaded == nil || reloaded.GetSecretKey() != "sk_live_123" {
		t.Fatalf("expected secret key to persist")
	}

	_, err = service.SaveStripeSettings(ctx, commerce.StripeSettingsInput{
		DashboardURL: ptrStripe("https://dashboard.stripe.com/alt"),
	})
	if err != nil {
		t.Fatalf("partial save failed: %v", err)
	}

	finalRecord, err := service.GetStripeSettings(ctx)
	if err != nil {
		t.Fatalf("final reload failed: %v", err)
	}
	if finalRecord.GetDashboardUrl() != "https://dashboard.stripe.com/alt" {
		t.Fatalf("dashboard url not updated")
	}
	if finalRecord.GetPublishableKey() != "pk_live_123" {
		t.Fatalf("publishable key should remain unchanged")
	}
}

func TestPaymentSettingsService_TrimsWhitespace(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	t.Cleanup(func() {
		if _, err := db.Exec("DELETE FROM payment_settings"); err != nil {
			t.Fatalf("failed to clean payment_settings: %v", err)
		}
	})

	service := NewPaymentSettingsService(db)
	ctx := context.Background()

	trimmed, err := service.SaveStripeSettings(ctx, commerce.StripeSettingsInput{
		PublishableKey: ptrStripe("  pk_trim  "),
		SecretKey:      ptrStripe("sk_trim  "),
		WebhookSecret:  ptrStripe("\twhsec_trim\n"),
		DashboardURL:   ptrStripe(" https://dashboard.example.com/test "),
	})
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	if trimmed.GetPublishableKey() != "pk_trim" {
		t.Fatalf("expected publishable key trimmed, got %q", trimmed.GetPublishableKey())
	}
	if trimmed.GetSecretKey() != "sk_trim" {
		t.Fatalf("expected secret key trimmed, got %q", trimmed.GetSecretKey())
	}
	if trimmed.GetWebhookSecret() != "whsec_trim" {
		t.Fatalf("expected webhook secret trimmed, got %q", trimmed.GetWebhookSecret())
	}
	if got := trimmed.GetDashboardUrl(); got != strings.TrimSpace(" https://dashboard.example.com/test ") {
		t.Fatalf("expected dashboard url trimmed, got %q", got)
	}
}

func TestPaymentSettingsServiceReturnsNilWhenNoRecord(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec("DELETE FROM payment_settings"); err != nil {
		t.Fatalf("failed to clean payment_settings: %v", err)
	}

	service := NewPaymentSettingsService(db)
	ctx := context.Background()

	record, err := service.GetStripeSettings(ctx)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if record != nil {
		t.Fatalf("expected nil record when no settings present, got %+v", record)
	}
}

func TestPaymentSettingsServicePropagatesCredentialProviderFailure(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec("DELETE FROM payment_settings"); err != nil {
		t.Fatalf("failed to clean payment_settings: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO payment_settings (id, dashboard_url) VALUES (1, 'https://dashboard.example.test') ON CONFLICT (id) DO UPDATE SET dashboard_url = EXCLUDED.dashboard_url`); err != nil {
		t.Fatalf("failed to seed payment_settings: %v", err)
	}

	providerErr := errors.New("credential provider unavailable")
	service := commerce.NewPaymentSettingsServiceWithCredentials(db, func(context.Context, string) (string, error) {
		return "", providerErr
	}, nil)

	_, err := service.GetStripeSettings(context.Background())
	if !errors.Is(err, providerErr) {
		t.Fatalf("expected provider error, got %v", err)
	}
}

func TestPaymentSettingsServiceTreatsUnconfiguredCredentialAsOptional(t *testing.T) {
	db := setupTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec("DELETE FROM payment_settings"); err != nil {
		t.Fatalf("failed to clean payment_settings: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO payment_settings (id, dashboard_url) VALUES (1, 'https://dashboard.example.test')`); err != nil {
		t.Fatalf("failed to seed payment_settings: %v", err)
	}

	service := commerce.NewPaymentSettingsServiceWithCredentials(db, func(context.Context, string) (string, error) {
		return "", credentialauthority.ErrUnconfigured
	}, nil)

	record, err := service.GetStripeSettings(context.Background())
	if err != nil {
		t.Fatalf("unconfigured credentials should preserve degraded startup: %v", err)
	}
	if record == nil || record.GetDashboardUrl() != "https://dashboard.example.test" {
		t.Fatalf("expected non-secret settings to remain available, got %+v", record)
	}
}

func TestPaymentSettingsServiceUsesStrictTestCredentialNamespace(t *testing.T) {
	t.Setenv("STRIPE_MODE", "test")
	db := setupTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`INSERT INTO payment_settings (id, dashboard_url) VALUES (1, 'https://dashboard.example.test') ON CONFLICT (id) DO UPDATE SET dashboard_url = EXCLUDED.dashboard_url`); err != nil {
		t.Fatalf("failed to seed payment_settings: %v", err)
	}

	values := map[string]string{
		"stripe-test-publishable-key": "pk_test_public",
		"stripe-test-secret-key":      "rk_test_secret",
		"stripe-test-webhook-secret":  "whsec_test_secret",
	}
	requested := make([]string, 0, len(values))
	service := commerce.NewPaymentSettingsServiceWithCredentials(db, func(_ context.Context, field string) (string, error) {
		requested = append(requested, field)
		value, ok := values[field]
		if !ok {
			return "", credentialauthority.ErrUnconfigured
		}
		return value, nil
	}, nil)

	record, err := service.GetStripeSettings(context.Background())
	if err != nil {
		t.Fatalf("get test settings failed: %v", err)
	}
	if record.GetPublishableKey() != values["stripe-test-publishable-key"] || record.GetSecretKey() != values["stripe-test-secret-key"] || record.GetWebhookSecret() != values["stripe-test-webhook-secret"] {
		t.Fatalf("test namespace values were not loaded: %+v", record)
	}
	if strings.Contains(strings.Join(requested, ","), "stripe-publishable-key") || strings.Contains(strings.Join(requested, ","), "stripe-secret-key") || strings.Contains(strings.Join(requested, ","), "stripe-webhook-secret") {
		t.Fatalf("test mode attempted legacy/live credential fallback: %v", requested)
	}
}

func TestPaymentSettingsServiceDefaultsToLiveMigrationFallback(t *testing.T) {
	t.Setenv("STRIPE_MODE", "")
	db := setupTestDB(t)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`INSERT INTO payment_settings (id, dashboard_url) VALUES (1, 'https://dashboard.example.test') ON CONFLICT (id) DO UPDATE SET dashboard_url = EXCLUDED.dashboard_url`); err != nil {
		t.Fatalf("failed to seed payment_settings: %v", err)
	}

	values := map[string]string{
		"stripe-publishable-key": "pk_live_legacy",
		"stripe-secret-key":      "rk_live_legacy",
		"stripe-webhook-secret":  "whsec_legacy",
	}
	requested := make([]string, 0, 6)
	service := commerce.NewPaymentSettingsServiceWithCredentials(db, func(_ context.Context, field string) (string, error) {
		requested = append(requested, field)
		if strings.HasPrefix(field, "stripe-live-") {
			return "", credentialauthority.ErrUnconfigured
		}
		value, ok := values[field]
		if !ok {
			return "", credentialauthority.ErrUnconfigured
		}
		return value, nil
	}, nil)

	record, err := service.GetStripeSettings(context.Background())
	if err != nil {
		t.Fatalf("get live settings failed: %v", err)
	}
	if record.GetPublishableKey() != values["stripe-publishable-key"] || record.GetSecretKey() != values["stripe-secret-key"] || record.GetWebhookSecret() != values["stripe-webhook-secret"] {
		t.Fatalf("legacy live fallback values were not loaded: %+v", record)
	}
	if len(requested) != 6 {
		t.Fatalf("requested fields = %v, want explicit live fields followed by legacy fallback fields", requested)
	}
}

func ptrStripe(value string) *string {
	return &value
}

func TestValidateStripeKeyModePairRejectsMixedMode(t *testing.T) {
	if err := commerce.ValidateStripeKeyModePair("pk_test_public", "rk_live_secret"); err == nil {
		t.Fatal("expected mixed Stripe modes to be rejected")
	}
}

func TestValidateStripeKeyModePairAcceptsMatchingAndOpaqueFixtures(t *testing.T) {
	for _, pair := range [][2]string{{"pk_test_public", "sk_test_secret"}, {"pk_live_public", "rk_live_secret"}, {"pk_fixture", "rk_fixture"}} {
		if err := commerce.ValidateStripeKeyModePair(pair[0], pair[1]); err != nil {
			t.Fatalf("ValidateStripeKeyModePair(%q, %q) = %v", pair[0], pair[1], err)
		}
	}
}
