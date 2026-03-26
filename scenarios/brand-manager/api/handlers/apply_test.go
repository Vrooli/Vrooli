package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"brand-manager/domain"
)

// [REQ:BM-REQ-APPLY-CSS] [REQ:BM-REQ-APPLY-JSON] [REQ:BM-REQ-APPLY-ASSETS] [REQ:BM-REQ-APPLY-PARTIAL]

func TestApplyBrand_Colors(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)

	_, r, brandRepo, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfigWithScenarioParent(t, parentDir))

	brandRepo.Seed(&domain.Brand{
		ID:   "b1",
		Name: "Test Brand",
		Colors: &domain.Colors{
			Primary:   "#ff0000",
			Secondary: "#00ff00",
			Text:      "#333333",
		},
		Version: 1,
	})

	body := `{"scenario_name":"` + scenarioName + `"}`
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/apply", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	applied, _ := result["applied"].([]interface{})
	if len(applied) == 0 {
		t.Fatal("expected at least one applied action")
	}

	// Verify CSS file was created with markers
	cssPath := filepath.Join(scenarioDir, "ui", "src", "styles", "brand.css")
	data, err := os.ReadFile(cssPath)
	if err != nil {
		t.Fatalf("CSS file not created: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "brand-manager:primary") {
		t.Error("CSS missing brand-manager:primary marker")
	}
	if !strings.Contains(content, "--brand-primary: #ff0000") {
		t.Error("CSS missing --brand-primary value")
	}
}

func TestApplyBrand_JSON(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)

	_, r, brandRepo, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfigWithScenarioParent(t, parentDir))

	brandRepo.Seed(&domain.Brand{
		ID:   "b1",
		Name: "Test Brand",
		Identity: &domain.Identity{
			DisplayName: "My App",
			Tagline:     "A great app",
		},
		Version: 2,
	})

	os.MkdirAll(filepath.Join(scenarioDir, "ui", "public"), 0o755)

	body := `{"scenario_name":"` + scenarioName + `","elements":["identity"]}`
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/apply", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify manifest was created with _brand keys [REQ:BM-REQ-APPLY-JSON]
	manifestPath := filepath.Join(scenarioDir, "ui", "public", "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("manifest not created: %v", err)
	}
	var manifest map[string]interface{}
	json.Unmarshal(data, &manifest)

	if manifest["_brand_display_name"] != "My App" {
		t.Errorf("_brand_display_name = %v, want My App", manifest["_brand_display_name"])
	}
	if manifest["_brand_id"] != "b1" {
		t.Errorf("_brand_id = %v, want b1", manifest["_brand_id"])
	}
	if manifest["name"] != "My App" {
		t.Errorf("name = %v, want My App", manifest["name"])
	}
}

func TestApplyBrand_Partial(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)

	_, r, brandRepo, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfigWithScenarioParent(t, parentDir))

	brandRepo.Seed(&domain.Brand{
		ID:   "b1",
		Name: "Test Brand",
		Colors: &domain.Colors{
			Primary: "#ff0000",
		},
		Typography: &domain.Typography{
			HeadingFont: "Inter",
		},
		Identity: &domain.Identity{
			DisplayName: "Should Not Apply",
		},
		Version: 1,
	})

	// Apply only colors (partial) [REQ:BM-REQ-APPLY-PARTIAL]
	body := `{"scenario_name":"` + scenarioName + `","elements":["colors"]}`
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/apply", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	applied, _ := result["applied"].([]interface{})
	for _, a := range applied {
		action := a.(map[string]interface{})
		if action["element"] == "identity" {
			t.Error("identity should not be applied when only colors requested")
		}
	}

	// Verify no manifest.json was created (identity was not requested)
	manifestPath := filepath.Join(scenarioDir, "ui", "public", "manifest.json")
	if _, err := os.Stat(manifestPath); err == nil {
		t.Error("manifest.json should not exist when identity not in elements")
	}
}

func TestApplyBrand_DryRun(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)

	_, r, brandRepo, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfigWithScenarioParent(t, parentDir))

	brandRepo.Seed(&domain.Brand{
		ID:      "b1",
		Name:    "Test Brand",
		Colors:  &domain.Colors{Primary: "#ff0000"},
		Version: 1,
	})

	body := `{"scenario_name":"` + scenarioName + `"}`
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/apply", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Dry-Run", "true")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	if result["dry_run"] != true {
		t.Error("expected dry_run = true")
	}

	// Verify no files were written
	cssPath := filepath.Join(scenarioDir, "ui", "src", "styles", "brand.css")
	if _, err := os.Stat(cssPath); err == nil {
		t.Error("CSS file should not exist after dry run")
	}
}

func TestApplyBrand_NotFound(t *testing.T) {
	_, router, _, _, _ := setupMockServer(t)

	body := `{"scenario_name":"test"}`
	req := httptest.NewRequest("POST", "/api/v1/brands/missing/apply", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestApplyBrand_ScenarioNotFound(t *testing.T) {
	_, r, brandRepo, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfig(t))

	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Test", Version: 1})

	body := `{"scenario_name":"nonexistent"}`
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/apply", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestApplyBrand_AssetCopy(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)
	assetDir := t.TempDir()

	cfg := testConfigWithScenarioParent(t, parentDir)
	cfg.AssetBasePath = assetDir
	_, r, brandRepo, _, _, _ := setupMockServerWithConfigAndRepos(t, cfg)

	// Create a fake favicon asset file
	brandAssetDir := filepath.Join(assetDir, "b1")
	os.MkdirAll(brandAssetDir, 0o755)
	os.WriteFile(filepath.Join(brandAssetDir, "favicon.ico"), []byte("fake-ico"), 0o644)

	os.MkdirAll(filepath.Join(scenarioDir, "ui", "public"), 0o755)

	brandRepo.Seed(&domain.Brand{
		ID:   "b1",
		Name: "Test Brand",
		Identity: &domain.Identity{
			FaviconPath: "favicon.ico",
		},
		Version: 1,
	})

	// [REQ:BM-REQ-APPLY-ASSETS]
	body := `{"scenario_name":"` + scenarioName + `","elements":["favicon"]}`
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/apply", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify favicon was copied
	destPath := filepath.Join(scenarioDir, "ui", "public", "favicon.ico")
	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("favicon not copied: %v", err)
	}
	if string(data) != "fake-ico" {
		t.Errorf("favicon content = %q, want %q", string(data), "fake-ico")
	}
}

func TestApplyBrand_SkipsUndefinedElements(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)

	_, r, brandRepo, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfigWithScenarioParent(t, parentDir))

	// Brand with no colors, no typography, no identity
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Empty Brand", Version: 1})

	body := `{"scenario_name":"` + scenarioName + `"}`
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
		t.Error("expected skipped elements for empty brand")
	}
}
