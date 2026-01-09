package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestAccountServiceCreditsReflectBundleMetadata(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bundleKey := configureAccountBundleEnv(t, "credits_env")
	productID := upsertTestBundleProduct(
		t,
		db,
		bundleKey,
		"Credits Test Bundle",
		"prod_credits_test",
		"credits_env",
		2_000_000,
		0.0025,
		"credits",
	)
	defer cleanupBundleProductRecords(t, db, productID)

	insertBundlePrice(
		t,
		db,
		productID,
		"price_credits_test",
		"Credits Plan",
		"solo",
		"month",
		"usd",
		3000,
		false,
		"none",
		0,
		0,
		"credits_lookup",
		3_000_000,
		500_000,
		1,
		10,
		"none",
		"subscription",
		nil,
	)

	accountService := NewAccountService(db, NewPlanService(db))
	email := "credits-test@example.com"
	if _, err := db.Exec(`
		INSERT INTO credit_wallets (customer_email, balance_credits, bonus_credits, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (customer_email) DO UPDATE
		SET balance_credits = EXCLUDED.balance_credits,
		    bonus_credits = EXCLUDED.bonus_credits,
		    updated_at = NOW()
	`, email, 1_000_000, 200_000); err != nil {
		t.Fatalf("failed to seed credit wallet: %v", err)
	}

	// [REQ:SUB-CREDITS] Credits response surfaces bundle multipliers and labels
	credits, err := accountService.GetCredits(email)
	if err != nil {
		t.Fatalf("GetCredits failed: %v", err)
	}

	if credits.DisplayCreditsLabel != "credits" {
		t.Fatalf("expected display label credits, got %s", credits.DisplayCreditsLabel)
	}
	if credits.DisplayCreditsMultiplier != 0.0025 {
		t.Fatalf("expected multiplier 0.0025, got %f", credits.DisplayCreditsMultiplier)
	}
	if credits.Balance.BalanceCredits != 1_000_000 {
		t.Fatalf("unexpected balance credits: %d", credits.Balance.BalanceCredits)
	}
}

func TestAccountServiceCreditsFallbackWithoutPricing(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bundleKey := configureAccountBundleEnv(t, "missing_pricing_env")
	accountService := NewAccountService(db, NewPlanService(db))

	credits, err := accountService.GetCredits("no-wallet@example.com")
	if err != nil {
		t.Fatalf("GetCredits failed without pricing: %v", err)
	}

	if credits.DisplayCreditsLabel != "credits" {
		t.Fatalf("expected default credits label, got %s", credits.DisplayCreditsLabel)
	}
	if credits.DisplayCreditsMultiplier != 1.0 {
		t.Fatalf("expected default multiplier 1.0, got %f", credits.DisplayCreditsMultiplier)
	}
	if credits.Balance.BundleKey != bundleKey {
		t.Fatalf("expected bundle key fallback %s, got %s", bundleKey, credits.Balance.BundleKey)
	}
	if credits.Balance.BalanceCredits != 0 {
		t.Fatalf("expected zero balance for missing wallet, got %d", credits.Balance.BalanceCredits)
	}
	if credits.Balance.CustomerEmail != "no-wallet@example.com" {
		t.Fatalf("expected envelope to carry customer email, got %s", credits.Balance.CustomerEmail)
	}
	if ts := credits.Balance.GetUpdatedAt(); ts == nil || ts.AsTime().IsZero() {
		t.Fatal("expected updated timestamp set even when wallet missing")
	}
}

