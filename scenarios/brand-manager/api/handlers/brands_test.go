package handlers_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"brand-manager/domain"
	"brand-manager/handlers"
	"brand-manager/internal/testutil"
	"brand-manager/repository"

	"github.com/gorilla/mux"
)

func setupTestServer(t *testing.T) (*handlers.Handlers, *mux.Router, *sql.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)

	brandRepo := repository.NewSQLiteBrandRepository(db)
	versionRepo := repository.NewSQLiteVersionRepository(db)
	assignRepo := repository.NewSQLiteAssignmentRepository(db)
	h := handlers.New(brandRepo, versionRepo, assignRepo)

	router := mux.NewRouter()
	h.RegisterRoutes(router)
	return h, router, db
}

// TestCreateBrandEndpoint verifies POST /api/v1/brands.
// [REQ:BM-REQ-CRUD-CREATE] [REQ:BM-REQ-API-BRANDS]
func TestCreateBrandEndpoint(t *testing.T) {
	_, router, _ := setupTestServer(t)

	body := `{"name":"Test Brand","description":"A test","colors":{"primary":"#ff0000"}}`
	req := httptest.NewRequest("POST", "/api/v1/brands", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var brand domain.Brand
	json.NewDecoder(w.Body).Decode(&brand)

	if brand.ID == "" {
		t.Error("expected non-empty ID")
	}
	if brand.Name != "Test Brand" {
		t.Errorf("name = %q, want %q", brand.Name, "Test Brand")
	}
	if brand.Version != 1 {
		t.Errorf("version = %d, want 1", brand.Version)
	}
}

// TestCreateBrandValidation verifies name is required.
// [REQ:BM-REQ-CRUD-CREATE]
func TestCreateBrandValidation(t *testing.T) {
	_, router, _ := setupTestServer(t)

	body := `{"description":"no name"}`
	req := httptest.NewRequest("POST", "/api/v1/brands", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestGetBrandEndpoint verifies GET /api/v1/brands/{id}.
// [REQ:BM-REQ-CRUD-READ] [REQ:BM-REQ-API-BRANDS]
func TestGetBrandEndpoint(t *testing.T) {
	_, router, _ := setupTestServer(t)

	// Create first
	body := `{"name":"Get Test"}`
	req := httptest.NewRequest("POST", "/api/v1/brands", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var created domain.Brand
	json.NewDecoder(w.Body).Decode(&created)

	// Get it
	req = httptest.NewRequest("GET", "/api/v1/brands/"+created.ID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got domain.Brand
	json.NewDecoder(w.Body).Decode(&got)
	if got.Name != "Get Test" {
		t.Errorf("name = %q, want %q", got.Name, "Get Test")
	}
}

// TestGetBrandNotFound verifies 404 for missing brand.
// [REQ:BM-REQ-API-BRANDS]
func TestGetBrandNotFound(t *testing.T) {
	_, router, _ := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/brands/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// TestListBrandsEndpoint verifies GET /api/v1/brands.
// [REQ:BM-REQ-CRUD-READ] [REQ:BM-REQ-API-BRANDS]
func TestListBrandsEndpoint(t *testing.T) {
	_, router, _ := setupTestServer(t)

	// Create two brands
	for _, name := range []string{"Brand A", "Brand B"} {
		body, _ := json.Marshal(map[string]string{"name": name})
		req := httptest.NewRequest("POST", "/api/v1/brands", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	// List all
	req := httptest.NewRequest("GET", "/api/v1/brands", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var brands []domain.Brand
	json.NewDecoder(w.Body).Decode(&brands)
	if len(brands) != 2 {
		t.Errorf("got %d brands, want 2", len(brands))
	}
}

// TestUpdateBrandEndpoint verifies PUT /api/v1/brands/{id}.
// [REQ:BM-REQ-CRUD-UPDATE] [REQ:BM-REQ-API-BRANDS]
func TestUpdateBrandEndpoint(t *testing.T) {
	_, router, _ := setupTestServer(t)

	// Create
	body := `{"name":"Original"}`
	req := httptest.NewRequest("POST", "/api/v1/brands", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var created domain.Brand
	json.NewDecoder(w.Body).Decode(&created)

	// Update
	updateBody := `{"name":"Updated","colors":{"primary":"#00ff00"}}`
	req = httptest.NewRequest("PUT", "/api/v1/brands/"+created.ID, bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var updated domain.Brand
	json.NewDecoder(w.Body).Decode(&updated)
	if updated.Name != "Updated" {
		t.Errorf("name = %q, want %q", updated.Name, "Updated")
	}
	if updated.Version != 2 {
		t.Errorf("version = %d, want 2", updated.Version)
	}
}

// TestDeleteBrandEndpoint verifies DELETE /api/v1/brands/{id}.
// [REQ:BM-REQ-API-BRANDS]
func TestDeleteBrandEndpoint(t *testing.T) {
	_, router, _ := setupTestServer(t)

	// Create
	body := `{"name":"ToDelete"}`
	req := httptest.NewRequest("POST", "/api/v1/brands", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var created domain.Brand
	json.NewDecoder(w.Body).Decode(&created)

	// Delete
	req = httptest.NewRequest("DELETE", "/api/v1/brands/"+created.ID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}

	// Verify gone via GET
	req = httptest.NewRequest("GET", "/api/v1/brands/"+created.ID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("after delete: GET status = %d, want %d", w.Code, http.StatusNotFound)
	}

	// Verify delete is idempotent — second delete returns 204, not 404
	req = httptest.NewRequest("DELETE", "/api/v1/brands/"+created.ID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("idempotent delete: status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

// TestVersionCreatedOnBrandCreate verifies version snapshot is created.
// [REQ:BM-REQ-CRUD-VERSION] [REQ:BM-REQ-API-VERSIONS]
func TestVersionCreatedOnBrandCreate(t *testing.T) {
	_, router, _ := setupTestServer(t)

	// Create brand
	body := `{"name":"Versioned Brand"}`
	req := httptest.NewRequest("POST", "/api/v1/brands", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var created domain.Brand
	json.NewDecoder(w.Body).Decode(&created)

	// List versions
	req = httptest.NewRequest("GET", "/api/v1/brands/"+created.ID+"/versions", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var versions []domain.BrandVersion
	json.NewDecoder(w.Body).Decode(&versions)
	if len(versions) != 1 {
		t.Errorf("got %d versions, want 1", len(versions))
	}
}

// TestScenarioStatusEndpoint verifies GET /api/v1/scenarios/{name}/status.
// [REQ:BM-REQ-API-STATUS]
func TestScenarioStatusEndpoint(t *testing.T) {
	_, router, _ := setupTestServer(t)

	// No assignment — should report no brand
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
