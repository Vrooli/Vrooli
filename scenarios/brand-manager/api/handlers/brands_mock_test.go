package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"brand-manager/apierr"
	"brand-manager/domain"
	"brand-manager/handlers"
	"brand-manager/repository/mocks"

	"github.com/gorilla/mux"
)

// setupMockServer creates a handler stack backed by in-memory mocks.
// The deterministic IDFunc produces "id-1", "id-2", … for reproducible tests.
func setupMockServer(t *testing.T) (*handlers.Handlers, *mux.Router, *mocks.BrandRepository, *mocks.VersionRepository, *mocks.AssignmentRepository) {
	t.Helper()
	brandRepo := mocks.NewBrandRepository()
	versionRepo := mocks.NewVersionRepository()
	assignRepo := mocks.NewAssignmentRepository()
	assetRepo := mocks.NewAssetRepository()

	var counter atomic.Int64
	h := handlers.New(brandRepo, versionRepo, assignRepo).
		WithAssets(assetRepo).
		WithIDFunc(func() string {
			return fmt.Sprintf("id-%d", counter.Add(1))
		})

	router := mux.NewRouter()
	h.RegisterRoutes(router)
	return h, router, brandRepo, versionRepo, assignRepo
}

// TestMockCreateBrand tests handler create logic without a database.
// [REQ:BM-REQ-CRUD-CREATE] [REQ:BM-REQ-API-BRANDS]
func TestMockCreateBrand(t *testing.T) {
	_, router, _, _, _ := setupMockServer(t)

	body := `{"name":"Mock Brand","description":"via mock"}`
	req := httptest.NewRequest("POST", "/api/v1/brands", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var brand domain.Brand
	json.NewDecoder(w.Body).Decode(&brand)

	if brand.ID != "id-1" {
		t.Errorf("ID = %q, want %q (deterministic)", brand.ID, "id-1")
	}
	if brand.Name != "Mock Brand" {
		t.Errorf("Name = %q, want %q", brand.Name, "Mock Brand")
	}
}

// TestMockCreateBrandRepoError tests handler behaviour when repo returns an error.
// [REQ:BM-REQ-CRUD-CREATE]
func TestMockCreateBrandRepoError(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.CreateErr = errors.New("disk full")

	body := `{"name":"Fail Brand"}`
	req := httptest.NewRequest("POST", "/api/v1/brands", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// TestMockGetBrandNotFound tests 404 via mock without DB.
// [REQ:BM-REQ-CRUD-READ]
func TestMockGetBrandNotFound(t *testing.T) {
	_, router, _, _, _ := setupMockServer(t)

	req := httptest.NewRequest("GET", "/api/v1/brands/missing", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// TestMockUpdateBrand tests update with version increment via mock.
// [REQ:BM-REQ-CRUD-UPDATE]
func TestMockUpdateBrand(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)

	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Original", Version: 1})

	body := `{"name":"Updated"}`
	req := httptest.NewRequest("PUT", "/api/v1/brands/b1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var updated domain.Brand
	json.NewDecoder(w.Body).Decode(&updated)
	if updated.Name != "Updated" {
		t.Errorf("Name = %q, want %q", updated.Name, "Updated")
	}
	if updated.Version != 2 {
		t.Errorf("Version = %d, want 2", updated.Version)
	}
}

// TestMockDeleteBrand tests delete via mock.
// [REQ:BM-REQ-API-BRANDS]
func TestMockDeleteBrand(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "ToDelete", Version: 1})

	req := httptest.NewRequest("DELETE", "/api/v1/brands/b1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

// TestMockScenarioStatusNoAssignment tests status without assignment via mock.
// [REQ:BM-REQ-API-STATUS]
func TestMockScenarioStatusNoAssignment(t *testing.T) {
	_, router, _, _, _ := setupMockServer(t)

	req := httptest.NewRequest("GET", "/api/v1/scenarios/test-scenario/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var status map[string]interface{}
	json.NewDecoder(w.Body).Decode(&status)
	if status["has_brand"] != false {
		t.Errorf("has_brand = %v, want false", status["has_brand"])
	}
}

// TestMockCreateAssignment tests assignment creation via mock.
// [REQ:BM-REQ-ASSIGN-LINK] [REQ:BM-REQ-API-ASSIGN]
func TestMockCreateAssignment(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Brand", Version: 3})

	body := `{"brand_id":"b1","scenario_name":"my-scenario","elements":["logo"]}`
	req := httptest.NewRequest("POST", "/api/v1/assignments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var a domain.Assignment
	json.NewDecoder(w.Body).Decode(&a)
	if a.BrandVersion != 3 {
		t.Errorf("BrandVersion = %d, want 3 (from seeded brand)", a.BrandVersion)
	}
}

// TestMockCreateAssignmentBrandNotFound tests 404 when brand missing.
// [REQ:BM-REQ-ASSIGN-LINK]
func TestMockCreateAssignmentBrandNotFound(t *testing.T) {
	_, router, _, _, _ := setupMockServer(t)

	body := `{"brand_id":"missing","scenario_name":"x"}`
	req := httptest.NewRequest("POST", "/api/v1/assignments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// TestDryRunCreateBrand verifies POST with X-Dry-Run returns realistic data without persisting.
// [REQ:BM-REQ-API-BRANDS]
func TestDryRunCreateBrand(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)

	body := `{"name":"Dry Brand"}`
	req := httptest.NewRequest("POST", "/api/v1/brands", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Dry-Run", "true")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	if result["dry_run"] != true {
		t.Error("expected dry_run = true")
	}
	if result["name"] != "Dry Brand" {
		t.Errorf("name = %v, want Dry Brand", result["name"])
	}

	// Verify nothing was persisted
	brands, _ := brandRepo.List(nil, domain.BrandFilter{})
	if len(brands) != 0 {
		t.Errorf("expected 0 brands after dry-run, got %d", len(brands))
	}
}

// TestDryRunUpdateBrand verifies PUT with X-Dry-Run returns merged data without persisting.
// [REQ:BM-REQ-CRUD-UPDATE]
func TestDryRunUpdateBrand(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Original", Version: 3})

	body := `{"name":"DryUpdated"}`
	req := httptest.NewRequest("PUT", "/api/v1/brands/b1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Dry-Run", "true")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	if result["dry_run"] != true {
		t.Error("expected dry_run = true")
	}
	if result["name"] != "DryUpdated" {
		t.Errorf("name = %v, want DryUpdated", result["name"])
	}
	// Version should be incremented in dry-run preview
	if v, ok := result["version"].(float64); !ok || v != 4 {
		t.Errorf("version = %v, want 4", result["version"])
	}

	// Verify original was not mutated
	brand, _ := brandRepo.GetByID(nil, "b1")
	if brand.Name != "Original" {
		t.Errorf("brand name = %q, want Original (not mutated)", brand.Name)
	}
}

// TestDryRunDeleteBrand verifies DELETE with X-Dry-Run validates but doesn't delete.
// [REQ:BM-REQ-API-BRANDS]
func TestDryRunDeleteBrand(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Keep", Version: 1})

	req := httptest.NewRequest("DELETE", "/api/v1/brands/b1", nil)
	req.Header.Set("X-Dry-Run", "true")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	if result["dry_run"] != true {
		t.Error("expected dry_run = true")
	}

	// Verify brand still exists
	brand, _ := brandRepo.GetByID(nil, "b1")
	if brand == nil {
		t.Error("brand was deleted during dry-run")
	}
}

// TestDryRunDeleteBrandNotFound verifies dry-run still validates existence.
// [REQ:BM-REQ-API-BRANDS]
func TestDryRunDeleteBrandNotFound(t *testing.T) {
	_, router, _, _, _ := setupMockServer(t)

	req := httptest.NewRequest("DELETE", "/api/v1/brands/missing", nil)
	req.Header.Set("X-Dry-Run", "true")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// TestDryRunCreateAssignment verifies assignment dry-run.
// [REQ:BM-REQ-ASSIGN-LINK]
func TestDryRunCreateAssignment(t *testing.T) {
	_, router, brandRepo, _, assignRepo := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Brand", Version: 5})

	body := `{"brand_id":"b1","scenario_name":"my-scenario"}`
	req := httptest.NewRequest("POST", "/api/v1/assignments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Dry-Run", "true")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	if result["dry_run"] != true {
		t.Error("expected dry_run = true")
	}

	// Verify nothing was persisted
	assignments, _ := assignRepo.ListByBrandID(nil, "b1")
	if len(assignments) != 0 {
		t.Errorf("expected 0 assignments after dry-run, got %d", len(assignments))
	}
}

// TestStructuredErrorResponse verifies 4xx/5xx errors return the structured
// {code, message, recovery} JSON shape for programmatic client handling.
func TestStructuredErrorResponse(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		setup      func(*mocks.BrandRepository)
		wantStatus int
		wantCode   apierr.Code
	}{
		{
			name:       "validation error (missing name)",
			method:     "POST",
			path:       "/api/v1/brands",
			body:       `{"description":"no name"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   apierr.CodeValidation,
		},
		{
			name:       "not found error",
			method:     "GET",
			path:       "/api/v1/brands/nonexistent",
			wantStatus: http.StatusNotFound,
			wantCode:   apierr.CodeNotFound,
		},
		{
			name:   "internal error (repo failure)",
			method: "POST",
			path:   "/api/v1/brands",
			body:   `{"name":"Fail"}`,
			setup: func(r *mocks.BrandRepository) {
				r.CreateErr = errors.New("disk full")
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   apierr.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, router, brandRepo, _, _ := setupMockServer(t)
			if tt.setup != nil {
				tt.setup(brandRepo)
			}

			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			var errResp apierr.Error
			if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
				t.Fatalf("failed to decode error JSON: %v", err)
			}
			if errResp.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", errResp.Code, tt.wantCode)
			}
			if errResp.Message == "" {
				t.Error("expected non-empty message")
			}
			if errResp.Recovery == "" {
				t.Error("expected non-empty recovery hint")
			}
		})
	}
}
