package sandbox_test

import (
	"context"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/audit"
	"workspace-sandbox/internal/clock"
	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/sandbox"
	"workspace-sandbox/internal/testutil/mocks"
	"workspace-sandbox/internal/testutil/mocks/sandboxiface"
	"workspace-sandbox/internal/types"
)

// TestInvariants is the canonical entry point for sandbox-package
// invariants from docs/internal/INVARIANTS.md. Each subtest name is a
// stable invariant ID — kept in lockstep with INVARIANTS.md by
// scripts/check-invariants.sh.
func TestInvariants(t *testing.T) {
	t.Run("I-MOUNT-1", invariantDeleteOwnsDaemonTeardown)
	t.Run("I-AUDIT-1", invariantStateTransitionEmitsOneAudit)
}

// I-MOUNT-1 — Service.Delete returns ⇒ no fuse-overlayfs daemon
// remains for that sandbox UUID. Pinned in fuller form by
// delete_daemon_lifecycle_test.go; this subtest gives the invariant
// ID a t.Run home that the CI scan picks up.
func invariantDeleteOwnsDaemonTeardown(t *testing.T) {
	t.Helper()
	id := uuid.New()
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("spawn helper: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	})

	procFS := sandboxiface.NewFakeProcFS(map[string]sandboxiface.FakeProcEntry{
		strconv.Itoa(cmd.Process.Pid): {
			Cmdline: []byte("fuse-overlayfs\x00-o\x00upperdir=/run/" + id.String() + "/home-upper\x00"),
		},
	})
	repo := mocks.NewFakeRepository()
	repo.Sandboxes[id] = &types.Sandbox{ID: id, Status: types.StatusActive}

	clk := clock.System{}
	svc := sandbox.NewService(
		repo, mocks.NewFakeDriver(), sandbox.ServiceConfig{},
		clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk), process.NewOSExecStarter(),
		sandbox.WithProcFS(procFS),
	)

	if err := svc.Delete(context.Background(), id); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
		// pass
	case <-time.After(7 * time.Second):
		t.Fatalf("daemon helper PID %d still alive after Delete", cmd.Process.Pid)
	}
}

// I-AUDIT-1 — every state transition emits exactly one audit-log
// entry. The Service's deleted/created/approved/rejected hooks all go
// through audit.Emitter; we pin the most-trafficked transition
// (Delete) here and rely on the per-transition unit tests for the
// rest.
func invariantStateTransitionEmitsOneAudit(t *testing.T) {
	t.Helper()
	id := uuid.New()
	repo := mocks.NewFakeRepository()
	repo.Sandboxes[id] = &types.Sandbox{ID: id, Status: types.StatusActive}

	clk := clock.System{}
	svc := sandbox.NewService(
		repo, mocks.NewFakeDriver(), sandbox.ServiceConfig{},
		clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk), process.NewOSExecStarter(),
	)

	before := countAuditEventsFor(repo, id)
	if err := svc.Delete(context.Background(), id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	delta := countAuditEventsFor(repo, id) - before
	if delta != 1 {
		t.Errorf("Delete emitted %d audit events, want exactly 1", delta)
	}
}

// countAuditEventsFor counts audit events recorded against a specific
// sandbox ID in the FakeRepository. Reads the exported slice directly
// — the test runs single-threaded so no synchronization is required.
func countAuditEventsFor(repo *mocks.FakeRepository, id uuid.UUID) int {
	count := 0
	for _, e := range repo.AuditEvents {
		if e.SandboxID != nil && *e.SandboxID == id {
			count++
		}
	}
	return count
}
