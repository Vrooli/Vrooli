package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	landing_page_react_vite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
)

// createTestPlansFile creates a temporary plans.json file for testing.
// Returns the path to the file.
func createTestPlansFile(t *testing.T, bundle bundleFileFormat, plans []planFileFormat) string {
	t.Helper()
	tmpDir := t.TempDir()
	plansPath := filepath.Join(tmpDir, ".vrooli", "plans.json")
	if err := os.MkdirAll(filepath.Dir(plansPath), 0o755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}

	fileData := plansFileFormat{
		Bundle: bundle,
		Plans:  plans,
	}

	data, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal plans: %v", err)
	}

	if err := os.WriteFile(plansPath, data, 0o644); err != nil {
		t.Fatalf("failed to write plans file: %v", err)
	}

	return plansPath
}

// createTestPlanService creates a PlanService with a test plans file.
func createTestPlanService(t *testing.T, bundle bundleFileFormat, plans []planFileFormat) *PlanService {
	t.Helper()
	plansPath := createTestPlansFile(t, bundle, plans)
	planStore := NewPlanStoreWithOptions(PlanStoreOptions{
		PlansPath:  plansPath,
		BundleKey:  bundle.BundleKey,
		DisplayEnv: bundle.Environment,
	})
	if err := planStore.LoadAll(); err != nil {
		t.Fatalf("failed to load plans: %v", err)
	}
	return NewPlanServiceWithOptions(PlanServiceOptions{
		PlanStore:     planStore,
		DefaultBundle: bundle.BundleKey,
		DisplayEnv:    bundle.Environment,
	})
}

// testBundle returns a standard test bundle configuration.
func testBundle(key, env string) bundleFileFormat {
	return bundleFileFormat{
		BundleKey:                key,
		Name:                     "Test Bundle",
		StripeProductID:          "prod_test",
		CreditsPerUSD:            1_000_000,
		DisplayCreditsMultiplier: 0.01,
		DisplayCreditsLabel:      "credits",
		Environment:              env,
	}
}

