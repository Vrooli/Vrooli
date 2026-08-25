package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type recordingApplyExecutor struct {
	mu     sync.Mutex
	calls  []string
	failOn string
}

func (e *recordingApplyExecutor) call(kind, name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.calls = append(e.calls, kind+":"+name)
	if e.failOn == kind+":"+name {
		return context.Canceled
	}
	return nil
}

func (e *recordingApplyExecutor) snapshotCalls() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.calls...)
}

func waitApplyTerminal(t *testing.T, id string) applyRun {
	t.Helper()
	// An apply run now computes the readiness verdict before it decides whether
	// the configuration marker may be written, and that verdict includes a
	// credential-authority diagnosis. The budget is generous because the point
	// of the wait is the terminal state, not the duration.
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		if run, ok := applyRunSnapshot(id); ok && run.Status != "pending" && run.Status != "applying" {
			return run
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("apply run %s did not reach a terminal state", id)
	return applyRun{}
}

func (e *recordingApplyExecutor) InstallTool(_ context.Context, name string) error {
	return e.call("tool", name)
}

func (e *recordingApplyExecutor) ApplySafeguard(_ context.Context, name string) error {
	return e.call("safeguard", name)
}

func (e *recordingApplyExecutor) EnableResource(_ context.Context, name string) error {
	return e.call("resource", name)
}

func (e *recordingApplyExecutor) StartScenario(_ context.Context, name string) error {
	return e.call("scenario", name)
}

func TestV2ApplyOrdersDependenciesAndIsIdempotent(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "operator-state.json"), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z","scenarios":{"alpha":{"enabled":true}}}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{"service":{"name":"alpha","system_required":true},"dependencies":{"resources":{"postgres":{"required":true}}}}`)
	writeFixtureFile(t, filepath.Join(root, "resources", "postgres", "resource.json"), `{"name":"postgres","hostTools":[],"hostSafeguards":[]}`)
	fake := &recordingApplyExecutor{}
	previous := onboardingApplyExecutor
	onboardingApplyExecutor = fake
	t.Cleanup(func() { onboardingApplyExecutor = previous })

	w := doRequest(t, NewServer(), http.MethodPost, "/api/v2/apply", "{}")
	if w.Code != http.StatusAccepted {
		t.Fatalf("apply status = %d: %s", w.Code, w.Body.String())
	}
	var first applyRun
	if err := json.Unmarshal(w.Body.Bytes(), &first); err != nil {
		t.Fatal(err)
	}
	waitApplyTerminal(t, first.ID)
	calls := fake.snapshotCalls()
	if len(calls) != 2 || calls[0] != "resource:postgres" || calls[1] != "scenario:alpha" {
		t.Fatalf("calls = %#v", calls)
	}

	w = doRequest(t, NewServer(), http.MethodPost, "/api/v2/apply", "{}")
	if w.Code != http.StatusAccepted {
		t.Fatalf("second apply status = %d: %s", w.Code, w.Body.String())
	}
	if len(fake.snapshotCalls()) != 2 {
		t.Fatalf("second apply issued mutating calls: %#v", fake.snapshotCalls())
	}
	var second applyRun
	if err := json.Unmarshal(w.Body.Bytes(), &second); err != nil {
		t.Fatal(err)
	}
	if second.Status != "already_satisfied" || !containsJSONString(w.Body.Bytes(), "already_satisfied") {
		t.Fatalf("second apply = %s", w.Body.String())
	}
	status := doGet(t, NewServer(), "/api/v2/apply/"+second.ID)
	if status.Code != http.StatusOK || !containsJSONString(status.Body.Bytes(), "already_satisfied") {
		t.Fatalf("apply status = %d: %s", status.Code, status.Body.String())
	}
	missing := doGet(t, NewServer(), "/api/v2/apply/missing")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing apply status = %d", missing.Code)
	}
}

