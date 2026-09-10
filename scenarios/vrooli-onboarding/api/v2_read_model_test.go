package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestV2ScenariosDerivesResourcesAndEffectiveStateFromManifests(t *testing.T) {
	root := newV2Root(t)
	path := filepath.Join(root, "scenarios", "alpha", ".vrooli")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "service.json"), []byte(`{"service":{"name":"alpha","description":"Alpha","system_required":true},"runtime":{"auto_restart_default":false},"dependencies":{"resources":{"redis":{"enabled":true},"postgres":{"enabled":true}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "operator-state.json"), []byte(`{"version":"1.0.0","updated_at":"2026-01-01T00:00:00Z","scenarios":{"alpha":{"enabled":false,"auto_restart":true}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	w := doRequest(t, NewServer(), http.MethodPost, "/vrooli.vrooli_onboarding.v1.selection.SelectionService/ListScenarios", `{"target":"local"}`)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"systemRequired":true`) || !strings.Contains(w.Body.String(), `"enabled":true`) || !strings.Contains(w.Body.String(), `"postgres"`) {
		t.Fatalf("response = %d: %s", w.Code, w.Body.String())
	}
}

func TestManifestRootUsesBundledCatalogWhenRepositoryIsUnavailable(t *testing.T) {
	bundle := t.TempDir()
	catalog := filepath.Join(bundle, "catalog")
	if err := os.MkdirAll(filepath.Join(catalog, "scenarios"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_ROOT", "")
	t.Setenv("BUNDLE_ROOT", bundle)
	root, err := manifestRoot()
	if err != nil {
		t.Fatalf("manifestRoot: %v", err)
	}
	if root != catalog {
		t.Fatalf("manifest root = %q, want %q", root, catalog)
	}
}
