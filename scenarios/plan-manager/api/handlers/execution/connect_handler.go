package execution

import (
	"context"
	"errors"
	"fmt"
	"log"

	internalexecution "plan-manager/internal/execution"

	"connectrpc.com/connect"

	executionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/execution"
)

// Deps wires the seams the Connect execution handler needs.
type Deps struct {
	Service internalexecution.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the ExecutionService Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) Start(ctx context.Context, req *connect.Request[executionv1.StartRequest]) (*connect.Response[executionv1.StartResponse], error) {
	e, pctx, step, err := h.deps.Service.Start(ctx, req.Msg.GetPlanId(), req.Msg.GetRunId())
	if err != nil {
		var conflict internalexecution.ErrActiveExecutionConflict
		if errors.As(err, &conflict) {
			return nil, activeExecutionConnectError(conflict)
		}
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.StartResponse{
		Execution: executionToProto(e),
		Step:      guidedStepToProto(step),
		Context:   phaseContextToProto(pctx),
	}), nil
}

func activeExecutionConnectError(conflict internalexecution.ErrActiveExecutionConflict) error {
	detailMessage := &executionv1.ActiveExecutionConflict{ExecutionIds: append([]string(nil), conflict.ExecutionIDs...)}
	for _, id := range conflict.ExecutionIDs {
		detailMessage.ResumeCommands = append(detailMessage.ResumeCommands, "plan-manager exec resume "+id)
		detailMessage.AbandonCommands = append(detailMessage.AbandonCommands, "plan-manager exec abandon "+id+" --reason <reason>")
	}
	err := connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("%w", conflict))
	detail, detailErr := connect.NewErrorDetail(detailMessage)
	if detailErr == nil {
		err.AddDetail(detail)
	}
	return err
}

func (h *connectHandler) GetStatus(ctx context.Context, req *connect.Request[executionv1.GetStatusRequest]) (*connect.Response[executionv1.GetStatusResponse], error) {
	e, pctx, step, err := h.deps.Service.GetStatus(ctx, req.Msg.GetExecutionId())
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.GetStatusResponse{
		Execution: executionToProto(e),
		Context:   phaseContextToProto(pctx),
		Step:      guidedStepToProto(step),
	}), nil
}

func (h *connectHandler) GetContext(ctx context.Context, req *connect.Request[executionv1.GetContextRequest]) (*connect.Response[executionv1.GetContextResponse], error) {
	e, pctx, step, err := h.deps.Service.GetContext(ctx, req.Msg.GetExecutionId(), req.Msg.GetPhaseId())
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.GetContextResponse{
		Execution: executionToProto(e),
		Context:   phaseContextToProto(pctx),
		Step:      guidedStepToProto(step),
	}), nil
}

func (h *connectHandler) Resume(ctx context.Context, req *connect.Request[executionv1.ResumeRequest]) (*connect.Response[executionv1.ResumeResponse], error) {
	e, pctx, step, err := h.deps.Service.Resume(ctx, req.Msg.GetPlanOrExecution(), req.Msg.GetPhaseId(), req.Msg.GetRunId())
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.ResumeResponse{
		Execution: executionToProto(e),
		Context:   phaseContextToProto(pctx),
		Step:      guidedStepToProto(step),
	}), nil
}

func (h *connectHandler) ContinueExecution(ctx context.Context, req *connect.Request[executionv1.ContinueExecutionRequest]) (*connect.Response[executionv1.ContinueExecutionResponse], error) {
	e, pctx, step, err := h.deps.Service.ContinueExecution(ctx, req.Msg.GetPlanOrExecution(), req.Msg.GetPhaseId(), req.Msg.GetRunId())
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.ContinueExecutionResponse{
		Execution: executionToProto(e),
		Context:   phaseContextToProto(pctx),
		Step:      guidedStepToProto(step),
	}), nil
}

func (h *connectHandler) AbandonExecution(ctx context.Context, req *connect.Request[executionv1.AbandonExecutionRequest]) (*connect.Response[executionv1.AbandonExecutionResponse], error) {
	e, replay, step, err := h.deps.Service.AbandonExecution(ctx, req.Msg.GetExecutionId(), req.Msg.GetReason(), req.Msg.GetActor())
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.AbandonExecutionResponse{Execution: executionToProto(e), AlreadyAbandoned: replay, Step: guidedStepToProto(step)}), nil
}

func (h *connectHandler) SyncBaseline(ctx context.Context, req *connect.Request[executionv1.SyncBaselineRequest]) (*connect.Response[executionv1.SyncBaselineResponse], error) {
	e, pctx, step, err := h.deps.Service.SyncBaseline(ctx, req.Msg.GetExecutionId())
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.SyncBaselineResponse{Execution: executionToProto(e), Context: phaseContextToProto(pctx), Step: guidedStepToProto(step)}), nil
}

func (h *connectHandler) AmendScope(ctx context.Context, req *connect.Request[executionv1.AmendScopeRequest]) (*connect.Response[executionv1.AmendScopeResponse], error) {
	e, pctx, step, err := h.deps.Service.AmendScope(ctx, req.Msg.GetExecutionId(), internalexecution.ScopeAmendmentRequest{PhaseID: req.Msg.GetPhaseId(), Members: req.Msg.GetMember(), Author: req.Msg.GetAuthor(), Reason: req.Msg.GetReason()})
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.AmendScopeResponse{Execution: executionToProto(e), Context: phaseContextToProto(pctx), Step: guidedStepToProto(step)}), nil
}

