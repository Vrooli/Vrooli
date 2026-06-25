package plans

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans"
	plansconnect "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans/plans_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"

	"github.com/vrooli/cli-core/cliapp"
	clitest "plan-manager/cli/internal/testutil"
)

// plansRecorder is a fake PlansService that captures the last request the CLI
// handler built and returns a canned response message (or error). It lets the
// tests assert on the flag->proto-field mapping the handler performs, which is
// where a silent enum/typo regression would hide. resp holds the response
// *message* (e.g. *plansv1.GetPlanResponse); when nil/wrong-type a method
// returns its own minimal default.
type plansRecorder struct {
	plansconnect.UnimplementedPlansServiceHandler
	mu   sync.Mutex
	req  proto.Message
	resp proto.Message
	err  error
}

func (r *plansRecorder) record(req proto.Message) {
	r.mu.Lock()
	r.req = req
	r.mu.Unlock()
}

func (r *plansRecorder) lastRequest() proto.Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.req
}

func (r *plansRecorder) ListPlans(_ context.Context, req *connect.Request[plansv1.ListPlansRequest]) (*connect.Response[plansv1.ListPlansResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*plansv1.ListPlansResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&plansv1.ListPlansResponse{}), nil
}

func (r *plansRecorder) GetPlan(_ context.Context, req *connect.Request[plansv1.GetPlanRequest]) (*connect.Response[plansv1.GetPlanResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*plansv1.GetPlanResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&plansv1.GetPlanResponse{Plan: &sharedv1.Plan{Id: req.Msg.GetId()}}), nil
}

func (r *plansRecorder) CreatePlan(_ context.Context, req *connect.Request[plansv1.CreatePlanRequest]) (*connect.Response[plansv1.CreatePlanResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*plansv1.CreatePlanResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&plansv1.CreatePlanResponse{Plan: &sharedv1.Plan{Id: "plan-new"}}), nil
}

