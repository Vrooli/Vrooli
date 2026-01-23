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

// ============================================================================
// GetPricingOverview Tests
// ============================================================================

func TestPlanService_GetPricingOverview_FreeTierAlwaysIncluded(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bundleKey := configureTestBundleEnv(t, "production")
	productID := upsertTestBundleProduct(t, db, bundleKey, "Free Tier Bundle", "prod_free_test", "production", 1_000_000, 0.01, "credits")
	defer cleanupBundleProductRecords(t, db, productID)

	// Insert only a free tier price
	insertBundlePrice(t, db, productID, "price_free", "Free Plan", "free", "month", "usd", 0, true, "none", 0, 0, "", 0, 0, 1, 100, "none", "subscription", map[string]interface{}{})

	service := NewPlanService(db)
	overview, err := service.GetPricingOverview()
	if err != nil {
		t.Fatalf("GetPricingOverview failed: %v", err)
	}

	if len(overview.Monthly) != 1 {
		t.Fatalf("expected 1 monthly option (free tier), got %d", len(overview.Monthly))
	}
	if overview.Monthly[0].PlanTier != "free" {
		t.Errorf("expected free tier, got %s", overview.Monthly[0].PlanTier)
	}
}

func TestPlanService_GetPricingOverview_BundleNotFound_ReturnsError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Configure env to point to non-existent bundle
	_ = os.Setenv("BUNDLE_KEY", "nonexistent_bundle_key_xyz")
	_ = os.Setenv("BUNDLE_ENVIRONMENT", "production")
	t.Cleanup(func() {
		_ = os.Unsetenv("BUNDLE_KEY")
		_ = os.Unsetenv("BUNDLE_ENVIRONMENT")
	})

	service := NewPlanService(db)
	_, err := service.GetPricingOverview()
	if err == nil {
		t.Error("Expected error for non-existent bundle, got nil")
	}
}

func TestPlanService_GetPricingOverview_NoPrices_ReturnsEmptySlices(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bundleKey := configureTestBundleEnv(t, "production")
	productID := upsertTestBundleProduct(t, db, bundleKey, "Empty Bundle", "prod_empty", "production", 1_000_000, 0.01, "credits")
	defer cleanupBundleProductRecords(t, db, productID)

	// No prices inserted

	service := NewPlanService(db)
	overview, err := service.GetPricingOverview()
	if err != nil {
		t.Fatalf("GetPricingOverview failed: %v", err)
	}

	if overview.Monthly == nil {
		t.Error("expected non-nil Monthly slice")
	}
	if overview.Yearly == nil {
		t.Error("expected non-nil Yearly slice")
	}
	if len(overview.Monthly) != 0 {
		t.Errorf("expected 0 monthly options, got %d", len(overview.Monthly))
	}
	if len(overview.Yearly) != 0 {
		t.Errorf("expected 0 yearly options, got %d", len(overview.Yearly))
	}
}

// ============================================================================
// GetBundleProduct Tests
// ============================================================================

func TestPlanService_GetBundleProduct_ReturnsProductMetadata(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bundleKey := configureTestBundleEnv(t, "production")
	productID := upsertTestBundleProduct(t, db, bundleKey, "Test Product", "prod_test_metadata", "production", 2_000_000, 0.02, "tokens")
	defer cleanupBundleProductRecords(t, db, productID)

	service := NewPlanService(db)
	product, err := service.GetBundleProduct()
	if err != nil {
		t.Fatalf("GetBundleProduct failed: %v", err)
	}

	if product.BundleKey != bundleKey {
		t.Errorf("expected bundle key %s, got %s", bundleKey, product.BundleKey)
	}
	if product.Name != "Test Product" {
		t.Errorf("expected name 'Test Product', got %s", product.Name)
	}
	if product.CreditsPerUsd != 2_000_000 {
		t.Errorf("expected credits_per_usd 2000000, got %d", product.CreditsPerUsd)
	}
}

