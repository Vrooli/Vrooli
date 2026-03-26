package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"brand-manager/domain"
)

// [REQ:BM-REQ-DISC-SCAN] [REQ:BM-REQ-DISC-IMPORT] [REQ:BM-REQ-DISC-LPBS] [REQ:BM-REQ-APPLY-ASSETS]

func TestDiscoverScenario_ServiceJSON(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)

	// Create .vrooli/service.json
	vrooliDir := filepath.Join(scenarioDir, ".vrooli")
	os.MkdirAll(vrooliDir, 0o755)
	os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte(`{
		"name": "My Scenario",
		"description": "A test scenario",
		"tags": ["test"]
	}`), 0o644)

	_, r, _, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfigWithScenarioParent(t, parentDir))

	// [REQ:BM-REQ-DISC-SCAN]
	req := httptest.NewRequest("GET", "/api/v1/discover/"+scenarioName, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	sources, _ := result["sources"].([]interface{})
	if len(sources) == 0 {
		t.Fatal("expected at least one source")
	}

	first := sources[0].(map[string]interface{})
	if first["type"] != "service_json" {
		t.Errorf("type = %v, want service_json", first["type"])
	}

	draft, _ := result["draft_brand"].(map[string]interface{})
	if draft == nil {
		t.Fatal("expected draft_brand")
	}
	identity, _ := draft["identity"].(map[string]interface{})
	if identity == nil || identity["display_name"] != "My Scenario" {
		t.Errorf("draft display_name = %v, want My Scenario", identity)
	}
}

func TestDiscoverScenario_BrandingJSON_LPBS(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)

	// Create .vrooli/branding.json in LPBS format [REQ:BM-REQ-DISC-LPBS]
	vrooliDir := filepath.Join(scenarioDir, ".vrooli")
	os.MkdirAll(vrooliDir, 0o755)
	os.WriteFile(filepath.Join(vrooliDir, "branding.json"), []byte(`{
		"site_name": "My Site",
		"tagline": "Build better",
		"theme": {
			"primary": "#3498db",
			"secondary": "#2ecc71",
			"background": "#ffffff"
		},
		"logo_url": "/images/logo.png"
	}`), 0o644)

	_, r, _, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfigWithScenarioParent(t, parentDir))

	req := httptest.NewRequest("GET", "/api/v1/discover/"+scenarioName, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	draft, _ := result["draft_brand"].(map[string]interface{})
	if draft == nil {
		t.Fatal("expected draft_brand")
	}

	identity, _ := draft["identity"].(map[string]interface{})
	if identity["display_name"] != "My Site" {
		t.Errorf("display_name = %v, want My Site", identity["display_name"])
	}
	if identity["tagline"] != "Build better" {
		t.Errorf("tagline = %v, want Build better", identity["tagline"])
	}
	if identity["logo_path"] != "/images/logo.png" {
		t.Errorf("logo_path = %v, want /images/logo.png", identity["logo_path"])
	}

	colors, _ := draft["colors"].(map[string]interface{})
	if colors == nil {
		t.Fatal("expected colors from LPBS branding.json")
	}
	if colors["primary"] != "#3498db" {
		t.Errorf("primary = %v, want #3498db", colors["primary"])
	}
}

func TestDiscoverScenario_Manifest(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)

	// Create manifest.json
	os.MkdirAll(filepath.Join(scenarioDir, "ui", "public"), 0o755)
	os.WriteFile(filepath.Join(scenarioDir, "ui", "public", "manifest.json"), []byte(`{
		"name": "PWA App",
		"description": "Progressive web app",
		"theme_color": "#6366f1",
		"background_color": "#f8fafc"
	}`), 0o644)

	_, r, _, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfigWithScenarioParent(t, parentDir))

	req := httptest.NewRequest("GET", "/api/v1/discover/"+scenarioName, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	draft, _ := result["draft_brand"].(map[string]interface{})
	if draft == nil {
		t.Fatal("expected draft_brand")
	}
	colors, _ := draft["colors"].(map[string]interface{})
	if colors["primary"] != "#6366f1" {
		t.Errorf("primary from theme_color = %v, want #6366f1", colors["primary"])
	}
}

