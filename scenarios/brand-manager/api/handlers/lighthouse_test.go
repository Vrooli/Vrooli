package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"brand-manager/domain"
	"brand-manager/handlers"
)

// [REQ:BM-REQ-LIGHTHOUSE]

func TestLighthouseAudit_DryRun_ReturnsPending(t *testing.T) {
	cfg := testConfig(t)
	_, router, brandRepo, _, _ := setupMockServerWithConfig(t, cfg)

	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "TestBrand"})

	body, _ := json.Marshal(handlers.LighthouseRequest{
		ScenarioName: "test-scenario",
		URL:          "http://localhost:3000",
	})
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/lighthouse", bytes.NewReader(body))
	req.Header.Set("X-Dry-Run", "true")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var result handlers.LighthouseResult
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Scenario != "test-scenario" {
		t.Errorf("scenario = %q, want test-scenario", result.Scenario)
	}
	if result.BrandID != "b1" {
		t.Errorf("brand_id = %q, want b1", result.BrandID)
	}
	if result.URL != "http://localhost:3000" {
		t.Errorf("url = %q, want http://localhost:3000", result.URL)
	}
	if result.Status != "pending" {
		t.Errorf("status = %q, want pending", result.Status)
	}
	if result.Threshold != 90.0 {
		t.Errorf("threshold = %f, want 90.0", result.Threshold)
	}
}

func TestLighthouseAudit_NonDryRun_ReturnsAccepted(t *testing.T) {
	cfg := testConfig(t)
	_, router, brandRepo, _, _ := setupMockServerWithConfig(t, cfg)

	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "TestBrand"})

	body, _ := json.Marshal(handlers.LighthouseRequest{
		ScenarioName: "test-scenario",
	})
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/lighthouse", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	var result handlers.LighthouseResult
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Status != "pending" {
		t.Errorf("status = %q, want pending", result.Status)
	}
	if result.URL != "http://localhost" {
		t.Errorf("url = %q, want http://localhost (default)", result.URL)
	}
}

func TestLighthouseAudit_MissingScenarioName(t *testing.T) {
	cfg := testConfig(t)
	_, router, brandRepo, _, _ := setupMockServerWithConfig(t, cfg)

	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "TestBrand"})

	body, _ := json.Marshal(handlers.LighthouseRequest{})
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/lighthouse", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestLighthouseAudit_BrandNotFound(t *testing.T) {
	cfg := testConfig(t)
	_, router, _, _, _ := setupMockServerWithConfig(t, cfg)

	body, _ := json.Marshal(handlers.LighthouseRequest{ScenarioName: "test"})
	req := httptest.NewRequest("POST", "/api/v1/brands/missing/lighthouse", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestLighthouseAudit_InvalidBody(t *testing.T) {
	cfg := testConfig(t)
	_, router, brandRepo, _, _ := setupMockServerWithConfig(t, cfg)

	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "TestBrand"})

	req := httptest.NewRequest("POST", "/api/v1/brands/b1/lighthouse", bytes.NewReader([]byte("bad")))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestLighthouseAudit_CustomURL(t *testing.T) {
	cfg := testConfig(t)
	_, router, brandRepo, _, _ := setupMockServerWithConfig(t, cfg)

	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "TestBrand"})

	body, _ := json.Marshal(handlers.LighthouseRequest{
		ScenarioName: "test-scenario",
		URL:          "https://custom.example.com",
	})
	req := httptest.NewRequest("POST", "/api/v1/brands/b1/lighthouse", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}

	var result handlers.LighthouseResult
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.URL != "https://custom.example.com" {
		t.Errorf("url = %q, want https://custom.example.com", result.URL)
	}
}

func TestClassifyBrandViolations(t *testing.T) {
	violations := []handlers.AccessViolation{
		{ID: "color-contrast", Impact: "serious", Description: "Low contrast"},
		{ID: "image-alt", Impact: "critical", Description: "Missing alt text"},
		{ID: "color-contrast-enhanced", Impact: "moderate", Description: "Enhanced contrast"},
		{ID: "label", Impact: "critical", Description: "Missing label"},
	}

	brandRelated, other := handlers.ClassifyBrandViolations(violations)
	if len(brandRelated) != 2 {
		t.Errorf("brand-related count = %d, want 2", len(brandRelated))
	}
	if len(other) != 2 {
		t.Errorf("other count = %d, want 2", len(other))
	}

	if brandRelated[0].ID != "color-contrast" {
		t.Errorf("first brand-related = %q, want color-contrast", brandRelated[0].ID)
	}
	if !brandRelated[0].BrandCaused {
		t.Error("expected brand_caused = true for color-contrast")
	}
	if other[0].BrandCaused {
		t.Error("expected brand_caused = false for image-alt")
	}
}

func TestClassifyBrandViolations_Empty(t *testing.T) {
	brandRelated, other := handlers.ClassifyBrandViolations(nil)
	if brandRelated != nil {
		t.Errorf("expected nil brand-related, got %v", brandRelated)
	}
	if other != nil {
		t.Errorf("expected nil other, got %v", other)
	}
}
