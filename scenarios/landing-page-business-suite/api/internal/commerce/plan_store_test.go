package commerce

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	landing_page_business_suite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

// testPlansJSON returns a sample plans.json content for testing.
func testPlansJSON() []byte {
	return []byte(`{
  "bundle": {
    "bundle_key": "test_bundle",
    "name": "Test Bundle",
    "stripe_product_id": "prod_test123",
    "credits_per_usd": 1000000,
    "display_credits_multiplier": 0.001,
    "display_credits_label": "credits",
    "environment": "test"
  },
  "plans": [
    {
      "stripe_price_id": "price_monthly_pro",
      "plan_name": "Pro Monthly",
      "plan_tier": "pro",
      "billing_interval": "month",
      "amount_cents": 2900,
      "currency": "usd",
      "display_weight": 20,
      "display_enabled": true,
      "monthly_included_credits": 200,
      "metadata": {
        "subtitle": "For professionals",
        "features": ["Feature 1", "Feature 2"]
      }
    },
    {
      "stripe_price_id": "price_yearly_pro",
      "plan_name": "Pro Yearly",
      "plan_tier": "pro",
      "billing_interval": "year",
      "amount_cents": 29000,
      "currency": "usd",
      "display_weight": 10,
      "display_enabled": true,
      "monthly_included_credits": 200
    },
    {
      "stripe_price_id": "price_free",
      "plan_name": "Free",
      "plan_tier": "free",
      "billing_interval": "month",
      "amount_cents": 0,
      "currency": "usd",
      "display_weight": 30,
      "display_enabled": false
    }
  ],
  "updated_at": "2025-01-01T00:00:00Z"
}`)
}

// setupTestPlansFile creates a temporary plans.json file for testing.
func setupTestPlansFile(t *testing.T, content []byte) string {
	t.Helper()
	tmpDir := t.TempDir()
	plansPath := filepath.Join(tmpDir, ".vrooli", "plans.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(plansPath), 0o755))
	require.NoError(t, os.WriteFile(plansPath, content, 0o644))
	return plansPath
}

func TestNewPlanStore(t *testing.T) {
	t.Setenv("BUNDLE_KEY", "test_key")
	t.Setenv("BUNDLE_ENVIRONMENT", "staging")

	ps := NewPlanStore("/tmp/test/plans.json")

	assert.NotNil(t, ps)
	assert.Equal(t, "/tmp/test/plans.json", ps.plansPath)
	assert.Equal(t, "test_key", ps.bundleKey)
	assert.Equal(t, "staging", ps.displayEnv)
	assert.Empty(t, ps.plans)
}

func TestNewPlanStoreWithOptions(t *testing.T) {
	ps := NewPlanStoreWithOptions(PlanStoreOptions{
		PlansPath:  "/custom/path/plans.json",
		BundleKey:  "custom_bundle",
		DisplayEnv: "development",
	})

	assert.NotNil(t, ps)
	assert.Equal(t, "/custom/path/plans.json", ps.plansPath)
	assert.Equal(t, "custom_bundle", ps.bundleKey)
	assert.Equal(t, "development", ps.displayEnv)
}

func TestNewPlanStoreWithOptions_Defaults(t *testing.T) {
	t.Setenv("BUNDLE_KEY", "env_bundle")
	t.Setenv("BUNDLE_ENVIRONMENT", "env_env")

	ps := NewPlanStoreWithOptions(PlanStoreOptions{
		PlansPath: "/some/path",
	})

	assert.Equal(t, "env_bundle", ps.bundleKey)
	assert.Equal(t, "env_env", ps.displayEnv)
}

func TestBundleKey(t *testing.T) {
	ps := NewPlanStoreWithOptions(PlanStoreOptions{
		PlansPath: "/test",
		BundleKey: "my_bundle",
	})

	assert.Equal(t, "my_bundle", ps.BundleKey())
}

func TestLoadAll_ValidFile(t *testing.T) {
	plansPath := setupTestPlansFile(t, testPlansJSON())

	ps := NewPlanStoreWithOptions(PlanStoreOptions{
		PlansPath: plansPath,
		BundleKey: "test_bundle",
	})

	err := ps.LoadAll()
	require.NoError(t, err)

	// Check bundle
	bundle := ps.GetBundle()
	require.NotNil(t, bundle)
	assert.Equal(t, "test_bundle", bundle.BundleKey)
	assert.Equal(t, "Test Bundle", bundle.Name)
	assert.Equal(t, "prod_test123", bundle.StripeProductId)
	assert.Equal(t, int64(1000000), bundle.CreditsPerUsd)
	assert.Equal(t, 0.001, bundle.DisplayCreditsMultiplier)
	assert.Equal(t, "credits", bundle.DisplayCreditsLabel)

	// Check plans
	plans := ps.GetPlans()
	assert.Len(t, plans, 3)
}

func TestLoadAll_MissingFile(t *testing.T) {
	ps := NewPlanStoreWithOptions(PlanStoreOptions{
		PlansPath: "/nonexistent/path/plans.json",
		BundleKey: "test",
	})

	err := ps.LoadAll()
	assert.NoError(t, err) // Should not error, just log and return empty
	assert.Empty(t, ps.GetPlans())
}

func TestLoadAll_EmptyPath(t *testing.T) {
	ps := NewPlanStoreWithOptions(PlanStoreOptions{
		PlansPath: "",
		BundleKey: "test",
	})

	err := ps.LoadAll()
	assert.NoError(t, err)
	assert.Empty(t, ps.GetPlans())
}

func TestLoadAll_InvalidJSON(t *testing.T) {
	plansPath := setupTestPlansFile(t, []byte(`{invalid json}`))

	ps := NewPlanStoreWithOptions(PlanStoreOptions{
		PlansPath: plansPath,
		BundleKey: "test",
	})

	err := ps.LoadAll()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse plans JSON")
}

func TestGetBundle_NilBundle(t *testing.T) {
	ps := NewPlanStore("/tmp/test")
	bundle := ps.GetBundle()
	assert.Nil(t, bundle)
}

func TestGetBundle_ReturnsClone(t *testing.T) {
	plansPath := setupTestPlansFile(t, testPlansJSON())
	ps := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps.LoadAll())

	bundle1 := ps.GetBundle()
	bundle2 := ps.GetBundle()

	// Modify bundle1
	bundle1.Name = "Modified"

	// bundle2 should be unaffected
	assert.NotEqual(t, bundle1.Name, bundle2.Name)
	assert.Equal(t, "Test Bundle", bundle2.Name)
}

