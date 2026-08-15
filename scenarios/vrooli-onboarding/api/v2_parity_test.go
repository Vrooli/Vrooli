package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestAPIPatchPreservesSharedOperatorStateFields(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "operator-state.json"), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z","trust_posture":"shared","core":{"seed":["postgres"],"trusted_base":["git"]}}`)
	w := doRequest(t, NewServer(), http.MethodPatch, "/api/v2/operator-state", `{"scenarios":{"demo":{"enabled":true}}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("patch = %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, preserved := range []string{`"trust_posture":"shared"`, `"trusted_base":["git"]`, `"enabled":true`} {
		if !strings.Contains(body, preserved) {
			t.Fatalf("shared state lost %s: %s", preserved, body)
		}
	}
}

// [REQ:ONB-PARITY-IDENTICAL-STATE]
func TestThreeSurfaceSelectionProducesByteIdenticalOperatorState(t *testing.T) {
	selection := SelectionForParityTest()
	document, err := json.Marshal(selection)
	if err != nil {
		t.Fatal(err)
	}
	var roundTripped paritySelection
	if err := json.Unmarshal(document, &roundTripped); err != nil {
		t.Fatal(err)
	}

	uiPatch := map[string]any{
		"scenarios":       map[string]any{"writer": map[string]any{"enabled": true, "auto_restart": true}},
		"resources":       map[string]any{"ollama": map[string]any{"enabled": false}},
		"host_tools":      map[string]any{"git": map[string]any{"opted_in": true}},
		"host_safeguards": map[string]any{"firewall": map[string]any{"opted_in": false}},
	}
	states := []map[string]any{
		applyParityPatch(t, uiPatch),
		applyParityPatch(t, paritySelectionPatch(selection)),
		applyParityPatch(t, paritySelectionPatch(roundTripped)),
	}
	canonical := canonicalParityState(t, states[0])
	for index, state := range states[1:] {
		if got := canonicalParityState(t, state); got != canonical {
			t.Fatalf("surface %d produced a different operator-state document:\nwant %s\ngot  %s", index+2, canonical, got)
		}
	}
}

type paritySelection struct {
	ScenarioState  map[string]bool `json:"scenario_state,omitempty"`
	Resources      map[string]bool `json:"resources,omitempty"`
	HostTools      map[string]bool `json:"host_tools,omitempty"`
	HostSafeguards map[string]bool `json:"host_safeguards,omitempty"`
	OperatingMode  map[string]struct {
		AutoRestart bool `json:"auto_restart"`
	} `json:"operating_mode,omitempty"`
}

func SelectionForParityTest() paritySelection {
	return paritySelection{
		ScenarioState:  map[string]bool{"writer": true},
		Resources:      map[string]bool{"ollama": false},
		HostTools:      map[string]bool{"git": true},
		HostSafeguards: map[string]bool{"firewall": false},
		OperatingMode: map[string]struct {
			AutoRestart bool `json:"auto_restart"`
		}{"writer": {AutoRestart: true}},
	}
}

func paritySelectionPatch(selection paritySelection) map[string]any {
	patch := map[string]any{
		"scenarios":       map[string]any{},
		"resources":       map[string]any{},
		"host_tools":      map[string]any{},
		"host_safeguards": map[string]any{},
	}
	for name, enabled := range selection.ScenarioState {
		patch["scenarios"].(map[string]any)[name] = map[string]any{"enabled": enabled}
	}
	for name, enabled := range selection.Resources {
		patch["resources"].(map[string]any)[name] = map[string]any{"enabled": enabled}
	}
	for name, optedIn := range selection.HostTools {
		patch["host_tools"].(map[string]any)[name] = map[string]any{"opted_in": optedIn}
	}
	for name, optedIn := range selection.HostSafeguards {
		patch["host_safeguards"].(map[string]any)[name] = map[string]any{"opted_in": optedIn}
	}
	for name, mode := range selection.OperatingMode {
		patch["scenarios"].(map[string]any)[name] = map[string]any{"enabled": true, "auto_restart": mode.AutoRestart}
	}
	return patch
}

func applyParityPatch(t *testing.T, patch map[string]any) map[string]any {
	t.Helper()
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	body, err := json.Marshal(patch)
	if err != nil {
		t.Fatal(err)
	}
	w := doRequest(t, NewServer(), http.MethodPatch, "/api/v2/operator-state", string(body))
	if w.Code != http.StatusOK {
		t.Fatalf("parity patch = %d: %s", w.Code, w.Body.String())
	}
	var state map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	return state
}

func canonicalParityState(t *testing.T, state map[string]any) string {
	t.Helper()
	delete(state, "updated_at")
	body, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
