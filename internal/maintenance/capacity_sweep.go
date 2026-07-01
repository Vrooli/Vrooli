package maintenance

import (
	"context"
	"fmt"
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
	GCTerminalClaims(ctx context.Context, olderThan time.Time) (capacity.GCResult, error)
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
	// capacityExecFn resolves the degrade actuator for autonomous idle-unload.
	// Production shells the owner's resource CLI; idle-unload only actuates under
	// enforce=on (advisory just logs the would-unload), so this is dormant by
	// default. Tests override it so no real exec runs.
	capacityExecFn    = func() capacity.ApplyExecutor { return capacity.DefaultExecutor() }
	capacityEnforceFn = func(policy capacity.Policy) string {
		return policy.EffectiveEnforce(os.Getenv(capacity.EnvEnforce))
	}
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
	now := capacityNowFn()
	result, err := capacitySweepFn(ctx, store, snapshot, capacityAttributorFn(), policy, now)
	if err != nil {
		return nil, err
	}
	items := make([]control.ResultItem, 0, len(result.Expired)+1)
	for _, claim := range result.Expired {
		items = append(items, control.Stopped(claim.OwnerID, "Expired stale capacity claim "+claim.ClaimID))
	}
	// Autonomous idle-unload (plan §Phase 3 keystone): proactively free VRAM from
	// claims idle beyond their idle_unload_ttl, accepting a cold start on next use.
	// Gated by enforce mode — advisory LOGS the would-unload (no actuation),
	// enforce=on actuates through the degrade path (debounce-respected, fail-open).
	// GC + sampling above stay safe regardless.
	enforce := capacityEnforceFn(policy)
	if active, listErr := store.ListClaims(ctx, capacity.ClaimFilter{Statuses: capacity.ActiveClaimStatuses()}); listErr == nil {
		plan, _, _ := capacity.RunIdleUnload(ctx, store, active, capacityExecFn(), policy, enforce, now)
		for _, a := range plan.Actions {
			if a.Action != capacity.ActionRequestDegrade {
				continue
			}
			verb := "would idle-unload (advisory)"
			if enforce == capacity.EnforceOn {
				verb = "idle-unloaded"
			}
			items = append(items, control.Stopped(a.OwnerID, fmt.Sprintf("%s claim %s to %q", verb, a.ClaimID, a.ToStep)))
		}

		// Capacity upshift (§8.8 hysteresis): a claim degraded under earlier GPU
		// pressure should climb back toward its preferred size once the contending
		// consumer frees VRAM and the claim is idle, so it is ready before its owner
		// is next used. Symmetric to idle-unload and gated identically — advisory
		// surfaces the would-upshift recommendation (non-acting), enforce=on actuates
		// through the adopter's resize verb (--upshift). Reuses the snapshot the
		// sweep already collected for per-GPU free-headroom accounting.
		upPlan, _, _ := capacity.RunUpshift(ctx, store, active, snapshot, capacityExecFn(), policy, enforce, now)
		for _, a := range upPlan.Actions {
			if a.Action != capacity.ActionRequestUpshift {
				continue
			}
			verb := "would upshift (advisory)"
			if enforce == capacity.EnforceOn {
				verb = "upshifted"
			}
			items = append(items, control.Started(a.OwnerID, fmt.Sprintf("%s claim %s to %q", verb, a.ClaimID, a.ToStep)))
		}
	}

	// Claim-on-observe adoption (plan §Phase 6): an observed GPU consumer whose
	// resource.json declares a capacity block but holds NO active claim (a resident
	// that predates the admission hook, e.g. kyutai-stt / speaker-verification) is
	// adopted into the ledger from its declared spec — idempotent, declared-only,
	// advisory-safe. The SpecLoader reads resource.json under the source root.
	loadSpec := func(name string) (capacity.ResourceClaimSpec, bool, error) {
		return capacity.LoadResourceClaimSpec(c.Root, name)
	}
	if adopted, adoptErr := capacity.AdoptObservedResidents(ctx, store, snapshot, capacityAttributorFn(), policy, loadSpec, now); adoptErr == nil {
		for _, cl := range adopted {
			items = append(items, control.Started(cl.OwnerID, "Adopted declared resident into capacity ledger ("+cl.ClaimID+")"))
		}
	}

	// Terminal-claim GC (plan §Phase 1): prune released/expired/preempted rows
	// past the retention window so the ledger does not accumulate dead history.
	// Always safe (frees no live capacity); disabled when retention <= 0.
	if policy.TerminalRetention > 0 {
		if gc, gcErr := store.GCTerminalClaims(ctx, now.Add(-policy.TerminalRetention)); gcErr == nil && gc.Count > 0 {
			items = append(items, control.Stopped("capacity", fmt.Sprintf("Pruned %d terminal capacity claim(s) past retention", gc.Count)))
		}
	}
	return items, nil
}
