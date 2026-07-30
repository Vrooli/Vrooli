package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"landing-page-business-suite-api/internal/commerce"
)

type (
	plansFileFormat  = commerce.PlansFileFormat
	bundleFileFormat = commerce.BundleFileFormat
	planFileFormat   = commerce.PlanFileFormat
)

func createTestPlansFile(t *testing.T, bundle bundleFileFormat, plans []planFileFormat) string {
	t.Helper()
	plansPath := filepath.Join(t.TempDir(), ".vrooli", "plans.json")
	if err := os.MkdirAll(filepath.Dir(plansPath), 0o755); err != nil {
		t.Fatalf("create plans directory: %v", err)
	}
	data, err := json.MarshalIndent(plansFileFormat{Bundle: bundle, Plans: plans}, "", "  ")
	if err != nil {
		t.Fatalf("marshal test plans: %v", err)
	}
	if err := os.WriteFile(plansPath, data, 0o644); err != nil {
		t.Fatalf("write test plans: %v", err)
	}
	return plansPath
}

func createTestPlanService(t *testing.T, bundle bundleFileFormat, plans []planFileFormat) *commerce.PlanService {
	t.Helper()
	store := NewPlanStoreWithOptions(commerce.PlanStoreOptions{
		PlansPath:  createTestPlansFile(t, bundle, plans),
		BundleKey:  bundle.BundleKey,
		DisplayEnv: bundle.Environment,
	})
	if err := store.LoadAll(); err != nil {
		t.Fatalf("load test plans: %v", err)
	}
	return NewPlanServiceWithOptions(commerce.PlanServiceOptions{PlanStore: store, DefaultBundle: bundle.BundleKey, DisplayEnv: bundle.Environment})
}

func testBundle(key, env string) bundleFileFormat {
	return bundleFileFormat{
		BundleKey: key, Name: "Test Bundle", StripeProductID: "prod_test",
		CreditsPerUSD: 1_000_000, DisplayCreditsMultiplier: 0.01,
		DisplayCreditsLabel: "credits", Environment: env,
	}
}
