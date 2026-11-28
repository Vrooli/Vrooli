package storagehealth

import (
	"fmt"
	"sync"
	"time"

	"github.com/vrooli/api-core/retention"
	corestorage "github.com/vrooli/api-core/storage"
)

// Successful owner manifests change only by review, so this hourly cache is
// ample. Errors use a shorter retry because an inventory read may recover.
const (
	declaredBudgetRefresh      = time.Hour
	declaredBudgetFailureRetry = time.Minute
)

// LoadDeclaredBudget reads the max_bytes budget an owner declares for one
// storage entry through api-core's owner inventory, the same parse
// storage-manager enforces, so the thresholds never follow a second number.
func LoadDeclaredBudget(repoRoot, owner, entry string) (Budget, error) {
	inventory, err := corestorage.LoadOwnerInventory(corestorage.InventoryOptions{RepoRoot: repoRoot})
	if err != nil {
		return Budget{}, err
	}
	for _, manifest := range inventory.Owners {
		if manifest.Kind != corestorage.OwnerScenario || manifest.ID != owner {
			continue
		}
		for _, declared := range manifest.StorageEntries {
			if declared.Name != entry {
				continue
			}
			if declared.Budget == nil || declared.Budget.MaxBytes == "" {
				return Budget{}, fmt.Errorf("%s storage entry %q declares no max_bytes budget", owner, entry)
			}
			bytes, err := retention.ParseBytes(declared.Budget.MaxBytes)
			if err != nil {
				return Budget{}, fmt.Errorf("%s storage entry %q budget: %w", owner, entry, err)
			}
			return Budget{Bytes: bytes, Source: manifest.ManifestPath + "#storage.entries." + entry + ".budget.max_bytes"}, nil
		}
		return Budget{}, fmt.Errorf("%s declares no storage entry %q", owner, entry)
	}
	return Budget{}, fmt.Errorf("no scenario manifest for %s under %s", owner, repoRoot)
}

// DeclaredBudget caches successful declarations for an hour. A failed read
// reports an unbudgeted file rather than failing measurement, then retries
// after a short interval so a transient inventory failure does not hide the
// budget for the full success-cache lifetime.
func DeclaredBudget(repoRoot, owner, entry string, clock func() time.Time) func() Budget {
	if clock == nil {
		clock = time.Now
	}
	var (
		mu           sync.Mutex
		cached       Budget
		fetched      time.Time
		refreshAfter time.Duration
	)
	return func() Budget {
		mu.Lock()
		defer mu.Unlock()
		now := clock()
		if !fetched.IsZero() && now.Sub(fetched) < refreshAfter {
			return cached
		}
		fetched = now
		if budget, err := LoadDeclaredBudget(repoRoot, owner, entry); err == nil {
			cached, refreshAfter = budget, declaredBudgetRefresh
		} else {
			cached, refreshAfter = Budget{Source: "unavailable: " + err.Error()}, declaredBudgetFailureRetry
		}
		return cached
	}
}