func TestDiscoverScenario_NotFound(t *testing.T) {
	_, r, _, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfig(t))

	req := httptest.NewRequest("GET", "/api/v1/discover/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDiscoverScenario_EmptyScenario(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)

	_, r, _, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfigWithScenarioParent(t, parentDir))

	req := httptest.NewRequest("GET", "/api/v1/discover/"+scenarioName, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	if result["confidence"].(float64) != 0.0 {
		t.Errorf("confidence = %v, want 0.0 for empty scenario", result["confidence"])
	}

	suggestions, _ := result["suggestions"].([]interface{})
	if len(suggestions) == 0 {
		t.Error("expected suggestions for empty scenario")
	}
}

func TestImportDiscovery(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)

	// Create branding.json [REQ:BM-REQ-DISC-IMPORT]
	vrooliDir := filepath.Join(scenarioDir, ".vrooli")
	os.MkdirAll(vrooliDir, 0o755)
	os.WriteFile(filepath.Join(vrooliDir, "branding.json"), []byte(`{
		"site_name": "Import Test",
		"theme": {"primary": "#ff0000"}
	}`), 0o644)

	_, r, brandRepo, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfigWithScenarioParent(t, parentDir))

	req := httptest.NewRequest("POST", "/api/v1/discover/"+scenarioName+"/import", bytes.NewReader(nil))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	// Verify brand was created in repo
	brands, _ := brandRepo.List(nil, domain.BrandFilter{})
	if len(brands) == 0 {
		t.Error("expected brand to be created")
	}
}

func TestImportDiscovery_Empty(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)

	_, r, _, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfigWithScenarioParent(t, parentDir))

	req := httptest.NewRequest("POST", "/api/v1/discover/"+scenarioName+"/import", bytes.NewReader(nil))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d (nothing to import)", w.Code, http.StatusBadRequest)
	}
}

func TestImportDiscovery_DryRun(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)

	vrooliDir := filepath.Join(scenarioDir, ".vrooli")
	os.MkdirAll(vrooliDir, 0o755)
	os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte(`{"name": "Test"}`), 0o644)

	_, r, brandRepo, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfigWithScenarioParent(t, parentDir))

	req := httptest.NewRequest("POST", "/api/v1/discover/"+scenarioName+"/import", bytes.NewReader(nil))
	req.Header.Set("X-Dry-Run", "true")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify nothing was persisted
	brands, _ := brandRepo.List(nil, domain.BrandFilter{})
	if len(brands) != 0 {
		t.Error("expected 0 brands after dry-run import")
	}
}

func TestDiscoverScenario_Assets(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)

	// Create asset files
	publicDir := filepath.Join(scenarioDir, "ui", "public")
	os.MkdirAll(publicDir, 0o755)
	os.WriteFile(filepath.Join(publicDir, "favicon.ico"), []byte("ico"), 0o644)
	os.WriteFile(filepath.Join(publicDir, "logo.png"), []byte("png"), 0o644)

	_, r, _, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfigWithScenarioParent(t, parentDir))

	req := httptest.NewRequest("GET", "/api/v1/discover/"+scenarioName, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	sources, _ := result["sources"].([]interface{})
	assetSources := 0
	for _, s := range sources {
		src := s.(map[string]interface{})
		if src["type"] == "asset" {
			assetSources++
		}
	}
	if assetSources < 2 {
		t.Errorf("expected at least 2 asset sources, got %d", assetSources)
	}

	draft, _ := result["draft_brand"].(map[string]interface{})
	identity, _ := draft["identity"].(map[string]interface{})
	if identity["favicon_path"] == nil {
		t.Error("expected favicon_path in draft identity")
	}
	if identity["logo_path"] == nil {
		t.Error("expected logo_path in draft identity")
	}
}
