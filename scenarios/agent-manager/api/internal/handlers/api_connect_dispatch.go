package handlers

import (
	"context"
	"errors"

	"agent-manager/internal/domain"
	"connectrpc.com/connect"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func (h *AgentManagerConnectHandler) IssueSupervisorDispatch(ctx context.Context, req *connect.Request[api.IssueSupervisorDispatchRequest]) (*connect.Response[pb.EffortEnrollment], error) {
	if req == nil || req.Msg == nil || req.Header().Get("X-Agent-Identity-Token") != "" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("explicit operator issuance required"))
	}
	s, err := h.effortService()
	if err != nil {
		return nil, err
	}
	actor, err := h.effortActor(ctx, bearerToken(req.Header().Get("Authorization")), pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR)
	if err != nil {
		return nil, err
	}
	e, err := s.IssueDispatch(ctx, req.Msg, actor)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(e), nil
}

func (h *AgentManagerConnectHandler) RevokeSupervisorDispatch(ctx context.Context, req *connect.Request[api.RevokeSupervisorDispatchRequest]) (*connect.Response[pb.EffortEnrollment], error) {
	if req == nil || req.Msg == nil || req.Header().Get("X-Agent-Identity-Token") != "" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("explicit operator revocation required"))
	}
	s, err := h.effortService()
	if err != nil {
		return nil, err
	}
	actor, err := h.effortActor(ctx, bearerToken(req.Header().Get("Authorization")), pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR)
	if err != nil {
		return nil, err
	}
	e, err := s.RevokeDispatch(ctx, req.Msg, actor)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(e), nil
}

func (h *AgentManagerConnectHandler) CreateSupervisorRun(ctx context.Context, req *connect.Request[api.CreateSupervisorRunRequest]) (*connect.Response[api.CreateRunResponse], error) {
	if req == nil || req.Msg == nil || req.Header().Get("X-Agent-Identity-Token") != "" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("purpose-bound dispatcher credential required"))
	}
	if h.h == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("supervisor dispatch unavailable"))
	}
	owner, ok := h.h.svc.RunService.(interface {
		CreateSupervisorRun(context.Context, *api.CreateSupervisorRunRequest, string) (*domain.Run, error)
	})
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("supervisor dispatch unavailable"))
	}
	run, err := owner.CreateSupervisorRun(ctx, req.Msg, bearerToken(req.Header().Get("Authorization")))
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(h.h.newCreateRunResponse(run)), nil
}
