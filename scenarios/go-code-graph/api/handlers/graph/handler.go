package graph

import (
	"context"
	"log"
	"strings"
	"time"

	"connectrpc.com/connect"

	intgraph "go-code-graph/internal/graph"
	intrewrite "go-code-graph/internal/rewrite"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph"
	"github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph/graph_v1connect"
)

// Deps is the wire-up the Connect handler needs. Kept narrow so tests
// can construct it without dragging in the modules registry.
//
// All three GoCodeGraphService RPCs route through this single handler
// because the proto is one service; Extract calls GraphService and
// RewritePlan/RewriteApply call RewriteService.
type Deps struct {
	GraphService   *intgraph.Service
	RewriteService *intrewrite.Service
	Logger         *log.Logger
}

// connectHandler implements graph_v1connect.GoCodeGraphServiceHandler.
type connectHandler struct {
	deps Deps
}

// NewConnectHandler returns a handler ready to mount via
// graph_v1connect.NewGoCodeGraphServiceHandler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// Extract handles the GoCodeGraphService/Extract RPC. The flow is
// translate proto request → ExtractInput → Service.Extract → proto
// ExtractResponse. Internal errors are logged; client errors are not.
func (h *connectHandler) Extract(ctx context.Context, req *connect.Request[graphv1.ExtractRequest]) (*connect.Response[graphv1.ExtractResponse], error) {
	in := intgraph.ExtractInput{
		ScenarioPath:  req.Msg.GetScenarioPath(),
		IncludeVendor: req.Msg.GetIncludeVendor(),
	}

	start := time.Now()
	g, warnings, err := h.deps.GraphService.Extract(ctx, in)
	elapsedMs := time.Since(start).Milliseconds()
	if err != nil {
		connectErr := intgraph.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("graph.Extract(%q): %v", in.ScenarioPath, err)
		}
		return nil, connectErr
	}

	resp := &graphv1.ExtractResponse{
		Graph:        domainToProtoGraph(g),
		Warnings:     warningsToProto(warnings),
		ExtractionMs: elapsedMs,
		GraphHash:    intgraph.GraphHash(g),
	}
	return connect.NewResponse(resp), nil
}

// RewritePlan translates the proto request into a PlanInput, calls the
// rewrite Service, and projects the resulting Plan back onto the
// proto response.
func (h *connectHandler) RewritePlan(ctx context.Context, req *connect.Request[graphv1.RewritePlanRequest]) (*connect.Response[graphv1.RewritePlanResponse], error) {
	in := intrewrite.PlanInput{
		ScenarioPath: req.Msg.GetScenarioPath(),
		Operations:   protoOperationsToDomain(req.Msg.GetOperations()),
	}

	plan, err := h.deps.RewriteService.Plan(ctx, in)
	if err != nil {
		connectErr := intrewrite.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("rewrite.Plan(%q): %v", in.ScenarioPath, err)
		}
		return nil, connectErr
	}

	resp := &graphv1.RewritePlanResponse{
		PlanId:               string(plan.ID),
		NormalizedOperations: domainOperationsToProto(plan.Operations),
	}
	return connect.NewResponse(resp), nil
}

// RewriteApply reads the X-Dry-Run header (per plan §8.5), translates
// the proto request into an ApplyInput, calls the rewrite Service, and
// projects the ApplyResult back onto the proto response.
func (h *connectHandler) RewriteApply(ctx context.Context, req *connect.Request[graphv1.RewriteApplyRequest]) (*connect.Response[graphv1.RewriteApplyResponse], error) {
	dryRun := strings.EqualFold(strings.TrimSpace(req.Header().Get("X-Dry-Run")), "true")
	in := intrewrite.ApplyInput{
		ScenarioPath: req.Msg.GetScenarioPath(),
		PlanID:       intrewrite.PlanID(req.Msg.GetPlanId()),
		Apply:        req.Msg.GetApply(),
		DryRun:       dryRun,
	}

	result, err := h.deps.RewriteService.Apply(ctx, in)
	if err != nil {
		connectErr := intrewrite.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("rewrite.Apply(%q, plan=%q): %v", in.ScenarioPath, in.PlanID, err)
		}
		return nil, connectErr
	}

	resp := &graphv1.RewriteApplyResponse{
		PlanId:  string(result.PlanID),
		Results: domainOperationResultsToProto(result.Results),
		DryRun:  result.DryRun,
	}
	return connect.NewResponse(resp), nil
}

// Compile-time assertion.
var _ graph_v1connect.GoCodeGraphServiceHandler = (*connectHandler)(nil)
