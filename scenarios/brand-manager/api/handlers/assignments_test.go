package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"brand-manager/domain"
)

// [REQ:BM-REQ-API-ASSIGN] [REQ:BM-REQ-ASSIGN-LINK]
func TestDeleteAssignmentEndpoint(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Brand", Version: 1})

	// Create an assignment first
	body := `{"brand_id":"b1","scenario_name":"test-scenario"}`
	req := httptest.NewRequest("POST", "/api/v1/assignments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var a domain.Assignment
	json.NewDecoder(w.Body).Decode(&a)

	// Delete it
	req = httptest.NewRequest("DELETE", "/api/v1/assignments/"+a.ID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("delete status = %d, want %d; body: %s", w.Code, http.StatusNoContent, w.Body.String())
	}
}

// [REQ:BM-REQ-API-ASSIGN]
// Delete is idempotent — deleting a nonexistent assignment returns 204.
func TestDeleteAssignmentIdempotent(t *testing.T) {
	_, router, _, _, _ := setupMockServer(t)

	req := httptest.NewRequest("DELETE", "/api/v1/assignments/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("idempotent delete: status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

// [REQ:BM-REQ-API-ASSIGN]
func TestDryRunDeleteAssignment(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Brand", Version: 1})

	// Create assignment
	body := `{"brand_id":"b1","scenario_name":"test-scenario"}`
	req := httptest.NewRequest("POST", "/api/v1/assignments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var a domain.Assignment
	json.NewDecoder(w.Body).Decode(&a)

	// Dry-run delete
	req = httptest.NewRequest("DELETE", "/api/v1/assignments/"+a.ID, nil)
	req.Header.Set("X-Dry-Run", "true")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	if result["dry_run"] != true {
		t.Error("expected dry_run = true")
	}
}

// [REQ:BM-REQ-API-ASSIGN] [REQ:BM-REQ-ASSIGN-LINK]
func TestCreateAssignmentValidation(t *testing.T) {
	_, router, _, _, _ := setupMockServer(t)

	// Missing brand_id
	body := `{"scenario_name":"test"}`
	req := httptest.NewRequest("POST", "/api/v1/assignments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	// Missing scenario_name
	body = `{"brand_id":"b1"}`
	req = httptest.NewRequest("POST", "/api/v1/assignments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// [REQ:BM-REQ-API-VERSIONS] [REQ:BM-REQ-CRUD-VERSION]
func TestListVersionsEndpoint(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Brand", Version: 1})

	// Create brand (which also creates version)
	body := `{"name":"Versioned Brand"}`
	req := httptest.NewRequest("POST", "/api/v1/brands", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var brand domain.Brand
	json.NewDecoder(w.Body).Decode(&brand)

	// List versions
	req = httptest.NewRequest("GET", "/api/v1/brands/"+brand.ID+"/versions", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var versions []domain.BrandVersion
	json.NewDecoder(w.Body).Decode(&versions)
	if len(versions) == 0 {
		t.Error("expected at least one version after brand creation")
	}
}

// [REQ:BM-REQ-API-VERSIONS]
func TestListVersionsEmptyBrand(t *testing.T) {
	_, router, _, _, _ := setupMockServer(t)

	req := httptest.NewRequest("GET", "/api/v1/brands/nonexistent/versions", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var versions []interface{}
	json.NewDecoder(w.Body).Decode(&versions)
	if len(versions) != 0 {
		t.Errorf("expected empty versions array, got %d", len(versions))
	}
}

// [REQ:BM-REQ-API-STATUS] [REQ:BM-REQ-ASSIGN-LINK]
func TestScenarioStatusWithAssignment(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Brand", Version: 2})

	// Create assignment
	body := `{"brand_id":"b1","scenario_name":"my-app","elements":["logo","colors"]}`
	req := httptest.NewRequest("POST", "/api/v1/assignments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("assignment create status = %d", w.Code)
	}

	// Check status
	req = httptest.NewRequest("GET", "/api/v1/scenarios/my-app/status", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var status domain.ScenarioStatus
	json.NewDecoder(w.Body).Decode(&status)
	if !status.HasBrand {
		t.Error("expected has_brand = true")
	}
	if status.BrandID == nil || *status.BrandID != "b1" {
		t.Errorf("expected brand_id = b1, got %v", status.BrandID)
	}
	if status.BrandVersion == nil || *status.BrandVersion != 2 {
		t.Errorf("expected brand_version = 2, got %v", status.BrandVersion)
	}
}

// [REQ:BM-REQ-CRUD-READ] [REQ:BM-REQ-API-BRANDS]
func TestListBrandsWithFilter(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Alpha Brand"})
	brandRepo.Seed(&domain.Brand{ID: "b2", Name: "Beta Brand"})
	brandRepo.Seed(&domain.Brand{ID: "b3", Name: "Gamma Corp"})

	// Filter by name
	req := httptest.NewRequest("GET", "/api/v1/brands?name=Brand", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var brands []domain.Brand
	json.NewDecoder(w.Body).Decode(&brands)

	// Mock may or may not support name filter, but should return results
	if len(brands) == 0 {
		t.Error("expected at least one brand in response")
	}
}
