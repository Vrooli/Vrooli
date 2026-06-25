package execution

import (
	"context"
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
	e, err := h.deps.Service.Start(ctx, req.Msg.GetPlanId(), req.Msg.GetRunId())
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.StartResponse{Execution: executionToProto(e)}), nil
}

func (h *connectHandler) GetStatus(ctx context.Context, req *connect.Request[executionv1.GetStatusRequest]) (*connect.Response[executionv1.GetStatusResponse], error) {
	e, pctx, err := h.deps.Service.GetStatus(ctx, req.Msg.GetExecutionId())
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.GetStatusResponse{
		Execution: executionToProto(e),
		Context:   phaseContextToProto(pctx),
	}), nil
}

func (h *connectHandler) GetNext(ctx context.Context, req *connect.Request[executionv1.GetNextRequest]) (*connect.Response[executionv1.GetNextResponse], error) {
	pctx, complete, err := h.deps.Service.GetNext(ctx, req.Msg.GetExecutionId())
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.GetNextResponse{
		Context:  phaseContextToProto(pctx),
		Complete: complete,
	}), nil
}

func (h *connectHandler) TransitionPhase(ctx context.Context, req *connect.Request[executionv1.TransitionPhaseRequest]) (*connect.Response[executionv1.TransitionPhaseResponse], error) {
	e, plan, err := h.deps.Service.TransitionPhase(ctx, req.Msg.GetExecutionId(), req.Msg.GetPhaseId(), phaseStatusFromProto(req.Msg.GetToStatus()))
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.TransitionPhaseResponse{
		Execution: executionToProto(e),
		Plan:      planToProto(plan),
	}), nil
}

func (h *connectHandler) RecordDecision(ctx context.Context, req *connect.Request[executionv1.RecordDecisionRequest]) (*connect.Response[executionv1.RecordDecisionResponse], error) {
	d, err := h.deps.Service.RecordDecision(ctx, req.Msg.GetExecutionId(), req.Msg.GetPhaseId(), req.Msg.GetSummary(), req.Msg.GetDetail())
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.RecordDecisionResponse{Decision: decisionToProto(d)}), nil
}

func (h *connectHandler) RecordFinding(ctx context.Context, req *connect.Request[executionv1.RecordFindingRequest]) (*connect.Response[executionv1.RecordFindingResponse], error) {
	f, err := h.deps.Service.RecordFinding(ctx, req.Msg.GetExecutionId(), req.Msg.GetPhaseId(), req.Msg.GetTitle(), req.Msg.GetDetail())
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.RecordFindingResponse{Finding: findingToProto(f)}), nil
}

func (h *connectHandler) Complete(ctx context.Context, req *connect.Request[executionv1.CompleteRequest]) (*connect.Response[executionv1.CompleteResponse], error) {
	handoff, nudges, err := h.deps.Service.Complete(ctx, req.Msg.GetExecutionId(), internalexecution.CompletionInputs{
		Tokens:     req.Msg.GetTokens(),
		Iterations: req.Msg.GetIterations(),
	})
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.CompleteResponse{
		Handoff: handoffToProto(handoff),
		Nudges:  nudgesToProto(nudges),
	}), nil
}

func (h *connectHandler) GetHandoff(ctx context.Context, req *connect.Request[executionv1.GetHandoffRequest]) (*connect.Response[executionv1.GetHandoffResponse], error) {
	handoff, err := h.deps.Service.GetHandoff(ctx, req.Msg.GetExecutionId())
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.GetHandoffResponse{Handoff: handoffToProto(handoff)}), nil
}

func (h *connectHandler) ListCandidateFindings(ctx context.Context, req *connect.Request[executionv1.ListCandidateFindingsRequest]) (*connect.Response[executionv1.ListCandidateFindingsResponse], error) {
	findings, err := h.deps.Service.ListCandidateFindings(ctx, req.Msg.GetExecutionId())
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.ListCandidateFindingsResponse{Findings: findingsToProto(findings)}), nil
}

func (h *connectHandler) TriageFinding(ctx context.Context, req *connect.Request[executionv1.TriageFindingRequest]) (*connect.Response[executionv1.TriageFindingResponse], error) {
	f, err := h.deps.Service.TriageFinding(ctx, req.Msg.GetFindingId(), triageFromProto(req.Msg.GetTriage()))
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.TriageFindingResponse{Finding: findingToProto(f)}), nil
}

func (h *connectHandler) GetVelocity(ctx context.Context, req *connect.Request[executionv1.GetVelocityRequest]) (*connect.Response[executionv1.GetVelocityResponse], error) {
	points, err := h.deps.Service.GetVelocity(ctx, req.Msg.GetPlanId())
	if err != nil {
		return nil, internalexecution.ToConnectError(err)
	}
	return connect.NewResponse(&executionv1.GetVelocityResponse{Points: velocitiesToProto(points)}), nil
}
