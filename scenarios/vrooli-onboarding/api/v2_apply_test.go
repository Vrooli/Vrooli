package main

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
)

type recordingApplyExecutor struct {
	calls  []string
	failOn string
}

func (e *recordingApplyExecutor) call(kind, name string) error {
	e.calls = append(e.calls, kind+":"+name)
	if e.failOn == kind+":"+name {
		return context.Canceled
	}
	return nil
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
	if len(fake.calls) != 2 || fake.calls[0] != "resource:postgres" || fake.calls[1] != "scenario:alpha" {
		t.Fatalf("calls = %#v", fake.calls)
	}

	w = doRequest(t, NewServer(), http.MethodPost, "/api/v2/apply", "{}")
	if w.Code != http.StatusAccepted {
		t.Fatalf("second apply status = %d: %s", w.Code, w.Body.String())
	}
	if len(fake.calls) != 2 || !containsJSONString(w.Body.Bytes(), "already_satisfied") {
		t.Fatalf("second apply calls/body = %#v / %s", fake.calls, w.Body.String())
	}
	var second applyRun
	if err := json.Unmarshal(w.Body.Bytes(), &second); err != nil {
		t.Fatal(err)
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
	if w.Code != http.StatusAccepted || !containsJSONString(w.Body.Bytes(), "partially_applied") || !containsJSONString(w.Body.Bytes(), "blocked") {
		t.Fatalf("apply = %d: %s", w.Code, w.Body.String())
	}
	if len(fake.calls) != 1 {
		t.Fatalf("dependent scenario was executed: %#v", fake.calls)
	}
}