func TestV2ApplySkipsDependentAfterFailure(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "operator-state.json"), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z","scenarios":{"alpha":{"enabled":true}}}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{"service":{"name":"alpha","system_required":true},"dependencies":{"resources":{"postgres":{"required":true}}}}`)
	writeFixtureFile(t, filepath.Join(root, "resources", "postgres", "resource.json"), `{"name":"postgres"}`)
	fake := &recordingApplyExecutor{failOn: "resource:postgres"}
	previous := onboardingApplyExecutor
	onboardingApplyExecutor = fake
	t.Cleanup(func() { onboardingApplyExecutor = previous })

	w := doRequest(t, NewServer(), http.MethodPost, "/api/v2/apply", "{}")
	if w.Code != http.StatusAccepted {
		t.Fatalf("apply = %d: %s", w.Code, w.Body.String())
	}
	var run applyRun
	if err := json.Unmarshal(w.Body.Bytes(), &run); err != nil {
		t.Fatal(err)
	}
	terminal := waitApplyTerminal(t, run.ID)
	if terminal.Status != "partially_applied" || !containsJSONString(mustJSON(t, terminal), "blocked") {
		t.Fatalf("apply = %#v", terminal)
	}
	if calls := fake.snapshotCalls(); len(calls) != 1 {
		t.Fatalf("dependent scenario was executed: %#v", calls)
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// TestApplyNeverRestartsItself pins the guard that keeps an apply run alive.
//
// vrooli-onboarding is a scenario like any other, so it appears in its own
// selection closure and therefore in its own apply plan. Executing that item
// means `vrooli scenario start vrooli-onboarding`, which stops the process
// running the apply: the run never reaches a terminal state, every item after
// it never executes, and the operator's wizard loses the API it is polling
// immediately after answering the consent prompt.
func TestApplyNeverRestartsItself(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "operator-state.json"),
		`{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z","scenarios":{"`+onboardingScenarioName+`":{"enabled":true},"alpha":{"enabled":true}}}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", onboardingScenarioName, ".vrooli", "service.json"),
		`{"service":{"name":"`+onboardingScenarioName+`","system_required":true}}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"),
		`{"service":{"name":"alpha","system_required":true}}`)

	fake := &recordingApplyExecutor{}
	previous := onboardingApplyExecutor
	onboardingApplyExecutor = fake
	t.Cleanup(func() { onboardingApplyExecutor = previous })

	w := doRequest(t, NewServer(), http.MethodPost, "/api/v2/apply", "{}")
	if w.Code != http.StatusAccepted {
		t.Fatalf("apply status = %d: %s", w.Code, w.Body.String())
	}
	var started applyRun
	if err := json.Unmarshal(w.Body.Bytes(), &started); err != nil {
		t.Fatal(err)
	}
	run := waitApplyTerminal(t, started.ID)

	for _, call := range fake.snapshotCalls() {
		if call == "scenario:"+onboardingScenarioName {
			t.Fatalf("apply restarted the scenario serving the run; calls = %#v", fake.snapshotCalls())
		}
	}

	// Skipping silently would be its own defect: the operator must be able to
	// see that the item was not executed, and why.
	var found bool
	for _, item := range run.Items {
		if item.Kind != "scenario" || item.Name != onboardingScenarioName {
			continue
		}
		found = true
		if item.Outcome != "skipped_self" {
			t.Fatalf("self item outcome = %q, want skipped_self", item.Outcome)
		}
		if !strings.Contains(item.Remediation, "already running") {
			t.Fatalf("a skipped item must explain itself, got %q", item.Remediation)
		}
	}
	if !found {
		t.Fatal("the plan did not contain the onboarding scenario; this test no longer covers the self-restart path")
	}

	// The skip must not be laundered into a failure. The run may still end
	// short of "applied" on its readiness verdict -- that is a separate,
	// legitimate outcome -- but it must not be reported as partially applied,
	// which is the status reserved for an item that actually failed.
	if run.Status == "partially_applied" {
		t.Fatalf("a skipped self-item was counted as a failure; status = %q", run.Status)
	}
	for _, item := range run.Items {
		if item.Outcome == "blocked" && item.BlockedBy == "scenario:"+onboardingScenarioName {
			t.Fatalf("item %s was blocked by the skipped self-item", item.Name)
		}
	}
}

