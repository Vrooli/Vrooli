package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestAPIPatchPreservesSharedOperatorStateFields(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", filepath.Join(root, "test-storage"))
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, operatorStateFixturePath(t, root), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z","trust_posture":"shared","core":{"seed":["git","postgres"],"trusted_base":["git"]}}`)
	w := doOperatorStatePatch(t, NewServer(), `{"scenarios":{"demo":{"enabled":true}}}`, "scenarios.demo.enabled")
	if w.Code != http.StatusOK {
		t.Fatalf("patch = %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, preserved := range []string{`"trustPosture":"shared"`, `"trustedBase":["git"]`, `"enabled":true`} {
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
	t.Setenv("VROOLI_STORAGE_ROOT", filepath.Join(root, "test-storage"))
	t.Setenv("BUNDLE_ROOT", "")
	writeHostSafeguardCatalog(t, root)
	body, err := json.Marshal(patch)
	if err != nil {
		t.Fatal(err)
	}
	var patchValue map[string]any
	if err := json.Unmarshal(body, &patchValue); err != nil {
		t.Fatal(err)
	}
	srv := NewServer()
	safeguards, _ := patchValue["host_safeguards"].(map[string]any)
	hostTools, _ := patchValue["host_tools"].(map[string]any)
	delete(patchValue, "host_safeguards")
	delete(patchValue, "host_tools")
	body, err = json.Marshal(patchValue)
	if err != nil {
		t.Fatal(err)
	}
	paths := make([]string, 0)
	for top, rawChoices := range patchValue {
		_, ok := rawChoices.(map[string]any)
		if !ok {
			paths = append(paths, top)
			continue
		}
		paths = append(paths, strings.ReplaceAll(top, "host_tools", "hostTools"))
	}
	var w *httptest.ResponseRecorder
	if len(patchValue) > 0 {
		w = doOperatorStatePatch(t, srv, string(body), paths...)
		if w.Code != http.StatusOK {
			t.Fatalf("parity patch = %d: %s", w.Code, w.Body.String())
		}
	}
	for name, rawChoice := range safeguards {
		choice, _ := rawChoice.(map[string]any)
		if err := writeSafeguardPermission(t, srv, name, choice); err != nil {
			t.Fatalf("write safeguard permission %s: %v", name, err)
		}
	}
	for name, rawChoice := range hostTools {
		choice, _ := rawChoice.(map[string]any)
		if err := writeHostToolPermission(t, srv, name, choice); err != nil {
			t.Fatalf("write host tool permission %s: %v", name, err)
		}
	}
	if len(safeguards) > 0 || len(hostTools) > 0 {
		w = doPost(t, srv, "/vrooli.vrooli_onboarding.v1.operatorstate.OperatorStateService/GetOperatorState", `{}`)
		if w.Code != http.StatusOK {
			t.Fatalf("read operator state after safeguard permission = %d: %s", w.Code, w.Body.String())
		}
	}
	if w == nil {
		t.Fatal("parity patch did not update state")
	}
	var state map[string]any
	state = operatorStateFromResponse(t, w.Body.Bytes())
	return state
}

func writeHostSafeguardCatalog(t *testing.T, root string) {
	t.Helper()
	writeFixtureFile(t, filepath.Join(root, "scenarios", "writer", ".vrooli", "service.json"), `{"service":{"name":"writer"},"dependencies":{"resources":{}},"hostTools":[{"name":"git","reason":"test fixture"}],"hostSafeguards":[{"name":"firewall","reason":"test fixture"}]}`)
	writeFixtureFile(t, filepath.Join(root, "internal", "tools", "git", "tool.json"), `{"name":"git","privilege":"none"}`)
	writeFixtureFile(t, filepath.Join(root, "internal", "safeguards", "firewall", "safeguard.json"), `{"name":"firewall","config":{"type":"object","properties":{"enabled":{"type":"boolean"}}}}`)
}

func writeHostToolPermission(t *testing.T, srv *Server, name string, choice map[string]any) error {
	t.Helper()
	permissionID := "onboarding/host-tool/selected-closure/" + name
	readBody, _ := json.Marshal(map[string]string{"permissionId": permissionID, "owner": "vrooli-onboarding"})
	readResponse := doRequest(t, srv, http.MethodPost, "/vrooli.vrooli_onboarding.v1.permissions.PermissionRegistryService/ReadPermission", string(readBody))
	if readResponse.Code != http.StatusOK {
		return fmt.Errorf("read permission returned %d: %s", readResponse.Code, readResponse.Body.String())
	}
	var read struct {
		Revision string `json:"revision"`
	}
	if err := json.Unmarshal(readResponse.Body.Bytes(), &read); err != nil {
		return err
	}
	value, _ := json.Marshal(choice)
	writeBody, _ := json.Marshal(map[string]any{"permissionId": permissionID, "owner": "vrooli-onboarding", "expectedRevision": read.Revision, "idempotencyKey": "parity-host-tool:" + name, "valueJson": base64.StdEncoding.EncodeToString(value)})
	writeResponse := doRequest(t, srv, http.MethodPost, "/vrooli.vrooli_onboarding.v1.permissions.PermissionRegistryService/WritePermission", string(writeBody))
	if writeResponse.Code != http.StatusOK {
		return fmt.Errorf("write permission returned %d: %s", writeResponse.Code, writeResponse.Body.String())
	}
	return nil
}

func writeSafeguardPermission(t *testing.T, srv *Server, name string, choice map[string]any) error {
	t.Helper()
	permissionID := "onboarding/host-safeguard/" + name
	readBody, _ := json.Marshal(map[string]string{"permissionId": permissionID, "owner": "vrooli-onboarding"})
	readResponse := doRequest(t, srv, http.MethodPost, "/vrooli.vrooli_onboarding.v1.permissions.PermissionRegistryService/ReadPermission", string(readBody))
	if readResponse.Code != http.StatusOK {
		return fmt.Errorf("read permission returned %d: %s", readResponse.Code, readResponse.Body.String())
	}
	var read struct {
		Revision string `json:"revision"`
	}
	if err := json.Unmarshal(readResponse.Body.Bytes(), &read); err != nil {
		return err
	}
	value, _ := json.Marshal(choice)
	writeBody, _ := json.Marshal(map[string]any{
		"permissionId": permissionID, "owner": "vrooli-onboarding", "expectedRevision": read.Revision,
		"idempotencyKey": "parity-safeguard:" + name, "valueJson": base64.StdEncoding.EncodeToString(value),
	})
	writeResponse := doRequest(t, srv, http.MethodPost, "/vrooli.vrooli_onboarding.v1.permissions.PermissionRegistryService/WritePermission", string(writeBody))
	if writeResponse.Code != http.StatusOK {
		return fmt.Errorf("write permission returned %d: %s", writeResponse.Code, writeResponse.Body.String())
	}
	return nil
}

func canonicalParityState(t *testing.T, state map[string]any) string {
	t.Helper()
	delete(state, "updated_at")
	delete(state, "updatedAt")
	body, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
