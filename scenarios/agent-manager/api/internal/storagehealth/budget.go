package storagehealth

import (
	"fmt"
	"sync"
	"time"

	"github.com/vrooli/api-core/retention"
	corestorage "github.com/vrooli/api-core/storage"
)

// declaredBudgetRefresh bounds how stale a cached declaration may be; the
// manifest changes only by review, so an hour is ample.
const declaredBudgetRefresh = time.Hour

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

// DeclaredBudget caches LoadDeclaredBudget. A failed read reports an
// unbudgeted file rather than failing measurement.
func DeclaredBudget(repoRoot, owner, entry string, clock func() time.Time) func() Budget {
	if clock == nil {
		clock = time.Now
	}
	var (
		mu      sync.Mutex
		cached  Budget
		fetched time.Time
	)
	return func() Budget {
		mu.Lock()
		defer mu.Unlock()
		if !fetched.IsZero() && clock().Sub(fetched) < declaredBudgetRefresh {
			return cached
		}
		fetched = clock()
		if budget, err := LoadDeclaredBudget(repoRoot, owner, entry); err == nil {
			cached = budget
		} else {
			cached = Budget{Source: "unavailable: " + err.Error()}
		}
		return cached
	}
}