func (r *plansRecorder) UpdatePlan(_ context.Context, req *connect.Request[plansv1.UpdatePlanRequest]) (*connect.Response[plansv1.UpdatePlanResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	return connect.NewResponse(&plansv1.UpdatePlanResponse{Plan: req.Msg.GetPlan()}), nil
}

func (r *plansRecorder) ArchivePlan(_ context.Context, req *connect.Request[plansv1.ArchivePlanRequest]) (*connect.Response[plansv1.ArchivePlanResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	return connect.NewResponse(&plansv1.ArchivePlanResponse{Plan: &sharedv1.Plan{Id: req.Msg.GetId()}}), nil
}

func (r *plansRecorder) RenderMarkdown(_ context.Context, req *connect.Request[plansv1.RenderMarkdownRequest]) (*connect.Response[plansv1.RenderMarkdownResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*plansv1.RenderMarkdownResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&plansv1.RenderMarkdownResponse{}), nil
}

func (r *plansRecorder) AddPhase(_ context.Context, req *connect.Request[plansv1.AddPhaseRequest]) (*connect.Response[plansv1.AddPhaseResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	return connect.NewResponse(&plansv1.AddPhaseResponse{Plan: &sharedv1.Plan{Id: req.Msg.GetPlanId()}}), nil
}

func (r *plansRecorder) UpdatePhase(_ context.Context, req *connect.Request[plansv1.UpdatePhaseRequest]) (*connect.Response[plansv1.UpdatePhaseResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	return connect.NewResponse(&plansv1.UpdatePhaseResponse{Plan: &sharedv1.Plan{Id: req.Msg.GetPlanId()}}), nil
}

func (r *plansRecorder) GetGraph(_ context.Context, req *connect.Request[plansv1.GetGraphRequest]) (*connect.Response[plansv1.GetGraphResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*plansv1.GetGraphResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&plansv1.GetGraphResponse{}), nil
}

func (r *plansRecorder) LinkSupersession(_ context.Context, req *connect.Request[plansv1.LinkSupersessionRequest]) (*connect.Response[plansv1.LinkSupersessionResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	return connect.NewResponse(&plansv1.LinkSupersessionResponse{Plan: &sharedv1.Plan{Id: req.Msg.GetSupersedingPlanId()}}), nil
}

func (r *plansRecorder) ImportPlan(_ context.Context, req *connect.Request[plansv1.ImportPlanRequest]) (*connect.Response[plansv1.ImportPlanResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	return connect.NewResponse(&plansv1.ImportPlanResponse{Plan: &sharedv1.Plan{Id: "plan-imported"}}), nil
}

func (r *plansRecorder) MigratePlan(_ context.Context, req *connect.Request[plansv1.MigratePlanRequest]) (*connect.Response[plansv1.MigratePlanResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	return connect.NewResponse(&plansv1.MigratePlanResponse{Plan: &sharedv1.Plan{Id: req.Msg.GetId()}}), nil
}

func (r *plansRecorder) ListTemplates(_ context.Context, req *connect.Request[plansv1.ListTemplatesRequest]) (*connect.Response[plansv1.ListTemplatesResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*plansv1.ListTemplatesResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&plansv1.ListTemplatesResponse{}), nil
}

func (r *plansRecorder) CreateFromTemplate(_ context.Context, req *connect.Request[plansv1.CreateFromTemplateRequest]) (*connect.Response[plansv1.CreateFromTemplateResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	return connect.NewResponse(&plansv1.CreateFromTemplateResponse{Plan: &sharedv1.Plan{Id: "plan-from-template"}}), nil
}

func newPlansFixture(t *testing.T, rec *plansRecorder) (*cliapp.ScenarioApp, []cliapp.SubcommandGroup) {
	t.Helper()
	mux := http.NewServeMux()
	path, handler := plansconnect.NewPlansServiceHandler(rec)
	mux.Handle(path, handler)
	app := clitest.NewTestApp(t, mux)
	groups, err := Register(app, clitest.ReadManifest(t))
	require.NoError(t, err, "register plans groups against manifest")
	return app, groups
}

// TestPlansRequestMapping drives every covered verb end-to-end through the real
// manifest schema + parser + Connect client, and asserts the typed request the
// handler built. The status-enum rows are the highest-value: a flag->enum typo
// would silently downgrade the filter/transition to *_UNSPECIFIED.
func TestPlansRequestMapping(t *testing.T) {
	tests := []struct {
		name   string
		group  string
		cmd    string
		argv   []string
		assert func(t *testing.T, req proto.Message)
	}{
		{
			// --include-archived is a proper boolean flag (manifest "bool": true):
			// bare presence sets it, no value required.
			name: "list maps status flag to ACTIVE enum", group: "plans", cmd: "list",
			argv: []string{"--status", "active", "--include-archived"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*plansv1.ListPlansRequest)
				require.Equal(t, sharedv1.PlanStatus_PLAN_STATUS_ACTIVE, m.GetStatus())
				require.True(t, m.GetIncludeArchived())
			},
		},
		{
			name: "list with no status leaves filter UNSPECIFIED", group: "plans", cmd: "list",
			argv: []string{},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*plansv1.ListPlansRequest)
				require.Equal(t, sharedv1.PlanStatus_PLAN_STATUS_UNSPECIFIED, m.GetStatus())
				require.False(t, m.GetIncludeArchived())
			},
		},
		{
			name: "get passes positional id", group: "plans", cmd: "get",
			argv: []string{"plan-123"},
			assert: func(t *testing.T, req proto.Message) {
				require.Equal(t, "plan-123", req.(*plansv1.GetPlanRequest).GetId())
			},
		},
		{
			name: "create maps every authored flag", group: "plans", cmd: "create",
			argv: []string{
				"--title", "T", "--slug", "s", "--purpose", "p", "--scope", "sc",
				"--constraints", "c", "--non-goals", "ng", "--dod", "done",
			},
			assert: func(t *testing.T, req proto.Message) {
				p := req.(*plansv1.CreatePlanRequest).GetPlan()
				require.Equal(t, "T", p.GetTitle())
				require.Equal(t, "s", p.GetSlug())
				require.Equal(t, "p", p.GetPurpose())
				require.Equal(t, "sc", p.GetScope())
				require.Equal(t, "c", p.GetConstraints())
				require.Equal(t, "ng", p.GetNonGoals())
				require.Equal(t, "done", p.GetDefinitionOfDone())
			},
		},
		{
			name: "update carries id positional + flags", group: "plans", cmd: "update",
			argv: []string{"plan-7", "--title", "T2", "--dod", "d2"},
			assert: func(t *testing.T, req proto.Message) {
				p := req.(*plansv1.UpdatePlanRequest).GetPlan()
				require.Equal(t, "plan-7", p.GetId())
				require.Equal(t, "T2", p.GetTitle())
				require.Equal(t, "d2", p.GetDefinitionOfDone())
			},
		},
		{
			name: "archive passes id", group: "plans", cmd: "archive",
			argv: []string{"plan-9"},
			assert: func(t *testing.T, req proto.Message) {
				require.Equal(t, "plan-9", req.(*plansv1.ArchivePlanRequest).GetId())
			},
		},
		{
			name: "render passes id", group: "plans", cmd: "render",
			argv: []string{"plan-r"},
			assert: func(t *testing.T, req proto.Message) {
				require.Equal(t, "plan-r", req.(*plansv1.RenderMarkdownRequest).GetId())
			},
		},
		{
			name: "graph maps plan flag to plan_id", group: "plans", cmd: "graph",
			argv: []string{"--plan", "plan-g"},
			assert: func(t *testing.T, req proto.Message) {
				require.Equal(t, "plan-g", req.(*plansv1.GetGraphRequest).GetPlanId())
			},
		},
		{
			name: "link maps both positionals", group: "plans", cmd: "link",
			argv: []string{"new-plan", "old-plan"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*plansv1.LinkSupersessionRequest)
				require.Equal(t, "new-plan", m.GetSupersedingPlanId())
				require.Equal(t, "old-plan", m.GetSupersededPlanId())
			},
		},
		{
			name: "import maps source+markdown", group: "plans", cmd: "import",
			argv: []string{"--source", "/tmp/p.md", "--markdown", "# Plan"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*plansv1.ImportPlanRequest)
				require.Equal(t, "/tmp/p.md", m.GetSourcePath())
				require.Equal(t, "# Plan", m.GetMarkdown())
			},
		},
		{
			name: "migrate passes id", group: "plans", cmd: "migrate",
			argv: []string{"plan-m"},
			assert: func(t *testing.T, req proto.Message) {
				require.Equal(t, "plan-m", req.(*plansv1.MigratePlanRequest).GetId())
			},
		},
		{
			name: "phase add maps plan positional + flags", group: "phase", cmd: "add",
			argv: []string{"plan-p", "--title", "Ph", "--intent", "In", "--acceptance", "Ac"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*plansv1.AddPhaseRequest)
				require.Equal(t, "plan-p", m.GetPlanId())
				require.Equal(t, "Ph", m.GetPhase().GetTitle())
				require.Equal(t, "In", m.GetPhase().GetIntent())
				require.Equal(t, "Ac", m.GetPhase().GetAcceptance())
			},
		},
		{
			name: "phase add maps comma-separated list flags", group: "phase", cmd: "add",
			argv: []string{"plan-p", "--title", "Ph", "--required-reading", "docs/A.md, docs/B.md", "--reminders", "never stash", "--baseline-scope", "git-control-tower baseline diff --scenario x"},
			assert: func(t *testing.T, req proto.Message) {
				ph := req.(*plansv1.AddPhaseRequest).GetPhase()
				require.Equal(t, []string{"docs/A.md", "docs/B.md"}, ph.GetRequiredReading())
				require.Equal(t, []string{"never stash"}, ph.GetReminders())
				require.Equal(t, []string{"git-control-tower baseline diff --scenario x"}, ph.GetBaselineScope())
			},
		},
		{
			name: "phase update maps status flag to DONE enum", group: "phase", cmd: "update",
			argv: []string{"plan-p", "phase-1", "--title", "Ph2", "--status", "done"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*plansv1.UpdatePhaseRequest)
				require.Equal(t, "plan-p", m.GetPlanId())
				require.Equal(t, "phase-1", m.GetPhase().GetId())
				require.Equal(t, "Ph2", m.GetPhase().GetTitle())
				require.Equal(t, sharedv1.PhaseStatus_PHASE_STATUS_DONE, m.GetPhase().GetStatus())
			},
		},
		{
			name: "phase update with blocked status", group: "phase", cmd: "update",
			argv: []string{"plan-p", "phase-2", "--status", "blocked"},
			assert: func(t *testing.T, req proto.Message) {
				require.Equal(t, sharedv1.PhaseStatus_PHASE_STATUS_BLOCKED, req.(*plansv1.UpdatePhaseRequest).GetPhase().GetStatus())
			},
		},
		{
			name: "template new maps template positional + flags", group: "template", cmd: "new",
			argv: []string{"tmpl-cli", "--title", "FromTmpl", "--slug", "ft"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*plansv1.CreateFromTemplateRequest)
				require.Equal(t, "tmpl-cli", m.GetTemplateId())
				require.Equal(t, "FromTmpl", m.GetTitle())
				require.Equal(t, "ft", m.GetSlug())
			},
		},
		{
			name: "template list sends empty request", group: "template", cmd: "list",
			argv: []string{},
			assert: func(t *testing.T, req proto.Message) {
				require.IsType(t, &plansv1.ListTemplatesRequest{}, req)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := &plansRecorder{}
			app, groups := newPlansFixture(t, rec)
			cmd := clitest.FindCommand(t, groups, tc.group, tc.cmd)
			_, err := clitest.RunCommand(t, cmd, app, tc.argv...)
			require.NoError(t, err)
			req := rec.lastRequest()
			require.NotNil(t, req, "handler must have issued a request")
			tc.assert(t, req)
		})
	}
}

