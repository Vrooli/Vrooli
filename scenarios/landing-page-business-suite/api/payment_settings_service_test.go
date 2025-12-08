package main

import (
	"context"
	"strings"
	"testing"
)

func TestPaymentSettingsServiceUpsert(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	t.Cleanup(func() {
		db.Exec("DELETE FROM payment_settings")
	})

	service := NewPaymentSettingsService(db)
	ctx := context.Background()

	record, err := service.SaveStripeSettings(ctx, StripeSettingsInput{
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

	_, err = service.SaveStripeSettings(ctx, StripeSettingsInput{
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
	defer db.Close()
	t.Cleanup(func() {
		db.Exec("DELETE FROM payment_settings")
	})

	service := NewPaymentSettingsService(db)
	ctx := context.Background()

	trimmed, err := service.SaveStripeSettings(ctx, StripeSettingsInput{
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
	defer db.Close()
	db.Exec("DELETE FROM payment_settings")

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

func ptrStripe(value string) *string {
	return &value
}
