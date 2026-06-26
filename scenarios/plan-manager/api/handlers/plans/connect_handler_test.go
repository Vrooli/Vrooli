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
	err       error

	gotListFilter        internalplans.ListFilter
	gotGetID             string
	gotCreate            internalplans.Plan
	gotUpdate            internalplans.Plan
	gotArchiveID         string
	gotRenderID          string
	gotAddPhasePlanID    string
	gotAddPhase          internalplans.Phase
	gotUpdatePhasePlanID string
	gotUpdatePhase       internalplans.Phase
	gotGraphPlanID       string
	gotSupersedingID     string
	gotSupersededID      string
	gotDependingID       string
	gotDependencyID      string
	gotImportPath        string
	gotImportMarkdown    string
	gotMigrateID         string
	gotTemplateID        string
	gotTemplateTitle     string
	gotTemplateSlug      string
}

func (f *fakePlansService) Create(_ context.Context, p internalplans.Plan) (internalplans.Plan, error) {
	f.gotCreate = p
	return f.plan, f.err
}

func (f *fakePlansService) Update(_ context.Context, p internalplans.Plan) (internalplans.Plan, error) {
	f.gotUpdate = p
	return f.plan, f.err
}

func (f *fakePlansService) Get(_ context.Context, idOrSlug string) (internalplans.Plan, error) {
	f.gotGetID = idOrSlug
	return f.plan, f.err
}

func (f *fakePlansService) List(_ context.Context, filter internalplans.ListFilter) ([]internalplans.Plan, error) {
	f.gotListFilter = filter
	return f.plans, f.err
}

func (f *fakePlansService) Archive(_ context.Context, idOrSlug string) (internalplans.Plan, error) {
	f.gotArchiveID = idOrSlug
	return f.plan, f.err
}

func (f *fakePlansService) Render(_ context.Context, idOrSlug string) (string, error) {
	f.gotRenderID = idOrSlug
	return f.markdown, f.err
}

func (f *fakePlansService) AddPhase(_ context.Context, planID string, phase internalplans.Phase) (internalplans.Plan, error) {
	f.gotAddPhasePlanID = planID
	f.gotAddPhase = phase
	return f.plan, f.err
}

func (f *fakePlansService) UpdatePhase(_ context.Context, planID string, phase internalplans.Phase) (internalplans.Plan, error) {
	f.gotUpdatePhasePlanID = planID
	f.gotUpdatePhase = phase
	return f.plan, f.err
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

func (f *fakePlansService) Import(_ context.Context, sourcePath, markdown string) (internalplans.Plan, error) {
	f.gotImportPath = sourcePath
	f.gotImportMarkdown = markdown
	return f.plan, f.err
}

func (f *fakePlansService) Migrate(_ context.Context, idOrSlug string) (internalplans.Plan, error) {
	f.gotMigrateID = idOrSlug
	return f.plan, f.err
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
	}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetPlans(), 2)
	require.Equal(t, "p1", resp.Msg.GetPlans()[0].GetId())
	require.Equal(t, sharedv1.PlanStatus_PLAN_STATUS_COMPLETE, resp.Msg.GetPlans()[1].GetStatus())
	// Filter is translated from the proto request.
	require.Equal(t, internalplans.PlanStatusActive, svc.gotListFilter.Status)
	require.True(t, svc.gotListFilter.IncludeArchived)
}

func TestGetPlanSuccess(t *testing.T) {
	svc := &fakePlansService{plan: internalplans.Plan{ID: "p1", Title: "Hello", Status: internalplans.PlanStatusDraft}}
	h := newPlansHandler(svc)

	resp, err := h.GetPlan(context.Background(), connect.NewRequest(&plansv1.GetPlanRequest{Id: "p1"}))
	require.NoError(t, err)
	require.Equal(t, "p1", resp.Msg.GetPlan().GetId())
	require.Equal(t, "Hello", resp.Msg.GetPlan().GetTitle())
	require.Equal(t, sharedv1.PlanStatus_PLAN_STATUS_DRAFT, resp.Msg.GetPlan().GetStatus())
	require.Equal(t, "p1", svc.gotGetID, "handler must forward the request id to the service")
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

func TestMigratePlanSuccess(t *testing.T) {
	svc := &fakePlansService{plan: internalplans.Plan{ID: "p1"}}
	h := newPlansHandler(svc)

	resp, err := h.MigratePlan(context.Background(), connect.NewRequest(&plansv1.MigratePlanRequest{Id: "p1"}))
	require.NoError(t, err)
	require.Equal(t, "p1", resp.Msg.GetPlan().GetId())
	require.Equal(t, "p1", svc.gotMigrateID)
}

func TestArchivePlanSuccess(t *testing.T) {
	svc := &fakePlansService{plan: internalplans.Plan{ID: "p1", Status: internalplans.PlanStatusArchived}}
	h := newPlansHandler(svc)

	resp, err := h.ArchivePlan(context.Background(), connect.NewRequest(&plansv1.ArchivePlanRequest{Id: "p1"}))
	require.NoError(t, err)
	require.Equal(t, sharedv1.PlanStatus_PLAN_STATUS_ARCHIVED, resp.Msg.GetPlan().GetStatus())
	require.Equal(t, "p1", svc.gotArchiveID)
}

func TestRenderMarkdownSuccess(t *testing.T) {
	svc := &fakePlansService{markdown: "# Title\n"}
	h := newPlansHandler(svc)

	resp, err := h.RenderMarkdown(context.Background(), connect.NewRequest(&plansv1.RenderMarkdownRequest{Id: "p1"}))
	require.NoError(t, err)
	require.Equal(t, "# Title\n", resp.Msg.GetMarkdown())
	require.Equal(t, "p1", svc.gotRenderID)
}

func TestAddPhaseSuccess(t *testing.T) {
	svc := &fakePlansService{plan: internalplans.Plan{ID: "p1"}}
	h := newPlansHandler(svc)

	resp, err := h.AddPhase(context.Background(), connect.NewRequest(&plansv1.AddPhaseRequest{
		PlanId: "p1",
		Phase:  &sharedv1.Phase{Title: "New Phase", Status: sharedv1.PhaseStatus_PHASE_STATUS_TODO},
	}))
	require.NoError(t, err)
	require.Equal(t, "p1", resp.Msg.GetPlan().GetId())
	require.Equal(t, "p1", svc.gotAddPhasePlanID)
	require.Equal(t, "New Phase", svc.gotAddPhase.Title)
	require.Equal(t, internalplans.PhaseStatusTodo, svc.gotAddPhase.Status)
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
	}))
	require.NoError(t, err)
	require.Equal(t, "imported", resp.Msg.GetPlan().GetId())
	require.Equal(t, "docs/plan.md", svc.gotImportPath)
	require.Equal(t, "# Plan", svc.gotImportMarkdown)
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
