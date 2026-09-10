package main

import (
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/selection"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestV2HandoffProjectsEffectiveSelectionWithoutOperatorStateInternals(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{"service":{"name":"alpha","system_required":true},"runtime":{"auto_restart_default":false},"dependencies":{"resources":{}}}`)
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "operator-state.json"), `{"version":"1.0.0","updated_at":"2026-08-12T00:00:00Z","scenarios":{"alpha":{"enabled":false,"auto_restart":true}},"completion":{"selection_digest":"private"}}`)

	w := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.selection.SelectionService/CreateHandoff", `{"target":"local","machineId":"machine-1","nodeId":"node-1","nodeKind":"agent"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("handoff = %d: %s", w.Code, w.Body.String())
	}
	var response selectionv1.CreateHandoffResponse
	if err := protojson.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	got := response.GetSelection()
	if got == nil || !got.Apply || len(got.Scenarios) != 1 || got.Scenarios[0] != "alpha" {
		t.Fatalf("selection = %+v", got)
	}
	if got.OperatingMode["alpha"] != "auto-restart" {
		t.Fatalf("operating mode = %+v", got.OperatingMode)
	}
	if len(got.OptionalResources) != 0 || len(got.HostTools) != 0 || len(got.HostSafeguards) != 0 {
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
		w := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.selection.SelectionService/CreateHandoff", `{"target":"local",`+strings.TrimPrefix(strings.TrimSuffix(body, "}"), "{")+`}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("body %s: status = %d, want %d: %s", body, w.Code, http.StatusBadRequest, w.Body.String())
		}
	}
}

func TestV2HandoffRejectsUnknownSelectionSchemaVersion(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{"service":{"name":"alpha","system_required":true},"dependencies":{"resources":{}}}`)
	w := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.selection.SelectionService/CreateHandoff", `{"target":"local","nodeId":"node-1","desiredSelection":{"schemaVersion":"v99","scenarios":["alpha"]}}`)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "unsupported setup selection schema version") {
		t.Fatalf("status = %d, want typed version refusal: %s", w.Code, w.Body.String())
	}
}

func TestV2HandoffUsesMachineDesiredSelection(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	w := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.selection.SelectionService/CreateHandoff", `{"target":"local","machineId":"machine-1","nodeId":"node-1","nodeKind":"agent","desiredSelection":{"scenarios":["machine-scenario"],"optionalResources":["machine-resource"],"hostTools":["machine-tool"],"apply":false}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("handoff = %d: %s", w.Code, w.Body.String())
	}
	var response selectionv1.CreateHandoffResponse
	if err := protojson.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	got := response.GetSelection()
	if !got.Apply || len(got.Scenarios) != 1 || got.Scenarios[0] != "machine-scenario" || len(got.OptionalResources) != 1 || got.OptionalResources[0] != "machine-resource" || len(got.HostTools) != 1 {
		t.Fatalf("desired selection was not returned: %+v", got)
	}
}

func TestV2HandoffRejectsTrailingJSON(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{"service":{"name":"alpha","system_required":true},"dependencies":{"resources":{}}}`)
	w := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.selection.SelectionService/CreateHandoff", `{"target":"local","nodeId":"node-1"}{"target":"local","nodeId":"node-2"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}
