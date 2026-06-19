package recovery

import (
	"context"
	"log"

	"tunnel-manager/internal/recovery"

	"connectrpc.com/connect"

	recoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/recovery"
)

// Deps wires the seams the Connect recovery handler needs.
type Deps struct {
	Service recovery.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetState(ctx context.Context, _ *connect.Request[recoveryv1.GetStateRequest]) (*connect.Response[recoveryv1.GetStateResponse], error) {
	state, err := h.deps.Service.GetState(ctx)
	if err != nil {
		h.deps.Logger.Printf("recovery.GetState: %v", err)
		return nil, recovery.ToConnectError(err)
	}
	return connect.NewResponse(&recoveryv1.GetStateResponse{State: stateToProto(state)}), nil
}

func (h *connectHandler) ListEvents(ctx context.Context, req *connect.Request[recoveryv1.ListEventsRequest]) (*connect.Response[recoveryv1.ListEventsResponse], error) {
	events, err := h.deps.Service.ListEvents(ctx, int(req.Msg.Limit))
	if err != nil {
		connectErr := recovery.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("recovery.ListEvents: %v", err)
		}
		return nil, connectErr
	}
	resp := &recoveryv1.ListEventsResponse{Events: make([]*recoveryv1.RecoveryEvent, 0, len(events))}
	for _, e := range events {
		resp.Events = append(resp.Events, eventToProto(e))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) Recover(ctx context.Context, req *connect.Request[recoveryv1.RecoverRequest]) (*connect.Response[recoveryv1.RecoverResponse], error) {
	outcome, event, err := h.deps.Service.Recover(ctx, req.Msg.Force)
	if err != nil {
		h.deps.Logger.Printf("recovery.Recover(force=%v): %v", req.Msg.Force, err)
		return nil, recovery.ToConnectError(err)
	}
	return connect.NewResponse(&recoveryv1.RecoverResponse{
		Outcome: outcomeToProto(outcome),
		Event:   eventToProto(event),
	}), nil
}
