package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

const applyRunnerProcessHelper = "VROOLI_ONBOARDING_APPLY_RUNNER_PROCESS_HELPER"

// TestApplyRunnerProcessParentHelper is run as a short-lived API-parent
// substitute. The actual runner uses a separately built server executable,
// because the Go test binary cannot enter production runner mode.
func TestApplyRunnerProcessParentHelper(t *testing.T) {
	if os.Getenv(applyRunnerProcessHelper) == "" {
		return
	}
	t.Setenv("VROOLI_ROOT", os.Getenv("SE2_TEST_REPO_ROOT"))
	t.Setenv("VROOLI_STATE_ROOT", os.Getenv("SE2_TEST_STATE_ROOT"))
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	t.Setenv("BUNDLE_ROOT", "")
	if err := configureOperatorStateRoots(); err != nil {
		t.Fatalf("configure test roots: %v", err)
	}
	applyRunnerExecutablePath = os.Getenv("VROOLI_ONBOARDING_APPLY_RUNNER_IMAGE")
	run, err := loadPersistedApplyRun(os.Getenv("VROOLI_ONBOARDING_APPLY_RUN_ID"))
	if err != nil {
		t.Fatalf("load persisted run: %v", err)
	}
	if err := spawnApplyRunner(context.Background(), run); err != nil {
		t.Fatalf("spawn detached runner: %v", err)
	}
}

// TestDetachedApplySurvivesParentExitAndComponentReplacement exercises the
// OS process boundary rather than the in-process executor substitute. The fake
// Vrooli command blocks safely in a temporary state root; it never mutates the
// host. A short-lived parent exits, the scenario component executable is
// removed to model replacement, and the detached runner must publish its
// terminal result through the durable run record.
func TestDetachedApplySurvivesParentExitAndComponentReplacement(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("setsid survival proof currently targets Linux")
	}
	root := t.TempDir()
	repo, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	stateRoot := filepath.Join(root, "state")
	home := filepath.Join(root, "home")
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatal(err)
	}
	componentDir := filepath.Join(root, "component")
	if err := os.MkdirAll(componentDir, 0o700); err != nil {
		t.Fatal(err)
	}
	marker, release := filepath.Join(root, "host-command-started"), filepath.Join(root, "release-host-command")
	fakeVrooli := "#!/bin/sh\nprintf started > \"$SE2_APPLY_MARKER\"\nwhile [ ! -f \"$SE2_APPLY_RELEASE\" ]; do sleep 0.1; done\nexit 1\n"
	if err := os.WriteFile(filepath.Join(binDir, "vrooli"), []byte(fakeVrooli), 0o700); err != nil {
		t.Fatal(err)
	}

	serverImage := filepath.Join(repo, "scenarios", "vrooli-onboarding", "api", "vrooli-onboarding-api")
	if info, err := os.Stat(serverImage); err != nil || info.Mode().Perm()&0o111 == 0 {
		t.Skip("run the supported scenario build before the Linux process-survival test")
	}
	componentImage := filepath.Join(componentDir, "vrooli-onboarding-api")
	if err := copyExecutable(serverImage, componentImage); err != nil {
		t.Fatalf("stage component image: %v", err)
	}
	t.Setenv("VROOLI_ROOT", repo)
	t.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
	t.Setenv("VROOLI_STATE_ROOT", stateRoot)
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	t.Setenv("BUNDLE_ROOT", "")
	t.Setenv("HOME", home)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("SE2_APPLY_MARKER", marker)
	t.Setenv("SE2_APPLY_RELEASE", release)
	previousRoots := operatorStateRoots
	t.Cleanup(func() { operatorStateRoots = previousRoots })
	if err := configureOperatorStateRoots(); err != nil {
		t.Fatalf("configure isolated roots: %v", err)
	}
	statePath, err := operatorStatePath()
	if err != nil {
		t.Fatalf("resolve isolated state path: %v", err)
	}
	runnerDir := filepath.Join(filepath.Dir(statePath), "apply-runs")
	if err := os.MkdirAll(runnerDir, 0o700); err != nil {
		t.Fatal(err)
	}
	runnerImage := filepath.Join(runnerDir, "vrooli-onboarding-api-runner")
	t.Setenv("SE2_APPLY_RUN_LOG", filepath.Join(runnerDir, "api-replacement-survival.log"))
	if err := copyExecutable(serverImage, runnerImage); err != nil {
		t.Fatalf("stage stable runner image: %v", err)
	}

	run := applyRun{
		ID:     "api-replacement-survival",
		Target: "local",
		Status: applyRunPending,
		Items:  []applyItemResult{{applyItem: applyItem{ID: "resource:fixture", Kind: "resource", Name: "fixture"}, Outcome: applyItemPending}},
	}
	if err := persistApplyRun(run); err != nil {
		t.Fatalf("persist pending run: %v", err)
	}

	parent := exec.Command(os.Args[0], "-test.run=^TestApplyRunnerProcessParentHelper$")
	parent.Env = append(os.Environ(), applyRunnerProcessHelper+"=1", "VROOLI_ONBOARDING_APPLY_RUN_ID="+run.ID, "VROOLI_ONBOARDING_APPLY_RUNNER_IMAGE="+runnerImage, "SE2_TEST_STATE_ROOT="+stateRoot, "SE2_TEST_REPO_ROOT="+repo)
	if output, err := parent.CombinedOutput(); err != nil {
		t.Fatalf("short-lived parent handoff failed: %v\n%s", err, output)
	}
	t.Cleanup(func() {
		_ = os.WriteFile(release, []byte("continue\n"), 0o600)
	})
	waitForApplyFile(t, marker, 10*time.Second)
	active, ok := applyRunSnapshot(run.ID)
	if !ok || active.Status != applyRunApplying || active.RunnerPID <= 0 {
		t.Fatalf("run after parent exit = %+v, want live applying runner", active)
	}
	pidGroup, err := syscall.Getpgid(active.RunnerPID)
	if err != nil {
		t.Fatalf("detached runner process %d is gone after parent exit: %v", active.RunnerPID, err)
	}
	if pidGroup == syscall.Getpgrp() {
		t.Fatalf("runner process group %d still belongs to the API parent's group", pidGroup)
	}
	if err := os.Remove(componentImage); err != nil {
		t.Fatalf("simulate atomic scenario component replacement: %v", err)
	}
	if err := os.WriteFile(release, []byte("continue\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		finished, ok := applyRunSnapshot(run.ID)
		if ok && finished.Status == applyRunPartiallyApplied && finished.Items[0].Outcome == applyItemFailed {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	final, _ := applyRunSnapshot(run.ID)
	encoded, _ := json.Marshal(final)
	t.Fatalf("detached run did not publish failure after component replacement: %s", strings.TrimSpace(string(encoded)))
}

func copyExecutable(source, destination string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return os.WriteFile(destination, data, 0o700)
}

func waitForApplyFile(t *testing.T, path string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	logPath := os.Getenv("SE2_APPLY_RUN_LOG")
	logData, _ := os.ReadFile(logPath)
	run, _ := applyRunSnapshot("api-replacement-survival")
	t.Fatalf("timed out waiting for safe fake host command marker %s; run=%+v runner-log=%s", path, run, strings.TrimSpace(string(logData)))
}