// TestPlansOutputRendering pins the human-readable rendering for the read +
// mutation verbs so a renderer regression (dropped slug, wrong count) is caught.
func TestPlansOutputRendering(t *testing.T) {
	t.Run("list renders count + plan lines", func(t *testing.T) {
		rec := &plansRecorder{resp: &plansv1.ListPlansResponse{Plans: []*sharedv1.Plan{
			{Slug: "alpha", Title: "Alpha Plan", Status: sharedv1.PlanStatus_PLAN_STATUS_ACTIVE},
			{Slug: "beta", Title: "Beta Plan", Status: sharedv1.PlanStatus_PLAN_STATUS_DRAFT},
		}}}
		app, groups := newPlansFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "plans", "list"), app)
		require.NoError(t, err)
		require.Contains(t, out, "Found 2 plan(s).")
		require.Contains(t, out, "alpha — Alpha Plan [active")
		require.Contains(t, out, "beta — Beta Plan [draft")
	})

	t.Run("get renders plan detail", func(t *testing.T) {
		rec := &plansRecorder{resp: &plansv1.GetPlanResponse{Plan: &sharedv1.Plan{
			Id: "plan-x", Slug: "xx", Title: "X Plan", Status: sharedv1.PlanStatus_PLAN_STATUS_COMPLETE,
			Phases: []*sharedv1.Phase{{Id: "ph1", Title: "Phase One", Status: sharedv1.PhaseStatus_PHASE_STATUS_DONE}},
		}}}
		app, groups := newPlansFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "plans", "get"), app, "plan-x")
		require.NoError(t, err)
		require.Contains(t, out, "id: plan-x")
		require.Contains(t, out, "status: complete")
		require.Contains(t, out, "phase 1: Phase One [done]")
	})

	t.Run("create renders mutation summary", func(t *testing.T) {
		rec := &plansRecorder{resp: &plansv1.CreatePlanResponse{Plan: &sharedv1.Plan{Id: "plan-created", Slug: "pc", Title: "Created"}}}
		app, groups := newPlansFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "plans", "create"), app, "--title", "Created")
		require.NoError(t, err)
		require.Contains(t, out, "Created plan plan-created.")
	})

	t.Run("render emits markdown", func(t *testing.T) {
		rec := &plansRecorder{resp: &plansv1.RenderMarkdownResponse{Markdown: "# Rendered Plan"}}
		app, groups := newPlansFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "plans", "render"), app, "plan-x")
		require.NoError(t, err)
		require.Contains(t, out, "# Rendered Plan")
	})

	t.Run("template list renders templates", func(t *testing.T) {
		rec := &plansRecorder{resp: &plansv1.ListTemplatesResponse{Templates: []*plansv1.PlanTemplate{
			{Id: "t1", Name: "CLI Template", Surface: "cli"},
		}}}
		app, groups := newPlansFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "template", "list"), app)
		require.NoError(t, err)
		require.Contains(t, out, "Found 1 template(s).")
		require.Contains(t, out, "t1 — CLI Template [cli]")
	})

	t.Run("graph renders edges", func(t *testing.T) {
		rec := &plansRecorder{resp: &plansv1.GetGraphResponse{Edges: []*sharedv1.PlanEdge{
			{FromPlanId: "a", ToPlanId: "b", Kind: "supersedes"},
		}}}
		app, groups := newPlansFixture(t, rec)
		out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "plans", "graph"), app)
		require.NoError(t, err)
		require.Contains(t, out, "a --supersedes--> b")
	})
}

