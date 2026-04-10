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
	"brand-manager/handlers"
)

// [REQ:BM-REQ-AGENT-SPAWN] [REQ:BM-REQ-AGENT-INSTRUCT] [REQ:BM-REQ-AGENT-VALIDATE]

func TestAgentApply_DryRun_ReturnsInstructions(t *testing.T) {
	cfg := testConfig(t)
	_, router, brandRepo, _, _ := setupMockServerWithConfig(t, cfg)

	brandRepo.Seed(&domain.Brand{
		ID:      "b1",
		Name:    "TestBrand",
		Version: 3,
		Colors: &domain.Colors{
			Primary:    "#ff0000",
			Secondary:  "#00ff00",
			Background: "#ffffff",
		},
		Typography: &domain.Typography{
			HeadingFont: "Inter",
			BodyFont:    "Roboto",
		},
	})

	body, _ := json.Marshal(handlers.AgentApplyRequest{
		ScenarioName: "test-scenario",
	})
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/agent-apply", bytes.NewReader(body))
	req.Header.Set("X-Dry-Run", "true")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result handlers.AgentApplyResult
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Scenario != "test-scenario" {
		t.Errorf("scenario = %q, want %q", result.Scenario, "test-scenario")
	}
	if result.BrandID != "b1" {
		t.Errorf("brand_id = %q, want %q", result.BrandID, "b1")
	}
	if result.BrandVersion != 3 {
		t.Errorf("brand_version = %d, want 3", result.BrandVersion)
	}
	if !result.DryRun {
		t.Error("expected dry_run = true")
	}
	if len(result.Elements) != 5 {
		t.Errorf("elements count = %d, want 5", len(result.Elements))
	}
	// Verify instructions mandate markers
	for _, marker := range []string{"brand-manager:primary", "MANDATORY", "TestBrand"} {
		if !containsStr(result.Instructions, marker) {
			t.Errorf("instructions missing %q", marker)
		}
	}
}

func TestAgentApply_PartialElements(t *testing.T) {
	cfg := testConfig(t)
	_, router, brandRepo, _, _ := setupMockServerWithConfig(t, cfg)

	brandRepo.Seed(&domain.Brand{
		ID:   "b1",
		Name: "TestBrand",
		Colors: &domain.Colors{
			Primary: "#ff0000",
		},
	})

	body, _ := json.Marshal(handlers.AgentApplyRequest{
		ScenarioName: "test-scenario",
		Elements:     []string{"colors"},
		Prompt:       "Focus on dark mode support",
	})
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/agent-apply", bytes.NewReader(body))
	req.Header.Set("X-Dry-Run", "true")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result handlers.AgentApplyResult
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result.Elements) != 1 || result.Elements[0] != "colors" {
		t.Errorf("elements = %v, want [colors]", result.Elements)
	}
	if !containsStr(result.Instructions, "Focus on dark mode support") {
		t.Error("instructions missing custom prompt")
	}
}

func TestAgentApply_InvalidElement(t *testing.T) {
	cfg := testConfig(t)
	_, router, brandRepo, _, _ := setupMockServerWithConfig(t, cfg)

	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "TestBrand"})

	body, _ := json.Marshal(handlers.AgentApplyRequest{
		ScenarioName: "test-scenario",
		Elements:     []string{"nonexistent"},
	})
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/agent-apply", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAgentApply_MissingScenarioName(t *testing.T) {
	cfg := testConfig(t)
	_, router, brandRepo, _, _ := setupMockServerWithConfig(t, cfg)

	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "TestBrand"})

	body, _ := json.Marshal(handlers.AgentApplyRequest{})
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/agent-apply", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAgentApply_BrandNotFound(t *testing.T) {
	cfg := testConfig(t)
	_, router, _, _, _ := setupMockServerWithConfig(t, cfg)

	body, _ := json.Marshal(handlers.AgentApplyRequest{ScenarioName: "test"})
	req := httptest.NewRequest("POST", "/api/v1/brands/missing/agent-apply", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestAgentApply_NonDryRun_ReturnsPending(t *testing.T) {
	cfg := testConfig(t)
	_, router, brandRepo, _, _ := setupMockServerWithConfig(t, cfg)

	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "TestBrand"})

	body, _ := json.Marshal(handlers.AgentApplyRequest{
		ScenarioName: "test-scenario",
	})
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/agent-apply", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	var result handlers.AgentApplyResult
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Status != "pending" {
		t.Errorf("status = %q, want pending", result.Status)
	}
	if result.DryRun {
		t.Error("expected dry_run = false")
	}
}

