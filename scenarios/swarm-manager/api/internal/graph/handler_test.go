package graph

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"swarm-manager/internal/backlog"
)

func newTestHandler() *Handler {
	svc := NewProjectionService(ProjectionConfig{
		Backlog: &mockBacklogLister{items: []backlog.BacklogItem{
			{Kind: "execute", Name: "test-task", Title: "Test", Status: "ready", Priority: 3},
		}},
		Scenario: &mockScenarioLister{scens: []ScenarioEntry{
			{Name: "my-app", Status: "running"},
		}},
	})
	return NewHandler(svc)
}

func TestGraphHandler(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest("GET", "/api/v1/graph?lens=topology", nil)
	w := httptest.NewRecorder()

	h.GetGraph(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp GraphResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Meta.Lens != LensTopology {
		t.Errorf("expected lens topology, got %s", resp.Meta.Lens)
	}
	if resp.Meta.NodeCount == 0 {
		t.Error("expected non-zero node count")
	}
}

func TestGraphHandlerDefaultLens(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest("GET", "/api/v1/graph", nil)
	w := httptest.NewRecorder()

	h.GetGraph(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp GraphResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Meta.Lens != LensTopology {
		t.Errorf("expected default lens topology, got %s", resp.Meta.Lens)
	}
}

func TestGraphHandlerInvalidLens(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest("GET", "/api/v1/graph?lens=invalid", nil)
	w := httptest.NewRecorder()

	h.GetGraph(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
