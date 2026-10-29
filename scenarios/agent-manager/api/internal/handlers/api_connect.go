package handlers

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/permissionpolicy"
	"agent-manager/internal/protoconv"
	"agent-manager/internal/rolepolicy"
	"agent-manager/internal/supervision"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
)

// AgentManagerConnectHandler mounts the typed agent-manager API without
// duplicating orchestration logic. The generated unimplemented embedding keeps
// the large compatibility service additive: methods are admitted deliberately
// as their typed adapters are implemented, while ListRuns is the first
// cross-scenario readiness-board contract.
type AgentManagerConnectHandler struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
	h               *Handler
	supervision     *supervision.Service
	watchActionAuth WatchActionAuthorizer
	managedTeams    ManagedTeamReader
	exactVerifier   *supervision.ExactVerifier
}

type WatchActionAuthorizer interface {
	AuthorizeWatchAction(context.Context, string, *domainpb.RequestCohortWatchActionRequest) error
}

var (
	ErrWatchActionUnauthenticated = errors.New("watch action caller is unauthenticated")
	ErrWatchActionForbidden       = errors.New("watch action caller is not authorized")
)

func NewAgentManagerConnectHandler(h *Handler, watchServices ...*supervision.Service) *AgentManagerConnectHandler {
	handler := &AgentManagerConnectHandler{h: h}
	if len(watchServices) > 0 {
		handler.supervision = watchServices[0]
	}
	return handler
}

func (h *AgentManagerConnectHandler) ListExecutionOptions(ctx context.Context, req *connect.Request[api.ListExecutionOptionsRequest]) (*connect.Response[api.ListExecutionOptionsResponse], error) {
	if h.h == nil || h.h.svc.ExecutionOptionsService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("execution options service unavailable"))
	}
	roleRef := ""
	if req != nil && req.Msg != nil {
		roleRef = req.Msg.RoleRef
	}
	options, err := h.h.svc.ListExecutionOptions(ctx, roleRef)
	if err != nil {
		return nil, workflowConnectError(err)
	}
	return connect.NewResponse(executionOptionsToProto(options)), nil
}

func (h *AgentManagerConnectHandler) SetWatchActionAuthorizer(authorizer WatchActionAuthorizer) {
	h.watchActionAuth = authorizer
}

func (h *AgentManagerConnectHandler) CreateCohortWatch(ctx context.Context, req *connect.Request[domainpb.CreateCohortWatchRequest]) (*connect.Response[domainpb.CohortWatch], error) {
	if h.supervision == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("cohort supervision is unavailable"))
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	watch, _, err := h.supervision.Create(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(watch), nil
}

func (h *AgentManagerConnectHandler) GetCohortWatch(ctx context.Context, req *connect.Request[domainpb.GetCohortWatchRequest]) (*connect.Response[domainpb.CohortWatch], error) {
	if h.supervision == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("cohort supervision is unavailable"))
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	watch, err := h.supervision.Get(ctx, req.Msg.GetWatchId())
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(watch), nil
}

func (h *AgentManagerConnectHandler) InspectCohortWatch(ctx context.Context, req *connect.Request[domainpb.InspectCohortWatchRequest]) (*connect.Response[domainpb.InspectCohortWatchResponse], error) {
	if h.supervision == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("cohort supervision is unavailable"))
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	response, err := h.supervision.Inspect(ctx, req.Msg)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(response), nil
}

func (h *AgentManagerConnectHandler) ListCohortWatches(ctx context.Context, req *connect.Request[domainpb.ListCohortWatchesRequest]) (*connect.Response[domainpb.ListCohortWatchesResponse], error) {
	if h.supervision == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("cohort supervision is unavailable"))
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	response, err := h.supervision.List(ctx, req.Msg)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(response), nil
}

func (h *AgentManagerConnectHandler) WaitCohortWatch(ctx context.Context, req *connect.Request[domainpb.WaitCohortWatchRequest]) (*connect.Response[domainpb.WaitCohortWatchResponse], error) {
	if h.supervision == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("cohort supervision is unavailable"))
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	response, err := h.supervision.Wait(ctx, req.Msg)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(response), nil
}

func (h *AgentManagerConnectHandler) CancelCohortWatch(ctx context.Context, req *connect.Request[domainpb.CancelCohortWatchRequest]) (*connect.Response[domainpb.CohortWatch], error) {
	if h.supervision == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("cohort supervision is unavailable"))
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	watch, err := h.supervision.Cancel(ctx, req.Msg)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(watch), nil
}

