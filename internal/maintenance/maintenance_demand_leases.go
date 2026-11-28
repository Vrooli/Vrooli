package maintenance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/templatevalidation"
)

func (c *Controller) reconcileDemandLeases(ctx context.Context, store runtimeMaintenanceStore, now time.Time) ([]control.ResultItem, []control.ResultItem, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	stopped := make([]control.ResultItem, 0)
	failed := make([]control.ResultItem, 0)
	activeInstances, err := store.ListInstances(ctx, scenarioruntime.InstanceFilter{Statuses: scenarioruntime.ActiveInstanceStatuses()})
	if err != nil {
		return nil, nil, err
	}
	activeConsumers := make(map[string]struct{}, len(activeInstances))
	for _, instance := range activeInstances {
		activeConsumers[scenarioruntime.DependencyConsumerID(instance.Scenario, instance.Variant)] = struct{}{}
	}
	dependencyLeases, err := store.ListDemandLeases(ctx, scenarioruntime.DemandLeaseFilter{Statuses: []string{scenarioruntime.DemandLeaseActive}})
	if err != nil {
		return nil, nil, err
	}
	for _, lease := range dependencyLeases {
		if lease.Kind != scenarioruntime.DemandLeaseDependency {
			continue
		}
		// Only lifecycle-owned dependency identities can be reconciled from
		// runtime rows. Transient CLI calls and watchers own their own lifetime;
		// absence from the instance registry does not mean they have stopped.
		key, lifecycleOwned := scenarioruntime.DependencyConsumerInstance(lease.ConsumerID)
		if !lifecycleOwned {
			continue
		}
		if _, ok := activeConsumers[lease.ConsumerID]; !ok {
			// Dependency bootstrap acquires its hold before the parent runtime
			// row exists. Keep that hold while the parent's durable start
			// operation is still running; otherwise a slow dependency start can
			// be reaped by the next supervisor tick.
			if op, opErr := store.GetLatestStartOperation(ctx, key.Scenario, key.Variant); opErr == nil && op.Status == scenarioruntime.StartOperationStatusRunning {
				activeConsumers[lease.ConsumerID] = struct{}{}
			}
		}
		if _, ok := activeConsumers[lease.ConsumerID]; !ok {
			if _, releaseErr := store.ReleaseDemandLease(ctx, lease.LeaseID, "dependency consumer stopped"); releaseErr != nil && !errors.Is(releaseErr, scenarioruntime.ErrDemandLeaseExpired) && !errors.Is(releaseErr, scenarioruntime.ErrNotFound) {
				failed = append(failed, control.Failed(lease.LeaseID, releaseErr))
			}
			continue
		}
		var renewErr error
		if lease.ExpiresAt.After(now) {
			_, renewErr = store.RenewDemandLease(ctx, lease.LeaseID, scenarioruntime.DefaultDemandLeaseTTL)
		} else {
			// A delayed supervisor tick may observe an active consumer after
			// the short crash-recovery deadline. Reacquire the same stable
			// identity before expiry archival instead of dropping a live
			// dependency hold.
			_, renewErr = store.AcquireDemandLease(ctx, lease, scenarioruntime.DefaultDemandLeaseTTL)
		}
		if renewErr != nil && !errors.Is(renewErr, scenarioruntime.ErrDemandLeaseExpired) {
			failed = append(failed, control.Failed(lease.LeaseID, renewErr))
		}
	}
	expired, err := store.ExpireDemandLeases(ctx, now)
	if err != nil {
		return nil, nil, err
	}
	if len(expired) > 0 {
		stopped = append(stopped, control.Stopped("demand-leases", fmt.Sprintf("Expired %d demand lease(s)", len(expired))))
	}
	if _, err := store.ClaimDemandStopCandidates(ctx, now, "demand lease expired"); err != nil {
		return nil, nil, err
	}
	candidates, err := store.PendingDemandStops(ctx)
	if err != nil {
		return nil, nil, err
	}
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return stopped, failed, err
		}
		if stopErr := c.stopDemandManagedInstance(ctx, store, candidate); stopErr != nil {
			if errors.Is(stopErr, errDemandStopBusy) {
				break // The registry-wide owner also excludes the remaining candidates.
			}
			if errors.Is(stopErr, scenarioruntime.ErrNotFound) {
				continue // Another owner already completed this intent.
			}
			failed = append(failed, control.Failed(candidate.Scenario+"/"+candidate.Variant, stopErr))
			continue
		}
		stopped = append(stopped, control.Stopped(candidate.Scenario+"/"+candidate.Variant, "Stopped demand-managed instance after demand expired"))
	}
	return stopped, failed, nil
}

