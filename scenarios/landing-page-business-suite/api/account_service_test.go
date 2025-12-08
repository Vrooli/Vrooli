package main

import (
	"testing"
	"time"

	landing_page_react_vite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
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
	if info.State != landing_page_react_vite_v1.SubscriptionState_SUBSCRIPTION_STATE_ACTIVE {
		t.Fatalf("expected status active, got %s", info.State)
	}

	if info.GetSubscriptionId() != subscriptionID {
		t.Fatalf("expected subscription id %s, got %s", subscriptionID, info.GetSubscriptionId())
	}

	_, err = db.Exec(`UPDATE subscriptions SET status = 'canceled', updated_at = NOW() WHERE subscription_id = $1`, subscriptionID)
	if err != nil {
		t.Fatalf("failed to update subscription: %v", err)
	}

	cached, err := accountService.GetSubscription(userEmail)
	if err != nil {
		t.Fatalf("cached GetSubscription failed: %v", err)
	}
	if cached.State != landing_page_react_vite_v1.SubscriptionState_SUBSCRIPTION_STATE_ACTIVE {
		t.Fatalf("expected cached status active, got %s", cached.State)
	}

	time.Sleep(accountService.cacheTTL + 10*time.Millisecond)

	refreshed, err := accountService.GetSubscription(userEmail)
	if err != nil {
		t.Fatalf("refresher GetSubscription failed: %v", err)
	}
	if refreshed.State != landing_page_react_vite_v1.SubscriptionState_SUBSCRIPTION_STATE_CANCELED {
		t.Fatalf("expected refreshed status canceled, got %s", refreshed.State)
	}
}

func TestAccountServiceSubscriptionMissingUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	accountService := NewAccountService(db, NewPlanService(db))

	status, err := accountService.GetSubscription("  ")
	if err != nil {
		t.Fatalf("unexpected error for missing user: %v", err)
	}

	if status.State != landing_page_react_vite_v1.SubscriptionState_SUBSCRIPTION_STATE_INACTIVE {
		t.Fatalf("expected inactive state for missing user, got %s", status.State)
	}
	if status.GetMessage() == "" || status.GetMessage() != "user not provided" {
		t.Fatalf("expected helpful message for missing user, got %q", status.GetMessage())
	}
	if status.GetUserIdentity() != "" {
		t.Fatalf("expected empty user identity, got %s", status.GetUserIdentity())
	}
}

func TestAccountServiceCreditsFallbacksWhenPlanUnavailable(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Force a plan lookup miss to exercise fallback defaults.
	planService := &PlanService{db: db, defaultBundle: "missing_bundle", displayEnv: "production"}
	accountService := NewAccountService(db, planService)

	credits, err := accountService.GetCredits("fallback@example.com")
	if err != nil {
		t.Fatalf("unexpected error in fallback path: %v", err)
	}

	if credits.DisplayCreditsLabel != "credits" {
		t.Fatalf("expected default label credits, got %s", credits.DisplayCreditsLabel)
	}
	if credits.DisplayCreditsMultiplier != 1.0 {
		t.Fatalf("expected default multiplier 1.0, got %f", credits.DisplayCreditsMultiplier)
	}
	if credits.Balance.BundleKey != "missing_bundle" {
		t.Fatalf("expected balance bundle key missing_bundle, got %s", credits.Balance.BundleKey)
	}
	if credits.Balance.CustomerEmail != "fallback@example.com" {
		t.Fatalf("expected customer email passthrough, got %s", credits.Balance.CustomerEmail)
	}
	if credits.Balance.UpdatedAt.AsTime().IsZero() {
		t.Fatal("expected updated_at to be populated")
	}
}
