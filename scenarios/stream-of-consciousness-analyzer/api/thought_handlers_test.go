package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// --- Thought Handler Behavioral Tests ---

// [REQ:P0-004] Test listing thoughts
func TestHandleListThoughts_Success(t *testing.T) {
	store := newMockThoughts()
	sid := "s1"
	store.seedThought("Idea A", &sid)
	store.seedThought("Idea B", &sid)

	handler := handleListThoughts(store)
	req := httptest.NewRequest("GET", "/api/v1/thoughts?scheme_id=s1", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	thoughts := decodeJSON[[]Thought](t, w)
	if len(thoughts) != 2 {
		t.Errorf("expected 2 thoughts, got %d", len(thoughts))
	}
}

// [REQ:P0-004] Test listing thoughts without filter returns all
func TestHandleListThoughts_NoFilter(t *testing.T) {
	store := newMockThoughts()
	store.seedThought("Unbound", nil)

	handler := handleListThoughts(store)
	req := httptest.NewRequest("GET", "/api/v1/thoughts", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	thoughts := decodeJSON[[]Thought](t, w)
	if len(thoughts) != 1 {
		t.Errorf("expected 1 thought, got %d", len(thoughts))
	}
}

// [REQ:P0-004] Test creating a thought
func TestHandleCreateThought_Success(t *testing.T) {
	store := newMockThoughts()
	handler := handleCreateThought(store)

	body := `{"title":"Big Idea","body":"Details here","scheme_id":"s1"}`
	req := httptest.NewRequest("POST", "/api/v1/thoughts", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusCreated)
	thought := decodeJSON[Thought](t, w)
	if thought.Title != "Big Idea" {
		t.Errorf("expected title=Big Idea, got %s", thought.Title)
	}
}

// [REQ:P0-004] Test that thought creation handler rejects invalid JSON
func TestHandleCreateThought_BadJSON(t *testing.T) {
	handler := handleCreateThought(newMockThoughts())
	req := httptest.NewRequest("POST", "/api/v1/thoughts", bytes.NewBufferString(""))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusBadRequest)
}

// [REQ:P0-004] Test getting a thought by ID
func TestHandleGetThought_Success(t *testing.T) {
	store := newMockThoughts()
	sid := "s1"
	thought := store.seedThought("Target", &sid)

	handler := handleGetThought(store)
	req := httptest.NewRequest("GET", "/api/v1/thoughts/"+thought.ID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": thought.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	got := decodeJSON[Thought](t, w)
	if got.Title != "Target" {
		t.Errorf("expected title=Target, got %s", got.Title)
	}
}

// [REQ:P0-004] Test getting a nonexistent thought returns 404
func TestHandleGetThought_NotFound(t *testing.T) {
	handler := handleGetThought(newMockThoughts())
	req := httptest.NewRequest("GET", "/api/v1/thoughts/missing", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "missing"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusNotFound)
}

// [REQ:P0-004] Test updating a thought
func TestHandleUpdateThought_Success(t *testing.T) {
	store := newMockThoughts()
	sid := "s1"
	thought := store.seedThought("Old Title", &sid)

	handler := handleUpdateThought(store)
	body := `{"title":"New Title"}`
	req := httptest.NewRequest("PUT", "/api/v1/thoughts/"+thought.ID, bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"id": thought.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	got := decodeJSON[Thought](t, w)
	if got.Title != "New Title" {
		t.Errorf("expected title=New Title, got %s", got.Title)
	}
}

// [REQ:P0-004] Test updating a nonexistent thought returns 404
func TestHandleUpdateThought_NotFound(t *testing.T) {
	handler := handleUpdateThought(newMockThoughts())
	body := `{"title":"Nope"}`
	req := httptest.NewRequest("PUT", "/api/v1/thoughts/missing", bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"id": "missing"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusNotFound)
}

// [REQ:P0-004] Test deleting a thought
func TestHandleDeleteThought_Success(t *testing.T) {
	store := newMockThoughts()
	sid := "s1"
	thought := store.seedThought("Doomed", &sid)

	handler := handleDeleteThought(store)
	req := httptest.NewRequest("DELETE", "/api/v1/thoughts/"+thought.ID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": thought.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusNoContent)
}

// [REQ:P0-004] Test deleting a nonexistent thought returns 404
func TestHandleDeleteThought_NotFound(t *testing.T) {
	handler := handleDeleteThought(newMockThoughts())
	req := httptest.NewRequest("DELETE", "/api/v1/thoughts/missing", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "missing"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusNotFound)
}

// [REQ:P2-002] Test cross-scheme thought creation via handler
func TestHandleCreateThought_CrossScheme(t *testing.T) {
	store := newMockThoughts()
	handler := handleCreateThought(store)

	body := `{"title":"Cross-scheme idea","body":"Spans schemes"}`
	req := httptest.NewRequest("POST", "/api/v1/thoughts", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusCreated)
	thought := decodeJSON[Thought](t, w)
	if thought.Title != "Cross-scheme idea" {
		t.Errorf("expected title=Cross-scheme idea, got %s", thought.Title)
	}
}

// --- Edge Handler Behavioral Tests ---

// [REQ:P0-004] Test creating an edge
func TestHandleCreateEdge_Success(t *testing.T) {
	store := newMockThoughts()
	handler := handleCreateEdge(store)

	body := `{"target_id":"thought-b","label":"causes"}`
	req := httptest.NewRequest("POST", "/api/v1/thoughts/thought-a/edges", bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"id": "thought-a"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusCreated)
	edge := decodeJSON[ThoughtEdge](t, w)
	if edge.SourceID != "thought-a" || edge.TargetID != "thought-b" {
		t.Errorf("expected edge thought-a→thought-b, got %s→%s", edge.SourceID, edge.TargetID)
	}
	if edge.Label != "causes" {
		t.Errorf("expected label=causes, got %s", edge.Label)
	}
}

// [REQ:P0-004] Test that edge creation handler rejects invalid JSON
func TestHandleCreateEdge_BadJSON(t *testing.T) {
	handler := handleCreateEdge(newMockThoughts())
	req := httptest.NewRequest("POST", "/api/v1/thoughts/test/edges", bytes.NewBufferString(""))
	req = mux.SetURLVars(req, map[string]string{"id": "test"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusBadRequest)
}

// [REQ:P0-004] Test listing edges for a thought
func TestHandleListEdges_Success(t *testing.T) {
	store := newMockThoughts()
	store.seedEdge("t1", "t2", "causes")
	store.seedEdge("t3", "t1", "supports")

	handler := handleListEdges(store)
	req := httptest.NewRequest("GET", "/api/v1/thoughts/t1/edges", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "t1"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusOK)
	edges := decodeJSON[[]ThoughtEdge](t, w)
	if len(edges) != 2 {
		t.Errorf("expected 2 edges for t1, got %d", len(edges))
	}
}

// [REQ:P0-004] Test deleting an edge
func TestHandleDeleteEdge_Success(t *testing.T) {
	store := newMockThoughts()
	edge := store.seedEdge("t1", "t2", "causes")

	handler := handleDeleteEdge(store)
	req := httptest.NewRequest("DELETE", "/api/v1/thoughts/t1/edges/"+edge.ID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": "t1", "edgeId": edge.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusNoContent)
}

// [REQ:P0-004] Test deleting a nonexistent edge returns 404
func TestHandleDeleteEdge_NotFound(t *testing.T) {
	handler := handleDeleteEdge(newMockThoughts())
	req := httptest.NewRequest("DELETE", "/api/v1/thoughts/t1/edges/missing", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "t1", "edgeId": "missing"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assertStatus(t, w, http.StatusNotFound)
}
