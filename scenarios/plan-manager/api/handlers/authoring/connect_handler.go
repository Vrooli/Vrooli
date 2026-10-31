package authoring

import (
	"context"
	"log"

	internalauthoring "plan-manager/internal/authoring"
	"plan-manager/internal/planproto"

	"connectrpc.com/connect"

	authoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/authoring"
)

// Deps wires the seams the Connect authoring handler needs.
type Deps struct {
	Service internalauthoring.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the AuthoringService Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) StartSession(ctx context.Context, req *connect.Request[authoringv1.StartSessionRequest]) (*connect.Response[authoringv1.StartSessionResponse], error) {
	sess, step, err := h.deps.Service.StartSession(ctx, req.Msg.GetTitle(), req.Msg.GetSlug(), req.Msg.GetTemplateId())
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.StartSessionResponse{Session: sessionToProto(sess), Step: guidedStepToProto(step)}), nil
}

// GetSession is the explicit full-state read; mutations no longer echo the
// session graph, so the UI/operator hydrates here deliberately.
func (h *connectHandler) GetSession(ctx context.Context, req *connect.Request[authoringv1.GetSessionRequest]) (*connect.Response[authoringv1.GetSessionResponse], error) {
	sess, step, err := h.deps.Service.GetSession(ctx, req.Msg.GetSessionId())
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.GetSessionResponse{Session: sessionToProto(sess), Step: guidedStepToProto(step)}), nil
}

func (h *connectHandler) GetSection(ctx context.Context, req *connect.Request[authoringv1.GetSectionRequest]) (*connect.Response[authoringv1.GetSectionResponse], error) {
	sec, step, err := h.deps.Service.GetSection(ctx, req.Msg.GetSessionId(), internalauthoring.SectionKey(req.Msg.GetSectionKey()))
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.GetSectionResponse{
		Section: sectionToProto(sec),
		Step:    guidedStepToProto(step),
	}), nil
}

func (h *connectHandler) SubmitSection(ctx context.Context, req *connect.Request[authoringv1.SubmitSectionRequest]) (*connect.Response[authoringv1.SubmitSectionResponse], error) {
	sess, violations, step, err := h.deps.Service.SubmitSection(ctx, req.Msg.GetSessionId(), internalauthoring.SectionKey(req.Msg.GetSectionKey()), req.Msg.GetContent())
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	sec := internalauthoring.Section{Key: internalauthoring.SectionKey(req.Msg.GetSectionKey()), Content: req.Msg.GetContent()}
	return connect.NewResponse(&authoringv1.SubmitSectionResponse{
		Summary:    mutationSummary("section", req.Msg.GetSectionKey(), "", internalauthoring.SectionSummary(sec)),
		Progress:   progressOf(sess),
		Violations: violationsToProto(violations),
		Step:       guidedStepToProto(step),
	}), nil
}

func (h *connectHandler) SubmitFields(ctx context.Context, req *connect.Request[authoringv1.SubmitFieldsRequest]) (*connect.Response[authoringv1.SubmitFieldsResponse], error) {
	writes := make([]internalauthoring.FieldWrite, 0, len(req.Msg.GetItems()))
	for _, item := range req.Msg.GetItems() {
		write := internalauthoring.FieldWrite{Content: item.GetContent()}
		if phase := item.GetPhase(); phase != nil {
			write.PhaseRef = phase.GetPhaseRef()
			write.PhaseField = internalauthoring.PhaseField(phase.GetField())
		} else {
			write.SectionKey = internalauthoring.SectionKey(item.GetSectionKey())
		}
		writes = append(writes, write)
	}
	sess, results, step, err := h.deps.Service.SubmitFields(ctx, req.Msg.GetSessionId(), writes)
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	out := make([]*authoringv1.FieldWriteResult, 0, len(results))
	for _, result := range results {
		out = append(out, &authoringv1.FieldWriteResult{
			Index:      int32(result.Index),
			Accepted:   result.Accepted,
			Summary:    result.Summary,
			Violations: violationsToProto(result.Violations),
		})
	}
	return connect.NewResponse(&authoringv1.SubmitFieldsResponse{
		Results:  out,
		Progress: progressOf(sess),
		Step:     guidedStepToProto(step),
	}), nil
}

