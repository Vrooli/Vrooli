package exec

import (
	"context"
	"os"
	osexec "os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/types"
)

func TestIsBwrapAvailable(t *testing.T) {
	ctx := context.Background()
	starter := process.NewOSExecStarter()
	_, lookErr := osexec.LookPath("bwrap")
	if lookErr != nil {
		// Not installed in CI; assert the probe agrees.
		available, _, _ := driver.IsBwrapAvailable(ctx, starter)
		if available {
			t.Error("driver.IsBwrapAvailable returned true but bwrap is not in PATH")
		}
		return
	}
	available, version, err := driver.IsBwrapAvailable(ctx, starter)
	if !available {
		t.Errorf("driver.IsBwrapAvailable returned false but bwrap is installed: %v", err)
	}
	if version == "" {
		t.Log("warning: bwrap version string is empty")
	}
}

func TestIsProcessRunning(t *testing.T) {
	if !IsProcessRunning(os.Getpid()) {
		t.Error("current process should be running")
	}
	if IsProcessRunning(999999) {
		t.Error("non-existent process should not be running")
	}
}

func TestExitInfoFromState_NormalExit(t *testing.T) {
	// Running a successful command and reading its state.
	cmd := osexec.Command("true")
	if err := cmd.Run(); err != nil {
		t.Fatalf("`true` failed unexpectedly: %v", err)
	}
	exitCode, signal, oom := ExitInfoFromState(cmd.ProcessState, nil)
	if exitCode != 0 || signal != 0 || oom {
		t.Errorf("ExitInfoFromState(true) = (%d, %d, %v), want (0, 0, false)", exitCode, signal, oom)
	}
}

func TestExitInfoFromState_NonZeroExit(t *testing.T) {
	cmd := osexec.Command("sh", "-c", "exit 7")
	err := cmd.Run()
	exitCode, signal, oom := ExitInfoFromState(cmd.ProcessState, err)
	if exitCode != 7 || signal != 0 || oom {
		t.Errorf("ExitInfoFromState(exit 7) = (%d, %d, %v), want (7, 0, false)", exitCode, signal, oom)
	}
}

// TestExec_TimeoutReturns124 verifies that wall-clock timeout enforcement
// returns the standard exit code 124 — required for handlers/process.go to
// surface TimedOut=true to the client.
func TestExec_TimeoutReturns124(t *testing.T) {
	tmp := t.TempDir()
	sandbox := &types.Sandbox{
		ID:        uuid.New(),
		MergedDir: tmp,
		LowerDir:  tmp,
	}
	cfg := DefaultBwrapConfig()
	cfg.ResourceLimits.TimeoutSec = 1
	// driver.ContainmentNone runs in s.MergedDir directly, no bwrap dependency.
	result, err := Exec(context.Background(), process.NewOSExecStarter(), sandbox, driver.ContainmentNone, cfg, "sh", "-c", "sleep 5")
	if err != nil {
		t.Fatalf("Exec returned unexpected error: %v", err)
	}
	if result.ExitCode != 124 {
		t.Errorf("expected ExitCode=124 on timeout, got %d", result.ExitCode)
	}
	if result.Error == nil {
		t.Error("expected non-nil result.Error on timeout")
	}
}

// TestStartProcess_OnExitFiresExactlyOnce locks in the OnExit reaper
// invariant: the callback is invoked exactly once per StartProcess, even
// when the process exits very quickly.
func TestStartProcess_OnExitFiresExactlyOnce(t *testing.T) {
	tmp := t.TempDir()
	sandbox := &types.Sandbox{
		ID:        uuid.New(),
		MergedDir: tmp,
		LowerDir:  tmp,
	}
	cfg := DefaultBwrapConfig()
	var fires atomic.Int32
	done := make(chan struct{})
	cfg.OnExit = func(exitCode, signal int, oomKilled bool) {
		fires.Add(1)
		close(done)
	}

	pid, _, err := StartProcess(context.Background(), process.NewOSExecStarter(), sandbox, driver.ContainmentNone, cfg, "true")
	if err != nil {
		t.Fatalf("StartProcess: %v", err)
	}
	if pid <= 0 {
		t.Errorf("expected positive PID, got %d", pid)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("OnExit did not fire within 5s")
	}

	// Sleep a moment to catch any spurious second invocation.
	time.Sleep(50 * time.Millisecond)
	if got := fires.Load(); got != 1 {
		t.Errorf("OnExit fired %d times, want exactly 1", got)
	}
}

// TestStartProcess_WaitHappensWhenOnExitNil ensures we still reap zombies
// when OnExit is not configured (background processes always need a
// goroutine that calls Wait).
func TestStartProcess_WaitHappensWhenOnExitNil(t *testing.T) {
	tmp := t.TempDir()
	sandbox := &types.Sandbox{
		ID:        uuid.New(),
		MergedDir: tmp,
		LowerDir:  tmp,
	}
	cfg := DefaultBwrapConfig()
	cfg.OnExit = nil

	// Use a marker file to know when the child has exited; that's an
	// indirect signal that the reaper goroutine ran (cmd.Wait returned).
	marker := filepath.Join(tmp, "done")
	pid, _, err := StartProcess(context.Background(), process.NewOSExecStarter(), sandbox, driver.ContainmentNone, cfg, "sh", "-c", "touch "+marker)
	if err != nil {
		t.Fatalf("StartProcess: %v", err)
	}
	if pid <= 0 {
		t.Fatalf("expected positive PID, got %d", pid)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("child never wrote marker file: %v", err)
	}

	// Give the reaper a chance to call Wait. If it didn't, the process
	// would still be in 'Z' state — best-effort check via /proc.
	time.Sleep(100 * time.Millisecond)
	if data, err := os.ReadFile(filepath.Join("/proc", itoa(pid), "status")); err == nil {
		if containsString(string(data), "State:\tZ") {
			t.Error("child is a zombie — Wait was not called by the reaper")
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func containsString(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
