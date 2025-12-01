package main

import (
	"fmt"
	"os"
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
	if credits.BalanceCredits != 1_000_000 {
		t.Fatalf("unexpected balance credits: %d", credits.BalanceCredits)
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
	if entitlements.Credits == nil {
		t.Fatal("expected credits populated")
	}
	if len(entitlements.Features) == 0 {
		t.Fatal("expected feature flags populated")
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
