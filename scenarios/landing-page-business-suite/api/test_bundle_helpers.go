package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"landing-page-business-suite-api/internal/envx"
)

const (
	minInt32 = -1 << 31
	maxInt32 = 1<<31 - 1
)

func requireInt32(t *testing.T, field string, value int) int32 {
	t.Helper()
	if value < minInt32 || value > maxInt32 {
		t.Fatalf("%s=%d is outside int32 range", field, value)
	}
	// #nosec G115 -- the bounds check immediately above proves this conversion is safe.
	return int32(value)
}

// globalTestPlanStore holds the plan store used by test helpers.
// This is needed because the old test pattern used database inserts,
// but now we use file-based storage.
var (
	globalTestPlanStore     *PlanStore
	globalTestPlanStoreMu   sync.Mutex
	globalTestPlansPath     string
	globalTestBundleCounter int
)

// configureTestBundleEnv sets up environment variables for a test bundle.
// Returns the generated bundle key based on the test name.
func configureTestBundleEnv(t *testing.T, env string) string {
	t.Helper()

	replacer := strings.NewReplacer("/", "_", ".", "_")
	bundleKey := fmt.Sprintf("bundle_%s", replacer.Replace(strings.ToLower(t.Name())))
	prevKey := envx.Get("BUNDLE_KEY")
	prevEnv := envx.Get("BUNDLE_ENVIRONMENT")

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

// setEnvOrClear sets an environment variable or clears it if value is empty.
func setEnvOrClear(key, value string) {
	if value == "" {
		_ = os.Unsetenv(key)
		return
	}
	_ = os.Setenv(key, value)
}

// upsertTestBundleProduct ensures a bundle product exists for the specified bundle key/environment.
// This is a compatibility shim that works with the file-based PlanStore.
// Note: The db parameter is ignored as plans are now file-based.
func upsertTestBundleProduct(
	t *testing.T,
	db *sql.DB,
	bundleKey, bundleName, stripeProductID, environment string,
	creditsPerUSD int64,
	displayMultiplier float64,
	displayLabel string,
) int64 {
	t.Helper()

	globalTestPlanStoreMu.Lock()
	defer globalTestPlanStoreMu.Unlock()

	// Create a fresh plans file for this test
	tmpDir := t.TempDir()
	plansPath := filepath.Join(tmpDir, ".vrooli", "plans.json")
	if err := os.MkdirAll(filepath.Dir(plansPath), 0o750); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}

	fileData := plansFileFormat{
		Bundle: bundleFileFormat{
			BundleKey:                bundleKey,
			Name:                     bundleName,
			StripeProductID:          stripeProductID,
			CreditsPerUSD:            creditsPerUSD,
			DisplayCreditsMultiplier: displayMultiplier,
			DisplayCreditsLabel:      displayLabel,
			Environment:              environment,
		},
		Plans: []planFileFormat{},
	}

	data, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal plans: %v", err)
	}

	if err := os.WriteFile(plansPath, data, 0o600); err != nil {
		t.Fatalf("failed to write plans file: %v", err)
	}

	globalTestPlansPath = plansPath
	globalTestPlanStore = NewPlanStoreWithOptions(PlanStoreOptions{
		PlansPath:  plansPath,
		BundleKey:  bundleKey,
		DisplayEnv: environment,
	})
	if err := globalTestPlanStore.LoadAll(); err != nil {
		t.Fatalf("failed to load plans: %v", err)
	}

	globalTestBundleCounter++
	return int64(globalTestBundleCounter)
}