func TestPlanServicePricingOverview(t *testing.T) {
	bundle := testBundle("pricing_bundle", "production")
	plans := []planFileFormat{
		{
			StripePriceID:          "price_pricing_monthly",
			PlanName:               "Pricing Monthly",
			PlanTier:               "pro",
			BillingInterval:        "month",
			AmountCents:            4999,
			Currency:               "usd",
			DisplayWeight:          30,
			DisplayEnabled:         true,
			MonthlyIncludedCredits: 5_000_000,
			IntroEnabled:           true,
			IntroType:              "flat_amount",
			Metadata: map[string]interface{}{
				"features": []interface{}{"Fast coupling", "Priority support"},
			},
		},
		{
			StripePriceID:          "price_pricing_yearly",
			PlanName:               "Pricing Yearly",
			PlanTier:               "pro",
			BillingInterval:        "year",
			AmountCents:            55999,
			Currency:               "usd",
			DisplayWeight:          10,
			DisplayEnabled:         true,
			MonthlyIncludedCredits: 60_000_000,
			OneTimeBonusCredits:    10_000_000,
			IntroEnabled:           false,
			Metadata: map[string]interface{}{
				"features": []interface{}{"Annual loyalty", "Bonus credits"},
			},
		},
	}

	service := createTestPlanService(t, bundle, plans)
	overview, err := service.GetPricingOverview()
	if err != nil {
		t.Fatalf("GetPricingOverview failed: %v", err)
	}

	if overview.Bundle.BundleKey != "pricing_bundle" {
		t.Fatalf("expected bundle key pricing_bundle, got %s", overview.Bundle.BundleKey)
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
	bundle := testBundle("lookup_bundle", "production")
	plans := []planFileFormat{
		{
			StripePriceID:          "price_lookup_test",
			PlanName:               "Lookup Plan",
			PlanTier:               "pro",
			BillingInterval:        "month",
			AmountCents:            9999,
			Currency:               "usd",
			DisplayWeight:          40,
			DisplayEnabled:         true,
			MonthlyIncludedCredits: 10_000_000,
			Metadata: map[string]interface{}{
				"features": []interface{}{"Lookup feature"},
			},
		},
	}

	service := createTestPlanService(t, bundle, plans)
	option, err := service.GetPlanByPriceID("price_lookup_test")
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
	bundle := testBundle("ordering_bundle", "production")
	plans := []planFileFormat{
		{
			StripePriceID:   "price_weight_10",
			PlanName:        "Weighted",
			PlanTier:        "pro",
			BillingInterval: "month",
			AmountCents:     1000,
			Currency:        "usd",
			DisplayWeight:   10,
			DisplayEnabled:  true,
			PlanRank:        5,
		},
		{
			StripePriceID:   "price_weight_5_rank1",
			PlanName:        "Rank 1",
			PlanTier:        "pro",
			BillingInterval: "month",
			AmountCents:     2000,
			Currency:        "usd",
			DisplayWeight:   5,
			DisplayEnabled:  true,
			PlanRank:        1,
		},
		{
			StripePriceID:   "price_weight_5_rank2",
			PlanName:        "Rank 2",
			PlanTier:        "pro",
			BillingInterval: "month",
			AmountCents:     3000,
			Currency:        "usd",
			DisplayWeight:   5,
			DisplayEnabled:  false, // Disabled - should be filtered out
			PlanRank:        2,
		},
	}

	service := createTestPlanService(t, bundle, plans)
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
	bundle := testBundle("error_bundle", "production")
	plans := []planFileFormat{}

	service := createTestPlanService(t, bundle, plans)

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

// ============================================================================
// GetPricingOverview Tests
// ============================================================================

func TestPlanService_GetPricingOverview_FreeTierAlwaysIncluded(t *testing.T) {
	bundle := testBundle("free_bundle", "production")
	plans := []planFileFormat{
		{
			StripePriceID:   "price_free",
			PlanName:        "Free Plan",
			PlanTier:        "free",
			BillingInterval: "month",
			AmountCents:     0,
			Currency:        "usd",
			DisplayWeight:   100,
			DisplayEnabled:  false, // Free tier should be included even when disabled
		},
	}

	service := createTestPlanService(t, bundle, plans)
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
	// Create an empty PlanStore (no plans file loaded)
	emptyStore := NewPlanStoreWithOptions(PlanStoreOptions{
		PlansPath:  "", // Empty path means no plans will be loaded
		BundleKey:  "nonexistent",
		DisplayEnv: "production",
	})
	service := NewPlanServiceWithPlanStore(emptyStore)

	_, err := service.GetPricingOverview()
	if err == nil {
		t.Error("Expected error for non-existent bundle, got nil")
	}
}

func TestPlanService_GetPricingOverview_NoPrices_ReturnsEmptySlices(t *testing.T) {
	bundle := testBundle("empty_bundle", "production")
	plans := []planFileFormat{} // No prices

	service := createTestPlanService(t, bundle, plans)
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
	bundle := bundleFileFormat{
		BundleKey:                "metadata_bundle",
		Name:                     "Test Product",
		StripeProductID:          "prod_test_metadata",
		CreditsPerUSD:            2_000_000,
		DisplayCreditsMultiplier: 0.02,
		DisplayCreditsLabel:      "tokens",
		Environment:              "production",
	}
	plans := []planFileFormat{}

	service := createTestPlanService(t, bundle, plans)
	product, err := service.GetBundleProduct()
	if err != nil {
		t.Fatalf("GetBundleProduct failed: %v", err)
	}

	if product.BundleKey != "metadata_bundle" {
		t.Errorf("expected bundle key metadata_bundle, got %s", product.BundleKey)
	}
	if product.Name != "Test Product" {
		t.Errorf("expected name 'Test Product', got %s", product.Name)
	}
	if product.CreditsPerUsd != 2_000_000 {
		t.Errorf("expected credits_per_usd 2000000, got %d", product.CreditsPerUsd)
	}
}

func TestPlanService_GetBundleProduct_NotFound_ReturnsNil(t *testing.T) {
	emptyStore := NewPlanStoreWithOptions(PlanStoreOptions{
		PlansPath:  "",
		BundleKey:  "nonexistent",
		DisplayEnv: "production",
	})
	service := NewPlanServiceWithPlanStore(emptyStore)

	product, err := service.GetBundleProduct()
	// When bundle doesn't exist, returns nil without error
	if err != nil {
		t.Errorf("Expected no error for non-existent bundle product, got: %v", err)
	}
	if product != nil {
		t.Error("Expected nil product for non-existent bundle")
	}
}

// ============================================================================
// ListBundleCatalog Tests
// ============================================================================

func TestPlanService_ListBundleCatalog_ReturnsBundlesWithPrices(t *testing.T) {
	bundle := testBundle("catalog_bundle", "catalog_env")
	plans := []planFileFormat{
		{
			StripePriceID:   "price_catalog_1",
			PlanName:        "Catalog Plan 1",
			PlanTier:        "pro",
			BillingInterval: "month",
			AmountCents:     999,
			Currency:        "usd",
			DisplayWeight:   10,
			DisplayEnabled:  true,
		},
		{
			StripePriceID:   "price_catalog_2",
			PlanName:        "Catalog Plan 2",
			PlanTier:        "business",
			BillingInterval: "year",
			AmountCents:     9999,
			Currency:        "usd",
			DisplayWeight:   5,
			DisplayEnabled:  true,
		},
	}

	plansPath := createTestPlansFile(t, bundle, plans)
	planStore := NewPlanStoreWithOptions(PlanStoreOptions{
		PlansPath:  plansPath,
		BundleKey:  "catalog_bundle",
		DisplayEnv: "catalog_env",
	})
	if err := planStore.LoadAll(); err != nil {
		t.Fatalf("failed to load plans: %v", err)
	}

	service := NewPlanServiceWithOptions(PlanServiceOptions{
		PlanStore:     planStore,
		DefaultBundle: "catalog_bundle",
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
		if entry.Bundle.BundleKey == "catalog_bundle" {
			found = true
			if len(entry.Prices) != 2 {
				t.Errorf("expected 2 prices for bundle, got %d", len(entry.Prices))
			}
		}
	}
	if !found {
		t.Errorf("expected to find bundle catalog_bundle in catalog")
	}
}

func TestPlanService_ListBundleCatalog_EmptyCatalog_ReturnsEmptySlice(t *testing.T) {
	emptyStore := NewPlanStoreWithOptions(PlanStoreOptions{
		PlansPath:  "",
		BundleKey:  "nonexistent",
		DisplayEnv: "nonexistent_env",
	})

	service := NewPlanServiceWithOptions(PlanServiceOptions{
		PlanStore:     emptyStore,
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
	bundle := testBundle("update_bundle", "production")
	plans := []planFileFormat{
		{
			StripePriceID:   "price_toggle",
			PlanName:        "Toggle Plan",
			PlanTier:        "pro",
			BillingInterval: "month",
			AmountCents:     999,
			Currency:        "usd",
			DisplayWeight:   10,
			DisplayEnabled:  true,
		},
	}

	service := createTestPlanService(t, bundle, plans)

	// Disable the price
	disabled := false
	updated, err := service.UpdateBundlePrice(t.Context(), "update_bundle", "price_toggle", UpdateBundlePriceInput{
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
	bundle := testBundle("weight_bundle", "production")
	plans := []planFileFormat{
		{
			StripePriceID:   "price_weight",
			PlanName:        "Weight Plan",
			PlanTier:        "pro",
			BillingInterval: "month",
			AmountCents:     999,
			Currency:        "usd",
			DisplayWeight:   10,
			DisplayEnabled:  true,
		},
	}

	service := createTestPlanService(t, bundle, plans)

	newWeight := 99
	updated, err := service.UpdateBundlePrice(t.Context(), "weight_bundle", "price_weight", UpdateBundlePriceInput{
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
	bundle := testBundle("name_bundle", "production")
	plans := []planFileFormat{
		{
			StripePriceID:   "price_name",
			PlanName:        "Old Name",
			PlanTier:        "pro",
			BillingInterval: "month",
			AmountCents:     999,
			Currency:        "usd",
			DisplayWeight:   10,
			DisplayEnabled:  true,
		},
	}

	service := createTestPlanService(t, bundle, plans)

	newName := "New Fancy Name"
	updated, err := service.UpdateBundlePrice(t.Context(), "name_bundle", "price_name", UpdateBundlePriceInput{
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
	bundle := testBundle("metadata_update_bundle", "production")
	plans := []planFileFormat{
		{
			StripePriceID:   "price_meta",
			PlanName:        "Metadata Plan",
			PlanTier:        "pro",
			BillingInterval: "month",
			AmountCents:     999,
			Currency:        "usd",
			DisplayWeight:   10,
			DisplayEnabled:  true,
		},
	}

	service := createTestPlanService(t, bundle, plans)

	subtitle := "Best value!"
	badge := "Popular"
	features := []string{"Feature 1", "Feature 2"}
	updated, err := service.UpdateBundlePrice(t.Context(), "metadata_update_bundle", "price_meta", UpdateBundlePriceInput{
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

func TestPlanService_UpdateBundlePrice_Highlight(t *testing.T) {
	bundle := testBundle("highlight_bundle", "production")
	plans := []planFileFormat{
		{
			StripePriceID:   "price_highlight",
			PlanName:        "Highlight Plan",
			PlanTier:        "pro",
			BillingInterval: "month",
			AmountCents:     999,
			Currency:        "usd",
			DisplayWeight:   10,
			DisplayEnabled:  true,
		},
	}

	service := createTestPlanService(t, bundle, plans)

	// Set highlight to true
	highlight := true
	updated, err := service.UpdateBundlePrice(t.Context(), "highlight_bundle", "price_highlight", UpdateBundlePriceInput{
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
	bundle := testBundle("notfound_bundle", "production")
	plans := []planFileFormat{} // No plans

	service := createTestPlanService(t, bundle, plans)

	_, err := service.UpdateBundlePrice(t.Context(), "notfound_bundle", "nonexistent_price_id", UpdateBundlePriceInput{})
	if err == nil {
		t.Error("Expected error for non-existent price, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestPlanService_UpdateBundlePrice_EmptyBundleKey_ReturnsNil(t *testing.T) {
	bundle := testBundle("test_bundle", "production")
	plans := []planFileFormat{}

	service := createTestPlanService(t, bundle, plans)

	// When bundle key is empty, returns nil without error
	result, err := service.UpdateBundlePrice(t.Context(), "", "price_id", UpdateBundlePriceInput{})
	if err != nil {
		t.Errorf("Expected no error for empty bundle key, got: %v", err)
	}
	if result != nil {
		t.Error("Expected nil result for empty bundle key")
	}
}

func TestPlanService_CreateBundlePrice_RejectsStripeProductMismatch(t *testing.T) {
	bundle := testBundle("create_bundle", "production")
	service := createTestPlanService(t, bundle, nil)

	fetcher := func(ctx context.Context, priceID string) (*StripePriceImport, error) {
		return &StripePriceImport{
			PriceID:     priceID,
			LookupKey:   "pro_monthly",
			Currency:    "usd",
			AmountCents: 1200,
			Interval:    "month",
			ProductID:   "prod_other",
			ProductName: "Other Bundle",
			Active:      true,
		}, nil
	}

	_, err := service.CreateBundlePrice(context.Background(), bundle.BundleKey, CreateBundlePriceInput{
		StripePriceID:   "price_new",
		PlanName:        "Pro Monthly",
		PlanTier:        "pro",
		BillingInterval: "month",
	}, fetcher)
	if err == nil {
		t.Fatal("expected error for mismatched stripe product")
	}
	if !strings.Contains(err.Error(), "product") {
		t.Fatalf("expected product mismatch error, got: %v", err)
	}
}

func TestPlanService_CreateBundlePrice_RejectsAmountMismatch(t *testing.T) {
	bundle := testBundle("create_bundle_amount", "production")
	service := createTestPlanService(t, bundle, nil)

	fetcher := func(ctx context.Context, priceID string) (*StripePriceImport, error) {
		return &StripePriceImport{
			PriceID:     priceID,
			LookupKey:   "pro_monthly",
			Currency:    "usd",
			AmountCents: 1500,
			Interval:    "month",
			ProductID:   bundle.StripeProductID,
			ProductName: "Test Bundle",
			Active:      true,
		}, nil
	}

	amount := int64(999)
	_, err := service.CreateBundlePrice(context.Background(), bundle.BundleKey, CreateBundlePriceInput{
		StripePriceID:   "price_new",
		PlanName:        "Pro Monthly",
		PlanTier:        "pro",
		BillingInterval: "month",
		AmountCents:     &amount,
	}, fetcher)
	if err == nil {
		t.Fatal("expected error for amount mismatch")
	}
	if !strings.Contains(err.Error(), "amount_cents") {
		t.Fatalf("expected amount mismatch error, got: %v", err)
	}
}

func TestEnsureStripePriceMatchesBundle(t *testing.T) {
	bundle := &BundleProduct{
		BundleKey:       "bundle_key",
		Name:            "Bundle",
		StripeProductId: "prod_bundle",
		CreditsPerUsd:   1_000_000,
		Environment:     "test",
	}

	err := ensureStripePriceMatchesBundle(bundle, &StripePriceImport{
		PriceID:   "price_123",
		ProductID: "prod_other",
	})
	if err == nil {
		t.Fatal("expected error for mismatched product")
	}
	if !strings.Contains(err.Error(), "expected") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlanService_ImportStripePrices_RequiresSelections(t *testing.T) {
	bundle := testBundle("import_bundle", "production")
	service := createTestPlanService(t, bundle, nil)

	_, err := service.ImportStripePrices(context.Background(), nil, func(ctx context.Context, priceID string) (*StripePriceImport, error) {
		return nil, nil
	})
	if err == nil {
		t.Fatal("expected error for empty selections")
	}
}

func TestPlanService_ImportStripePrices_ImportsNewPrice(t *testing.T) {
	bundle := testBundle("import_bundle_valid", "production")
	service := createTestPlanService(t, bundle, nil)

	fetcher := func(ctx context.Context, priceID string) (*StripePriceImport, error) {
		return &StripePriceImport{
			PriceID:     priceID,
			LookupKey:   "pro_monthly",
			Currency:    "usd",
			AmountCents: 2000,
			Interval:    "month",
			ProductID:   bundle.StripeProductID,
			ProductName: "Test Bundle",
			Active:      true,
		}, nil
	}

	result, err := service.ImportStripePrices(context.Background(), []ImportPlanSelection{
		{PriceID: "price_new", Action: "import"},
	}, fetcher)
	if err != nil {
		t.Fatalf("unexpected import error: %v", err)
	}
	if result.Imported != 1 {
		t.Fatalf("expected 1 imported, got %d", result.Imported)
	}
}

// ============================================================================
// NewPlanServiceWithOptions Tests
// ============================================================================

func TestPlanService_NewPlanServiceWithOptions_OverridesEnvVars(t *testing.T) {
	// Set env vars
	_ = os.Setenv("BUNDLE_KEY", "env_bundle")
	_ = os.Setenv("BUNDLE_ENVIRONMENT", "env_environment")
	t.Cleanup(func() {
		_ = os.Unsetenv("BUNDLE_KEY")
		_ = os.Unsetenv("BUNDLE_ENVIRONMENT")
	})

	bundle := testBundle("options_bundle", "options_env")
	plansPath := createTestPlansFile(t, bundle, nil)
	planStore := NewPlanStoreWithOptions(PlanStoreOptions{
		PlansPath:  plansPath,
		BundleKey:  "options_bundle",
		DisplayEnv: "options_env",
	})
	if err := planStore.LoadAll(); err != nil {
		t.Fatalf("failed to load plans: %v", err)
	}

	// Create service with explicit options
	service := NewPlanServiceWithOptions(PlanServiceOptions{
		PlanStore:     planStore,
		DefaultBundle: "options_bundle",
		DisplayEnv:    "options_env",
	})

	if service.BundleKey() != "options_bundle" {
		t.Errorf("expected bundle key 'options_bundle', got %s", service.BundleKey())
	}
}
