package exec

// contract_test.go — failure-mode contract tests for the exec layer.
//
// run_test.go covers the real-binary happy-path: actually spawning
// `true`, `sh -c "exit 7"`, etc. via OSExecStarter. That coverage is
// load-bearing for end-to-end correctness, but real-binary tests can't
// deterministically simulate every failure mode (OOM-kill, signaled
// process exit, bwrap-not-installed, hung process exceeding wall
// clock). This file pins those failure modes against a FakeStarter so
// the contract is observable on any host inside `go test`.
//
// Coverage matrix (mirrors plan §8 Phase 4):
//
//   - ExitZero (fast path)         — Exec returns ExitCode=0, no error
//   - ExitNonZero                  — Exec returns ExitCode=7, no error
//   - SignalKilled (SIGKILL)       — surfaced via StartProcess.OnExit signal arg
//   - OOMKilled                    — surfaced via StartProcess.OnExit oomKilled arg
//   - WallClockTimeout (124)       — Exec returns ExitCode=124, non-nil Error
//   - StartError propagation       — Exec/StartProcess surface starter.Start failures
//   - BwrapRequired_NoBwrap        — Exec hard-errors with structured message
//   - BwrapPreferred_NoBwrap       — Exec falls back to direct exec
//   - BwrapPreferred_HasBwrap      — Exec routes through bwrap
//   - WaitError propagation        — Wait error reported alongside ExecResult
//
// All tests use FakeStarter from internal/testutil/mocks/procmocks. No
// real binary is invoked; every assertion is deterministic and
// independent of the host's PATH.

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/testutil/mocks/procmocks"
	"workspace-sandbox/internal/types"
)

// newSandboxFor returns a fully-wired sandbox stub usable by Exec. The
// MergedDir must be non-empty (Exec rejects empty merged dirs early).
func newSandboxFor(t *testing.T) *types.Sandbox {
	t.Helper()
	return &types.Sandbox{
		ID:        uuid.New(),
		MergedDir: t.TempDir(),
		LowerDir:  t.TempDir(),
	}
}

// TestExecContract_ExitZero pins the happy path: a process exits 0
// quickly, Exec reports ExitCode=0 with no error.
func TestExecContract_ExitZero(t *testing.T) {
	starter := procmocks.NewFakeStarter()
	starter.AddCommand("/bin/echo hello", procmocks.CommandBehavior{
		Exit:   process.ProcessExit{ExitCode: 0},
		Stdout: []byte("hello\n"),
	})
	res, err := Exec(context.Background(), starter, newSandboxFor(t), driver.ContainmentNone, DefaultBwrapConfig(), "/bin/echo", "hello")
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode: got %d, want 0", res.ExitCode)
	}
	if res.Error != nil {
		t.Errorf("Error: got %v, want nil", res.Error)
	}
	if string(res.Stdout) != "hello\n" {
		t.Errorf("Stdout: got %q, want %q", res.Stdout, "hello\n")
	}
}

// TestExecContract_ExitNonZero pins: a process exits non-zero, Exec
// reports the exit code with no error (the convention is "ran fine,
// the program decided to exit non-zero" — that's a runtime outcome,
// not an exec-layer failure).
func TestExecContract_ExitNonZero(t *testing.T) {
	starter := procmocks.NewFakeStarter()
	starter.AddCommand("/bin/false", procmocks.CommandBehavior{
		Exit: process.ProcessExit{ExitCode: 7},
	})
	res, err := Exec(context.Background(), starter, newSandboxFor(t), driver.ContainmentNone, DefaultBwrapConfig(), "/bin/false")
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if res.ExitCode != 7 {
		t.Errorf("ExitCode: got %d, want 7", res.ExitCode)
	}
	if res.Error != nil {
		t.Errorf("Error: got %v, want nil", res.Error)
	}
}

// TestExecContract_StartError pins: when starter.Start fails (e.g.,
// fork EAGAIN, "binary not found" surfaced from the syscall), Exec
// surfaces the error in result.Error with a non-zero exit code.
func TestExecContract_StartError(t *testing.T) {
	starter := procmocks.NewFakeStarter()
	starter.AddCommand("/bin/echo", procmocks.CommandBehavior{
		StartErr: errors.New("fork EAGAIN"),
	})
	res, err := Exec(context.Background(), starter, newSandboxFor(t), driver.ContainmentNone, DefaultBwrapConfig(), "/bin/echo", "test")
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if res.Error == nil {
		t.Fatal("expected non-nil Error from StartErr")
	}
	if !strings.Contains(res.Error.Error(), "fork EAGAIN") {
		t.Errorf("Error should mention fork EAGAIN: %v", res.Error)
	}
	// Convention: failed-to-start gets ExitCode=-1 to distinguish from
	// "exited 0 successfully."
	if res.ExitCode != -1 {
		t.Errorf("ExitCode on StartErr: got %d, want -1", res.ExitCode)
	}
}

