package handlers

import (
	"context"
	"errors"
	"fmt"

	"agent-manager/internal/domain"
	"agent-manager/internal/supervision"

	"connectrpc.com/connect"
	"github.com/google/uuid"
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
	response, err := h.h.listRunsProto(ctx, request, nil, nil)
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
