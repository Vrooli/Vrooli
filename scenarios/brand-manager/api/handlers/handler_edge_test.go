package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"brand-manager/config"
	"brand-manager/domain"
)

// Edge case and error path tests for handlers.
// These complement the happy-path tests in other files.

// --- Brand CRUD edge cases ---

// [REQ:BM-REQ-CRUD-CREATE]
func TestCreateBrand_InvalidJSON(t *testing.T) {
	_, router, _, _, _ := setupMockServer(t)

	req := httptest.NewRequest("POST", "/api/v1/brands", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// [REQ:BM-REQ-CRUD-UPDATE]
func TestUpdateBrand_InvalidJSON(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Test", Version: 1})

	req := httptest.NewRequest("PUT", "/api/v1/brands/b1", bytes.NewBufferString("broken"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// [REQ:BM-REQ-CRUD-UPDATE]
func TestUpdateBrand_IfMatchConflict(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Test", Version: 5})

	body := `{"name":"Updated"}`
	req := httptest.NewRequest("PUT", "/api/v1/brands/b1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", "3") // stale version
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d (version conflict)", w.Code, http.StatusConflict)
	}
}

// [REQ:BM-REQ-CRUD-UPDATE]
func TestUpdateBrand_IfMatchSuccess(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Test", Version: 3})

	body := `{"name":"Updated"}`
	req := httptest.NewRequest("PUT", "/api/v1/brands/b1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", "3") // matching version
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var updated domain.Brand
	json.NewDecoder(w.Body).Decode(&updated)
	if updated.Name != "Updated" {
		t.Errorf("name = %q, want %q", updated.Name, "Updated")
	}
}

// [REQ:BM-REQ-CRUD-UPDATE]
func TestUpdateBrand_IfMatchInvalidFormat(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Test", Version: 1})

	body := `{"name":"Updated"}`
	req := httptest.NewRequest("PUT", "/api/v1/brands/b1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", "not-a-number")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// [REQ:BM-REQ-CRUD-UPDATE]
func TestUpdateBrand_RepoError(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Test", Version: 1})
	brandRepo.UpdateErr = errors.New("disk full")

	body := `{"name":"Updated"}`
	req := httptest.NewRequest("PUT", "/api/v1/brands/b1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// [REQ:BM-REQ-CRUD-READ]
func TestListBrands_WithNameFilter(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Alpha Brand", Version: 1})
	brandRepo.Seed(&domain.Brand{ID: "b2", Name: "Beta Brand", Version: 1})
	brandRepo.Seed(&domain.Brand{ID: "b3", Name: "Alpha Other", Version: 1})

	req := httptest.NewRequest("GET", "/api/v1/brands?name=Alpha", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var brands []domain.Brand
	json.NewDecoder(w.Body).Decode(&brands)
	if len(brands) != 2 {
		t.Errorf("expected 2 brands matching 'Alpha', got %d", len(brands))
	}
}

// [REQ:BM-REQ-CRUD-READ]
func TestListBrands_EmptyResult(t *testing.T) {
	_, router, _, _, _ := setupMockServer(t)

	req := httptest.NewRequest("GET", "/api/v1/brands", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Should return empty array, not null
	body := strings.TrimSpace(w.Body.String())
	if !strings.HasPrefix(body, "[") {
		t.Errorf("empty list should return JSON array, got: %s", body)
	}
}

// [REQ:BM-REQ-CRUD-READ]
func TestListBrands_RepoError(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.ListErr = errors.New("connection lost")

	req := httptest.NewRequest("GET", "/api/v1/brands", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// --- Assignment edge cases ---

// [REQ:BM-REQ-ASSIGN-LINK]
func TestCreateAssignment_MissingFields(t *testing.T) {
	_, router, _, _, _ := setupMockServer(t)

	tests := []struct {
		name string
		body string
	}{
		{"missing brand_id", `{"scenario_name":"test"}`},
		{"missing scenario_name", `{"brand_id":"b1"}`},
		{"both empty", `{"brand_id":"","scenario_name":""}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/assignments", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

// [REQ:BM-REQ-API-ASSIGN]
func TestDeleteAssignment_Idempotent(t *testing.T) {
	_, router, _, _, _ := setupMockServer(t)

	// Delete nonexistent — should still return 204 (idempotent)
	req := httptest.NewRequest("DELETE", "/api/v1/assignments/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d (idempotent delete)", w.Code, http.StatusNoContent)
	}
}

// --- Apply edge cases ---

// [REQ:BM-REQ-APPLY-CSS]
func TestApplyBrand_Typography(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)

	_, r, brandRepo, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfigWithScenarioParent(t, parentDir))

	brandRepo.Seed(&domain.Brand{
		ID:   "b1",
		Name: "Test Brand",
		Typography: &domain.Typography{
			HeadingFont:  "Inter",
			BodyFont:     "Open Sans",
			BaseFontSize: "18px",
		},
		Version: 1,
	})

	body := `{"scenario_name":"` + scenarioName + `","elements":["typography"]}`
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/apply", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify typography CSS was written
	cssPath := filepath.Join(scenarioDir, "ui", "src", "styles", "brand.css")
	data, err := os.ReadFile(cssPath)
	if err != nil {
		t.Fatalf("CSS file not created: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "--brand-heading-font: Inter") {
		t.Error("missing heading font CSS variable")
	}
	if !strings.Contains(content, "--brand-body-font: Open Sans") {
		t.Error("missing body font CSS variable")
	}
}

// [REQ:BM-REQ-APPLY-PARTIAL]
func TestApplyBrand_UnknownElement(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)

	_, r, brandRepo, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfigWithScenarioParent(t, parentDir))

	brandRepo.Seed(&domain.Brand{
		ID:      "b1",
		Name:    "Test",
		Colors:  &domain.Colors{Primary: "#ff0000"},
		Version: 1,
	})

	body := `{"scenario_name":"` + scenarioName + `","elements":["unknown_element"]}`
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/apply", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	skipped, _ := result["skipped"].([]interface{})
	if len(skipped) == 0 {
		t.Error("expected unknown element to appear in skipped")
	}
	first := skipped[0].(map[string]interface{})
	if first["element"] != "unknown_element" {
		t.Errorf("skipped element = %v, want unknown_element", first["element"])
	}
}

// [REQ:BM-REQ-APPLY-CSS]
func TestApplyBrand_MissingScenarioName(t *testing.T) {
	_, r, brandRepo, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfig(t))

	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Test", Version: 1})

	body := `{"scenario_name":""}`
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/apply", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// [REQ:BM-REQ-APPLY-JSON]
func TestApplyBrand_JSONMergesExistingManifest(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)

	// Create pre-existing manifest
	publicDir := filepath.Join(scenarioDir, "ui", "public")
	os.MkdirAll(publicDir, 0o755)
	os.WriteFile(filepath.Join(publicDir, "manifest.json"), []byte(`{
  "start_url": "/",
  "display": "standalone"
}`), 0o644)

	_, r, brandRepo, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfigWithScenarioParent(t, parentDir))

	brandRepo.Seed(&domain.Brand{
		ID:       "b1",
		Name:     "Test",
		Identity: &domain.Identity{DisplayName: "My App"},
		Version:  1,
	})

	body := `{"scenario_name":"` + scenarioName + `","elements":["identity"]}`
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/apply", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify existing keys preserved and brand keys added
	data, _ := os.ReadFile(filepath.Join(publicDir, "manifest.json"))
	var manifest map[string]interface{}
	json.Unmarshal(data, &manifest)

	if manifest["start_url"] != "/" {
		t.Error("existing start_url was overwritten")
	}
	if manifest["display"] != "standalone" {
		t.Error("existing display was overwritten")
	}
	if manifest["name"] != "My App" {
		t.Errorf("name = %v, want My App", manifest["name"])
	}
	if manifest["_brand_display_name"] != "My App" {
		t.Errorf("_brand_display_name = %v, want My App", manifest["_brand_display_name"])
	}
}

// --- Discovery edge cases ---

// [REQ:BM-REQ-DISC-SCAN]
func TestDiscoverScenario_CSSThemeDiscovery(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)

	// Create theme CSS with brand-related custom properties
	styleDir := filepath.Join(scenarioDir, "ui", "src", "styles")
	os.MkdirAll(styleDir, 0o755)
	os.WriteFile(filepath.Join(styleDir, "theme.css"), []byte(`:root {
  --brand-primary: #3498db;
  --brand-secondary: #2ecc71;
  --primary: #6366f1;
}
`), 0o644)

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
	hasCSS := false
	for _, s := range sources {
		src := s.(map[string]interface{})
		if src["type"] == "theme_css" {
			hasCSS = true
			fields := int(src["fields"].(float64))
			if fields < 2 {
				t.Errorf("expected at least 2 CSS fields, got %d", fields)
			}
		}
	}
	if !hasCSS {
		t.Error("expected theme_css source from CSS discovery")
	}
}

// [REQ:BM-REQ-DISC-SCAN]
func TestDiscoverScenario_MultipleSources_ConfidenceAverage(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)

	// Create service.json + manifest.json
	vrooliDir := filepath.Join(scenarioDir, ".vrooli")
	os.MkdirAll(vrooliDir, 0o755)
	os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte(`{"name": "Test"}`), 0o644)

	publicDir := filepath.Join(scenarioDir, "ui", "public")
	os.MkdirAll(publicDir, 0o755)
	os.WriteFile(filepath.Join(publicDir, "manifest.json"), []byte(`{"name": "PWA", "theme_color": "#ff0000"}`), 0o644)

	_, r, _, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfigWithScenarioParent(t, parentDir))

	req := httptest.NewRequest("GET", "/api/v1/discover/"+scenarioName, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	sources, _ := result["sources"].([]interface{})
	if len(sources) < 2 {
		t.Errorf("expected at least 2 sources, got %d", len(sources))
	}

	confidence := result["confidence"].(float64)
	if confidence <= 0 || confidence > 1.0 {
		t.Errorf("confidence = %f, want between 0 and 1.0", confidence)
	}
}

// [REQ:BM-REQ-DISC-SCAN]
func TestDiscoverScenario_MalformedJSON_SilentlySkipped(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)

	// Create malformed service.json
	vrooliDir := filepath.Join(scenarioDir, ".vrooli")
	os.MkdirAll(vrooliDir, 0o755)
	os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte("{bad json"), 0o644)

	_, r, _, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfigWithScenarioParent(t, parentDir))

	req := httptest.NewRequest("GET", "/api/v1/discover/"+scenarioName, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (malformed files should be skipped)", w.Code, http.StatusOK)
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	// Should return 0 confidence since malformed JSON was skipped
	if result["confidence"].(float64) != 0 {
		t.Errorf("confidence = %v, want 0 (malformed file)", result["confidence"])
	}
}

// [REQ:BM-REQ-DISC-IMPORT]
func TestImportDiscovery_NotFoundScenario(t *testing.T) {
	_, r, _, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfig(t))

	req := httptest.NewRequest("POST", "/api/v1/discover/nonexistent/import", bytes.NewReader(nil))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// --- Scan endpoint edge cases ---

// [REQ:BM-REQ-SCAN-CSS]
func TestScanScenario_NotFound(t *testing.T) {
	_, r, _, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfig(t))

	req := httptest.NewRequest("GET", "/api/v1/scan/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// [REQ:BM-REQ-SCAN-CSS] [REQ:BM-REQ-SCAN-JSON]
func TestScanScenario_MixedFileTypes(t *testing.T) {
	dir := t.TempDir()
	scenarioDir := filepath.Join(dir, "mixed")
	os.MkdirAll(filepath.Join(scenarioDir, "ui"), 0o755)

	os.WriteFile(filepath.Join(scenarioDir, "ui", "brand.css"),
		[]byte("/* brand-manager:primary */\n/* brand-manager:text */\n"), 0o644)
	os.WriteFile(filepath.Join(scenarioDir, "manifest.json"),
		[]byte(`{"_brand_id": "b1"}`), 0o644)
	os.WriteFile(filepath.Join(scenarioDir, "README.md"),
		[]byte("# No brand markers here"), 0o644)

	_, r, _, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfigWithScenarioParent(t, dir))

	req := httptest.NewRequest("GET", "/api/v1/scan/mixed", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var report domain.ScanReport
	json.NewDecoder(w.Body).Decode(&report)

	if report.CSSMarkers != 2 {
		t.Errorf("css_markers = %d, want 2", report.CSSMarkers)
	}
	if report.JSONKeys != 1 {
		t.Errorf("json_keys = %d, want 1", report.JSONKeys)
	}
	if report.Total != 3 {
		t.Errorf("total = %d, want 3", report.Total)
	}
}

// --- Theme preview edge cases ---

// [REQ:BM-REQ-UI-THEME]
func TestThemePreview_WithTypography(t *testing.T) {
	cfg := config.Default()
	_, router, brandRepo, _, _ := setupMockServerWithConfig(t, cfg)

	brandRepo.SeedBrand("brand-1", "Test Brand", "#ff0000", "#00ff00")

	req := httptest.NewRequest("GET", "/api/v1/brands/brand-1/theme-preview", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	tokens := resp["tokens"].(map[string]interface{})
	// SeedBrand includes typography
	if tokens["heading-font"] != "Inter" {
		t.Errorf("heading-font = %v, want Inter", tokens["heading-font"])
	}
	// CSS should include typography variables
	css := resp["css"].(string)
	if !strings.Contains(css, "--brand-heading-font") {
		t.Error("CSS missing typography variables")
	}
}

// [REQ:BM-REQ-UI-THEME]
func TestThemePreview_DefaultsToLightMode(t *testing.T) {
	cfg := config.Default()
	_, router, brandRepo, _, _ := setupMockServerWithConfig(t, cfg)
	brandRepo.SeedBrand("brand-1", "Test", "#ff0000", "#00ff00")

	// No mode parameter
	req := httptest.NewRequest("GET", "/api/v1/brands/brand-1/theme-preview", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["mode"] != "light" {
		t.Errorf("mode = %v, want light (default)", resp["mode"])
	}
}

// --- Scan ext endpoint edge cases ---

// [REQ:BM-REQ-SCAN-PLUGINS]
func TestScanScenarioWithPlugins_NotFound(t *testing.T) {
	cfg := config.Default()
	cfg.ScenariosDir = t.TempDir()
	_, router, _, _, _ := setupMockServerWithConfig(t, cfg)

	req := httptest.NewRequest("GET", "/api/v1/scan-ext/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- Version listing edge cases ---

// [REQ:BM-REQ-API-VERSIONS]
func TestListVersions_EmptyResult(t *testing.T) {
	_, router, _, _, _ := setupMockServer(t)

	req := httptest.NewRequest("GET", "/api/v1/brands/nonexistent/versions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := strings.TrimSpace(w.Body.String())
	if !strings.HasPrefix(body, "[") {
		t.Errorf("empty version list should return JSON array, got: %s", body)
	}
}

// --- Scenario status with assignment ---

// [REQ:BM-REQ-API-STATUS]
func TestScenarioStatus_WithAssignment(t *testing.T) {
	_, router, brandRepo, _, assignRepo := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Test", Version: 2})

	// Create assignment via API
	body := `{"brand_id":"b1","scenario_name":"my-scenario"}`
	req := httptest.NewRequest("POST", "/api/v1/assignments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create assignment: status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	_ = assignRepo // used above indirectly through router

	// Check status
	req = httptest.NewRequest("GET", "/api/v1/scenarios/my-scenario/status", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var status map[string]interface{}
	json.NewDecoder(w.Body).Decode(&status)
	if status["has_brand"] != true {
		t.Error("expected has_brand = true after assignment")
	}
	if status["brand_id"] != "b1" {
		t.Errorf("brand_id = %v, want b1", status["brand_id"])
	}
}

// --- Contrast endpoint edge cases ---

// [REQ:BM-REQ-WCAG-VALIDATE]
func TestCheckBrandContrast_InvalidJSON(t *testing.T) {
	_, router, _, _, _ := setupMockServer(t)

	req := httptest.NewRequest("POST", "/api/v1/contrast/brand", bytes.NewBufferString("not json"))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// [REQ:BM-REQ-WCAG-VALIDATE]
func TestCheckContrast_InvalidJSON(t *testing.T) {
	_, router, _, _, _ := setupMockServer(t)

	req := httptest.NewRequest("POST", "/api/v1/contrast/check", bytes.NewBufferString("{broken"))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
