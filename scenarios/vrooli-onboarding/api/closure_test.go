package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestV2ClosureFollowsScenarioAndResourceDependenciesWithProvenance(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", filepath.Join(root, "test-storage"))
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, operatorStateFixturePath(t, root), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z","scenarios":{"alpha":{"enabled":true}}}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{"service":{"name":"alpha"},"dependencies":{"scenarios":{"beta":{"required":true}},"resources":{"redis":{"required":true}}}}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "beta", ".vrooli", "service.json"), `{"service":{"name":"beta"},"dependencies":{"resources":{"qdrant":{"startup_policy":"try_start"}}}}`)

	w := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.selection.SelectionService/GetClosure", `{"target":"local"}`)
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

	w = doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.selection.SelectionService/GetUnion", `{"target":"local"}`)
	if w.Code != http.StatusOK || !containsJSONString(w.Body.Bytes(), "beta") || !containsJSONString(w.Body.Bytes(), "qdrant") {
		t.Fatalf("union = %d: %s", w.Code, w.Body.String())
	}
}

func TestV2ClosureRejectsDependencyCycles(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", filepath.Join(root, "test-storage"))
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, operatorStateFixturePath(t, root), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z","scenarios":{"alpha":{"enabled":true}}}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{"service":{"name":"alpha"},"dependencies":{"scenarios":{"beta":{"required":true}}}}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "beta", ".vrooli", "service.json"), `{"service":{"name":"beta"},"dependencies":{"scenarios":{"alpha":{"required":true}}}}`)

	w := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.selection.SelectionService/GetClosure", `{"target":"local"}`)
	if w.Code == http.StatusOK || !containsJSONString(w.Body.Bytes(), "cycle") {
		t.Fatalf("cycle response = %d: %s", w.Code, w.Body.String())
	}
}

func TestClosurePreservesIgnoredAndOptionalDependencySemantics(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, filepath.Join(root, "scenarios", "writer", ".vrooli", "service.json"), `{
  "service":{"name":"writer"},
  "dependencies":{"resources":{"required-db":{"required":true,"startup_policy":"must_start"},"ignored-cache":{"required":true,"startup_policy":"ignore"}},"scenarios":{"helper":{"required":false,"startup_policy":"try_start"}}}
}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "helper", ".vrooli", "service.json"), `{"service":{"name":"helper"}}`)
	writeFixtureFile(t, filepath.Join(root, "resources", "required-db", "resource.json"), `{"name":"required-db"}`)
	writeFixtureFile(t, filepath.Join(root, "resources", "ignored-cache", "resource.json"), `{"name":"ignored-cache"}`)
	writeFixtureFile(t, filepath.Join(root, "resources", "optional-cache", "resource.json"), `{"name":"optional-cache"}`)
	enabled := true
	state := OperatorState{Resources: map[string]EnabledChoice{"optional-cache": {Enabled: &enabled}}}
	result, err := resolveClosureForState(root, []ScenarioReadModel{{Name: "writer", Enabled: true}}, state)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]closureMember{}
	for _, item := range result.Resources {
		byName[item.Name] = item
	}
	if got := byName["required-db"]; got.State != "included" || got.Policy != "must_start" || !got.Required {
		t.Fatalf("required resource = %#v", got)
	}
	if got := byName["ignored-cache"]; got.State != "ignored" || got.Policy != "ignore" {
		t.Fatalf("ignored resource = %#v", got)
	}
	if got := byName["optional-cache"]; got.State != "selected" || !got.Direct {
		t.Fatalf("standalone selected resource = %#v", got)
	}
	if len(result.Scenarios) != 2 {
		t.Fatalf("scenario closure = %#v", result.Scenarios)
	}
	byScenario := map[string]closureMember{}
	for _, item := range result.Scenarios {
		byScenario[item.Name] = item
	}
	if byScenario["helper"].State != "included" || byScenario["writer"].State != "selected" {
		t.Fatalf("scenario states = %#v", byScenario)
	}
}

func TestClosureRejectsDisabledRequiredDependency(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, filepath.Join(root, "scenarios", "writer", ".vrooli", "service.json"), `{"service":{"name":"writer"},"dependencies":{"resources":{"db":{"required":true,"enabled":false}}}}`)
	_, err := resolveClosureForState(root, []ScenarioReadModel{{Name: "writer", Enabled: true}}, OperatorState{})
	if err == nil || !strings.Contains(err.Error(), `required resource "db"`) {
		t.Fatalf("error = %v, want disabled required dependency", err)
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