func TestPlanService_GetBundleProduct_NotFound_ReturnsError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_ = os.Setenv("BUNDLE_KEY", "nonexistent_product_bundle")
	_ = os.Setenv("BUNDLE_ENVIRONMENT", "production")
	t.Cleanup(func() {
		_ = os.Unsetenv("BUNDLE_KEY")
		_ = os.Unsetenv("BUNDLE_ENVIRONMENT")
	})

	service := NewPlanService(db)
	_, err := service.GetBundleProduct()
	if err == nil {
		t.Error("Expected error for non-existent bundle product, got nil")
	}
}

// ============================================================================
// ListBundleCatalog Tests
// ============================================================================

func TestPlanService_ListBundleCatalog_ReturnsBundlesWithPrices(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bundleKey := configureTestBundleEnv(t, "catalog_env")
	productID := upsertTestBundleProduct(t, db, bundleKey, "Catalog Bundle", "prod_catalog", "catalog_env", 1_000_000, 0.01, "credits")
	defer cleanupBundleProductRecords(t, db, productID)

	insertBundlePrice(t, db, productID, "price_catalog_1", "Catalog Plan 1", "pro", "month", "usd", 999, true, "none", 0, 0, "", 0, 0, 1, 10, "none", "subscription", map[string]interface{}{})
	insertBundlePrice(t, db, productID, "price_catalog_2", "Catalog Plan 2", "business", "year", "usd", 9999, true, "none", 0, 0, "", 0, 0, 2, 5, "none", "subscription", map[string]interface{}{})

	service := NewPlanServiceWithOptions(PlanServiceOptions{
		DB:            db,
		DefaultBundle: bundleKey,
		DisplayEnv:    "catalog_env",
	})

	entries, err := service.ListBundleCatalog(t.Context())
	if err != nil {
		t.Fatalf("ListBundleCatalog failed: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("expected at least 1 catalog entry")
	}

	found := false
	for _, entry := range entries {
		if entry.Bundle.BundleKey == bundleKey {
			found = true
			if len(entry.Prices) != 2 {
				t.Errorf("expected 2 prices for bundle, got %d", len(entry.Prices))
			}
		}
	}
	if !found {
		t.Errorf("expected to find bundle %s in catalog", bundleKey)
	}
}

func TestPlanService_ListBundleCatalog_FiltersByEnvironment(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create bundles in different environments
	bundleKey1 := "catalog_staging_bundle"
	bundleKey2 := "catalog_production_bundle"

	productID1 := upsertTestBundleProduct(t, db, bundleKey1, "Staging Bundle", "prod_staging", "staging", 1_000_000, 0.01, "credits")
	defer cleanupBundleProductRecords(t, db, productID1)

	productID2 := upsertTestBundleProduct(t, db, bundleKey2, "Production Bundle", "prod_production", "production", 1_000_000, 0.01, "credits")
	defer cleanupBundleProductRecords(t, db, productID2)

	// Service configured for staging environment
	service := NewPlanServiceWithOptions(PlanServiceOptions{
		DB:            db,
		DefaultBundle: bundleKey1,
		DisplayEnv:    "staging",
	})

	entries, err := service.ListBundleCatalog(t.Context())
	if err != nil {
		t.Fatalf("ListBundleCatalog failed: %v", err)
	}

	// Should only see staging bundle
	for _, entry := range entries {
		if entry.Bundle.Environment != "staging" {
			t.Errorf("expected only staging bundles, got environment %s", entry.Bundle.Environment)
		}
	}
}

func TestPlanService_ListBundleCatalog_EmptyCatalog_ReturnsEmptySlice(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewPlanServiceWithOptions(PlanServiceOptions{
		DB:            db,
		DefaultBundle: "nonexistent",
		DisplayEnv:    "nonexistent_env",
	})

	entries, err := service.ListBundleCatalog(t.Context())
	if err != nil {
		t.Fatalf("ListBundleCatalog failed: %v", err)
	}

	// Empty catalog returns nil or empty slice - both are acceptable
	if len(entries) != 0 {
		t.Errorf("expected empty catalog, got %d entries", len(entries))
	}
}

// ============================================================================
// UpdateBundlePrice Tests
// ============================================================================

func TestPlanService_UpdateBundlePrice_ToggleDisplayEnabled(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bundleKey := configureTestBundleEnv(t, "production")
	productID := upsertTestBundleProduct(t, db, bundleKey, "Update Bundle", "prod_update", "production", 1_000_000, 0.01, "credits")
	defer cleanupBundleProductRecords(t, db, productID)

	insertBundlePrice(t, db, productID, "price_toggle", "Toggle Plan", "pro", "month", "usd", 999, true, "none", 0, 0, "", 0, 0, 1, 10, "none", "subscription", map[string]interface{}{})

	service := NewPlanService(db)

	// Disable the price
	disabled := false
	updated, err := service.UpdateBundlePrice(t.Context(), bundleKey, "price_toggle", UpdateBundlePriceInput{
		DisplayEnabled: &disabled,
	})
	if err != nil {
		t.Fatalf("UpdateBundlePrice failed: %v", err)
	}

	if updated.DisplayEnabled {
		t.Error("expected DisplayEnabled to be false after update")
	}
}

func TestPlanService_UpdateBundlePrice_ChangeDisplayWeight(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bundleKey := configureTestBundleEnv(t, "production")
	productID := upsertTestBundleProduct(t, db, bundleKey, "Weight Bundle", "prod_weight", "production", 1_000_000, 0.01, "credits")
	defer cleanupBundleProductRecords(t, db, productID)

	insertBundlePrice(t, db, productID, "price_weight", "Weight Plan", "pro", "month", "usd", 999, true, "none", 0, 0, "", 0, 0, 1, 10, "none", "subscription", map[string]interface{}{})

	service := NewPlanService(db)

	newWeight := 99
	updated, err := service.UpdateBundlePrice(t.Context(), bundleKey, "price_weight", UpdateBundlePriceInput{
		DisplayWeight: &newWeight,
	})
	if err != nil {
		t.Fatalf("UpdateBundlePrice failed: %v", err)
	}

	if updated.DisplayWeight != 99 {
		t.Errorf("expected DisplayWeight 99, got %d", updated.DisplayWeight)
	}
}

func TestPlanService_UpdateBundlePrice_UpdatePlanName(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bundleKey := configureTestBundleEnv(t, "production")
	productID := upsertTestBundleProduct(t, db, bundleKey, "Name Bundle", "prod_name", "production", 1_000_000, 0.01, "credits")
	defer cleanupBundleProductRecords(t, db, productID)

	insertBundlePrice(t, db, productID, "price_name", "Old Name", "pro", "month", "usd", 999, true, "none", 0, 0, "", 0, 0, 1, 10, "none", "subscription", map[string]interface{}{})

	service := NewPlanService(db)

	newName := "New Fancy Name"
	updated, err := service.UpdateBundlePrice(t.Context(), bundleKey, "price_name", UpdateBundlePriceInput{
		PlanName: &newName,
	})
	if err != nil {
		t.Fatalf("UpdateBundlePrice failed: %v", err)
	}

	if updated.PlanName != "New Fancy Name" {
		t.Errorf("expected plan name 'New Fancy Name', got %s", updated.PlanName)
	}
}

func TestPlanService_UpdateBundlePrice_UpdateMetadata(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bundleKey := configureTestBundleEnv(t, "production")
	productID := upsertTestBundleProduct(t, db, bundleKey, "Metadata Bundle", "prod_metadata", "production", 1_000_000, 0.01, "credits")
	defer cleanupBundleProductRecords(t, db, productID)

	insertBundlePrice(t, db, productID, "price_meta", "Metadata Plan", "pro", "month", "usd", 999, true, "none", 0, 0, "", 0, 0, 1, 10, "none", "subscription", map[string]interface{}{})

	service := NewPlanService(db)

	subtitle := "Best value!"
	badge := "Popular"
	features := []string{"Feature 1", "Feature 2"}
	updated, err := service.UpdateBundlePrice(t.Context(), bundleKey, "price_meta", UpdateBundlePriceInput{
		Subtitle: &subtitle,
		Badge:    &badge,
		Features: &features,
	})
	if err != nil {
		t.Fatalf("UpdateBundlePrice failed: %v", err)
	}

	if updated.Metadata == nil {
		t.Fatal("expected metadata to be present")
	}
	if _, ok := updated.Metadata["subtitle"]; !ok {
		t.Error("expected subtitle in metadata")
	}
	if _, ok := updated.Metadata["badge"]; !ok {
		t.Error("expected badge in metadata")
	}
	if _, ok := updated.Metadata["features"]; !ok {
		t.Error("expected features in metadata")
	}
}

func TestPlanService_UpdateBundlePrice_ClearStripePriceID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bundleKey := configureTestBundleEnv(t, "production")
	productID := upsertTestBundleProduct(t, db, bundleKey, "Clear ID Bundle", "prod_clear", "production", 1_000_000, 0.01, "credits")
	defer cleanupBundleProductRecords(t, db, productID)

	insertBundlePrice(t, db, productID, "price_clear", "Clear Plan", "pro", "month", "usd", 999, true, "none", 0, 0, "", 0, 0, 1, 10, "none", "subscription", map[string]interface{}{})

	service := NewPlanService(db)

	// Clear the stripe price ID
	emptyStr := ""
	updated, err := service.UpdateBundlePrice(t.Context(), bundleKey, "price_clear", UpdateBundlePriceInput{
		StripePriceID: &emptyStr,
	})
	if err != nil {
		t.Fatalf("UpdateBundlePrice failed: %v", err)
	}

	// The price should still be returned (using internal ID)
	if updated == nil {
		t.Error("expected non-nil updated price")
	}
}

