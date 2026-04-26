package scenarios

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/initiatives"

	"github.com/gorilla/mux"
)

// stubInitiativesLister returns a canned list of initiatives for context tests.
type stubInitiativesLister struct {
	items []initiatives.InitiativeWithRollup
	err   error
}

func (s stubInitiativesLister) List() ([]initiatives.InitiativeWithRollup, error) {
	return s.items, s.err
}

func newContextTestHandler(backlogItems []backlog.BacklogItem, inits []initiatives.InitiativeWithRollup) *Handler {
	h := NewHandler("")
	h.SetBacklogLister(stubBacklogLister{items: backlogItems})
	h.SetInitiativesLister(stubInitiativesLister{items: inits})
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

func TestGetContext_ReturnsInitiativesOrphansAndRollup(t *testing.T) {
	inits := []initiatives.InitiativeWithRollup{
		{
			Initiative: initiatives.Initiative{Name: "audio-platform", Title: "Audio", Status: "active"},
			Rollup:     initiatives.RollupStatus{Total: 3, Completed: 1, Pending: 2},
			TargetScenarios: []string{"web-console"},
		},
		{
			Initiative: initiatives.Initiative{Name: "other-init", Title: "Other", Status: "active"},
			Rollup:     initiatives.RollupStatus{Total: 2, Completed: 0, Pending: 2},
			TargetScenarios: []string{"command-center"},
		},
	}
	items := []backlog.BacklogItem{
		// Orphan targeting web-console.
		makeItem("orphan-a", backlog.KindExecute, backlog.StatusBacklog, []string{"web-console"}, nil),
		// Assigned to an initiative — should NOT appear as orphan.
		{
			Name: "assigned-b", Kind: backlog.KindFix, Status: backlog.StatusBacklog,
			Initiative:      "audio-platform",
			AcceptanceAllow: []string{"scenarios/web-console/**"},
		},
		// Orphan targeting different scenario — should be excluded.
		makeItem("orphan-c", backlog.KindIdea, backlog.StatusBacklog, []string{"command-center"}, nil),
	}
	h := newContextTestHandler(items, inits)

	ctx := getContext(t, h, "web-console")

	if ctx.ScenarioName != "web-console" {
		t.Errorf("scenario_name = %q, want web-console", ctx.ScenarioName)
	}
	if len(ctx.Initiatives) != 1 || ctx.Initiatives[0].Initiative.Name != "audio-platform" {
		t.Errorf("initiatives = %+v, want [audio-platform]", ctx.Initiatives)
	}
	if len(ctx.OrphanItems) != 1 || ctx.OrphanItems[0].Name != "orphan-a" {
		t.Errorf("orphan_items = %+v, want [orphan-a]", ctx.OrphanItems)
	}
	// Rollup = initiative rollup (3/1/0/0/2) + orphan (1 pending)
	if ctx.Rollup.Total != 4 {
		t.Errorf("rollup.Total = %d, want 4", ctx.Rollup.Total)
	}
	if ctx.Rollup.Completed != 1 {
		t.Errorf("rollup.Completed = %d, want 1", ctx.Rollup.Completed)
	}
	if ctx.Rollup.Pending != 3 {
		t.Errorf("rollup.Pending = %d, want 3", ctx.Rollup.Pending)
	}
}

func TestGetContext_EmptyScenario_ReturnsWellFormedEmpty(t *testing.T) {
	h := newContextTestHandler(nil, nil)

	ctx := getContext(t, h, "no-such-scenario")

	if ctx.ScenarioName != "no-such-scenario" {
		t.Errorf("scenario_name = %q, want no-such-scenario", ctx.ScenarioName)
	}
	if len(ctx.Initiatives) != 0 {
		t.Errorf("expected empty initiatives, got %d", len(ctx.Initiatives))
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
		// Active fix in an initiative — must still appear in fixes.
		{
			Name: "fix-in-init", Title: "In initiative", Kind: backlog.KindFix,
			Status: backlog.StatusBacklog, Priority: 3, Updated: "2026-04-23T10:00:00Z",
			Initiative:      "audio-platform",
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
	if ctx.Fixes.Active[0].Initiative != "audio-platform" {
		t.Errorf("initiative not surfaced on fix: %+v", ctx.Fixes.Active[0])
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

func TestGetContext_InitiativesRollupAggregatesAllFields(t *testing.T) {
	inits := []initiatives.InitiativeWithRollup{
		{
			Initiative: initiatives.Initiative{Name: "one", Status: "active"},
			Rollup: initiatives.RollupStatus{
				Total:      5,
				Completed:  2,
				InProgress: 1,
				Failed:     0,
				Pending:    2,
				Archived:   1,
			},
			TargetScenarios: []string{"target"},
		},
		{
			Initiative: initiatives.Initiative{Name: "two", Status: "active"},
			Rollup: initiatives.RollupStatus{
				Total:      3,
				Completed:  1,
				InProgress: 0,
				Failed:     1,
				Pending:    1,
				Archived:   0,
			},
			TargetScenarios: []string{"target"},
		},
	}
	h := newContextTestHandler(nil, inits)

	ctx := getContext(t, h, "target")

	if ctx.Rollup.Total != 8 {
		t.Errorf("Total = %d, want 8", ctx.Rollup.Total)
	}
	if ctx.Rollup.Completed != 3 {
		t.Errorf("Completed = %d, want 3", ctx.Rollup.Completed)
	}
	if ctx.Rollup.InProgress != 1 {
		t.Errorf("InProgress = %d, want 1", ctx.Rollup.InProgress)
	}
	if ctx.Rollup.Failed != 1 {
		t.Errorf("Failed = %d, want 1", ctx.Rollup.Failed)
	}
	if ctx.Rollup.Pending != 3 {
		t.Errorf("Pending = %d, want 3", ctx.Rollup.Pending)
	}
	if ctx.Rollup.Archived != 1 {
		t.Errorf("Archived = %d, want 1", ctx.Rollup.Archived)
	}
}
