// Package sandboxiface holds fakes that implement interfaces declared
// in internal/sandbox. They live in a subpackage of testutil/mocks so
// internal-test files in package sandbox can still import the rest of
// mocks/ (FakeRepository, FakeDriver, FakeGitOps, …) without
// triggering an import cycle.
package sandboxiface

import (
	"context"
	"sync"
	"time"

	"workspace-sandbox/internal/sandbox"
)

// FakeReconciler is a minimal sandbox.Reconciler used by the Runner
// tests. Each Run call appends to Runs, makes the report's
// LastError reflect ErrText (when set), and returns the canned report.
type FakeReconciler struct {
	mu sync.Mutex

	NameValue string
	Report    sandbox.ReconcileReport
	ErrText   string

	// RunFn allows a test to override the entire Run body when it
	// needs custom behavior (e.g., panic, sleep, count). When nil
	// the canned Report path is taken.
	RunFn func(ctx context.Context) sandbox.ReconcileReport

	// Runs records every call. Tests assert on len(Runs) and order
	// to validate scheduling behavior.
	Runs []context.Context
}

// NewFakeReconciler returns a reconciler whose Run reports zero
// processed, zero failed, and no error. The Name defaults to "fake".
func NewFakeReconciler(name string) *FakeReconciler {
	return &FakeReconciler{NameValue: name}
}

func (f *FakeReconciler) Name() string { return f.NameValue }

func (f *FakeReconciler) Run(ctx context.Context) sandbox.ReconcileReport {
	f.mu.Lock()
	f.Runs = append(f.Runs, ctx)
	f.mu.Unlock()
	if f.RunFn != nil {
		return f.RunFn(ctx)
	}
	r := f.Report
	if f.ErrText != "" {
		r.LastError = f.ErrText
	}
	// Synthesize a non-zero Duration so the Runner's metrics record
	// something meaningful even when the test didn't bother setting one.
	if r.Duration == 0 {
		r.Duration = time.Microsecond
	}
	return r
}

// CallCount returns the number of recorded Run invocations.
func (f *FakeReconciler) CallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.Runs)
}

var _ sandbox.Reconciler = (*FakeReconciler)(nil)
