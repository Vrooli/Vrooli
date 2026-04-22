package overview

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/initiatives"
)

func TestHandlerGetOverview_EmptyBacklog(t *testing.T) {
	svc := NewService(
		&mockBacklogLister{items: []backlog.BacklogItem{}},
		&mockInitiativeLister{items: []initiatives.InitiativeWithRollup{}},
	)
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/overview", nil)
	rec := httptest.NewRecorder()

	h.GetOverview(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var resp OverviewResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(resp.Items))
	}
	if len(resp.Initiatives) != 0 {
		t.Errorf("expected 0 initiatives, got %d", len(resp.Initiatives))
	}
	if resp.Summary.TotalItems != 0 {
		t.Errorf("expected total_items=0, got %d", resp.Summary.TotalItems)
	}
	if resp.Summary.ActiveInitiatives != 0 {
		t.Errorf("expected active_initiatives=0, got %d", resp.Summary.ActiveInitiatives)
	}
}

func TestHandlerGetOverview_PopulatedData(t *testing.T) {
	items := []backlog.BacklogItem{
		{Name: "a", Kind: backlog.KindIdea, Status: backlog.StatusBacklog, Priority: 5, Tags: []string{}},
		{Name: "b", Kind: backlog.KindFix, Status: backlog.StatusReady, Priority: 3, Tags: []string{}},
		{Name: "c", Kind: backlog.KindExecute, Status: backlog.StatusCompleted, Priority: 1, Tags: []string{}, DependsOn: []string{"idea/a"}},
	}
	inits := []initiatives.InitiativeWithRollup{
		{Initiative: initiatives.Initiative{Name: "i1", Status: "active"}},
		{Initiative: initiatives.Initiative{Name: "i2", Status: "completed"}},
	}

	svc := NewService(
		&mockBacklogLister{items: items},
		&mockInitiativeLister{items: inits},
	)
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/overview", nil)
	rec := httptest.NewRecorder()

	h.GetOverview(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp OverviewResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify items are returned.
	if len(resp.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(resp.Items))
	}

	// Verify initiatives are returned.
	if len(resp.Initiatives) != 2 {
		t.Errorf("expected 2 initiatives, got %d", len(resp.Initiatives))
	}

	// Verify dependency graph has edges (c depends on a).
	if len(resp.DependencyGraph.Edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(resp.DependencyGraph.Edges))
	}

	// Verify summary.
	if resp.Summary.TotalItems != 3 {
		t.Errorf("expected total_items=3, got %d", resp.Summary.TotalItems)
	}
	if resp.Summary.ActiveInitiatives != 1 {
		t.Errorf("expected active_initiatives=1, got %d", resp.Summary.ActiveInitiatives)
	}
}

func TestHandlerGetOverview_BacklogError(t *testing.T) {
	svc := NewService(
		&mockBacklogLister{err: fmt.Errorf("disk error")},
		&mockInitiativeLister{items: []initiatives.InitiativeWithRollup{}},
	)
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/overview", nil)
	rec := httptest.NewRecorder()

	h.GetOverview(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestHandlerGetOverview_InitiativeError(t *testing.T) {
	svc := NewService(
		&mockBacklogLister{items: []backlog.BacklogItem{}},
		&mockInitiativeLister{err: fmt.Errorf("disk error")},
	)
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/overview", nil)
	rec := httptest.NewRecorder()

	h.GetOverview(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}