func (h *AgentManagerConnectHandler) RequestCohortWatchAction(ctx context.Context, req *connect.Request[domainpb.RequestCohortWatchActionRequest]) (*connect.Response[domainpb.RequestCohortWatchActionResponse], error) {
	if h.supervision == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("cohort supervision is unavailable"))
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	if h.watchActionAuth == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("watch action authorizer is unavailable"))
	}
	request := proto.Clone(req.Msg).(*domainpb.RequestCohortWatchActionRequest)
	if err := h.watchActionAuth.AuthorizeWatchAction(ctx, bearerToken(req.Header().Get("Authorization")), request); err != nil {
		if errors.Is(err, ErrWatchActionUnauthenticated) {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	response, err := h.supervision.RequestAction(ctx, request)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(response), nil
}

func (h *AgentManagerConnectHandler) ListCohortWatchActions(ctx context.Context, req *connect.Request[domainpb.ListCohortWatchActionsRequest]) (*connect.Response[domainpb.ListCohortWatchActionsResponse], error) {
	if h.supervision == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("cohort supervision is unavailable"))
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	response, err := h.supervision.ListActions(ctx, req.Msg)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(response), nil
}

func (h *AgentManagerConnectHandler) GetSupervisionPolicy(ctx context.Context, req *connect.Request[domainpb.GetSupervisionPolicyRequest]) (*connect.Response[domainpb.SupervisionPolicyRecord], error) {
	if h.supervision == nil || req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("supervision service and request are required"))
	}
	result, err := h.supervision.GetPolicy(ctx, req.Msg)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(result), nil
}

func (h *AgentManagerConnectHandler) CreateSupervisionPolicyCandidate(ctx context.Context, req *connect.Request[domainpb.CreateSupervisionPolicyCandidateRequest]) (*connect.Response[domainpb.SupervisionPolicyRecord], error) {
	if h.supervision == nil || req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("supervision service and request are required"))
	}
	actor, err := h.authorizePolicyOperator(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}
	req.Msg.CreatedBy = actor
	result, err := h.supervision.CreatePolicyCandidate(ctx, req.Msg)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(result), nil
}

func (h *AgentManagerConnectHandler) RecordSupervisionOutcome(ctx context.Context, req *connect.Request[domainpb.RecordSupervisionOutcomeRequest]) (*connect.Response[domainpb.RecordSupervisionOutcomeResponse], error) {
	if h.supervision == nil || req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("supervision service and request are required"))
	}
	if _, err := h.authorizePolicyOperator(ctx, req.Header().Get("Authorization")); err != nil {
		return nil, err
	}
	result, err := h.supervision.RecordPolicyOutcome(ctx, req.Msg)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(result), nil
}

func (h *AgentManagerConnectHandler) EvaluateSupervisionPolicy(ctx context.Context, req *connect.Request[domainpb.EvaluateSupervisionPolicyRequest]) (*connect.Response[domainpb.SupervisionReplayReport], error) {
	if h.supervision == nil || req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("supervision service and request are required"))
	}
	if _, err := h.authorizePolicyOperator(ctx, req.Header().Get("Authorization")); err != nil {
		return nil, err
	}
	result, err := h.supervision.EvaluatePolicy(ctx, req.Msg)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(result), nil
}

func (h *AgentManagerConnectHandler) PromoteSupervisionPolicy(ctx context.Context, req *connect.Request[domainpb.PromoteSupervisionPolicyRequest]) (*connect.Response[domainpb.SupervisionPolicyRecord], error) {
	if h.supervision == nil || req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	actor, err := h.authorizePolicyOperator(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}
	req.Msg.ReviewedBy = actor
	result, err := h.supervision.PromotePolicy(ctx, req.Msg)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(result), nil
}

func (h *AgentManagerConnectHandler) RejectSupervisionPolicy(ctx context.Context, req *connect.Request[domainpb.RejectSupervisionPolicyRequest]) (*connect.Response[domainpb.SupervisionPolicyRecord], error) {
	if h.supervision == nil || req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	actor, err := h.authorizePolicyOperator(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}
	req.Msg.ReviewedBy = actor
	result, err := h.supervision.RejectPolicy(ctx, req.Msg)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(result), nil
}

