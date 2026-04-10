package handlers_test

// Tests for temporal flows, idempotency, replay safety, and optimistic locking.
// These tests verify that the brand-manager API behaves predictably under
// retries, concurrent modifications, and repeated operations.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"brand-manager/domain"
)

// --- Optimistic Locking (If-Match / ETag) ---

// TestETagReturnedOnCreate verifies that brand creation returns an ETag header.
// [REQ:BM-REQ-CRUD-CREATE] [REQ:BM-REQ-API-BRANDS]
func TestETagReturnedOnCreate(t *testing.T) {
	_, router, _, _, _ := setupMockServer(t)

	body := `{"name":"ETag Brand"}`
	req := httptest.NewRequest("POST", "/api/v1/brands", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	etag := w.Header().Get("ETag")
	if etag != "1" {
		t.Errorf("ETag = %q, want %q (version 1)", etag, "1")
	}
}

// TestETagReturnedOnGet verifies that GET /brands/{id} returns an ETag header.
// [REQ:BM-REQ-CRUD-READ]
func TestETagReturnedOnGet(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Brand", Version: 3})

	req := httptest.NewRequest("GET", "/api/v1/brands/b1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	etag := w.Header().Get("ETag")
	if etag != "3" {
		t.Errorf("ETag = %q, want %q", etag, "3")
	}
}

// TestOptimisticLockingAcceptsMatchingVersion verifies that If-Match with the
// current version allows the update.
// [REQ:BM-REQ-CRUD-UPDATE]
func TestOptimisticLockingAcceptsMatchingVersion(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Original", Version: 1})

	body := `{"name":"Updated"}`
	req := httptest.NewRequest("PUT", "/api/v1/brands/b1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", "1")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var brand domain.Brand
	json.NewDecoder(w.Body).Decode(&brand)
	if brand.Name != "Updated" {
		t.Errorf("Name = %q, want %q", brand.Name, "Updated")
	}
	if brand.Version != 2 {
		t.Errorf("Version = %d, want 2", brand.Version)
	}

	// New ETag should reflect updated version
	etag := w.Header().Get("ETag")
	if etag != "2" {
		t.Errorf("ETag after update = %q, want %q", etag, "2")
	}
}

// TestOptimisticLockingRejectsStaleVersion verifies that If-Match with an
// outdated version returns 409 Conflict.
// [REQ:BM-REQ-CRUD-UPDATE]
func TestOptimisticLockingRejectsStaleVersion(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Current", Version: 5})

	body := `{"name":"Stale Update"}`
	req := httptest.NewRequest("PUT", "/api/v1/brands/b1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", "3") // Stale: current is 5
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, http.StatusConflict, w.Body.String())
	}

	// Verify brand was NOT modified
	brand, _ := brandRepo.GetByID(nil, "b1")
	if brand.Name != "Current" {
		t.Errorf("Name = %q, want %q (unchanged)", brand.Name, "Current")
	}
	if brand.Version != 5 {
		t.Errorf("Version = %d, want 5 (unchanged)", brand.Version)
	}
}

// TestOptimisticLockingInvalidHeader verifies If-Match rejects non-integer values.
// [REQ:BM-REQ-CRUD-UPDATE]
func TestOptimisticLockingInvalidHeader(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Brand", Version: 1})

	body := `{"name":"Bad Update"}`
	req := httptest.NewRequest("PUT", "/api/v1/brands/b1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", "not-a-number")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestUpdateWithoutIfMatchStillWorks verifies that omitting If-Match preserves
// backwards compatibility (no locking enforced).
// [REQ:BM-REQ-CRUD-UPDATE]
func TestUpdateWithoutIfMatchStillWorks(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Original", Version: 3})

	body := `{"name":"NoLock"}`
	req := httptest.NewRequest("PUT", "/api/v1/brands/b1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	// No If-Match header
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// --- Idempotency Key ---

// TestIdempotencyKeyPreventsDoubleCreate verifies that sending the same
// Idempotency-Key twice returns the cached first response without creating
// a duplicate brand.
// [REQ:BM-REQ-CRUD-CREATE]
func TestIdempotencyKeyPreventsDoubleCreate(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)

	body := `{"name":"Idem Brand"}`

	// First request
	req := httptest.NewRequest("POST", "/api/v1/brands", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "create-abc-123")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("first request: status = %d, want %d", w.Code, http.StatusCreated)
	}

	var first domain.Brand
	json.NewDecoder(w.Body).Decode(&first)
	if first.Name != "Idem Brand" {
		t.Fatalf("first request: name = %q, want %q", first.Name, "Idem Brand")
	}

	// Second request with same key — should return cached response
	req = httptest.NewRequest("POST", "/api/v1/brands", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "create-abc-123")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("replay: status = %d, want %d", w.Code, http.StatusCreated)
	}

	// Should indicate this was a replay
	if w.Header().Get("X-Idempotent-Replayed") != "true" {
		t.Error("replay: expected X-Idempotent-Replayed = true")
	}

	var second domain.Brand
	json.NewDecoder(w.Body).Decode(&second)
	if second.ID != first.ID {
		t.Errorf("replay: ID = %q, want %q (same as first)", second.ID, first.ID)
	}

	// Verify only one brand was created
	brands, _ := brandRepo.List(nil, domain.BrandFilter{})
	if len(brands) != 1 {
		t.Errorf("expected 1 brand, got %d (duplicate created!)", len(brands))
	}
}

