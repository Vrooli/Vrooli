package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// [REQ:P0-003] Test creating information with default type
func TestHandleCreateInformation_DefaultType(t *testing.T) {
	store := newMockInfo()
	handler := handleCreateInformation(store)

	body := `{"content":"no type specified"}`
	req := httptest.NewRequest("POST", "/api/v1/schemes/s1/information", bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"schemeId": "s1"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusCreated)
	info := decodeJSON[Information](t, w)
	if info.Type != "text" {
		t.Errorf("expected default type=text, got %s", info.Type)
	}
}

// [REQ:P0-003] Test creating information with explicit type
func TestHandleCreateInformation_ExplicitType(t *testing.T) {
	store := newMockInfo()
	handler := handleCreateInformation(store)

	body := `{"type":"voice","content":"transcribed audio"}`
	req := httptest.NewRequest("POST", "/api/v1/schemes/s1/information", bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"schemeId": "s1"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusCreated)
	info := decodeJSON[Information](t, w)
	if info.Type != "voice" {
		t.Errorf("expected type=voice, got %s", info.Type)
	}
}

// [REQ:P0-003] Test creating information rejects bad JSON
func TestHandleCreateInformation_BadJSON(t *testing.T) {
	handler := handleCreateInformation(newMockInfo())
	req := httptest.NewRequest("POST", "/api/v1/schemes/s1/information", bytes.NewBufferString("{bad"))
	req = mux.SetURLVars(req, map[string]string{"schemeId": "s1"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusBadRequest)
}

// [REQ:P0-003] Test listing information returns empty array for unknown scheme
func TestHandleListInformation_Empty(t *testing.T) {
	store := newMockInfo()
	handler := handleListInformation(store)
	req := httptest.NewRequest("GET", "/api/v1/schemes/unknown/information", nil)
	req = mux.SetURLVars(req, map[string]string{"schemeId": "unknown"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	items := decodeJSON[[]Information](t, w)
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

// [REQ:P0-003] Test listing information returns 500 on service error
func TestHandleListInformation_Error(t *testing.T) {
	store := newMockInfo().WithListError(fmt.Errorf("db failure"))
	handler := handleListInformation(store)
	req := httptest.NewRequest("GET", "/api/v1/schemes/s1/information", nil)
	req = mux.SetURLVars(req, map[string]string{"schemeId": "s1"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusInternalServerError)
}

// [REQ:P0-003] Test creating information returns 500 on service error
func TestHandleCreateInformation_ServiceError(t *testing.T) {
	store := newMockInfo().WithCreateError(fmt.Errorf("db write failed"))
	handler := handleCreateInformation(store)
	req := httptest.NewRequest("POST", "/api/v1/schemes/s1/information", bytes.NewBufferString(`{"content":"test"}`))
	req = mux.SetURLVars(req, map[string]string{"schemeId": "s1"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusInternalServerError)
}

// [REQ:P0-003] Test updating information returns 500 on service error
func TestHandleUpdateInformation_ServiceError(t *testing.T) {
	store := newMockInfo().WithUpdateError(fmt.Errorf("db update failed"))
	handler := handleUpdateInformation(store)
	body := `{"content":"test"}`
	req := httptest.NewRequest("PUT", "/api/v1/schemes/s1/information/info1", bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"schemeId": "s1", "infoId": "info1"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusInternalServerError)
}

// [REQ:P0-003] Test updating information with canvas coordinates
func TestHandleUpdateInformation_CanvasCoords(t *testing.T) {
	store := newMockInfo()
	info := store.seed("s1", "positioned note")

	handler := handleUpdateInformation(store)
	body := `{"canvas_x":150.5,"canvas_y":275.3}`
	req := httptest.NewRequest("PUT", "/api/v1/schemes/s1/information/"+info.ID, bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"schemeId": "s1", "infoId": info.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	updated := decodeJSON[Information](t, w)
	if updated.CanvasX != 150.5 || updated.CanvasY != 275.3 {
		t.Errorf("expected canvas coords 150.5,275.3, got %f,%f", updated.CanvasX, updated.CanvasY)
	}
}

// [REQ:P0-003] Test creating information with canvas coordinates
func TestHandleCreateInformation_WithCanvasCoords(t *testing.T) {
	store := newMockInfo()
	handler := handleCreateInformation(store)

	body := `{"content":"positioned","canvas_x":100.5,"canvas_y":200.3}`
	req := httptest.NewRequest("POST", "/api/v1/schemes/s1/information", bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"schemeId": "s1"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusCreated)
	info := decodeJSON[Information](t, w)
	if info.CanvasX != 100.5 || info.CanvasY != 200.3 {
		t.Errorf("expected coords 100.5,200.3, got %f,%f", info.CanvasX, info.CanvasY)
	}
}

// [REQ:P0-003] Test listing information with multiple items
func TestHandleListInformation_WithItems(t *testing.T) {
	store := newMockInfo()
	store.seed("s1", "note one")
	store.seed("s1", "note two")
	store.seed("s2", "different scheme")

	handler := handleListInformation(store)
	req := httptest.NewRequest("GET", "/api/v1/schemes/s1/information", nil)
	req = mux.SetURLVars(req, map[string]string{"schemeId": "s1"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	items := decodeJSON[[]Information](t, w)
	if len(items) != 2 {
		t.Errorf("expected 2 items for s1, got %d", len(items))
	}
}

// [REQ:P0-003] Test updating information content preserves other fields
func TestHandleUpdateInformation_ContentOnly(t *testing.T) {
	store := newMockInfo()
	info := store.seed("s1", "original")

	handler := handleUpdateInformation(store)
	body := `{"content":"updated content"}`
	req := httptest.NewRequest("PUT", "/api/v1/schemes/s1/information/"+info.ID, bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"schemeId": "s1", "infoId": info.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	updated := decodeJSON[Information](t, w)
	if updated.Content != "updated content" {
		t.Errorf("expected content='updated content', got %s", updated.Content)
	}
}
