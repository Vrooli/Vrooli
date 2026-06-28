package authoring

import (
	"context"
	"log"

	internalauthoring "plan-manager/internal/authoring"

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
	return connect.NewResponse(&authoringv1.SubmitSectionResponse{
		Session:    sessionToProto(sess),
		Violations: violationsToProto(violations),
		Step:       guidedStepToProto(step),
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
		Session:         sessionToProto(sess),
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
		Session: sessionToProto(sess),
		Results: autofillResultsToProto(results),
		Step:    guidedStepToProto(step),
	}), nil
}

func (h *connectHandler) SubmitRelevantContextItem(ctx context.Context, req *connect.Request[authoringv1.SubmitRelevantContextItemRequest]) (*connect.Response[authoringv1.SubmitRelevantContextItemResponse], error) {
	sess, item, violations, step, err := h.deps.Service.SubmitRelevantContextItem(ctx, req.Msg.GetSessionId(), req.Msg.GetPhaseId(), relevantContextItemFromProto(req.Msg.GetItem()))
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.SubmitRelevantContextItemResponse{
		Session:    sessionToProto(sess),
		Item:       relevantContextItemToProto(item),
		Violations: violationsToProto(violations),
		Step:       guidedStepToProto(step),
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

func (h *connectHandler) DiscoverContextCandidates(ctx context.Context, req *connect.Request[authoringv1.DiscoverContextCandidatesRequest]) (*connect.Response[authoringv1.DiscoverContextCandidatesResponse], error) {
	sess, candidates, step, err := h.deps.Service.DiscoverContextCandidates(ctx, req.Msg.GetSessionId(), req.Msg.GetConcepts(), req.Msg.GetComplexity())
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.DiscoverContextCandidatesResponse{
		Session:    sessionToProto(sess),
		Candidates: contextCandidatesToProto(candidates),
		Step:       guidedStepToProto(step),
	}), nil
}

func (h *connectHandler) AcceptContextCandidate(ctx context.Context, req *connect.Request[authoringv1.AcceptContextCandidateRequest]) (*connect.Response[authoringv1.AcceptContextCandidateResponse], error) {
	sess, candidate, item, violations, step, err := h.deps.Service.AcceptContextCandidate(ctx, req.Msg.GetSessionId(), req.Msg.GetCandidateId(), req.Msg.GetPhaseId())
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.AcceptContextCandidateResponse{
		Session:    sessionToProto(sess),
		Candidate:  contextCandidateToProto(candidate),
		Item:       relevantContextItemToProto(item),
		Violations: violationsToProto(violations),
		Step:       guidedStepToProto(step),
	}), nil
}

func (h *connectHandler) RejectContextCandidate(ctx context.Context, req *connect.Request[authoringv1.RejectContextCandidateRequest]) (*connect.Response[authoringv1.RejectContextCandidateResponse], error) {
	sess, candidate, step, err := h.deps.Service.RejectContextCandidate(ctx, req.Msg.GetSessionId(), req.Msg.GetCandidateId(), req.Msg.GetReason())
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.RejectContextCandidateResponse{
		Session:   sessionToProto(sess),
		Candidate: contextCandidateToProto(candidate),
		Step:      guidedStepToProto(step),
	}), nil
}

func (h *connectHandler) AddPhase(ctx context.Context, req *connect.Request[authoringv1.AddPhaseRequest]) (*connect.Response[authoringv1.AddPhaseResponse], error) {
	sess, phase, violations, step, err := h.deps.Service.AddPhase(ctx, req.Msg.GetSessionId(), req.Msg.GetTitle(), req.Msg.GetIntent())
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.AddPhaseResponse{
		Session:    sessionToProto(sess),
		Phase:      phaseDraftToProto(phase),
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
	return connect.NewResponse(&authoringv1.SubmitPhaseFieldResponse{
		Session:    sessionToProto(sess),
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
	plan, step, err := h.deps.Service.Finalize(ctx, req.Msg.GetSessionId())
	if err != nil {
		return nil, internalauthoring.ToConnectError(err)
	}
	return connect.NewResponse(&authoringv1.FinalizeResponse{Plan: planToProto(plan), Step: guidedStepToProto(step)}), nil
}