func TestAccountServiceEntitlementsIncludesFeatures(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bundleKey := configureAccountBundleEnv(t, "entitlements_env")
	productID := upsertTestBundleProduct(
		t,
		db,
		bundleKey,
		"Entitlements Test Bundle",
		"prod_entitlements_test",
		"entitlements_env",
		2_500_000,
		0.003,
		"entitlements",
	)
	defer cleanupBundleProductRecords(t, db, productID)

	insertBundlePrice(
		t,
		db,
		productID,
		"price_entitlements_plan",
		"Entitlements Plan",
		"studio",
		"month",
		"usd",
		9999,
		false,
		"none",
		0,
		0,
		"entitlements_lookup",
		5_000_000,
		0,
		3,
		20,
		"none",
		"subscription",
		map[string]interface{}{
			"features": []string{"Download gating", "Credits top-up"},
		},
	)

	email := "entitlements@example.com"
	if _, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_email, status, plan_tier, price_id, bundle_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT (subscription_id) DO UPDATE
		SET status = EXCLUDED.status, customer_email = EXCLUDED.customer_email, updated_at = NOW()
	`, "sub_entitlements_123", email, "active", "studio", "price_entitlements_plan", bundleKey); err != nil {
		t.Fatalf("failed to seed subscription: %v", err)
	}

	// Seed credit wallet so entitlements can return some credit info
	if _, err := db.Exec(`
		INSERT INTO credit_wallets (customer_email, balance_credits, bonus_credits, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (customer_email) DO UPDATE
		SET balance_credits = EXCLUDED.balance_credits, bonus_credits = EXCLUDED.bonus_credits, updated_at = $4
	`, email, 2_000_000, 300_000, time.Now()); err != nil {
		t.Fatalf("failed to insert credit wallet: %v", err)
	}

	accountService := NewAccountService(db, NewPlanService(db))
	// [REQ:SUB-ENTITLEMENTS] Entitlements payload surfaces feature flags and credits
	entitlements, err := accountService.GetEntitlements(email)
	if err != nil {
		t.Fatalf("GetEntitlements failed: %v", err)
	}

	if entitlements.Status != "active" {
		t.Fatalf("expected active entitlements, got %s", entitlements.Status)
	}
	if entitlements.PlanTier != "studio" {
		t.Fatalf("expected plan tier studio, got %s", entitlements.PlanTier)
	}
	if entitlements.PriceID != "price_entitlements_plan" {
		t.Fatalf("expected price passthrough price_entitlements_plan, got %s", entitlements.PriceID)
	}
	if entitlements.Credits == nil {
		t.Fatal("expected credits populated")
	}
	if entitlements.Credits.BalanceCredits != 2_000_000 {
		t.Fatalf("expected credit balance to match wallet seed, got %d", entitlements.Credits.BalanceCredits)
	}
	expectedFeatures := []string{"Download gating", "Credits top-up"}
	if !reflect.DeepEqual(entitlements.Features, expectedFeatures) {
		t.Fatalf("expected feature flags %v, got %v", expectedFeatures, entitlements.Features)
	}
	if entitlements.Subscription == nil || entitlements.Subscription.GetStripePriceId() != "price_entitlements_plan" {
		t.Fatalf("expected subscription payload to carry price id, got %+v", entitlements.Subscription)
	}
}

func TestAccountServiceEntitlements_NoSubscriptionDefaults(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	accountService := NewAccountService(db, NewPlanService(db))

	entitlements, err := accountService.GetEntitlements("nosub@example.com")
	if err != nil {
		t.Fatalf("GetEntitlements for missing user failed: %v", err)
	}

	if entitlements.Status != "inactive" {
		t.Fatalf("expected inactive status when subscription missing, got %s", entitlements.Status)
	}
	if entitlements.PlanTier != "" {
		t.Fatalf("expected empty plan tier for missing subscription, got %s", entitlements.PlanTier)
	}
	if entitlements.Subscription == nil {
		t.Fatal("expected subscription payload to be populated for missing user")
	}
	if entitlements.Credits == nil || entitlements.Credits.CustomerEmail != "nosub@example.com" {
		t.Fatalf("expected credit envelope for user, got %+v", entitlements.Credits)
	}
	if len(entitlements.Features) != 0 {
		t.Fatalf("expected no feature flags without plan metadata, got %v", entitlements.Features)
	}
}

func configureAccountBundleEnv(t *testing.T, env string) string {
	t.Helper()

	replacer := strings.NewReplacer("/", "_", ".", "_")
	bundleKey := fmt.Sprintf("bundle_%s", replacer.Replace(strings.ToLower(t.Name())))
	prevKey := os.Getenv("BUNDLE_KEY")
	prevEnv := os.Getenv("BUNDLE_ENVIRONMENT")

	if err := os.Setenv("BUNDLE_KEY", bundleKey); err != nil {
		t.Fatalf("failed to set BUNDLE_KEY: %v", err)
	}
	if err := os.Setenv("BUNDLE_ENVIRONMENT", env); err != nil {
		t.Fatalf("failed to set BUNDLE_ENVIRONMENT: %v", err)
	}

	t.Cleanup(func() {
		setEnvOrClear("BUNDLE_KEY", prevKey)
		setEnvOrClear("BUNDLE_ENVIRONMENT", prevEnv)
	})

	return bundleKey
}

func TestAccountServiceEntitlements_InfersPlanTierFromPrice(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bundleKey := configureAccountBundleEnv(t, "infer_env")
	productID := upsertTestBundleProduct(
		t,
		db,
		bundleKey,
		"Infer Plan Tier",
		"prod_infer_tier",
		"infer_env",
		1_000_000,
		0.001,
		"infer",
	)
	defer cleanupBundleProductRecords(t, db, productID)

	insertBundlePrice(
		t,
		db,
		productID,
		"price_infer_plan",
		"Infer Plan",
		"studio",
		"month",
		"usd",
		1299,
		false,
		"none",
		0,
		0,
		"infer_lookup",
		1_500_000,
		0,
		1,
		10,
		"none",
		"subscription",
		nil,
	)

	email := "infer-plan@example.com"
	if _, err := db.Exec(`
		INSERT INTO subscriptions (subscription_id, customer_email, status, plan_tier, price_id, bundle_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`, "sub_infer_123", email, "trialing", "", "price_infer_plan", bundleKey); err != nil {
		t.Fatalf("failed to seed subscription: %v", err)
	}

	accountService := NewAccountService(db, NewPlanService(db))
	entitlements, err := accountService.GetEntitlements(email)
	if err != nil {
		t.Fatalf("GetEntitlements failed: %v", err)
	}

	if entitlements.PlanTier != "studio" {
		t.Fatalf("expected plan tier to be inferred from price metadata, got %s", entitlements.PlanTier)
	}
	if entitlements.Status != "trialing" {
		t.Fatalf("expected legacy status label trialing, got %s", entitlements.Status)
	}
	if entitlements.Subscription == nil || entitlements.Subscription.GetPlanTier() != "studio" {
		t.Fatalf("expected subscription proto to be updated with inferred plan tier, got %+v", entitlements.Subscription)
	}
}

func TestAccountServiceCredits_EmptyUserUsesDefaults(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bundleKey := configureAccountBundleEnv(t, "empty_user_env")
	accountService := NewAccountService(db, NewPlanService(db))

	credits, err := accountService.GetCredits("   ")
	if err != nil {
		t.Fatalf("GetCredits for empty user failed: %v", err)
	}

	if credits.DisplayCreditsLabel != "credits" {
		t.Fatalf("expected default display label credits, got %s", credits.DisplayCreditsLabel)
	}
	if credits.DisplayCreditsMultiplier != 1.0 {
		t.Fatalf("expected default multiplier 1.0, got %f", credits.DisplayCreditsMultiplier)
	}
	if credits.Balance.BundleKey != bundleKey {
		t.Fatalf("expected bundle key to match configured env %s, got %s", bundleKey, credits.Balance.BundleKey)
	}
	if credits.Balance.CustomerEmail != "" {
		t.Fatalf("expected empty customer email passthrough, got %s", credits.Balance.CustomerEmail)
	}
	if credits.Balance.BalanceCredits != 0 {
		t.Fatalf("expected zero balance for empty user, got %d", credits.Balance.BalanceCredits)
	}
	if ts := credits.Balance.GetUpdatedAt(); ts == nil || ts.AsTime().IsZero() {
		t.Fatal("expected updated_at timestamp populated for empty user credits response")
	}
}
