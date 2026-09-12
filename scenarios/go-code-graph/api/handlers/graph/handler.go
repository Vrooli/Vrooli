package graph

import (
	"context"
	"log"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/schedule"

	intgraph "go-code-graph/internal/graph"
	intrewrite "go-code-graph/internal/rewrite"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
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
	Clock          schedule.Clock
	// FixturesDir is the directory holding golden determinism fixtures
	// (bas/fixtures by default, resolved relative to the server's working
	// directory). Empty falls back to "bas/fixtures". Powers the
	// ListFixtures / ValidateFixture RPCs.
	FixturesDir string
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
	if d.Clock == nil {
		d.Clock = schedule.System()
	}
	return &connectHandler{deps: d}
}

// Extract handles the GoCodeGraphService/Extract RPC. The flow is
// translate proto request → ExtractInput → Service.Extract → proto
// ExtractResponse. Internal errors are logged; client errors are not.
func (h *connectHandler) Extract(ctx context.Context, req *connect.Request[graphv1.ExtractRequest]) (*connect.Response[graphv1.ExtractResponse], error) {
	in := intgraph.ExtractInput{
		ModulePath:      req.Msg.GetModulePath(),
		IncludeVendor:   req.Msg.GetIncludeVendor(),
		Profile:         extractionProfile(req.Msg.GetProfile()),
		PackagePatterns: req.Msg.GetPackagePatterns(),
	}

	start := h.deps.Clock.Now()
	g, warnings, stats, err := h.deps.GraphService.ExtractWithStats(ctx, in)
	elapsedMs := schedule.Since(start).Milliseconds()
	if err != nil {
		connectErr := intgraph.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("graph.Extract(%q): %v", in.ModulePath, err)
		}
		return nil, connectErr
	}

	resp := &graphv1.ExtractResponse{
		Graph:              domainToProtoGraph(g),
		Warnings:           warningsToProto(warnings),
		ExtractionMs:       elapsedMs,
		GraphHash:          intgraph.GraphHash(g),
		FingerprintMs:      stats.Fingerprint.Milliseconds(),
		LoadMs:             stats.Load.Milliseconds(),
		NormalizeMs:        stats.Normalize.Milliseconds(),
		CacheHit:           stats.CacheHit,
		Profile:            profileToProto(in.Profile),
		OmittedInformation: omissionsToProto(in.Profile),
	}
	return connect.NewResponse(resp), nil
}

func profileToProto(profile intgraph.ExtractionProfile) graphv1.ExtractionProfile {
	switch profile {
	case intgraph.ExtractionProfileSemantic:
		return graphv1.ExtractionProfile_EXTRACTION_PROFILE_SEMANTIC
	case intgraph.ExtractionProfileStructural:
		return graphv1.ExtractionProfile_EXTRACTION_PROFILE_STRUCTURAL
	default:
		return graphv1.ExtractionProfile_EXTRACTION_PROFILE_FULL
	}
}

func omissionsToProto(profile intgraph.ExtractionProfile) []*commonv1.CodeGraphOmission {
	omissions := profile.OmittedInformation()
	out := make([]*commonv1.CodeGraphOmission, 0, len(omissions))
	for _, omission := range omissions {
		out = append(out, &commonv1.CodeGraphOmission{
			Capability: omission.Capability,
			Reason:     omission.Reason,
		})
	}
	return out
}

func extractionProfile(profile graphv1.ExtractionProfile) intgraph.ExtractionProfile {
	switch profile {
	case graphv1.ExtractionProfile_EXTRACTION_PROFILE_STRUCTURAL:
		return intgraph.ExtractionProfileStructural
	case graphv1.ExtractionProfile_EXTRACTION_PROFILE_SEMANTIC:
		return intgraph.ExtractionProfileSemantic
	default:
		return intgraph.ExtractionProfileFull
	}
}

// RewritePlan translates the proto request into a PlanInput, calls the
// rewrite Service, and projects the resulting Plan back onto the
// proto response.
func (h *connectHandler) RewritePlan(ctx context.Context, req *connect.Request[graphv1.RewritePlanRequest]) (*connect.Response[graphv1.RewritePlanResponse], error) {
	in := intrewrite.PlanInput{
		ModulePath: req.Msg.GetModulePath(),
		Operations: protoOperationsToDomain(req.Msg.GetOperations()),
	}

	plan, err := h.deps.RewriteService.Plan(ctx, in)
	if err != nil {
		connectErr := intrewrite.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("rewrite.Plan(%q): %v", in.ModulePath, err)
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
		ModulePath: req.Msg.GetModulePath(),
		PlanID:     intrewrite.PlanID(req.Msg.GetPlanId()),
		Apply:      req.Msg.GetApply(),
		DryRun:     dryRun,
	}

	result, err := h.deps.RewriteService.Apply(ctx, in)
	if err != nil {
		connectErr := intrewrite.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("rewrite.Apply(%q, plan=%q): %v", in.ModulePath, in.PlanID, err)
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
