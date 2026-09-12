package rewrite

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	rewritedom "typescript-code-graph/internal/rewrite"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph"
)

// RewritePlan is the per-RPC translator the graph handler delegates
// to. Validates the proto request, calls rewrite.Service.Plan, maps
// errors to Connect codes.
//
// The graph handler owns the function-signature side of the Connect
// interface (because all three RPCs share one service); this package
// owns the proto ↔ domain shape work.
func RewritePlan(ctx context.Context, req *connect.Request[graphv1.RewritePlanRequest], svc *rewritedom.Service) (*connect.Response[graphv1.RewritePlanResponse], error) {
	if svc == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("rewrite service not wired"))
	}
	in := rewritedom.PlanInput{
		ProjectPath: req.Msg.GetProjectPath(),
		Operations:  protoToDomainOperations(req.Msg.GetOperations()),
	}
	out, err := svc.Plan(ctx, in)
	if err != nil {
		return nil, rewritedom.ToConnectError(err)
	}
	return connect.NewResponse(&graphv1.RewritePlanResponse{
		PlanId:               string(out.PlanID),
		NormalizedOperations: domainToProtoOperations(out.NormalizedOperations),
	}), nil
}

// RewriteApply is the per-RPC translator the graph handler delegates
// to. The dryRun bool is supplied by the caller — the graph handler
// reads it from the X-Dry-Run request header before calling here.
//
// Per the proto comment, apply=false is rejected with InvalidArgument
// because dry-run is signaled by the header, not the bool. apply=true
// + X-Dry-Run:true is the canonical dry-run shape.
func RewriteApply(ctx context.Context, req *connect.Request[graphv1.RewriteApplyRequest], svc *rewritedom.Service, dryRun bool) (*connect.Response[graphv1.RewriteApplyResponse], error) {
	if svc == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("rewrite service not wired"))
	}
	if !req.Msg.GetApply() {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("apply must be true; dry-run is signaled via the X-Dry-Run header"))
	}
	in := rewritedom.ApplyInput{
		ProjectPath: req.Msg.GetProjectPath(),
		PlanID:      rewritedom.PlanID(req.Msg.GetPlanId()),
		DryRun:      dryRun,
	}
	out, err := svc.Apply(ctx, in)
	if err != nil {
		return nil, rewritedom.ToConnectError(err)
	}
	return connect.NewResponse(&graphv1.RewriteApplyResponse{
		PlanId:  string(out.PlanID),
		Results: domainResultsToProto(out.Results),
		DryRun:  out.DryRun,
	}), nil
}
