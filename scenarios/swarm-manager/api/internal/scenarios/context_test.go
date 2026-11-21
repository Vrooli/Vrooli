package scenarios

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/goals"

	"github.com/gorilla/mux"
)

// stubGoalsLister returns canned derived goal scopes for context tests.
type stubGoalsLister struct {
	items []goals.GoalWithScope
	err   error
}

func (s stubGoalsLister) List() ([]goals.GoalWithScope, error) {
	return s.items, s.err
}

func newContextTestHandler(backlogItems []backlog.BacklogItem, goalList []goals.GoalWithScope) *Handler {
	h := NewHandler("")
	h.SetBacklogLister(stubBacklogLister{items: backlogItems})
	h.SetGoalsLister(stubGoalsLister{items: goalList})
	return h
}

func getContext(t *testing.T, h *Handler, name string) ScenarioContext {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/v1/scenarios/"+name+"/context", nil)
	req = mux.SetURLVars(req, map[string]string{"name": name})
	rec := httptest.NewRecorder()
	h.GetContext(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var ctx ScenarioContext
	if err := json.NewDecoder(rec.Body).Decode(&ctx); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return ctx
}

func TestGetContext_ReturnsGoalsOrphansAndRollup(t *testing.T) {
	goalList := []goals.GoalWithScope{{Goal: goals.Goal{Name: "audio-platform", Title: "Audio", Status: goals.StatusActive}, Scope: goals.Scope{Closure: []string{"fix/assigned-b"}}}}
	items := []backlog.BacklogItem{
		// Orphan targeting web-console.
		makeItem("orphan-a", backlog.KindExecute, backlog.StatusBacklog, []string{"web-console"}, nil),
		// Included by a goal closure — should NOT appear as orphan.
		{
			Name: "assigned-b", Kind: backlog.KindFix, Status: backlog.StatusBacklog,
			AcceptanceAllow: []string{"scenarios/web-console/**"},
		},
		// Orphan targeting different scenario — should be excluded.
		makeItem("orphan-c", backlog.KindIdea, backlog.StatusBacklog, []string{"command-center"}, nil),
	}
	h := newContextTestHandler(items, goalList)

	ctx := getContext(t, h, "web-console")

	if ctx.ScenarioName != "web-console" {
		t.Errorf("scenario_name = %q, want web-console", ctx.ScenarioName)
	}
	if len(ctx.Goals) != 1 || ctx.Goals[0].Name != "audio-platform" {
		t.Errorf("goals = %+v, want [audio-platform]", ctx.Goals)
	}
	if len(ctx.OrphanItems) != 1 || ctx.OrphanItems[0].Name != "orphan-a" {
		t.Errorf("orphan_items = %+v, want [orphan-a]", ctx.OrphanItems)
	}
	if ctx.Rollup.Total != 2 || ctx.Rollup.Pending != 2 {
		t.Errorf("rollup = %+v, want two pending items", ctx.Rollup)
	}
}

func TestGetContext_EmptyScenario_ReturnsWellFormedEmpty(t *testing.T) {
	h := newContextTestHandler(nil, nil)

	ctx := getContext(t, h, "no-such-scenario")

	if ctx.ScenarioName != "no-such-scenario" {
		t.Errorf("scenario_name = %q, want no-such-scenario", ctx.ScenarioName)
	}
	if len(ctx.Goals) != 0 {
		t.Errorf("expected empty goals, got %d", len(ctx.Goals))
	}
	if len(ctx.OrphanItems) != 0 {
		t.Errorf("expected empty orphans, got %d", len(ctx.OrphanItems))
	}
	if ctx.Rollup.Total != 0 {
		t.Errorf("expected zero rollup, got %+v", ctx.Rollup)
	}
}

func TestGetContext_MissingName_Returns400(t *testing.T) {
	h := newContextTestHandler(nil, nil)
	req := httptest.NewRequest("GET", "/api/v1/scenarios//context", nil)
	req = mux.SetURLVars(req, map[string]string{"name": ""})
	rec := httptest.NewRecorder()
	h.GetContext(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetContext_OrphanWithDifferentScenarioExcluded(t *testing.T) {
	items := []backlog.BacklogItem{
		makeItem("wc-orphan", backlog.KindExecute, backlog.StatusBacklog, []string{"web-console"}, nil),
		makeItem("cc-orphan", backlog.KindExecute, backlog.StatusBacklog, []string{"command-center"}, nil),
	}
	h := newContextTestHandler(items, nil)

	ctx := getContext(t, h, "web-console")
	if len(ctx.OrphanItems) != 1 || ctx.OrphanItems[0].Name != "wc-orphan" {
		t.Errorf("expected only wc-orphan, got %+v", ctx.OrphanItems)
	}
}

func TestGetContext_ItemInMultipleScenarios_CountedOnce(t *testing.T) {
	// An item targeting both web-console and command-center should appear
	// once in each scenario's context — orphan for each.
	item := backlog.BacklogItem{
		Name:            "multi-target",
		Kind:            backlog.KindExecute,
		Status:          backlog.StatusBacklog,
		AcceptanceAllow: []string{"scenarios/web-console/**", "scenarios/command-center/**"},
	}
	h := newContextTestHandler([]backlog.BacklogItem{item}, nil)

	webCtx := getContext(t, h, "web-console")
	if len(webCtx.OrphanItems) != 1 {
		t.Errorf("web-console orphans = %d, want 1", len(webCtx.OrphanItems))
	}
	ccCtx := getContext(t, h, "command-center")
	if len(ccCtx.OrphanItems) != 1 {
		t.Errorf("command-center orphans = %d, want 1", len(ccCtx.OrphanItems))
	}
}

func TestGetContext_FixHistory_ActiveAndArchivedPartitioned(t *testing.T) {
	archived := "2026-04-20T12:00:00Z"
	items := []backlog.BacklogItem{
		// Active orphan fix.
		{
			Name: "fix-orphan-active", Title: "Active orphan", Kind: backlog.KindFix,
			Status: backlog.StatusBacklog, Priority: 2, Updated: "2026-04-22T10:00:00Z",
			AcceptanceAllow: []string{"scenarios/web-console/**"},
		},
		// Archived orphan fix.
		{
			Name: "fix-orphan-archived", Title: "Archived orphan", Kind: backlog.KindFix,
			Status: backlog.StatusCompleted, Priority: 1, Updated: "2026-04-15T10:00:00Z",
			AcceptanceAllow: []string{"scenarios/web-console/**"},
			ArchivedAt:      &archived,
		},
		// Active fix in a goal — must still appear in fixes.
		{
			Name: "fix-in-init", Title: "In initiative", Kind: backlog.KindFix,
			Status: backlog.StatusBacklog, Priority: 3, Updated: "2026-04-23T10:00:00Z",
			AcceptanceAllow: []string{"scenarios/web-console/**"},
		},
		// Non-fix kind targeting same scenario — must NOT appear in fixes.
		makeItem("not-a-fix", backlog.KindExecute, backlog.StatusBacklog, []string{"web-console"}, nil),
		// Fix targeting different scenario — must NOT appear.
		{
			Name: "fix-other", Kind: backlog.KindFix, Status: backlog.StatusBacklog,
			AcceptanceAllow: []string{"scenarios/command-center/**"},
		},
	}
	h := newContextTestHandler(items, nil)

	ctx := getContext(t, h, "web-console")

	if len(ctx.Fixes.Active) != 2 {
		t.Fatalf("active fixes = %d, want 2: %+v", len(ctx.Fixes.Active), ctx.Fixes.Active)
	}
	if ctx.Fixes.Active[0].Name != "fix-in-init" {
		t.Errorf("active[0] = %q, want fix-in-init (highest priority first)", ctx.Fixes.Active[0].Name)
	}
	if ctx.Fixes.Active[0].Goal != "" {
		t.Errorf("unexpected goal attribution without a goal closure: %+v", ctx.Fixes.Active[0])
	}
	if ctx.Fixes.Active[1].Name != "fix-orphan-active" {
		t.Errorf("active[1] = %q, want fix-orphan-active", ctx.Fixes.Active[1].Name)
	}
	if len(ctx.Fixes.Archived) != 1 || ctx.Fixes.Archived[0].Name != "fix-orphan-archived" {
		t.Errorf("archived fixes = %+v, want [fix-orphan-archived]", ctx.Fixes.Archived)
	}
	if ctx.Fixes.Active[0].Path != "fix/fix-in-init" {
		t.Errorf("path = %q, want fix/fix-in-init", ctx.Fixes.Active[0].Path)
	}
	if ctx.Fixes.Archived[0].ArchivedAt == nil || *ctx.Fixes.Archived[0].ArchivedAt != archived {
		t.Errorf("archivedAt not preserved: %+v", ctx.Fixes.Archived[0])
	}
}

func TestGetContext_FixHistory_EmptyWhenNoFixes(t *testing.T) {
	items := []backlog.BacklogItem{
		makeItem("only-exec", backlog.KindExecute, backlog.StatusBacklog, []string{"web-console"}, nil),
	}
	h := newContextTestHandler(items, nil)

	ctx := getContext(t, h, "web-console")

	if ctx.Fixes.Active == nil || ctx.Fixes.Archived == nil {
		t.Errorf("fixes arrays must be non-nil for stable JSON shape: %+v", ctx.Fixes)
	}
	if len(ctx.Fixes.Active) != 0 || len(ctx.Fixes.Archived) != 0 {
		t.Errorf("expected empty fixes, got %+v", ctx.Fixes)
	}
}

func TestGetContext_GoalRollupAggregatesItemStates(t *testing.T) {
	archived := "2026-01-01T00:00:00Z"
	items := []backlog.BacklogItem{
		{Name: "done", Kind: backlog.KindExecute, Status: backlog.StatusCompleted, AcceptanceAllow: []string{"scenarios/target/**"}},
		{Name: "working", Kind: backlog.KindExecute, Status: backlog.StatusInProgress, AcceptanceAllow: []string{"scenarios/target/**"}},
		{Name: "failed", Kind: backlog.KindExecute, Status: backlog.StatusFailed, AcceptanceAllow: []string{"scenarios/target/**"}, ArchivedAt: &archived},
	}
	goalList := []goals.GoalWithScope{{Goal: goals.Goal{Name: "one", Status: goals.StatusActive}, Scope: goals.Scope{Closure: []string{"execute/done", "execute/working", "execute/failed"}}}}
	ctx := getContext(t, newContextTestHandler(items, goalList), "target")
	if ctx.Rollup.Total != 3 || ctx.Rollup.Completed != 1 || ctx.Rollup.InProgress != 1 || ctx.Rollup.Failed != 1 || ctx.Rollup.Archived != 1 {
		t.Errorf("rollup = %+v", ctx.Rollup)
	}
}