// ReconcileDemandLeases runs only the demand-retention portion of maintenance.
// The runtime supervisor uses this bounded entry point on its normal tick so
// idle demand-managed scenarios do not depend on an unrelated operator start
// or explicit cleanup command to stop.
func (c *Controller) ReconcileDemandLeases(ctx context.Context) (control.StopReport, error) {
	return c.ReconcileDemandLeasesAt(ctx, time.Now().UTC())
}

// ReconcileDemandLeasesAt is the clock-injectable form used by the runtime
// supervisor and deterministic tests. Production callers should use
// ReconcileDemandLeases unless they already own a control-plane clock.
func (c *Controller) ReconcileDemandLeasesAt(ctx context.Context, at time.Time) (control.StopReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var store runtimeMaintenanceStore
	var closeStore func()
	var err error
	if strings.TrimSpace(c.DBPath) != "" {
		opened, openErr := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: c.Home, DBPath: c.DBPath})
		if openErr != nil {
			return control.StopReport{}, openErr
		}
		store, closeStore = opened, func() { _ = opened.Close() }
	} else {
		store, closeStore, err = openRuntimeRegistryIfPresent(c.Home)
		if err != nil || store == nil {
			return control.StopReport{}, err
		}
	}
	defer closeStore()
	stopped, failed, err := c.reconcileDemandLeases(ctx, store, at)
	if err != nil {
		return control.StopReport{}, err
	}
	return control.StopReport{Stopped: stopped, Failed: failed, Message: control.StopSummary(len(stopped), len(failed))}, nil
}

// ReconcileCapacity runs the recurring capacity maintenance owner: sample
// resident claims, refresh or expire only claims supported by fresh GPU
// evidence, and apply policy-authorized idle unloads through each resource's
// declared degrade callback. The runtime supervisor calls this on its normal
// tick so an idle TTL does not depend on an unrelated cleanup command.
func (c *Controller) ReconcileCapacity(ctx context.Context) (control.StopReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return c.sweepCapacityClaims(ctx)
}

