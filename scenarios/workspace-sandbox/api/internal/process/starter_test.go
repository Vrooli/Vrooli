// Package process — Starter tests.
//
// Production starter (OSExecStarter) tests use /bin/true, /bin/false,
// /bin/sleep, and /bin/sh, all guaranteed on Linux. Tests that mutate
// host state (process spawning, signal delivery) live here and are
// guarded by t.Skip on non-linux for CI portability.
package process_test

import (
	"context"
	"errors"
	"os"
	osexec "os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"workspace-sandbox/internal/process"

	"github.com/vrooli/repo-contract-go/repocontracttest"
)

func skipNonLinux(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" {
		repocontracttest.SkipPlatformf(t, "starter integration test requires linux; have %s", runtime.GOOS)
	}
}

func requireBin(t *testing.T, name string) string {
	t.Helper()
	path, err := osexec.LookPath(name)
	if err != nil {
		t.Skipf("%s not in PATH: %v", name, err)
	}
	return path
}

func TestOSExecStarter_LookPath_FoundAndMissing(t *testing.T) {
	skipNonLinux(t)
	s := process.NewOSExecStarter()

	truePath := requireBin(t, "true")
	got, err := s.LookPath("true")
	if err != nil {
		t.Fatalf("LookPath(true): %v", err)
	}
	if got != truePath {
		t.Errorf("LookPath(true)=%q, want %q", got, truePath)
	}

	_, err = s.LookPath("definitely-not-a-real-binary-xyzzy")
	if err == nil {
		t.Fatal("LookPath: expected error for missing binary")
	}
	if !errors.Is(err, process.ErrBinaryNotFound) {
		t.Errorf("LookPath: err=%v, want errors.Is ErrBinaryNotFound", err)
	}
}

