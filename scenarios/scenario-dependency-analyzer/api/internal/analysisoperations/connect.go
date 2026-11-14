package analysisoperations

import (
	"context"
	"errors"
	"net/http"
	"time"

	"connectrpc.com/connect"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/analysis_operations"
	rpc "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/analysis_operations/analysis_operations_v1connect"
)

type ConnectHandler struct{ manager *Manager }

func NewHTTPHandler(manager *Manager) (string, http.Handler) {
	return rpc.NewAnalysisOperationsServiceHandler(&ConnectHandler{manager: manager})
}

// ClearWaitDeadline must wrap the raw HTTP writer, before router wrappers.
// Ordinary requests retain the shared server timeout; observers may wait once.
func ClearWaitDeadline(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == rpc.AnalysisOperationsServiceWaitOperationProcedure {
			_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
		}
		next.ServeHTTP(w, r)
	})
}

func (h *ConnectHandler) StartOperation(ctx context.Context, req *connect.Request[pb.StartOperationRequest]) (*connect.Response[pb.StartOperationResponse], error) {
	kind := FleetAnalysis
	switch req.Msg.GetKind() {
	case pb.AnalysisOperationKind_ANALYSIS_OPERATION_KIND_UNSPECIFIED, pb.AnalysisOperationKind_ANALYSIS_OPERATION_KIND_FLEET_ANALYSIS:
	case pb.AnalysisOperationKind_ANALYSIS_OPERATION_KIND_GRAPH_REBUILD:
		kind = GraphRebuild
	default:
		return nil, wireError(ErrInvalid)
	}
	op, reused, err := h.manager.Submit(ctx, req.Msg.GetClientOperationId(), Spec{Kind: kind, ApplyGraph: req.Msg.GetApplyGraph()})
	if err != nil {
		return nil, wireError(err)
	}
	return connect.NewResponse(&pb.StartOperationResponse{Operation: operationProto(op), Reused: reused}), nil
}

func (h *ConnectHandler) GetOperation(ctx context.Context, req *connect.Request[pb.GetOperationRequest]) (*connect.Response[pb.GetOperationResponse], error) {
	op, err := h.manager.Get(ctx, req.Msg.GetOperationId())
	if err != nil {
		return nil, wireError(err)
	}
	return connect.NewResponse(&pb.GetOperationResponse{Operation: operationProto(op)}), nil
}

func (h *ConnectHandler) WaitOperation(ctx context.Context, req *connect.Request[pb.WaitOperationRequest]) (*connect.Response[pb.WaitOperationResponse], error) {
	op, err := h.manager.Wait(ctx, req.Msg.GetOperationId())
	if err != nil {
		return nil, wireError(err)
	}
	return connect.NewResponse(&pb.WaitOperationResponse{Operation: operationProto(op)}), nil
}

func wireError(err error) error {
	code := connect.CodeInternal
	switch {
	case errors.Is(err, ErrInvalid):
		code = connect.CodeInvalidArgument
	case errors.Is(err, ErrConflict):
		code = connect.CodeAlreadyExists
	case errors.Is(err, ErrNotFound):
		code = connect.CodeNotFound
	case errors.Is(err, ErrCapacity):
		code = connect.CodeResourceExhausted
	case errors.Is(err, ErrStopped):
		code = connect.CodeUnavailable
	case errors.Is(err, context.Canceled):
		code = connect.CodeCanceled
	case errors.Is(err, context.DeadlineExceeded):
		code = connect.CodeDeadlineExceeded
	}
	return connect.NewError(code, err)
}

func stateProto(state State) pb.AnalysisOperationState {
	switch state {
	case Queued:
		return pb.AnalysisOperationState_ANALYSIS_OPERATION_STATE_QUEUED
	case Running:
		return pb.AnalysisOperationState_ANALYSIS_OPERATION_STATE_RUNNING
	case Succeeded:
		return pb.AnalysisOperationState_ANALYSIS_OPERATION_STATE_SUCCEEDED
	case Partial:
		return pb.AnalysisOperationState_ANALYSIS_OPERATION_STATE_PARTIAL
	case Failed:
		return pb.AnalysisOperationState_ANALYSIS_OPERATION_STATE_FAILED
	case Interrupted:
		return pb.AnalysisOperationState_ANALYSIS_OPERATION_STATE_INTERRUPTED
	default:
		return pb.AnalysisOperationState_ANALYSIS_OPERATION_STATE_UNSPECIFIED
	}
}

func operationProto(op Operation) *pb.AnalysisOperation {
	kind := pb.AnalysisOperationKind_ANALYSIS_OPERATION_KIND_FLEET_ANALYSIS
	if op.Spec.Kind == GraphRebuild {
		kind = pb.AnalysisOperationKind_ANALYSIS_OPERATION_KIND_GRAPH_REBUILD
	}
	result := &pb.AnalysisOperation{Id: op.ID, Kind: kind, ApplyGraph: op.Spec.ApplyGraph, State: stateProto(op.State), CreatedAt: stamp(op.CreatedAt), ScenariosTotal: uint32(op.Total), ScenariosCompleted: uint32(op.Completed), ScenariosFailed: uint32(op.Failures), Error: op.Error, DetailsPruned: op.DetailsPruned}
	if op.StartedAt != nil {
		result.StartedAt = stamp(*op.StartedAt)
	}
	if op.CompletedAt != nil {
		result.CompletedAt = stamp(*op.CompletedAt)
	}
	for _, item := range op.Scenarios {
		result.Scenarios = append(result.Scenarios, &pb.ScenarioAnalysisOutcome{Scenario: item.Scenario, State: stateProto(item.State), DetectedResources: uint32(item.DetectedResources), ScenarioDependencies: uint32(item.ScenarioDependencies), DeploymentDependencies: uint32(item.DeploymentDependencies), DurationMs: uint64(item.Duration.Milliseconds()), Error: item.Error})
	}
	if op.Graph != nil {
		result.Graph = &pb.GraphRebuildOutcome{Applied: op.Graph.Applied, Edges: uint32(op.Graph.Edges), ScenarioEdges: uint32(op.Graph.ScenarioEdges), ResourceEdges: uint32(op.Graph.ResourceEdges), DegradedSources: op.Graph.DegradedSources}
	}
	return result
}
