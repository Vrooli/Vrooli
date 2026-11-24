package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
)

func TestRecommendationUsesTypedSelectionRoute(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", filepath.Join(root, "test-storage"))
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, operatorStateFixturePath(t, root), `{"version":"1.0.0"}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{"service":{"name":"alpha","system_required":true},"dependencies":{"resources":{}}}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "vrooli-onboarding", "profiles", "general-purpose.json"), `{"schemaVersion":"1.0.0","id":"general-purpose","version":"1.0.0","default":true,"titleKey":"test.title","descriptionKey":"test.description","owner":"test","compatibleCatalogMajor":1,"questions":[],"rules":[{"id":"default","recommend":[{"scenarioRefs":["alpha"],"reasonKey":"test.reason"}]}],"manualSelection":{"available":true},"provenance":{"source":"test","reviewRevision":"r1"}}`)

	server := NewServer()
	typed := doRequest(t, server, http.MethodPost, "/vrooli.vrooli_onboarding.v1.selection.SelectionService/GetRecommendation", `{}`)
	if typed.Code != http.StatusOK {
		t.Fatalf("typed recommendation status = %d, body=%s", typed.Code, typed.Body.String())
	}
	var recommendation recommendationResponse
	if err := json.Unmarshal(typed.Body.Bytes(), &recommendation); err != nil {
		t.Fatalf("decode typed recommendation: %v", err)
	}
	if recommendation.Profile != "general-purpose" || len(recommendation.Scenarios) != 1 || recommendation.Scenarios[0] != "alpha" {
		t.Fatalf("typed recommendation = %+v, want the fixture profile and scenario", recommendation)
	}

	legacy := doRequest(t, server, http.MethodGet, "/api/v2/recommendation", "")
	if legacy.Code != http.StatusNotFound {
		t.Fatalf("unregistered compatibility route status = %d, want 404", legacy.Code)
	}
}
