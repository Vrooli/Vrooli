// Tests that the Runner actually invokes the filesystem orphan
// reconciler on both startup and every periodic tick. These are the
// "verify the schedule fires" gates the user asked for after the
// 2026-04-28 mount-leak incident — without them, a future refactor
// could silently drop the orphan pass and we'd be back to the same
// failure mode.

package sandbox

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/types"
)

// countingDriver records every ListSandboxDirs call so tests can
// assert the reconciler ran the expected number of times.
type countingDriver struct {
	*fakeOrphanDriver
	listCalls atomic.Int32
}

func newCountingDriver() *countingDriver {
	return &countingDriver{fakeOrphanDriver: &fakeOrphanDriver{}}
}

func (d *countingDriver) ListSandboxDirs(ctx context.Context) ([]uuid.UUID, error) {
	d.listCalls.Add(1)
	return d.fakeOrphanDriver.ListSandboxDirs(ctx)
}

// TestRunner_Startup_InvokesOrphanReconciler — when Start()
// is called, the very first synchronous pass MUST run the orphan
// reconciler. Without this, a fresh process boot would delay orphan
// cleanup until the first tick (potentially 15 minutes later).
func TestRunner_Startup_InvokesOrphanReconciler(t *testing.T) {
	repo := newFakeOrphanRepo()
	drv := newCountingDriver()
	svc := newReconcilerService(repo, drv.fakeOrphanDriver)
	// Replace driver with the counting wrapper.
	svc.driver = drv

	r := DefaultRunner(svc, time.Hour, 0, HealConfig{})
	r.Start()
	defer r.Stop()

	// The startup pass is synchronous before the goroutine enters its
	// select loop. Give the goroutine a tiny window to record the
	// initial pass — under 50ms is comfortable since the pass is in-mem.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if drv.listCalls.Load() >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if got := drv.listCalls.Load(); got < 1 {
		t.Fatalf("expected orphan reconciler to fire at least once on startup, got %d list calls", got)
	}
}

// TestRunner_PeriodicTick_InvokesOrphanReconciler — after
// the startup pass, each ticker fire must also run the orphan
// reconciler. We use a 30ms interval so the test is fast.
func TestRunner_PeriodicTick_InvokesOrphanReconciler(t *testing.T) {
	repo := newFakeOrphanRepo()
	drv := newCountingDriver()
	svc := newReconcilerService(repo, drv.fakeOrphanDriver)
	svc.driver = drv

	r := DefaultRunner(svc, 30*time.Millisecond, 0, HealConfig{})
	r.Start()
	defer r.Stop()

	// Wait for at least 2 list calls (1 startup + ≥1 tick). Bound the
	// wait at 2s — generous for a 30ms ticker but tight enough to fail
	// fast if the wiring is wrong.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if drv.listCalls.Load() >= 2 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if got := drv.listCalls.Load(); got < 2 {
		t.Fatalf("expected orphan reconciler to fire on at least one periodic tick (total ≥2), got %d", got)
	}
}

// TestRunner_Stop_ReleasesGoroutine — verifies Stop()
// terminates the goroutine. If it didn't, every test in this file
// would leak goroutines and `go test -race` would eventually flag
// them. Pins the contract for future readers.
func TestRunner_Stop_ReleasesGoroutine(t *testing.T) {
	repo := newFakeOrphanRepo()
	drv := newCountingDriver()
	svc := newReconcilerService(repo, drv.fakeOrphanDriver)
	svc.driver = drv

	r := DefaultRunner(svc, 10*time.Millisecond, 0, HealConfig{})
	r.Start()
	r.Stop()

	// Stop() blocks on doneCh; if it returned, the goroutine exited.
	// A second Stop() must not panic or block.
	r.Stop()
}

// TestRunner_NilSafety — Start() and Stop() on a nil Runner must not
// panic. Defensive coding so an initialization failure can't take the
// whole API down.
func TestRunner_NilSafety(t *testing.T) {
	var nilRunner *Runner
	nilRunner.Start() // no-op, no panic
	nilRunner.Stop()  // no-op, no panic
}

// Compile-time guard: ensure the Driver interface still includes the
// orphan-reconciliation methods. If they're removed in a future refactor,
// this declaration fails to compile and the regression is loud.
var _ interface {
	ListSandboxDirs(ctx context.Context) ([]uuid.UUID, error)
	CleanupOrphan(ctx context.Context, id uuid.UUID) error
} = (driver.Driver)(nil)

// Compile-time guard: ensure types.Sandbox still has the Status field
// at the type the reconciler reads. If the schema changes, this fails
// loudly rather than silently bypassing the orphan-vs-active check.
var _ = func() bool {
	var s types.Sandbox
	_ = s.Status
	return true
}()
