package initiatives

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"swarm-manager/internal/backlog"
	"testing"

	"github.com/gorilla/mux"
)

// TestHandler_GetContext_ReturnsFullShape verifies the context endpoint
// returns an initiative with its member items (compact view), upstream
// initiatives, and downstream initiatives in one call.
func TestHandler_GetContext_ReturnsFullShape(t *testing.T) {
	store := setupTestStore(t)
	loader := &mockBacklogLoader{items: map[string]backlog.BacklogItem{
		"idea/one": {Kind: "idea", Name: "one", Title: "One", Status: backlog.StatusBacklog, Priority: 3, Initiative: "main"},
		"fix/two":  {Kind: "fix", Name: "two", Title: "Two", Status: backlog.StatusCompleted, Priority: 5, Initiative: "main"},
	}}
	svc := NewService(store, loader)
	h := NewHandler(svc)

	if _, err := svc.Create(CreateRequest{Name: "upstream-a", Title: "Upstream A"}); err != nil {
		t.Fatalf("create upstream: %v", err)
	}
	if _, err := svc.Create(CreateRequest{
		Name: "main", Title: "Main Initiative", Priority: 3,
		DependsOn: []string{"upstream-a"},
		Items:     []string{"idea/one", "fix/two"},
	}); err != nil {
		t.Fatalf("create main: %v", err)
	}
	if _, err := svc.Create(CreateRequest{
		Name: "downstream-b", Title: "Downstream B",
		DependsOn: []string{"main"},
	}); err != nil {
		t.Fatalf("create downstream: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/initiatives/main/context", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "main"})
	w := httptest.NewRecorder()
	h.GetContext(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp InitiativeContext
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Initiative.Name != "main" {
		t.Errorf("initiative name: got %q, want main", resp.Initiative.Name)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 member items, got %d", len(resp.Items))
	}
	if resp.Items[0].Kind != "idea" || resp.Items[0].Name != "one" || resp.Items[0].Title != "One" {
		t.Errorf("first item mismatch: %+v", resp.Items[0])
	}
	if len(resp.UpstreamInitiatives) != 1 || resp.UpstreamInitiatives[0].Name != "upstream-a" {
		t.Errorf("expected 1 upstream (upstream-a), got %+v", resp.UpstreamInitiatives)
	}
	if len(resp.DownstreamInitiatives) != 1 || resp.DownstreamInitiatives[0].Name != "downstream-b" {
		t.Errorf("expected 1 downstream (downstream-b), got %+v", resp.DownstreamInitiatives)
	}
}

// TestHandler_GetContext_EmptyArraysWhenIsolated verifies an isolated
// initiative returns empty arrays rather than nil.
func TestHandler_GetContext_EmptyArraysWhenIsolated(t *testing.T) {
	h := setupTestHandler(t)

	createReq := CreateRequest{Name: "solo", Title: "Solo"}
	rec := httptest.NewRecorder()
	h.Create(rec, requestWithVars("POST", "/api/v1/initiatives", createReq, nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}

	req := httptest.NewRequest("GET", "/api/v1/initiatives/solo/context", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "solo"})
	w := httptest.NewRecorder()
	h.GetContext(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("got %d: %s", w.Code, w.Body.String())
	}

	var resp InitiativeContext
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Items == nil {
		t.Error("Items should be empty slice, not nil")
	}
	if resp.UpstreamInitiatives == nil {
		t.Error("UpstreamInitiatives should be empty slice, not nil")
	}
	if resp.DownstreamInitiatives == nil {
		t.Error("DownstreamInitiatives should be empty slice, not nil")
	}
}

// TestHandler_GetContext_404OnUnknown verifies the endpoint returns 404
// for a non-existent initiative.
func TestHandler_GetContext_404OnUnknown(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest("GET", "/api/v1/initiatives/nope/context", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "nope"})
	w := httptest.NewRecorder()
	h.GetContext(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
