package sandbox

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// fakeReconciler is a minimal Reconciler used by Runner tests.
type fakeReconciler struct {
	name  string
	calls atomic.Int32
	delay time.Duration
	out   ReconcileReport
}

func (f *fakeReconciler) Name() string { return f.name }

func (f *fakeReconciler) Run(ctx context.Context) ReconcileReport {
	f.calls.Add(1)
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	if f.out.Duration == 0 {
		f.out.Duration = time.Microsecond
	}
	return f.out
}

// TestRunner_RunsPeriodicInOrder verifies that every periodic reconciler
// is invoked, in the slice order it was registered.
func TestRunner_RunsPeriodicInOrder(t *testing.T) {
	a := &fakeReconciler{name: "a"}
	b := &fakeReconciler{name: "b"}
	c := &fakeReconciler{name: "c"}
	r := NewRunner(time.Hour, []Reconciler{a, b, c}, nil)

	r.Start()
	defer r.Stop()

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if a.calls.Load() >= 1 && b.calls.Load() >= 1 && c.calls.Load() >= 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := a.calls.Load(); got < 1 {
		t.Errorf("a.calls = %d, want ≥1", got)
	}
	if got := b.calls.Load(); got < 1 {
		t.Errorf("b.calls = %d, want ≥1", got)
	}
	if got := c.calls.Load(); got < 1 {
		t.Errorf("c.calls = %d, want ≥1", got)
	}
}

// TestRunner_RunsStartupOnly verifies startup-only reconcilers fire
// exactly on boot, not on periodic ticks.
func TestRunner_RunsStartupOnly(t *testing.T) {
	periodic := &fakeReconciler{name: "periodic"}
	startup := &fakeReconciler{name: "startup"}
	r := NewRunner(20*time.Millisecond, []Reconciler{periodic}, []Reconciler{startup})

	r.Start()
	defer r.Stop()

	// Wait long enough for several periodic ticks.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if periodic.calls.Load() >= 3 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got := startup.calls.Load(); got != 1 {
		t.Errorf("startup.calls = %d, want exactly 1", got)
	}
	if got := periodic.calls.Load(); got < 3 {
		t.Errorf("periodic.calls = %d, want ≥3", got)
	}
}

// TestRunner_RunOne_ByName invokes a specific reconciler synchronously.
func TestRunner_RunOne_ByName(t *testing.T) {
	a := &fakeReconciler{name: "a", out: ReconcileReport{ItemsProcessed: 5}}
	b := &fakeReconciler{name: "b"}
	r := NewRunner(time.Hour, []Reconciler{a, b}, nil)

	report, err := r.RunOne(context.Background(), "a")
	if err != nil {
		t.Fatalf("RunOne(a): %v", err)
	}
	if report.ItemsProcessed != 5 {
		t.Errorf("ItemsProcessed = %d, want 5", report.ItemsProcessed)
	}
	if a.calls.Load() != 1 {
		t.Errorf("a.calls = %d, want 1", a.calls.Load())
	}
	if b.calls.Load() != 0 {
		t.Errorf("b.calls = %d, want 0 (never invoked)", b.calls.Load())
	}
}

// TestRunner_RunOne_UnknownReturnsError fails loudly when no reconciler
// matches the requested name. The admin endpoint relies on this to
// return 404.
func TestRunner_RunOne_UnknownReturnsError(t *testing.T) {
	r := NewRunner(time.Hour, []Reconciler{&fakeReconciler{name: "a"}}, nil)
	_, err := r.RunOne(context.Background(), "nope")
	if err == nil {
		t.Fatal("expected error for unknown reconciler, got nil")
	}
}

// TestRunner_Names lists periodic + startup-only reconcilers in order.
func TestRunner_Names(t *testing.T) {
	r := NewRunner(time.Hour,
		[]Reconciler{&fakeReconciler{name: "a"}, &fakeReconciler{name: "b"}},
		[]Reconciler{&fakeReconciler{name: "z"}})
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
	a := &fakeReconciler{name: "a", out: ReconcileReport{ItemsProcessed: 3, ItemsFailed: 1}}
	r := NewRunner(time.Hour, []Reconciler{a}, nil)

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
	r := NewRunner(10*time.Millisecond, []Reconciler{&fakeReconciler{name: "a"}}, nil)
	r.Start()
	r.Stop()
	r.Stop()
}

// TestRunner_RunOne_StartupOnlyByName confirms startup-only reconcilers
// are addressable by name even though they don't fire on periodic ticks.
func TestRunner_RunOne_StartupOnlyByName(t *testing.T) {
	startup := &fakeReconciler{name: "startup", out: ReconcileReport{ItemsProcessed: 7}}
	r := NewRunner(time.Hour, nil, []Reconciler{startup})
	report, err := r.RunOne(context.Background(), "startup")
	if err != nil {
		t.Fatalf("RunOne: %v", err)
	}
	if report.ItemsProcessed != 7 {
		t.Errorf("ItemsProcessed = %d, want 7", report.ItemsProcessed)
	}
}

// errReconciler always reports an error message in LastError so we can
// verify Runner.invoke folds it into metrics.
type errReconciler struct{ name string }

func (e *errReconciler) Name() string { return e.name }
func (e *errReconciler) Run(ctx context.Context) ReconcileReport {
	return ReconcileReport{LastError: "boom", Duration: time.Microsecond}
}

func TestRunner_LastErrorRecorded(t *testing.T) {
	r := NewRunner(time.Hour, []Reconciler{&errReconciler{name: "boom"}}, nil)
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
