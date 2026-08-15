package main

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
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
	deadline := time.Now().Add(2 * time.Second)
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