func TestAgentApply_InvalidBody(t *testing.T) {
	cfg := testConfig(t)
	_, router, brandRepo, _, _ := setupMockServerWithConfig(t, cfg)

	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "TestBrand"})

	req := httptest.NewRequest("POST", "/api/v1/brands/b1/agent-apply", bytes.NewReader([]byte("{")))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAgentValidate_AllMarkersPresent(t *testing.T) {
	cfg := testConfig(t)
	scenarioDir := filepath.Join(cfg.ScenariosDir, "test-scenario")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cssContent := `:root {
  --color-primary: #ff0000; /* brand-manager:primary */
  --color-secondary: #00ff00; /* brand-manager:secondary */
  --color-accent: #0000ff; /* brand-manager:accent */
  --color-bg: #ffffff; /* brand-manager:background */
  --color-surface: #f5f5f5; /* brand-manager:surface */
  --color-text: #333333; /* brand-manager:text */
}
`
	if err := os.WriteFile(filepath.Join(scenarioDir, "theme.css"), []byte(cssContent), 0o644); err != nil {
		t.Fatal(err)
	}

	_, router, brandRepo, _, _ := setupMockServerWithConfig(t, cfg)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "TestBrand"})

	body, _ := json.Marshal(handlers.AgentValidateRequest{
		ScenarioName: "test-scenario",
		Elements:     []string{"colors"},
	})
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/agent-validate", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result handlers.AgentValidateResult
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !result.Valid {
		t.Errorf("expected valid=true, missing=%v", result.Missing)
	}
	if len(result.Found) != 6 {
		t.Errorf("found count = %d, want 6", len(result.Found))
	}
	if len(result.Missing) != 0 {
		t.Errorf("missing = %v, want empty", result.Missing)
	}
}

func TestAgentValidate_MissingMarkers(t *testing.T) {
	cfg := testConfig(t)
	scenarioDir := filepath.Join(cfg.ScenariosDir, "test-scenario")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cssContent := `:root {
  --color-primary: #ff0000; /* brand-manager:primary */
}
`
	if err := os.WriteFile(filepath.Join(scenarioDir, "theme.css"), []byte(cssContent), 0o644); err != nil {
		t.Fatal(err)
	}

	_, router, brandRepo, _, _ := setupMockServerWithConfig(t, cfg)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "TestBrand"})

	body, _ := json.Marshal(handlers.AgentValidateRequest{
		ScenarioName: "test-scenario",
		Elements:     []string{"colors"},
	})
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/agent-validate", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result handlers.AgentValidateResult
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Valid {
		t.Error("expected valid=false")
	}
	if !sliceContains(result.Found, "primary") {
		t.Error("expected 'primary' in found")
	}
	for _, m := range []string{"secondary", "accent", "background", "surface", "text"} {
		if !sliceContains(result.Missing, m) {
			t.Errorf("expected %q in missing", m)
		}
	}
}

func TestAgentValidate_EmptyScenario(t *testing.T) {
	cfg := testConfig(t)
	scenarioDir := filepath.Join(cfg.ScenariosDir, "empty-scenario")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}

	_, router, brandRepo, _, _ := setupMockServerWithConfig(t, cfg)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "TestBrand"})

	body, _ := json.Marshal(handlers.AgentValidateRequest{
		ScenarioName: "empty-scenario",
		Elements:     []string{"colors"},
	})
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/agent-validate", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result handlers.AgentValidateResult
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Valid {
		t.Error("expected valid=false")
	}
	if len(result.Missing) != 6 {
		t.Errorf("missing count = %d, want 6", len(result.Missing))
	}
	if len(result.Found) != 0 {
		t.Errorf("found = %v, want empty", result.Found)
	}
}