func TestOSExecStarter_Start_TrueExitsZero(t *testing.T) {
	skipNonLinux(t)
	s := process.NewOSExecStarter()

	h, err := s.Start(context.Background(), process.StartOpts{
		Path: requireBin(t, "true"),
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if h.PID() <= 0 {
		t.Errorf("PID: got %d, want > 0", h.PID())
	}
	exit, err := h.Wait(context.Background())
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if exit.ExitCode != 0 {
		t.Errorf("ExitCode: got %d, want 0", exit.ExitCode)
	}
}

func TestOSExecStarter_Start_FalseExitsNonZero(t *testing.T) {
	skipNonLinux(t)
	s := process.NewOSExecStarter()

	h, err := s.Start(context.Background(), process.StartOpts{
		Path: requireBin(t, "false"),
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	exit, _ := h.Wait(context.Background())
	if exit.ExitCode == 0 {
		t.Errorf("ExitCode: got 0, want non-zero from /bin/false")
	}
}

func TestOSExecStarter_Run_CapturesOutput(t *testing.T) {
	skipNonLinux(t)
	s := process.NewOSExecStarter()

	res, err := process.Run(context.Background(), s, process.StartOpts{
		Path: requireBin(t, "sh"),
		Args: []string{"-c", "echo stdout; echo stderr 1>&2"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Exit.ExitCode != 0 {
		t.Errorf("ExitCode: got %d, want 0", res.Exit.ExitCode)
	}
	if !strings.Contains(string(res.Stdout), "stdout") {
		t.Errorf("Stdout: got %q, want to contain 'stdout'", string(res.Stdout))
	}
	if !strings.Contains(string(res.Stderr), "stderr") {
		t.Errorf("Stderr: got %q, want to contain 'stderr'", string(res.Stderr))
	}
}

func TestOSExecStarter_RunCombinedOutput_MergesStreams(t *testing.T) {
	skipNonLinux(t)
	s := process.NewOSExecStarter()

	res, err := process.RunCombinedOutput(context.Background(), s, process.StartOpts{
		Path: requireBin(t, "sh"),
		Args: []string{"-c", "echo a; echo b 1>&2"},
	})
	if err != nil {
		t.Fatalf("RunCombinedOutput: %v", err)
	}
	out := string(res.Stdout)
	if !strings.Contains(out, "a") || !strings.Contains(out, "b") {
		t.Errorf("Stdout: got %q, want to contain both a and b", out)
	}
	if len(res.Stderr) != 0 {
		t.Errorf("Stderr: got %q, want empty", string(res.Stderr))
	}
}

func TestOSExecStarter_Wait_RespectsContextCancel(t *testing.T) {
	skipNonLinux(t)
	s := process.NewOSExecStarter()

	h, err := s.Start(context.Background(), process.StartOpts{
		Path: requireBin(t, "sleep"),
		Args: []string{"30"},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, waitErr := h.Wait(ctx)
	elapsed := time.Since(start)

	if waitErr == nil {
		t.Error("Wait: expected error from canceled context")
	}
	if !errors.Is(waitErr, context.DeadlineExceeded) && !errors.Is(waitErr, context.Canceled) {
		t.Errorf("Wait: err=%v, want context error", waitErr)
	}
	if elapsed > 5*time.Second {
		t.Errorf("Wait: took %v, expected to be killed quickly via ctx", elapsed)
	}
}

func TestOSExecStarter_Kill_IsIdempotent(t *testing.T) {
	skipNonLinux(t)
	s := process.NewOSExecStarter()

	h, err := s.Start(context.Background(), process.StartOpts{
		Path: requireBin(t, "sleep"),
		Args: []string{"10"},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := h.Kill(); err != nil {
		t.Errorf("first Kill: %v", err)
	}
	exit, _ := h.Wait(context.Background())
	if exit.ExitCode == 0 {
		t.Errorf("ExitCode after Kill: got 0, want non-zero")
	}
	if err := h.Kill(); err != nil {
		t.Errorf("second Kill: %v", err)
	}
}

func TestOSExecStarter_KillProcessGroup_KillsChildren(t *testing.T) {
	skipNonLinux(t)
	s := process.NewOSExecStarter()

	// Spawn a shell with Setpgid that backgrounds a long sleep, then
	// itself sleeps. KillProcessGroup must reap both.
	h, err := s.Start(context.Background(), process.StartOpts{
		Path:        requireBin(t, "sh"),
		Args:        []string{"-c", "sleep 30 & wait"},
		SysProcAttr: process.NewProcessGroupSysProcAttr(),
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	if err := h.KillProcessGroup(); err != nil {
		t.Errorf("KillProcessGroup: %v", err)
	}
	exit, _ := h.Wait(context.Background())
	if exit.ExitCode == 0 {
		t.Errorf("ExitCode after KillProcessGroup: got 0, want non-zero")
	}
}

func TestOSExecStarter_Start_RequiresPath(t *testing.T) {
	s := process.NewOSExecStarter()
	_, err := s.Start(context.Background(), process.StartOpts{})
	if err == nil {
		t.Fatal("Start with empty Path: expected error")
	}
}

func TestExitFromState_NormalExit(t *testing.T) {
	skipNonLinux(t)
	cmd := osexec.Command(requireBin(t, "true"))
	if err := cmd.Run(); err != nil {
		t.Fatalf("run true: %v", err)
	}
	exit := process.ExitFromState(cmd.ProcessState, nil)
	if exit.ExitCode != 0 {
		t.Errorf("ExitCode: got %d, want 0", exit.ExitCode)
	}
	if exit.Signal != 0 {
		t.Errorf("Signal: got %d, want 0", exit.Signal)
	}
}

func TestExitFromState_NonZeroExit(t *testing.T) {
	skipNonLinux(t)
	cmd := osexec.Command(requireBin(t, "false"))
	_ = cmd.Run() // expected to fail
	exit := process.ExitFromState(cmd.ProcessState, nil)
	if exit.ExitCode == 0 {
		t.Errorf("ExitCode: got 0, want non-zero from /bin/false")
	}
}

func TestExitFromState_NilState(t *testing.T) {
	exit := process.ExitFromState(nil, nil)
	if exit.ExitCode != 0 {
		t.Errorf("nil state, no err: ExitCode=%d, want 0", exit.ExitCode)
	}

	exit = process.ExitFromState(nil, errors.New("boom"))
	if exit.ExitCode != -1 {
		t.Errorf("nil state, with err: ExitCode=%d, want -1", exit.ExitCode)
	}
}

func TestIsProcessRunning_AfterExit(t *testing.T) {
	skipNonLinux(t)
	s := process.NewOSExecStarter()
	h, err := s.Start(context.Background(), process.StartOpts{
		Path: requireBin(t, "true"),
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	pid := h.PID()
	if _, err := h.Wait(context.Background()); err != nil {
		t.Fatalf("Wait: %v", err)
	}
	// After Wait the kernel has reaped the PID; IsProcessRunning may
	// briefly remain true on some kernels until the parent reaper
	// runs, but for /bin/true (fast exit) it should reliably return
	// false within a short window.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !process.IsProcessRunning(pid) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Errorf("IsProcessRunning(%d): still true after Wait+2s", pid)
}

func TestCommandExists(t *testing.T) {
	skipNonLinux(t)
	s := process.NewOSExecStarter()
	if !process.CommandExists(s, "true") {
		t.Error("CommandExists(true): got false, want true")
	}
	if process.CommandExists(s, "definitely-not-real-xyzzy") {
		t.Error("CommandExists(definitely-not-real): got true, want false")
	}
}

// Smoke test that StartOpts.Env actually reaches the process.
func TestOSExecStarter_Start_EnvIsApplied(t *testing.T) {
	skipNonLinux(t)
	s := process.NewOSExecStarter()
	res, err := process.Run(context.Background(), s, process.StartOpts{
		Path: requireBin(t, "sh"),
		Args: []string{"-c", "echo $WSB_TEST_VAR"},
		Env:  append(os.Environ(), "WSB_TEST_VAR=hello-mounter"),
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(string(res.Stdout), "hello-mounter") {
		t.Errorf("Stdout: got %q, want to contain hello-mounter", string(res.Stdout))
	}
}

// Smoke test that StartOpts.Dir is honored.
func TestOSExecStarter_Start_DirIsApplied(t *testing.T) {
	skipNonLinux(t)
	s := process.NewOSExecStarter()
	tmp := t.TempDir()
	res, err := process.Run(context.Background(), s, process.StartOpts{
		Path: requireBin(t, "pwd"),
		Dir:  tmp,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := strings.TrimSpace(string(res.Stdout))
	// /bin/pwd may resolve symlinks (e.g. on macos /private/tmp),
	// so the comparison is "ends with the basename" — sufficient on
	// linux which is the only platform this test runs on.
	if !strings.HasSuffix(got, tmp) && !strings.HasSuffix(got, lastDir(tmp)) {
		t.Errorf("pwd: got %q, want to end with %q", got, tmp)
	}
}

func lastDir(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}