func TestPlanService_UpdateBundlePrice_Highlight(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bundleKey := configureTestBundleEnv(t, "production")
	productID := upsertTestBundleProduct(t, db, bundleKey, "Highlight Bundle", "prod_highlight", "production", 1_000_000, 0.01, "credits")
	defer cleanupBundleProductRecords(t, db, productID)

	insertBundlePrice(t, db, productID, "price_highlight", "Highlight Plan", "pro", "month", "usd", 999, true, "none", 0, 0, "", 0, 0, 1, 10, "none", "subscription", map[string]interface{}{})

	service := NewPlanService(db)

	// Set highlight to true
	highlight := true
	updated, err := service.UpdateBundlePrice(t.Context(), bundleKey, "price_highlight", UpdateBundlePriceInput{
		Highlight: &highlight,
	})
	if err != nil {
		t.Fatalf("UpdateBundlePrice failed: %v", err)
	}

	if updated.Metadata == nil {
		t.Fatal("expected metadata to be present")
	}
	if _, ok := updated.Metadata["highlight"]; !ok {
		t.Error("expected highlight in metadata")
	}
}

func TestPlanService_UpdateBundlePrice_NotFound_ReturnsError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	bundleKey := configureTestBundleEnv(t, "production")
	productID := upsertTestBundleProduct(t, db, bundleKey, "NotFound Bundle", "prod_notfound", "production", 1_000_000, 0.01, "credits")
	defer cleanupBundleProductRecords(t, db, productID)

	service := NewPlanService(db)

	_, err := service.UpdateBundlePrice(t.Context(), bundleKey, "nonexistent_price_id", UpdateBundlePriceInput{})
	if err == nil {
		t.Error("Expected error for non-existent price, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestPlanService_UpdateBundlePrice_EmptyBundleKey_ReturnsError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	service := NewPlanService(db)

	_, err := service.UpdateBundlePrice(t.Context(), "", "price_id", UpdateBundlePriceInput{})
	if err == nil {
		t.Error("Expected error for empty bundle key, got nil")
	}
}

// ============================================================================
// NewPlanServiceWithOptions Tests
// ============================================================================

func TestPlanService_NewPlanServiceWithOptions_OverridesEnvVars(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Set env vars
	_ = os.Setenv("BUNDLE_KEY", "env_bundle")
	_ = os.Setenv("BUNDLE_ENVIRONMENT", "env_environment")
	t.Cleanup(func() {
		_ = os.Unsetenv("BUNDLE_KEY")
		_ = os.Unsetenv("BUNDLE_ENVIRONMENT")
	})

	// Create service with explicit options
	service := NewPlanServiceWithOptions(PlanServiceOptions{
		DB:            db,
		DefaultBundle: "options_bundle",
		DisplayEnv:    "options_env",
	})

	if service.BundleKey() != "options_bundle" {
		t.Errorf("expected bundle key 'options_bundle', got %s", service.BundleKey())
	}
}