// staleSelfExecutor is a recording executor that also answers the freshness
// probe, so a test can choose whether this scenario would be rebuilt.
type staleSelfExecutor struct {
	recordingApplyExecutor
	stale     bool
	probeErr  error
	probeCall int
}

func (e *staleSelfExecutor) ScenarioIsStale(_ context.Context, _ string) (bool, error) {
	e.probeCall++
	return e.stale, e.probeErr
}

// TestApplyRefusesWhenACascadeWouldRestartIt covers the failure the self-item
// skip does not: another scenario in the plan (vrooli-bridge, in the real
// catalogue) declares this one as a try_start dependency, so starting it
// cascades here. While this scenario is fresh that cascade is a no-op; while it
// is stale the cascade rebuilds and swaps its artifacts, stopping the process
// mid-apply. Refusing before touching the host beats dying halfway through.
func TestApplyRefusesWhenACascadeWouldRestartIt(t *testing.T) {
	setup := func(t *testing.T, executor applyExecutor) applyRun {
		t.Helper()
		root := t.TempDir()
		t.Setenv("VROOLI_ROOT", root)
		t.Setenv("BUNDLE_ROOT", "")
		writeFixtureFile(t, filepath.Join(root, ".vrooli", "operator-state.json"),
			`{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z","scenarios":{"`+onboardingScenarioName+`":{"enabled":true},"alpha":{"enabled":true}}}`)
		writeFixtureFile(t, filepath.Join(root, "scenarios", onboardingScenarioName, ".vrooli", "service.json"),
			`{"service":{"name":"`+onboardingScenarioName+`","system_required":true}}`)
		writeFixtureFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"),
			`{"service":{"name":"alpha","system_required":true}}`)

		previous := onboardingApplyExecutor
		onboardingApplyExecutor = executor
		t.Cleanup(func() { onboardingApplyExecutor = previous })

		w := doRequest(t, NewServer(), http.MethodPost, "/api/v2/apply", "{}")
		if w.Code != http.StatusAccepted {
			t.Fatalf("apply status = %d: %s", w.Code, w.Body.String())
		}
		var started applyRun
		if err := json.Unmarshal(w.Body.Bytes(), &started); err != nil {
			t.Fatal(err)
		}
		return waitApplyTerminal(t, started.ID)
	}

	t.Run("stale refuses without touching the host", func(t *testing.T) {
		executor := &staleSelfExecutor{stale: true}
		run := setup(t, executor)

		if run.Status != "blocked" {
			t.Fatalf("run status = %q, want blocked", run.Status)
		}
		if calls := executor.snapshotCalls(); len(calls) != 0 {
			t.Fatalf("a refused run must change nothing; calls = %#v", calls)
		}
		if len(run.Blockers) != 1 || run.Blockers[0].Name != onboardingScenarioName {
			t.Fatalf("blockers = %#v, want one naming %s", run.Blockers, onboardingScenarioName)
		}
		if !strings.Contains(run.Blockers[0].Remediation, "vrooli scenario start") {
			t.Fatalf("the operator must be told how to clear this, got %q", run.Blockers[0].Remediation)
		}
	})

	t.Run("fresh applies normally", func(t *testing.T) {
		executor := &staleSelfExecutor{stale: false}
		run := setup(t, executor)

		if run.Status == "blocked" {
			t.Fatal("a fresh scenario must not block its own apply; the cascade is a no-op")
		}
		if executor.probeCall == 0 {
			t.Fatal("the freshness probe never ran, so this test is not exercising the guard")
		}
		if len(executor.snapshotCalls()) == 0 {
			t.Fatal("a fresh run must still apply the plan")
		}
	})

	t.Run("an unusable probe does not block consented work", func(t *testing.T) {
		executor := &staleSelfExecutor{probeErr: errors.New("freshness unavailable")}
		run := setup(t, executor)

		if run.Status == "blocked" {
			t.Fatal("a probe failure is not evidence that a restart would happen; the run must proceed")
		}
	})
}