func (h *connectHandler) AdoptBaseline(ctx context.Context, req *connect.Request[executionv1.AdoptBaselineRequest]) (*connect.Response[executionv1.AdoptBaselineResponse], error) {
	e, pctx, step, err := h.deps.Service.AdoptBaseline(ctx, req.Msg.GetExecutionId(), internalexecution.BaselineAdoptionRequest{Mode: internalexecution.BaselineAdoptionMode(req.Msg.GetMode()), Name: req.Msg.GetName(), Members: req.Msg.GetMember(), RepoPaths: req.Msg.GetPath(), Reason: req.Msg.GetReason()})
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.AdoptBaselineResponse{Execution: executionToProto(e), Context: phaseContextToProto(pctx), Step: guidedStepToProto(step)}), nil
}

func (h *connectHandler) RepairSourceScope(ctx context.Context, req *connect.Request[executionv1.RepairSourceScopeRequest]) (*connect.Response[executionv1.RepairSourceScopeResponse], error) {
	e, pctx, step, err := h.deps.Service.RepairSourceScope(ctx, req.Msg.GetExecutionId(), internalexecution.SourceScopeRepairRequest{Paths: req.Msg.GetPath(), Reason: req.Msg.GetReason()})
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.RepairSourceScopeResponse{Execution: executionToProto(e), Context: phaseContextToProto(pctx), Step: guidedStepToProto(step)}), nil
}

func (h *connectHandler) ExtendBoundary(ctx context.Context, req *connect.Request[executionv1.ExtendBoundaryRequest]) (*connect.Response[executionv1.ExtendBoundaryResponse], error) {
	e, pctx, step, added, err := h.deps.Service.ExtendBoundary(ctx, req.Msg.GetExecutionId(), internalexecution.BoundaryExtensionRequest{Paths: req.Msg.GetPath(), Reason: req.Msg.GetReason(), Author: req.Msg.GetAuthor()})
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.ExtendBoundaryResponse{Execution: executionToProto(e), Context: phaseContextToProto(pctx), Step: guidedStepToProto(step), AddedAllow: append([]string(nil), added...)}), nil
}

func (h *connectHandler) GetNext(ctx context.Context, req *connect.Request[executionv1.GetNextRequest]) (*connect.Response[executionv1.GetNextResponse], error) {
	pctx, complete, step, err := h.deps.Service.GetNext(ctx, req.Msg.GetExecutionId())
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.GetNextResponse{
		Context:  phaseContextToProto(pctx),
		Complete: complete,
		Step:     guidedStepToProto(step),
	}), nil
}

func (h *connectHandler) TransitionPhase(ctx context.Context, req *connect.Request[executionv1.TransitionPhaseRequest]) (*connect.Response[executionv1.TransitionPhaseResponse], error) {
	e, plan, step, err := h.deps.Service.TransitionPhase(ctx, req.Msg.GetExecutionId(), req.Msg.GetPhaseId(), internalexecution.PhaseTransitionInputs{
		ToStatus:                 phaseStatusFromProto(req.Msg.GetToStatus()),
		ValidationOverrideReason: req.Msg.GetValidationOverride().GetReason(),
		FeedbackOverrideReason:   req.Msg.GetFeedbackOverride().GetReason(),
	})
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.TransitionPhaseResponse{
		Execution: executionToProto(e),
		Plan:      planToProto(plan),
		Step:      guidedStepToProto(step),
	}), nil
}

func (h *connectHandler) Complete(ctx context.Context, req *connect.Request[executionv1.CompleteRequest]) (*connect.Response[executionv1.CompleteResponse], error) {
	handoff, nudges, step, err := h.deps.Service.Complete(ctx, req.Msg.GetExecutionId(), internalexecution.CompletionInputs{
		Tokens:     req.Msg.GetTokens(),
		Iterations: req.Msg.GetIterations(),
	})
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.CompleteResponse{
		Handoff: handoffToProto(handoff),
		Nudges:  nudgesToProto(nudges),
		Step:    guidedStepToProto(step),
	}), nil
}

func (h *connectHandler) PartialHandoff(ctx context.Context, req *connect.Request[executionv1.PartialHandoffRequest]) (*connect.Response[executionv1.PartialHandoffResponse], error) {
	handoff, nudges, step, err := h.deps.Service.PartialHandoff(ctx, req.Msg.GetExecutionId(), internalexecution.CompletionInputs{Tokens: req.Msg.GetTokens(), Iterations: req.Msg.GetIterations()})
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.PartialHandoffResponse{Handoff: handoffToProto(handoff), Nudges: nudgesToProto(nudges), Step: guidedStepToProto(step)}), nil
}

func (h *connectHandler) GetHandoff(ctx context.Context, req *connect.Request[executionv1.GetHandoffRequest]) (*connect.Response[executionv1.GetHandoffResponse], error) {
	handoff, step, err := h.deps.Service.GetHandoff(ctx, req.Msg.GetExecutionId())
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.GetHandoffResponse{Handoff: handoffToProto(handoff), Step: guidedStepToProto(step)}), nil
}

func (h *connectHandler) GetVelocity(ctx context.Context, req *connect.Request[executionv1.GetVelocityRequest]) (*connect.Response[executionv1.GetVelocityResponse], error) {
	points, step, err := h.deps.Service.GetVelocity(ctx, req.Msg.GetPlanId())
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.GetVelocityResponse{Points: velocitiesToProto(points), Step: guidedStepToProto(step)}), nil
}