func (c *Controller) CleanStaleLocks() (control.StopReport, error) {
	ctx := context.Background()
	stopped := make([]control.ResultItem, 0)
	failed := make([]control.ResultItem, 0)

	store, closeStore, err := openRuntimeRegistryFn(c.Home)
	if err != nil {
		return control.StopReport{}, err
	}
	if store != nil {
		defer closeStore()
		now := time.Now().UTC()
		// The guard is what keeps this sweep from reaping a live start. Setup
		// phases routinely outrun the heartbeat TTL, and every `--clean-stale`
		// start runs this sweep, so without owner-liveness the fleet reaps its
		// own in-flight starts whenever two of them overlap.
		guard := scenarioruntime.StartingLeaseGuard{PIDRunning: newPIDLivenessMemo(processIsRunning)}
		if host, hostErr := (hostsession.DefaultProvider{}).Current(ctx, ""); hostErr == nil {
			guard.CurrentBootID = host.BootID
		}
		demandStopped, demandFailed, err := c.reconcileDemandLeases(ctx, store, now)
		if err != nil {
			return control.StopReport{}, err
		}
		stopped = append(stopped, demandStopped...)
		failed = append(failed, demandFailed...)
		expiredLeases, err := store.ExpireStaleStartingLeases(ctx, now, guard)
		if err != nil {
			return control.StopReport{}, err
		}
		for _, instance := range expiredLeases {
			stopped = append(stopped, control.Stopped(instance.Scenario+"/"+instance.InstanceID, "Expired stale starting runtime lease"))
		}
		expiredRuntime, reclaimable, err := expireNonAuthoritativeRegistryState(ctx, store)
		if err != nil {
			return control.StopReport{}, err
		}
		stopped = append(stopped, expiredRuntime...)

		// Port reclamation: expiring the registry record frees the CLAIM, but a
		// leaked process from the dead instance may still hold the OS port, which
		// is exactly the orphan-squat that makes test-genie's stricter
		// resolveURLs time out. Evict each holder — but ONLY after re-validating
		// it is still a Vrooli process from this install (stillVrooliOrphan), so a
		// foreign service that legitimately reused the port is never killed.
		for _, candidate := range reclaimable {
			item, evicted := c.reclaimSquattedPort(candidate)
			switch {
			case evicted:
				stopped = append(stopped, item)
			case item.Error != "":
				failed = append(failed, item)
			}
		}

		finalized, err := finalizeStuckStoppingInstances(ctx, store)
		if err != nil {
			return control.StopReport{}, err
		}
		stopped = append(stopped, finalized...)

		expiredClaims, err := store.ListExpiredActivePortClaims(ctx, now)
		if err != nil {
			return control.StopReport{}, err
		}
		for _, claim := range expiredClaims {
			if claim.Status != scenarioruntime.ClaimStatusReserved {
				continue
			}
			expired, err := store.ExpirePortClaim(ctx, claim.ClaimID)
			if err != nil {
				failed = append(failed, control.Failed(claim.ClaimID, err))
				continue
			}
			stopped = append(stopped, control.Stopped(fmt.Sprintf("%d", expired.Port), "Expired abandoned registry port reservation"))
		}

		// Supervisor sessions only reached a terminal status through graceful
		// shutdown, so every SIGKILLed supervisor left a row claiming to be
		// running. Retire the ones this host can prove are dead — a live peer
		// is never touched, because the same guard demands positive evidence.
		expiredSessions, err := store.ExpireStaleSupervisorSessions(ctx, now, guard)
		if err != nil {
			failed = append(failed, control.Failed("supervisor-sessions", err))
		} else if len(expiredSessions) > 0 {
			stopped = append(stopped, control.Stopped("supervisor-sessions",
				fmt.Sprintf("Retired %d supervisor session(s) whose process is gone", len(expiredSessions))))
		}

		// Editor leases are agent sessions; the same proof-of-death guard
		// applies. A session that is slow to heartbeat is building, not gone.
		expiredEditors, err := store.ExpireStaleEditorLeases(ctx, now, guard)
		if err != nil {
			failed = append(failed, control.Failed("editor-leases", err))
		}
		for _, lease := range expiredEditors {
			stopped = append(stopped, control.Stopped("agent-session/"+lease.SessionID, "Expired editor lease: "+lease.StopReason))
		}
		prunedClaims, err := store.PruneTerminalPortClaims(ctx, now.Add(-terminalPortClaimRetention))
		if err != nil {
			failed = append(failed, control.Failed("claim-retention", err))
		} else if prunedClaims > 0 {
			stopped = append(stopped, control.Stopped("registry-claims", fmt.Sprintf("Pruned %d expired/released registry claim rows older than %s", prunedClaims, terminalPortClaimRetention)))
		}
	}

	// Always-on resident-claim sweep (capacity §8.6): refresh GPU claims still
	// observed on the host and expire dead ones, so resident model-server claims
	// no longer depend on a 6h ttl_seconds stopgap. Best-effort: a ledger that
	// does not exist yet, or transient sensing/DB trouble, never fails the lock
	// sweep this rides on.
	var observed []control.ResultItem
	if capacityExpired, capErr := c.sweepCapacityClaims(ctx); capErr == nil {
		stopped = append(stopped, capacityExpired.Stopped...)
		failed = append(failed, capacityExpired.Failed...)
		observed = append(observed, capacityExpired.Observed...)
	} else {
		failed = append(failed, control.Failed("capacity-sweep", capErr))
	}

	return control.StopReport{
		Stopped:  stopped,
		Observed: observed,
		Failed:   failed,
		Message:  control.StopSummary(len(stopped), len(failed)),
	}, nil
}

func (c *Controller) ListOrphans() ([]SystemProcess, error) {
	snapshot, err := c.Snapshot()
	if err != nil {
		return nil, err
	}
	return append([]SystemProcess(nil), snapshot.Orphans...), nil
}

func (c *Controller) KillOrphans() (control.StopReport, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	orphans, err := c.ListOrphans()
	if err != nil {
		return control.StopReport{}, err
	}

	stopped := make([]control.ResultItem, 0, len(orphans))
	failed := make([]control.ResultItem, 0)
	observed := make([]control.ResultItem, 0)
	for _, item := range orphans {
		if err := c.stopOrphan(ctx, item.PID); err != nil {
			if errors.Is(err, errOrphanSkipped) {
				observed = append(observed, control.ResultItem{Name: strconv.Itoa(item.PID), Message: err.Error()})
				continue
			}
			failed = append(failed, control.Failed(strconv.Itoa(item.PID), err))
			continue
		}
		stopped = append(stopped, control.Stopped(strconv.Itoa(item.PID), item.Command))
	}

	// Opportunistically prune scenario process records pointing at dead PIDs.
	// These accumulate whenever a scenario process exits outside the normal
	// stop path (crash, external kill, host reboot). Failures here are
	// swallowed so record hygiene never blocks the orphan sweep.
	if staleStopped, _ := c.cleanStaleScenarioRecords(); len(staleStopped) > 0 {
		stopped = append(stopped, staleStopped...)
	}

	return control.StopReport{
		Observed: observed,
		Stopped:  stopped,
		Failed:   failed,
		Message:  control.StopSummary(len(stopped), len(failed)),
	}, nil
}

