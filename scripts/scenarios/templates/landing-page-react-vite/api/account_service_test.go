package main

import (
	"testing"
	"time"
)

func TestAccountServiceSubscriptionCache(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	planService := NewPlanService(db)
	accountService := NewAccountService(db, planService)
	accountService.cacheTTL = 40 * time.Millisecond

	const userEmail = "cache-test@example.com"
	const subscriptionID = "sub-cache-test"

	defer db.Exec("DELETE FROM subscriptions WHERE subscription_id = $1", subscriptionID)

	_, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_email, status, plan_tier, price_id, bundle_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT (subscription_id) DO UPDATE
		SET status = EXCLUDED.status, updated_at = NOW()
	`, subscriptionID, userEmail, "active", "solo", "price_solo_monthly", accountService.bundleKey)
	if err != nil {
		t.Fatalf("failed to seed subscription: %v", err)
	}

	info, err := accountService.GetSubscription(userEmail)
	if err != nil {
		t.Fatalf("initial GetSubscription failed: %v", err)
	}
	if info.Status != "active" {
		t.Fatalf("expected status active, got %s", info.Status)
	}

	if info.SubscriptionID != subscriptionID {
		t.Fatalf("expected subscription id %s, got %s", subscriptionID, info.SubscriptionID)
	}

	_, err = db.Exec(`UPDATE subscriptions SET status = 'canceled', updated_at = NOW() WHERE subscription_id = $1`, subscriptionID)
	if err != nil {
		t.Fatalf("failed to update subscription: %v", err)
	}

	cached, err := accountService.GetSubscription(userEmail)
	if err != nil {
		t.Fatalf("cached GetSubscription failed: %v", err)
	}
	if cached.Status != "active" {
		t.Fatalf("expected cached status active, got %s", cached.Status)
	}

	time.Sleep(accountService.cacheTTL + 10*time.Millisecond)

	refreshed, err := accountService.GetSubscription(userEmail)
	if err != nil {
		t.Fatalf("refresher GetSubscription failed: %v", err)
	}
	if refreshed.Status != "canceled" {
		t.Fatalf("expected refreshed status canceled, got %s", refreshed.Status)
	}
}
