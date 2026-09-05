package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestApplyRunnerRequestParsesBothFlagForms(t *testing.T) {
	for _, testCase := range []struct {
		name string
		args []string
		want string
		mode bool
	}{
		{name: "server start", args: []string{}, want: "", mode: false},
		{name: "server start with other flags", args: []string{"--verbose"}, want: "", mode: false},
		{name: "separate value", args: []string{"--apply-run", "apply-1"}, want: "apply-1", mode: true},
		{name: "equals value", args: []string{"--apply-run=apply-2"}, want: "apply-2", mode: true},
		{name: "flag with no value", args: []string{"--apply-run"}, want: "", mode: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			id, ok := applyRunnerRequest(testCase.args)
			if ok != testCase.mode {
				t.Fatalf("runner mode = %v, want %v", ok, testCase.mode)
			}
			if id != testCase.want {
				t.Fatalf("id = %q, want %q", id, testCase.want)
			}
		})
	}
}

func TestPrepareApplyRunnerExecutableCreatesStateOwnedCopy(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	t.Setenv("BUNDLE_ROOT", "")
	previous := applyRunnerExecutablePath
	t.Cleanup(func() { applyRunnerExecutablePath = previous })

	if err := prepareApplyRunnerExecutable(); err != nil {
		t.Fatalf("prepareApplyRunnerExecutable: %v", err)
	}
	if applyRunnerExecutablePath == "" {
		t.Fatal("apply runner executable path is empty")
	}
	info, err := os.Stat(applyRunnerExecutablePath)
	if err != nil {
		t.Fatalf("stat stable runner: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("stable runner mode = %o, want executable", info.Mode().Perm())
	}
}

// TestApplyRunnerModeExecutesAPersistedRun covers the handoff itself: a run
// written by one process is picked up and executed by another, with nothing
// passed between them but the run id.
func TestApplyRunnerModeExecutesAPersistedRun(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "operator-state.json"), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z","scenarios":{"alpha":{"enabled":true}}}`)
	stubExternalReadinessProbes(t)

	fake := &recordingApplyExecutor{}
	previous := onboardingApplyExecutor
	onboardingApplyExecutor = fake
	t.Cleanup(func() { onboardingApplyExecutor = previous })

	run := applyRun{
		ID:        "apply-handoff-fixture",
		Status:    "pending",
		StartedAt: operatorStateNow().UTC().Format(time.RFC3339),
		Items: []applyItemResult{
			{applyItem: applyItem{ID: "resource:postgres", Kind: "resource", Name: "postgres"}, Outcome: "pending"},
		},
	}
	if err := persistApplyRun(run); err != nil {
		t.Fatalf("persist run: %v", err)
	}

	if err := runApplyRunner(context.Background(), run.ID); err != nil {
		t.Fatalf("runApplyRunner: %v", err)
	}

	executed, ok := applyRunSnapshot(run.ID)
	if !ok {
		t.Fatal("the run disappeared")
	}
	if !isTerminalApplyStatus(executed.Status) {
		t.Fatalf("status = %q, want terminal", executed.Status)
	}
	if calls := fake.snapshotCalls(); len(calls) != 1 || calls[0] != "resource:postgres" {
		t.Fatalf("calls = %#v, want the persisted item to have been executed", calls)
	}
	if executed.RunnerPID != os.Getpid() {
		t.Fatalf("runner pid = %d, want the executing process %d recorded", executed.RunnerPID, os.Getpid())
	}
}

// TestApplyRunnerRefusesARunItDoesNotOwn keeps the handoff single-shot. Two
// runners on one run would repeat host mutations that already happened.
func TestApplyRunnerRefusesARunItDoesNotOwn(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "operator-state.json"), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z"}`)

	fake := &recordingApplyExecutor{}
	previous := onboardingApplyExecutor
	onboardingApplyExecutor = fake
	t.Cleanup(func() { onboardingApplyExecutor = previous })

	run := applyRun{ID: "apply-already-done", Status: "applied"}
	if err := persistApplyRun(run); err != nil {
		t.Fatalf("persist run: %v", err)
	}
	err := runApplyRunner(context.Background(), run.ID)
	if err == nil {
		t.Fatal("a runner claimed a run that was already terminal")
	}
	if !strings.Contains(err.Error(), "not pending") {
		t.Fatalf("error = %v, want it to name the reason", err)
	}
	if calls := fake.snapshotCalls(); len(calls) != 0 {
		t.Fatalf("a refused runner mutated the host: %#v", calls)
	}
}

// TestObservedApplyRunReportsAnAbandonedRun is the other half of surviving a
// restart. A run whose executor was killed leaves "applying" behind, and a
// client polling that status waits forever for a process that no longer exists.
func TestObservedApplyRunReportsAnAbandonedRun(t *testing.T) {
	fresh := operatorStateNow().UTC().Format(time.RFC3339)
	stale := operatorStateNow().UTC().Add(-2 * staleApplyHeartbeat).Format(time.RFC3339)

	previousAlive := processAlive
	t.Cleanup(func() { processAlive = previousAlive })

	t.Run("a working runner is left alone", func(t *testing.T) {
		processAlive = func(int) bool { return true }
		got := observedApplyRun(applyRun{Status: "applying", Heartbeat: fresh, RunnerPID: 4242})
		if got.Status != "applying" {
			t.Fatalf("status = %q, want applying", got.Status)
		}
	})

	t.Run("a slow runner that is still alive is left alone", func(t *testing.T) {
		// Both signals are required. A heartbeat can lag on a slow filesystem,
		// and burying a live run would lose work that is still in progress.
		processAlive = func(int) bool { return true }
		got := observedApplyRun(applyRun{Status: "applying", Heartbeat: stale, RunnerPID: 4242})
		if got.Status != "applying" {
			t.Fatalf("status = %q, want applying for a stale heartbeat with a live process", got.Status)
		}
	})

	t.Run("a run accepted moments ago is not yet judged", func(t *testing.T) {
		processAlive = func(int) bool { return false }
		got := observedApplyRun(applyRun{Status: "pending"})
		if got.Status != "pending" {
			t.Fatalf("status = %q, want pending; the runner is still starting", got.Status)
		}
	})

	t.Run("a dead runner is reported as interrupted", func(t *testing.T) {
		processAlive = func(int) bool { return false }
		got := observedApplyRun(applyRun{Status: "applying", Heartbeat: stale, RunnerPID: 4242})
		if got.Status != "interrupted" {
			t.Fatalf("status = %q, want interrupted", got.Status)
		}
		if got.Error == "" {
			t.Fatal("an interrupted run must say what happened")
		}
		if len(got.Blockers) != 1 || !strings.Contains(got.Blockers[0].Remediation, "apply the selection again") {
			t.Fatalf("blockers = %#v, want one telling the operator how to continue", got.Blockers)
		}
	})

	t.Run("a terminal run is never second-guessed", func(t *testing.T) {
		processAlive = func(int) bool { return false }
		got := observedApplyRun(applyRun{Status: "applied", Heartbeat: stale, RunnerPID: 4242})
		if got.Status != "applied" {
			t.Fatalf("status = %q, want applied", got.Status)
		}
	})
}
