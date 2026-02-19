package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// [REQ:RM-001] Helper function tests

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

func TestWriteJSONError_Format(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSONError(w, http.StatusBadRequest, "something went wrong")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var result map[string]string
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result["error"] != "something went wrong" {
		t.Fatalf("expected error message, got %q", result["error"])
	}
}

func TestWriteJSONError_InternalServer(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSONError(w, http.StatusInternalServerError, "db fail")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
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
