package events

import (
	"context"
	"log"

	"connectrpc.com/connect"

	eventsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/events"
)

// Deps wires the seams the Connect events handler needs.
type Deps struct {
	Service Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect-RPC handler implementing
// EventsServiceHandler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

const (
	defaultLimit = 50
	maxLimit     = 1000
)

func (h *connectHandler) List(ctx context.Context, req *connect.Request[eventsv1.ListRequest]) (*connect.Response[eventsv1.ListResponse], error) {
	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	events := h.deps.Service.Recent(ctx, limit)
	total := h.deps.Service.Count(ctx)

	return connect.NewResponse(&eventsv1.ListResponse{
		Events: eventsToProto(events),
		Total:  int32(total),
	}), nil
}

func eventsToProto(in []Event) []*eventsv1.Event {
	out := make([]*eventsv1.Event, len(in))
	for i, e := range in {
		out[i] = &eventsv1.Event{
			Type:      e.Type,
			SessionId: e.SessionID,
			Timestamp: e.Timestamp,
			Details:   e.Details,
		}
	}
	return out
}
