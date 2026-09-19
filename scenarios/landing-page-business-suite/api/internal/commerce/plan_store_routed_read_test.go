package commerce

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/storage"
)

func TestGetPricingOverviewForBundleReadsLeasedClassConfigSnapshot(t *testing.T) {
	primaryDir := t.TempDir()
	primaryPath := writePlansSnapshot(t, primaryDir, "price_primary")
	testDir := t.TempDir()
	writePlansSnapshot(t, testDir, "price_test")

	roots := filerouting.New(storage.Paths{ConfigDir: primaryDir})
	store := NewPlanStoreWithOptions(PlanStoreOptions{
		PlansPath:      primaryPath,
		BundleKey:      "signal-studio",
		DisplayEnv:     "production",
		ConfigFileName: "plans.json",
	})
	if err := store.LoadAll(); err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if err := roots.InstallTestRoots(storage.Paths{ConfigDir: testDir}, "pricing-lease", time.Minute); err != nil {
		t.Fatalf("InstallTestRoots: %v", err)
	}

	service := NewPlanServiceWithPlanStore(store)
	service.SetFileRoots(roots)
	overview, err := service.GetPricingOverviewForBundle(database.WithTestMode(context.Background()), "signal-studio")
	if err != nil {
		t.Fatalf("GetPricingOverviewForBundle: %v", err)
	}
	if len(overview.Monthly) != 1 || overview.Monthly[0].StripePriceId != "price_test" {
		t.Fatalf("overview = %#v, want leased ClassConfig price_test", overview)
	}
}

func TestGetPricingOverviewForBundleNeverFallsBackToPrimaryInTestMode(t *testing.T) {
	primaryDir := t.TempDir()
	primaryPath := writePlansSnapshot(t, primaryDir, "price_primary")
	missingTestDir := t.TempDir()
	roots := filerouting.New(storage.Paths{ConfigDir: primaryDir})
	store := NewPlanStoreWithOptions(PlanStoreOptions{
		PlansPath:   primaryPath,
		BundleKey:   "signal-studio",
		DisplayEnv:  "production",
		ConfigRoots: roots,
	})
	if err := store.LoadAll(); err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if err := roots.InstallTestRoots(storage.Paths{ConfigDir: missingTestDir}, "pricing-lease", time.Minute); err != nil {
		t.Fatalf("InstallTestRoots: %v", err)
	}

	_, err := store.GetPricingOverviewForBundle(database.WithTestMode(context.Background()), "signal-studio")
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("error = %v, want missing leased snapshot error", err)
	}
}

func TestGetPricingOverviewForBundleRequiresExactBundle(t *testing.T) {
	path := writePlansSnapshot(t, t.TempDir(), "price_primary")
	store := NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: path, BundleKey: "signal-studio", DisplayEnv: "production"})
	if err := store.LoadAll(); err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if _, err := store.GetPricingOverviewForBundle(context.Background(), "other-bundle"); err == nil {
		t.Fatal("expected exact bundle mismatch")
	}
}

func TestGetPricingOverviewForBundleProductionUsesOwnerLoadedCatalog(t *testing.T) {
	primaryDir := t.TempDir()
	primaryPath := writePlansSnapshot(t, primaryDir, "price_primary")
	testDir := t.TempDir()
	writePlansSnapshot(t, testDir, "price_test")
	roots := filerouting.New(storage.Paths{ConfigDir: primaryDir})
	store := NewPlanStoreWithOptions(PlanStoreOptions{
		PlansPath:   primaryPath,
		BundleKey:   "signal-studio",
		DisplayEnv:  "production",
		ConfigRoots: roots,
	})
	if err := store.LoadAll(); err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if err := roots.InstallTestRoots(storage.Paths{ConfigDir: testDir}, "pricing-lease", time.Minute); err != nil {
		t.Fatalf("InstallTestRoots: %v", err)
	}
	overview, err := store.GetPricingOverviewForBundle(context.Background(), "signal-studio")
	if err != nil {
		t.Fatalf("GetPricingOverviewForBundle: %v", err)
	}
	if len(overview.Monthly) != 1 || overview.Monthly[0].StripePriceId != "price_primary" {
		t.Fatalf("overview = %#v, want owner-loaded primary price", overview)
	}
}

