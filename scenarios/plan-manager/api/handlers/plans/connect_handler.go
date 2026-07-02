package plans

import (
	"context"
	"log"

	"plan-manager/internal/planproto"
	internalplans "plan-manager/internal/plans"

	"connectrpc.com/connect"

	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans"
)

// Deps wires the seams the Connect plans handler needs.
type Deps struct {
	Service internalplans.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the PlansService Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ListPlans(ctx context.Context, req *connect.Request[plansv1.ListPlansRequest]) (*connect.Response[plansv1.ListPlansResponse], error) {
	plans, err := h.deps.Service.List(ctx, internalplans.ListFilter{
		Status:          planStatusFromProto(req.Msg.GetStatus()),
		IncludeArchived: req.Msg.GetIncludeArchived(),
		WorkspaceID:     workspaceScopeFromProto(req.Msg.GetWorkspace()).ID,
		WorkspaceRoot:   workspaceScopeFromProto(req.Msg.GetWorkspace()).Root,
	})
	if err != nil {
		return nil, internalplans.ToConnectError(err)
	}
	resp := &plansv1.ListPlansResponse{Plans: make([]*sharedPlan, 0, len(plans))}
	for _, p := range plans {
		resp.Plans = append(resp.Plans, planToProto(p))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetPlan(ctx context.Context, req *connect.Request[plansv1.GetPlanRequest]) (*connect.Response[plansv1.GetPlanResponse], error) {
	p, err := h.deps.Service.Get(ctx, req.Msg.GetId(), workspaceScopeFromProto(req.Msg.GetWorkspace()))
	if err != nil {
		return nil, internalplans.ToConnectError(err)
	}
	return connect.NewResponse(&plansv1.GetPlanResponse{Plan: planToProto(p)}), nil
}

func (h *connectHandler) CreatePlan(ctx context.Context, req *connect.Request[plansv1.CreatePlanRequest]) (*connect.Response[plansv1.CreatePlanResponse], error) {
	in, err := planFromProtoChecked(req.Msg.GetPlan())
	if err != nil {
		return nil, internalplans.ToConnectError(err)
	}
	p, err := h.deps.Service.Create(ctx, in)
	if err != nil {
		return nil, internalplans.ToConnectError(err)
	}
	return connect.NewResponse(&plansv1.CreatePlanResponse{Plan: planToProto(p)}), nil
}

func (h *connectHandler) UpdatePlan(ctx context.Context, req *connect.Request[plansv1.UpdatePlanRequest]) (*connect.Response[plansv1.UpdatePlanResponse], error) {
	in, err := planFromProtoChecked(req.Msg.GetPlan())
	if err != nil {
		return nil, internalplans.ToConnectError(err)
	}
	p, err := h.deps.Service.Update(ctx, in)
	if err != nil {
		return nil, internalplans.ToConnectError(err)
	}
	return connect.NewResponse(&plansv1.UpdatePlanResponse{Plan: planToProto(p)}), nil
}

func (h *connectHandler) ArchivePlan(ctx context.Context, req *connect.Request[plansv1.ArchivePlanRequest]) (*connect.Response[plansv1.ArchivePlanResponse], error) {
	p, err := h.deps.Service.Archive(ctx, req.Msg.GetId(), workspaceScopeFromProto(req.Msg.GetWorkspace()))
	if err != nil {
		return nil, internalplans.ToConnectError(err)
	}
	return connect.NewResponse(&plansv1.ArchivePlanResponse{Plan: planToProto(p)}), nil
}

func (h *connectHandler) RenderMarkdown(ctx context.Context, req *connect.Request[plansv1.RenderMarkdownRequest]) (*connect.Response[plansv1.RenderMarkdownResponse], error) {
	rendered, err := h.deps.Service.Render(ctx, req.Msg.GetId(), workspaceScopeFromProto(req.Msg.GetWorkspace()), internalplans.RenderOptions{Compact: req.Msg.GetCompact()})
	if err != nil {
		return nil, internalplans.ToConnectError(err)
	}
	return connect.NewResponse(&plansv1.RenderMarkdownResponse{
		Markdown:        rendered.Markdown,
		Mirror:          planproto.MirrorToProto(rendered.Mirror),
		Repaired:        rendered.Repaired,
		Plan:            planToProto(rendered.Plan),
		QualityStatus:   rendered.QualityStatus,
		QualityFindings: rendered.QualityFindings,
	}), nil
}

func (h *connectHandler) AddPhase(ctx context.Context, req *connect.Request[plansv1.AddPhaseRequest]) (*connect.Response[plansv1.AddPhaseResponse], error) {
	phase, err := phaseFromProtoChecked(req.Msg.GetPhase())
	if err != nil {
		return nil, internalplans.ToConnectError(err)
	}
	p, err := h.deps.Service.AddPhase(ctx, req.Msg.GetPlanId(), phase)
	if err != nil {
		return nil, internalplans.ToConnectError(err)
	}
	return connect.NewResponse(&plansv1.AddPhaseResponse{Plan: planToProto(p)}), nil
}

func (h *connectHandler) UpdatePhase(ctx context.Context, req *connect.Request[plansv1.UpdatePhaseRequest]) (*connect.Response[plansv1.UpdatePhaseResponse], error) {
	phase, err := phaseFromProtoChecked(req.Msg.GetPhase())
	if err != nil {
		return nil, internalplans.ToConnectError(err)
	}
	p, err := h.deps.Service.UpdatePhase(ctx, req.Msg.GetPlanId(), phase)
	if err != nil {
		return nil, internalplans.ToConnectError(err)
	}
	return connect.NewResponse(&plansv1.UpdatePhaseResponse{Plan: planToProto(p)}), nil
}

func (h *connectHandler) ListRelevantContext(ctx context.Context, req *connect.Request[plansv1.ListRelevantContextRequest]) (*connect.Response[plansv1.ListRelevantContextResponse], error) {
	items, err := h.deps.Service.ListRelevantContext(ctx, req.Msg.GetId(), workspaceScopeFromProto(req.Msg.GetWorkspace()), req.Msg.GetPhaseId())
	if err != nil {
		return nil, internalplans.ToConnectError(err)
	}
	return connect.NewResponse(&plansv1.ListRelevantContextResponse{Items: planproto.RelevantContextItemsToProto(items)}), nil
}

func (h *connectHandler) UpdateRelevantContext(ctx context.Context, req *connect.Request[plansv1.UpdateRelevantContextRequest]) (*connect.Response[plansv1.UpdateRelevantContextResponse], error) {
	items := planproto.RelevantContextItemsFromProto([]*sharedRelevantContextItem{req.Msg.GetItem()})
	var item internalplans.RelevantContextItem
	if len(items) > 0 {
		item = items[0]
	}
	p, err := h.deps.Service.UpdateRelevantContext(ctx, req.Msg.GetId(), workspaceScopeFromProto(req.Msg.GetWorkspace()), req.Msg.GetPhaseId(), req.Msg.GetItemId(), item)
	if err != nil {
		return nil, internalplans.ToConnectError(err)
	}
	return connect.NewResponse(&plansv1.UpdateRelevantContextResponse{Plan: planToProto(p)}), nil
}

func (h *connectHandler) RemoveRelevantContext(ctx context.Context, req *connect.Request[plansv1.RemoveRelevantContextRequest]) (*connect.Response[plansv1.RemoveRelevantContextResponse], error) {
	p, err := h.deps.Service.RemoveRelevantContext(ctx, req.Msg.GetId(), workspaceScopeFromProto(req.Msg.GetWorkspace()), req.Msg.GetPhaseId(), req.Msg.GetItemId())
	if err != nil {
		return nil, internalplans.ToConnectError(err)
	}
	return connect.NewResponse(&plansv1.RemoveRelevantContextResponse{Plan: planToProto(p)}), nil
}

func (h *connectHandler) ListReferences(ctx context.Context, req *connect.Request[plansv1.ListReferencesRequest]) (*connect.Response[plansv1.ListReferencesResponse], error) {
	refs, err := h.deps.Service.ListReferences(ctx, req.Msg.GetId(), workspaceScopeFromProto(req.Msg.GetWorkspace()), req.Msg.GetPhaseId())
	if err != nil {
		return nil, internalplans.ToConnectError(err)
	}
	return connect.NewResponse(&plansv1.ListReferencesResponse{References: planproto.ReferencesToProto(refs)}), nil
}

func (h *connectHandler) UpdateReference(ctx context.Context, req *connect.Request[plansv1.UpdateReferenceRequest]) (*connect.Response[plansv1.UpdateReferenceResponse], error) {
	refs := planproto.ReferencesFromProto([]*sharedReference{req.Msg.GetReference()})
	var ref internalplans.Reference
	if len(refs) > 0 {
		ref = refs[0]
	}
	p, err := h.deps.Service.UpdateReference(ctx, req.Msg.GetId(), workspaceScopeFromProto(req.Msg.GetWorkspace()), req.Msg.GetPhaseId(), req.Msg.GetReferenceId(), ref)
	if err != nil {
		return nil, internalplans.ToConnectError(err)
	}
	return connect.NewResponse(&plansv1.UpdateReferenceResponse{Plan: planToProto(p)}), nil
}

func (h *connectHandler) RemoveReference(ctx context.Context, req *connect.Request[plansv1.RemoveReferenceRequest]) (*connect.Response[plansv1.RemoveReferenceResponse], error) {
	p, err := h.deps.Service.RemoveReference(ctx, req.Msg.GetId(), workspaceScopeFromProto(req.Msg.GetWorkspace()), req.Msg.GetPhaseId(), req.Msg.GetReferenceId())
	if err != nil {
		return nil, internalplans.ToConnectError(err)
	}
	return connect.NewResponse(&plansv1.RemoveReferenceResponse{Plan: planToProto(p)}), nil
}

func (h *connectHandler) GetGraph(ctx context.Context, req *connect.Request[plansv1.GetGraphRequest]) (*connect.Response[plansv1.GetGraphResponse], error) {
	edges, err := h.deps.Service.GetGraph(ctx, req.Msg.GetPlanId())
	if err != nil {
		return nil, internalplans.ToConnectError(err)
	}
	resp := &plansv1.GetGraphResponse{Edges: make([]*sharedPlanEdge, 0, len(edges))}
	for _, e := range edges {
		resp.Edges = append(resp.Edges, edgeToProto(e))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) LinkSupersession(ctx context.Context, req *connect.Request[plansv1.LinkSupersessionRequest]) (*connect.Response[plansv1.LinkSupersessionResponse], error) {
	p, err := h.deps.Service.LinkSupersession(ctx, req.Msg.GetSupersedingPlanId(), req.Msg.GetSupersededPlanId())
	if err != nil {
		return nil, internalplans.ToConnectError(err)
	}
	return connect.NewResponse(&plansv1.LinkSupersessionResponse{Plan: planToProto(p)}), nil
}

func (h *connectHandler) LinkDependency(ctx context.Context, req *connect.Request[plansv1.LinkDependencyRequest]) (*connect.Response[plansv1.LinkDependencyResponse], error) {
	p, err := h.deps.Service.LinkDependency(ctx, req.Msg.GetDependingPlanId(), req.Msg.GetDependencyPlanId())
	if err != nil {
		return nil, internalplans.ToConnectError(err)
	}
	return connect.NewResponse(&plansv1.LinkDependencyResponse{Plan: planToProto(p)}), nil
}

func (h *connectHandler) ImportPlan(ctx context.Context, req *connect.Request[plansv1.ImportPlanRequest]) (*connect.Response[plansv1.ImportPlanResponse], error) {
	p, err := h.deps.Service.Import(ctx, req.Msg.GetSourcePath(), req.Msg.GetMarkdown(), req.Msg.GetTitle(), req.Msg.GetSlug(), workspaceScopeFromProto(req.Msg.GetWorkspace()))
	if err != nil {
		return nil, internalplans.ToConnectError(err)
	}
	return connect.NewResponse(&plansv1.ImportPlanResponse{Plan: planToProto(p)}), nil
}

func (h *connectHandler) MigratePlan(ctx context.Context, req *connect.Request[plansv1.MigratePlanRequest]) (*connect.Response[plansv1.MigratePlanResponse], error) {
	p, err := h.deps.Service.Migrate(ctx, req.Msg.GetId())
	if err != nil {
		return nil, internalplans.ToConnectError(err)
	}
	return connect.NewResponse(&plansv1.MigratePlanResponse{Plan: planToProto(p)}), nil
}

func (h *connectHandler) ReconcilePlans(ctx context.Context, req *connect.Request[plansv1.ReconcilePlansRequest]) (*connect.Response[plansv1.ReconcilePlansResponse], error) {
	result, err := h.deps.Service.Reconcile(ctx, reconcileRequestFromProto(req.Msg))
	if err != nil {
		return nil, internalplans.ToConnectError(err)
	}
	return connect.NewResponse(reconcileResultToProto(result)), nil
}

func (h *connectHandler) ListTemplates(ctx context.Context, req *connect.Request[plansv1.ListTemplatesRequest]) (*connect.Response[plansv1.ListTemplatesResponse], error) {
	templates, err := h.deps.Service.ListTemplates(ctx)
	if err != nil {
		return nil, internalplans.ToConnectError(err)
	}
	resp := &plansv1.ListTemplatesResponse{Templates: make([]*plansv1.PlanTemplate, 0, len(templates))}
	for _, t := range templates {
		resp.Templates = append(resp.Templates, &plansv1.PlanTemplate{
			Id:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Surface:     t.Surface,
		})
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) CreateFromTemplate(ctx context.Context, req *connect.Request[plansv1.CreateFromTemplateRequest]) (*connect.Response[plansv1.CreateFromTemplateResponse], error) {
	p, err := h.deps.Service.CreateFromTemplate(ctx, req.Msg.GetTemplateId(), req.Msg.GetTitle(), req.Msg.GetSlug())
	if err != nil {
		return nil, internalplans.ToConnectError(err)
	}
	return connect.NewResponse(&plansv1.CreateFromTemplateResponse{Plan: planToProto(p)}), nil
}