func (h *connectHandler) Next(ctx context.Context, req *connect.Request[authoringv1.NextRequest]) (*connect.Response[authoringv1.NextResponse], error) {
	sec, step, complete, err := h.deps.Service.Next(ctx, req.Msg.GetSessionId())
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	resp := &authoringv1.NextResponse{Complete: complete, Step: guidedStepToProto(step)}
	if !complete {
		resp.Section = sectionToProto(sec)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ContinueAuthoring(ctx context.Context, req *connect.Request[authoringv1.ContinueAuthoringRequest]) (*connect.Response[authoringv1.ContinueAuthoringResponse], error) {
	sess, sec, phase, ready, violations, step, err := h.deps.Service.ContinueAuthoring(ctx, req.Msg.GetSessionId())
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	resp := &authoringv1.ContinueAuthoringResponse{
		Progress:        progressOf(sess),
		ReadyToFinalize: ready,
		Violations:      violationsToProto(violations),
		Step:            guidedStepToProto(step),
	}
	if sec.Key != "" {
		resp.Section = sectionToProto(sec)
	}
	if phase.ID != "" {
		resp.Phase = phaseDraftToProto(phase)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ValidateStructure(ctx context.Context, req *connect.Request[authoringv1.ValidateStructureRequest]) (*connect.Response[authoringv1.ValidateStructureResponse], error) {
	valid, violations, step, err := h.deps.Service.ValidateStructure(ctx, req.Msg.GetSessionId())
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.ValidateStructureResponse{
		Valid:      valid,
		Violations: violationsToProto(violations),
		Step:       guidedStepToProto(step),
	}), nil
}

func (h *connectHandler) Autofill(ctx context.Context, req *connect.Request[authoringv1.AutofillRequest]) (*connect.Response[authoringv1.AutofillResponse], error) {
	sess, results, step, err := h.deps.Service.Autofill(ctx, req.Msg.GetSessionId(), autofillSourcesFromProto(req.Msg.GetSources()))
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.AutofillResponse{
		Results:  autofillResultsToProto(results),
		Progress: progressOf(sess),
		Step:     guidedStepToProto(step),
	}), nil
}

func (h *connectHandler) SubmitRelevantContextItem(ctx context.Context, req *connect.Request[authoringv1.SubmitRelevantContextItemRequest]) (*connect.Response[authoringv1.SubmitRelevantContextItemResponse], error) {
	sess, item, violations, step, err := h.deps.Service.SubmitRelevantContextItem(ctx, req.Msg.GetSessionId(), req.Msg.GetPhaseId(), relevantContextItemFromProto(req.Msg.GetItem()))
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.SubmitRelevantContextItemResponse{
		Item:       relevantContextItemToProto(item),
		Summary:    mutationSummary("context", item.ID, "", internalauthoring.ContextItemSummary(item)),
		Progress:   progressOf(sess),
		Violations: violationsToProto(violations),
		Step:       guidedStepToProto(step),
		Accepted:   len(violations) == 0,
	}), nil
}

func (h *connectHandler) ListRelevantContext(ctx context.Context, req *connect.Request[authoringv1.ListRelevantContextRequest]) (*connect.Response[authoringv1.ListRelevantContextResponse], error) {
	items, step, err := h.deps.Service.ListRelevantContext(ctx, req.Msg.GetSessionId(), req.Msg.GetPhaseId())
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.ListRelevantContextResponse{
		Items: relevantContextItemsToProto(items),
		Step:  guidedStepToProto(step),
	}), nil
}

func (h *connectHandler) UpdateRelevantContextItem(ctx context.Context, req *connect.Request[authoringv1.UpdateRelevantContextItemRequest]) (*connect.Response[authoringv1.UpdateRelevantContextItemResponse], error) {
	sess, item, violations, step, err := h.deps.Service.UpdateRelevantContextItem(ctx, req.Msg.GetSessionId(), req.Msg.GetPhaseId(), req.Msg.GetItemId(), relevantContextItemFromProto(req.Msg.GetItem()))
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.UpdateRelevantContextItemResponse{
		Item:       relevantContextItemToProto(item),
		Summary:    mutationSummary("context", item.ID, "", "updated "+internalauthoring.ContextItemSummary(item)),
		Progress:   progressOf(sess),
		Violations: violationsToProto(violations),
		Step:       guidedStepToProto(step),
	}), nil
}

func (h *connectHandler) RemoveRelevantContextItem(ctx context.Context, req *connect.Request[authoringv1.RemoveRelevantContextItemRequest]) (*connect.Response[authoringv1.RemoveRelevantContextItemResponse], error) {
	sess, violations, step, err := h.deps.Service.RemoveRelevantContextItem(ctx, req.Msg.GetSessionId(), req.Msg.GetPhaseId(), req.Msg.GetItemId())
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.RemoveRelevantContextItemResponse{
		Summary:    mutationSummary("context", req.Msg.GetItemId(), "", "removed context item"),
		Progress:   progressOf(sess),
		Violations: violationsToProto(violations),
		Step:       guidedStepToProto(step),
	}), nil
}

func (h *connectHandler) DiscoverSkillPack(ctx context.Context, req *connect.Request[authoringv1.DiscoverSkillPackRequest]) (*connect.Response[authoringv1.DiscoverSkillPackResponse], error) {
	sess, result, added, kept, violations, step, err := h.deps.Service.DiscoverSkillPack(ctx, req.Msg.GetSessionId(), req.Msg.GetPhaseId(), req.Msg.GetConcepts(), req.Msg.GetComplexity())
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.DiscoverSkillPackResponse{
		AddedItems:             relevantContextItemsToProto(added),
		KeptItems:              relevantContextItemsToProto(kept),
		ReadCommand:            result.ReadCommand,
		RecommendedReadCommand: result.RecommendedReadCommand,
		BudgetStatus:           result.BudgetStatus,
		ResultsSummary:         result.Summary,
		Progress:               progressOf(sess),
		Step:                   guidedStepToProto(step),
		Violations:             violationsToProto(violations),
		Degraded:               result.Degraded,
		DegradedReason:         result.DegradedReason,
	}), nil
}

