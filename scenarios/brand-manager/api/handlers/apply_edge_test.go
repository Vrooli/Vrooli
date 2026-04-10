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

// Edge case and deeper tests for apply handler.
// [REQ:BM-REQ-APPLY-CSS] [REQ:BM-REQ-APPLY-JSON] [REQ:BM-REQ-APPLY-ASSETS] [REQ:BM-REQ-APPLY-PARTIAL]

func TestApplyBrand_InvalidBody_Error(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Test", Version: 1})

	req := httptest.NewRequest("POST", "/api/v1/brands/b1/apply", bytes.NewBufferString("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestApplyBrand_MissingScenarioName_Error(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Test", Version: 1})

	body := `{"elements":["colors"]}`
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/apply", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestApplyBrand_UnknownElement_Skipped(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)

	_, r, brandRepo, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfigWithScenarioParent(t, parentDir))
	brandRepo.Seed(&domain.Brand{
		ID:      "b1",
		Name:    "Test",
		Version: 1,
		Colors:  &domain.Colors{Primary: "#ff0000"},
	})

	// Request unknown element "widgets" [REQ:BM-REQ-APPLY-PARTIAL]
	body := `{"scenario_name":"` + scenarioName + `","elements":["widgets"]}`
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
		t.Fatal("expected unknown element to be skipped")
	}
	first := skipped[0].(map[string]interface{})
	if first["element"] != "widgets" {
		t.Errorf("skipped element = %v, want widgets", first["element"])
	}
	if first["reason"] != "unknown element" {
		t.Errorf("skip reason = %v, want 'unknown element'", first["reason"])
	}
}

func TestApplyBrand_Typography_CSSOutput(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)

	_, r, brandRepo, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfigWithScenarioParent(t, parentDir))
	brandRepo.Seed(&domain.Brand{
		ID:   "b1",
		Name: "Typo Brand",
		Typography: &domain.Typography{
			HeadingFont:  "Inter",
			BodyFont:     "Roboto",
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

	// Verify CSS file was created with typography markers
	cssPath := filepath.Join(scenarioDir, "ui", "src", "styles", "brand.css")
	data, err := os.ReadFile(cssPath)
	if err != nil {
		t.Fatalf("CSS file not created: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		"brand-manager:typography",
		"--brand-heading-font: Inter",
		"--brand-body-font: Roboto",
		"--brand-base-font-size: 18px",
	} {
		if !contains(content, want) {
			t.Errorf("CSS missing: %s", want)
		}
	}
}

func TestApplyBrand_AllElementsApplied(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)
	assetDir := t.TempDir()

	cfg := testConfigWithScenarioParent(t, parentDir)
	cfg.AssetBasePath = assetDir
	_, r, brandRepo, _, _, _ := setupMockServerWithConfigAndRepos(t, cfg)

	// Create asset files
	brandAssetDir := filepath.Join(assetDir, "b1")
	os.MkdirAll(brandAssetDir, 0o755)
	os.WriteFile(filepath.Join(brandAssetDir, "logo.png"), []byte("logo-data"), 0o644)
	os.WriteFile(filepath.Join(brandAssetDir, "favicon.ico"), []byte("fav-data"), 0o644)

	os.MkdirAll(filepath.Join(scenarioDir, "ui", "public"), 0o755)

	brandRepo.Seed(&domain.Brand{
		ID:   "b1",
		Name: "Complete Brand",
		Colors: &domain.Colors{
			Primary:   "#111",
			Secondary: "#222",
		},
		Typography: &domain.Typography{
			HeadingFont: "Inter",
			BodyFont:    "Sans",
		},
		Identity: &domain.Identity{
			DisplayName: "Complete",
			Tagline:     "All elements",
			LogoPath:    "logo.png",
			FaviconPath: "favicon.ico",
		},
		Version: 2,
	})

	// Apply all (empty elements = all)
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
	if len(applied) < 5 {
		t.Errorf("expected 5 applied actions (colors, typography, identity, favicon, logo), got %d", len(applied))
	}

	// Verify result metadata
	if result["brand_id"] != "b1" {
		t.Errorf("brand_id = %v, want b1", result["brand_id"])
	}
	if v, ok := result["brand_version"].(float64); !ok || v != 2 {
		t.Errorf("brand_version = %v, want 2", result["brand_version"])
	}
}

func TestApplyBrand_CreatesAssignment(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)

	_, r, brandRepo, _, assignRepo, _ := setupMockServerWithConfigAndRepos(t, testConfigWithScenarioParent(t, parentDir))
	brandRepo.Seed(&domain.Brand{
		ID:      "b1",
		Name:    "Test",
		Colors:  &domain.Colors{Primary: "#f00"},
		Version: 3,
	})

	body := `{"scenario_name":"` + scenarioName + `","elements":["colors"]}`
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/apply", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Non-dry-run should create an assignment
	assignment, _ := assignRepo.GetByScenario(nil, scenarioName)
	if assignment == nil {
		t.Error("expected assignment to be created after non-dry-run apply")
	}
}

func TestApplyBrand_DryRunNoAssignment(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)

	_, r, brandRepo, _, assignRepo, _ := setupMockServerWithConfigAndRepos(t, testConfigWithScenarioParent(t, parentDir))
	brandRepo.Seed(&domain.Brand{
		ID:      "b1",
		Name:    "Test",
		Colors:  &domain.Colors{Primary: "#f00"},
		Version: 1,
	})

	body := `{"scenario_name":"` + scenarioName + `"}`
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/apply", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Dry-Run", "true")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	// Dry-run should not create assignment
	assignment, _ := assignRepo.GetByScenario(nil, scenarioName)
	if assignment != nil {
		t.Error("expected no assignment after dry-run")
	}
}

