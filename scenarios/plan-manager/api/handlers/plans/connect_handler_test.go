package plans

import (
	"context"
	"errors"
	"io"
	"log"
	"testing"

	internalplans "plan-manager/internal/plans"

	"connectrpc.com/connect"

	"github.com/stretchr/testify/require"

	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
)

// fakePlansService is a minimal in-memory stand-in for internalplans.Service.
// Each method returns the configured value/err and records the arguments it was
// called with so handler field-passthrough can be asserted.
type fakePlansService struct {
	plan      internalplans.Plan
	plans     []internalplans.Plan
	edges     []internalplans.PlanEdge
	templates []internalplans.PlanTemplate
	markdown  string
	reconcile internalplans.ReconcileResult
	candidate internalplans.CandidateRevision
	preview   internalplans.CandidateRevisionPreview
	err       error
	context   []internalplans.RelevantContextItem
	refs      []internalplans.Reference

	gotListFilter            internalplans.ListFilter
	gotGetID                 string
	gotGetWorkspace          internalplans.WorkspaceScope
	gotCreate                internalplans.Plan
	gotUpdate                internalplans.Plan
	gotArchiveID             string
	gotArchiveWorkspace      internalplans.WorkspaceScope
	gotRenderID              string
	gotRenderWorkspace       internalplans.WorkspaceScope
	gotRenderOptions         internalplans.RenderOptions
	gotAddPhasePlanID        string
	gotAddPhaseWorkspace     internalplans.WorkspaceScope
	gotAddPhase              internalplans.Phase
	gotExtendBoundaryPlanID  string
	gotExtendBoundaryGlobs   []string
	gotUpdatePhasePlanID     string
	gotUpdatePhaseWorkspace  internalplans.WorkspaceScope
	gotUpdatePhase           internalplans.Phase
	gotRepairID              string
	gotRepairWorkspace       internalplans.WorkspaceScope
	gotRepairPhaseID         string
	gotRepairItemID          string
	gotRepairContext         internalplans.RelevantContextItem
	gotRepairReference       internalplans.Reference
	gotGraphPlanID           string
	gotSupersedingID         string
	gotSupersededID          string
	gotDependingID           string
	gotDependencyID          string
	gotImportPath            string
	gotImportMarkdown        string
	gotImportWorkspace       internalplans.WorkspaceScope
	gotMigrateID             string
	gotReconcile             internalplans.ReconcileRequest
	gotTemplateID            string
	gotTemplateTitle         string
	gotTemplateSlug          string
	gotCandidate             internalplans.CandidateRevision
	gotCandidateID           string
	gotCandidateBaseHash     string
	gotCandidateAcknowledged bool
	gotDiscardReason         string
}

func (f *fakePlansService) Create(_ context.Context, p internalplans.Plan) (internalplans.Plan, error) {
	f.gotCreate = p
	return f.plan, f.err
}

func (f *fakePlansService) Update(_ context.Context, p internalplans.Plan) (internalplans.Plan, error) {
	f.gotUpdate = p
	return f.plan, f.err
}

func (f *fakePlansService) CreateCandidate(_ context.Context, candidate internalplans.CandidateRevision) (internalplans.CandidateRevision, error) {
	f.gotCandidate = candidate
	return f.candidate, f.err
}

func (f *fakePlansService) GetCandidate(_ context.Context, id string) (internalplans.CandidateRevision, error) {
	f.gotCandidateID = id
	return f.candidate, f.err
}

func (f *fakePlansService) PreviewCandidate(_ context.Context, id string) (internalplans.CandidateRevisionPreview, error) {
	f.gotCandidateID = id
	return f.preview, f.err
}

func (f *fakePlansService) ValidateCandidate(_ context.Context, id string) (internalplans.CandidateRevisionPreview, error) {
	f.gotCandidateID = id
	return f.preview, f.err
}

func (f *fakePlansService) ApplyCandidate(_ context.Context, id, baseHash string, acknowledged bool) (internalplans.CandidateRevision, internalplans.Plan, internalplans.CandidateRevisionPreview, error) {
	f.gotCandidateID = id
	f.gotCandidateBaseHash = baseHash
	f.gotCandidateAcknowledged = acknowledged
	return f.candidate, f.plan, f.preview, f.err
}

func (f *fakePlansService) DiscardCandidate(_ context.Context, id, reason string) (internalplans.CandidateRevision, error) {
	f.gotCandidateID = id
	f.gotDiscardReason = reason
	return f.candidate, f.err
}

func (f *fakePlansService) Get(_ context.Context, idOrSlug string, workspace internalplans.WorkspaceScope) (internalplans.Plan, error) {
	f.gotGetID = idOrSlug
	f.gotGetWorkspace = workspace
	return f.plan, f.err
}