// TestExecContract_WaitError pins: Wait can return an error (e.g.,
// "process killed externally") alongside an exit state. The exec
// layer must surface that as result.Error and preserve the exit code
// the runtime reported.
func TestExecContract_WaitError(t *testing.T) {
	starter := procmocks.NewFakeStarter()
	starter.AddCommand("/bin/echo", procmocks.CommandBehavior{
		Exit:    process.ProcessExit{ExitCode: -1, Signal: 9},
		WaitErr: errors.New("waitpid: no child"),
	})
	res, err := Exec(context.Background(), starter, newSandboxFor(t), driver.ContainmentNone, DefaultBwrapConfig(), "/bin/echo", "x")
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if res.Error == nil {
		t.Fatal("expected non-nil Error from WaitErr")
	}
	if !strings.Contains(res.Error.Error(), "waitpid") {
		t.Errorf("Error should mention waitpid: %v", res.Error)
	}
}

// TestExecContract_WallClockTimeout pins the 124 contract:
// cfg.ResourceLimits.TimeoutSec triggers a context deadline; Exec
// must rewrite the exit code to 124 and surface a clear timeout
// message. The Phase 4 plan calls this out explicitly because
// handlers/process.go keys off ExitCode==124 to surface TimedOut=true.
func TestExecContract_WallClockTimeout(t *testing.T) {
	starter := procmocks.NewFakeStarter()
	// Hold blocks Wait until Release; ctx deadline kicks in first.
	starter.AddCommand("/bin/sleep 5", procmocks.CommandBehavior{Hold: true})
	cfg := DefaultBwrapConfig()
	cfg.ResourceLimits.TimeoutSec = 1
	start := time.Now()
	res, err := Exec(context.Background(), starter, newSandboxFor(t), driver.ContainmentNone, cfg, "/bin/sleep", "5")
	dur := time.Since(start)
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if res.ExitCode != 124 {
		t.Errorf("ExitCode on timeout: got %d, want 124", res.ExitCode)
	}
	if res.Error == nil || !strings.Contains(res.Error.Error(), "timed out") {
		t.Errorf("Error should mention timeout, got %v", res.Error)
	}
	// Sanity: timeout actually fired around the configured second; we
	// allow a generous upper bound for slow CI but the lower bound
	// catches a regression where the deadline isn't being applied.
	if dur < 900*time.Millisecond {
		t.Errorf("returned in %s, want ≥ ~1s (deadline was 1s)", dur)
	}
	if dur > 5*time.Second {
		t.Errorf("returned in %s, want ≤ 5s (deadline was 1s)", dur)
	}
}

// TestExecContract_BwrapRequired_NoBwrap pins the hard-fail behavior
// of driver.ContainmentRequired when bwrap is not on PATH. Used by kernel
// overlayfs flavors which can't fall back to direct exec.
func TestExecContract_BwrapRequired_NoBwrap(t *testing.T) {
	starter := procmocks.NewFakeStarter()
	// LookPath table empty → bwrap returns ErrBinaryNotFound.
	res, err := Exec(context.Background(), starter, newSandboxFor(t), driver.ContainmentRequired, DefaultBwrapConfig(), "/bin/echo", "x")
	if err == nil {
		t.Fatal("expected error from bwrap-required without bwrap")
	}
	if !strings.Contains(err.Error(), "bwrap") {
		t.Errorf("error should mention bwrap, got %v", err)
	}
	if res != nil {
		t.Errorf("expected nil result on bwrap-required failure, got %+v", res)
	}
}

