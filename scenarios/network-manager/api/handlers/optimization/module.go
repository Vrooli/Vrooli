package optimization

import (
	"context"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"network-manager/internal/module"

	optimizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/optimization"
	optimizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/optimization/optimization_v1connect"
)

type handler struct{}

func Module() module.Module {
	path, h := optimizationconnect.NewOptimizationServiceHandler(&handler{})
	return module.Module{Name: "optimization", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
	}, Endpoints: Endpoints}
}

func Schema() string { return "" }

func (h *handler) CreateOptimizationRun(_ context.Context, req *connect.Request[optimizationv1.CreateOptimizationRunRequest]) (*connect.Response[optimizationv1.CreateOptimizationRunResponse], error) {
	profile := req.Msg.GetScoringProfile()
	if profile == "" {
		profile = "balanced-reliability"
	}
	return connect.NewResponse(&optimizationv1.CreateOptimizationRunResponse{Run: run("optimization-preview", profile, "preview")}), nil
}

func (h *handler) RunCandidate(context.Context, *connect.Request[optimizationv1.RunCandidateRequest]) (*connect.Response[optimizationv1.RunCandidateResponse], error) {
	return connect.NewResponse(&optimizationv1.RunCandidateResponse{Run: run("optimization-preview", "balanced-reliability", "candidate_preview")}), nil
}

func (h *handler) ScoreCandidates(context.Context, *connect.Request[optimizationv1.ScoreCandidatesRequest]) (*connect.Response[optimizationv1.ScoreCandidatesResponse], error) {
	return connect.NewResponse(&optimizationv1.ScoreCandidatesResponse{Run: run("optimization-preview", "balanced-reliability", "scored")}), nil
}

func (h *handler) ApproveCandidate(context.Context, *connect.Request[optimizationv1.ApproveCandidateRequest]) (*connect.Response[optimizationv1.ApproveCandidateResponse], error) {
	return connect.NewResponse(&optimizationv1.ApproveCandidateResponse{Run: run("optimization-preview", "balanced-reliability", "approval_required")}), nil
}

func (h *handler) RollbackOptimization(context.Context, *connect.Request[optimizationv1.RollbackOptimizationRequest]) (*connect.Response[optimizationv1.RollbackOptimizationResponse], error) {
	return connect.NewResponse(&optimizationv1.RollbackOptimizationResponse{Run: run("optimization-preview", "balanced-reliability", "rollback_preview")}), nil
}

func run(id, profile, status string) *optimizationv1.OptimizationRun {
	return &optimizationv1.OptimizationRun{Id: id, Status: status, ScoringProfile: profile, Recommendation: "No candidate applied in scaffold mode.", Candidates: []*optimizationv1.Candidate{{Id: "dns-upstream-preview", Description: "Compare resolver upstreams after adapter implementation.", Status: "not_run", ApprovalRequired: true, Evidence: []string{"Requires resolver adapter."}}}}
}

var Endpoints = []module.EndpointDescriptor{
	connectEndpoint("optimization_create", optimizationconnect.OptimizationServiceCreateOptimizationRunProcedure, "Create optimization run"),
	connectEndpoint("optimization_candidate_run", optimizationconnect.OptimizationServiceRunCandidateProcedure, "Run optimization candidate"),
	connectEndpoint("optimization_score", optimizationconnect.OptimizationServiceScoreCandidatesProcedure, "Score optimization candidates"),
	connectEndpoint("optimization_approve", optimizationconnect.OptimizationServiceApproveCandidateProcedure, "Approve optimization candidate"),
	connectEndpoint("optimization_rollback", optimizationconnect.OptimizationServiceRollbackOptimizationProcedure, "Rollback optimization"),
}

func connectEndpoint(id, path, summary string) module.EndpointDescriptor {
	return module.EndpointDescriptor{ID: id, Path: path, Method: "POST", Summary: summary, Category: "optimization", Request: &module.Schema{Type: "object", Properties: map[string]string{}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"run": "OptimizationRun"}}}
}
