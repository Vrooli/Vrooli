package maintenance

import (
	"context"
	"os"
	"time"

	"github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/hostinventory"
)

// capacitySweepStore is the ledger surface the maintenance-pass sweep needs: the
// claim repository (for Sweep) plus the policy repository (for the cadence/TTL
// levers). It is satisfied by *capacity.SQLiteStore.
type capacitySweepStore interface {
	capacity.ClaimRepository
	capacity.PolicyRepository
	Close() error
}

// Injectable seams (mirror the existing fn-var pattern in maintenance.go) so
// unit runs never open the real ledger DB, shell out to nvidia-smi, or touch
// docker. Production resolves the SQLite ledger + live host snapshot + docker
// attributor.
var (
	openCapacityStoreFn = openCapacityStoreIfPresent
	capacitySnapshotFn  = func(ctx context.Context) (hostinventory.Snapshot, error) {
		return capacity.HostInventorySource{}.Snapshot(ctx)
	}
	capacityAttributorFn = func() capacity.Attributor { return capacity.NewDockerAttributor() }
	capacitySweepFn      = capacity.Sweep
	capacityNowFn        = func() time.Time { return time.Now().UTC() }
)

// openCapacityStoreIfPresent opens the capacity ledger ONLY when it already
// exists, mirroring openRuntimeRegistryIfPresent: the maintenance pass must not
// materialize an empty ledger before any adopter has ever claimed. A nil store
// (with nil error) means "no ledger yet — nothing to sweep".
func openCapacityStoreIfPresent(ctx context.Context, home string) (capacitySweepStore, error) {
	path, err := capacity.DefaultDBPath(home)
	if err != nil {
		return nil, err
	}
	if _, statErr := os.Stat(path); statErr != nil {
		if os.IsNotExist(statErr) {
			return nil, nil
		}
		return nil, statErr
	}
	return capacity.NewSQLiteStore(ctx, capacity.Config{HomeDir: home})
}

// sweepCapacityClaims runs the always-on resident-claim sweep (capacity §8.6)
// from the maintenance pass: resident GPU claims still observed holding memory
// are heartbeat-refreshed (so a third-party model-server container that cannot
// heartbeat itself is not falsely expired), and dead ones past their deadline
// are expired. A sensing failure is a clean no-op — the sweep must never expire
// a claim it cannot verify against the GPU, or a transient nvidia-smi hiccup
// would strand a live resident; the next pass recovers. Expired claims are
// surfaced as StopReport items so the cleanup is observable (no silent caps).
func (c *Controller) sweepCapacityClaims(ctx context.Context) ([]control.ResultItem, error) {
	store, err := openCapacityStoreFn(ctx, c.Home)
	if err != nil || store == nil {
		return nil, err
	}
	defer store.Close()

	policy, err := store.GetPolicy(ctx)
	if err != nil {
		return nil, err
	}
	snapshot, err := capacitySnapshotFn(ctx)
	if err != nil {
		// Sensing unavailable: skip the sweep entirely (never expire unverified).
		return nil, nil
	}
	result, err := capacitySweepFn(ctx, store, snapshot, capacityAttributorFn(), policy, capacityNowFn())
	if err != nil {
		return nil, err
	}
	items := make([]control.ResultItem, 0, len(result.Expired))
	for _, claim := range result.Expired {
		items = append(items, control.Stopped(claim.OwnerID, "Expired stale capacity claim "+claim.ClaimID))
	}
	return items, nil
}