func (h *AgentManagerConnectHandler) RollbackSupervisionPolicy(ctx context.Context, req *connect.Request[domainpb.RollbackSupervisionPolicyRequest]) (*connect.Response[domainpb.SupervisionPolicyRecord], error) {
	if h.supervision == nil || req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	actor, err := h.authorizePolicyOperator(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}
	req.Msg.ReviewedBy = actor
	result, err := h.supervision.RollbackPolicy(ctx, req.Msg)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(result), nil
}

func (h *AgentManagerConnectHandler) SetSupervisionPolicyDisabled(ctx context.Context, req *connect.Request[domainpb.SetSupervisionPolicyDisabledRequest]) (*connect.Response[domainpb.SupervisionPolicyControl], error) {
	if h.supervision == nil || req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	actor, err := h.authorizePolicyOperator(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}
	req.Msg.Actor = actor
	result, err := h.supervision.SetPolicyDisabled(ctx, req.Msg)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(result), nil
}

func (h *AgentManagerConnectHandler) ListSupervisionOutcomes(ctx context.Context, req *connect.Request[domainpb.ListSupervisionOutcomesRequest]) (*connect.Response[domainpb.ListSupervisionOutcomesResponse], error) {
	if h.supervision == nil || req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	result, err := h.supervision.ListPolicyOutcomes(ctx, req.Msg)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(result), nil
}

func (h *AgentManagerConnectHandler) authorizePolicyOperator(ctx context.Context, token string) (string, error) {
	if h.watchActionAuth == nil {
		return "", connect.NewError(connect.CodeUnavailable, errors.New("supervision policy authorizer is unavailable"))
	}
	request := &domainpb.RequestCohortWatchActionRequest{Authority: domainpb.WatchAuthority_WATCH_AUTHORITY_OPERATOR}
	if err := h.watchActionAuth.AuthorizeWatchAction(ctx, bearerToken(token), request); err != nil {
		if errors.Is(err, ErrWatchActionUnauthenticated) {
			return "", connect.NewError(connect.CodeUnauthenticated, err)
		}
		return "", connect.NewError(connect.CodePermissionDenied, err)
	}
	return request.GetRequestedBy(), nil
}

func watchConnectError(err error) error {
	if errors.Is(err, supervision.ErrExactRetryable) {
		return connect.NewError(connect.CodeUnavailable, err)
	}
	if errors.Is(err, supervision.ErrExactAuthority) {
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	if errors.Is(err, supervision.ErrNotFound) || errors.Is(err, supervision.ErrPolicyNotFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	if errors.Is(err, supervision.ErrConflict) {
		return connect.NewError(connect.CodeAborted, err)
	}
	if errors.Is(err, supervision.ErrPromotionGate) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewError(connect.CodeInvalidArgument, err)
}

func (h *AgentManagerConnectHandler) ListRuns(ctx context.Context, req *connect.Request[api.ListRunsRequest]) (*connect.Response[api.ListRunsResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("request is required"))
	}
	request := proto.Clone(req.Msg).(*api.ListRunsRequest)
	if request.Limit == nil {
		limit := int32(defaultRunListLimit)
		request.Limit = &limit
	}
	if h.h.validator != nil {
		if err := h.h.validator.Validate(request); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, protovalidateToDomainError(err))
		}
	}
	response, err := h.h.listRunsProto(ctx, request, nil, nil, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(response), nil
}