// TestExecContract_BwrapPreferred_NoBwrap_FallsBackDirect pins:
// when mode is driver.ContainmentPreferred and bwrap is missing, Exec falls
// back to direct execution (used by fuse-overlayfs whose mount is
// host-visible). The fallback must run the command in s.MergedDir.
func TestExecContract_BwrapPreferred_NoBwrap_FallsBackDirect(t *testing.T) {
	starter := procmocks.NewFakeStarter()
	// Note: no bwrap in LookPath. Direct exec command must succeed.
	starter.AddCommand("/bin/echo direct", procmocks.CommandBehavior{
		Exit: process.ProcessExit{ExitCode: 0},
	})
	sb := newSandboxFor(t)
	res, err := Exec(context.Background(), starter, sb, driver.ContainmentPreferred, DefaultBwrapConfig(), "/bin/echo", "direct")
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode: got %d, want 0", res.ExitCode)
	}
	// Verify the start was the direct exec, not bwrap.
	calls := starter.Calls
	if len(calls) == 0 {
		t.Fatal("expected at least one Start call")
	}
	last := calls[len(calls)-1]
	if last.Path != "/bin/echo" {
		t.Errorf("preferred fallback should call /bin/echo directly, got %q", last.Path)
	}
	if last.Dir != sb.MergedDir {
		t.Errorf("Dir: got %q, want %q", last.Dir, sb.MergedDir)
	}
}

// TestExecContract_BwrapPreferred_HasBwrap pins: when bwrap is on
// PATH, driver.ContainmentPreferred routes through the bwrap exec. The
// resulting Start call uses bwrap as the path (resolved via LookPath)
// and includes the user command somewhere in the args.
func TestExecContract_BwrapPreferred_HasBwrap(t *testing.T) {
	starter := procmocks.NewFakeStarter()
	starter.SetLookPath("bwrap", "/usr/bin/bwrap")
	starter.AddCommand("/usr/bin/bwrap", procmocks.CommandBehavior{
		Exit: process.ProcessExit{ExitCode: 0},
	})
	sb := newSandboxFor(t)
	res, err := Exec(context.Background(), starter, sb, driver.ContainmentPreferred, DefaultBwrapConfig(), "/bin/echo", "via-bwrap")
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode: got %d, want 0", res.ExitCode)
	}
	calls := starter.MatchedCalls("/usr/bin/bwrap")
	if len(calls) != 1 {
		t.Fatalf("expected 1 bwrap call, got %d", len(calls))
	}
	// User command must appear somewhere in the bwrap args.
	joined := strings.Join(calls[0].Args, " ")
	if !strings.Contains(joined, "/bin/echo") || !strings.Contains(joined, "via-bwrap") {
		t.Errorf("bwrap args should contain user cmd, got %q", joined)
	}
}

// TestExecContract_BwrapRequired_ResourceLimitsRequirePrlimit pins:
// when ResourceLimits.HasLimits() is true but prlimit is missing,
// the error message names prlimit specifically (so operators can
// install util-linux instead of guessing about bwrap). The wrapper
// exists because BuildExecCommand routes through prlimit when limits
// are set.
func TestExecContract_BwrapRequired_ResourceLimitsRequirePrlimit(t *testing.T) {
	starter := procmocks.NewFakeStarter()
	starter.SetLookPath("bwrap", "/usr/bin/bwrap")
	// Note: prlimit NOT in LookPath table → ErrBinaryNotFound.
	cfg := DefaultBwrapConfig()
	cfg.ResourceLimits.MemoryLimitMB = 256
	_, err := Exec(context.Background(), starter, newSandboxFor(t), driver.ContainmentRequired, cfg, "/bin/echo", "x")
	if err == nil {
		t.Fatal("expected error when ResourceLimits set but prlimit missing")
	}
	if !strings.Contains(err.Error(), "prlimit") {
		t.Errorf("error should mention prlimit, got %v", err)
	}
}

// TestExecContract_StartProcess_OnExitFastExit pins: a process that
// exits 0 fires OnExit exactly once with (0, 0, false). Mirrors the
// real-binary TestStartProcess_OnExitFiresExactlyOnce but is
// deterministic — no race against process scheduling.
func TestExecContract_StartProcess_OnExitFastExit(t *testing.T) {
	starter := procmocks.NewFakeStarter()
	starter.AddCommand("/bin/true", procmocks.CommandBehavior{
		Exit: process.ProcessExit{ExitCode: 0},
	})

	var fires atomic.Int32
	var gotExit, gotSignal int
	var gotOOM bool
	var mu sync.Mutex
	done := make(chan struct{})
	cfg := DefaultBwrapConfig()
	cfg.OnExit = func(exitCode, signal int, oom bool) {
		mu.Lock()
		defer mu.Unlock()
		gotExit, gotSignal, gotOOM = exitCode, signal, oom
		if fires.Add(1) == 1 {
			close(done)
		}
	}

	pid, _, err := StartProcess(context.Background(), starter, newSandboxFor(t), driver.ContainmentNone, cfg, "/bin/true")
	if err != nil {
		t.Fatalf("StartProcess: %v", err)
	}
	if pid <= 0 {
		t.Errorf("PID: got %d, want > 0", pid)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("OnExit did not fire within 2s")
	}
	// Catch any spurious second invocation.
	time.Sleep(50 * time.Millisecond)
	if got := fires.Load(); got != 1 {
		t.Errorf("OnExit fires: got %d, want 1", got)
	}
	mu.Lock()
	defer mu.Unlock()
	if gotExit != 0 || gotSignal != 0 || gotOOM {
		t.Errorf("OnExit args: got (%d, %d, %v), want (0, 0, false)", gotExit, gotSignal, gotOOM)
	}
}

