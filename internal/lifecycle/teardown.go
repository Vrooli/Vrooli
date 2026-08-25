package lifecycle

import (
	"context"
	"fmt"
	"time"

	"github.com/vrooli/vrooli/internal/process"
)

// terminateAndAwait signals every tracked process group, polls the tracked
// record PIDs, and escalates only when the terminate deadline expires. The
// process PID—not the group leader—is the exit condition so descendants that
// survive a leader signal cannot make teardown appear complete.
func (r *Runner) terminateAndAwait(ctx context.Context, records []process.Record) error {
	if ctx == nil {
		ctx = context.Background()
	}
	groups := make(map[int]struct{}, len(records))
	pids := make([]int, 0, len(records))
	seenPIDs := make(map[int]struct{}, len(records))
	for _, record := range records {
		if record.PID > 0 {
			if _, ok := seenPIDs[record.PID]; !ok {
				seenPIDs[record.PID] = struct{}{}
				pids = append(pids, record.PID)
			}
		}
		group := record.PGID
		if group <= 0 {
			group = record.PID
		}
		if group > 0 {
			groups[group] = struct{}{}
		}
	}
	for group := range groups {
		_ = r.runtimeDeps().signalProcessGroup(group, false)
	}
	if awaitErr := r.awaitPIDs(ctx, pids, teardownTerminatePolicy); awaitErr == nil {
		for _, record := range records {
			r.releaseContainment(record.PID)
		}
		return nil
	}
	for group := range groups {
		_ = r.runtimeDeps().signalProcessGroup(group, true)
	}
	if err := r.awaitPIDs(ctx, pids, teardownForcePolicy); err != nil {
		return fmt.Errorf("tracked processes did not exit after termination and force-kill: %w", err)
	}
	for _, record := range records {
		r.releaseContainment(record.PID)
	}
	return nil
}

func (r *Runner) registerContainment(pid int, release func()) {
	if r == nil || pid <= 0 || release == nil {
		return
	}
	r.containmentMu.Lock()
	if r.containment == nil {
		r.containment = make(map[int]func())
	}
	r.containment[pid] = release
	r.containmentMu.Unlock()
}

func (r *Runner) releaseContainment(pid int) {
	if r == nil || pid <= 0 {
		return
	}
	r.containmentMu.Lock()
	release := r.containment[pid]
	delete(r.containment, pid)
	r.containmentMu.Unlock()
	if release != nil {
		release()
	}
}

func (r *Runner) awaitPIDs(ctx context.Context, pids []int, policy AwaitPolicy) error {
	return AwaitContext(ctx, r.awaitClock(), policy, func() (bool, error) {
		for _, pid := range pids {
			if r.runtimeDeps().isPIDRunning(pid) {
				return false, nil
			}
		}
		return true, nil
	})
}

func (r *Runner) terminatePIDs(ctx context.Context, pids []int) error {
	if ctx == nil {
		ctx = context.Background()
	}
	seen := make(map[int]struct{}, len(pids))
	unique := make([]int, 0, len(pids))
	for _, pid := range pids {
		if pid <= 0 {
			continue
		}
		if _, ok := seen[pid]; ok {
			continue
		}
		seen[pid] = struct{}{}
		unique = append(unique, pid)
		_ = r.runtimeDeps().signalPID(pid, false)
	}
	if len(unique) == 0 {
		return nil
	}
	if err := r.awaitPIDs(ctx, unique, teardownTerminatePolicy); err == nil {
		return nil
	}
	for _, pid := range unique {
		if r.runtimeDeps().isPIDRunning(pid) {
			_ = r.runtimeDeps().signalPID(pid, true)
		}
	}
	if err := r.awaitPIDs(ctx, unique, teardownForcePolicy); err != nil {
		// Orphan cleanup is defensive. The caller has already removed its own
		// registry state; retaining the historical best-effort contract is safer
		// than turning an unrelated listener into a failed scenario start.
		r.logWarn("Orphan listener did not exit before teardown policy", "error", err.Error(), "pids", unique)
	}
	return nil
}

func (r *Runner) waitForInstanceReleased(ctx context.Context, name, variant string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	item, err := r.loadScenario(name, "")
	if err != nil {
		return err
	}
	item.Variant = variant
	return AwaitContext(ctx, r.awaitClock(), restartReleasePolicy, func() (bool, error) {
		view, lookupErr := r.lookupRegistryRuntime(ctx, item)
		if lookupErr != nil {
			// A missing/unavailable row cannot hold a claim visible to the next
			// start, so proceed. The stop path has already verified fixed ports.
			return true, nil
		}
		return !view.Authoritative, nil
	})
}

var (
	// Termination grace is a deadline, not a mandatory delay. A process that
	// exits on the first poll makes teardown return on that first observation.
	teardownTerminatePolicy = AwaitPolicy{Timeout: 2 * time.Second, Interval: 100 * time.Millisecond}
	teardownForcePolicy     = AwaitPolicy{Timeout: 2 * time.Second, Interval: 100 * time.Millisecond}
	restartReleasePolicy    = AwaitPolicy{Timeout: 2 * time.Second, Interval: 100 * time.Millisecond}
	backgroundLaunchPolicy  = AwaitPolicy{Timeout: 2 * time.Second, Interval: 50 * time.Millisecond}
)