// GetRunReport exposes the same bounded projection used by the CLI and REST
// handlers. Keeping the projection in orchestration prevents the Connect
// surface from growing a second, potentially less-private report shape.
func (h *AgentManagerConnectHandler) GetRunReport(ctx context.Context, req *connect.Request[api.GetRunReportRequest]) (*connect.Response[api.RunReport], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("request is required"))
	}
	runID, err := uuid.Parse(req.Msg.GetRunId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	report, err := h.h.svc.BuildRunReport(ctx, runID)
	if err != nil {
		if domain.GetErrorCode(err) == domain.ErrCodeNotFoundRun {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(runReportToProto(report)), nil
}

// GetRunAccounting exposes the same metered usage as the REST route, so a
// consumer settling a standalone run reads one owner projection.
func (h *AgentManagerConnectHandler) GetRunAccounting(ctx context.Context, req *connect.Request[api.GetRunAccountingRequest]) (*connect.Response[api.RunAccounting], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("request is required"))
	}
	runID, err := uuid.Parse(req.Msg.GetRunId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	accounting, err := h.h.svc.RunAccounting(ctx, runID)
	if err != nil {
		if domain.GetErrorCode(err) == domain.ErrCodeNotFoundRun {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(runAccountingToProto(accounting)), nil
}

// validateMessage applies the descriptor's buf.validate rules the REST handlers
// applied through Handler.validateProto.
func (h *AgentManagerConnectHandler) validateMessage(msg proto.Message) error {
	if h.h == nil || h.h.validator == nil {
		return nil
	}
	if err := h.h.validator.Validate(msg); err != nil {
		return connect.NewError(connect.CodeInvalidArgument, protovalidateToDomainError(err))
	}
	return nil
}

func (h *AgentManagerConnectHandler) ReconcileScenarioDeclarations(ctx context.Context, req *connect.Request[api.ReconcileScenarioDeclarationsRequest]) (*connect.Response[api.ReconcileScenarioDeclarationsResponse], error) {
	return h.reconcileDeclarations(ctx, req.Msg, false)
}

// PlanScenarioDeclarations validates every declaration source without writes.
func (h *AgentManagerConnectHandler) PlanScenarioDeclarations(ctx context.Context, req *connect.Request[api.ReconcileScenarioDeclarationsRequest]) (*connect.Response[api.ReconcileScenarioDeclarationsResponse], error) {
	return h.reconcileDeclarations(ctx, req.Msg, true)
}

func (h *AgentManagerConnectHandler) reconcileDeclarations(ctx context.Context, msg *api.ReconcileScenarioDeclarationsRequest, forceDryRun bool) (*connect.Response[api.ReconcileScenarioDeclarationsResponse], error) {
	if err := h.validateMessage(msg); err != nil {
		return nil, err
	}
	result, err := h.h.profiles.ReconcileScenarioDeclarations(ctx, orchestration.ReconcileScenarioDeclarationsRequest{
		Scenario:     msg.Scenario,
		DryRun:       msg.DryRun || forceDryRun,
		ValidateOnly: msg.ValidateOnly,
	})
	if err != nil {
		return nil, workflowConnectError(err)
	}
	return connect.NewResponse(reconcileScenarioDeclarationsToProto(result)), nil
}

func (h *AgentManagerConnectHandler) rolePolicyState() (*rolepolicy.State, error) {
	if h.h == nil || h.h.rolePolicy == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("role policy state is not configured"))
	}
	return h.h.rolePolicy, nil
}

func (h *AgentManagerConnectHandler) GetRolePolicyStatus(_ context.Context, _ *connect.Request[api.GetRolePolicyStatusRequest]) (*connect.Response[api.GetRolePolicyStatusResponse], error) {
	state, err := h.rolePolicyState()
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&api.GetRolePolicyStatusResponse{Status: protoconv.RolePolicyStatusToProto(state.Status())}), nil
}

func (h *AgentManagerConnectHandler) GetRolePolicyCatalog(_ context.Context, _ *connect.Request[api.GetRolePolicyCatalogRequest]) (*connect.Response[api.GetRolePolicyCatalogResponse], error) {
	if _, err := h.rolePolicyState(); err != nil {
		return nil, err
	}
	return connect.NewResponse(h.h.rolePolicyCatalogResponse()), nil
}

func (h *AgentManagerConnectHandler) ValidateRolePolicyCatalog(_ context.Context, _ *connect.Request[api.ValidateRolePolicyCatalogRequest]) (*connect.Response[api.ValidateRolePolicyCatalogResponse], error) {
	state, err := h.rolePolicyState()
	if err != nil {
		return nil, err
	}
	revision, err := state.Validate()
	response := &api.ValidateRolePolicyCatalogResponse{ActiveDigest: state.Status().ActiveDigest}
	if err != nil {
		response.Diagnostic = rolePolicyInvalidDiagnostic(err)
	} else {
		response.Valid = true
		response.CandidateDigest = revision.Digest()
	}
	return connect.NewResponse(response), nil
}

func (h *AgentManagerConnectHandler) ReloadRolePolicyCatalog(_ context.Context, _ *connect.Request[api.ReloadRolePolicyCatalogRequest]) (*connect.Response[api.ReloadRolePolicyCatalogResponse], error) {
	state, err := h.rolePolicyState()
	if err != nil {
		return nil, err
	}
	_, reloadErr := state.Reload()
	response := &api.ReloadRolePolicyCatalogResponse{Activated: reloadErr == nil, Status: protoconv.RolePolicyStatusToProto(state.Status())}
	if reloadErr != nil {
		response.Diagnostic = rolePolicyInvalidDiagnostic(reloadErr)
	}
	return connect.NewResponse(response), nil
}