func (f *fakePlansService) List(_ context.Context, filter internalplans.ListFilter) ([]internalplans.Plan, error) {
	f.gotListFilter = filter
	return f.plans, f.err
}

func (f *fakePlansService) Archive(_ context.Context, idOrSlug string, workspace internalplans.WorkspaceScope) (internalplans.Plan, error) {
	f.gotArchiveID = idOrSlug
	f.gotArchiveWorkspace = workspace
	return f.plan, f.err
}

func (f *fakePlansService) Render(_ context.Context, idOrSlug string, workspace internalplans.WorkspaceScope, opts internalplans.RenderOptions) (internalplans.RenderResult, error) {
	f.gotRenderID = idOrSlug
	f.gotRenderWorkspace = workspace
	f.gotRenderOptions = opts
	return internalplans.RenderResult{
		Markdown:        f.markdown,
		Mirror:          internalplans.RenderedPlanMirror{Path: "/tmp/rendered.md", Status: internalplans.RenderedMirrorStatusFresh},
		Repaired:        true,
		QualityStatus:   "pass",
		QualityFindings: []string{"ok"},
	}, f.err
}

func (f *fakePlansService) AddPhase(_ context.Context, planID string, workspace internalplans.WorkspaceScope, phase internalplans.Phase) (internalplans.Plan, error) {
	f.gotAddPhasePlanID = planID
	f.gotAddPhaseWorkspace = workspace
	f.gotAddPhase = phase
	return f.plan, f.err
}

func (f *fakePlansService) AddPhaseWithImpact(ctx context.Context, planID string, workspace internalplans.WorkspaceScope, phase internalplans.Phase, _ bool) (internalplans.Plan, internalplans.MutationImpact, error) {
	p, err := f.AddPhase(ctx, planID, workspace, phase)
	return p, internalplans.MutationImpact{BeforeGrade: "pass", AfterGrade: "pass"}, err
}

func (f *fakePlansService) ExtendChangeBoundary(_ context.Context, planID string, _ internalplans.WorkspaceScope, globs []string) (internalplans.Plan, []string, error) {
	f.gotExtendBoundaryPlanID, f.gotExtendBoundaryGlobs = planID, globs
	return f.plan, globs, f.err
}

func (f *fakePlansService) UpdatePhase(_ context.Context, planID string, workspace internalplans.WorkspaceScope, phase internalplans.Phase) (internalplans.Plan, error) {
	f.gotUpdatePhasePlanID = planID
	f.gotUpdatePhaseWorkspace = workspace
	f.gotUpdatePhase = phase
	return f.plan, f.err
}

func (f *fakePlansService) ReplacePhaseWithImpact(ctx context.Context, planID string, workspace internalplans.WorkspaceScope, phase internalplans.Phase, _ bool) (internalplans.Plan, internalplans.MutationImpact, error) {
	p, err := f.UpdatePhase(ctx, planID, workspace, phase)
	return p, internalplans.MutationImpact{BeforeGrade: "pass", AfterGrade: "pass"}, err
}

func (f *fakePlansService) PatchPhase(ctx context.Context, planID string, workspace internalplans.WorkspaceScope, phase internalplans.Phase, _ []string, _ bool) (internalplans.Plan, internalplans.MutationImpact, error) {
	p, err := f.UpdatePhase(ctx, planID, workspace, phase)
	return p, internalplans.MutationImpact{BeforeGrade: "pass", AfterGrade: "pass"}, err
}

func (f *fakePlansService) ListRelevantContext(_ context.Context, idOrSlug string, workspace internalplans.WorkspaceScope, phaseID string) ([]internalplans.RelevantContextItem, error) {
	f.gotRepairID = idOrSlug
	f.gotRepairWorkspace = workspace
	f.gotRepairPhaseID = phaseID
	return f.context, f.err
}

func (f *fakePlansService) AddRelevantContext(_ context.Context, idOrSlug string, workspace internalplans.WorkspaceScope, phaseID string, item internalplans.RelevantContextItem, _ bool) (internalplans.Plan, internalplans.MutationImpact, error) {
	f.gotRepairID = idOrSlug
	f.gotRepairWorkspace = workspace
	f.gotRepairPhaseID = phaseID
	f.gotRepairContext = item
	return f.plan, internalplans.MutationImpact{BeforeGrade: "pass", AfterGrade: "pass"}, f.err
}