// TestIdempotencyKeyDifferentKeysCreateSeparateBrands verifies that different
// idempotency keys produce different brands.
// [REQ:BM-REQ-CRUD-CREATE]
func TestIdempotencyKeyDifferentKeysCreateSeparateBrands(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)

	for _, key := range []string{"key-1", "key-2"} {
		body := `{"name":"Brand ` + key + `"}`
		req := httptest.NewRequest("POST", "/api/v1/brands", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", key)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("key %s: status = %d, want %d", key, w.Code, http.StatusCreated)
		}
	}

	brands, _ := brandRepo.List(nil, domain.BrandFilter{})
	if len(brands) != 2 {
		t.Errorf("expected 2 brands (different keys), got %d", len(brands))
	}
}

// TestNoIdempotencyKeyAllowsDuplicates verifies that without an idempotency key,
// repeated POSTs create separate brands (backwards compatible).
// [REQ:BM-REQ-CRUD-CREATE]
func TestNoIdempotencyKeyAllowsDuplicates(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)

	body := `{"name":"Dup Brand"}`

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/api/v1/brands", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		// No Idempotency-Key
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("request %d: status = %d, want %d", i, w.Code, http.StatusCreated)
		}
	}

	brands, _ := brandRepo.List(nil, domain.BrandFilter{})
	if len(brands) != 3 {
		t.Errorf("expected 3 brands (no idempotency key), got %d", len(brands))
	}
}

// --- Delete Idempotency ---

// TestDeleteBrandIdempotent verifies that deleting a brand twice returns 204 both times.
// [REQ:BM-REQ-API-BRANDS]
func TestDeleteBrandIdempotent(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "ToDelete", Version: 1})

	// First delete
	req := httptest.NewRequest("DELETE", "/api/v1/brands/b1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("first delete: status = %d, want %d", w.Code, http.StatusNoContent)
	}

	// Second delete — should still be 204 (idempotent)
	req = httptest.NewRequest("DELETE", "/api/v1/brands/b1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("second delete: status = %d, want %d (idempotent)", w.Code, http.StatusNoContent)
	}
}