// ExplainRolePolicy resolves a profile now or returns a run's persisted
// snapshot. Historical runs never borrow provenance from the current catalog.
func (h *AgentManagerConnectHandler) ExplainRolePolicy(ctx context.Context, req *connect.Request[api.ExplainRolePolicyRequest]) (*connect.Response[api.ExplainRolePolicyResponse], error) {
	if err := h.validateMessage(req.Msg); err != nil {
		return nil, err
	}
	response := &api.ExplainRolePolicyResponse{}
	var snapshot *domain.ExecutionPolicySnapshot
	var err error
	switch {
	case req.Msg.GetProfileId() != "":
		id, parseErr := uuid.Parse(req.Msg.GetProfileId())
		if parseErr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, parseErr)
		}
		response.TargetType, response.TargetId = "profile", id.String()
		snapshot, err = h.h.svc.ExplainProfilePolicy(ctx, id)
	case req.Msg.GetRunId() != "":
		id, parseErr := uuid.Parse(req.Msg.GetRunId())
		if parseErr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, parseErr)
		}
		response.TargetType, response.TargetId = "run", id.String()
		snapshot, err = h.h.svc.ExplainRunPolicy(ctx, id)
		response.HistoricalWithoutSnapshot = err == nil && snapshot == nil
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("exactly one of profile_id or run_id is required"))
	}
	if err != nil {
		return nil, workflowConnectError(err)
	}
	response.Snapshot = protoconv.ExecutionPolicySnapshotToProto(snapshot)
	if snapshot != nil {
		response.Summary = snapshot.Explanation.Summary
	} else if response.HistoricalWithoutSnapshot {
		response.Summary = "historical run has no persisted policy snapshot; current catalog provenance was not fabricated"
	}
	return connect.NewResponse(response), nil
}

func (h *AgentManagerConnectHandler) ProbeRunner(ctx context.Context, req *connect.Request[api.ProbeRunnerRequest]) (*connect.Response[api.ProbeRunnerResponse], error) {
	if err := h.validateMessage(req.Msg); err != nil {
		return nil, err
	}
	runnerType := protoconv.RunnerTypeFromProto(req.Msg.RunnerType)
	if !runnerType.IsValid() {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid runner type"))
	}
	result, err := h.h.svc.ProbeRunner(ctx, runnerType)
	if err != nil {
		return nil, workflowConnectError(err)
	}
	return connect.NewResponse(&api.ProbeRunnerResponse{Result: protoconv.ProbeResultToProto(&protoconv.ProbeResult{
		RunnerType: result.RunnerType, Success: result.Success, Message: result.Message, Response: result.Response, DurationMs: result.DurationMs,
	})}), nil
}

// PurgeData deletes profiles, tasks, or runs matching a regex pattern.
func (h *AgentManagerConnectHandler) PurgeData(ctx context.Context, req *connect.Request[api.PurgeDataRequest]) (*connect.Response[api.PurgeDataResponse], error) {
	if err := h.validateMessage(req.Msg); err != nil {
		return nil, err
	}
	targets := make([]orchestration.PurgeTarget, 0, len(req.Msg.Targets))
	for _, target := range req.Msg.Targets {
		switch target {
		case api.PurgeTarget_PURGE_TARGET_PROFILES:
			targets = append(targets, orchestration.PurgeTargetProfiles)
		case api.PurgeTarget_PURGE_TARGET_TASKS:
			targets = append(targets, orchestration.PurgeTargetTasks)
		case api.PurgeTarget_PURGE_TARGET_RUNS:
			targets = append(targets, orchestration.PurgeTargetRuns)
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid purge target"))
		}
	}
	result, err := h.h.svc.PurgeData(ctx, orchestration.PurgeRequest{Pattern: req.Msg.Pattern, Targets: targets, DryRun: req.Msg.DryRun})
	if err != nil {
		return nil, workflowConnectError(err)
	}
	return connect.NewResponse(&api.PurgeDataResponse{Matched: purgeCountsToProto(result.Matched), Deleted: purgeCountsToProto(result.Deleted), DryRun: result.DryRun}), nil
}

func purgeCountsToProto(counts orchestration.PurgeCounts) *api.PurgeCounts {
	return &api.PurgeCounts{Profiles: int32(counts.Profiles), Tasks: int32(counts.Tasks), Runs: int32(counts.Runs)}
}

