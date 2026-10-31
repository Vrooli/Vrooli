package validation

import (
	"context"
	"log"

	internalvalidation "plan-manager/internal/validation"

	"connectrpc.com/connect"

	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/validation"
)

// Deps wires the seams the Connect validation handler needs.
type Deps struct {
	Service internalvalidation.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the ValidationService Connect handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ResolveReferences(ctx context.Context, req *connect.Request[validationv1.ResolveReferencesRequest]) (*connect.Response[validationv1.ResolveReferencesResponse], error) {
	report, err := h.deps.Service.ResolveReferences(ctx, req.Msg.GetPlanId(), req.Msg.GetPhaseId())
	if err != nil {
		return nil, internalvalidation.ToConnectError(err)
	}
	return connect.NewResponse(&validationv1.ResolveReferencesResponse{
		References: referencesToProto(report.References),
		Degraded:   report.Degraded,
	}), nil
}

func (h *connectHandler) ComputeStaleness(ctx context.Context, req *connect.Request[validationv1.ComputeStalenessRequest]) (*connect.Response[validationv1.ComputeStalenessResponse], error) {
	report, err := h.deps.Service.ComputeStaleness(ctx, req.Msg.GetPlanId(), req.Msg.GetPhaseId())
	if err != nil {
		return nil, internalvalidation.ToConnectError(err)
	}
	return connect.NewResponse(&validationv1.ComputeStalenessResponse{
		Overall:    stalenessToProto(report.Overall),
		References: referencesToProto(report.References),
		Degraded:   report.Degraded,
	}), nil
}

func (h *connectHandler) StartValidation(ctx context.Context, req *connect.Request[validationv1.StartValidationRequest]) (*connect.Response[validationv1.StartValidationResponse], error) {
	testRuns := make([]internalvalidation.TestRunEvidence, 0, len(req.Msg.GetTestRuns()))
	for _, run := range req.Msg.GetTestRuns() {
		testRuns = append(testRuns, internalvalidation.TestRunEvidence{Scenario: run.GetScenario(), RunID: run.GetRunId()})
	}
	op, deduplicated, err := h.deps.Service.StartValidationTicket(ctx, internalvalidation.ValidationTicketRequest{PlanID: req.Msg.GetPlanId(), PhaseID: req.Msg.GetPhaseId(), IdempotencyKey: req.Msg.GetIdempotencyKey(), ExecutionID: req.Msg.GetExecutionId(), ScopeGeneration: int(req.Msg.GetScopeGeneration()), SelectedMembers: req.Msg.GetMember(), TestRuns: testRuns})
	if err != nil {
		return nil, internalvalidation.ToConnectError(err)
	}
	return connect.NewResponse(&validationv1.StartValidationResponse{
		Operation: operationToProto(op), Deduplicated: deduplicated,
	}), nil
}

func (h *connectHandler) GetValidationOperation(ctx context.Context, req *connect.Request[validationv1.GetValidationOperationRequest]) (*connect.Response[validationv1.GetValidationOperationResponse], error) {
	op, err := h.deps.Service.GetValidationOperation(ctx, req.Msg.GetOperationId())
	if err != nil {
		return nil, internalvalidation.ToConnectError(err)
	}
	return connect.NewResponse(&validationv1.GetValidationOperationResponse{Operation: operationToProto(op)}), nil
}

func (h *connectHandler) SyncValidation(ctx context.Context, req *connect.Request[validationv1.SyncValidationRequest]) (*connect.Response[validationv1.SyncValidationResponse], error) {
	op, err := h.deps.Service.SyncValidation(ctx, req.Msg.GetOperationId())
	if err != nil {
		return nil, internalvalidation.ToConnectError(err)
	}
	return connect.NewResponse(&validationv1.SyncValidationResponse{Operation: operationToProto(op)}), nil
}
