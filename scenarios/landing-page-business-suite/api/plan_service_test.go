package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	landing_page_react_vite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
)

func TestPlanServicePricingOverview(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bundleKey := configureTestBundleEnv(t, "pricing_env")
	productID := upsertTestBundleProduct(
		t,
		db,
		bundleKey,
		"Pricing Test Bundle",
		"prod_pricing_test",
		"pricing_env",
		1_000_000,
		0.01,
		"credits",
	)
	defer cleanupBundleProductRecords(t, db, productID)

	insertBundlePrice(
		t,
		db,
		productID,
		"price_pricing_monthly",
		"Pricing Monthly",
		"pro",
		"month",
		"usd",
		4999,
		true,
		"flat_amount",
		100,
		1,
		"monthly_intro_key",
		5_000_000,
		0,
		1,
		30,
		"none",
		"subscription",
		map[string]interface{}{
			"features": []string{"Fast coupling", "Priority support"},
		},
	)

	insertBundlePrice(
		t,
		db,
		productID,
		"price_pricing_yearly",
		"Pricing Yearly",
		"pro",
		"year",
		"usd",
		55999,
		false,
		"none",
		0,
		0,
		"yearly_lookup_key",
		60_000_000,
		10_000_000,
		2,
		10,
		"yearly_bonus",
		"subscription",
		map[string]interface{}{
			"features": []string{"Annual loyalty", "Bonus credits"},
		},
	)

	planService := NewPlanService(db)
	overview, err := planService.GetPricingOverview()
	if err != nil {
		t.Fatalf("GetPricingOverview failed: %v", err)
	}

	if overview.Bundle.BundleKey != bundleKey {
		t.Fatalf("expected bundle key %s, got %s", bundleKey, overview.Bundle.BundleKey)
	}

	if len(overview.Monthly) != 1 {
		t.Fatalf("expected 1 monthly option, got %d", len(overview.Monthly))
	}
	if len(overview.Yearly) != 1 {
		t.Fatalf("expected 1 yearly option, got %d", len(overview.Yearly))
	}

	monthly := overview.Monthly[0]
	if monthly.StripePriceId != "price_pricing_monthly" {
		t.Fatalf("unexpected monthly price id %s", monthly.StripePriceId)
	}
	if !monthly.IntroEnabled {
		t.Fatal("expected monthly intro to be enabled")
	}

	yearly := overview.Yearly[0]
	if yearly.BillingInterval != landing_page_react_vite_v1.BillingInterval_BILLING_INTERVAL_YEAR {
		t.Fatalf("expected yearly billing interval, got %v", yearly.BillingInterval)
	}
	if yearly.IntroEnabled {
		t.Fatal("expected yearly intro disabled")
	}
}

func TestPlanServiceGetPlanByPriceID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bundleKey := configureTestBundleEnv(t, "pricing_env")
	productID := upsertTestBundleProduct(
		t,
		db,
		bundleKey,
		"Pricing Test Bundle",
		"prod_pricing_test",
		"pricing_env",
		1_000_000,
		0.01,
		"credits",
	)
	defer cleanupBundleProductRecords(t, db, productID)

	insertBundlePrice(
		t,
		db,
		productID,
		"price_lookup_test",
		"Lookup Plan",
		"pro",
		"month",
		"usd",
		9999,
		true,
		"flat_amount",
		100,
		1,
		"lookup_key",
		10_000_000,
		0,
		5,
		40,
		"none",
		"subscription",
		map[string]interface{}{
			"features": []string{"Lookup feature"},
		},
	)

	planService := NewPlanService(db)
	option, err := planService.GetPlanByPriceID("price_lookup_test")
	if err != nil {
		t.Fatalf("GetPlanByPriceID failed: %v", err)
	}

	if option.PlanName != "Lookup Plan" {
		t.Fatalf("expected plan named Lookup Plan, got %s", option.PlanName)
	}
	if option.Metadata == nil {
		t.Fatal("expected metadata to be present")
	}
	if _, ok := option.Metadata["features"]; !ok {
		t.Fatal("expected features metadata")
	}
}

func TestPlanServiceGetPricingOverviewOrdersAndFiltersDisabled(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bundleKey := configureTestBundleEnv(t, "production")
	productID := upsertTestBundleProduct(
		t,
		db,
		bundleKey,
		"Ordering Bundle",
		"prod_ordering",
		"production",
		1_000_000,
		0.01,
		"credits",
	)
	defer cleanupBundleProductRecords(t, db, productID)

	insertBundlePrice(t, db, productID, "price_weight_10", "Weighted", "pro", "month", "usd", 1000, false, "none", 0, 0, "", 0, 0, 5, 10, "none", "subscription", map[string]interface{}{})
	insertBundlePrice(t, db, productID, "price_weight_5_rank1", "Rank 1", "pro", "month", "usd", 2000, false, "none", 0, 0, "", 0, 0, 1, 5, "none", "subscription", map[string]interface{}{})
	insertBundlePrice(t, db, productID, "price_weight_5_rank2", "Rank 2", "pro", "month", "usd", 3000, false, "none", 0, 0, "", 0, 0, 2, 5, "none", "subscription", map[string]interface{}{})
	if _, err := db.Exec(`UPDATE bundle_prices SET display_enabled = false WHERE stripe_price_id = $1`, "price_weight_5_rank2"); err != nil {
		t.Fatalf("failed to disable price: %v", err)
	}

	service := NewPlanService(db)
	overview, err := service.GetPricingOverview()
	if err != nil {
		t.Fatalf("GetPricingOverview failed: %v", err)
	}

	if got := len(overview.Monthly); got != 2 {
		t.Fatalf("expected 2 visible monthly options, got %d", got)
	}
	if overview.Monthly[0].GetStripePriceId() != "price_weight_10" {
		t.Fatalf("expected highest weight first, got %s", overview.Monthly[0].GetStripePriceId())
	}
	if overview.Monthly[1].GetStripePriceId() != "price_weight_5_rank1" {
		t.Fatalf("expected rank tie-breaker next, got %s", overview.Monthly[1].GetStripePriceId())
	}
}

func TestPlanServiceGetPlanByPriceIDErrorsForEmptyOrMissing(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewPlanService(db)

	if _, err := service.GetPlanByPriceID(""); err == nil {
		t.Fatal("expected error when price id missing")
	}

	_, err := service.GetPlanByPriceID("price_missing")
	if err == nil {
		t.Fatal("expected error for missing price record")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found message, got %v", err)
	}
}

func configureTestBundleEnv(t *testing.T, env string) string {
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

func setEnvOrClear(key, value string) {
	if value == "" {
		_ = os.Unsetenv(key)
		return
	}
	_ = os.Setenv(key, value)
}