func (h *AgentManagerConnectHandler) permissionPolicyReady() error {
	if h.h == nil || h.h.permissionPolicyState == nil || h.h.permissionPolicy == nil {
		return connect.NewError(connect.CodeUnavailable, errors.New("permission policy service is not configured"))
	}
	return nil
}

// GetPermissionPolicyStatus returns activation state and the last persisted
// reconcile evidence. It never triggers a native resource operation.
func (h *AgentManagerConnectHandler) GetPermissionPolicyStatus(ctx context.Context, _ *connect.Request[api.GetPermissionPolicyStatusRequest]) (*connect.Response[api.GetPermissionPolicyStatusResponse], error) {
	if err := h.permissionPolicyReady(); err != nil {
		return nil, err
	}
	last, err := h.h.permissionPolicy.LastReconcile(ctx)
	if err != nil {
		return nil, workflowConnectError(err)
	}
	return connect.NewResponse(&api.GetPermissionPolicyStatusResponse{
		Status:        protoconv.PermissionPolicyStatusToProto(h.h.permissionPolicyState.Status()),
		LastReconcile: protoconv.PermissionPolicyReconcileResultToProto(last),
	}), nil
}

// GetPermissionPolicyCatalog exposes the global portable desired state, not
// native resource configuration or runner-specific syntax.
func (h *AgentManagerConnectHandler) GetPermissionPolicyCatalog(_ context.Context, _ *connect.Request[api.GetPermissionPolicyCatalogRequest]) (*connect.Response[api.GetPermissionPolicyCatalogResponse], error) {
	if err := h.permissionPolicyReady(); err != nil {
		return nil, err
	}
	var catalog *permissionpolicy.Catalog
	if active := h.h.permissionPolicyState.Active(); active != nil {
		catalog = active.Catalog()
	}
	return connect.NewResponse(&api.GetPermissionPolicyCatalogResponse{
		Status:  protoconv.PermissionPolicyStatusToProto(h.h.permissionPolicyState.Status()),
		Catalog: protoconv.PermissionPolicyCatalogToProto(catalog),
	}), nil
}

func permissionPolicyInvalidDiagnostic(err error) *api.PermissionPolicyDiagnostic {
	return protoconv.PermissionPolicyDiagnosticToProto(&permissionpolicy.Diagnostic{Code: permissionpolicy.DiagnosticCodeCatalogInvalid, Message: err.Error(), Cause: err.Error()})
}

func (h *AgentManagerConnectHandler) ValidatePermissionPolicyCatalog(_ context.Context, _ *connect.Request[api.ValidatePermissionPolicyCatalogRequest]) (*connect.Response[api.ValidatePermissionPolicyCatalogResponse], error) {
	if err := h.permissionPolicyReady(); err != nil {
		return nil, err
	}
	state := h.h.permissionPolicyState
	revision, err := state.Validate()
	response := &api.ValidatePermissionPolicyCatalogResponse{ActiveDigest: state.Status().ActiveDigest}
	if err != nil {
		response.Diagnostic = permissionPolicyInvalidDiagnostic(err)
	} else {
		response.Valid = true
		response.CandidateDigest = revision.Digest()
	}
	return connect.NewResponse(response), nil
}

func (h *AgentManagerConnectHandler) ReloadPermissionPolicyCatalog(_ context.Context, _ *connect.Request[api.ReloadPermissionPolicyCatalogRequest]) (*connect.Response[api.ReloadPermissionPolicyCatalogResponse], error) {
	if err := h.permissionPolicyReady(); err != nil {
		return nil, err
	}
	state := h.h.permissionPolicyState
	_, err := state.Reload()
	response := &api.ReloadPermissionPolicyCatalogResponse{Activated: err == nil, Status: protoconv.PermissionPolicyStatusToProto(state.Status())}
	if err != nil {
		response.Diagnostic = permissionPolicyInvalidDiagnostic(err)
		obs.Component("permission-policy").Error("permission_policy_reload_failed", obs.KeyError, err.Error(), obs.KeyPermissionPolicyDigest, response.Status.GetActiveDigest())
	} else {
		obs.Component("permission-policy").Info("permission_policy_reloaded", obs.KeyPermissionPolicyDigest, response.Status.GetActiveDigest())
	}
	return connect.NewResponse(response), nil
}

