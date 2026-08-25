package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeBundleFixture(t *testing.T, includeTools bool) (string, string) {
	t.Helper()
	bundle := t.TempDir()
	storageRoot := t.TempDir()
	catalog := filepath.Join(bundle, "catalog")
	for _, dir := range []string{
		filepath.Join(catalog, "scenarios", "alpha", ".vrooli"),
		filepath.Join(catalog, "resources", "demo"),
		filepath.Join(catalog, "internal", "safeguards"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	service := `{"service":{"name":"alpha","description":"Alpha","system_required":true},"dependencies":{"resources":{"demo":{}}},"hostTools":[{"name":"tmux","required":true,"reason":"session support"}]}`
	resource := `{"name":"demo","display_name":"Demo","description":"Demo resource","category":"general"}`
	writeFixtureFile(t, filepath.Join(catalog, "scenarios", "alpha", ".vrooli", "service.json"), service)
	writeFixtureFile(t, filepath.Join(catalog, "resources", "demo", "resource.json"), resource)
	if includeTools {
		toolDir := filepath.Join(catalog, "internal", "tools", "tmux")
		if err := os.MkdirAll(toolDir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeFixtureFile(t, filepath.Join(toolDir, "tool.json"), `{"name":"tmux","description":"terminal multiplexer","commands":["tmux"]}`)
	}
	stateDir := filepath.Join(storageRoot, ".vrooli")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFixtureFile(t, filepath.Join(storageRoot, "operator-state.json"), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z","scenarios":{"alpha":{"enabled":true}}}`)
	return bundle, storageRoot
}

func writeFixtureFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// [REQ:ONB-TIER-BUNDLE-COMPLETENESS]
func TestV2BundleModeServesAllCatalogReadModels(t *testing.T) {
	bundle, storageRoot := writeBundleFixture(t, true)
	t.Setenv("VROOLI_ROOT", "")
	t.Setenv("BUNDLE_ROOT", bundle)
	t.Setenv("VROOLI_STORAGE_ROOT", storageRoot)

	for _, endpoint := range []string{"/api/v2/scenarios", "/api/v2/resources", "/api/v2/host-requirements", "/api/v2/readiness", "/api/v2/closure", "/api/v2/union", "/api/v2/credentials", "/api/v2/session"} {
		w := doGet(t, NewServer(), endpoint)
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d: %s", endpoint, w.Code, w.Body.String())
		}
	}
	step := doRequest(t, NewServer(), http.MethodPost, "/api/v2/session/step", `{"step":2}`)
	if step.Code != http.StatusOK || !strings.Contains(step.Body.String(), `"step":2`) {
		t.Fatalf("session step = %d: %s", step.Code, step.Body.String())
	}
	w := doGet(t, NewServer(), "/api/v2/resources")
	var body struct {
		Resources []resourceReadModel `json:"resources"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Resources) != 1 || body.Resources[0].Name != "demo" {
		t.Fatalf("resource read model = %#v", body.Resources)
	}
}

func TestV2BundleModeUsesBundleLocalAppDataWithOnlyBundleRoot(t *testing.T) {
	bundle, _ := writeBundleFixture(t, true)
	t.Setenv("VROOLI_ROOT", "")
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	t.Setenv("BUNDLE_ROOT", bundle)

	for _, endpoint := range []string{"/api/v2/scenarios", "/api/v2/resources", "/api/v2/host-requirements", "/api/v2/readiness", "/api/v2/closure", "/api/v2/union", "/api/v2/credentials", "/api/v2/session"} {
		w := doGet(t, NewServer(), endpoint)
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d: %s", endpoint, w.Code, w.Body.String())
		}
	}
}

func TestV2BundleModeNamesMissingCatalogAsDegraded(t *testing.T) {
	bundle, storageRoot := writeBundleFixture(t, false)
	t.Setenv("VROOLI_ROOT", "")
	t.Setenv("BUNDLE_ROOT", bundle)
	t.Setenv("VROOLI_STORAGE_ROOT", storageRoot)

	w := doGet(t, NewServer(), "/api/v2/host-requirements")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "degraded" {
		t.Fatalf("body = %s", w.Body.String())
	}
	errBody, ok := body["error"].(map[string]any)
	if !ok || errBody["code"] != "catalog_unavailable" || errBody["missing_catalog"] != "catalog/internal/tools" {
		t.Fatalf("degraded error = %v", body["error"])
	}
}
