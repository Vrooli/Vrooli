package orchestration

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
)

// Await-handle registry (durable park/resume, Phase 3).
//
// The registry is agent-manager's robust waiter: for every parked run it owns a
// background goroutine that blocks on the run's await-handle via the matching
// Waiter and then wakes the run. It survives an agent-manager restart —
// RecoverParkedRuns re-spawns a watcher for every persisted parked run on boot
// — and wakes idempotently, so a double-resolve (waiter racing an external
// wake/stop) never double-wakes (WakeRun no-ops a non-parked run).
//
// Lifecycle of one watcher:
//   - resolve   → the Waiter returns       → WakeRun(result)
//   - deadline  → the handle's deadline hits → WakeRun(result, timedOut)
//   - cancel    → Cancel/Stop fires the ctx  → exit WITHOUT waking (the run was
//     stopped/woken elsewhere; that path owns the terminal/running transition)

// wakeTimeout bounds the WakeRun call a watcher makes once its wait resolves, so
// a stuck resume cannot leak the watcher goroutine forever.
const wakeTimeout = 30 * time.Second

// awaitWaker is the narrow orchestrator surface the registry drives. The
// concrete *Orchestrator satisfies it; the seam keeps registry tests free of a
// full orchestrator.
type awaitWaker interface {
	// WakeRun resumes a parked run with the awaited result. It is idempotent:
	// a non-parked run is a no-op.
	WakeRun(ctx context.Context, in WakeRunInput) (*domain.Run, error)
	// ListParkedRuns returns all parked runs with their await-handle populated
	// (used for restart recovery).
	ListParkedRuns(ctx context.Context) ([]*domain.Run, error)
}

// AwaitRegistry owns the per-parked-run waiter goroutines.
type AwaitRegistry struct {
	waker   awaitWaker
	waiters map[string]Waiter

	mu     sync.Mutex
	active map[uuid.UUID]context.CancelFunc
	wg     sync.WaitGroup
	closed bool
}

// NewAwaitRegistry builds a registry over the given waker, indexing each Waiter
// by its Producer key. A Waiter with a duplicate producer overrides the prior
// one (last wins); nil waiters are ignored.
func NewAwaitRegistry(waker awaitWaker, waiters ...Waiter) *AwaitRegistry {
	idx := make(map[string]Waiter, len(waiters))
	for _, w := range waiters {
		if w == nil {
			continue
		}
		idx[w.Producer()] = w
	}
	return &AwaitRegistry{
		waker:   waker,
		waiters: idx,
		active:  make(map[uuid.UUID]context.CancelFunc),
	}
}

func (r *AwaitRegistry) log() *slog.Logger { return obs.Component("await-registry") }

