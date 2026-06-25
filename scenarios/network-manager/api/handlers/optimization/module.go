package optimization

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	domainadapters "network-manager/internal/adapters"
	"network-manager/internal/module"
	domainoptimization "network-manager/internal/optimization"
	domainpolicy "network-manager/internal/policy"
	domainresolver "network-manager/internal/resolver"
	domainsnapshot "network-manager/internal/snapshot"

	optimizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/optimization"
	optimizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/optimization/optimization_v1connect"
)

type handler struct {
	service *domainoptimization.Service
}

func Module(db domainoptimization.SQLExecutor) module.Module {
	service := newService(db)
	path, h := optimizationconnect.NewOptimizationServiceHandler(&handler{service: service})
	return module.Module{Name: "optimization", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
	}, Endpoints: Endpoints}
}

func newService(db domainoptimization.SQLExecutor) *domainoptimization.Service {
	snapshotRepo := domainsnapshot.NewSQLiteRepository(db)
	snapshotService := domainsnapshot.NewService(domainsnapshot.Config{Repo: snapshotRepo})
	resolverRepo := domainresolver.NewSQLiteRepository(db)
	adapterService := domainadapters.NewService(domainadapters.Config{
		Repo: domainadapters.NewSQLiteRepository(db),
		Registry: domainadapters.ResolverAwareRegistry{
			Base:             domainadapters.NewStaticRegistry(),
			ResolverBackends: resolverRepo,
		},
	})
	policyAdapter := domainpolicy.NewAdGuardResolverPolicyAdapter(resolverRepo, domainresolver.NewVaultSecretResolver())
	return domainoptimization.NewService(domainoptimization.Config{
		Repo:         domainoptimization.NewSQLiteRepository(db),
		Capabilities: adapterService,
		Snapshots:    snapshotRepo,
		Runner:       snapshotService,
		Applier:      domainoptimization.AdGuardPolicyApplier{Adapter: policyAdapter},
	})
}

func Schema() string { return domainoptimization.Schema() }

func (h *handler) CreateOptimizationRun(ctx context.Context, req *connect.Request[optimizationv1.CreateOptimizationRunRequest]) (*connect.Response[optimizationv1.CreateOptimizationRunResponse], error) {
	run, err := h.service.CreateRun(ctx, req.Msg.GetScoringProfile(), req.Msg.GetDryRun())
	if err != nil {
		return nil, optimizationError(err)
	}
	return connect.NewResponse(&optimizationv1.CreateOptimizationRunResponse{Run: toProtoRun(run)}), nil
}

func (h *handler) RunCandidate(ctx context.Context, req *connect.Request[optimizationv1.RunCandidateRequest]) (*connect.Response[optimizationv1.RunCandidateResponse], error) {
	run, err := h.service.RunCandidate(ctx, req.Msg.GetRunId(), req.Msg.GetCandidateId())
	if err != nil {
		return nil, optimizationError(err)
	}
	return connect.NewResponse(&optimizationv1.RunCandidateResponse{Run: toProtoRun(run)}), nil
}

func (h *handler) ScoreCandidates(ctx context.Context, req *connect.Request[optimizationv1.ScoreCandidatesRequest]) (*connect.Response[optimizationv1.ScoreCandidatesResponse], error) {
	run, err := h.service.Score(ctx, req.Msg.GetRunId())
	if err != nil {
		return nil, optimizationError(err)
	}
	return connect.NewResponse(&optimizationv1.ScoreCandidatesResponse{Run: toProtoRun(run)}), nil
}

func (h *handler) ApproveCandidate(ctx context.Context, req *connect.Request[optimizationv1.ApproveCandidateRequest]) (*connect.Response[optimizationv1.ApproveCandidateResponse], error) {
	run, err := h.service.Approve(ctx, req.Msg.GetRunId(), req.Msg.GetCandidateId(), req.Msg.GetApproved())
	if err != nil {
		return nil, optimizationError(err)
	}
	return connect.NewResponse(&optimizationv1.ApproveCandidateResponse{Run: toProtoRun(run)}), nil
}

func (h *handler) RollbackOptimization(ctx context.Context, req *connect.Request[optimizationv1.RollbackOptimizationRequest]) (*connect.Response[optimizationv1.RollbackOptimizationResponse], error) {
	run, err := h.service.Rollback(ctx, req.Msg.GetRunId())
	if err != nil {
		return nil, optimizationError(err)
	}
	return connect.NewResponse(&optimizationv1.RollbackOptimizationResponse{Run: toProtoRun(run)}), nil
}

func optimizationError(err error) error {
	switch {
	case errors.Is(err, domainoptimization.ErrNotFound), errors.Is(err, domainoptimization.ErrCandidateNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, domainoptimization.ErrBaselineRequired):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
}

func toProtoRun(run domainoptimization.Run) *optimizationv1.OptimizationRun {
	out := &optimizationv1.OptimizationRun{
		Id:             run.ID,
		Status:         run.Status,
		ScoringProfile: run.ScoringProfile,
		Recommendation: run.Recommendation,
		Candidates:     make([]*optimizationv1.Candidate, 0, len(run.Candidates)),
	}
	for _, candidate := range run.Candidates {
		out.Candidates = append(out.Candidates, &optimizationv1.Candidate{
			Id:               candidate.ID,
			Description:      candidate.Description,
			Status:           candidate.Status,
			Score:            candidate.Score,
			Evidence:         candidate.Evidence,
			ApprovalRequired: candidate.ApprovalRequired,
		})
	}
	return out
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
