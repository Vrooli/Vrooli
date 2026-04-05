package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"brand-manager/config"
	"brand-manager/handlers"
)

// [REQ:BM-REQ-SCAN-PLUGINS] [REQ:BM-REQ-SCAN-EXTEND] [REQ:BM-REQ-SCAN-CSS] [REQ:BM-REQ-SCAN-JSON]

func TestNewScannerRegistry_HasDefaultPlugins(t *testing.T) {
	reg := handlers.NewScannerRegistry()
	plugins := reg.ListPlugins()

	if len(plugins) < 2 {
		t.Fatalf("expected at least 2 default plugins, got %d", len(plugins))
	}

	names := make(map[string]bool)
	for _, p := range plugins {
		names[p.Name] = true
	}
	if !names["css"] {
		t.Error("missing default css plugin")
	}
	if !names["json"] {
		t.Error("missing default json plugin")
	}
}

func TestScannerRegistry_Register(t *testing.T) {
	reg := handlers.NewScannerRegistry()
	reg.Register(&handlers.YAMLPlugin{})
	reg.Register(&handlers.HTMLPlugin{})

	plugins := reg.ListPlugins()
	if len(plugins) != 4 {
		t.Fatalf("expected 4 plugins after registration, got %d", len(plugins))
	}

	yamlPlugin := reg.PluginForExt(".yaml")
	if yamlPlugin == nil || yamlPlugin.Name() != "yaml" {
		t.Error("expected yaml plugin for .yaml extension")
	}

	htmlPlugin := reg.PluginForExt(".html")
	if htmlPlugin == nil || htmlPlugin.Name() != "html" {
		t.Error("expected html plugin for .html extension")
	}
}

func TestCSSPlugin_ScanFile(t *testing.T) {
	dir := t.TempDir()
	cssPath := filepath.Join(dir, "test.css")
	os.WriteFile(cssPath, []byte(":root {\n  --brand-primary: #ff0000; /* brand-manager:primary */\n}\n"), 0o644)

	plugin := &handlers.CSSPlugin{}
	results := plugin.ScanFile(cssPath, "test.css")

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Type != "css" {
		t.Errorf("expected type css, got %s", results[0].Type)
	}
	if results[0].Element != "primary" {
		t.Errorf("expected element primary, got %s", results[0].Element)
	}
}

func TestJSONPlugin_ScanFile(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "manifest.json")
	os.WriteFile(jsonPath, []byte(`{"_brand_name": "test", "other": "val"}`), 0o644)

	plugin := &handlers.JSONPlugin{}
	results := plugin.ScanFile(jsonPath, "manifest.json")

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Type != "json" {
		t.Errorf("expected type json, got %s", results[0].Type)
	}
}

func TestYAMLPlugin_ScanFile(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "brand.yaml")
	os.WriteFile(yamlPath, []byte("_brand_name: MyBrand\nother: value\n_brand_color: \"#ff0000\"\n"), 0o644)

	plugin := &handlers.YAMLPlugin{}
	results := plugin.ScanFile(yamlPath, "brand.yaml")

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Type != "yaml" {
			t.Errorf("expected type yaml, got %s", r.Type)
		}
	}
}

func TestHTMLPlugin_ScanFile(t *testing.T) {
	dir := t.TempDir()
	htmlPath := filepath.Join(dir, "index.html")
	os.WriteFile(htmlPath, []byte(`<div data-brand-primary="#ff0000" data-brand-logo="logo.svg">Hello</div>`), 0o644)

	plugin := &handlers.HTMLPlugin{}
	results := plugin.ScanFile(htmlPath, "index.html")

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	elements := map[string]bool{}
	for _, r := range results {
		elements[r.Element] = true
		if r.Type != "html" {
			t.Errorf("expected type html, got %s", r.Type)
		}
	}
	if !elements["primary"] || !elements["logo"] {
		t.Error("expected primary and logo elements")
	}
}