// reclaimSquattedPort evicts a process still holding a port whose owning
// instance was just expired as non-authoritative. It returns the result item
// and whether an eviction actually happened. The guard is the same one
// KillOrphans uses: a held incarnation must still be a current, unowned
// candidate. Native lifecycle trees remain with their owner. An expired claim
// does not authorize stopping a replacement service that acquired the port.
func (c *Controller) reclaimSquattedPort(candidate PortReclaimCandidate) (control.ResultItem, bool) {
	name := fmt.Sprintf("%s:%d", candidate.Scenario, candidate.Port)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := c.stopOrphan(ctx, candidate.PID); err != nil {
		if errors.Is(err, errOrphanSkipped) {
			return control.ResultItem{}, false
		}
		return control.Failed(name, err), false
	}
	return control.Stopped(name, fmt.Sprintf("Reclaimed port %d from orphaned pid %d", candidate.Port, candidate.PID)), true
}

// CleanStaleRecords removes scenario process records whose PID no longer
// resolves to a live process. Returns a StopReport describing each record
// that was pruned; intended for use by maintenance commands that want to
// surface hygiene cleanup alongside their primary action.
func (c *Controller) CleanStaleRecords() (control.StopReport, error) {
	stopped, err := c.cleanStaleScenarioRecords()
	if err != nil {
		return control.StopReport{}, err
	}
	return control.StopReport{
		Stopped: stopped,
		Message: control.StopSummary(len(stopped), 0),
	}, nil
}

func (c *Controller) ListTemplateValidationRuns(opts templatevalidation.CleanupOptions) (templatevalidation.CleanupResult, error) {
	opts.RepoRoot = c.Root
	opts.DryRun = true
	result := templatevalidation.ExecuteCleanup(templatevalidation.PlanCleanup(opts))
	return result, templatevalidation.ResultError(result)
}

func (c *Controller) CleanTemplateValidationRuns(opts templatevalidation.CleanupOptions) (templatevalidation.CleanupResult, error) {
	opts.RepoRoot = c.Root
	result := templatevalidation.ExecuteCleanup(templatevalidation.PlanCleanup(opts))
	if result.NeedsProtoGenerate && !result.DryRun && len(result.Failures) == 0 {
		if err := runProtoGenerateFn(c.Root); err != nil {
			return result, err
		}
		result.ProtoGenerateRan = true
	}
	return result, templatevalidation.ResultError(result)
}

func runProtoGenerate(repoRoot string) error {
	protoDir := filepath.Join(repoRoot, "packages", "proto")
	if _, err := os.Stat(filepath.Join(protoDir, "Makefile")); err != nil {
		return nil
	}
	cmd := shell.NewCommand("make", "generate")
	cmd.Dir = protoDir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// cleanStaleScenarioRecords walks $HOME/.vrooli/processes/scenarios/<name>/
// and removes every record whose PID is not running. It returns the set of
// pruned records as StopReport result items. The sweep is best-effort: an
// unreadable scenario directory is skipped, not reported as an error.
func (c *Controller) cleanStaleScenarioRecords() ([]control.ResultItem, error) {
	processesDir, err := repocontract.RuntimeHomeEntryPath(c.Home, repocontract.HomeKeyProcesses)
	if err != nil {
		return nil, err
	}
	root := filepath.Join(processesDir, repocontractmeta.ScenarioDir)
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	pruned := make([]control.ResultItem, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		scenario := entry.Name()
		records, err := process.ReadScenarioRecords(c.Home, scenario)
		if err != nil {
			continue
		}
		for _, record := range records {
			if record.PID > 0 && process.IsPIDRunning(record.PID) {
				// A reused numeric PID does not keep an old launch record alive.
				// Historical records without an OS identity remain conservative.
				if record.Incarnation == "" {
					continue
				}
				actual, err := process.Incarnation(record.PID)
				if err != nil || actual == record.Incarnation {
					continue
				}
			}
			step := record.Step
			if step == "" {
				continue
			}
			if removed, err := process.RemoveScenarioRecordIfMatches(c.Home, scenario, step, record); err != nil || !removed {
				continue
			}
			pruned = append(pruned, control.Stopped(scenario+"/"+step, "Removed stale process record (pid "+strconv.Itoa(record.PID)+")"))
		}
	}
	return pruned, nil
}
