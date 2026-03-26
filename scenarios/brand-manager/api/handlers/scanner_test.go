package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"brand-manager/config"
	"brand-manager/domain"
)

// [REQ:BM-REQ-SCAN-CSS] [REQ:BM-REQ-SCAN-JSON] [REQ:BM-REQ-SCAN-PARTIAL]

func TestScanScenarioCSS(t *testing.T) {
	h, _, _, _, _ := setupMockServer(t)

	// Create temp scenario directory with CSS file
	tmpDir := t.TempDir()
	scenarioDir := filepath.Join(tmpDir, "test-scenario")
	os.MkdirAll(filepath.Join(scenarioDir, "ui", "src"), 0o755)

	cssContent := `
:root {
  --primary: #3498db; /* brand-manager:primary */
  --secondary: #2ecc71; /* brand-manager:secondary */
  --bg: #ffffff;
}
`
	os.WriteFile(filepath.Join(scenarioDir, "ui", "src", "theme.css"), []byte(cssContent), 0o644)

	cfg := config.Default()
	cfg.ScenariosDir = tmpDir
	h2 := h.WithConfig(cfg)
	router2 := setupRouterWith(h2)

	req := httptest.NewRequest("GET", "/api/v1/scan/test-scenario", nil)
	w := httptest.NewRecorder()
	router2.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var report domain.ScanReport
	json.NewDecoder(w.Body).Decode(&report)

	if report.CSSMarkers != 2 {
		t.Errorf("CSSMarkers = %d, want 2", report.CSSMarkers)
	}
	if report.Scenario != "test-scenario" {
		t.Errorf("Scenario = %q, want %q", report.Scenario, "test-scenario")
	}

	// Check individual results
	foundPrimary := false
	foundSecondary := false
	for _, r := range report.Results {
		if r.Element == "primary" {
			foundPrimary = true
		}
		if r.Element == "secondary" {
			foundSecondary = true
		}
	}
	if !foundPrimary {
		t.Error("expected to find 'primary' marker")
	}
	if !foundSecondary {
		t.Error("expected to find 'secondary' marker")
	}
}

func TestScanScenarioJSON(t *testing.T) {
	h, _, _, _, _ := setupMockServer(t)

	tmpDir := t.TempDir()
	scenarioDir := filepath.Join(tmpDir, "json-scenario")
	os.MkdirAll(scenarioDir, 0o755)

	jsonContent := `{
  "name": "Test App",
  "_brand_primary": "#3498db",
  "_brand_logo_url": "/assets/logo.png",
  "other_key": "value"
}`
	os.WriteFile(filepath.Join(scenarioDir, "manifest.json"), []byte(jsonContent), 0o644)

	cfg := config.Default()
	cfg.ScenariosDir = tmpDir
	h2 := h.WithConfig(cfg)
	router2 := setupRouterWith(h2)

	req := httptest.NewRequest("GET", "/api/v1/scan/json-scenario", nil)
	w := httptest.NewRecorder()
	router2.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var report domain.ScanReport
	json.NewDecoder(w.Body).Decode(&report)

	if report.JSONKeys != 2 {
		t.Errorf("JSONKeys = %d, want 2", report.JSONKeys)
	}
}

func TestScanScenarioPartial(t *testing.T) {
	h, _, _, _, _ := setupMockServer(t)

	// Create scenario with both CSS and JSON markers
	tmpDir := t.TempDir()
	scenarioDir := filepath.Join(tmpDir, "mixed-scenario")
	os.MkdirAll(scenarioDir, 0o755)

	os.WriteFile(filepath.Join(scenarioDir, "style.css"),
		[]byte("/* brand-manager:accent */"), 0o644)
	os.WriteFile(filepath.Join(scenarioDir, "config.json"),
		[]byte(`{"_brand_name": "test"}`), 0o644)

	cfg := config.Default()
	cfg.ScenariosDir = tmpDir
	h2 := h.WithConfig(cfg)
	router2 := setupRouterWith(h2)

	req := httptest.NewRequest("GET", "/api/v1/scan/mixed-scenario", nil)
	w := httptest.NewRecorder()
	router2.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var report domain.ScanReport
	json.NewDecoder(w.Body).Decode(&report)

	if report.CSSMarkers != 1 {
		t.Errorf("CSSMarkers = %d, want 1", report.CSSMarkers)
	}
	if report.JSONKeys != 1 {
		t.Errorf("JSONKeys = %d, want 1", report.JSONKeys)
	}
	if report.Total != 2 {
		t.Errorf("Total = %d, want 2", report.Total)
	}
}