func TestGetPlans_ReturnsClones(t *testing.T) {
	plansPath := setupTestPlansFile(t, testPlansJSON())
	ps := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps.LoadAll())

	plans1 := ps.GetPlans()
	plans2 := ps.GetPlans()

	// Modify plans1[0]
	plans1[0].PlanName = "Modified"

	// plans2 should be unaffected
	assert.NotEqual(t, plans1[0].PlanName, plans2[0].PlanName)
}

func TestGetPlanByPriceID_Found(t *testing.T) {
	plansPath := setupTestPlansFile(t, testPlansJSON())
	ps := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps.LoadAll())

	plan, err := ps.GetPlanByPriceID("price_monthly_pro")
	require.NoError(t, err)
	assert.NotNil(t, plan)
	assert.Equal(t, "Pro Monthly", plan.PlanName)
	assert.Equal(t, int64(2900), plan.AmountCents)
}

func TestGetPlanByPriceID_NotFound(t *testing.T) {
	plansPath := setupTestPlansFile(t, testPlansJSON())
	ps := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps.LoadAll())

	plan, err := ps.GetPlanByPriceID("price_nonexistent")
	assert.Error(t, err)
	assert.Nil(t, plan)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetPlanByPriceID_EmptyID(t *testing.T) {
	ps := NewPlanStore("/tmp/test")
	plan, err := ps.GetPlanByPriceID("")
	assert.Error(t, err)
	assert.Nil(t, plan)
	assert.Contains(t, err.Error(), "required")
}

func TestGetPlanByExternalProductID(t *testing.T) {
	ps := NewPlanStore("/tmp/test")
	ps.plans = []*PlanOption{{
		PlanName: "Apple Studio",
		PlanTier: "studio",
		Metadata: map[string]*commonv1.JsonValue{"external_product_id": newStringJsonValue("apple.studio")},
	}}

	plan, err := ps.GetPlanByExternalProductID("apple.studio")
	require.NoError(t, err)
	assert.Equal(t, "Apple Studio", plan.PlanName)

	missing, err := ps.GetPlanByExternalProductID("google.missing")
	assert.Error(t, err)
	assert.Nil(t, missing)
}

func TestGetPricingOverview(t *testing.T) {
	plansPath := setupTestPlansFile(t, testPlansJSON())
	ps := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps.LoadAll())

	overview, err := ps.GetPricingOverview()
	require.NoError(t, err)
	require.NotNil(t, overview)

	// Check bundle
	assert.Equal(t, "Test Bundle", overview.Bundle.Name)

	// Monthly plans: free (display_enabled=false but tier=free so included) + monthly_pro
	// The free plan has display_enabled=false but tier="free" so it should be included
	assert.GreaterOrEqual(t, len(overview.Monthly), 1)

	// Yearly plans: yearly_pro
	assert.Len(t, overview.Yearly, 1)
	assert.Equal(t, "Pro Yearly", overview.Yearly[0].PlanName)
}

func TestGetPricingOverview_NilBundle(t *testing.T) {
	ps := NewPlanStore("/tmp/test")
	overview, err := ps.GetPricingOverview()
	assert.Error(t, err)
	assert.Nil(t, overview)
	assert.Contains(t, err.Error(), "bundle not configured")
}

func TestListBundleCatalog(t *testing.T) {
	plansPath := setupTestPlansFile(t, testPlansJSON())
	ps := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps.LoadAll())

	catalog, err := ps.ListBundleCatalog(context.Background())
	require.NoError(t, err)
	require.Len(t, catalog, 1)

	entry := catalog[0]
	assert.Equal(t, "Test Bundle", entry.Bundle.Name)
	assert.Len(t, entry.Prices, 3)
}