// TestExecContract_StartProcess_OnExitSignalKilled pins: a signal
// kill (e.g. SIGKILL, signal 9) propagates through OnExit's signal
// arg. The exec layer translates ProcessExit.Signal into the OnExit
// signal int verbatim.
func TestExecContract_StartProcess_OnExitSignalKilled(t *testing.T) {
	starter := procmocks.NewFakeStarter()
	starter.AddCommand("/bin/sleep 100", procmocks.CommandBehavior{
		Exit: process.ProcessExit{ExitCode: -1, Signal: 9},
	})

	var gotExit, gotSignal atomic.Int32
	done := make(chan struct{})
	cfg := DefaultBwrapConfig()
	cfg.OnExit = func(exitCode, signal int, _ bool) {
		gotExit.Store(int32(exitCode))
		gotSignal.Store(int32(signal))
		close(done)
	}

	if _, _, err := StartProcess(context.Background(), starter, newSandboxFor(t), driver.ContainmentNone, cfg, "/bin/sleep", "100"); err != nil {
		t.Fatalf("StartProcess: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("OnExit did not fire within 2s")
	}
	if gotExit.Load() != -1 {
		t.Errorf("OnExit exitCode: got %d, want -1", gotExit.Load())
	}
	if gotSignal.Load() != 9 {
		t.Errorf("OnExit signal: got %d, want 9 (SIGKILL)", gotSignal.Load())
	}
}

// TestExecContract_StartProcess_OnExitOOMKilled pins: OOM-killed
// processes propagate the oomKilled=true flag through OnExit. This
// is the regression guard for the metric/audit pipeline that keys
// off oom-killed events.
func TestExecContract_StartProcess_OnExitOOMKilled(t *testing.T) {
	starter := procmocks.NewFakeStarter()
	starter.AddCommand("/bin/heavy", procmocks.CommandBehavior{
		Exit: process.ProcessExit{ExitCode: -1, Signal: 9, OOMKilled: true},
	})

	var gotOOM atomic.Bool
	done := make(chan struct{})
	cfg := DefaultBwrapConfig()
	cfg.OnExit = func(_ int, _ int, oom bool) {
		gotOOM.Store(oom)
		close(done)
	}

	if _, _, err := StartProcess(context.Background(), starter, newSandboxFor(t), driver.ContainmentNone, cfg, "/bin/heavy"); err != nil {
		t.Fatalf("StartProcess: %v", err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("OnExit did not fire within 2s")
	}
	if !gotOOM.Load() {
		t.Errorf("OnExit OOMKilled: got false, want true")
	}
}

// TestExecContract_StartProcess_OnExitNilStillReaped pins: when
// OnExit is nil, the spawned process is still Wait'd on so it can't
// linger as a zombie. Verified by ensuring no deadlock and that a
// follow-up Exec sees a fresh Calls slice (i.e. the goroutine
// completed).
func TestExecContract_StartProcess_OnExitNilStillReaped(t *testing.T) {
	starter := procmocks.NewFakeStarter()
	starter.AddCommand("/bin/true", procmocks.CommandBehavior{
		Exit: process.ProcessExit{ExitCode: 0},
	})
	cfg := DefaultBwrapConfig()
	cfg.OnExit = nil

	if _, _, err := StartProcess(context.Background(), starter, newSandboxFor(t), driver.ContainmentNone, cfg, "/bin/true"); err != nil {
		t.Fatalf("StartProcess: %v", err)
	}
	// Allow the reaper goroutine to run.
	time.Sleep(50 * time.Millisecond)
	// Sanity: a subsequent Exec works (the fake starter isn't deadlocked).
	starter.AddCommand("/bin/echo done", procmocks.CommandBehavior{
		Exit: process.ProcessExit{ExitCode: 0},
	})
	res, err := Exec(context.Background(), starter, newSandboxFor(t), driver.ContainmentNone, DefaultBwrapConfig(), "/bin/echo", "done")
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("follow-up Exec failed: err=%v res=%+v", err, res)
	}
}

// TestExecContract_StartProcess_StartErrorSurfaces pins: starter.Start
// failures (fork errors, "binary not found") surface as the
// StartProcess error directly — OnExit must NOT fire when Start
// fails (no process exists to reap).
func TestExecContract_StartProcess_StartErrorSurfaces(t *testing.T) {
	starter := procmocks.NewFakeStarter()
	starter.AddCommand("/bin/missing", procmocks.CommandBehavior{
		StartErr: errors.New("fork failed"),
	})

	var fires atomic.Int32
	cfg := DefaultBwrapConfig()
	cfg.OnExit = func(_, _ int, _ bool) { fires.Add(1) }

	pid, _, err := StartProcess(context.Background(), starter, newSandboxFor(t), driver.ContainmentNone, cfg, "/bin/missing")
	if err == nil {
		t.Fatal("expected StartProcess to surface fork failure")
	}
	if !strings.Contains(err.Error(), "fork failed") {
		t.Errorf("error should mention fork failed, got %v", err)
	}
	if pid != 0 {
		t.Errorf("PID on Start failure: got %d, want 0", pid)
	}
	// Generous wait to confirm OnExit doesn't fire.
	time.Sleep(50 * time.Millisecond)
	if fires.Load() != 0 {
		t.Errorf("OnExit fired %d times after Start failure, want 0", fires.Load())
	}
}

// TestExecContract_StartProcess_StdoutPiped pins: cfg.StdoutWriter
// receives the spawned process's stdout. The underlying FakeHandle
// flushes scripted Stdout at Wait time so the writer must catch it
// before the OnExit reaper fires.
func TestExecContract_StartProcess_StdoutPiped(t *testing.T) {
	starter := procmocks.NewFakeStarter()
	starter.AddCommand("/bin/echo piped", procmocks.CommandBehavior{
		Exit:   process.ProcessExit{ExitCode: 0},
		Stdout: []byte("piped\n"),
	})

	var stdout bytes.Buffer
	done := make(chan struct{})
	cfg := DefaultBwrapConfig()
	cfg.StdoutWriter = &stdout
	cfg.OnExit = func(_, _ int, _ bool) { close(done) }

	if _, _, err := StartProcess(context.Background(), starter, newSandboxFor(t), driver.ContainmentNone, cfg, "/bin/echo", "piped"); err != nil {
		t.Fatalf("StartProcess: %v", err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("OnExit did not fire within 2s")
	}
	if got := stdout.String(); got != "piped\n" {
		t.Errorf("stdout: got %q, want %q", got, "piped\n")
	}
}

// TestExecContract_RejectsUnmountedSandbox pins: Exec / StartProcess
// refuse to run when MergedDir is empty (the sandbox is not mounted).
// This is the load-bearing precondition that prevents agent commands
// from running in the wrong filesystem after a Stop.
func TestExecContract_RejectsUnmountedSandbox(t *testing.T) {
	starter := procmocks.NewFakeStarter()
	starter.SetDefault(procmocks.CommandBehavior{Exit: process.ProcessExit{ExitCode: 0}})
	sb := &types.Sandbox{ID: uuid.New()} // no MergedDir
	if _, err := Exec(context.Background(), starter, sb, driver.ContainmentNone, DefaultBwrapConfig(), "/bin/true"); err == nil {
		t.Error("Exec should reject sandbox with empty MergedDir")
	}
	if _, _, err := StartProcess(context.Background(), starter, sb, driver.ContainmentNone, DefaultBwrapConfig(), "/bin/true"); err == nil {
		t.Error("StartProcess should reject sandbox with empty MergedDir")
	}
}

// TestExecContract_NilStarterPanics pins: passing a nil starter
// panics with a structured message at the call boundary so wiring
// bugs surface loud at startup rather than nil-dereffing inside the
// run.
func TestExecContract_NilStarterPanics(t *testing.T) {
	cases := []struct {
		name string
		fn   func()
	}{
		{
			name: "Exec",
			fn: func() {
				_, _ = Exec(context.Background(), nil, newSandboxFor(t), driver.ContainmentNone, DefaultBwrapConfig(), "/bin/true")
			},
		},
		{
			name: "StartProcess",
			fn: func() {
				_, _, _ = StartProcess(context.Background(), nil, newSandboxFor(t), driver.ContainmentNone, DefaultBwrapConfig(), "/bin/true")
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("%s with nil starter should panic", tc.name)
				}
			}()
			tc.fn()
		})
	}
}
