package main

import (
	"context"
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
		DashboardURL: ptrStripe("https://dashboard.stripe.com/alt"),
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

func ptrStripe(value string) *string {
	return &value
}
