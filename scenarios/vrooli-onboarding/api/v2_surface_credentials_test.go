package main

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

func TestV2SurfaceAndCredentialsProjectCatalogMetadata(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "operator-state.json"), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z","scenarios":{"alpha":{"enabled":true}}}`)
	writeFixtureFile(t, filepath.Join(root, "scenarios", "alpha", ".vrooli", "service.json"), `{"service":{"name":"alpha","system_required":true},"dependencies":{"resources":{"demo":{"required":true}}},"credentials":{"descriptors":[{"logical_id":"vrooli/alpha","field":"token","label":"Alpha token","description":"Token for Alpha","obtain_url":"https://example.test/token","required":true}]}}`)
	writeFixtureFile(t, filepath.Join(root, "resources", "demo", "resource.json"), `{"name":"demo","credentials":{"descriptors":[{"logical_id":"vrooli/demo","field":"key","label":"Demo key","description":"Key for Demo","obtain_url":"https://example.test/key","required":false}]}}`)
	writeFixtureFile(t, filepath.Join(root, "internal", "tools", "git", "tool.json"), `{"name":"git","risk":"low","privilege":"user","config_schema":{"type":"object"}}`)
	writeFixtureFile(t, filepath.Join(root, "internal", "safeguards", "firewall", "safeguard.json"), `{"name":"firewall","risk":"high","privilege":"elevated","config_schema":{"type":"object"}}`)

	previous := credentialStatusCommand
	credentialStatusCommand = func(context.Context, string, string) ([]byte, error) { return []byte(`{"configured":true}`), nil }
	t.Cleanup(func() { credentialStatusCommand = previous })

	w := doGet(t, NewServer(), "/api/v2/surface")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "firewall") || !strings.Contains(w.Body.String(), `"schema"`) {
		t.Fatalf("surface = %d: %s", w.Code, w.Body.String())
	}
	w = doGet(t, NewServer(), "/api/v2/credentials")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Token for Alpha") || !strings.Contains(w.Body.String(), "https://example.test/key") || !strings.Contains(w.Body.String(), `"status":"configured"`) {
		t.Fatalf("credentials = %d: %s", w.Code, w.Body.String())
	}
}

func TestV2SessionRejectsInvalidStepAndCatalogSurfaceDegrades(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("BUNDLE_ROOT", "")
	writeFixtureFile(t, filepath.Join(root, ".vrooli", "operator-state.json"), `{"version":"1.0.0","updated_at":"2026-08-11T00:00:00Z"}`)
	w := doRequest(t, NewServer(), http.MethodPost, "/api/v2/session/step", `{"step":8}`)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "between 0 and 7") {
		t.Fatalf("invalid session = %d: %s", w.Code, w.Body.String())
	}
	w = doGet(t, NewServer(), "/api/v2/surface")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "catalog_unavailable") {
		t.Fatalf("missing surface catalog = %d: %s", w.Code, w.Body.String())
	}
}
