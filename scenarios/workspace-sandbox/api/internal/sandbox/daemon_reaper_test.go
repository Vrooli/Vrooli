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
	"workspace-sandbox/internal/sandbox"
	"workspace-sandbox/internal/testutil/mocks"
	"workspace-sandbox/internal/testutil/mocks/sandboxiface"
	"workspace-sandbox/internal/types"
)

// TestReapStaleDaemons_FindsOrphan_SkipsLive_HonorsGrace covers the
// three branches of the per-PID decision tree: orphan (reap), live
// (skip), young (skip).
func TestReapStaleDaemons_FindsOrphan_SkipsLive_HonorsGrace(t *testing.T) {
	// Spawn three real-but-trivial child processes whose PIDs we feed
	// to the synthetic /proc fixture. Killing them is the only behavior
	// we observe, so use `sleep 60` (cheap, easy to verify alive/dead).
	cmds := make([]*exec.Cmd, 3)
	for i := range cmds {
		cmds[i] = exec.Command("sleep", "60")
		if err := cmds[i].Start(); err != nil {
			t.Fatalf("spawn helper #%d: %v", i, err)
		}
	}
	t.Cleanup(func() {
		for _, c := range cmds {
			if c.Process != nil {
				_ = c.Process.Kill()
				_ = c.Wait()
			}
		}
	})

	orphanID := uuid.New()
	liveID := uuid.New()
	youngID := uuid.New()
	now := time.Now()

	fixture := sandboxiface.NewFakeProcFS(map[string]sandboxiface.FakeProcEntry{
		strconv.Itoa(cmds[0].Process.Pid): {
			Cmdline:   []byte("fuse-overlayfs\x00-o\x00lowerdir=/home,upperdir=/run/" + orphanID.String() + "/home-upper,workdir=/run/" + orphanID.String() + "/home-work\x00/run/" + orphanID.String() + "/home-merged\x00"),
			StartTime: now.Add(-1 * time.Hour),
		},
		strconv.Itoa(cmds[1].Process.Pid): {
			Cmdline:   []byte("fuse-overlayfs\x00-o\x00lowerdir=/home,upperdir=/run/" + liveID.String() + "/home-upper,workdir=/run/" + liveID.String() + "/home-work\x00/run/" + liveID.String() + "/home-merged\x00"),
			StartTime: now.Add(-1 * time.Hour),
		},
		strconv.Itoa(cmds[2].Process.Pid): {
			Cmdline:   []byte("fuse-overlayfs\x00-o\x00upperdir=/run/" + youngID.String() + "/home-upper\x00/run/" + youngID.String() + "/home-merged\x00"),
			StartTime: now, // within grace
		},
	})

	repo := mocks.NewFakeRepository()
	// Live sandbox lives in the repo with StatusActive; orphanID and
	// youngID are absent, which the reaper treats as orphans.
	repo.SetSandbox(&types.Sandbox{ID: liveID, Status: types.StatusActive})
	svc := sandbox.NewService(repo, nil, sandbox.ServiceConfig{}, clock.System{}, audit.NewRepoEmitter(repo.LogAuditEvent, clock.System{}))

	cfg := sandbox.DaemonReaperConfig{GracePeriod: 30 * time.Second, TermWait: 200 * time.Millisecond}
	report := svc.ReconcileStaleDaemonsWithConfig(context.Background(), cfg, fixture)

	if report.Scanned != 3 {
		t.Errorf("Scanned = %d, want 3", report.Scanned)
	}
	if report.Reaped != 1 {
		t.Errorf("Reaped = %d, want 1 (orphan only)", report.Reaped)
	}
	if report.SkippedAlive != 1 {
		t.Errorf("SkippedAlive = %d, want 1", report.SkippedAlive)
	}
	if report.SkippedYoung != 1 {
		t.Errorf("SkippedYoung = %d, want 1", report.SkippedYoung)
	}

	// Verify the orphan helper actually died, and the live one survived.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cmds[0].ProcessState != nil {
			break
		}
		_, _ = cmds[0].Process.Wait() // non-blocking via WNOHANG would be nicer; just retry
		time.Sleep(20 * time.Millisecond)
	}
	if cmds[1].ProcessState != nil && cmds[1].ProcessState.Exited() {
		t.Errorf("live helper PID %d should still be running", cmds[1].Process.Pid)
	}
}
