package handlers

import (
	"context"
	"errors"

	"agent-manager/internal/supervision"
	"connectrpc.com/connect"

	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// ListEffortSignals is the check-in dispatcher's read. Like the board, it is
// an unauthenticated read of owner facts.
func (h *AgentManagerConnectHandler) ListEffortSignals(ctx context.Context, req *connect.Request[pb.ListEffortSignalsRequest]) (*connect.Response[pb.ListEffortSignalsResponse], error) {
	s, err := h.effortService()
	if err != nil {
		return nil, err
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request required"))
	}
	out, err := s.ListSignals(ctx, req.Msg)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(out), nil
}

// RecordEffortSignal accepts only step_back and feedback. A named run must carry the
// effort reference; the signal can only wake a check-in, so it needs no
// further authority.
func (h *AgentManagerConnectHandler) RecordEffortSignal(ctx context.Context, req *connect.Request[pb.RecordEffortSignalRequest]) (*connect.Response[pb.EffortSignal], error) {
	s, err := h.effortService()
	if err != nil {
		return nil, err
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request required"))
	}
	out, err := s.RecordCallerSignal(ctx, req.Msg)
	if errors.Is(err, supervision.ErrSignalKindNotAccepted) {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(out), nil
}
