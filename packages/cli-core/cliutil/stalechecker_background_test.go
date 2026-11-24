//go:build unix

package cliutil

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// refreshFixture is a stale CLI whose owner refresh is a real child process: a
// script that records each launch and then runs until released. The lock and
// detach behaviour under test is real process and file-lock behaviour, so a
// fake starter would prove nothing about it.
type refreshFixture struct {
	checker *StaleChecker
	count   string
	release string
}

func newRefreshFixture(t *testing.T, scriptBody string) *refreshFixture {
	t.Helper()
	repoRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoRoot, "packages", "cli-core"), 0o755); err != nil {
		t.Fatal(err)
	}
	srcRoot := filepath.Join(repoRoot, "scenarios", "demo")
	if err := os.MkdirAll(srcRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	f := &refreshFixture{count: filepath.Join(work, "launches"), release: filepath.Join(work, "release")}
	controlPlane := filepath.Join(work, "vrooli")
	script := fmt.Sprintf("#!/bin/sh\necho \"$@\" >> %q\n%s\n", f.count, strings.ReplaceAll(scriptBody, "$RELEASE", f.release))
	if err := os.WriteFile(controlPlane, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(LaunchRuntimeBinaryEnv, controlPlane)
	t.Cleanup(func() { _ = os.WriteFile(f.release, nil, 0o644) })

	f.checker = NewStaleChecker("demo", "installed", "ts", srcRoot)
	f.checker.InstallKind = "scenario"
	f.checker.InstallName = "demo"
	f.checker.RefreshStateDir = filepath.Join(work, "state")
	f.checker.FingerprintFunc = func(FreshnessSpec) (string, error) { return "current", nil }
	f.checker.LookPathFunc = func(string) (string, error) { return "/usr/bin/go", nil }
	f.checker.CommandRunner = func(*exec.Cmd) error {
		t.Error("a latency-bound check must not run the refresh synchronously")
		return errors.New("synchronous refresh")
	}
	f.checker.Reexec = func(string, []string) error {
		t.Error("a latency-bound check must not restart the process")
		return errors.New("restart")
	}
	return f
}

const waitForRelease = `while [ ! -e "$RELEASE" ]; do sleep 0.02; done`

func (f *refreshFixture) launches() int {
	data, _ := os.ReadFile(f.count)
	return strings.Count(string(data), "\n")
}

func (f *refreshFixture) releaseRefresh() {
	_ = os.WriteFile(f.release, nil, 0o644)
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestLatencyBoundCheckReturnsWhileRefreshIsStillRunning(t *testing.T) {
	f := newRefreshFixture(t, waitForRelease)

	start := time.Now()
	f.checker.CheckLatencyBound()
	elapsed := time.Since(start)

	waitFor(t, "background refresh to start", func() bool { return f.launches() == 1 })
	// The refresh is parked on the release file, so a check that waited for it
	// would not have returned at all.
	if elapsed > 5*time.Second {
		t.Fatalf("CheckLatencyBound took %s", elapsed)
	}
	data, _ := os.ReadFile(f.count)
	if !strings.Contains(string(data), "cli-refresh scenario demo") {
		t.Fatalf("refresh launched as %q, want the owner cli-refresh command", data)
	}
}

func TestLatencyBoundCheckStartsAtMostOneRefreshWhileOneRuns(t *testing.T) {
	f := newRefreshFixture(t, waitForRelease)
	// Cooldown is irrelevant here: only the lock may dedupe these callers.
	f.checker.RefreshCooldown = time.Nanosecond

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f.checker.CheckLatencyBound()
		}()
	}
	wg.Wait()
	waitFor(t, "background refresh to start", func() bool { return f.launches() >= 1 })
	time.Sleep(200 * time.Millisecond)
	if got := f.launches(); got != 1 {
		t.Fatalf("%d refreshes started for concurrent hook calls, want 1", got)
	}

	// Once the refresh finishes the lock is free again, so a CLI that is still
	// stale (the refresh failed) gets another attempt.
	f.releaseRefresh()
	waitFor(t, "a later call to start a new refresh after the first finished", func() bool {
		f.checker.CheckLatencyBound()
		return f.launches() >= 2
	})
}

func TestLatencyBoundCheckHonoursRefreshCooldown(t *testing.T) {
	f := newRefreshFixture(t, "exit 1")
	f.checker.RefreshCooldown = time.Hour

	f.checker.CheckLatencyBound()
	waitFor(t, "first refresh to run", func() bool { return f.launches() == 1 })
	// The first refresh exited (failed) and released the lock; the cooldown,
	// not the lock, must keep every following hook call from relaunching it.
	time.Sleep(100 * time.Millisecond)
	f.checker.CheckLatencyBound()
	f.checker.CheckLatencyBound()
	time.Sleep(200 * time.Millisecond)
	if got := f.launches(); got != 1 {
		t.Fatalf("%d refreshes started inside the cooldown, want 1", got)
	}
}

func TestLatencyBoundCheckLeavesFreshCLIAlone(t *testing.T) {
	f := newRefreshFixture(t, waitForRelease)
	f.checker.FingerprintFunc = func(FreshnessSpec) (string, error) { return "installed", nil }

	f.checker.CheckLatencyBound()
	time.Sleep(100 * time.Millisecond)
	if got := f.launches(); got != 0 {
		t.Fatalf("a fresh CLI started %d refreshes", got)
	}
}

func TestLatencyBoundCheckNeverFailsTheCommand(t *testing.T) {
	f := newRefreshFixture(t, waitForRelease)
	f.checker.LookPathFunc = func(string) (string, error) { return "", errors.New("no go toolchain") }

	// A stale CLI without a usable refresh path still runs: the interactive
	// check reports an error here, the latency-bound one stays silent.
	f.checker.CheckLatencyBound()
	if got := f.launches(); got != 0 {
		t.Fatalf("refresh launched without a Go toolchain: %d", got)
	}
}

func TestLatencyBoundEnvironmentSignalAppliesToTheGenericCheck(t *testing.T) {
	f := newRefreshFixture(t, waitForRelease)
	t.Setenv(LatencyBoundEnvVar, "1")

	restarted, err := f.checker.CheckAndMaybeRebuild()
	if err != nil || restarted {
		t.Fatalf("CheckAndMaybeRebuild() = %v, %v; want no restart and no error", restarted, err)
	}
	waitFor(t, "background refresh to start", func() bool { return f.launches() == 1 })
}

func TestInteractiveCheckStillRebuildsSynchronously(t *testing.T) {
	f := newRefreshFixture(t, waitForRelease)
	ran, restarted := false, false
	f.checker.CommandRunner = func(*exec.Cmd) error { ran = true; return nil }
	f.checker.Reexec = func(string, []string) error { restarted = true; return nil }

	got, err := f.checker.CheckAndMaybeRebuild()
	if err != nil || !got || !ran || !restarted {
		t.Fatalf("interactive check = restarted %v, err %v, ran %v, reexec %v; want a synchronous rebuild and restart", got, err, ran, restarted)
	}
	if n := f.launches(); n != 0 {
		t.Fatalf("interactive check started %d background refreshes", n)
	}
}
