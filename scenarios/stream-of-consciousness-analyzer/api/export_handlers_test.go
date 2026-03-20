package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// --- Export Handler Behavioral Tests ---

// [REQ:P1-002] Test export handler returns full graph data
func TestHandleExportScheme_Success(t *testing.T) {
	store := newMockExport()
	store.seed("s1", "Test Scheme")

	handler := handleExportScheme(store)
	req := httptest.NewRequest("GET", "/api/v1/schemes/s1/export", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "s1"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	data := decodeJSON[ExportData](t, w)
	if data.ExportFormat != "vrooli-graph-v1" {
		t.Errorf("expected format vrooli-graph-v1, got %s", data.ExportFormat)
	}
	if data.Scheme.Name != "Test Scheme" {
		t.Errorf("expected scheme name=Test Scheme, got %s", data.Scheme.Name)
	}
}

// [REQ:P1-002] Test export handler returns 404 for unknown scheme
func TestHandleExportScheme_NotFound(t *testing.T) {
	handler := handleExportScheme(newMockExport())
	req := httptest.NewRequest("GET", "/api/v1/schemes/missing/export", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "missing"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusNotFound)
}

// [REQ:P1-002] Test export handler factory initialization
func TestHandleExportScheme_Init(t *testing.T) {
	handler := handleExportScheme(newMockExport())
	if handler == nil {
		t.Error("expected non-nil handler")
	}
}
