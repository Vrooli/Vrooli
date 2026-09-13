package handlers

import (
	"agent-manager/internal/supervision"
	"connectrpc.com/connect"
	"context"
	"errors"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"net/http"
)

type effortActionAuthorizer interface {
	AuthorizeEffortAction(context.Context, string, pb.WatchAuthority) (supervision.EffortActor, error)
}

func effortToken(header http.Header, authority pb.WatchAuthority) string {
	if authority == pb.WatchAuthority_WATCH_AUTHORITY_FAMILY_PARENT {
		if token := header.Get("X-Agent-Identity-Token"); token != "" {
			return token
		}
	}
	return bearerToken(header.Get("Authorization"))
}

func (h *AgentManagerConnectHandler) effortService() (*supervision.EffortService, error) {
	if h.supervision == nil || h.supervision.Efforts == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("effort supervision unavailable"))
	}
	return h.supervision.Efforts, nil
}

func (h *AgentManagerConnectHandler) RecordEffortAssessment(ctx context.Context, req *connect.Request[pb.RecordEffortAssessmentRequest]) (*connect.Response[pb.EffortAssessment], error) {
	s, err := h.effortService()
	if err != nil {
		return nil, err
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request required"))
	}
	actor, err := h.effortActor(ctx, effortToken(req.Header(), req.Msg.Authority), req.Msg.Authority)
	if err != nil {
		return nil, err
	}
	out, err := s.RecordAssessment(ctx, req.Msg, actor)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(out), nil
}
func (h *AgentManagerConnectHandler) effortActor(ctx context.Context, token string, authority pb.WatchAuthority) (supervision.EffortActor, error) {
	if h.watchActionAuth == nil {
		return supervision.EffortActor{}, connect.NewError(connect.CodeUnavailable, errors.New("owner authorizer unavailable"))
	}
	if auth, ok := h.watchActionAuth.(effortActionAuthorizer); ok {
		actor, err := auth.AuthorizeEffortAction(ctx, token, authority)
		if err == nil {
			return actor, nil
		}
		code := connect.CodePermissionDenied
		if errors.Is(err, ErrWatchActionUnauthenticated) {
			code = connect.CodeUnauthenticated
		}
		return supervision.EffortActor{}, connect.NewError(code, err)
	}
	r := &pb.RequestCohortWatchActionRequest{Authority: authority}
	if err := h.watchActionAuth.AuthorizeWatchAction(ctx, token, r); err != nil {
		code := connect.CodePermissionDenied
		if errors.Is(err, ErrWatchActionUnauthenticated) {
			code = connect.CodeUnauthenticated
		}
		return supervision.EffortActor{}, connect.NewError(code, err)
	}
	return supervision.EffortActor{ID: r.RequestedBy, Operator: authority == pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR}, nil
}

func (h *AgentManagerConnectHandler) ListEfforts(ctx context.Context, req *connect.Request[pb.ListEffortsRequest]) (*connect.Response[pb.ListEffortsResponse], error) {
	s, err := h.effortService()
	if err != nil {
		return nil, err
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request required"))
	}
	out, err := s.List(ctx, req.Msg)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(out), nil
}

func (h *AgentManagerConnectHandler) GetEffortBoard(ctx context.Context, req *connect.Request[pb.GetEffortBoardRequest]) (*connect.Response[pb.EffortBoard], error) {
	s, err := h.effortService()
	if err != nil {
		return nil, err
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request required"))
	}
	out, err := s.Board(ctx, req.Msg)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(out), nil
}

func (h *AgentManagerConnectHandler) EnrollEffort(ctx context.Context, req *connect.Request[pb.EnrollEffortRequest]) (*connect.Response[pb.EffortEnrollment], error) {
	s, err := h.effortService()
	if err != nil {
		return nil, err
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request required"))
	}
	authority := pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR
	actor, err := h.effortActor(ctx, effortToken(req.Header(), authority), authority)
	if err != nil {
		return nil, err
	}
	out, err := s.Enroll(ctx, req.Msg, actor)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(out), nil
}

func (h *AgentManagerConnectHandler) WithdrawEffort(ctx context.Context, req *connect.Request[pb.WithdrawEffortRequest]) (*connect.Response[pb.EffortEnrollment], error) {
	s, err := h.effortService()
	if err != nil {
		return nil, err
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request required"))
	}
	authority := pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR
	actor, err := h.effortActor(ctx, effortToken(req.Header(), authority), authority)
	if err != nil {
		return nil, err
	}
	out, err := s.Withdraw(ctx, req.Msg, actor)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(out), nil
}

func (h *AgentManagerConnectHandler) RequestEffortDirective(ctx context.Context, req *connect.Request[pb.RequestEffortDirectiveRequest]) (*connect.Response[pb.EffortDirective], error) {
	s, err := h.effortService()
	if err != nil {
		return nil, err
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request required"))
	}
	authority := pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR
	authority = req.Msg.GetAuthority()
	actor, err := h.effortActor(ctx, effortToken(req.Header(), authority), authority)
	if err != nil {
		return nil, err
	}
	out, err := s.RequestDirective(ctx, req.Msg, actor)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(out), nil
}

func (h *AgentManagerConnectHandler) ListEffortDirectives(ctx context.Context, req *connect.Request[pb.ListEffortDirectivesRequest]) (*connect.Response[pb.ListEffortDirectivesResponse], error) {
	s, err := h.effortService()
	if err != nil {
		return nil, err
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request required"))
	}
	out, err := s.ListDirectives(ctx, req.Msg)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(out), nil
}

func (h *AgentManagerConnectHandler) UpdateEffortDirective(ctx context.Context, req *connect.Request[pb.UpdateEffortDirectiveRequest]) (*connect.Response[pb.EffortDirective], error) {
	s, err := h.effortService()
	if err != nil {
		return nil, err
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request required"))
	}
	authority := pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR
	authority = req.Msg.GetAuthority()
	actor, err := h.effortActor(ctx, effortToken(req.Header(), authority), authority)
	if err != nil {
		return nil, err
	}
	out, err := s.UpdateDirective(ctx, req.Msg, actor)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(out), nil
}

func (h *AgentManagerConnectHandler) ReconcileEffortDiscovery(ctx context.Context, req *connect.Request[pb.ReconcileEffortDiscoveryRequest]) (*connect.Response[pb.EffortDiscovery], error) {
	s, err := h.effortService()
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request required"))
	}
	// Explicit scan is authenticated; ordinary board reads stay free of enrollment effects.
	if _, err = h.effortActor(ctx, bearerToken(req.Header().Get("Authorization")), pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR); err != nil {
		return nil, err
	}
	out, err := s.ReconcileDiscovery(ctx)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(out), nil
}
