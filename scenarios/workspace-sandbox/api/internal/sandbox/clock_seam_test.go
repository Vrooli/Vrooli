// Round 4 Phase 2 — deterministic-time tests for the per-consumer
// behaviors that flow through the injected Clock.
//
// These tests are not just smoke checks: each one drives a
// time-dependent contract that previously could only be verified by
// sleeping the wall clock or trusting external coordinates. Every
// branch is now reachable through FakeClock.Advance / SetNow, with no
// real-walltime sleeps in the test body.

package sandbox

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/audit"
	"workspace-sandbox/internal/testutil/mocks"
	"workspace-sandbox/internal/types"
)

// countingReconcilerInternal is an in-package Reconciler whose
// invocations the test can count via atomic loads. Using an in-package
// type avoids the cycle between sandbox and testutil/mocks/sandboxiface
// (which imports sandbox itself).
type countingReconcilerInternal struct {
	name  string
	calls atomic.Int32
}

func newCountingReconciler(name string) *countingReconcilerInternal {
	return &countingReconcilerInternal{name: name}
}

func (c *countingReconcilerInternal) Name() string { return c.name }
func (c *countingReconcilerInternal) Run(ctx context.Context) ReconcileReport {
	c.calls.Add(1)
	return ReconcileReport{}
}

// waitForCount polls c.calls every 5ms up to 500ms, then fails the
// test. The Runner kicks the reconciler from a goroutine so the test
// has to give the scheduler time to dispatch; waiting here keeps the
// determinism of FakeClock without bleeding wall-time assumptions
// into the assertion.
func waitForCount(t *testing.T, c *countingReconcilerInternal, want int32, msg string) {
	t.Helper()
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if c.calls.Load() >= want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("%s: got %d calls, want %d", msg, c.calls.Load(), want)
}

// newClockTestService wires a Service whose every time-dependent path
// reads from the supplied FakeClock — including auto-heal grace
// windows, lifecycle TTL evaluation, manual-review expiry, daemon
// reaper start-time math, and the orphan reconciler's duration field.
func newClockTestService(t *testing.T, repo *mocks.FakeRepository, drv *mocks.FakeDriver, clk *mocks.FakeClock) *Service {
	t.Helper()
	return NewService(repo, drv, ServiceConfig{
		DefaultProjectRoot: "/tmp/project",
		MaxSandboxes:       100,
		DefaultTTL:         24 * time.Hour,
	}, clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk), WithGitOps(mocks.NewFakeGitOps()))
}

// TestService_Stop_TimestampsViaClock pins that Service.Stop records
// the StoppedAt timestamp from the injected clock — not the wall.
// Without this guarantee, audit timelines would jitter against the OS
// clock and idle TTL evaluation against StoppedAt would be flaky.
func TestService_Stop_TimestampsViaClock(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	pinned := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	clk := mocks.NewFakeClock(pinned)
	svc := newClockTestService(t, repo, drv, clk)

	id := uuid.New()
	repo.Sandboxes[id] = &types.Sandbox{
		ID:        id,
		Status:    types.StatusActive,
		ScopePath: "/tmp/scope",
		MergedDir: "/tmp/merged",
		UpperDir:  "/tmp/upper",
		WorkDir:   "/tmp/work",
		LowerDir:  "/tmp/lower",
	}

	if _, err := svc.Stop(context.Background(), id); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	got := repo.Sandboxes[id]
	if got.Status != types.StatusStopped {
		t.Fatalf("status = %s, want stopped", got.Status)
	}
	if got.StoppedAt == nil {
		t.Fatal("expected StoppedAt to be set")
	}
	if !got.StoppedAt.Equal(pinned) {
		t.Errorf("StoppedAt = %s, want %s (the FakeClock pin)", got.StoppedAt, pinned)
	}
}