func TestScanScenarioWithPlugins(t *testing.T) {
	dir := t.TempDir()
	scenarioDir := filepath.Join(dir, "test-scenario")
	os.MkdirAll(filepath.Join(scenarioDir, "ui", "src"), 0o755)

	// Create CSS file with markers
	os.WriteFile(
		filepath.Join(scenarioDir, "ui", "src", "brand.css"),
		[]byte(":root {\n  --brand-primary: #ff0000; /* brand-manager:primary */\n}\n"),
		0o644,
	)
	// Create YAML file with brand keys
	os.WriteFile(
		filepath.Join(scenarioDir, "brand.yaml"),
		[]byte("_brand_color: \"#ff0000\"\n"),
		0o644,
	)

	cfg := config.Default()
	cfg.ScenariosDir = dir

	_, router, _, _, _ := setupMockServerWithConfig(t, cfg)

	req := httptest.NewRequest("GET", "/api/v1/scan-ext/test-scenario", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var report map[string]interface{}
	json.NewDecoder(w.Body).Decode(&report)

	results, ok := report["results"].([]interface{})
	if !ok {
		t.Fatal("expected results array")
	}
	if len(results) < 2 {
		t.Errorf("expected at least 2 results (css + yaml), got %d", len(results))
	}
}

func TestListScanPlugins(t *testing.T) {
	cfg := config.Default()
	_, router, _, _, _ := setupMockServerWithConfig(t, cfg)

	req := httptest.NewRequest("GET", "/api/v1/scanner/plugins", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)

	plugins, ok := body["plugins"].([]interface{})
	if !ok || len(plugins) < 4 {
		t.Errorf("expected at least 4 plugins (css, json, yaml, html), got %v", len(plugins))
	}
}

func TestApplyPreview(t *testing.T) {
	dir := t.TempDir()
	scenarioDir := filepath.Join(dir, "test-scenario")
	os.MkdirAll(filepath.Join(scenarioDir, "ui", "src", "styles"), 0o755)

	cfg := config.Default()
	cfg.ScenariosDir = dir

	_, router, brandRepo, _, _ := setupMockServerWithConfig(t, cfg)

	// Seed a brand with colors
	brandRepo.SeedBrand("brand-1", "Test Brand", "#ff0000", "#00ff00")

	body := `{"scenario_name":"test-scenario"}`
	req := httptest.NewRequest("POST", "/api/v1/brands/brand-1/apply/preview", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	if dryRun, ok := result["dry_run"].(bool); !ok || !dryRun {
		t.Error("expected dry_run=true in preview response")
	}
}

func TestThemePreview_LightMode(t *testing.T) {
	cfg := config.Default()
	_, router, brandRepo, _, _ := setupMockServerWithConfig(t, cfg)

	brandRepo.SeedBrand("brand-1", "Test Brand", "#ff0000", "#00ff00")

	req := httptest.NewRequest("GET", "/api/v1/brands/brand-1/theme-preview?mode=light", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["mode"] != "light" {
		t.Errorf("expected mode=light, got %v", resp["mode"])
	}
	if css, ok := resp["css"].(string); !ok || !strings.Contains(css, "--brand-primary") {
		t.Error("expected CSS with --brand-primary")
	}
}

func TestThemePreview_DarkMode(t *testing.T) {
	cfg := config.Default()
	_, router, brandRepo, _, _ := setupMockServerWithConfig(t, cfg)

	brandRepo.SeedBrand("brand-1", "Test Brand", "#ff0000", "#00ff00")

	req := httptest.NewRequest("GET", "/api/v1/brands/brand-1/theme-preview?mode=dark", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["mode"] != "dark" {
		t.Errorf("expected mode=dark, got %v", resp["mode"])
	}

	tokens, ok := resp["tokens"].(map[string]interface{})
	if !ok {
		t.Fatal("expected tokens map")
	}
	// Dark mode should invert background
	if bg, ok := tokens["background"].(string); ok && bg == "#ffffff" {
		t.Error("dark mode should not use light background")
	}
}

func TestThemePreview_BrandNotFound(t *testing.T) {
	cfg := config.Default()
	_, router, _, _, _ := setupMockServerWithConfig(t, cfg)

	req := httptest.NewRequest("GET", "/api/v1/brands/nonexistent/theme-preview", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGenerateOptions(t *testing.T) {
	cfg := config.Default()
	_, router, _, _, _ := setupMockServerWithConfig(t, cfg)

	req := httptest.NewRequest("GET", "/api/v1/brands/generate/options", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)

	providers, ok := body["providers"].([]interface{})
	if !ok || len(providers) < 1 {
		t.Error("expected at least 1 provider")
	}

	elements, ok := body["elements"].([]interface{})
	if !ok || len(elements) < 1 {
		t.Error("expected at least 1 element type")
	}
}

func TestGenerateOptions_Availability(t *testing.T) {
	// Helper to extract provider availability from the response body.
	providerAvail := func(t *testing.T, body map[string]interface{}) map[string]bool {
		t.Helper()
		result := map[string]bool{}
		providers := body["providers"].([]interface{})
		for _, p := range providers {
			pm := p.(map[string]interface{})
			result[pm["id"].(string)] = pm["available"].(bool)
		}
		return result
	}

	// Fake Ollama server that responds 200 to /api/tags.
	fakeOllama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"models":[]}`))
	}))
	defer fakeOllama.Close()

	tests := []struct {
		name             string
		ollamaURL        string
		openRouterKey    string
		expectOllama     bool
		expectOpenRouter bool
	}{
		{
			name:             "neither configured",
			expectOllama:     false,
			expectOpenRouter: false,
		},
		{
			name:             "ollama only",
			ollamaURL:        fakeOllama.URL,
			expectOllama:     true,
			expectOpenRouter: false,
		},
		{
			name:             "openrouter only",
			openRouterKey:    "sk-test-key",
			expectOllama:     false,
			expectOpenRouter: true,
		},
		{
			name:             "both configured",
			ollamaURL:        fakeOllama.URL,
			openRouterKey:    "sk-test-key",
			expectOllama:     true,
			expectOpenRouter: true,
		},
		{
			name:             "ollama configured but unreachable",
			ollamaURL:        "http://127.0.0.1:1", // nothing listening
			expectOllama:     false,
			expectOpenRouter: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.OllamaURL = tt.ollamaURL
			cfg.OpenRouterAPIKey = tt.openRouterKey
			_, router, _, _, _ := setupMockServerWithConfig(t, cfg)

			req := httptest.NewRequest("GET", "/api/v1/brands/generate/options", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", w.Code)
			}

			var body map[string]interface{}
			json.NewDecoder(w.Body).Decode(&body)

			avail := providerAvail(t, body)
			if avail["ollama"] != tt.expectOllama {
				t.Errorf("ollama: got available=%v, want %v", avail["ollama"], tt.expectOllama)
			}
			if avail["openrouter"] != tt.expectOpenRouter {
				t.Errorf("openrouter: got available=%v, want %v", avail["openrouter"], tt.expectOpenRouter)
			}
			// Manual should always be available.
			if !avail["manual"] {
				t.Error("manual should always be available")
			}
		})
	}
}