func (f *fakePlansService) UpdateRelevantContext(_ context.Context, idOrSlug string, workspace internalplans.WorkspaceScope, phaseID, itemID string, item internalplans.RelevantContextItem) (internalplans.Plan, error) {
	f.gotRepairID = idOrSlug
	f.gotRepairWorkspace = workspace
	f.gotRepairPhaseID = phaseID
	f.gotRepairItemID = itemID
	f.gotRepairContext = item
	return f.plan, f.err
}

func (f *fakePlansService) UpdateRelevantContextWithImpact(ctx context.Context, idOrSlug string, workspace internalplans.WorkspaceScope, phaseID, itemID string, item internalplans.RelevantContextItem, _ bool) (internalplans.Plan, internalplans.MutationImpact, error) {
	p, err := f.UpdateRelevantContext(ctx, idOrSlug, workspace, phaseID, itemID, item)
	return p, internalplans.MutationImpact{BeforeGrade: "pass", AfterGrade: "pass"}, err
}

func (f *fakePlansService) RemoveRelevantContext(_ context.Context, idOrSlug string, workspace internalplans.WorkspaceScope, phaseID, itemID string) (internalplans.Plan, error) {
	f.gotRepairID = idOrSlug
	f.gotRepairWorkspace = workspace
	f.gotRepairPhaseID = phaseID
	f.gotRepairItemID = itemID
	return f.plan, f.err
}

func (f *fakePlansService) RemoveRelevantContextWithImpact(ctx context.Context, idOrSlug string, workspace internalplans.WorkspaceScope, phaseID, itemID string, _ bool) (internalplans.Plan, internalplans.MutationImpact, error) {
	p, err := f.RemoveRelevantContext(ctx, idOrSlug, workspace, phaseID, itemID)
	return p, internalplans.MutationImpact{BeforeGrade: "pass", AfterGrade: "pass"}, err
}

func (f *fakePlansService) ListReferences(_ context.Context, idOrSlug string, workspace internalplans.WorkspaceScope, phaseID string) ([]internalplans.Reference, error) {
	f.gotRepairID = idOrSlug
	f.gotRepairWorkspace = workspace
	f.gotRepairPhaseID = phaseID
	return f.refs, f.err
}

func (f *fakePlansService) AddReference(_ context.Context, idOrSlug string, workspace internalplans.WorkspaceScope, phaseID string, ref internalplans.Reference, _ bool) (internalplans.Plan, internalplans.MutationImpact, error) {
	f.gotRepairID = idOrSlug
	f.gotRepairWorkspace = workspace
	f.gotRepairPhaseID = phaseID
	f.gotRepairReference = ref
	return f.plan, internalplans.MutationImpact{BeforeGrade: "pass", AfterGrade: "pass"}, f.err
}

func (f *fakePlansService) UpdateReference(_ context.Context, idOrSlug string, workspace internalplans.WorkspaceScope, phaseID, referenceID string, ref internalplans.Reference) (internalplans.Plan, error) {
	f.gotRepairID = idOrSlug
	f.gotRepairWorkspace = workspace
	f.gotRepairPhaseID = phaseID
	f.gotRepairItemID = referenceID
	f.gotRepairReference = ref
	return f.plan, f.err
}

func (f *fakePlansService) UpdateReferenceWithImpact(ctx context.Context, idOrSlug string, workspace internalplans.WorkspaceScope, phaseID, referenceID string, ref internalplans.Reference, _ bool) (internalplans.Plan, internalplans.MutationImpact, error) {
	p, err := f.UpdateReference(ctx, idOrSlug, workspace, phaseID, referenceID, ref)
	return p, internalplans.MutationImpact{BeforeGrade: "pass", AfterGrade: "pass"}, err
}

func (f *fakePlansService) RemoveReference(_ context.Context, idOrSlug string, workspace internalplans.WorkspaceScope, phaseID, referenceID string) (internalplans.Plan, error) {
	f.gotRepairID = idOrSlug
	f.gotRepairWorkspace = workspace
	f.gotRepairPhaseID = phaseID
	f.gotRepairItemID = referenceID
	return f.plan, f.err
}

func (f *fakePlansService) RemoveReferenceWithImpact(ctx context.Context, idOrSlug string, workspace internalplans.WorkspaceScope, phaseID, referenceID string, _ bool) (internalplans.Plan, internalplans.MutationImpact, error) {
	p, err := f.RemoveReference(ctx, idOrSlug, workspace, phaseID, referenceID)
	return p, internalplans.MutationImpact{BeforeGrade: "pass", AfterGrade: "pass"}, err
}

func (f *fakePlansService) GetGraph(_ context.Context, planID string) ([]internalplans.PlanEdge, error) {
	f.gotGraphPlanID = planID
	return f.edges, f.err
}

