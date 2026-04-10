package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- Structured Error Response Tests ---

// [REQ:P0-001] Test writeAPIError produces structured JSON with category, message, retryable
func TestWriteAPIError_Structure(t *testing.T) {
	w := httptest.NewRecorder()
	writeAPIError(w, http.StatusBadRequest, ErrCategoryValidation, "name is required", nil)

	assertStatus(t, w, http.StatusBadRequest)

	var resp APIError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode API error: %v", err)
	}
	if resp.Category != ErrCategoryValidation {
		t.Errorf("expected category=%s, got %s", ErrCategoryValidation, resp.Category)
	}
	if resp.Message != "name is required" {
		t.Errorf("expected message='name is required', got %s", resp.Message)
	}
	if resp.Retryable {
		t.Error("validation errors should not be retryable")
	}
}

// [REQ:P2-001] Test dependency errors are marked as retryable
func TestWriteAPIError_DependencyRetryable(t *testing.T) {
	w := httptest.NewRecorder()
	writeAPIError(w, http.StatusServiceUnavailable, ErrCategoryDependency, "LLM unavailable", nil)

	var resp APIError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if !resp.Retryable {
		t.Error("dependency errors should be retryable")
	}
	if resp.Category != ErrCategoryDependency {
		t.Errorf("expected category=%s, got %s", ErrCategoryDependency, resp.Category)
	}
}

// [REQ:P0-001] Test classifyAndWriteError maps sql.ErrNoRows to 404
func TestClassifyAndWriteError_NotFound(t *testing.T) {
	w := httptest.NewRecorder()
	classifyAndWriteError(w, sql.ErrNoRows, "scheme")

	assertStatus(t, w, http.StatusNotFound)
	var resp APIError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Category != ErrCategoryNotFound {
		t.Errorf("expected category=not_found, got %s", resp.Category)
	}
}

// [REQ:P0-001] Test classifyAndWriteError maps unknown errors to 500 without leaking details
func TestClassifyAndWriteError_Internal(t *testing.T) {
	w := httptest.NewRecorder()
	classifyAndWriteError(w, errors.New("pq: connection refused to database"), "scheme")

	assertStatus(t, w, http.StatusInternalServerError)
	var resp APIError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Category != ErrCategoryInternal {
		t.Errorf("expected category=internal, got %s", resp.Category)
	}
	// The message should NOT contain raw internal error text
	if resp.Message == "pq: connection refused to database" {
		t.Error("internal error details should not be exposed to clients")
	}
}

// [REQ:P0-001] Test classifyAndWriteError maps unique violation to 409
func TestClassifyAndWriteError_UniqueViolation(t *testing.T) {
	w := httptest.NewRecorder()
	classifyAndWriteError(w, errors.New("pq: duplicate key value violates unique constraint (SQLSTATE 23505)"), "edge")

	assertStatus(t, w, http.StatusConflict)
	var resp APIError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Category != ErrCategoryConflict {
		t.Errorf("expected category=conflict, got %s", resp.Category)
	}
}

// [REQ:P0-001] Test classifyAndWriteError maps FK violation to 400
func TestClassifyAndWriteError_ForeignKeyViolation(t *testing.T) {
	w := httptest.NewRecorder()
	classifyAndWriteError(w, errors.New("pq: insert violates foreign key constraint (SQLSTATE 23503)"), "thought")

	assertStatus(t, w, http.StatusBadRequest)
	var resp APIError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Category != ErrCategoryValidation {
		t.Errorf("expected category=validation, got %s", resp.Category)
	}
}

// [REQ:P0-001] Test writeValidationError convenience function
func TestWriteValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	writeValidationError(w, "field X is required")

	assertStatus(t, w, http.StatusBadRequest)
	var resp APIError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Category != ErrCategoryValidation {
		t.Errorf("expected category=validation, got %s", resp.Category)
	}
	if resp.Message != "field X is required" {
		t.Errorf("expected message='field X is required', got %s", resp.Message)
	}
}

// --- Input Validation Tests ---

// [REQ:P0-001] Test update scheme rejects empty name
func TestHandleUpdateScheme_EmptyName(t *testing.T) {
	store := newMockSchemes()
	s := store.seed("Valid Name")

	handler := handleUpdateScheme(store)
	req := httptest.NewRequest("PUT", "/api/v1/schemes/"+s.ID, jsonBody(`{"name":""}`))
	req = setURLVars(req, "id", s.ID)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusBadRequest)
	var resp APIError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Category != ErrCategoryValidation {
		t.Errorf("expected category=validation, got %s", resp.Category)
	}
}

// [REQ:P0-004] Test create edge rejects empty target_id
func TestHandleCreateEdge_EmptyTarget(t *testing.T) {
	store := newMockThoughts()
	handler := handleCreateEdge(store)

	req := httptest.NewRequest("POST", "/api/v1/thoughts/t1/edges", jsonBody(`{"target_id":"","label":"test"}`))
	req = setURLVars(req, "id", "t1")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusBadRequest)
	var resp APIError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Category != ErrCategoryValidation {
		t.Errorf("expected category=validation, got %s", resp.Category)
	}
}

// [REQ:P0-004] Test create edge rejects self-loop
func TestHandleCreateEdge_SelfLoop(t *testing.T) {
	store := newMockThoughts()
	handler := handleCreateEdge(store)

	req := httptest.NewRequest("POST", "/api/v1/thoughts/t1/edges", jsonBody(`{"target_id":"t1","label":"self"}`))
	req = setURLVars(req, "id", "t1")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusBadRequest)
	var resp APIError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Message != "cannot create an edge from a thought to itself" {
		t.Errorf("expected self-loop message, got %s", resp.Message)
	}
}

// [REQ:P2-001] Test suggestion handler returns structured error with dependency category
func TestHandleGenerateSuggestions_StructuredError(t *testing.T) {
	svc := newMockSuggestions().WithGenerateError(errors.New("no LLM provider available"))
	handler := handleGenerateSuggestions(svc)

	req := httptest.NewRequest("POST", "/api/v1/schemes/s1/suggestions", nil)
	req = setURLVars(req, "id", "s1")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusServiceUnavailable)
	var resp APIError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Category != ErrCategoryDependency {
		t.Errorf("expected category=dependency, got %s", resp.Category)
	}
	if !resp.Retryable {
		t.Error("dependency errors should be retryable")
	}
}

// --- Logging Middleware Test ---

// [REQ:P0-001] Test statusWriter captures status code
func TestStatusWriter_CapturesStatus(t *testing.T) {
	w := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
	sw.WriteHeader(http.StatusNotFound)

	if sw.status != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", sw.status)
	}
	if w.Code != http.StatusNotFound {
		t.Errorf("underlying writer should also have 404, got %d", w.Code)
	}
}
