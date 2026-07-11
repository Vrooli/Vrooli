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

func (h *connectHandler) DeriveBaselineScope(ctx context.Context, req *connect.Request[validationv1.DeriveBaselineScopeRequest]) (*connect.Response[validationv1.DeriveBaselineScopeResponse], error) {
	scope, err := h.deps.Service.DeriveBaselineScope(ctx, req.Msg.GetPlanId(), req.Msg.GetPhaseId())
	if err != nil {
		return nil, internalvalidation.ToConnectError(err)
	}
	return connect.NewResponse(&validationv1.DeriveBaselineScopeResponse{
		Commands:  scope.Commands,
		Locations: scope.Locations,
	}), nil
}

func (h *connectHandler) StartValidation(ctx context.Context, req *connect.Request[validationv1.StartValidationRequest]) (*connect.Response[validationv1.StartValidationResponse], error) {
	op, deduplicated, err := h.deps.Service.StartValidation(ctx, req.Msg.GetPlanId(), req.Msg.GetPhaseId(), req.Msg.GetIdempotencyKey())
	if err != nil {
		return nil, internalvalidation.ToConnectError(err)
	}
	return connect.NewResponse(&validationv1.StartValidationResponse{
		Operation: operationToProto(op), Deduplicated: deduplicated,
	}), nil
}

func (h *connectHandler) GetValidationOperation(ctx context.Context, req *connect.Request[validationv1.GetValidationOperationRequest]) (*connect.Response[validationv1.GetValidationOperationResponse], error) {
	op, err := h.deps.Service.GetValidationOperation(ctx, req.Msg.GetOperationId(), req.Msg.GetWait())
	if err != nil {
		return nil, internalvalidation.ToConnectError(err)
	}
	return connect.NewResponse(&validationv1.GetValidationOperationResponse{Operation: operationToProto(op)}), nil
}

func (h *connectHandler) WaitValidationOperation(ctx context.Context, req *connect.Request[validationv1.GetValidationOperationRequest]) (*connect.Response[validationv1.GetValidationOperationResponse], error) {
	req.Msg.Wait = true
	return h.GetValidationOperation(ctx, req)
}

func (h *connectHandler) ResumeValidationOperation(ctx context.Context, req *connect.Request[validationv1.GetValidationOperationRequest]) (*connect.Response[validationv1.GetValidationOperationResponse], error) {
	req.Msg.Wait = true
	return h.GetValidationOperation(ctx, req)
}

func (h *connectHandler) RunValidation(ctx context.Context, req *connect.Request[validationv1.RunValidationRequest]) (*connect.Response[validationv1.RunValidationResponse], error) {
	res, err := h.deps.Service.RunValidation(ctx, req.Msg.GetPlanId(), req.Msg.GetPhaseId())
	if err != nil {
		return nil, internalvalidation.ToConnectError(err)
	}
	return connect.NewResponse(&validationv1.RunValidationResponse{Result: resultToProto(res)}), nil
}

func (h *connectHandler) VerifyDefinitionOfDone(ctx context.Context, req *connect.Request[validationv1.VerifyDefinitionOfDoneRequest]) (*connect.Response[validationv1.VerifyDefinitionOfDoneResponse], error) {
	res, met, err := h.deps.Service.VerifyDefinitionOfDone(ctx, req.Msg.GetPlanId())
	if err != nil {
		return nil, internalvalidation.ToConnectError(err)
	}
	return connect.NewResponse(&validationv1.VerifyDefinitionOfDoneResponse{
		Result: resultToProto(res),
		DodMet: met,
	}), nil
}