func (f *fakePlansService) LinkSupersession(_ context.Context, supersedingID, supersededID string) (internalplans.Plan, error) {
	f.gotSupersedingID = supersedingID
	f.gotSupersededID = supersededID
	return f.plan, f.err
}

func (f *fakePlansService) LinkDependency(_ context.Context, dependingID, dependencyID string) (internalplans.Plan, error) {
	f.gotDependingID = dependingID
	f.gotDependencyID = dependencyID
	return f.plan, f.err
}

func (f *fakePlansService) ListTemplates(_ context.Context) ([]internalplans.PlanTemplate, error) {
	return f.templates, f.err
}

func (f *fakePlansService) CreateFromTemplate(_ context.Context, templateID, title, slug string) (internalplans.Plan, error) {
	f.gotTemplateID = templateID
	f.gotTemplateTitle = title
	f.gotTemplateSlug = slug
	return f.plan, f.err
}

func (f *fakePlansService) Import(_ context.Context, sourcePath, markdown, title, slug string, workspace internalplans.WorkspaceScope) (internalplans.Plan, error) {
	f.gotImportPath = sourcePath
	f.gotImportMarkdown = markdown
	f.gotImportWorkspace = workspace
	f.gotCreate.Title = title
	f.gotCreate.Slug = slug
	return f.plan, f.err
}

func (f *fakePlansService) ImportSuperseding(ctx context.Context, sourcePath, markdown, title, slug string, workspace internalplans.WorkspaceScope, _ string) (internalplans.Plan, internalplans.Plan, error) {
	p, err := f.Import(ctx, sourcePath, markdown, title, slug, workspace)
	return p, internalplans.Plan{ID: "superseded", Status: internalplans.PlanStatusArchived}, err
}

func (f *fakePlansService) Migrate(_ context.Context, idOrSlug string) (internalplans.Plan, error) {
	f.gotMigrateID = idOrSlug
	return f.plan, f.err
}

func (f *fakePlansService) Reconcile(_ context.Context, req internalplans.ReconcileRequest) (internalplans.ReconcileResult, error) {
	f.gotReconcile = req
	return f.reconcile, f.err
}

var _ internalplans.Service = (*fakePlansService)(nil)

func newPlansHandler(svc internalplans.Service) *connectHandler {
	return NewConnectHandler(Deps{Service: svc, Logger: log.New(io.Discard, "", 0)})
}

