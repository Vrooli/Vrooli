package main

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/operatorstate"
)

func TestV2RecommendationAcceptIsIdempotent(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", filepath.Join(root, "test-storage"))
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, operatorStateFixturePath(t, root), `{"version":"1.0.0"}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{"service":{"name":"alpha","system_required":true},"dependencies":{"resources":{"postgres":{"required":true}}}}`)
	writeFixtureFile(t, filepath.Join(root, "resources", "postgres", "resource.json"), `{"name":"postgres"}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "vrooli-onboarding", "profiles", "general-purpose.json"), `{"schemaVersion":"1.0.0","id":"general-purpose","version":"1.0.0","default":true,"titleKey":"test.title","descriptionKey":"test.description","owner":"test","compatibleCatalogMajor":1,"questions":[],"rules":[{"id":"default","recommend":[{"scenarioRefs":["alpha"],"reasonKey":"test.reason"}]}],"manualSelection":{"available":true},"provenance":{"source":"test","reviewRevision":"r1"}}`)

	first := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.selection.SelectionService/AcceptRecommendation", `{"target":"local"}`)
	second := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.selection.SelectionService/AcceptRecommendation", `{"target":"local"}`)
	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("accept statuses = %d and %d", first.Code, second.Code)
	}
	var firstBody, secondBody map[string]any
	if err := json.Unmarshal(first.Body.Bytes(), &firstBody); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(second.Body.Bytes(), &secondBody); err != nil {
		t.Fatal(err)
	}
	if !jsonEqual(firstBody, secondBody) {
		t.Fatalf("idempotent responses differ: %#v vs %#v", firstBody, secondBody)
	}
	state, err := loadOperatorStateFor(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if state.Scenarios["alpha"].Enabled == nil || !*state.Scenarios["alpha"].Enabled {
		t.Fatalf("accepted recommendation did not enable alpha: %#v", state.Scenarios)
	}
}

func TestSessionPositionPrefersStableStepIDOverLegacyIndex(t *testing.T) {
	state := OperatorState{Session: &operatorstate.Session{Step: 8, StepID: "welcome"}}
	ordinal, id := sessionPosition(state)
	if ordinal != 0 || id != "welcome" {
		t.Fatalf("session position = %d/%q", ordinal, id)
	}
}

func jsonEqual(left, right map[string]any) bool {
	a, _ := json.Marshal(left)
	b, _ := json.Marshal(right)
	return string(a) == string(b)
}