// TestDeleteNeverExistedBrand verifies that deleting a brand that never existed
// returns 204 (idempotent behavior).
// [REQ:BM-REQ-API-BRANDS]
func TestDeleteNeverExistedBrand(t *testing.T) {
	_, router, _, _, _ := setupMockServer(t)

	req := httptest.NewRequest("DELETE", "/api/v1/brands/phantom", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

// TestDeleteAssignmentIdempotentReplay verifies double-delete on assignments.
// [REQ:BM-REQ-API-ASSIGN]
func TestDeleteAssignmentIdempotentReplay(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Brand", Version: 1})

	// Create assignment
	body := `{"brand_id":"b1","scenario_name":"s1"}`
	req := httptest.NewRequest("POST", "/api/v1/assignments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var a domain.Assignment
	json.NewDecoder(w.Body).Decode(&a)

	// First delete
	req = httptest.NewRequest("DELETE", "/api/v1/assignments/"+a.ID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("first delete: status = %d, want %d", w.Code, http.StatusNoContent)
	}

	// Second delete — idempotent
	req = httptest.NewRequest("DELETE", "/api/v1/assignments/"+a.ID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("replay delete: status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

// --- Assignment Upsert Replay Safety ---

// TestAssignmentReassignSameScenarioIsIdempotent verifies that assigning
// the same brand to the same scenario twice produces the same result
// (INSERT OR REPLACE semantics).
// [REQ:BM-REQ-ASSIGN-LINK]
func TestAssignmentReassignSameScenarioIsIdempotent(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Brand", Version: 2})

	body := `{"brand_id":"b1","scenario_name":"my-app","elements":["logo"]}`

	// First assignment
	req := httptest.NewRequest("POST", "/api/v1/assignments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("first assign: status = %d", w.Code)
	}

	// Second assignment — same scenario, same brand
	req = httptest.NewRequest("POST", "/api/v1/assignments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("re-assign: status = %d", w.Code)
	}

	// Verify scenario status still works (no duplicates)
	req = httptest.NewRequest("GET", "/api/v1/scenarios/my-app/status", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var status domain.ScenarioStatus
	json.NewDecoder(w.Body).Decode(&status)
	if !status.HasBrand {
		t.Error("expected has_brand = true after re-assignment")
	}
	if status.BrandID == nil || *status.BrandID != "b1" {
		t.Errorf("brand_id = %v, want b1", status.BrandID)
	}
}

// --- Version Snapshot Temporal Ordering ---

// TestVersionSnapshotsOrderedAfterMultipleUpdates verifies that version
// snapshots maintain correct ordering after sequential updates.
// [REQ:BM-REQ-CRUD-VERSION] [REQ:BM-REQ-API-VERSIONS]
func TestVersionSnapshotsOrderedAfterMultipleUpdates(t *testing.T) {
	_, router, _ := setupTestServer(t)

	// Create brand
	body := `{"name":"Versioned"}`
	req := httptest.NewRequest("POST", "/api/v1/brands", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var brand domain.Brand
	json.NewDecoder(w.Body).Decode(&brand)

	// Update 3 times
	for i, name := range []string{"V2", "V3", "V4"} {
		updateBody := `{"name":"` + name + `"}`
		req = httptest.NewRequest("PUT", "/api/v1/brands/"+brand.ID, bytes.NewBufferString(updateBody))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("update %d: status = %d", i, w.Code)
		}
	}

	// List versions — should be 4 (1 create + 3 updates), newest first
	req = httptest.NewRequest("GET", "/api/v1/brands/"+brand.ID+"/versions", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var versions []domain.BrandVersion
	json.NewDecoder(w.Body).Decode(&versions)

	if len(versions) != 4 {
		t.Fatalf("got %d versions, want 4", len(versions))
	}

	// Verify descending order
	for i := 1; i < len(versions); i++ {
		if versions[i-1].Version <= versions[i].Version {
			t.Errorf("versions not in descending order: v%d before v%d",
				versions[i-1].Version, versions[i].Version)
		}
	}
}

// --- Dry-Run Does Not Mutate State ---

// TestDryRunSequenceDoesNotAffectRealOperations verifies that interleaving
// dry-run and real operations produces correct results.
// [REQ:BM-REQ-API-BRANDS]
func TestDryRunSequenceDoesNotAffectRealOperations(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Real", Version: 1})

	// Dry-run update
	body := `{"name":"DryName"}`
	req := httptest.NewRequest("PUT", "/api/v1/brands/b1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Dry-Run", "true")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("dry-run: status = %d", w.Code)
	}

	// Real read — should still be "Real"
	brand, _ := brandRepo.GetByID(nil, "b1")
	if brand.Name != "Real" {
		t.Errorf("after dry-run: name = %q, want %q", brand.Name, "Real")
	}
	if brand.Version != 1 {
		t.Errorf("after dry-run: version = %d, want 1", brand.Version)
	}

	// Real update should work with correct version
	body = `{"name":"RealUpdate"}`
	req = httptest.NewRequest("PUT", "/api/v1/brands/b1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", "1")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("real update: status = %d; body: %s", w.Code, w.Body.String())
	}
}

// --- Conflict Error Structure ---

// TestConflictErrorStructure verifies the 409 response carries the standard
// {code, message, recovery} shape for client classification.
func TestConflictErrorStructure(t *testing.T) {
	_, router, brandRepo, _, _ := setupMockServer(t)
	brandRepo.Seed(&domain.Brand{ID: "b1", Name: "Brand", Version: 10})

	body := `{"name":"Stale"}`
	req := httptest.NewRequest("PUT", "/api/v1/brands/b1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", "5") // stale
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}

	var errResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&errResp)

	if errResp["code"] != "conflict" {
		t.Errorf("code = %v, want conflict", errResp["code"])
	}
	if errResp["recovery"] == nil || errResp["recovery"] == "" {
		t.Error("expected non-empty recovery hint")
	}
}