func TestApplyBrand_AssetAbsolutePath(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)
	assetDir := t.TempDir()

	cfg := testConfigWithScenarioParent(t, parentDir)
	cfg.AssetBasePath = assetDir
	_, r, brandRepo, _, _, _ := setupMockServerWithConfigAndRepos(t, cfg)

	// Create asset at absolute path
	absAssetPath := filepath.Join(assetDir, "absolute-logo.png")
	os.WriteFile(absAssetPath, []byte("abs-logo"), 0o644)

	os.MkdirAll(filepath.Join(scenarioDir, "ui", "public"), 0o755)

	brandRepo.Seed(&domain.Brand{
		ID:   "b1",
		Name: "Test",
		Identity: &domain.Identity{
			LogoPath: absAssetPath, // absolute path
		},
		Version: 1,
	})

	// [REQ:BM-REQ-APPLY-ASSETS]
	body := `{"scenario_name":"` + scenarioName + `","elements":["logo"]}`
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/apply", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	destPath := filepath.Join(scenarioDir, "ui", "public", "absolute-logo.png")
	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("logo not copied: %v", err)
	}
	if string(data) != "abs-logo" {
		t.Errorf("content = %q, want %q", string(data), "abs-logo")
	}
}

func TestApplyBrand_JSON_PreservesExistingKeys(t *testing.T) {
	scenarioDir := t.TempDir()
	scenarioName := filepath.Base(scenarioDir)
	parentDir := filepath.Dir(scenarioDir)

	_, r, brandRepo, _, _, _ := setupMockServerWithConfigAndRepos(t, testConfigWithScenarioParent(t, parentDir))

	brandRepo.Seed(&domain.Brand{
		ID:   "b1",
		Name: "Test",
		Identity: &domain.Identity{
			DisplayName: "New Name",
		},
		Version: 1,
	})

	// Create pre-existing manifest with custom keys
	manifestDir := filepath.Join(scenarioDir, "ui", "public")
	os.MkdirAll(manifestDir, 0o755)
	os.WriteFile(filepath.Join(manifestDir, "manifest.json"), []byte(`{
		"start_url": "/",
		"display": "standalone",
		"name": "Old Name"
	}`), 0o644)

	body := `{"scenario_name":"` + scenarioName + `","elements":["identity"]}`
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/apply", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify existing keys are preserved and brand keys are added [REQ:BM-REQ-APPLY-JSON]
	data, _ := os.ReadFile(filepath.Join(manifestDir, "manifest.json"))
	var manifest map[string]interface{}
	json.Unmarshal(data, &manifest)

	if manifest["start_url"] != "/" {
		t.Errorf("start_url = %v, expected / (preserved)", manifest["start_url"])
	}
	if manifest["display"] != "standalone" {
		t.Errorf("display = %v, expected standalone (preserved)", manifest["display"])
	}
	if manifest["name"] != "New Name" {
		t.Errorf("name = %v, expected New Name (overwritten by brand)", manifest["name"])
	}
	if manifest["_brand_display_name"] != "New Name" {
		t.Errorf("_brand_display_name = %v, expected New Name", manifest["_brand_display_name"])
	}
}

// contains is a simple test helper.
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && bytes.Contains([]byte(s), []byte(substr))
}