func TestAgentValidate_MissingScenarioName(t *testing.T) {
	cfg := testConfig(t)
	_, router, brandRepo, _, _ := setupMockServerWithConfig(t, cfg)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "TestBrand"})

	body, _ := json.Marshal(handlers.AgentValidateRequest{})
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/agent-validate", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAgentValidate_ScenarioNotFound(t *testing.T) {
	cfg := testConfig(t)
	_, router, brandRepo, _, _ := setupMockServerWithConfig(t, cfg)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "TestBrand"})

	body, _ := json.Marshal(handlers.AgentValidateRequest{
		ScenarioName: "nonexistent",
	})
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/agent-validate", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestAgentValidate_InvalidBody(t *testing.T) {
	cfg := testConfig(t)
	_, router, brandRepo, _, _ := setupMockServerWithConfig(t, cfg)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "TestBrand"})

	req := httptest.NewRequest("POST", "/api/v1/brands/b1/agent-validate", bytes.NewReader([]byte("not json")))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAgentValidate_AllElementsDefault(t *testing.T) {
	cfg := testConfig(t)
	scenarioDir := filepath.Join(cfg.ScenariosDir, "full-scenario")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "empty.css"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	_, router, brandRepo, _, _ := setupMockServerWithConfig(t, cfg)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "TestBrand"})

	body, _ := json.Marshal(handlers.AgentValidateRequest{
		ScenarioName: "full-scenario",
		// Elements omitted = all
	})
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/agent-validate", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result handlers.AgentValidateResult
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Valid {
		t.Error("expected valid=false (empty scenario)")
	}
	// All markers should be missing: 6 colors + 4 typography + 2 identity + 1 favicon + 1 logo = 14
	if len(result.Expected) < 10 {
		t.Errorf("expected at least 10 markers, got %d", len(result.Expected))
	}
}

func TestBuildAgentInstructions_IncludesAllColorMarkers(t *testing.T) {
	brand := &domain.Brand{
		Name: "TestBrand",
		Colors: &domain.Colors{
			Primary:    "#ff0000",
			Secondary:  "#00ff00",
			Accent:     "#0000ff",
			Background: "#ffffff",
			Surface:    "#f5f5f5",
			Text:       "#333333",
		},
	}

	instructions := handlers.BuildAgentInstructions(brand, []string{"colors"}, "")
	for _, marker := range []string{"brand-manager:primary", "brand-manager:secondary",
		"brand-manager:accent", "brand-manager:background", "brand-manager:surface",
		"brand-manager:text", "MANDATORY"} {
		if !containsStr(instructions, marker) {
			t.Errorf("instructions missing %q", marker)
		}
	}
}

func TestBuildAgentInstructions_IncludesTypography(t *testing.T) {
	brand := &domain.Brand{
		Name: "TestBrand",
		Typography: &domain.Typography{
			HeadingFont: "Inter",
			BodyFont:    "Roboto",
		},
	}

	instructions := handlers.BuildAgentInstructions(brand, []string{"typography"}, "custom note")
	for _, marker := range []string{"brand-manager:heading-font", "brand-manager:body-font", "custom note"} {
		if !containsStr(instructions, marker) {
			t.Errorf("instructions missing %q", marker)
		}
	}
}

func TestExpectedMarkers_Colors(t *testing.T) {
	markers := handlers.ExpectedMarkers([]string{"colors"})
	if len(markers) != 6 {
		t.Errorf("got %d markers, want 6", len(markers))
	}
	for _, m := range []string{"primary", "secondary", "accent", "background", "surface", "text"} {
		if !sliceContains(markers, m) {
			t.Errorf("missing marker %q", m)
		}
	}
}

func TestExpectedMarkers_Multiple(t *testing.T) {
	markers := handlers.ExpectedMarkers([]string{"colors", "typography"})
	if len(markers) != 10 {
		t.Errorf("got %d markers, want 10", len(markers))
	}
}

func TestExpectedMarkers_NoDuplicates(t *testing.T) {
	markers := handlers.ExpectedMarkers([]string{"colors", "colors"})
	if len(markers) != 6 {
		t.Errorf("got %d markers, want 6 (no duplicates)", len(markers))
	}
}

func TestExpectedMarkers_UnknownElement(t *testing.T) {
	markers := handlers.ExpectedMarkers([]string{"unknown"})
	if len(markers) != 0 {
		t.Errorf("got %d markers, want 0 for unknown element", len(markers))
	}
}

// containsStr checks if s contains substr.
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && findSubstring(s, substr))
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// sliceContains checks if a slice contains a value.
func sliceContains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
