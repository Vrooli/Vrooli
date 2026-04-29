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

// waitForExit returns nil if the spawned cmd exits within the bounded
// timeout. The exit status itself doesn't matter — only that the
// process terminated (SIGTERM/SIGKILL counts). Reaps the zombie so
// pidAlive cleanup works.
func waitForExit(t *testing.T, cmd *exec.Cmd, timeout time.Duration) error {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return context.DeadlineExceeded
	}
}

// I-MOUNT-1: Service.Delete returns ⇒ no fuse-overlayfs daemon
// remains for the deleted sandbox UUID. This test pins the
// deterministic daemon teardown contract — the background reaper is
// NEVER consulted for these assertions, so a successful test means
// Delete itself owned the kill.
func TestDelete_Daemon_Lifecycle(t *testing.T) {
	id := uuid.New()

	// Spawn a real child process to play the role of the
	// fuse-overlayfs daemon. The cmdline embedded in the synthetic
	// /proc fixture is what the kill seam matches against — the actual
	// argv of the spawned process is irrelevant.
	cmd := exec.Command("sleep", "60")
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
			Cmdline: []byte("fuse-overlayfs\x00-o\x00lowerdir=/home,upperdir=/run/" + id.String() + "/home-upper\x00/run/" + id.String() + "/home-merged\x00"),
		},
	})

	repo := mocks.NewFakeRepository()
	repo.Sandboxes[id] = &types.Sandbox{
		ID:        id,
		Status:    types.StatusActive,
		CreatedAt: time.Now().Add(-time.Hour),
	}

	clk := clock.System{}
	svc := sandbox.NewService(
		repo, mocks.NewFakeDriver(), sandbox.ServiceConfig{},
		clk, audit.NewRepoEmitter(repo.LogAuditEvent, clk), process.NewOSExecStarter(),
		sandbox.WithProcFS(procFS),
	)

	if err := svc.Delete(context.Background(), id); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Daemon must be dead within the bounded wait. The kill seam uses
	// 5s by default; allow a small extra slack for the host scheduler.
	if err := waitForExit(t, cmd, 7*time.Second); err != nil {
		t.Fatalf("daemon helper PID %d still alive after Delete; killDaemonsForSandbox did not own the kill", cmd.Process.Pid)
	}
}

// TestDelete_Daemon_Lifecycle_AllowsRemountSameID verifies the second
// half of the contract: after Delete returns, the sandbox row is
// removed and a freshly created sandbox at the SAME id slot should
// not collide with any leftover daemon. This pins that the reaper
// did not have to run between Delete and re-create.
func TestDelete_Daemon_Lifecycle_AllowsRemountSameID(t *testing.T) {
	id := uuid.New()
	cmd := exec.Command("sleep", "60")
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
	if _, ok := repo.Sandboxes[id]; ok {
		t.Errorf("Delete should remove the sandbox row from the repo, but %s still present", id)
	}

	// Wait for the helper to actually exit before declaring the slot
	// reusable. If this loop times out, Delete didn't kill the daemon
	// synchronously.
	if err := waitForExit(t, cmd, 7*time.Second); err != nil {
		t.Fatalf("daemon helper PID %d still alive after Delete", cmd.Process.Pid)
	}
}