func TestListPlansSuccess(t *testing.T) {
	svc := &fakePlansService{plans: []internalplans.Plan{
		{ID: "p1", Status: internalplans.PlanStatusActive},
		{ID: "p2", Status: internalplans.PlanStatusComplete},
	}}
	h := newPlansHandler(svc)

	resp, err := h.ListPlans(context.Background(), connect.NewRequest(&plansv1.ListPlansRequest{
		Status:          sharedv1.PlanStatus_PLAN_STATUS_ACTIVE,
		IncludeArchived: true,
		Workspace:       &plansv1.WorkspaceScope{Root: "/workspace"},
	}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetPlans(), 2)
	require.Equal(t, "p1", resp.Msg.GetPlans()[0].GetId())
	require.Equal(t, sharedv1.PlanStatus_PLAN_STATUS_COMPLETE, resp.Msg.GetPlans()[1].GetStatus())
	// Filter is translated from the proto request.
	require.Equal(t, internalplans.PlanStatusActive, svc.gotListFilter.Status)
	require.True(t, svc.gotListFilter.IncludeArchived)
	require.Equal(t, "/workspace", svc.gotListFilter.WorkspaceRoot)
}

func TestGetPlanSuccess(t *testing.T) {
	svc := &fakePlansService{plan: internalplans.Plan{ID: "p1", Title: "Hello", Status: internalplans.PlanStatusDraft}}
	h := newPlansHandler(svc)

	resp, err := h.GetPlan(context.Background(), connect.NewRequest(&plansv1.GetPlanRequest{Id: "p1", Workspace: &plansv1.WorkspaceScope{Root: "/workspace"}}))
	require.NoError(t, err)
	require.Equal(t, "p1", resp.Msg.GetPlan().GetId())
	require.Equal(t, "Hello", resp.Msg.GetPlan().GetTitle())
	require.Equal(t, sharedv1.PlanStatus_PLAN_STATUS_DRAFT, resp.Msg.GetPlan().GetStatus())
	require.Equal(t, "p1", svc.gotGetID, "handler must forward the request id to the service")
	require.Equal(t, "/workspace", svc.gotGetWorkspace.Root)
}

func TestCreatePlanForwardsAuthoredFields(t *testing.T) {
	svc := &fakePlansService{plan: internalplans.Plan{ID: "new-id"}}
	h := newPlansHandler(svc)

	resp, err := h.CreatePlan(context.Background(), connect.NewRequest(&plansv1.CreatePlanRequest{
		Plan: &sharedv1.Plan{Title: "Authored", Slug: "authored"},
	}))
	require.NoError(t, err)
	require.Equal(t, "new-id", resp.Msg.GetPlan().GetId())
	require.Equal(t, "Authored", svc.gotCreate.Title, "handler must translate the proto plan into the domain plan")
	require.Equal(t, "authored", svc.gotCreate.Slug)
}

func TestUpdatePlanForwardsAuthoredFields(t *testing.T) {
	svc := &fakePlansService{plan: internalplans.Plan{ID: "p1", Title: "Updated", Status: internalplans.PlanStatusActive}}
	h := newPlansHandler(svc)

	resp, err := h.UpdatePlan(context.Background(), connect.NewRequest(&plansv1.UpdatePlanRequest{
		Plan: &sharedv1.Plan{Id: "p1", Title: "Updated"},
	}))
	require.NoError(t, err)
	require.Equal(t, "p1", resp.Msg.GetPlan().GetId())
	require.Equal(t, "Updated", resp.Msg.GetPlan().GetTitle())
	require.Equal(t, "p1", svc.gotUpdate.ID, "handler must translate the proto plan into the domain plan")
	require.Equal(t, "Updated", svc.gotUpdate.Title)
}

func TestUpdatePlanForwardsBaselineSet(t *testing.T) {
	svc := &fakePlansService{plan: internalplans.Plan{ID: "p1"}}
	h := newPlansHandler(svc)
	baseline := &sharedv1.BaselineSetIntent{Name: "before", ScenarioTargets: []string{"plan-manager"}, CapturePolicy: "execution_start", Compatibility: "baseline_set"}

	_, err := h.UpdatePlan(context.Background(), connect.NewRequest(&plansv1.UpdatePlanRequest{Plan: &sharedv1.Plan{Id: "p1", BaselineSet: baseline}}))
	require.NoError(t, err)
	require.Equal(t, "before", svc.gotUpdate.BaselineSet.Name)
	require.Equal(t, []string{"plan-manager"}, svc.gotUpdate.BaselineSet.ScenarioTargets)
	require.Equal(t, "baseline_set", svc.gotUpdate.BaselineSet.Compatibility)
}

func TestMigratePlanSuccess(t *testing.T) {
	svc := &fakePlansService{plan: internalplans.Plan{ID: "p1"}}
	h := newPlansHandler(svc)

	resp, err := h.MigratePlan(context.Background(), connect.NewRequest(&plansv1.MigratePlanRequest{Id: "p1"}))
	require.NoError(t, err)
	require.Equal(t, "p1", resp.Msg.GetPlan().GetId())
	require.Equal(t, "p1", svc.gotMigrateID)
}

func TestReconcilePlansForwardsRequest(t *testing.T) {
	svc := &fakePlansService{reconcile: internalplans.ReconcileResult{
		DryRun: true,
		Items: []internalplans.ReconcileItem{{
			Action:                  internalplans.ReconcileActionImportPlanned,
			Title:                   "Legacy",
			SourcePath:              "docs/plans/legacy.md",
			SourceUntouched:         true,
			SourceRetirementPlanned: true,
		}},
	}}
	h := newPlansHandler(svc)

	resp, err := h.ReconcilePlans(context.Background(), connect.NewRequest(&plansv1.ReconcilePlansRequest{
		DryRun:          true,
		RepairMirrors:   true,
		SourceIntake:    true,
		RetireSources:   true,
		ConflictPolicy:  plansv1.ReconcileConflictPolicy_RECONCILE_CONFLICT_POLICY_REPORT_ONLY,
		SourceDocsPlans: true,
		Workspace:       &plansv1.WorkspaceScope{Root: "/workspace"},
	}))
	require.NoError(t, err)
	require.True(t, svc.gotReconcile.DryRun)
	require.True(t, svc.gotReconcile.RepairMirrors)
	require.True(t, svc.gotReconcile.SourceIntake)
	require.True(t, svc.gotReconcile.RetireSources)
	require.Equal(t, internalplans.ReconcileConflictReportOnly, svc.gotReconcile.ConflictPolicy)
	require.True(t, svc.gotReconcile.SourceDocsPlans)
	require.Equal(t, "/workspace", svc.gotReconcile.Workspace.Root)
	require.Len(t, resp.Msg.GetItems(), 1)
	require.Equal(t, plansv1.ReconcileAction_RECONCILE_ACTION_IMPORT_PLANNED, resp.Msg.GetItems()[0].GetAction())
	require.True(t, resp.Msg.GetItems()[0].GetSourceUntouched())
	require.True(t, resp.Msg.GetItems()[0].GetSourceRetirementPlanned())
}

func TestArchivePlanSuccess(t *testing.T) {
	svc := &fakePlansService{plan: internalplans.Plan{ID: "p1", Status: internalplans.PlanStatusArchived}}
	h := newPlansHandler(svc)

	resp, err := h.ArchivePlan(context.Background(), connect.NewRequest(&plansv1.ArchivePlanRequest{Id: "p1", Workspace: &plansv1.WorkspaceScope{Root: "/workspace"}}))
	require.NoError(t, err)
	require.Equal(t, sharedv1.PlanStatus_PLAN_STATUS_ARCHIVED, resp.Msg.GetPlan().GetStatus())
	require.Equal(t, "p1", svc.gotArchiveID)
	require.Equal(t, "/workspace", svc.gotArchiveWorkspace.Root)
}

func TestRenderMarkdownSuccess(t *testing.T) {
	svc := &fakePlansService{markdown: "# Title\n"}
	h := newPlansHandler(svc)

	resp, err := h.RenderMarkdown(context.Background(), connect.NewRequest(&plansv1.RenderMarkdownRequest{Id: "p1", Workspace: &plansv1.WorkspaceScope{Root: "/workspace"}, Compact: true}))
	require.NoError(t, err)
	require.Equal(t, "# Title\n", resp.Msg.GetMarkdown())
	require.Equal(t, "pass", resp.Msg.GetQualityStatus())
	require.Equal(t, []string{"ok"}, resp.Msg.GetQualityFindings())
	require.Equal(t, "p1", svc.gotRenderID)
	require.Equal(t, "/workspace", svc.gotRenderWorkspace.Root)
	require.True(t, svc.gotRenderOptions.Compact)
}

func TestAddPhaseSuccess(t *testing.T) {
	svc := &fakePlansService{plan: internalplans.Plan{ID: "p1"}}
	h := newPlansHandler(svc)

	resp, err := h.AddPhase(context.Background(), connect.NewRequest(&plansv1.AddPhaseRequest{
		PlanId:    "p1",
		Workspace: &plansv1.WorkspaceScope{Root: "/workspace"},
		Phase:     &sharedv1.Phase{Title: "New Phase", Status: sharedv1.PhaseStatus_PHASE_STATUS_TODO},
	}))
	require.NoError(t, err)
	require.Equal(t, "p1", resp.Msg.GetPlan().GetId())
	require.Equal(t, "p1", svc.gotAddPhasePlanID)
	require.Equal(t, "/workspace", svc.gotAddPhaseWorkspace.Root)
	require.Equal(t, "New Phase", svc.gotAddPhase.Title)
	require.Equal(t, internalplans.PhaseStatusTodo, svc.gotAddPhase.Status)
}

func TestRelevantContextRepairRPCs(t *testing.T) {
	svc := &fakePlansService{
		plan: internalplans.Plan{ID: "p1"},
		context: []internalplans.RelevantContextItem{{
			ID:      "ctx1",
			Kind:    internalplans.RelevantContextCommand,
			Command: "plan-manager author validate s1",
		}},
	}
	h := newPlansHandler(svc)

	listResp, err := h.ListRelevantContext(context.Background(), connect.NewRequest(&plansv1.ListRelevantContextRequest{
		Id:        "p1",
		Workspace: &plansv1.WorkspaceScope{Root: "/workspace"},
		PhaseId:   "phase-1",
	}))
	require.NoError(t, err)
	require.Len(t, listResp.Msg.GetItems(), 1)
	require.Equal(t, "ctx1", listResp.Msg.GetItems()[0].GetId())
	require.Equal(t, "p1", svc.gotRepairID)
	require.Equal(t, "/workspace", svc.gotRepairWorkspace.Root)
	require.Equal(t, "phase-1", svc.gotRepairPhaseID)

	updateResp, err := h.UpdateRelevantContext(context.Background(), connect.NewRequest(&plansv1.UpdateRelevantContextRequest{
		Id:      "p1",
		PhaseId: "phase-1",
		ItemId:  "ctx1",
		Item: &sharedv1.RelevantContextItem{
			Kind:    sharedv1.RelevantContextKind_RELEVANT_CONTEXT_KIND_COMMAND,
			Command: "plan-manager author validate s2",
		},
	}))
	require.NoError(t, err)
	require.Equal(t, "p1", updateResp.Msg.GetPlan().GetId())
	require.Equal(t, "ctx1", svc.gotRepairItemID)
	require.Equal(t, "plan-manager author validate s2", svc.gotRepairContext.Command)

	_, err = h.RemoveRelevantContext(context.Background(), connect.NewRequest(&plansv1.RemoveRelevantContextRequest{Id: "p1", ItemId: "ctx1"}))
	require.NoError(t, err)
	require.Equal(t, "ctx1", svc.gotRepairItemID)
}

func TestReferenceRepairRPCs(t *testing.T) {
	svc := &fakePlansService{
		plan: internalplans.Plan{ID: "p1"},
		refs: []internalplans.Reference{{
			ID:     "ref1",
			Kind:   internalplans.ReferenceCode,
			Target: "old.go",
		}},
	}
	h := newPlansHandler(svc)

	listResp, err := h.ListReferences(context.Background(), connect.NewRequest(&plansv1.ListReferencesRequest{Id: "p1", PhaseId: "phase-1"}))
	require.NoError(t, err)
	require.Len(t, listResp.Msg.GetReferences(), 1)
	require.Equal(t, "ref1", listResp.Msg.GetReferences()[0].GetId())

	updateResp, err := h.UpdateReference(context.Background(), connect.NewRequest(&plansv1.UpdateReferenceRequest{
		Id:          "p1",
		PhaseId:     "phase-1",
		ReferenceId: "ref1",
		Reference: &sharedv1.Reference{
			Kind:   sharedv1.ReferenceKind_REFERENCE_KIND_CODE,
			Target: "new.go",
		},
	}))
	require.NoError(t, err)
	require.Equal(t, "p1", updateResp.Msg.GetPlan().GetId())
	require.Equal(t, "ref1", svc.gotRepairItemID)
	require.Equal(t, internalplans.ReferenceCode, svc.gotRepairReference.Kind)
	require.Equal(t, "new.go", svc.gotRepairReference.Target)

	_, err = h.RemoveReference(context.Background(), connect.NewRequest(&plansv1.RemoveReferenceRequest{Id: "p1", PhaseId: "phase-1", ReferenceId: "ref1"}))
	require.NoError(t, err)
	require.Equal(t, "ref1", svc.gotRepairItemID)
}

func TestGetGraphSuccess(t *testing.T) {
	svc := &fakePlansService{edges: []internalplans.PlanEdge{
		{FromPlanID: "a", ToPlanID: "b", Kind: "supersedes"},
	}}
	h := newPlansHandler(svc)

	resp, err := h.GetGraph(context.Background(), connect.NewRequest(&plansv1.GetGraphRequest{PlanId: "a"}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetEdges(), 1)
	require.Equal(t, "a", resp.Msg.GetEdges()[0].GetFromPlanId())
	require.Equal(t, "supersedes", resp.Msg.GetEdges()[0].GetKind())
	require.Equal(t, "a", svc.gotGraphPlanID)
}

func TestLinkSupersessionForwardsBothIDs(t *testing.T) {
	svc := &fakePlansService{plan: internalplans.Plan{ID: "super"}}
	h := newPlansHandler(svc)

	_, err := h.LinkSupersession(context.Background(), connect.NewRequest(&plansv1.LinkSupersessionRequest{
		SupersedingPlanId: "super",
		SupersededPlanId:  "old",
	}))
	require.NoError(t, err)
	require.Equal(t, "super", svc.gotSupersedingID)
	require.Equal(t, "old", svc.gotSupersededID)
}

func TestLinkDependencyForwardsBothIDs(t *testing.T) {
	svc := &fakePlansService{plan: internalplans.Plan{ID: "depending"}}
	h := newPlansHandler(svc)

	_, err := h.LinkDependency(context.Background(), connect.NewRequest(&plansv1.LinkDependencyRequest{
		DependingPlanId:  "depending",
		DependencyPlanId: "dependency",
	}))
	require.NoError(t, err)
	require.Equal(t, "depending", svc.gotDependingID)
	require.Equal(t, "dependency", svc.gotDependencyID)
}

func TestImportPlanForwardsArgs(t *testing.T) {
	svc := &fakePlansService{plan: internalplans.Plan{ID: "imported"}}
	h := newPlansHandler(svc)

	resp, err := h.ImportPlan(context.Background(), connect.NewRequest(&plansv1.ImportPlanRequest{
		SourcePath: "docs/plan.md",
		Markdown:   "# Plan",
		Title:      "Override",
		Slug:       "override",
		Workspace:  &plansv1.WorkspaceScope{Root: "/workspace"},
	}))
	require.NoError(t, err)
	require.Equal(t, "imported", resp.Msg.GetPlan().GetId())
	require.Equal(t, "docs/plan.md", svc.gotImportPath)
	require.Equal(t, "# Plan", svc.gotImportMarkdown)
	require.Equal(t, "/workspace", svc.gotImportWorkspace.Root)
	require.Equal(t, "Override", svc.gotCreate.Title)
	require.Equal(t, "override", svc.gotCreate.Slug)
}

func TestImportPlanCanSupersedeAndReturnArchivedTarget(t *testing.T) {
	svc := &fakePlansService{plan: internalplans.Plan{ID: "imported"}}
	h := newPlansHandler(svc)

	resp, err := h.ImportPlan(context.Background(), connect.NewRequest(&plansv1.ImportPlanRequest{Markdown: "# Plan", Supersede: "old"}))
	require.NoError(t, err)
	require.Equal(t, "imported", resp.Msg.GetPlan().GetId())
	require.Equal(t, "superseded", resp.Msg.GetSupersededPlan().GetId())
	require.Equal(t, sharedv1.PlanStatus_PLAN_STATUS_ARCHIVED, resp.Msg.GetSupersededPlan().GetStatus())
}

func TestListTemplatesSuccess(t *testing.T) {
	svc := &fakePlansService{templates: []internalplans.PlanTemplate{
		{ID: "cli", Name: "CLI", Description: "d", Surface: "cli"},
	}}
	h := newPlansHandler(svc)

	resp, err := h.ListTemplates(context.Background(), connect.NewRequest(&plansv1.ListTemplatesRequest{}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetTemplates(), 1)
	require.Equal(t, "cli", resp.Msg.GetTemplates()[0].GetId())
	require.Equal(t, "CLI", resp.Msg.GetTemplates()[0].GetName())
	require.Equal(t, "cli", resp.Msg.GetTemplates()[0].GetSurface())
}

func TestCreateFromTemplateForwardsArgs(t *testing.T) {
	svc := &fakePlansService{plan: internalplans.Plan{ID: "from-tmpl"}}
	h := newPlansHandler(svc)

	resp, err := h.CreateFromTemplate(context.Background(), connect.NewRequest(&plansv1.CreateFromTemplateRequest{
		TemplateId: "cli",
		Title:      "My Plan",
		Slug:       "my-plan",
	}))
	require.NoError(t, err)
	require.Equal(t, "from-tmpl", resp.Msg.GetPlan().GetId())
	require.Equal(t, "cli", svc.gotTemplateID)
	require.Equal(t, "My Plan", svc.gotTemplateTitle)
	require.Equal(t, "my-plan", svc.gotTemplateSlug)
}

// TestPlansErrorMapping asserts each domain sentinel maps to the documented
// Connect code (see internal/plans/service_error_mapping.go), exercised through a
// representative handler per error class.
func TestPlansErrorMapping(t *testing.T) {
	t.Run("plan_not_found_is_not_found", func(t *testing.T) {
		h := newPlansHandler(&fakePlansService{err: internalplans.ErrPlanNotFound{ID: "x"}})
		_, err := h.GetPlan(context.Background(), connect.NewRequest(&plansv1.GetPlanRequest{Id: "x"}))
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	})
	t.Run("invalid_plan_is_invalid_argument", func(t *testing.T) {
		h := newPlansHandler(&fakePlansService{err: internalplans.ErrInvalidPlan{Reason: "title is required"}})
		_, err := h.CreatePlan(context.Background(), connect.NewRequest(&plansv1.CreatePlanRequest{Plan: &sharedv1.Plan{}}))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})
	t.Run("phase_not_found_is_not_found", func(t *testing.T) {
		h := newPlansHandler(&fakePlansService{err: internalplans.ErrPhaseNotFound{PlanID: "p", PhaseID: "ph"}})
		_, err := h.UpdatePhase(context.Background(), connect.NewRequest(&plansv1.UpdatePhaseRequest{PlanId: "p", Phase: &sharedv1.Phase{Id: "ph"}}))
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	})
	t.Run("template_not_found_is_not_found", func(t *testing.T) {
		h := newPlansHandler(&fakePlansService{err: internalplans.ErrTemplateNotFound{ID: "nope"}})
		_, err := h.CreateFromTemplate(context.Background(), connect.NewRequest(&plansv1.CreateFromTemplateRequest{TemplateId: "nope"}))
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	})
	t.Run("unknown_error_is_internal", func(t *testing.T) {
		h := newPlansHandler(&fakePlansService{err: errors.New("disk on fire")})
		_, err := h.GetPlan(context.Background(), connect.NewRequest(&plansv1.GetPlanRequest{Id: "x"}))
		require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	})
}
