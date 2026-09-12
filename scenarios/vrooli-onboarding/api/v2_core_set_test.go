// [REQ:ONB-CORE-SUPERVISION-AUTHORITY]
package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestV2CoreSetPreviewsTypedClosureAndPatchRejectsInvalidTrustedBase(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("VROOLI_STORAGE_ROOT", filepath.Join(root, "test-storage"))
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, operatorStateFixturePath(t, root), `{"version":"1.0.0","core":{"seed":["seed"],"trusted_base":["seed"]}}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "seed", ".vrooli", "service.json"), `{"service":{"name":"seed"},"dependencies":{"resources":{"redis":{"enabled":true,"required":false,"startup_policy":"try_start"}}}}`)

	oldPath := operatorStatePath
	t.Cleanup(func() { operatorStatePath = oldPath })
	operatorStatePath = func() (string, error) { return operatorStateFixturePath(t, root), nil }
	srv := NewServer()
	response := doRequest(t, srv, http.MethodPost, "/vrooli.vrooli_onboarding.v1.selection.SelectionService/GetCoreSet", `{"target":"local"}`)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"available":true`) || !strings.Contains(response.Body.String(), `"name":"redis"`) || !strings.Contains(response.Body.String(), `"source":"core.seed"`) {
		t.Fatalf("core-set response = %d: %s", response.Code, response.Body.String())
	}

	before, err := os.ReadFile(operatorStateFixturePath(t, root))
	if err != nil {
		t.Fatal(err)
	}
	response = doOperatorStatePatch(t, srv, `{"core":{"seed":["other"]}}`, "core")
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "not a core seed") {
		t.Fatalf("invalid core patch = %d: %s", response.Code, response.Body.String())
	}
	after, err := os.ReadFile(operatorStateFixturePath(t, root))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("rejected core patch changed operator state")
	}
}

func TestV2CoreSetKeepsSeedVisibleWhenClosureUnavailable(t *testing.T) {
	bundle := t.TempDir()
	t.Setenv("VROOLI_ROOT", "")
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	t.Setenv("BUNDLE_ROOT", bundle)
	statePath := filepath.Join(bundle, "app-data", "operator-state.json")
	writeFixtureFile(t, statePath, `{"version":"1.0.0","core":{"seed":["seed"],"trusted_base":["seed"]}}`)
	oldPath := operatorStatePath
	t.Cleanup(func() { operatorStatePath = oldPath })
	operatorStatePath = func() (string, error) { return statePath, nil }

	response := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.selection.SelectionService/GetCoreSet", `{"target":"local"}`)
	body := response.Body.String()
	if response.Code != http.StatusOK || strings.Contains(body, `"available":true`) || !strings.Contains(body, `"seed"`) || !strings.Contains(body, "closure unavailable") {
		t.Fatalf("fallback response = %d: %s", response.Code, response.Body.String())
	}
}