func (h *AgentManagerConnectHandler) PlanPermissionPolicy(ctx context.Context, _ *connect.Request[api.PlanPermissionPolicyRequest]) (*connect.Response[api.PlanPermissionPolicyResponse], error) {
	if err := h.permissionPolicyReady(); err != nil {
		return nil, err
	}
	plan, err := h.h.permissionPolicy.Plan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	logPermissionPolicyPlan("permission_policy_planned", plan)
	return connect.NewResponse(&api.PlanPermissionPolicyResponse{Plan: protoconv.PermissionPolicyPlanToProto(plan)}), nil
}

func (h *AgentManagerConnectHandler) ReconcilePermissionPolicy(ctx context.Context, req *connect.Request[api.ReconcilePermissionPolicyRequest]) (*connect.Response[api.ReconcilePermissionPolicyResponse], error) {
	if err := h.permissionPolicyReady(); err != nil {
		return nil, err
	}
	if !req.Msg.ExplicitlyAuthorized {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("explicit human authorization is required"))
	}
	obs.Component("permission-policy").Info("permission_policy_reconcile_started", obs.KeyPermissionPolicyDigest, h.h.permissionPolicyState.Status().ActiveDigest)
	result, err := h.h.permissionPolicy.Reconcile(ctx, true)
	if err != nil {
		return nil, workflowConnectError(err)
	}
	logPermissionPolicyReconcile(result)
	return connect.NewResponse(&api.ReconcilePermissionPolicyResponse{Result: protoconv.PermissionPolicyReconcileResultToProto(&result)}), nil
}

