package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"tunnel-manager/domain"
)

func TestWriteJSON_StatusAndHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusCreated, map[string]string{"key": "value"})

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %s", ct)
	}
}

func TestWriteJSON_Body(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]int{"count": 42})

	var result map[string]int
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["count"] != 42 {
		t.Fatalf("expected count=42, got %d", result["count"])
	}
}

func TestWriteJSON_NilData(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestWriteJSON_ArrayData(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, []string{"a", "b", "c"})

	var result []string
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(result))
	}
}

func TestWriteError_MapsValidationTo400(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, domain.ErrValidation("bad input"))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	assertErrorCode(t, w, domain.ErrCodeValidation)
}

func TestWriteError_MapsNotFoundTo404(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, domain.ErrNotFound("missing"))

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	assertErrorCode(t, w, domain.ErrCodeNotFound)
}

func TestWriteError_MapsConflictTo409(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, domain.ErrConflict("duplicate"))

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
	assertErrorCode(t, w, domain.ErrCodeConflict)
}

func TestWriteError_MapsUnavailableTo503(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, domain.ErrUnavailable("down"))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
	assertErrorCode(t, w, domain.ErrCodeUnavailable)
}

func TestWriteError_MapsInternalTo500(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, domain.ErrInternal("db broke", fmt.Errorf("connection refused")))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	assertErrorCode(t, w, domain.ErrCodeInternal)
}

func TestWriteError_UnknownErrorTo500(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, fmt.Errorf("something unexpected"))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	assertErrorCode(t, w, domain.ErrCodeInternal)
}

func assertErrorCode(t *testing.T, w *httptest.ResponseRecorder, expectedCode string) {
	t.Helper()
	var resp domain.ApiErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.ErrorCode != expectedCode {
		t.Fatalf("expected error_code %q, got %q", expectedCode, resp.ErrorCode)
	}
}
