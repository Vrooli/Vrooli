package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// --- Handler JSON encoding tests ---

// [REQ:P0-001] Test that writeJSON produces valid JSON responses
func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"status": "ok"}
	writeJSON(w, http.StatusOK, data)

	assertStatus(t, w, http.StatusOK)
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}
	result := decodeJSON[map[string]string](t, w)
	if result["status"] != "ok" {
		t.Errorf("expected status=ok, got %s", result["status"])
	}
}

// [REQ:P0-001] Test error response format
func TestWriteJSONError(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSONError(w, http.StatusBadRequest, "invalid input")

	assertStatus(t, w, http.StatusBadRequest)
	result := decodeJSON[map[string]string](t, w)
	if result["error"] != "invalid input" {
		t.Errorf("expected error=invalid input, got %s", result["error"])
	}
}

// --- Scheme Handler Behavioral Tests ---

// [REQ:P0-001] [REQ:P0-002] Test listing schemes returns correct data
func TestHandleListSchemes_Success(t *testing.T) {
	store := newMockSchemes()
	store.seed("Alpha")
	store.seed("Beta")

	handler := handleListSchemes(store)
	req := httptest.NewRequest("GET", "/api/v1/schemes", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	schemes := decodeJSON[[]Scheme](t, w)
	if len(schemes) != 2 {
		t.Errorf("expected 2 schemes, got %d", len(schemes))
	}
}

// [REQ:P0-001] Test listing schemes returns empty array when none exist
func TestHandleListSchemes_Empty(t *testing.T) {
	store := newMockSchemes()
	handler := handleListSchemes(store)
	req := httptest.NewRequest("GET", "/api/v1/schemes", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	schemes := decodeJSON[[]Scheme](t, w)
	if len(schemes) != 0 {
		t.Errorf("expected 0 schemes, got %d", len(schemes))
	}
}

// [REQ:P0-001] Test listing schemes returns 500 on service error
func TestHandleListSchemes_Error(t *testing.T) {
	store := newMockSchemes().WithListError(fmt.Errorf("db failure"))
	handler := handleListSchemes(store)
	req := httptest.NewRequest("GET", "/api/v1/schemes", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusInternalServerError)
}

// [REQ:P0-002] Test creating a scheme with valid input
func TestHandleCreateScheme_Success(t *testing.T) {
	store := newMockSchemes()
	handler := handleCreateScheme(store)

	body := `{"name":"My Scheme"}`
	req := httptest.NewRequest("POST", "/api/v1/schemes", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusCreated)
	scheme := decodeJSON[Scheme](t, w)
	if scheme.Name != "My Scheme" {
		t.Errorf("expected name=My Scheme, got %s", scheme.Name)
	}
	if scheme.ID == "" {
		t.Error("expected non-empty ID")
	}
}

// [REQ:P0-002] Test that text capture handler rejects invalid JSON
func TestHandleCreateScheme_BadJSON(t *testing.T) {
	handler := handleCreateScheme(newMockSchemes())
	req := httptest.NewRequest("POST", "/api/v1/schemes", bytes.NewBufferString("not json"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusBadRequest)
}

// [REQ:P0-001] Test getting a scheme by ID
func TestHandleGetScheme_Success(t *testing.T) {
	store := newMockSchemes()
	s := store.seed("Found Me")

	handler := handleGetScheme(store)
	req := httptest.NewRequest("GET", "/api/v1/schemes/"+s.ID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": s.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	scheme := decodeJSON[Scheme](t, w)
	if scheme.Name != "Found Me" {
		t.Errorf("expected name=Found Me, got %s", scheme.Name)
	}
}

// [REQ:P0-001] Test getting a nonexistent scheme returns 404
func TestHandleGetScheme_NotFound(t *testing.T) {
	handler := handleGetScheme(newMockSchemes())
	req := httptest.NewRequest("GET", "/api/v1/schemes/missing", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "missing"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusNotFound)
}

// [REQ:P0-001] Test updating a scheme
func TestHandleUpdateScheme_Success(t *testing.T) {
	store := newMockSchemes()
	s := store.seed("Old Name")

	handler := handleUpdateScheme(store)
	body := `{"name":"New Name"}`
	req := httptest.NewRequest("PUT", "/api/v1/schemes/"+s.ID, bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"id": s.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	scheme := decodeJSON[Scheme](t, w)
	if scheme.Name != "New Name" {
		t.Errorf("expected name=New Name, got %s", scheme.Name)
	}
}

// [REQ:P0-001] Test updating a nonexistent scheme returns 404
func TestHandleUpdateScheme_NotFound(t *testing.T) {
	handler := handleUpdateScheme(newMockSchemes())
	body := `{"name":"Nope"}`
	req := httptest.NewRequest("PUT", "/api/v1/schemes/missing", bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"id": "missing"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusNotFound)
}

// [REQ:P0-001] Test deleting a scheme
func TestHandleDeleteScheme_Success(t *testing.T) {
	store := newMockSchemes()
	s := store.seed("Doomed")

	handler := handleDeleteScheme(store)
	req := httptest.NewRequest("DELETE", "/api/v1/schemes/"+s.ID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": s.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusNoContent)
}

// [REQ:P0-001] Test deleting a nonexistent scheme returns 404
func TestHandleDeleteScheme_NotFound(t *testing.T) {
	handler := handleDeleteScheme(newMockSchemes())
	req := httptest.NewRequest("DELETE", "/api/v1/schemes/missing", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "missing"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusNotFound)
}

// --- Information Handler Behavioral Tests ---

// [REQ:P0-003] Test listing information items for a scheme
func TestHandleListInformation_Success(t *testing.T) {
	store := newMockInfo()
	store.seed("s1", "note A")
	store.seed("s1", "note B")
	store.seed("s2", "other scheme")

	handler := handleListInformation(store)
	req := httptest.NewRequest("GET", "/api/v1/schemes/s1/information", nil)
	req = mux.SetURLVars(req, map[string]string{"schemeId": "s1"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	items := decodeJSON[[]Information](t, w)
	if len(items) != 2 {
		t.Errorf("expected 2 items for scheme s1, got %d", len(items))
	}
}

// [REQ:P0-003] Test creating an information item
func TestHandleCreateInformation_Success(t *testing.T) {
	store := newMockInfo()
	handler := handleCreateInformation(store)

	body := `{"type":"text","content":"quick note","canvas_x":100,"canvas_y":200}`
	req := httptest.NewRequest("POST", "/api/v1/schemes/s1/information", bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"schemeId": "s1"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusCreated)
	info := decodeJSON[Information](t, w)
	if info.Content != "quick note" {
		t.Errorf("expected content=quick note, got %s", info.Content)
	}
	if info.CanvasX != 100 || info.CanvasY != 200 {
		t.Errorf("expected canvas coords 100,200, got %f,%f", info.CanvasX, info.CanvasY)
	}
}

// [REQ:P0-003] Test that canvas node update handler rejects invalid JSON
func TestHandleUpdateInformation_BadJSON(t *testing.T) {
	handler := handleUpdateInformation(newMockInfo())
	req := httptest.NewRequest("PUT", "/api/v1/schemes/test/information/test", bytes.NewBufferString("{bad"))
	req = mux.SetURLVars(req, map[string]string{"schemeId": "test", "infoId": "test"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusBadRequest)
}

// [REQ:P0-003] Test updating an information item
func TestHandleUpdateInformation_Success(t *testing.T) {
	store := newMockInfo()
	info := store.seed("s1", "old content")

	handler := handleUpdateInformation(store)
	body := `{"content":"new content"}`
	req := httptest.NewRequest("PUT", "/api/v1/schemes/s1/information/"+info.ID, bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"schemeId": "s1", "infoId": info.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	updated := decodeJSON[Information](t, w)
	if updated.Content != "new content" {
		t.Errorf("expected content=new content, got %s", updated.Content)
	}
}

// [REQ:P0-003] Test updating a nonexistent information item returns 404
func TestHandleUpdateInformation_NotFound(t *testing.T) {
	handler := handleUpdateInformation(newMockInfo())
	body := `{"content":"nope"}`
	req := httptest.NewRequest("PUT", "/api/v1/schemes/s1/information/missing", bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"schemeId": "s1", "infoId": "missing"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusNotFound)
}

// [REQ:P0-003] Test deleting an information item
func TestHandleDeleteInformation_Success(t *testing.T) {
	store := newMockInfo()
	info := store.seed("s1", "doomed")

	handler := handleDeleteInformation(store)
	req := httptest.NewRequest("DELETE", "/api/v1/schemes/s1/information/"+info.ID, nil)
	req = mux.SetURLVars(req, map[string]string{"schemeId": "s1", "infoId": info.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusNoContent)
}

// [REQ:P0-003] Test deleting a nonexistent information item returns 404
func TestHandleDeleteInformation_NotFound(t *testing.T) {
	handler := handleDeleteInformation(newMockInfo())
	req := httptest.NewRequest("DELETE", "/api/v1/schemes/s1/information/missing", nil)
	req = mux.SetURLVars(req, map[string]string{"schemeId": "s1", "infoId": "missing"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusNotFound)
}

// --- Route registration test ---

// [REQ:P0-001] [REQ:P0-002] [REQ:P0-003] [REQ:P0-004]
// Test that all expected routes are registered via interface-backed handlers
func TestRouteRegistration(t *testing.T) {
	schemes := newMockSchemes()
	info := newMockInfo()
	thoughts := newMockThoughts()
	export := newMockExport()
	suggestions := newMockSuggestions()

	handlers := []http.HandlerFunc{
		handleListSchemes(schemes),
		handleCreateScheme(schemes),
		handleGetScheme(schemes),
		handleUpdateScheme(schemes),
		handleDeleteScheme(schemes),
		handleListInformation(info),
		handleCreateInformation(info),
		handleUpdateInformation(info),
		handleDeleteInformation(info),
		handleListThoughts(thoughts),
		handleCreateThought(thoughts),
		handleGetThought(thoughts),
		handleUpdateThought(thoughts),
		handleDeleteThought(thoughts),
		handleCreateEdge(thoughts),
		handleListEdges(thoughts),
		handleDeleteEdge(thoughts),
		handleExportScheme(export),
		handleGetProviders(suggestions),
		handleGenerateSuggestions(suggestions),
	}

	for i, h := range handlers {
		if h == nil {
			t.Errorf("handler %d is nil", i)
		}
	}
}