// Register spawns (if not already watching the run) a background watcher for a
// parked run's await-handle. It is safe to call repeatedly for the same run:
// only one watcher exists per run (mirroring the one-open-handle-per-run rule),
// so a re-register while parked is a no-op. No-op after Stop.
func (r *AwaitRegistry) Register(runID uuid.UUID, handle *domain.AwaitHandle) {
	if r == nil || handle == nil {
		return
	}

	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	if _, exists := r.active[runID]; exists {
		r.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.active[runID] = cancel
	r.wg.Add(1)
	r.mu.Unlock()

	go r.watch(ctx, runID, *handle)
}

// Cancel stops the watcher for a run without waking it (used when the run is
// stopped/aborted while parked, or when an external wake already resumed it).
// Idempotent and safe for an unknown run.
func (r *AwaitRegistry) Cancel(runID uuid.UUID) {
	if r == nil {
		return
	}
	r.mu.Lock()
	cancel := r.active[runID]
	delete(r.active, runID)
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// Stop cancels every watcher and blocks until they exit. Called on shutdown so
// no waiter goroutine outlives the process; parked runs are durable (handles
// persisted) and re-spawned on the next boot via RecoverParkedRuns.
func (r *AwaitRegistry) Stop() {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.closed = true
	cancels := make([]context.CancelFunc, 0, len(r.active))
	for id, cancel := range r.active {
		cancels = append(cancels, cancel)
		delete(r.active, id)
	}
	r.mu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
	r.wg.Wait()
}

// RecoverParkedRuns re-spawns a watcher for every persisted parked run. Called
// once on boot (alongside reconciler recovery) so an agent-manager restart does
// not strand parked runs without a waiter. Returns the count of watchers
// (re-)registered.
func (r *AwaitRegistry) RecoverParkedRuns(ctx context.Context) (int, error) {
	if r == nil {
		return 0, nil
	}
	runs, err := r.waker.ListParkedRuns(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, run := range runs {
		if run.AwaitHandle == nil {
			// A parked run without a handle cannot be waited on. This should not
			// happen (park always records one), but if it does the reconciler's
			// park-TTL path is the backstop; log so it is visible.
			r.log().Warn("parked run has no await-handle; not re-spawning waiter",
				obs.KeyRunID, run.ID.String())
			continue
		}
		r.Register(run.ID, run.AwaitHandle)
		n++
	}
	if n > 0 {
		r.log().Info("re-spawned waiters for parked runs", "count", n)
	}
	return n, nil
}

// watch is one parked run's waiter goroutine.
func (r *AwaitRegistry) watch(ctx context.Context, runID uuid.UUID, handle domain.AwaitHandle) {
	defer r.wg.Done()
	defer r.deregister(runID)

	waiter := r.waiters[handle.Producer]
	if waiter == nil {
		// No Waiter for this producer: we can never resolve the handle, so wake
		// immediately with a typed error rather than letting the run hang until
		// its deadline. The agent decides how to proceed.
		r.wake(runID, handle,
			fmt.Sprintf("[wait error] no waiter registered for producer %q (key %q); cannot await this work",
				handle.Producer, handle.Key),
			false)
		return
	}

	waitCtx := ctx
	if handle.Deadline != nil {
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithDeadline(ctx, *handle.Deadline)
		defer cancel()
	}

	result, waitErr := waiter.Wait(waitCtx, handle.Key)

	// Distinguish the three terminal conditions. Check the parent ctx first: if
	// it was cancelled (Cancel/Stop), the run was stopped or externally woken
	// elsewhere — do not wake.
	if ctx.Err() != nil {
		return
	}
	// Deadline elapsed → typed timeout-wake (result carries any partial output).
	if waitCtx.Err() != nil && errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
		r.wake(runID, handle, result, true)
		return
	}
	if waitErr != nil {
		r.wake(runID, handle, fmt.Sprintf("[wait error] %v", waitErr), false)
		return
	}
	r.wake(runID, handle, result, false)
}

// wake resumes the run with the resolved result. WakeRun is idempotent, so a
// race against an external wake/stop is safe.
func (r *AwaitRegistry) wake(runID uuid.UUID, handle domain.AwaitHandle, result string, timedOut bool) {
	ctx, cancel := context.WithTimeout(context.Background(), wakeTimeout)
	defer cancel()
	if _, err := r.waker.WakeRun(ctx, WakeRunInput{RunID: runID, Result: result, TimedOut: timedOut}); err != nil {
		r.log().Error("wake failed",
			obs.KeyRunID, runID.String(),
			"producer", handle.Producer,
			"timed_out", timedOut,
			obs.KeyError, err.Error())
	}
}

// deregister removes a watcher's entry on exit. The map may already lack the
// entry (Cancel/Stop removed it); that is fine.
func (r *AwaitRegistry) deregister(runID uuid.UUID) {
	r.mu.Lock()
	delete(r.active, runID)
	r.mu.Unlock()
}

// Watching reports whether a watcher is currently registered for runID.
// Exposed for observability (e.g. surfacing live park waits) and tests.
func (r *AwaitRegistry) Watching(runID uuid.UUID) bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.active[runID]
	return ok
}

// ActiveCount returns the number of live waiters. Exposed for observability and
// tests.
func (r *AwaitRegistry) ActiveCount() int {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.active)
}