func TestScanScenarioNotFound(t *testing.T) {
	h, _, _, _, _ := setupMockServer(t)

	cfg := config.Default()
	cfg.ScenariosDir = t.TempDir()
	h2 := h.WithConfig(cfg)
	router2 := setupRouterWith(h2)

	req := httptest.NewRequest("GET", "/api/v1/scan/nonexistent", nil)
	w := httptest.NewRecorder()
	router2.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// [REQ:BM-REQ-SCAN-CSS] Tests SCSS and LESS files are scanned for CSS markers
func TestScanScenarioSCSSandLESS(t *testing.T) {
	h, _, _, _, _ := setupMockServer(t)

	tmpDir := t.TempDir()
	scenarioDir := filepath.Join(tmpDir, "scss-scenario")
	os.MkdirAll(scenarioDir, 0o755)

	os.WriteFile(filepath.Join(scenarioDir, "theme.scss"),
		[]byte("$primary: #333; /* brand-manager:primary */\n$accent: #f00; /* brand-manager:accent */"), 0o644)
	os.WriteFile(filepath.Join(scenarioDir, "vars.less"),
		[]byte("@bg: #fff; /* brand-manager:background */"), 0o644)

	cfg := config.Default()
	cfg.ScenariosDir = tmpDir
	h2 := h.WithConfig(cfg)
	router2 := setupRouterWith(h2)

	req := httptest.NewRequest("GET", "/api/v1/scan/scss-scenario", nil)
	w := httptest.NewRecorder()
	router2.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var report domain.ScanReport
	json.NewDecoder(w.Body).Decode(&report)

	if report.CSSMarkers != 3 {
		t.Errorf("CSSMarkers = %d, want 3 (2 scss + 1 less)", report.CSSMarkers)
	}
	if report.Total != 3 {
		t.Errorf("Total = %d, want 3", report.Total)
	}
}

// [REQ:BM-REQ-SCAN-CSS] Tests multiple markers per line in a single CSS file
func TestScanScenarioMultipleMarkersPerLine(t *testing.T) {
	h, _, _, _, _ := setupMockServer(t)

	tmpDir := t.TempDir()
	scenarioDir := filepath.Join(tmpDir, "multi-marker")
	os.MkdirAll(scenarioDir, 0o755)

	// Two markers on the same line
	os.WriteFile(filepath.Join(scenarioDir, "style.css"),
		[]byte("/* brand-manager:primary */ /* brand-manager:secondary */\n"), 0o644)

	cfg := config.Default()
	cfg.ScenariosDir = tmpDir
	h2 := h.WithConfig(cfg)
	router2 := setupRouterWith(h2)

	req := httptest.NewRequest("GET", "/api/v1/scan/multi-marker", nil)
	w := httptest.NewRecorder()
	router2.ServeHTTP(w, req)

	var report domain.ScanReport
	json.NewDecoder(w.Body).Decode(&report)

	if report.CSSMarkers != 2 {
		t.Errorf("CSSMarkers = %d, want 2 (two markers on one line)", report.CSSMarkers)
	}
}

// [REQ:BM-REQ-SCAN-CSS] [REQ:BM-REQ-SCAN-JSON] Tests that node_modules and .git are skipped
func TestScanScenarioSkipsDirs(t *testing.T) {
	h, _, _, _, _ := setupMockServer(t)

	tmpDir := t.TempDir()
	scenarioDir := filepath.Join(tmpDir, "skip-test")
	os.MkdirAll(filepath.Join(scenarioDir, "node_modules"), 0o755)
	os.MkdirAll(filepath.Join(scenarioDir, ".git"), 0o755)
	os.MkdirAll(filepath.Join(scenarioDir, "src"), 0o755)

	// Markers in skip dirs should be ignored
	os.WriteFile(filepath.Join(scenarioDir, "node_modules", "vendor.css"),
		[]byte("/* brand-manager:ignored */"), 0o644)
	os.WriteFile(filepath.Join(scenarioDir, ".git", "hooks.json"),
		[]byte(`{"_brand_ignored": true}`), 0o644)
	// This one should be found
	os.WriteFile(filepath.Join(scenarioDir, "src", "app.css"),
		[]byte("/* brand-manager:found */"), 0o644)

	cfg := config.Default()
	cfg.ScenariosDir = tmpDir
	h2 := h.WithConfig(cfg)
	router2 := setupRouterWith(h2)

	req := httptest.NewRequest("GET", "/api/v1/scan/skip-test", nil)
	w := httptest.NewRecorder()
	router2.ServeHTTP(w, req)

	var report domain.ScanReport
	json.NewDecoder(w.Body).Decode(&report)

	if report.CSSMarkers != 1 {
		t.Errorf("CSSMarkers = %d, want 1 (node_modules and .git should be skipped)", report.CSSMarkers)
	}
	if report.JSONKeys != 0 {
		t.Errorf("JSONKeys = %d, want 0 (.git should be skipped)", report.JSONKeys)
	}
}

// [REQ:BM-REQ-SCAN-PARTIAL] Tests CSS-only scenario reports zero JSON keys
func TestScanScenarioCSSOnlyPartial(t *testing.T) {
	h, _, _, _, _ := setupMockServer(t)

	tmpDir := t.TempDir()
	scenarioDir := filepath.Join(tmpDir, "css-only")
	os.MkdirAll(scenarioDir, 0o755)

	os.WriteFile(filepath.Join(scenarioDir, "theme.css"),
		[]byte("/* brand-manager:primary */\n/* brand-manager:bg */"), 0o644)

	cfg := config.Default()
	cfg.ScenariosDir = tmpDir
	h2 := h.WithConfig(cfg)
	router2 := setupRouterWith(h2)

	req := httptest.NewRequest("GET", "/api/v1/scan/css-only", nil)
	w := httptest.NewRecorder()
	router2.ServeHTTP(w, req)

	var report domain.ScanReport
	json.NewDecoder(w.Body).Decode(&report)

	if report.CSSMarkers != 2 {
		t.Errorf("CSSMarkers = %d, want 2", report.CSSMarkers)
	}
	if report.JSONKeys != 0 {
		t.Errorf("JSONKeys = %d, want 0 for CSS-only scenario", report.JSONKeys)
	}
	if report.Total != 2 {
		t.Errorf("Total = %d, want 2", report.Total)
	}
}

// [REQ:BM-REQ-SCAN-CSS] [REQ:BM-REQ-SCAN-JSON] Tests line numbers are correctly tracked in results
func TestScanScenarioResultLineNumbers(t *testing.T) {
	h, _, _, _, _ := setupMockServer(t)

	tmpDir := t.TempDir()
	scenarioDir := filepath.Join(tmpDir, "lineno-test")
	os.MkdirAll(scenarioDir, 0o755)

	cssContent := "/* no marker */\n/* brand-manager:first */\n/* nothing */\n/* brand-manager:second */"
	os.WriteFile(filepath.Join(scenarioDir, "style.css"), []byte(cssContent), 0o644)

	cfg := config.Default()
	cfg.ScenariosDir = tmpDir
	h2 := h.WithConfig(cfg)
	router2 := setupRouterWith(h2)

	req := httptest.NewRequest("GET", "/api/v1/scan/lineno-test", nil)
	w := httptest.NewRecorder()
	router2.ServeHTTP(w, req)

	var report domain.ScanReport
	json.NewDecoder(w.Body).Decode(&report)

	if len(report.Results) != 2 {
		t.Fatalf("Results count = %d, want 2", len(report.Results))
	}
	if report.Results[0].Line != 2 {
		t.Errorf("first result line = %d, want 2", report.Results[0].Line)
	}
	if report.Results[1].Line != 4 {
		t.Errorf("second result line = %d, want 4", report.Results[1].Line)
	}
}

func TestScanScenarioEmpty(t *testing.T) {
	h, _, _, _, _ := setupMockServer(t)

	tmpDir := t.TempDir()
	scenarioDir := filepath.Join(tmpDir, "empty-scenario")
	os.MkdirAll(scenarioDir, 0o755)

	cfg := config.Default()
	cfg.ScenariosDir = tmpDir
	h2 := h.WithConfig(cfg)
	router2 := setupRouterWith(h2)

	req := httptest.NewRequest("GET", "/api/v1/scan/empty-scenario", nil)
	w := httptest.NewRecorder()
	router2.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var report domain.ScanReport
	json.NewDecoder(w.Body).Decode(&report)

	if report.Total != 0 {
		t.Errorf("Total = %d, want 0", report.Total)
	}
}
