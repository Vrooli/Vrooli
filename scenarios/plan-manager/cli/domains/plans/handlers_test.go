package plans

import (
	"context"
	"encoding/json"
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
	return connect.NewResponse(&plansv1.GetPlanResponse{Plan: &sharedv1.Plan{Id: req.Msg.GetId(), Phases: []*sharedv1.Phase{{Id: "phase-1"}}}}), nil
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

func (r *plansRecorder) ListRelevantContext(_ context.Context, req *connect.Request[plansv1.ListRelevantContextRequest]) (*connect.Response[plansv1.ListRelevantContextResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*plansv1.ListRelevantContextResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&plansv1.ListRelevantContextResponse{}), nil
}

func (r *plansRecorder) UpdateRelevantContext(_ context.Context, req *connect.Request[plansv1.UpdateRelevantContextRequest]) (*connect.Response[plansv1.UpdateRelevantContextResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*plansv1.UpdateRelevantContextResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&plansv1.UpdateRelevantContextResponse{Plan: &sharedv1.Plan{Id: req.Msg.GetId()}}), nil
}

func (r *plansRecorder) RemoveRelevantContext(_ context.Context, req *connect.Request[plansv1.RemoveRelevantContextRequest]) (*connect.Response[plansv1.RemoveRelevantContextResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*plansv1.RemoveRelevantContextResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&plansv1.RemoveRelevantContextResponse{Plan: &sharedv1.Plan{Id: req.Msg.GetId()}}), nil
}

func (r *plansRecorder) ListReferences(_ context.Context, req *connect.Request[plansv1.ListReferencesRequest]) (*connect.Response[plansv1.ListReferencesResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*plansv1.ListReferencesResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&plansv1.ListReferencesResponse{}), nil
}

func (r *plansRecorder) UpdateReference(_ context.Context, req *connect.Request[plansv1.UpdateReferenceRequest]) (*connect.Response[plansv1.UpdateReferenceResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*plansv1.UpdateReferenceResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&plansv1.UpdateReferenceResponse{Plan: &sharedv1.Plan{Id: req.Msg.GetId()}}), nil
}

func (r *plansRecorder) RemoveReference(_ context.Context, req *connect.Request[plansv1.RemoveReferenceRequest]) (*connect.Response[plansv1.RemoveReferenceResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*plansv1.RemoveReferenceResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&plansv1.RemoveReferenceResponse{Plan: &sharedv1.Plan{Id: req.Msg.GetId()}}), nil
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

func (r *plansRecorder) LinkDependency(_ context.Context, req *connect.Request[plansv1.LinkDependencyRequest]) (*connect.Response[plansv1.LinkDependencyResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	return connect.NewResponse(&plansv1.LinkDependencyResponse{Plan: &sharedv1.Plan{Id: req.Msg.GetDependingPlanId()}}), nil
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

func (r *plansRecorder) ReconcilePlans(_ context.Context, req *connect.Request[plansv1.ReconcilePlansRequest]) (*connect.Response[plansv1.ReconcilePlansResponse], error) {
	r.record(req.Msg)
	if r.err != nil {
		return nil, r.err
	}
	if m, ok := r.resp.(*plansv1.ReconcilePlansResponse); ok && m != nil {
		return connect.NewResponse(m), nil
	}
	return connect.NewResponse(&plansv1.ReconcilePlansResponse{DryRun: req.Msg.GetDryRun()}), nil
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
		resp   proto.Message
		assert func(t *testing.T, req proto.Message)
	}{
		{
			// --include-archived is a proper boolean flag (manifest "bool": true):
			// bare presence sets it, no value required.
			name: "list maps status flag to ACTIVE enum and workspace", group: "plans", cmd: "list",
			argv: []string{"--status", "active", "--include-archived", "--workspace", "/workspace"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*plansv1.ListPlansRequest)
				require.Equal(t, sharedv1.PlanStatus_PLAN_STATUS_ACTIVE, m.GetStatus())
				require.True(t, m.GetIncludeArchived())
				require.Equal(t, "/workspace", m.GetWorkspace().GetRoot())
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
			name: "get passes positional id and workspace", group: "plans", cmd: "get",
			argv: []string{"plan-123", "--workspace", "/workspace"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*plansv1.GetPlanRequest)
				require.Equal(t, "plan-123", m.GetId())
				require.Equal(t, "/workspace", m.GetWorkspace().GetRoot())
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
				require.NotEmpty(t, p.GetWorkspaceRoot())
			},
		},
		{
			name: "update patches existing plan and exposes execution-grade repair fields", group: "plans", cmd: "update",
			resp: &plansv1.GetPlanResponse{Plan: &sharedv1.Plan{
				Id:                      "plan-7",
				Title:                   "Old title",
				Purpose:                 "existing purpose",
				TechnicalApproach:       "old approach",
				ChangeBoundary:          &sharedv1.ChangeBoundary{AcceptanceAllow: []string{"old/**"}},
				RegressionAnchor:        &sharedv1.RegressionAnchor{Strategy: "legacy_prose", BaselineName: "old anchor"},
				DefinitionOfDone:        "old done",
				ValidationStrategy:      "old validation",
				ProblemStatement:        "old problem",
				TargetOutcome:           "old outcome",
				FinalValidationCommands: []string{"old command"},
			}},
			argv: []string{
				"plan-7", "--title", "T2", "--dod", "d2",
				"--technical-approach", "repaired approach",
				"--change-allow", "scenarios/plan-manager/**,packages/proto/**",
				"--change-deny", "scenarios/other/**",
				"--anchor-strategy", "change_boundary",
				"--anchor-baseline", "repaired-baseline",
				"--anchor-command", "git diff --stat -- scenarios/plan-manager/**",
			},
			assert: func(t *testing.T, req proto.Message) {
				p := req.(*plansv1.UpdatePlanRequest).GetPlan()
				require.Equal(t, "plan-7", p.GetId())
				require.Equal(t, "T2", p.GetTitle())
				require.Equal(t, "d2", p.GetDefinitionOfDone())
				require.Equal(t, "existing purpose", p.GetPurpose(), "unsupplied authored fields must survive a CLI patch")
				require.Equal(t, "repaired approach", p.GetTechnicalApproach())
				require.Equal(t, []string{"scenarios/plan-manager/**", "packages/proto/**"}, p.GetChangeBoundary().GetAcceptanceAllow())
				require.Equal(t, []string{"scenarios/other/**"}, p.GetChangeBoundary().GetAcceptanceDeny())
				require.Equal(t, "change_boundary", p.GetRegressionAnchor().GetStrategy())
				require.Equal(t, "repaired-baseline", p.GetRegressionAnchor().GetBaselineName())
				require.Equal(t, []string{"git diff --stat -- scenarios/plan-manager/**"}, p.GetRegressionAnchor().GetCommands())
			},
		},
		{
			name: "archive passes id and workspace", group: "plans", cmd: "archive",
			argv: []string{"plan-9", "--workspace", "/workspace"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*plansv1.ArchivePlanRequest)
				require.Equal(t, "plan-9", m.GetId())
				require.Equal(t, "/workspace", m.GetWorkspace().GetRoot())
			},
		},
		{
			name: "render passes id workspace and compact mode", group: "plans", cmd: "render",
			argv: []string{"plan-r", "--workspace", "/workspace", "--compact"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*plansv1.RenderMarkdownRequest)
				require.Equal(t, "plan-r", m.GetId())
				require.Equal(t, "/workspace", m.GetWorkspace().GetRoot())
				require.True(t, m.GetCompact())
			},
		},
		{
			name: "context list passes id workspace and phase", group: "plans", cmd: "context-list",
			argv: []string{"plan-c", "--workspace", "/workspace", "--phase", "phase-1"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*plansv1.ListRelevantContextRequest)
				require.Equal(t, "plan-c", m.GetId())
				require.Equal(t, "/workspace", m.GetWorkspace().GetRoot())
				require.Equal(t, "phase-1", m.GetPhaseId())
			},
		},
		{
			name: "context update maps authored repair fields", group: "plans", cmd: "context-update",
			argv: []string{
				"plan-c", "item-1", "--workspace", "/workspace", "--phase", "phase-1",
				"--kind", "command", "--label", "Validate", "--reason", "repair reason",
				"--instruction", "Run validation", "--command", "plan-manager author validate sess-1",
				"--required", "--repeat", "phase_entry",
			},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*plansv1.UpdateRelevantContextRequest)
				require.Equal(t, "plan-c", m.GetId())
				require.Equal(t, "/workspace", m.GetWorkspace().GetRoot())
				require.Equal(t, "phase-1", m.GetPhaseId())
				require.Equal(t, "item-1", m.GetItemId())
				item := m.GetItem()
				require.Equal(t, "item-1", item.GetId())
				require.Equal(t, sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_COMMAND, item.GetKind())
				require.Equal(t, "Validate", item.GetLabel())
				require.Equal(t, "repair reason", item.GetReason())
				require.Equal(t, "Run validation", item.GetInstruction())
				require.Equal(t, "plan-manager author validate sess-1", item.GetCommand())
				require.True(t, item.GetRequired())
				require.Equal(t, sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_PHASE_ENTRY, item.GetRepeatPolicy())
				require.Equal(t, sharedv1.RelevantContextSource_RELEVANT_CONTEXT_SOURCE_AUTHORED, item.GetSource())
				require.Equal(t, sharedv1.RelevantContextStatus_RELEVANT_CONTEXT_STATUS_READY, item.GetStatus())
			},
		},
		{
			name: "context remove passes item id", group: "plans", cmd: "context-remove",
			argv: []string{"plan-c", "item-2", "--phase", "phase-1"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*plansv1.RemoveRelevantContextRequest)
				require.Equal(t, "plan-c", m.GetId())
				require.Equal(t, "item-2", m.GetItemId())
				require.Equal(t, "phase-1", m.GetPhaseId())
			},
		},
		{
			name: "reference list passes id workspace and phase", group: "plans", cmd: "reference-list",
			argv: []string{"plan-r", "--workspace", "/workspace", "--phase", "phase-2"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*plansv1.ListReferencesRequest)
				require.Equal(t, "plan-r", m.GetId())
				require.Equal(t, "/workspace", m.GetWorkspace().GetRoot())
				require.Equal(t, "phase-2", m.GetPhaseId())
			},
		},
		{
			name: "reference update maps authored repair fields", group: "plans", cmd: "reference-update",
			argv: []string{
				"plan-r", "ref-1", "--phase", "phase-2",
				"--kind", "doc", "--target", "docs/README.md", "--future", "--note", "operator note",
			},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*plansv1.UpdateReferenceRequest)
				require.Equal(t, "plan-r", m.GetId())
				require.Equal(t, "phase-2", m.GetPhaseId())
				require.Equal(t, "ref-1", m.GetReferenceId())
				ref := m.GetReference()
				require.Equal(t, "ref-1", ref.GetId())
				require.Equal(t, sharedv1.ReferenceKind_REFERENCE_KIND_DOC, ref.GetKind())
				require.Equal(t, "docs/README.md", ref.GetTarget())
				require.True(t, ref.GetFuture())
				require.Equal(t, "operator note", ref.GetNote())
			},
		},
		{
			name: "reference remove passes reference id", group: "plans", cmd: "reference-remove",
			argv: []string{"plan-r", "ref-2", "--phase", "phase-2"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*plansv1.RemoveReferenceRequest)
				require.Equal(t, "plan-r", m.GetId())
				require.Equal(t, "ref-2", m.GetReferenceId())
				require.Equal(t, "phase-2", m.GetPhaseId())
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
			name: "depend maps both positionals", group: "plans", cmd: "depend",
			argv: []string{"downstream", "upstream"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*plansv1.LinkDependencyRequest)
				require.Equal(t, "downstream", m.GetDependingPlanId())
				require.Equal(t, "upstream", m.GetDependencyPlanId())
			},
		},
		{
			name: "import maps source markdown and workspace", group: "plans", cmd: "import",
			argv: []string{"--source", "/tmp/p.md", "--markdown", "# Plan", "--workspace", "/workspace"},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*plansv1.ImportPlanRequest)
				require.Equal(t, "/tmp/p.md", m.GetSourcePath())
				require.Equal(t, "# Plan", m.GetMarkdown())
				require.Equal(t, "/workspace", m.GetWorkspace().GetRoot())
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
			name: "reconcile maps dry-run repair source intake and retirement flags", group: "plans", cmd: "reconcile",
			argv: []string{
				"--dry-run", "--repair-mirrors", "--source-intake", "--retire-sources", "--include-archived",
				"--conflict-policy", "report_only", "--source-docs-plans", "--workspace", "/workspace",
			},
			assert: func(t *testing.T, req proto.Message) {
				m := req.(*plansv1.ReconcilePlansRequest)
				require.True(t, m.GetDryRun())
				require.True(t, m.GetRepairMirrors())
				require.True(t, m.GetSourceIntake())
				require.True(t, m.GetRetireSources())
				require.True(t, m.GetIncludeArchived())
				require.Equal(t, plansv1.ReconcileConflictPolicy_RECONCILE_CONFLICT_POLICY_REPORT_ONLY, m.GetConflictPolicy())
				require.True(t, m.GetSourceDocsPlans())
				require.False(t, m.GetSourceRuntimeHomePlans())
				require.False(t, m.GetSourceRepoPlans())
				require.Equal(t, "/workspace", m.GetWorkspace().GetRoot())
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
			name: "phase add maps the canonical rich phase fields", group: "phase", cmd: "add",
			argv: []string{
				"plan-p", "--title", "Contract",
				"--affected-areas", "render.go,parse.go",
				"--steps", "Add section\nWire parser",
				"--expected-outputs", "comprehensive markdown",
				"--validation", "go test ./internal/plans",
				"--risks-hazards", "parser drift",
				"--handoff-notes", "rebuild ui/dist after",
			},
			assert: func(t *testing.T, req proto.Message) {
				ph := req.(*plansv1.AddPhaseRequest).GetPhase()
				require.Equal(t, []string{"render.go", "parse.go"}, ph.GetAffectedAreas())
				require.Equal(t, []string{"Add section", "Wire parser"}, ph.GetSteps())
				require.Equal(t, []string{"comprehensive markdown"}, ph.GetExpectedOutputs())
				require.Equal(t, "go test ./internal/plans", ph.GetValidation())
				require.Equal(t, []string{"parser drift"}, ph.GetRisksHazards())
				require.Equal(t, "rebuild ui/dist after", ph.GetHandoffNotes())
			},
		},
		{
			name: "phase update maps the canonical rich phase fields", group: "phase", cmd: "update",
			argv: []string{
				"plan-p", "phase-1", "--steps", "Only step", "--validation", "go test ./...",
			},
			assert: func(t *testing.T, req proto.Message) {
				ph := req.(*plansv1.UpdatePhaseRequest).GetPhase()
				require.Equal(t, "phase-1", ph.GetId())
				require.Equal(t, []string{"Only step"}, ph.GetSteps())
				require.Equal(t, "go test ./...", ph.GetValidation())
			},
		},
		{
			name: "phase update maps full-plan validation scope", group: "phase", cmd: "update",
			argv: []string{"plan-p", "phase-1", "--validation-scope", "full_plan:cross-scenario contract changed"},
			assert: func(t *testing.T, req proto.Message) {
				scope := req.(*plansv1.UpdatePhaseRequest).GetPhase().GetValidationScope()
				require.Equal(t, sharedv1.ValidationScopeMode_VALIDATION_SCOPE_MODE_FULL_PLAN, scope.GetMode())
				require.Equal(t, "cross-scenario contract changed", scope.GetRationale())
			},
		},
		{
			name: "phase add maps comma-separated list flags", group: "phase", cmd: "add",
			argv: []string{"plan-p", "--title", "Ph", "--context", "kind=doc;label=Testing docs;target=docs/TESTING.md;reason=Use server-owned wait protocol;instruction=Read before running tests,kind=command;command=prompt-manager skill read scientific-debugging;repeat=on_resume", "--reminders", "never stash", "--baseline-scope", "git-control-tower baseline diff --scenario x"},
			assert: func(t *testing.T, req proto.Message) {
				ph := req.(*plansv1.AddPhaseRequest).GetPhase()
				require.Empty(t, ph.GetRequiredReading(), "phase CLI should not author legacy required_reading")
				require.Len(t, ph.GetRelevantContext(), 2)
				require.Equal(t, sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_DOC, ph.GetRelevantContext()[0].GetKind())
				require.Equal(t, sharedv1.RelevantContextScope_RELEVANT_CONTEXT_SCOPE_PHASE, ph.GetRelevantContext()[0].GetScope())
				require.Equal(t, "Testing docs", ph.GetRelevantContext()[0].GetLabel())
				require.Equal(t, "docs/TESTING.md", ph.GetRelevantContext()[0].GetTarget())
				require.Equal(t, "Use server-owned wait protocol", ph.GetRelevantContext()[0].GetReason())
				require.Equal(t, "Read before running tests", ph.GetRelevantContext()[0].GetInstruction())
				require.True(t, ph.GetRelevantContext()[0].GetRequired())
				require.Equal(t, sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_PHASE_ENTRY, ph.GetRelevantContext()[0].GetRepeatPolicy())
				require.Equal(t, sharedv1.RelevantContextSource_RELEVANT_CONTEXT_SOURCE_AUTHORED, ph.GetRelevantContext()[0].GetSource())
				require.Equal(t, sharedv1.RelevantContextStatus_RELEVANT_CONTEXT_STATUS_READY, ph.GetRelevantContext()[0].GetStatus())
				require.Equal(t, sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_COMMAND, ph.GetRelevantContext()[1].GetKind())
				require.Equal(t, "prompt-manager skill read scientific-debugging", ph.GetRelevantContext()[1].GetCommand())
				require.Equal(t, []string{"prompt-manager", "skill", "read", "scientific-debugging"}, ph.GetRelevantContext()[1].GetArgv())
				require.Equal(t, sharedv1.RelevantContextRepeatPolicy_RELEVANT_CONTEXT_REPEAT_POLICY_ON_RESUME, ph.GetRelevantContext()[1].GetRepeatPolicy())
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
			rec := &plansRecorder{resp: tc.resp}
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

	var payload map[string]map[string]string
	require.NoError(t, json.Unmarshal([]byte(out), &payload))
	require.Equal(t, "plan-j", payload["plan"]["id"])
	require.Equal(t, "jj", payload["plan"]["slug"])
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

func errBoom() error { return &boomError{} }

type boomError struct{}

func (*boomError) Error() string { return "boom" }