// TestReconcileLifecycle_IdleTimeoutFiresOnAdvance pins the idle-TTL
// branch: a sandbox whose LastUsedAt sits before now-IdleTimeout is
// stopped on this pass; advancing the clock past the threshold flips
// the decision deterministically.
func TestReconcileLifecycle_IdleTimeoutFiresOnAdvance(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	start := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	clk := mocks.NewFakeClock(start)
	svc := newClockTestService(t, repo, drv, clk)

	id := uuid.New()
	repo.Sandboxes[id] = &types.Sandbox{
		ID:         id,
		Status:     types.StatusActive,
		LastUsedAt: start, // matches "now"
		MergedDir:  "/tmp/merged",
		Behavior: types.SandboxBehavior{
			Lifecycle: types.LifecycleConfig{
				IdleTimeout: time.Hour,
			},
		},
	}

	// First pass: still within the idle window — sandbox stays active.
	svc.ReconcileLifecycle(context.Background())
	if got := repo.Sandboxes[id]; got.Status != types.StatusActive {
		t.Fatalf("first pass: status = %s, want active", got.Status)
	}

	// Advance past the idle window; the next pass must stop it.
	clk.Advance(2 * time.Hour)
	svc.ReconcileLifecycle(context.Background())
	if got := repo.Sandboxes[id]; got.Status != types.StatusStopped {
		t.Fatalf("after 2h advance: status = %s, want stopped", got.Status)
	}
}

// TestReconcileManualReviewExpiry_TTLBoundary pins the boundary
// semantics of the manual-review TTL evaluator: the predicate is
// `cutoff - LastUsedAt < ttl` (skip), so 1ns short of the TTL skips
// and exactly-at-or-past TTL rejects. Without this gate, off-by-one
// changes to the predicate would silently widen or narrow the window
// by a clock tick — undetectable on a wall clock.
func TestReconcileManualReviewExpiry_TTLBoundary(t *testing.T) {
	lastUsed := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	ttl := 7 * 24 * time.Hour

	cases := []struct {
		name           string
		now            time.Time
		wantDeny       bool
		boundaryReason string
	}{
		{"one_ns_short_of_ttl", lastUsed.Add(ttl).Add(-time.Nanosecond), false, "elapsed < ttl"},
		{"exactly_at_ttl", lastUsed.Add(ttl), true, "elapsed == ttl (predicate <)"},
		{"one_ns_past_ttl", lastUsed.Add(ttl).Add(time.Nanosecond), true, "elapsed > ttl"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewFakeRepository()
			drv := mocks.NewFakeDriver()
			clk := mocks.NewFakeClock(tc.now)
			svc := newClockTestService(t, repo, drv, clk)

			id := uuid.New()
			repo.Sandboxes[id] = &types.Sandbox{
				ID:         id,
				Status:     types.StatusActive,
				LastUsedAt: lastUsed,
				Behavior:   types.SandboxBehavior{ManualReview: true},
			}

			svc.ReconcileManualReviewExpiry(context.Background(), ttl)

			var rejected bool
			for _, evt := range repo.AuditEvents {
				if evt.EventType == "rejected" {
					rejected = true
					break
				}
			}
			if rejected != tc.wantDeny {
				t.Fatalf("rejected = %v, want %v (%s)", rejected, tc.wantDeny, tc.boundaryReason)
			}
		})
	}
}

// TestRunner_TickerFiresThroughFakeClock pins that the periodic
// reconciler ticker is driven by the injected clock — advancing the
// fake clock by one interval produces exactly one extra reconciler
// invocation. The reconciler runner previously used time.NewTicker
// directly, which forced wall-time sleeps in tests.
func TestRunner_TickerFiresThroughFakeClock(t *testing.T) {
	rc := newCountingReconciler("rc")
	clk := mocks.NewFakeClock(time.Time{})
	r := NewRunner(time.Hour, []Reconciler{rc}, nil, clk)
	r.Start()
	defer r.Stop()

	// The startup pass is synchronous before the goroutine enters the
	// select loop, but the goroutine still needs a moment to register
	// the ticker against FakeClock. Wait for the registration via a
	// brief poll on the count, capped tightly so failure is fast.
	waitForCount(t, rc, 1, "startup pass did not run")

	// One interval forward → one tick.
	clk.Advance(time.Hour)
	waitForCount(t, rc, 2, "first tick did not fire after Advance(1h)")

	// Second interval forward → second tick.
	clk.Advance(time.Hour)
	waitForCount(t, rc, 3, "second tick did not fire after second Advance(1h)")
}
