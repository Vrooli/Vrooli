package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestV2HandoffProjectsEffectiveSelectionWithoutOperatorStateInternals(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{"service":{"name":"alpha","system_required":true},"runtime":{"auto_restart_default":false},"dependencies":{"resources":{}}}`)
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "operator-state.json"), `{"version":"1.0.0","updated_at":"2026-08-12T00:00:00Z","scenarios":{"alpha":{"enabled":false,"auto_restart":true}},"completion":{"selection_digest":"private"}}`)

	w := doRequest(t, NewServer(), http.MethodPost, "/api/v2/handoff", `{"machine_id":"machine-1","node_id":"node-1","node_kind":"agent"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("handoff = %d: %s", w.Code, w.Body.String())
	}
	var got onboardingHandoffSelection
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !got.Apply || len(got.Scenarios) != 1 || got.Scenarios[0] != "alpha" {
		t.Fatalf("selection = %+v", got)
	}
	if got.OperatingMode["alpha"].AutoRestart != true {
		t.Fatalf("operating mode = %+v", got.OperatingMode)
	}
	if len(got.OptionalResources) != 0 || len(got.Host.Tools) != 0 || len(got.Host.Safeguards) != 0 {
		t.Fatalf("unexpected optional capabilities = %+v", got)
	}
	for _, forbidden := range []string{"selection_digest", "private", "operator-state"} {
		if strings.Contains(w.Body.String(), forbidden) {
			t.Fatalf("handoff leaked operator-state internals: %s", w.Body.String())
		}
	}
}

func TestV2HandoffRejectsInvalidIdentityAndUnknownFields(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{"service":{"name":"alpha","system_required":true},"dependencies":{"resources":{}}}`)

	for _, body := range []string{`{"machine_id":"machine-1"}`, `{"node_id":"node-1","unexpected":true}`} {
		w := doRequest(t, NewServer(), http.MethodPost, "/api/v2/handoff", body)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("body %s: status = %d, want %d: %s", body, w.Code, http.StatusBadRequest, w.Body.String())
		}
	}
}

func TestV2HandoffRejectsTrailingJSON(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{"service":{"name":"alpha","system_required":true},"dependencies":{"resources":{}}}`)
	w := doRequest(t, NewServer(), http.MethodPost, "/api/v2/handoff", `{"node_id":"node-1"}{"node_id":"node-2"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}
