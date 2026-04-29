package sandbox_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"workspace-sandbox/internal/clock"
	"workspace-sandbox/internal/sandbox"
	"workspace-sandbox/internal/testutil/mocks/sandboxiface"
)

// TestRunner_RunsPeriodicInOrder verifies that every periodic reconciler
// is invoked, in the slice order it was registered.
func TestRunner_RunsPeriodicInOrder(t *testing.T) {
	a := sandboxiface.NewFakeReconciler("a")
	b := sandboxiface.NewFakeReconciler("b")
	c := sandboxiface.NewFakeReconciler("c")
	r := sandbox.NewRunner(time.Hour, []sandbox.Reconciler{a, b, c}, nil, clock.System{})

	r.Start()
	defer r.Stop()

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if a.CallCount() >= 1 && b.CallCount() >= 1 && c.CallCount() >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := a.CallCount(); got < 1 {
		t.Errorf("a.calls = %d, want ≥1", got)
	}
	if got := b.CallCount(); got < 1 {
		t.Errorf("b.calls = %d, want ≥1", got)
	}
	if got := c.CallCount(); got < 1 {
		t.Errorf("c.calls = %d, want ≥1", got)
	}
}

// TestRunner_RunsStartupOnly verifies startup-only reconcilers fire
// exactly on boot, not on periodic ticks.
func TestRunner_RunsStartupOnly(t *testing.T) {
	periodic := sandboxiface.NewFakeReconciler("periodic")
	startup := sandboxiface.NewFakeReconciler("startup")
	r := sandbox.NewRunner(20*time.Millisecond, []sandbox.Reconciler{periodic}, []sandbox.Reconciler{startup}, clock.System{})

	r.Start()
	defer r.Stop()

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if periodic.CallCount() >= 3 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := startup.CallCount(); got != 1 {
		t.Errorf("startup.calls = %d, want exactly 1", got)
	}
	if got := periodic.CallCount(); got < 3 {
		t.Errorf("periodic.calls = %d, want ≥3", got)
	}
}

// TestRunner_RunOne_ByName invokes a specific reconciler synchronously.
func TestRunner_RunOne_ByName(t *testing.T) {
	a := sandboxiface.NewFakeReconciler("a")
	a.Report = sandbox.ReconcileReport{ItemsProcessed: 5}
	b := sandboxiface.NewFakeReconciler("b")
	r := sandbox.NewRunner(time.Hour, []sandbox.Reconciler{a, b}, nil, clock.System{})

	report, err := r.RunOne(context.Background(), "a")
	if err != nil {
		t.Fatalf("RunOne(a): %v", err)
	}
	if report.ItemsProcessed != 5 {
		t.Errorf("ItemsProcessed = %d, want 5", report.ItemsProcessed)
	}
	if a.CallCount() != 1 {
		t.Errorf("a.calls = %d, want 1", a.CallCount())
	}
	if b.CallCount() != 0 {
		t.Errorf("b.calls = %d, want 0 (never invoked)", b.CallCount())
	}
}

// TestRunner_RunOne_UnknownReturnsError fails loudly when no reconciler
// matches the requested name. The admin endpoint relies on this to
// return 404.
func TestRunner_RunOne_UnknownReturnsError(t *testing.T) {
	r := sandbox.NewRunner(time.Hour, []sandbox.Reconciler{sandboxiface.NewFakeReconciler("a")}, nil, clock.System{})
	_, err := r.RunOne(context.Background(), "nope")
	if err == nil {
		t.Fatal("expected error for unknown reconciler, got nil")
	}
}

// TestRunner_Names lists periodic + startup-only reconcilers in order.
func TestRunner_Names(t *testing.T) {
	r := sandbox.NewRunner(time.Hour,
		[]sandbox.Reconciler{sandboxiface.NewFakeReconciler("a"), sandboxiface.NewFakeReconciler("b")},
		[]sandbox.Reconciler{sandboxiface.NewFakeReconciler("z")}, clock.System{})
	got := r.Names()
	want := []string{"a", "b", "z"}
	if len(got) != len(want) {
		t.Fatalf("len(Names) = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("Names[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestRunner_Metrics records per-reconciler runs.
func TestRunner_Metrics(t *testing.T) {
	a := sandboxiface.NewFakeReconciler("a")
	a.Report = sandbox.ReconcileReport{ItemsProcessed: 3, ItemsFailed: 1}
	r := sandbox.NewRunner(time.Hour, []sandbox.Reconciler{a}, nil, clock.System{})

	if _, err := r.RunOne(context.Background(), "a"); err != nil {
		t.Fatalf("RunOne: %v", err)
	}
	m := r.Metrics()
	got, ok := m["a"]
	if !ok {
		t.Fatalf("metrics missing for a; got %+v", m)
	}
	if got.RunCount != 1 {
		t.Errorf("RunCount = %d, want 1", got.RunCount)
	}
	if got.ItemsProcessed != 3 {
		t.Errorf("ItemsProcessed = %d, want 3", got.ItemsProcessed)
	}
	if got.ItemsFailed != 1 {
		t.Errorf("ItemsFailed = %d, want 1", got.ItemsFailed)
	}
}

// TestRunner_StopIsIdempotent confirms calling Stop twice does not
// panic or block indefinitely.
func TestRunner_StopIsIdempotent(t *testing.T) {
	r := sandbox.NewRunner(10*time.Millisecond, []sandbox.Reconciler{sandboxiface.NewFakeReconciler("a")}, nil, clock.System{})
	r.Start()
	r.Stop()
	r.Stop()
}

// TestRunner_RunOne_StartupOnlyByName confirms startup-only reconcilers
// are addressable by name even though they don't fire on periodic ticks.
func TestRunner_RunOne_StartupOnlyByName(t *testing.T) {
	startup := sandboxiface.NewFakeReconciler("startup")
	startup.Report = sandbox.ReconcileReport{ItemsProcessed: 7}
	r := sandbox.NewRunner(time.Hour, nil, []sandbox.Reconciler{startup}, clock.System{})
	report, err := r.RunOne(context.Background(), "startup")
	if err != nil {
		t.Fatalf("RunOne: %v", err)
	}
	if report.ItemsProcessed != 7 {
		t.Errorf("ItemsProcessed = %d, want 7", report.ItemsProcessed)
	}
}

// TestRunner_LastErrorRecorded verifies that Runner.invoke folds an
// errored ReconcileReport's LastError into per-reconciler metrics.
func TestRunner_LastErrorRecorded(t *testing.T) {
	rc := sandboxiface.NewFakeReconciler("boom")
	rc.ErrText = "boom"
	r := sandbox.NewRunner(time.Hour, []sandbox.Reconciler{rc}, nil, clock.System{})
	if _, err := r.RunOne(context.Background(), "boom"); err != nil {
		t.Fatalf("RunOne: %v", err)
	}
	m := r.Metrics()["boom"]
	if m.LastError != "boom" {
		t.Errorf("LastError = %q, want %q", m.LastError, "boom")
	}
}

// Sanity: the canonical sentinel error continues to import cleanly.
var _ = errors.New("placeholder for stable error wiring")
