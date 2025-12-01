package main

import (
	"context"
	"testing"
)

func TestPaymentSettingsServiceUpsert(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewPaymentSettingsService(db)
	ctx := context.Background()

	record, err := service.SaveStripeSettings(ctx, StripeSettingsInput{
		PublishableKey: strPtr("pk_live_123"),
		SecretKey:      strPtr("sk_live_123"),
		WebhookSecret:  strPtr("whsec_live_456"),
		DashboardURL:   strPtr("https://dashboard.stripe.com/test"),
	})
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	if record.PublishableKey != "pk_live_123" {
		t.Fatalf("unexpected publishable key %s", record.PublishableKey)
	}

	reloaded, err := service.GetStripeSettings(ctx)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if reloaded == nil || reloaded.SecretKey != "sk_live_123" {
		t.Fatalf("expected secret key to persist")
	}

	_, err = service.SaveStripeSettings(ctx, StripeSettingsInput{
		DashboardURL: strPtr("https://dashboard.stripe.com/alt"),
	})
	if err != nil {
		t.Fatalf("partial save failed: %v", err)
	}

	finalRecord, err := service.GetStripeSettings(ctx)
	if err != nil {
		t.Fatalf("final reload failed: %v", err)
	}
	if finalRecord.DashboardURL != "https://dashboard.stripe.com/alt" {
		t.Fatalf("dashboard url not updated")
	}
	if finalRecord.PublishableKey != "pk_live_123" {
		t.Fatalf("publishable key should remain unchanged")
	}
}

func strPtr(value string) *string {
	return &value
}
