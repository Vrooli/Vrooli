package handlers

import (
	"context"
	"errors"
	"fmt"

	"agent-manager/internal/domain"
	"connectrpc.com/connect"
	"github.com/google/uuid"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

// GetRunEvents serves the typed Connect contract through the same bounded
// event projection used by the REST compatibility route.
func (h *AgentManagerConnectHandler) GetRunEvents(ctx context.Context, req *connect.Request[apipb.GetRunEventsRequest]) (*connect.Response[apipb.GetRunEventsResponse], error) {
	if h == nil || h.h == nil || h.h.svc.EventService == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("run event service unavailable"))
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request is required"))
	}
	runID, err := uuid.Parse(req.Msg.GetRunId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid run ID: %w", err))
	}
	if h.h.validator != nil {
		if err := h.h.validator.Validate(req.Msg); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, protovalidateToDomainError(err))
		}
	}
	response, err := h.h.getRunEventsPage(ctx, runID, req.Msg)
	if err != nil {
		return nil, runEventsConnectError(err)
	}
	return connect.NewResponse(response), nil
}

func runEventsConnectError(err error) error {
	code := domain.GetErrorCode(err)
	switch code.Category() {
	case "NOT":
		return connect.NewError(connect.CodeNotFound, err)
	case "VALIDATION":
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