func TestListBundleCatalog_NilBundle(t *testing.T) {
	ps := NewPlanStore("/tmp/test")
	catalog, err := ps.ListBundleCatalog(context.Background())
	require.NoError(t, err)
	assert.Empty(t, catalog)
}

func TestAddPlan(t *testing.T) {
	plansPath := setupTestPlansFile(t, testPlansJSON())
	ps := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps.LoadAll())

	initialCount := len(ps.GetPlans())

	newPlan := &PlanOption{
		StripePriceId:   "price_new_plan",
		PlanName:        "New Plan",
		PlanTier:        "solo",
		BillingInterval: landing_page_business_suite_v1.BillingInterval_BILLING_INTERVAL_MONTH,
		AmountCents:     1900,
		Currency:        "usd",
		DisplayWeight:   15,
		DisplayEnabled:  true,
	}

	created, err := ps.AddPlan(newPlan)
	require.NoError(t, err)
	require.NotNil(t, created)

	plans := ps.GetPlans()
	assert.Len(t, plans, initialCount+1)

	// Verify it was saved to file
	ps2 := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps2.LoadAll())
	assert.Len(t, ps2.GetPlans(), initialCount+1)
}

func TestAddPlan_DuplicatePriceID(t *testing.T) {
	plansPath := setupTestPlansFile(t, testPlansJSON())
	ps := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps.LoadAll())

	duplicatePlan := &PlanOption{
		StripePriceId:   "price_monthly_pro", // Already exists
		PlanName:        "Duplicate",
		PlanTier:        "pro",
		BillingInterval: landing_page_business_suite_v1.BillingInterval_BILLING_INTERVAL_MONTH,
		AmountCents:     1000,
		Currency:        "usd",
		DisplayWeight:   10,
		DisplayEnabled:  true,
	}

	_, err := ps.AddPlan(duplicatePlan)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestAddPlan_EmptyPriceID(t *testing.T) {
	ps := NewPlanStore("/tmp/test")
	_, err := ps.AddPlan(&PlanOption{PlanName: "No Price ID"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stripe_price_id is required")
}

func TestAddPlan_SetsBundleKey(t *testing.T) {
	plansPath := setupTestPlansFile(t, testPlansJSON())
	ps := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps.LoadAll())

	newPlan := &PlanOption{
		StripePriceId:   "price_bundle_test",
		PlanName:        "Bundle Test",
		PlanTier:        "pro",
		BillingInterval: landing_page_business_suite_v1.BillingInterval_BILLING_INTERVAL_MONTH,
		AmountCents:     1000,
		Currency:        "usd",
		DisplayWeight:   10,
		DisplayEnabled:  true,
	}

	created, err := ps.AddPlan(newPlan)
	require.NoError(t, err)
	require.NotNil(t, created)

	plan, err := ps.GetPlanByPriceID("price_bundle_test")
	require.NoError(t, err)
	assert.Equal(t, "test_bundle", plan.BundleKey)
}

func TestUpdatePlan(t *testing.T) {
	plansPath := setupTestPlansFile(t, testPlansJSON())
	ps := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps.LoadAll())

	newName := "Updated Pro Monthly"
	newWeight := 99
	newEnabled := false

	updated, err := ps.UpdatePlan("price_monthly_pro", UpdateBundlePriceInput{
		PlanName:       &newName,
		DisplayWeight:  &newWeight,
		DisplayEnabled: &newEnabled,
	})
	require.NoError(t, err)

	assert.Equal(t, "Updated Pro Monthly", updated.PlanName)
	assert.Equal(t, int32(99), updated.DisplayWeight)
	assert.False(t, updated.DisplayEnabled)

	// Verify persisted
	ps2 := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps2.LoadAll())
	plan, _ := ps2.GetPlanByPriceID("price_monthly_pro")
	assert.Equal(t, "Updated Pro Monthly", plan.PlanName)
}

