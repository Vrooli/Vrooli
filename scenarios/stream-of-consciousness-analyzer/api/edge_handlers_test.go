package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// [REQ:P0-004] Test creating edge with label
func TestHandleCreateEdge_WithLabel(t *testing.T) {
	store := newMockThoughts()
	sid := strPtr("s1")
	source := store.seedThought("Cause", sid)
	target := store.seedThought("Effect", sid)

	handler := handleCreateEdge(store)
	body := `{"target_id":"` + target.ID + `","label":"causes"}`
	req := httptest.NewRequest("POST", "/api/v1/thoughts/"+source.ID+"/edges", bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"id": source.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusCreated)
	edge := decodeJSON[ThoughtEdge](t, w)
	if edge.Label != "causes" {
		t.Errorf("expected label=causes, got %s", edge.Label)
	}
	if edge.SourceID != source.ID {
		t.Errorf("expected source=%s, got %s", source.ID, edge.SourceID)
	}
	if edge.TargetID != target.ID {
		t.Errorf("expected target=%s, got %s", target.ID, edge.TargetID)
	}
}

// [REQ:P0-004] Test creating edge rejects self-referencing target
func TestHandleCreateEdge_SelfReferenceLoop(t *testing.T) {
	store := newMockThoughts()
	sid := strPtr("s1")
	thought := store.seedThought("Lonely", sid)

	handler := handleCreateEdge(store)
	body := `{"target_id":"` + thought.ID + `"}`
	req := httptest.NewRequest("POST", "/api/v1/thoughts/"+thought.ID+"/edges", bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"id": thought.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusBadRequest)
}

// [REQ:P0-004] Test creating edge rejects blank target ID
func TestHandleCreateEdge_BlankTarget(t *testing.T) {
	store := newMockThoughts()
	sid := strPtr("s1")
	source := store.seedThought("Source", sid)

	handler := handleCreateEdge(store)
	body := `{"target_id":"","label":"broken"}`
	req := httptest.NewRequest("POST", "/api/v1/thoughts/"+source.ID+"/edges", bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"id": source.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusBadRequest)
}

// [REQ:P0-004] Test listing edges returns empty for thought with no edges
func TestHandleListEdges_Empty(t *testing.T) {
	store := newMockThoughts()
	sid := strPtr("s1")
	thought := store.seedThought("Isolated", sid)

	handler := handleListEdges(store)
	req := httptest.NewRequest("GET", "/api/v1/thoughts/"+thought.ID+"/edges", nil)
	req = mux.SetURLVars(req, map[string]string{"id": thought.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	edges := decodeJSON[[]ThoughtEdge](t, w)
	if len(edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(edges))
	}
}

// [REQ:P0-004] Test listing edges returns 500 on service error
func TestHandleListEdges_Error(t *testing.T) {
	store := newMockThoughts().WithListEdgesError(fmt.Errorf("db failure"))
	handler := handleListEdges(store)
	req := httptest.NewRequest("GET", "/api/v1/thoughts/t1/edges", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "t1"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusInternalServerError)
}

// [REQ:P0-004] Test creating edge returns 500 on service error
func TestHandleCreateEdge_ServiceError(t *testing.T) {
	store := newMockThoughts().WithCreateEdgeError(fmt.Errorf("db write failed"))
	handler := handleCreateEdge(store)
	body := `{"target_id":"t2","label":"test"}`
	req := httptest.NewRequest("POST", "/api/v1/thoughts/t1/edges", bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"id": "t1"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusInternalServerError)
}

// [REQ:P0-004] Test listing edges returns connected edges
func TestHandleListEdges_WithEdges(t *testing.T) {
	store := newMockThoughts()
	sid := strPtr("s1")
	t1 := store.seedThought("A", sid)
	t2 := store.seedThought("B", sid)
	store.seedEdge(t1.ID, t2.ID, "related")

	handler := handleListEdges(store)
	req := httptest.NewRequest("GET", "/api/v1/thoughts/"+t1.ID+"/edges", nil)
	req = mux.SetURLVars(req, map[string]string{"id": t1.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	edges := decodeJSON[[]ThoughtEdge](t, w)
	if len(edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(edges))
	}
}

// [REQ:P0-004] Test deleting an existing edge removes it
func TestHandleDeleteEdge_Existing(t *testing.T) {
	store := newMockThoughts()
	sid := strPtr("s1")
	t1 := store.seedThought("A", sid)
	t2 := store.seedThought("B", sid)
	e := store.seedEdge(t1.ID, t2.ID, "temp")

	handler := handleDeleteEdge(store)
	req := httptest.NewRequest("DELETE", "/api/v1/thoughts/"+t1.ID+"/edges/"+e.ID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": t1.ID, "edgeId": e.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusNoContent)
}

// [REQ:P0-004] Test deleting missing edge returns 404
func TestHandleDeleteEdge_Missing(t *testing.T) {
	store := newMockThoughts()
	handler := handleDeleteEdge(store)
	req := httptest.NewRequest("DELETE", "/api/v1/thoughts/t1/edges/missing", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "t1", "edgeId": "missing"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusNotFound)
}

// [REQ:P0-004] [REQ:P2-004a] Test creating edge rejects malformed JSON body
func TestHandleCreateEdge_MalformedJSON(t *testing.T) {
	store := newMockThoughts()
	sid := strPtr("s1")
	source := store.seedThought("Source", sid)

	handler := handleCreateEdge(store)
	req := httptest.NewRequest("POST", "/api/v1/thoughts/"+source.ID+"/edges", bytes.NewBufferString("{bad"))
	req = mux.SetURLVars(req, map[string]string{"id": source.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusBadRequest)
}

// [REQ:P0-004] [REQ:P2-004a] Test edge creation without label uses empty string
func TestHandleCreateEdge_NoLabel(t *testing.T) {
	store := newMockThoughts()
	sid := strPtr("s1")
	source := store.seedThought("A", sid)
	target := store.seedThought("B", sid)

	handler := handleCreateEdge(store)
	body := `{"target_id":"` + target.ID + `"}`
	req := httptest.NewRequest("POST", "/api/v1/thoughts/"+source.ID+"/edges", bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"id": source.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusCreated)
	edge := decodeJSON[ThoughtEdge](t, w)
	if edge.Label != "" {
		t.Errorf("expected empty label, got %q", edge.Label)
	}
}

// [REQ:P0-004] Test listing edges from target perspective includes edge
func TestHandleListEdges_TargetPerspective(t *testing.T) {
	store := newMockThoughts()
	sid := strPtr("s1")
	t1 := store.seedThought("A", sid)
	t2 := store.seedThought("B", sid)
	store.seedEdge(t1.ID, t2.ID, "flows-to")

	// List edges from target's perspective
	handler := handleListEdges(store)
	req := httptest.NewRequest("GET", "/api/v1/thoughts/"+t2.ID+"/edges", nil)
	req = mux.SetURLVars(req, map[string]string{"id": t2.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	edges := decodeJSON[[]ThoughtEdge](t, w)
	if len(edges) != 1 {
		t.Errorf("expected 1 edge from target perspective, got %d", len(edges))
	}
}