// TestPlansJSONOutput verifies the --json path emits the proto wire shape.
func TestPlansJSONOutput(t *testing.T) {
	rec := &plansRecorder{resp: &plansv1.GetPlanResponse{Plan: &sharedv1.Plan{Id: "plan-j", Slug: "jj"}}}
	app, groups := newPlansFixture(t, rec)
	out, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "plans", "get"), app, "plan-j", "--json")
	require.NoError(t, err)
	require.Contains(t, out, `"id": "plan-j"`)
	require.Contains(t, out, `"slug": "jj"`)
}

// TestPlansErrorHandling covers the two failure shapes: a server-side Connect
// error must surface as a wrapped handler error, and a missing required
// positional must be rejected by the parser before the handler runs.
func TestPlansErrorHandling(t *testing.T) {
	t.Run("server error is wrapped with operation context", func(t *testing.T) {
		rec := &plansRecorder{err: connect.NewError(connect.CodeInternal, errBoom())}
		app, groups := newPlansFixture(t, rec)
		_, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "plans", "get"), app, "plan-x")
		require.Error(t, err)
		require.Contains(t, err.Error(), "get plan")
	})

	t.Run("missing required positional is a parser error", func(t *testing.T) {
		rec := &plansRecorder{}
		app, groups := newPlansFixture(t, rec)
		_, err := clitest.RunCommand(t, clitest.FindCommand(t, groups, "plans", "get"), app)
		require.Error(t, err)
		require.Contains(t, err.Error(), "id")
		require.Nil(t, rec.lastRequest(), "handler must not be reached when required positional is missing")
	})
}

