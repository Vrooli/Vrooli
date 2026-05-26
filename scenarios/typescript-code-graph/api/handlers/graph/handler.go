package graph

import (
	"context"
	"log"
	"time"

	"connectrpc.com/connect"

	rewriteH "typescript-code-graph/handlers/rewrite"
	intgraph "typescript-code-graph/internal/graph"
	intrewrite "typescript-code-graph/internal/rewrite"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph"
	"github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph/graph_v1connect"
)

// Deps is the wire-up the Connect handler needs. Kept narrow so tests
// can construct it without dragging in the modules registry.
//
// All three TypeScriptCodeGraphService RPCs route through this single
// handler because the proto is one service. Extract calls the graph
// Service; RewritePlan / RewriteApply delegate to the rewrite handler
// package, which translates proto ↔ domain and drives
// rewrite.Service.
type Deps struct {
	GraphService   *intgraph.Service
	RewriteService *intrewrite.Service
	Logger         *log.Logger
	// FixturesDir overrides the golden-fixtures root for ListFixtures /
	// ValidateFixture. Empty falls back to the env var / scenario-relative
	// default (see fixtures_handler.go).
	FixturesDir string
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler returns a handler ready to mount via
// graph_v1connect.NewTypeScriptCodeGraphServiceHandler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// Extract handles the TypeScriptCodeGraphService/Extract RPC. The flow
// is translate proto request → ExtractInput → Service.Extract → proto
// ExtractResponse. ExtractionMs is measured at this layer (the domain
// is forbidden from importing time per the substrate boundary).
func (h *connectHandler) Extract(ctx context.Context, req *connect.Request[graphv1.ExtractRequest]) (*connect.Response[graphv1.ExtractResponse], error) {
	in := requestToInput(req.Msg)
	start := time.Now()
	out, err := h.deps.GraphService.Extract(ctx, in)
	elapsedMs := time.Since(start).Milliseconds()
	if err != nil {
		return nil, h.toClientError(in.ScenarioPath, err)
	}

	resp := &graphv1.ExtractResponse{
		Graph:            domainToProtoGraph(out.Graph),
		Warnings:         warningsToProto(out.Warnings),
		ExtractionMs:     elapsedMs,
		GraphHash:        out.GraphHash,
		SidecarRequestId: out.SidecarRequestID,
	}
	return connect.NewResponse(resp), nil
}

// RewritePlan delegates to the rewrite handler package. The proto
// service is one Connect mount; this method is the thin per-RPC
// entrypoint that thread-locks the rewrite.Service dependency.
func (h *connectHandler) RewritePlan(ctx context.Context, req *connect.Request[graphv1.RewritePlanRequest]) (*connect.Response[graphv1.RewritePlanResponse], error) {
	return rewriteH.RewritePlan(ctx, req, h.deps.RewriteService)
}

// RewriteApply delegates to the rewrite handler package. The X-Dry-Run
// header is read here (transport-layer concern) and threaded through
// as a bool so the rewrite domain never has to learn about HTTP.
func (h *connectHandler) RewriteApply(ctx context.Context, req *connect.Request[graphv1.RewriteApplyRequest]) (*connect.Response[graphv1.RewriteApplyResponse], error) {
	dryRun := req.Header().Get("X-Dry-Run") == "true"
	return rewriteH.RewriteApply(ctx, req, h.deps.RewriteService, dryRun)
}

// toClientError translates a domain error into a connect.Error and
// logs internal failures (client-attributable codes are not logged so
// a buggy caller cannot spam the operator's log).
func (h *connectHandler) toClientError(scenarioPath string, err error) error {
	connectErr := intgraph.ToConnectError(err)
	if connect.CodeOf(connectErr) == connect.CodeInternal {
		h.deps.Logger.Printf("graph.Extract(%q): %v", scenarioPath, err)
	}
	return connectErr
}

// Compile-time assertion: this handler satisfies the generated Connect
// server interface. If a proto rpc is added/removed/renamed this line
// fails first.
var _ graph_v1connect.TypeScriptCodeGraphServiceHandler = (*connectHandler)(nil)
