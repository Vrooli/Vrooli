package maintenance

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/clock"
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
	capacityNowFn        = clock.Real{}.Now
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
func (c *Controller) sweepCapacityClaims(ctx context.Context) (control.StopReport, error) {
	store, err := openCapacityStoreFn(ctx, c.Home)
	if err != nil || store == nil {
		return control.StopReport{}, err
	}
	defer store.Close()

	policy, err := store.GetPolicy(ctx)
	if err != nil {
		return control.StopReport{}, err
	}
	snapshot, err := capacitySnapshotFn(ctx)
	if err != nil {
		// Sensing unavailable: skip the sweep entirely (never expire unverified).
		return control.StopReport{}, nil
	}
	now := capacityNowFn()
	result, err := capacitySweepFn(ctx, store, snapshot, capacityAttributorFn(), policy, now)
	if err != nil {
		return control.StopReport{}, err
	}
	report := control.StopReport{}
	for _, claim := range result.Expired {
		report.Observed = append(report.Observed, control.ResultItem{Name: claim.OwnerID, Message: "Expired stale capacity claim " + claim.ClaimID})
	}
	// Autonomous idle-unload (plan §Phase 3 keystone): proactively free VRAM from
	// claims idle beyond their idle_unload_ttl, accepting a cold start on next use.
	// Gated by enforce mode — advisory LOGS the would-unload (no actuation),
	// enforce=on actuates through the degrade path (debounce-respected, fail-open).
	// GC + sampling above stay safe regardless.
	enforce := capacityEnforceFn(policy)
	active, listErr := store.ListClaims(ctx, capacity.ClaimFilter{Statuses: capacity.ActiveClaimStatuses()})
	if listErr != nil {
		report.Failed = append(report.Failed, control.Failed("capacity-actions", listErr))
	} else {
		plan, outcomes, runErr := capacity.RunIdleUnload(ctx, store, active, capacityExecFn(), policy, enforce, now)
		appendCapacityOutcomes(&report, plan, outcomes, runErr, enforce, "idle-unload", true)
		// Preserve the pre-action cohort: an unload must not immediately plan its
		// own reversal. Actuation still validates every captured generation.
		upPlan, upOutcomes, upErr := capacity.RunUpshift(ctx, store, active, snapshot, capacityExecFn(), policy, enforce, now)
		appendCapacityOutcomes(&report, upPlan, upOutcomes, upErr, enforce, "upshift", false)
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
			report.Observed = append(report.Observed, control.ResultItem{Name: cl.OwnerID, Message: "Adopted declared resident into capacity ledger (" + cl.ClaimID + ")"})
		}
	}

	// Terminal-claim GC (plan §Phase 1): prune released/expired/preempted rows
	// past the retention window so the ledger does not accumulate dead history.
	// Always safe (frees no live capacity); disabled when retention <= 0.
	if policy.TerminalRetention > 0 {
		if gc, gcErr := store.GCTerminalClaims(ctx, now.Add(-policy.TerminalRetention)); gcErr == nil && gc.Count > 0 {
			report.Observed = append(report.Observed, control.ResultItem{Name: "capacity", Message: fmt.Sprintf("Pruned %d terminal capacity claim(s) past retention", gc.Count)})
		}
	}
	// A ledger that stays large after a prune is not a retention problem, it is
	// a churn problem: the surviving rows are younger than the retention window
	// because something is creating one per state change. Retention alone will
	// never catch up with that, so say so rather than pruning quietly forever.
	if warning := ledgerSizeWarning(ctx, store); warning != "" {
		report.Observed = append(report.Observed, control.ResultItem{Name: "capacity", Message: warning})
	}
	return report, nil
}

// appendCapacityOutcomes distinguishes a proposal from a completed callback.
// Upshift and ledger changes are observations, never stopped resources.
func appendCapacityOutcomes(report *control.StopReport, plan capacity.EscalationPlan, result capacity.ActuationResult, runErr error, enforce, verb string, stopEffect bool) {
	for _, action := range plan.Actions {
		context := fmt.Sprintf("%s claim %s to %q", verb, action.ClaimID, action.ToStep)
		if enforce != capacity.EnforceOn {
			report.Observed = append(report.Observed, control.ResultItem{Name: action.OwnerID, Message: "would " + context + " (" + enforce + ")"})
			continue
		}
		var outcome *capacity.ActuationOutcome
		for i := range result.Outcomes {
			o := &result.Outcomes[i]
			if o.ClaimID == action.ClaimID && o.Action == action.Action && o.ToStep == action.ToStep {
				outcome = o
				break
			}
		}
		if outcome == nil {
			cause := runErr
			if cause == nil {
				cause = fmt.Errorf("actuator returned no outcome")
			}
			report.Failed = append(report.Failed, control.Failed(action.OwnerID, fmt.Errorf("%s not completed: %w", context, cause)))
			continue
		}
		if outcome.Err != "" {
			report.Failed = append(report.Failed, control.Failed(action.OwnerID, fmt.Errorf("%s failed: %s", context, outcome.Err)))
			continue
		}
		if outcome.Applied {
			completed := "upshifted"
			if stopEffect {
				completed = "idle-unloaded"
			}
			item := control.ResultItem{Name: action.OwnerID, Message: fmt.Sprintf("%s claim %s to %q", completed, action.ClaimID, action.ToStep)}
			if stopEffect {
				report.Stopped = append(report.Stopped, item)
			} else {
				report.Observed = append(report.Observed, item)
			}
			continue
		}
		report.Observed = append(report.Observed, control.ResultItem{Name: action.OwnerID, Message: "skipped " + context + ": " + outcome.Reason})
	}
	if runErr != nil && len(plan.Actions) == 0 {
		report.Failed = append(report.Failed, control.Failed("capacity-actions", runErr))
	}
}

// LedgerSizeWarnThreshold is the claim count above which a post-prune ledger is
// reported as churn rather than accepted as history. It is a round number
// chosen to sit far above a healthy fleet's steady state (a few dozen claims)
// and far below the six thousand rows one resource's release-and-reclaim loop
// produced in a day.
const LedgerSizeWarnThreshold = 500

// ledgerSizeWarning reports a ledger that is still large after a prune.
func ledgerSizeWarning(ctx context.Context, store capacitySweepStore) string {
	claims, err := store.ListClaims(ctx, capacity.ClaimFilter{})
	if err != nil || len(claims) <= LedgerSizeWarnThreshold {
		return ""
	}
	terminal := 0
	byOwner := map[string]int{}
	for _, claim := range claims {
		if !capacity.IsActiveClaimStatus(claim.Status) {
			terminal++
		}
		byOwner[claim.OwnerID]++
	}
	worstOwner, worstCount := "", 0
	for owner, count := range byOwner {
		if count > worstCount || (count == worstCount && owner < worstOwner) {
			worstOwner, worstCount = owner, count
		}
	}
	return fmt.Sprintf(
		"Capacity ledger holds %d claim(s) after pruning, %d of them terminal and %d owned by %q; a ledger this size after a prune means an owner is releasing and re-claiming instead of resizing",
		len(claims), terminal, worstCount, worstOwner)
}