// insertBundlePrice stores a pricing tier connected to the given bundle product.
// This is a compatibility shim that works with the file-based PlanStore.
// Note: The db and productID parameters are ignored as plans are now file-based.
func insertBundlePrice(
	t *testing.T,
	db *sql.DB,
	productID int64,
	priceID, planName, planTier, billingInterval, currency string,
	amountCents int,
	introEnabled bool,
	introType string,
	introAmountCents int,
	introPeriods int,
	introLookupKey string,
	monthlyIncluded, oneTimeBonus int,
	planRank, displayWeight int,
	bonusType, kind string,
	metadata map[string]interface{},
) {
	t.Helper()

	globalTestPlanStoreMu.Lock()
	defer globalTestPlanStoreMu.Unlock()

	if globalTestPlanStore == nil {
		t.Fatal("must call upsertTestBundleProduct before insertBundlePrice")
	}

	// Read current file
	// #nosec G304 -- test setup creates globalTestPlansPath in t.TempDir.
	data, err := os.ReadFile(globalTestPlansPath)
	if err != nil {
		t.Fatalf("failed to read plans file: %v", err)
	}

	var fileData plansFileFormat
	if err := json.Unmarshal(data, &fileData); err != nil {
		t.Fatalf("failed to parse plans file: %v", err)
	}

	// Add the new plan
	plan := planFileFormat{
		StripePriceID:          priceID,
		PlanName:               planName,
		PlanTier:               planTier,
		BillingInterval:        billingInterval,
		AmountCents:            int64(amountCents),
		Currency:               currency,
		DisplayWeight:          requireInt32(t, "display weight", displayWeight),
		DisplayEnabled:         true,
		MonthlyIncludedCredits: int64(monthlyIncluded),
		OneTimeBonusCredits:    int64(oneTimeBonus),
		PlanRank:               requireInt32(t, "plan rank", planRank),
		BonusType:              bonusType,
		Kind:                   kind,
		IntroEnabled:           introEnabled,
		IntroPeriods:           requireInt32(t, "intro periods", introPeriods),
		IntroPriceLookupKey:    introLookupKey,
		Metadata:               metadata,
	}

	if introEnabled && introType != "" && introType != "none" {
		plan.IntroType = introType
		if introAmountCents > 0 {
			cents := int64(introAmountCents)
			plan.IntroAmountCents = &cents
		}
	}

	fileData.Plans = append(fileData.Plans, plan)

	// Write back
	newData, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal plans: %v", err)
	}

	if err := os.WriteFile(globalTestPlansPath, newData, 0o600); err != nil {
		t.Fatalf("failed to write plans file: %v", err)
	}

	// Reload the plan store
	if err := globalTestPlanStore.LoadAll(); err != nil {
		t.Fatalf("failed to reload plans: %v", err)
	}
}

// cleanupBundleProductRecords removes all data tied to a bundle product.
// This is a no-op in the file-based approach as temp dirs are cleaned up automatically.
func cleanupBundleProductRecords(t *testing.T, db *sql.DB, productID int64) {
	t.Helper()
	// No-op: temp directories are automatically cleaned up by t.TempDir()
}

// getTestPlanStore returns the global test plan store.
// This can be used by tests that need direct access to the plan store.
func getTestPlanStore() *PlanStore {
	globalTestPlanStoreMu.Lock()
	defer globalTestPlanStoreMu.Unlock()
	return globalTestPlanStore
}

// requireTestPlanService returns a PlanService backed by the test plan store.
func requireTestPlanService(t *testing.T) *PlanService {
	t.Helper()
	planStore := getTestPlanStore()
	if planStore == nil {
		t.Fatal("test plan store not initialized; call upsertTestBundleProduct first")
	}
	return NewPlanServiceWithPlanStore(planStore)
}

// requireTestStripeService returns a StripeService wired to the test plan store
// and a PaymentAnomalyService so logIntroAnomaly forwards through the unified
// payment_anomaly_log pipeline.
type StripeTestStore interface {
	StripeServiceStore
	PaymentAnomalyStore
}

func requireTestStripeService(t *testing.T, db StripeTestStore) *StripeService {
	t.Helper()
	svc := NewStripeServiceWithSettings(db, requireTestPlanService(t), NewPaymentSettingsService(db))
	anomaly := NewPaymentAnomalyService(context.Background(), db, context.Background())
	svc.SetPaymentAnomaly(anomaly)
	return svc
}