func TestUpdatePlan_NotFound(t *testing.T) {
	plansPath := setupTestPlansFile(t, testPlansJSON())
	ps := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps.LoadAll())

	name := "test"
	_, err := ps.UpdatePlan("nonexistent", UpdateBundlePriceInput{PlanName: &name})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUpdatePlan_EmptyPriceID(t *testing.T) {
	ps := NewPlanStore("/tmp/test")
	_, err := ps.UpdatePlan("", UpdateBundlePriceInput{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestUpdatePlan_Metadata(t *testing.T) {
	plansPath := setupTestPlansFile(t, testPlansJSON())
	ps := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps.LoadAll())

	subtitle := "New Subtitle"
	badge := "Popular"
	ctaLabel := "Get Started"
	highlight := true
	features := []string{"Feature A", "Feature B", "Feature C"}

	updated, err := ps.UpdatePlan("price_monthly_pro", UpdateBundlePriceInput{
		Subtitle:  &subtitle,
		Badge:     &badge,
		CtaLabel:  &ctaLabel,
		Highlight: &highlight,
		Features:  &features,
	})
	require.NoError(t, err)

	assert.NotNil(t, updated.Metadata)
	assert.NotNil(t, updated.Metadata["subtitle"])
	assert.NotNil(t, updated.Metadata["badge"])
	assert.NotNil(t, updated.Metadata["cta_label"])
	assert.NotNil(t, updated.Metadata["highlight"])
	assert.NotNil(t, updated.Metadata["features"])
}

func TestUpdatePlan_ClearMetadata(t *testing.T) {
	plansPath := setupTestPlansFile(t, testPlansJSON())
	ps := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps.LoadAll())

	emptyStr := ""
	noHighlight := false
	emptyFeatures := []string{}

	updated, err := ps.UpdatePlan("price_monthly_pro", UpdateBundlePriceInput{
		Subtitle:  &emptyStr,
		Highlight: &noHighlight,
		Features:  &emptyFeatures,
	})
	require.NoError(t, err)

	// These should be removed from metadata
	_, hasSubtitle := updated.Metadata["subtitle"]
	_, hasHighlight := updated.Metadata["highlight"]
	_, hasFeatures := updated.Metadata["features"]

	assert.False(t, hasSubtitle)
	assert.False(t, hasHighlight)
	assert.False(t, hasFeatures)
}

func TestDeletePlan(t *testing.T) {
	plansPath := setupTestPlansFile(t, testPlansJSON())
	ps := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps.LoadAll())

	initialCount := len(ps.GetPlans())

	err := ps.DeletePlan("price_monthly_pro")
	require.NoError(t, err)

	plans := ps.GetPlans()
	assert.Len(t, plans, initialCount-1)

	// Verify it was persisted
	ps2 := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps2.LoadAll())
	assert.Len(t, ps2.GetPlans(), initialCount-1)

	// Verify the specific plan is gone
	_, err = ps.GetPlanByPriceID("price_monthly_pro")
	assert.Error(t, err)
}

func TestDeletePlan_NotFound(t *testing.T) {
	plansPath := setupTestPlansFile(t, testPlansJSON())
	ps := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps.LoadAll())

	err := ps.DeletePlan("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDeletePlan_EmptyPriceID(t *testing.T) {
	ps := NewPlanStore("/tmp/test")
	err := ps.DeletePlan("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestSetPlans(t *testing.T) {
	plansPath := setupTestPlansFile(t, testPlansJSON())
	ps := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps.LoadAll())

	newBundle := &BundleProduct{
		BundleKey:       "test_bundle",
		Name:            "New Bundle",
		StripeProductId: "prod_new",
		CreditsPerUsd:   500000,
	}

	newPlans := []*PlanOption{
		{
			StripePriceId:   "price_set_1",
			PlanName:        "Set Plan 1",
			PlanTier:        "solo",
			BillingInterval: landing_page_business_suite_v1.BillingInterval_BILLING_INTERVAL_MONTH,
			AmountCents:     999,
			Currency:        "usd",
			DisplayWeight:   10,
			DisplayEnabled:  true,
		},
		{
			StripePriceId:   "price_set_2",
			PlanName:        "Set Plan 2",
			PlanTier:        "pro",
			BillingInterval: landing_page_business_suite_v1.BillingInterval_BILLING_INTERVAL_YEAR,
			AmountCents:     9999,
			Currency:        "usd",
			DisplayWeight:   10,
			DisplayEnabled:  true,
		},
	}

	err := ps.SetPlans(newBundle, newPlans)
	require.NoError(t, err)

	// Verify bundle
	bundle := ps.GetBundle()
	assert.Equal(t, "New Bundle", bundle.Name)

	// Verify plans
	plans := ps.GetPlans()
	assert.Len(t, plans, 2)

	// Verify bundle key was set on plans
	for _, plan := range plans {
		assert.Equal(t, "test_bundle", plan.BundleKey)
	}

	// Verify persisted
	ps2 := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps2.LoadAll())
	assert.Len(t, ps2.GetPlans(), 2)
}

func TestSetPlans_NilBundle(t *testing.T) {
	plansPath := setupTestPlansFile(t, testPlansJSON())
	ps := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps.LoadAll())

	originalBundle := ps.GetBundle()

	newPlans := []*PlanOption{
		{
			StripePriceId:   "price_only",
			PlanName:        "Only Plan",
			PlanTier:        "pro",
			BillingInterval: landing_page_business_suite_v1.BillingInterval_BILLING_INTERVAL_MONTH,
			AmountCents:     1000,
			Currency:        "usd",
			DisplayWeight:   10,
			DisplayEnabled:  true,
		},
	}

	err := ps.SetPlans(nil, newPlans)
	require.NoError(t, err)

	// Bundle should remain unchanged
	bundle := ps.GetBundle()
	assert.Equal(t, originalBundle.Name, bundle.Name)

	// Plans should be replaced
	assert.Len(t, ps.GetPlans(), 1)
}

func TestSavePlans_EmptyPath(t *testing.T) {
	ps := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: ""})
	err := ps.SavePlans()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestSavePlans_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	plansPath := filepath.Join(tmpDir, "deep", "nested", "dir", "plans.json")

	ps := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	ps.bundle = &BundleProduct{
		BundleKey:       "test_bundle",
		Name:            "Test",
		StripeProductId: "prod_test",
		CreditsPerUsd:   1000000,
		Environment:     "test",
	}
	ps.plans = []*PlanOption{
		{
			StripePriceId:   "price_test",
			PlanName:        "Test",
			PlanTier:        "pro",
			BillingInterval: landing_page_business_suite_v1.BillingInterval_BILLING_INTERVAL_MONTH,
			AmountCents:     1000,
			Currency:        "usd",
			DisplayWeight:   10,
			DisplayEnabled:  true,
		},
	}

	err := ps.SavePlans()
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(plansPath)
	assert.NoError(t, err)
}

