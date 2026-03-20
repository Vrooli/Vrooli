package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// [REQ:P0-001] Test update scheme rejects blank name
func TestHandleUpdateScheme_BlankName(t *testing.T) {
	store := newMockSchemes()
	s := store.seed("Valid")

	handler := handleUpdateScheme(store)
	body := `{"name":""}`
	req := httptest.NewRequest("PUT", "/api/v1/schemes/"+s.ID, bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"id": s.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusBadRequest)
}

// [REQ:P0-001] Test update scheme rejects bad JSON
func TestHandleUpdateScheme_BadJSON(t *testing.T) {
	handler := handleUpdateScheme(newMockSchemes())
	req := httptest.NewRequest("PUT", "/api/v1/schemes/test", bytes.NewBufferString("{bad"))
	req = mux.SetURLVars(req, map[string]string{"id": "test"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusBadRequest)
}

// [REQ:P0-001] Test create scheme with empty name defaults to provided name
func TestHandleCreateScheme_EmptyName(t *testing.T) {
	store := newMockSchemes()
	handler := handleCreateScheme(store)

	body := `{"name":""}`
	req := httptest.NewRequest("POST", "/api/v1/schemes", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusCreated)
	scheme := decodeJSON[Scheme](t, w)
	if scheme.Name != "Untitled" {
		t.Errorf("expected name=Untitled for empty input, got %s", scheme.Name)
	}
}

// [REQ:P0-002] Test scheme lifecycle: create, get, update, delete
func TestSchemeLifecycle(t *testing.T) {
	store := newMockSchemes()

	// Create
	createHandler := handleCreateScheme(store)
	req := httptest.NewRequest("POST", "/api/v1/schemes", bytes.NewBufferString(`{"name":"Lifecycle Test"}`))
	w := httptest.NewRecorder()
	createHandler.ServeHTTP(w, req)
	assertStatus(t, w, http.StatusCreated)
	created := decodeJSON[Scheme](t, w)

	// Get
	getHandler := handleGetScheme(store)
	req = httptest.NewRequest("GET", "/api/v1/schemes/"+created.ID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": created.ID})
	w = httptest.NewRecorder()
	getHandler.ServeHTTP(w, req)
	assertStatus(t, w, http.StatusOK)
	got := decodeJSON[Scheme](t, w)
	if got.Name != "Lifecycle Test" {
		t.Errorf("expected name=Lifecycle Test, got %s", got.Name)
	}

	// Update
	updateHandler := handleUpdateScheme(store)
	req = httptest.NewRequest("PUT", "/api/v1/schemes/"+created.ID, bytes.NewBufferString(`{"name":"Updated"}`))
	req = mux.SetURLVars(req, map[string]string{"id": created.ID})
	w = httptest.NewRecorder()
	updateHandler.ServeHTTP(w, req)
	assertStatus(t, w, http.StatusOK)
	updated := decodeJSON[Scheme](t, w)
	if updated.Name != "Updated" {
		t.Errorf("expected name=Updated, got %s", updated.Name)
	}

	// Delete
	deleteHandler := handleDeleteScheme(store)
	req = httptest.NewRequest("DELETE", "/api/v1/schemes/"+created.ID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": created.ID})
	w = httptest.NewRecorder()
	deleteHandler.ServeHTTP(w, req)
	assertStatus(t, w, http.StatusNoContent)

	// Verify deletion
	req = httptest.NewRequest("GET", "/api/v1/schemes/"+created.ID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": created.ID})
	w = httptest.NewRecorder()
	getHandler.ServeHTTP(w, req)
	assertStatus(t, w, http.StatusNotFound)
}