func (h *connectHandler) AddPhase(ctx context.Context, req *connect.Request[authoringv1.AddPhaseRequest]) (*connect.Response[authoringv1.AddPhaseResponse], error) {
	sess, phase, violations, step, err := h.deps.Service.AddPhase(ctx, req.Msg.GetSessionId(), req.Msg.GetTitle(), req.Msg.GetIntent())
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.AddPhaseResponse{
		Phase:      phaseDraftToProto(phase),
		Summary:    mutationSummary("phase", phase.ID, "", internalauthoring.PhaseAddSummary(phase)),
		Progress:   progressOf(sess),
		Violations: violationsToProto(violations),
		Step:       guidedStepToProto(step),
	}), nil
}

func (h *connectHandler) MovePhase(ctx context.Context, req *connect.Request[authoringv1.MovePhaseRequest]) (*connect.Response[authoringv1.MovePhaseResponse], error) {
	sess, phase, violations, step, err := h.deps.Service.MovePhase(ctx, req.Msg.GetSessionId(), req.Msg.GetPhaseId(), req.Msg.GetBeforePhaseId(), req.Msg.GetAfterPhaseId())
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.MovePhaseResponse{
		Phase:      phaseDraftToProto(phase),
		Summary:    mutationSummary("phase", phase.ID, "order", internalauthoring.PhaseFieldSummary(internalauthoring.PhaseFieldTitle, phase)),
		Progress:   progressOf(sess),
		Violations: violationsToProto(violations),
		Step:       guidedStepToProto(step),
	}), nil
}

func (h *connectHandler) GetPhase(ctx context.Context, req *connect.Request[authoringv1.GetPhaseRequest]) (*connect.Response[authoringv1.GetPhaseResponse], error) {
	phase, step, err := h.deps.Service.GetPhase(ctx, req.Msg.GetSessionId(), req.Msg.GetPhaseId())
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.GetPhaseResponse{
		Phase: phaseDraftToProto(phase),
		Step:  guidedStepToProto(step),
	}), nil
}

func (h *connectHandler) SubmitPhaseField(ctx context.Context, req *connect.Request[authoringv1.SubmitPhaseFieldRequest]) (*connect.Response[authoringv1.SubmitPhaseFieldResponse], error) {
	sess, violations, step, err := h.deps.Service.SubmitPhaseField(ctx, req.Msg.GetSessionId(), req.Msg.GetPhaseId(), internalauthoring.PhaseField(req.Msg.GetField()), req.Msg.GetContent())
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	field := internalauthoring.PhaseField(req.Msg.GetField())
	phase, _ := internalauthoring.FindPhaseDraft(sess, req.Msg.GetPhaseId())
	return connect.NewResponse(&authoringv1.SubmitPhaseFieldResponse{
		Phase:      phaseDraftToProto(phase),
		Summary:    mutationSummary("phase", phase.ID, req.Msg.GetField(), internalauthoring.PhaseFieldSummary(field, phase)),
		Progress:   progressOf(sess),
		Violations: violationsToProto(violations),
		Step:       guidedStepToProto(step),
	}), nil
}

func (h *connectHandler) NextPhase(ctx context.Context, req *connect.Request[authoringv1.NextPhaseRequest]) (*connect.Response[authoringv1.NextPhaseResponse], error) {
	phase, step, complete, err := h.deps.Service.NextPhase(ctx, req.Msg.GetSessionId())
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	resp := &authoringv1.NextPhaseResponse{Complete: complete, Step: guidedStepToProto(step)}
	if !complete {
		resp.Phase = phaseDraftToProto(phase)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) PreviewPlan(ctx context.Context, req *connect.Request[authoringv1.PreviewPlanRequest]) (*connect.Response[authoringv1.PreviewPlanResponse], error) {
	markdown, step, err := h.deps.Service.PreviewPlan(ctx, req.Msg.GetSessionId())
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.PreviewPlanResponse{Markdown: markdown, Step: guidedStepToProto(step)}), nil
}

func (h *connectHandler) Finalize(ctx context.Context, req *connect.Request[authoringv1.FinalizeRequest]) (*connect.Response[authoringv1.FinalizeResponse], error) {
	result, step, err := h.deps.Service.Finalize(ctx, req.Msg.GetSessionId(), internalauthoring.FinalizeOptions{
		WorkspaceRoot: req.Msg.GetWorkspaceRoot(),
	})
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.FinalizeResponse{
		Plan:             planToProto(result.Plan),
		Step:             guidedStepToProto(step),
		StorePath:        result.StorePath,
		Mirror:           planproto.MirrorToProto(result.Mirror),
		AlreadyFinalized: result.AlreadyFinalized,
		FinalizedAt:      result.FinalizedAt,
		WorkspaceRoot:    result.Plan.WorkspaceRoot,
	}), nil
}
