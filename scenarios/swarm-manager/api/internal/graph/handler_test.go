package graph

import (
	"net/http"
	"net/http/httptest"
	"swarm-manager/internal/backlog"
	"testing"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"google.golang.org/protobuf/encoding/protojson"
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

	var resp apipb.GraphResponse
	if err := protojson.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.GetMeta().GetLens() != string(LensTopology) {
		t.Errorf("expected lens topology, got %s", resp.GetMeta().GetLens())
	}
	if resp.GetMeta().GetNodeCount() == 0 {
		t.Error("expected non-zero node count")
	}
	if len(resp.GetNodes()) == 0 || resp.GetNodes()[0].GetData().GetBacklog() == nil {
		t.Fatal("expected backlog node data in proto response")
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

	var resp apipb.GraphResponse
	if err := protojson.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.GetMeta().GetLens() != string(LensTopology) {
		t.Errorf("expected default lens topology, got %s", resp.GetMeta().GetLens())
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
