package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
)

func TestV2ClosureFollowsScenarioAndResourceDependenciesWithProvenance(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "operator-state.json"), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z","scenarios":{"alpha":{"enabled":true}}}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{"service":{"name":"alpha"},"dependencies":{"scenarios":{"beta":{"required":true}},"resources":{"redis":{"required":true}}}}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "beta", ".vrooli", "service.json"), `{"service":{"name":"beta"},"dependencies":{"resources":{"qdrant":{"startup_policy":"try_start"}}}}`)

	w := doGet(t, NewServer(), "/api/v2/closure")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	var body closureResult
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Scenarios) != 2 || len(body.Resources) != 2 {
		t.Fatalf("closure = %#v", body)
	}
	if body.Scenarios[1].Name != "beta" || body.Scenarios[1].Provenance[0].Kind != "required" || body.Scenarios[1].Provenance[0].From != "alpha" {
		t.Fatalf("beta provenance = %#v", body.Scenarios[1])
	}

	w = doGet(t, NewServer(), "/api/v2/union")
	if w.Code != http.StatusOK || !containsJSONString(w.Body.Bytes(), "beta") || !containsJSONString(w.Body.Bytes(), "qdrant") {
		t.Fatalf("union = %d: %s", w.Code, w.Body.String())
	}
}

func TestV2ClosureRejectsDependencyCycles(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "operator-state.json"), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z","scenarios":{"alpha":{"enabled":true}}}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{"service":{"name":"alpha"},"dependencies":{"scenarios":{"beta":{"required":true}}}}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "beta", ".vrooli", "service.json"), `{"service":{"name":"beta"},"dependencies":{"scenarios":{"alpha":{"required":true}}}}`)

	w := doGet(t, NewServer(), "/api/v2/closure")
	if w.Code != http.StatusUnprocessableEntity || !containsJSONString(w.Body.Bytes(), "cycle") {
		t.Fatalf("cycle response = %d: %s", w.Code, w.Body.String())
	}
}

func containsJSONString(data []byte, want string) bool {
	var value any
	if json.Unmarshal(data, &value) != nil {
		return false
	}
	encoded, _ := json.Marshal(value)
	return bytes.Contains(encoded, []byte(want))
}