func (h *AgentManagerConnectHandler) DoctorPermissionPolicy(ctx context.Context, _ *connect.Request[api.DoctorPermissionPolicyRequest]) (*connect.Response[api.DoctorPermissionPolicyResponse], error) {
	if err := h.permissionPolicyReady(); err != nil {
		return nil, err
	}
	plan, err := h.h.permissionPolicy.Plan(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	logPermissionPolicyPlan("permission_policy_doctor_planned", plan)
	status := h.h.permissionPolicyState.Status()
	healthy := status.Ready && plan.HardEnforcementSatisfied && plan.AssessmentComplete()
	summary := "permission policy is ready; no required hard-enforcement gap was detected"
	if !status.Ready {
		summary = "permission policy catalog is not ready"
	} else if !plan.HardEnforcementSatisfied {
		summary = "one or more required permission rules lack native or hook-backed enforcement"
	} else if !plan.AssessmentComplete() {
		summary = "permission policy assessment is incomplete; inspect failed or unavailable resource probes"
	}
	return connect.NewResponse(&api.DoctorPermissionPolicyResponse{
		Status: protoconv.PermissionPolicyStatusToProto(status), Plan: protoconv.PermissionPolicyPlanToProto(plan), Healthy: healthy, Summary: summary,
	}), nil
}

// GetRun returns one run with its observed receipts attached to the result.
func (h *AgentManagerConnectHandler) GetRun(ctx context.Context, req *connect.Request[api.GetRunRequest]) (*connect.Response[api.GetRunResponse], error) {
	if err := h.validateMessage(req.Msg); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.Msg.RunId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	run, err := h.h.svc.GetRun(ctx, id)
	if err != nil {
		return nil, workflowConnectError(err)
	}
	pbRun := protoconv.RunToProto(run)
	h.h.attachObservedReceipts(ctx, id.String(), pbRun.Result)
	return connect.NewResponse(&api.GetRunResponse{Run: pbRun}), nil
}

// ReconcileScenarioProfiles resolves all profile sources a scenario declares.
func (h *AgentManagerConnectHandler) ReconcileScenarioProfiles(ctx context.Context, req *connect.Request[api.ReconcileScenarioProfilesRequest]) (*connect.Response[api.ReconcileScenarioProfilesResponse], error) {
	if err := h.validateMessage(req.Msg); err != nil {
		return nil, err
	}
	result, err := h.h.profiles.ReconcileScenarioProfiles(ctx, orchestration.ReconcileScenarioProfilesRequest{Scenario: req.Msg.Scenario, DryRun: req.Msg.DryRun})
	if err != nil {
		return nil, workflowConnectError(err)
	}
	return connect.NewResponse(reconcileScenarioProfilesToProto(result)), nil
}

func (h *AgentManagerConnectHandler) GetProfile(ctx context.Context, req *connect.Request[api.GetProfileRequest]) (*connect.Response[api.GetProfileResponse], error) {
	if err := h.validateMessage(req.Msg); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.Msg.ProfileId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	profile, err := h.h.profiles.GetProfile(ctx, id)
	if err != nil {
		return nil, workflowConnectError(err)
	}
	return connect.NewResponse(&api.GetProfileResponse{Profile: protoconv.AgentProfileToProto(profile)}), nil
}

func (h *AgentManagerConnectHandler) ListProfiles(ctx context.Context, req *connect.Request[api.ListProfilesRequest]) (*connect.Response[api.ListProfilesResponse], error) {
	if err := h.validateMessage(req.Msg); err != nil {
		return nil, err
	}
	profiles, err := h.h.profiles.ListProfiles(ctx, orchestration.ListOptions{Limit: int(req.Msg.GetLimit()), Offset: int(req.Msg.GetOffset())})
	if err != nil {
		return nil, workflowConnectError(err)
	}
	return connect.NewResponse(&api.ListProfilesResponse{Profiles: protoconv.AgentProfilesToProto(profiles), Total: int32(len(profiles))}), nil
}

// CreateTask is the task leg of run creation. It shares the REST handler's
// caller authorization and storage path (authorizeTaskCreate, createTask).
func (h *AgentManagerConnectHandler) CreateTask(ctx context.Context, req *connect.Request[api.CreateTaskRequest]) (*connect.Response[api.CreateTaskResponse], error) {
	trustedCreator, err := h.h.authorizeTaskCreate(ctx, req.Header().Get(cliutil.HeaderAgentIdentityToken))
	if err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	if err := h.validateMessage(req.Msg); err != nil {
		return nil, err
	}
	response, err := h.h.createTask(ctx, trustedCreator, req.Msg)
	if err != nil {
		return nil, workflowConnectError(err)
	}
	return connect.NewResponse(response), nil
}

// CreateRun runs the REST create-run handler in process. Its caller-proof,
// delegated-child and run-initiated-lifecycle guards write HTTP responses and
// read request headers, so the Connect procedure carries the caller's headers
// and context into that one implementation instead of restating the guards.
func (h *AgentManagerConnectHandler) CreateRun(ctx context.Context, req *connect.Request[api.CreateRunRequest]) (*connect.Response[api.CreateRunResponse], error) {
	response := &api.CreateRunResponse{}
	if err := callRESTHandler(ctx, req.Header(), req.Peer().Addr, h.h.CreateRun, req.Msg, response); err != nil {
		return nil, err
	}
	return connect.NewResponse(response), nil
}

// capturedResponse records what a REST handler wrote.
type capturedResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (c *capturedResponse) Header() http.Header         { return c.header }
func (c *capturedResponse) WriteHeader(status int)      { c.status = status }
func (c *capturedResponse) Write(p []byte) (int, error) { return c.body.Write(p) }

// callRESTHandler invokes a REST handler with a proto request and decodes its
// proto response, mapping a refusal to the Connect code of its HTTP status.
func callRESTHandler(ctx context.Context, header http.Header, peer string, handler http.HandlerFunc, request, response proto.Message) error {
	payload, err := protoconv.MarshalJSON(request)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, "/", bytes.NewReader(payload))
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	for name, values := range header {
		if !strings.HasPrefix(name, "Connect-") && name != "Content-Length" && name != "Content-Type" {
			httpRequest.Header[name] = append([]string(nil), values...)
		}
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.RemoteAddr = peer
	captured := &capturedResponse{header: http.Header{}, status: http.StatusOK}
	handler(captured, httpRequest)
	if captured.status >= http.StatusBadRequest {
		var refusal struct {
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		_ = json.Unmarshal(captured.body.Bytes(), &refusal)
		message := cmp.Or(refusal.Message, refusal.Error, http.StatusText(captured.status))
		return connect.NewError(connectCodeForStatus(captured.status), errors.New(message))
	}
	if err := protoconv.UnmarshalJSON(captured.body.Bytes(), response); err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	return nil
}

func connectCodeForStatus(status int) connect.Code {
	switch status {
	case http.StatusBadRequest:
		return connect.CodeInvalidArgument
	case http.StatusUnauthorized:
		return connect.CodeUnauthenticated
	case http.StatusForbidden:
		return connect.CodePermissionDenied
	case http.StatusNotFound:
		return connect.CodeNotFound
	case http.StatusConflict:
		return connect.CodeFailedPrecondition
	case http.StatusTooManyRequests:
		return connect.CodeResourceExhausted
	case http.StatusServiceUnavailable:
		return connect.CodeUnavailable
	}
	return connect.CodeInternal
}