func TestConcurrentAccess(t *testing.T) {
	plansPath := setupTestPlansFile(t, testPlansJSON())
	ps := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps.LoadAll())

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent readers
	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = ps.GetBundle()
			_ = ps.GetPlans()
			_, _ = ps.GetPricingOverview()
		}()
	}

	// Concurrent writers (adding and deleting different plans)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			priceID := "price_concurrent_" + string(rune('a'+idx))
			_, _ = ps.AddPlan(&PlanOption{
				StripePriceId:   priceID,
				PlanName:        "Concurrent Plan",
				PlanTier:        "pro",
				BillingInterval: landing_page_business_suite_v1.BillingInterval_BILLING_INTERVAL_MONTH,
				AmountCents:     1000,
				Currency:        "usd",
				DisplayWeight:   10,
				DisplayEnabled:  true,
			})
			_ = ps.DeletePlan(priceID)
		}(i)
	}

	wg.Wait()
}

func TestBillingIntervalMapping(t *testing.T) {
	tests := []struct {
		input    string
		expected landing_page_business_suite_v1.BillingInterval
	}{
		{"month", landing_page_business_suite_v1.BillingInterval_BILLING_INTERVAL_MONTH},
		{"year", landing_page_business_suite_v1.BillingInterval_BILLING_INTERVAL_YEAR},
		{"one_time", landing_page_business_suite_v1.BillingInterval_BILLING_INTERVAL_ONE_TIME},
		{"unknown", landing_page_business_suite_v1.BillingInterval_BILLING_INTERVAL_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapBillingInterval(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPlanKindMapping(t *testing.T) {
	tests := []struct {
		input    string
		expected landing_page_business_suite_v1.PlanKind
	}{
		{"subscription", landing_page_business_suite_v1.PlanKind_PLAN_KIND_SUBSCRIPTION},
		{"credits", landing_page_business_suite_v1.PlanKind_PLAN_KIND_CREDITS_TOPUP},
		{"credits_topup", landing_page_business_suite_v1.PlanKind_PLAN_KIND_CREDITS_TOPUP},
		{"supporter", landing_page_business_suite_v1.PlanKind_PLAN_KIND_SUPPORTER_CONTRIBUTION},
		{"supporter_contribution", landing_page_business_suite_v1.PlanKind_PLAN_KIND_SUPPORTER_CONTRIBUTION},
		{"unknown", landing_page_business_suite_v1.PlanKind_PLAN_KIND_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapPlanKind(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizePlanOptionRejectsKindMismatch(t *testing.T) {
	plan := &PlanOption{
		StripePriceId:   "price_mismatch",
		PlanName:        "Credits Topup",
		PlanTier:        "credits",
		BillingInterval: landing_page_business_suite_v1.BillingInterval_BILLING_INTERVAL_MONTH,
		AmountCents:     5000,
		Currency:        "usd",
		DisplayWeight:   10,
		DisplayEnabled:  true,
		Kind:            landing_page_business_suite_v1.PlanKind_PLAN_KIND_SUBSCRIPTION,
	}

	err := normalizePlanOption(plan, "test_bundle")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "plan_kind")
}

func TestNormalizePlanOptionRejectsFreeNonZeroAmount(t *testing.T) {
	plan := &PlanOption{
		StripePriceId:   "price_free_mismatch",
		PlanName:        "Free Plan",
		PlanTier:        "free",
		BillingInterval: landing_page_business_suite_v1.BillingInterval_BILLING_INTERVAL_MONTH,
		AmountCents:     100,
		Currency:        "usd",
		DisplayWeight:   10,
		DisplayEnabled:  true,
		Kind:            planKindForTier("free"),
	}

	err := normalizePlanOption(plan, "test_bundle")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "free plan amount_cents")
}

func TestNormalizePlanOptionRejectsCreditsNonOneTime(t *testing.T) {
	plan := &PlanOption{
		StripePriceId:   "price_credits_monthly",
		PlanName:        "Credits Monthly",
		PlanTier:        "credits",
		BillingInterval: landing_page_business_suite_v1.BillingInterval_BILLING_INTERVAL_MONTH,
		AmountCents:     1000,
		Currency:        "usd",
		DisplayWeight:   10,
		DisplayEnabled:  true,
		Kind:            planKindForTier("credits"),
	}

	err := normalizePlanOption(plan, "test_bundle")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "credits plans must use one_time")
}

func TestApplyStripeImportSelections_OverwritesTier(t *testing.T) {
	tmpDir := t.TempDir()
	plansPath := filepath.Join(tmpDir, ".vrooli", "plans.json")

	ps := NewPlanStoreWithOptions(PlanStoreOptions{
		PlansPath:  plansPath,
		BundleKey:  "test_bundle",
		DisplayEnv: "test",
	})

	bundle := &BundleProduct{
		BundleKey:                "test_bundle",
		Name:                     "Test Bundle",
		StripeProductId:          "prod_test",
		CreditsPerUsd:            1_000_000,
		DisplayCreditsMultiplier: 1,
		DisplayCreditsLabel:      "credits",
		Environment:              "test",
	}

	existing := &PlanOption{
		StripePriceId:   "price_existing",
		PlanName:        "Solo Plan",
		PlanTier:        "solo",
		BillingInterval: landing_page_business_suite_v1.BillingInterval_BILLING_INTERVAL_MONTH,
		AmountCents:     1000,
		Currency:        "usd",
		DisplayWeight:   10,
		DisplayEnabled:  true,
		PlanRank:        planRankForTier("solo"),
		Kind:            planKindForTier("solo"),
	}

	require.NoError(t, ps.SetPlans(bundle, []*PlanOption{existing}))

	fetcher := func(ctx context.Context, priceID string) (*StripePriceImport, error) {
		return &StripePriceImport{
			PriceID:     priceID,
			LookupKey:   "pro_monthly",
			Currency:    "usd",
			AmountCents: 2000,
			Interval:    "month",
			ProductID:   "prod_test",
			ProductName: "Pro Monthly",
			Active:      true,
		}, nil
	}

	result, err := ps.ApplyStripeImportSelections(context.Background(), []ImportPlanSelection{
		{PriceID: "price_existing", Action: "overwrite"},
	}, fetcher)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Overwritten)

	updated, err := ps.GetPlanByPriceID("price_existing")
	require.NoError(t, err)
	assert.Equal(t, "pro", updated.PlanTier)
	assert.Equal(t, int64(2000), updated.AmountCents)
	assert.Equal(t, "Pro Monthly", updated.PlanName)
	assert.Equal(t, planKindForTier("pro"), updated.Kind)
}

func TestUpdatePlanWithStripeDetails_RejectsMismatchedProduct(t *testing.T) {
	tmpDir := t.TempDir()
	plansPath := filepath.Join(tmpDir, ".vrooli", "plans.json")

	ps := NewPlanStoreWithOptions(PlanStoreOptions{
		PlansPath:  plansPath,
		BundleKey:  "test_bundle",
		DisplayEnv: "test",
	})

	bundle := &BundleProduct{
		BundleKey:                "test_bundle",
		Name:                     "Test Bundle",
		StripeProductId:          "prod_bundle",
		CreditsPerUsd:            1_000_000,
		DisplayCreditsMultiplier: 1,
		DisplayCreditsLabel:      "credits",
		Environment:              "test",
	}

	existing := &PlanOption{
		StripePriceId:   "price_existing",
		PlanName:        "Existing Plan",
		PlanTier:        "pro",
		BillingInterval: landing_page_business_suite_v1.BillingInterval_BILLING_INTERVAL_MONTH,
		AmountCents:     1000,
		Currency:        "usd",
		DisplayWeight:   10,
		DisplayEnabled:  true,
		PlanRank:        planRankForTier("pro"),
		Kind:            planKindForTier("pro"),
	}

	require.NoError(t, ps.SetPlans(bundle, []*PlanOption{existing}))

	newName := "Updated Plan"
	_, err := ps.UpdatePlanWithStripeDetails("price_existing", UpdateBundlePriceInput{
		PlanName: &newName,
	}, &StripePriceImport{
		PriceID:     "price_existing",
		Currency:    "usd",
		AmountCents: 2000,
		Interval:    "month",
		ProductID:   "prod_other",
		ProductName: "Other Bundle",
		Active:      true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "product")
}

func TestUpdatePlanWithStripeDetails_RejectsFreeAmountMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	plansPath := filepath.Join(tmpDir, ".vrooli", "plans.json")

	ps := NewPlanStoreWithOptions(PlanStoreOptions{
		PlansPath:  plansPath,
		BundleKey:  "test_bundle",
		DisplayEnv: "test",
	})

	bundle := &BundleProduct{
		BundleKey:                "test_bundle",
		Name:                     "Test Bundle",
		StripeProductId:          "prod_bundle",
		CreditsPerUsd:            1_000_000,
		DisplayCreditsMultiplier: 1,
		DisplayCreditsLabel:      "credits",
		Environment:              "test",
	}

	freePlan := &PlanOption{
		StripePriceId:   "price_free",
		PlanName:        "Free Plan",
		PlanTier:        "free",
		BillingInterval: landing_page_business_suite_v1.BillingInterval_BILLING_INTERVAL_MONTH,
		AmountCents:     0,
		Currency:        "usd",
		DisplayWeight:   10,
		DisplayEnabled:  true,
		PlanRank:        planRankForTier("free"),
		Kind:            planKindForTier("free"),
	}

	require.NoError(t, ps.SetPlans(bundle, []*PlanOption{freePlan}))

	newName := "Updated Free"
	_, err := ps.UpdatePlanWithStripeDetails("price_free", UpdateBundlePriceInput{
		PlanName: &newName,
	}, &StripePriceImport{
		PriceID:     "price_free",
		Currency:    "usd",
		AmountCents: 500,
		Interval:    "month",
		ProductID:   "prod_bundle",
		ProductName: "Test Bundle",
		Active:      true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "free plan amount_cents")
}

func TestUpdatePlanWithStripeDetails_RejectsCreditsIntervalMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	plansPath := filepath.Join(tmpDir, ".vrooli", "plans.json")

	ps := NewPlanStoreWithOptions(PlanStoreOptions{
		PlansPath:  plansPath,
		BundleKey:  "test_bundle",
		DisplayEnv: "test",
	})

	bundle := &BundleProduct{
		BundleKey:                "test_bundle",
		Name:                     "Test Bundle",
		StripeProductId:          "prod_bundle",
		CreditsPerUsd:            1_000_000,
		DisplayCreditsMultiplier: 1,
		DisplayCreditsLabel:      "credits",
		Environment:              "test",
	}

	creditsPlan := &PlanOption{
		StripePriceId:   "price_credits",
		PlanName:        "Credits Pack",
		PlanTier:        "credits",
		BillingInterval: landing_page_business_suite_v1.BillingInterval_BILLING_INTERVAL_ONE_TIME,
		AmountCents:     1000,
		Currency:        "usd",
		DisplayWeight:   10,
		DisplayEnabled:  true,
		PlanRank:        planRankForTier("credits"),
		Kind:            planKindForTier("credits"),
	}

	require.NoError(t, ps.SetPlans(bundle, []*PlanOption{creditsPlan}))

	newName := "Updated Credits"
	_, err := ps.UpdatePlanWithStripeDetails("price_credits", UpdateBundlePriceInput{
		PlanName: &newName,
	}, &StripePriceImport{
		PriceID:     "price_credits",
		Currency:    "usd",
		AmountCents: 1000,
		Interval:    "month",
		ProductID:   "prod_bundle",
		ProductName: "Test Bundle",
		Active:      true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "credits plans must use one_time")
}

func TestApplyStripeImportSelections_RejectsMismatchedProduct(t *testing.T) {
	tmpDir := t.TempDir()
	plansPath := filepath.Join(tmpDir, ".vrooli", "plans.json")

	ps := NewPlanStoreWithOptions(PlanStoreOptions{
		PlansPath:  plansPath,
		BundleKey:  "test_bundle",
		DisplayEnv: "test",
	})

	bundle := &BundleProduct{
		BundleKey:                "test_bundle",
		Name:                     "Test Bundle",
		StripeProductId:          "prod_bundle",
		CreditsPerUsd:            1_000_000,
		DisplayCreditsMultiplier: 1,
		DisplayCreditsLabel:      "credits",
		Environment:              "test",
	}

	require.NoError(t, ps.SetPlans(bundle, []*PlanOption{}))

	fetcher := func(ctx context.Context, priceID string) (*StripePriceImport, error) {
		return &StripePriceImport{
			PriceID:     priceID,
			LookupKey:   "pro_monthly",
			Currency:    "usd",
			AmountCents: 2000,
			Interval:    "month",
			ProductID:   "prod_other",
			ProductName: "Other Bundle",
			Active:      true,
		}, nil
	}

	result, err := ps.ApplyStripeImportSelections(context.Background(), []ImportPlanSelection{
		{PriceID: "price_new", Action: "import"},
	}, fetcher)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Imported)
	assert.NotEmpty(t, result.Errors)

	plans := ps.GetPlans()
	assert.Len(t, plans, 0)
}

func TestIntroPricingTypeMapping(t *testing.T) {
	tests := []struct {
		input    string
		expected landing_page_business_suite_v1.IntroPricingType
	}{
		{"percentage", landing_page_business_suite_v1.IntroPricingType_INTRO_PRICING_TYPE_PERCENTAGE},
		{"percent", landing_page_business_suite_v1.IntroPricingType_INTRO_PRICING_TYPE_PERCENTAGE},
		{"pct", landing_page_business_suite_v1.IntroPricingType_INTRO_PRICING_TYPE_PERCENTAGE},
		{"flat_amount", landing_page_business_suite_v1.IntroPricingType_INTRO_PRICING_TYPE_FLAT_AMOUNT},
		{"flat-amount", landing_page_business_suite_v1.IntroPricingType_INTRO_PRICING_TYPE_FLAT_AMOUNT},
		{"flat", landing_page_business_suite_v1.IntroPricingType_INTRO_PRICING_TYPE_FLAT_AMOUNT},
		{"amount", landing_page_business_suite_v1.IntroPricingType_INTRO_PRICING_TYPE_FLAT_AMOUNT},
		{"unknown", landing_page_business_suite_v1.IntroPricingType_INTRO_PRICING_TYPE_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapIntroPricingTypeFromString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMetadataConversion(t *testing.T) {
	original := map[string]interface{}{
		"string": "value",
		"number": float64(42),
		"bool":   true,
		"nested": map[string]interface{}{"key": "val"},
		"list":   []interface{}{"a", "b"},
	}

	protoMap := convertMetadataToProto(original)
	assert.NotNil(t, protoMap)
	assert.Len(t, protoMap, 5)

	// Convert back
	result := convertProtoMetadataToMap(protoMap)
	assert.Equal(t, "value", result["string"])
	// Number can be returned as float64 or int64 depending on protobuf conversion
	numVal := result["number"]
	switch v := numVal.(type) {
	case float64:
		assert.Equal(t, float64(42), v)
	case int64:
		assert.Equal(t, int64(42), v)
	default:
		t.Errorf("unexpected number type: %T", numVal)
	}
	assert.Equal(t, true, result["bool"])
}

func TestMetadataConversion_Nil(t *testing.T) {
	assert.Nil(t, convertMetadataToProto(nil))
	assert.Nil(t, convertProtoMetadataToMap(nil))
}

func TestJsonValueHelpers(t *testing.T) {
	// String
	strVal := newStringJsonValue("hello")
	assert.NotNil(t, strVal)
	assert.Equal(t, "hello", strVal.GetStringValue())

	// Bool
	boolVal := newBoolJsonValue(true)
	assert.NotNil(t, boolVal)
	assert.True(t, boolVal.GetBoolValue())

	// List
	listVal := newListJsonValue([]*commonv1.JsonValue{
		newStringJsonValue("item1"),
		newStringJsonValue("item2"),
	})
	assert.NotNil(t, listVal)
	assert.Len(t, listVal.GetListValue().GetValues(), 2)
}

func TestResolvePlansPath(t *testing.T) {
	// This tests the resolution logic - the function looks for plans.json
	// in multiple locations and returns the first found or default
	path := ResolvePlansPath()
	assert.Contains(t, path, "plans.json")
}

func TestRoundTripSaveLoad(t *testing.T) {
	plansPath := setupTestPlansFile(t, testPlansJSON())

	// Load
	ps := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps.LoadAll())

	originalBundle := ps.GetBundle()
	originalPlans := ps.GetPlans()

	// Modify and save
	name := "Modified Plan"
	_, err := ps.UpdatePlan("price_monthly_pro", UpdateBundlePriceInput{PlanName: &name})
	require.NoError(t, err)

	// Load fresh
	ps2 := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps2.LoadAll())

	// Verify bundle persisted correctly
	loadedBundle := ps2.GetBundle()
	assert.Equal(t, originalBundle.BundleKey, loadedBundle.BundleKey)
	assert.Equal(t, originalBundle.Name, loadedBundle.Name)
	assert.Equal(t, originalBundle.CreditsPerUsd, loadedBundle.CreditsPerUsd)

	// Verify plan count
	assert.Len(t, ps2.GetPlans(), len(originalPlans))

	// Verify modification persisted
	plan, _ := ps2.GetPlanByPriceID("price_monthly_pro")
	assert.Equal(t, "Modified Plan", plan.PlanName)
}

func TestSaveLoadWithIntroFields(t *testing.T) {
	plansJSON := []byte(`{
  "bundle": {
    "bundle_key": "test_bundle",
    "name": "Test",
    "stripe_product_id": "prod_test",
    "credits_per_usd": 1000000,
    "display_credits_multiplier": 1.0,
    "display_credits_label": "credits"
  },
  "plans": [
    {
      "stripe_price_id": "price_intro",
      "plan_name": "Intro Plan",
      "plan_tier": "pro",
      "billing_interval": "month",
      "amount_cents": 4900,
      "currency": "usd",
      "intro_enabled": true,
      "intro_type": "percentage",
      "intro_amount_cents": 2450,
      "intro_periods": 3,
      "intro_price_lookup_key": "intro_key"
    }
  ]
}`)

	plansPath := setupTestPlansFile(t, plansJSON)
	ps := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps.LoadAll())

	plan, err := ps.GetPlanByPriceID("price_intro")
	require.NoError(t, err)

	assert.True(t, plan.IntroEnabled)
	assert.Equal(t, landing_page_business_suite_v1.IntroPricingType_INTRO_PRICING_TYPE_PERCENTAGE, plan.IntroType)
	assert.NotNil(t, plan.IntroAmountCents)
	assert.Equal(t, int64(2450), *plan.IntroAmountCents)
	assert.Equal(t, int32(3), plan.IntroPeriods)
	assert.Equal(t, "intro_key", plan.IntroPriceLookupKey)

	// Save and reload
	require.NoError(t, ps.SavePlans())

	ps2 := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps2.LoadAll())

	plan2, _ := ps2.GetPlanByPriceID("price_intro")
	assert.True(t, plan2.IntroEnabled)
	assert.NotNil(t, plan2.IntroAmountCents)
	assert.Equal(t, int64(2450), *plan2.IntroAmountCents)
}

func TestFileFormat_UpdatedAt(t *testing.T) {
	plansPath := setupTestPlansFile(t, testPlansJSON())
	ps := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: "test_bundle"})
	require.NoError(t, ps.LoadAll())

	// Save to update the timestamp
	require.NoError(t, ps.SavePlans())

	// Read file directly to check updated_at
	data, err := os.ReadFile(plansPath)
	require.NoError(t, err)

	var fileData plansFileFormat
	require.NoError(t, json.Unmarshal(data, &fileData))

	assert.NotEmpty(t, fileData.UpdatedAt)
}