// --- direct unit tests of the flag->enum + enum->label helpers ---

func TestPlanStatusFlag(t *testing.T) {
	cases := map[string]sharedv1.PlanStatus{
		"draft":    sharedv1.PlanStatus_PLAN_STATUS_DRAFT,
		"active":   sharedv1.PlanStatus_PLAN_STATUS_ACTIVE,
		"complete": sharedv1.PlanStatus_PLAN_STATUS_COMPLETE,
		"archived": sharedv1.PlanStatus_PLAN_STATUS_ARCHIVED,
		"  Active": sharedv1.PlanStatus_PLAN_STATUS_ACTIVE, // case-insensitive + trimmed
		"DRAFT":    sharedv1.PlanStatus_PLAN_STATUS_DRAFT,
		"":         sharedv1.PlanStatus_PLAN_STATUS_UNSPECIFIED,
		"bogus":    sharedv1.PlanStatus_PLAN_STATUS_UNSPECIFIED,
	}
	for in, want := range cases {
		require.Equalf(t, want, planStatusFlag(in), "planStatusFlag(%q)", in)
	}
}

func TestPhaseStatusFlag(t *testing.T) {
	cases := map[string]sharedv1.PhaseStatus{
		"todo":    sharedv1.PhaseStatus_PHASE_STATUS_TODO,
		"active":  sharedv1.PhaseStatus_PHASE_STATUS_ACTIVE,
		"done":    sharedv1.PhaseStatus_PHASE_STATUS_DONE,
		"blocked": sharedv1.PhaseStatus_PHASE_STATUS_BLOCKED,
		" Done ":  sharedv1.PhaseStatus_PHASE_STATUS_DONE,
		"":        sharedv1.PhaseStatus_PHASE_STATUS_UNSPECIFIED,
		"unknown": sharedv1.PhaseStatus_PHASE_STATUS_UNSPECIFIED,
	}
	for in, want := range cases {
		require.Equalf(t, want, phaseStatusFlag(in), "phaseStatusFlag(%q)", in)
	}
}

func TestPlanStatusLabel(t *testing.T) {
	require.Equal(t, "draft", planStatusLabel(sharedv1.PlanStatus_PLAN_STATUS_DRAFT))
	require.Equal(t, "active", planStatusLabel(sharedv1.PlanStatus_PLAN_STATUS_ACTIVE))
	require.Equal(t, "complete", planStatusLabel(sharedv1.PlanStatus_PLAN_STATUS_COMPLETE))
	require.Equal(t, "archived", planStatusLabel(sharedv1.PlanStatus_PLAN_STATUS_ARCHIVED))
	require.Equal(t, "unspecified", planStatusLabel(sharedv1.PlanStatus_PLAN_STATUS_UNSPECIFIED))
}

func TestPhaseStatusLabel(t *testing.T) {
	require.Equal(t, "todo", phaseStatusLabel(sharedv1.PhaseStatus_PHASE_STATUS_TODO))
	require.Equal(t, "active", phaseStatusLabel(sharedv1.PhaseStatus_PHASE_STATUS_ACTIVE))
	require.Equal(t, "done", phaseStatusLabel(sharedv1.PhaseStatus_PHASE_STATUS_DONE))
	require.Equal(t, "blocked", phaseStatusLabel(sharedv1.PhaseStatus_PHASE_STATUS_BLOCKED))
	// Unspecified falls back to "todo" (see handlers.go: a phase with no status
	// is treated as not-yet-started rather than "unspecified").
	require.Equal(t, "todo", phaseStatusLabel(sharedv1.PhaseStatus_PHASE_STATUS_UNSPECIFIED))
}

func errBoom() error { return &boomError{} }

type boomError struct{}

func (*boomError) Error() string { return "boom" }