func TestGetPricingOverviewForBundleRejectsNilContext(t *testing.T) {
	store := NewPlanStoreWithOptions(PlanStoreOptions{BundleKey: "signal-studio"})
	if _, err := store.GetPricingOverviewForBundle(nil, "signal-studio"); err == nil {
		t.Fatal("expected nil context error")
	}
}

func TestGetPricingOverviewForBundleRequiresActiveLease(t *testing.T) {
	roots := filerouting.New(storage.Paths{ConfigDir: t.TempDir()})
	store := NewPlanStoreWithOptions(PlanStoreOptions{BundleKey: "signal-studio", ConfigRoots: roots})
	_, err := store.GetPricingOverviewForBundle(database.WithTestMode(context.Background()), "signal-studio")
	if err == nil {
		t.Fatal("expected absent lease error")
	}

	clock := schedule.NewFake(time.Unix(100, 0))
	roots.SetClock(clock)
	if err := roots.InstallTestRoots(storage.Paths{ConfigDir: t.TempDir()}, "expired-lease", time.Minute); err != nil {
		t.Fatalf("InstallTestRoots: %v", err)
	}
	clock.Advance(2 * time.Minute)
	_, err = store.GetPricingOverviewForBundle(database.WithTestMode(context.Background()), "signal-studio")
	if err == nil {
		t.Fatal("expected expired lease error")
	}
}

func TestGetPricingOverviewForBundleRejectsUnsafeConfigFileName(t *testing.T) {
	for _, name := range []string{"../plans.json", "/tmp/plans.json", "foo/../../plans.json"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			roots := filerouting.New(storage.Paths{ConfigDir: root})
			if err := roots.InstallTestRoots(storage.Paths{ConfigDir: root}, "unsafe-name", time.Minute); err != nil {
				t.Fatalf("InstallTestRoots: %v", err)
			}
			store := NewPlanStoreWithOptions(PlanStoreOptions{BundleKey: "signal-studio", ConfigRoots: roots, ConfigFileName: name})
			_, err := store.GetPricingOverviewForBundle(database.WithTestMode(context.Background()), "signal-studio")
			if err == nil {
				t.Fatal("expected unsafe filename error")
			}
		})
	}
}

func TestGetPricingOverviewForBundleRejectsSymlinkOutsideRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on Windows")
	}
	root := t.TempDir()
	outside := writePlansSnapshot(t, t.TempDir(), "price_outside")
	if err := os.Symlink(outside, filepath.Join(root, "plans.json")); err != nil {
		t.Fatalf("Symlink: %v", err)
	}
	roots := filerouting.New(storage.Paths{ConfigDir: t.TempDir()})
	if err := roots.InstallTestRoots(storage.Paths{ConfigDir: root}, "symlink-lease", time.Minute); err != nil {
		t.Fatalf("InstallTestRoots: %v", err)
	}
	store := NewPlanStoreWithOptions(PlanStoreOptions{BundleKey: "signal-studio", ConfigRoots: roots})
	_, err := store.GetPricingOverviewForBundle(database.WithTestMode(context.Background()), "signal-studio")
	if err == nil {
		t.Fatal("expected symlink escape rejection")
	}
}

func writePlansSnapshot(t *testing.T, dir, priceID string) string {
	t.Helper()
	path := filepath.Join(dir, "plans.json")
	data, err := json.Marshal(plansFileFormat{
		Bundle: testBundle("signal-studio", "production"),
		Plans: []planFileFormat{{
			StripePriceID:   priceID,
			PlanName:        "Studio",
			PlanTier:        "studio",
			BillingInterval: "month",
			AmountCents:     1200,
			Currency:        "usd",
			DisplayWeight:   1,
			DisplayEnabled:  true,
		}},
	})
	if err != nil {
		t.Fatalf("marshal plans: %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write plans: %v", err)
	}
	return path
}
