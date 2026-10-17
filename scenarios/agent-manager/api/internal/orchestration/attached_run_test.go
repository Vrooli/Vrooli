package orchestration

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration/testutil"

	"github.com/google/uuid"
)

func TestAttachRunMintsScopedIdentityAndDetachRevokesIt(t *testing.T) {
	ctx := context.Background()
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	secret := []byte("attached-run-test-secret")
	orchestrator := New(nil, nil, repos.Runs, WithEvents(events), WithIdentitySecret(secret))

	result, err := orchestrator.AttachRun(ctx, AttachRunRequest{
		HarnessKind:    "claude-code",
		HarnessSession: "session-123",
		HarnessTitle:   "Safe attached fixture",
	})
	if err != nil {
		t.Fatalf("attach run: %v", err)
	}
	if result.Run.TaskID != uuid.Nil || result.Run.ExecutionMode != domain.ExecutionModeAttached {
		t.Fatalf("unexpected attached run identity: task=%s mode=%q", result.Run.TaskID, result.Run.ExecutionMode)
	}
	claims, err := identity.VerifyToken(result.Token, secret)
	if err != nil {
		t.Fatalf("verify attached token: %v", err)
	}
	if claims.RunID != result.Run.ID || claims.TaskID != uuid.Nil || claims.ProfileKey != "claude-code" {
		t.Fatalf("unexpected token claims: %#v", claims)
	}
	if claims.Meta["execution_mode"] != string(domain.ExecutionModeAttached) {
		t.Fatalf("token execution mode metadata = %q", claims.Meta["execution_mode"])
	}

	terminal, err := orchestrator.DetachRun(ctx, result.Run.ID, "fixture complete")
	if err != nil {
		t.Fatalf("detach run: %v", err)
	}
	if !terminal.Status.IsTerminal() || terminal.IdentityTokenRevokedAt == nil {
		t.Fatalf("detach did not terminalize and revoke token: status=%q revoked=%v", terminal.Status, terminal.IdentityTokenRevokedAt)
	}
	stored, err := repos.Runs.Get(ctx, result.Run.ID)
	if err != nil {
		t.Fatalf("reload detached run: %v", err)
	}
	if stored.Status != domain.RunStatusComplete || stored.IdentityTokenRevokedAt == nil {
		t.Fatalf("stored detached run: status=%q revoked=%v", stored.Status, stored.IdentityTokenRevokedAt)
	}
	count, err := events.Count(ctx, result.Run.ID)
	if err != nil || count != 2 {
		t.Fatalf("lifecycle event count = %d, err=%v; want attach and detach", count, err)
	}
}

func TestAttachedRunLivenessSweepOnlyObservesHarmlessProcess(t *testing.T) {
	ctx := context.Background()
	repos, events, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)

	// /bin/true is a harmless, self-terminating fixture. The test never starts
	// a shell, writes a command, or sends a signal to a user process.
	fixture := exec.Command("/bin/true")
	if err := fixture.Start(); err != nil {
		t.Fatalf("start harmless fixture: %v", err)
	}
	pid := fixture.Process.Pid
	if err := fixture.Wait(); err != nil {
		t.Fatalf("wait harmless fixture: %v", err)
	}

	now := time.Now().UTC()
	run := &domain.Run{
		ID:               uuid.New(),
		Tag:              "attached-liveness",
		RunMode:          domain.RunModeInPlace,
		ExecutionMode:    domain.ExecutionModeAttached,
		HarnessKind:      "test-harness",
		HarnessSessionID: "session-liveness",
		Status:           domain.RunStatusRunning,
		Phase:            domain.RunPhaseExecuting,
		RunnerPID:        pid,
		CreatedAt:        now.Add(-2 * time.Minute),
		UpdatedAt:        now.Add(-2 * time.Minute),
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("create liveness fixture: %v", err)
	}
	reconciler := NewReconciler(repos.Runs, nil,
		WithReconcilerEvents(events),
		// Advance the injected clock past the one-minute attach handoff grace
		// period without changing the database or any host process.
		WithReconcilerClock(func() time.Time { return now.Add(2 * time.Minute) }),
		WithReconcilerConfig(ReconcilerConfig{PendingThreshold: time.Hour, MaxStaleRuns: 10}),
	)
	if reconciler.attachedRunProcessAlive(run) {
		t.Fatal("exited harmless fixture was reported alive")
	}
	stats := reconciler.reconcile(ctx)
	if stats.StaleRuns != 1 {
		t.Fatalf("stale attached runs = %d, want 1 (errors=%v)", stats.StaleRuns, stats.Errors)
	}
	stored, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("reload swept run: %v", err)
	}
	if stored.Status != domain.RunStatusFailed {
		t.Fatalf("swept run status = %q, want failed", stored.Status)
	}
}
