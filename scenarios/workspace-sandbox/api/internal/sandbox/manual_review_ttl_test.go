// Tests for the manualReview TTL GC path added by Phase 4 of
// agent-sandbox-audit-foundation. The reconciler auto-denies abandoned
// manualReview=true sandboxes whose idle window exceeds the configured
// ManualReviewTTL, recording Source=SourceWorkspaceSandboxGC on the audit
// event so reviewers can tell GC-driven denials apart from operator denials.
//
// Time is sourced from the service's FakeClock (Round 4 Phase 2) so expiry
// is deterministic without bleeding the wall clock into assertions.

package sandbox

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/testutil/mocks"
	"workspace-sandbox/internal/types"
)

// newManualReviewTestService wires a Service with a FakeClock pinned to
// `now` so the manual-review TTL reconciler reads a deterministic time.
func newManualReviewTestService(t *testing.T, repo *mocks.FakeRepository, drv *mocks.FakeDriver, now time.Time) (*Service, *mocks.FakeClock) {
	t.Helper()
	clk := mocks.NewFakeClock(now)
	svc := NewService(repo, drv, ServiceConfig{
		DefaultProjectRoot: "/tmp/project",
		MaxSandboxes:       100,
		DefaultTTL:         24 * time.Hour,
	}, clk, WithGitOps(mocks.NewFakeGitOps()))
	return svc, clk
}

// TestReconcileManualReviewExpiry_AutoDeniesExpired verifies the happy path:
// a manualReview=true sandbox idle past the TTL window is rejected and torn
// down, with Source=SourceWorkspaceSandboxGC on the audit event.
func TestReconcileManualReviewExpiry_AutoDeniesExpired(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	// FakeClock sits well past LastUsedAt + ttl.
	svc, _ := newManualReviewTestService(t, repo, drv, time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC))

	id := uuid.New()
	repo.Sandboxes[id] = &types.Sandbox{
		ID:          id,
		ScopePath:   "/tmp/project/scope",
		ProjectRoot: "/tmp/project",
		Status:      types.StatusActive,
		// LastUsedAt simulates "run-end timestamp" — no further activity.
		LastUsedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		Behavior: types.SandboxBehavior{
			ManualReview: true,
		},
	}

	ttl := 7 * 24 * time.Hour
	svc.ReconcileManualReviewExpiry(context.Background(), ttl)

	// Mock Delete removes the sandbox from the map; that path is the
	// best-effort tear-down. The audit event is the load-bearing artifact:
	// it carries Source=SourceWorkspaceSandboxGC.
	var rejected *types.AuditEvent
	for _, evt := range repo.AuditEvents {
		if evt.EventType == "rejected" {
			rejected = evt
			break
		}
	}
	if rejected == nil {
		t.Fatal("expected a 'rejected' audit event for expired manualReview sandbox")
	}
	if rejected.Source != types.SourceWorkspaceSandboxGC {
		t.Errorf("expected Source=SourceWorkspaceSandboxGC, got %q", rejected.Source)
	}
	if rejected.Actor != "system" {
		t.Errorf("expected Actor=system, got %q", rejected.Actor)
	}
	if reason, _ := rejected.Details["reason"].(string); reason != "manualReview-ttl-expired" {
		t.Errorf("expected reason=manualReview-ttl-expired, got %v", rejected.Details["reason"])
	}
}

// TestReconcileManualReviewExpiry_PreservesIdleSandboxes verifies the negative
// case: a manualReview=true sandbox still inside the TTL window is left
// alone. Without this, the reconciler would race against operators who are
// genuinely mid-review.
func TestReconcileManualReviewExpiry_PreservesIdleSandboxes(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	// FakeClock at LastUsedAt + 1 day (< ttl).
	svc, _ := newManualReviewTestService(t, repo, drv, time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC))

	id := uuid.New()
	repo.Sandboxes[id] = &types.Sandbox{
		ID:         id,
		Status:     types.StatusActive,
		LastUsedAt: time.Date(2026, 4, 14, 0, 0, 0, 0, time.UTC),
		Behavior:   types.SandboxBehavior{ManualReview: true},
	}

	ttl := 7 * 24 * time.Hour
	svc.ReconcileManualReviewExpiry(context.Background(), ttl)

	for _, evt := range repo.AuditEvents {
		if evt.EventType == "rejected" {
			t.Fatalf("did not expect a 'rejected' audit event before TTL window: %+v", evt)
		}
	}
	if got := repo.Sandboxes[id]; got == nil || got.Status != types.StatusActive {
		t.Errorf("expected sandbox to remain Active before TTL, got status=%v", got)
	}
}

// TestReconcileManualReviewExpiry_IgnoresAutoApplySandboxes pins the contract
// that ManualReview=false (the contract default) sandboxes are NOT subject
// to the manualReview TTL — they have their own apply-on-terminal cleanup
// path via the existing applyLifecycleTerminal logic.
func TestReconcileManualReviewExpiry_IgnoresAutoApplySandboxes(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc, _ := newManualReviewTestService(t, repo, drv, time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC))

	id := uuid.New()
	repo.Sandboxes[id] = &types.Sandbox{
		ID:         id,
		Status:     types.StatusActive,
		LastUsedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), // ancient
		Behavior:   types.SandboxBehavior{ManualReview: false},
	}

	svc.ReconcileManualReviewExpiry(context.Background(), 7*24*time.Hour)

	for _, evt := range repo.AuditEvents {
		if evt.EventType == "rejected" {
			t.Fatalf("manualReview=false sandbox must not be GC-denied: %+v", evt)
		}
	}
}

// TestReconcileManualReviewExpiry_ZeroTTLIsDisabled pins the
// "ManualReviewTTL=0 disables expiry" semantics from the config doc string.
func TestReconcileManualReviewExpiry_ZeroTTLIsDisabled(t *testing.T) {
	repo := mocks.NewFakeRepository()
	drv := mocks.NewFakeDriver()
	svc, _ := newManualReviewTestService(t, repo, drv, time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC))

	id := uuid.New()
	repo.Sandboxes[id] = &types.Sandbox{
		ID:         id,
		Status:     types.StatusActive,
		LastUsedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Behavior:   types.SandboxBehavior{ManualReview: true},
	}

	// ttl=0 disables.
	svc.ReconcileManualReviewExpiry(context.Background(), 0)

	for _, evt := range repo.AuditEvents {
		if evt.EventType == "rejected" {
			t.Fatalf("ttl=0 must disable GC; got rejected event %+v", evt)
		}
	}
}
