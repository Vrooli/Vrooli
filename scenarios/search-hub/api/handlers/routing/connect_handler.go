package routing

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"

	internalrouting "search-hub/internal/routing"
)

// Querier is the router seam the Connect handler depends on. Production wires
// *internalrouting.Router; handler tests substitute a fake so the transport
// edge is exercised without real provider fan-out.
type Querier interface {
	Query(ctx context.Context, req *routingv1.QueryRequest) (*routingv1.QueryResponse, error)
}

// Deps wires the seams the routing Connect handler needs.
type Deps struct {
	Router Querier
	Logger *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the RoutingService handler. Logger defaults to
// log.Default() when nil.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// Compile-time guarantee the handler satisfies the generated interface. A new
// RPC added to routing.proto that the handler hasn't implemented fails here.
var _ = func() any {
	type routingServiceHandler interface {
		Query(context.Context, *connect.Request[routingv1.QueryRequest]) (*connect.Response[routingv1.QueryResponse], error)
		Status(context.Context, *connect.Request[routingv1.StatusRequest]) (*connect.Response[routingv1.StatusResponse], error)
	}
	var _ routingServiceHandler = (*connectHandler)(nil)
	return nil
}()

func (h *connectHandler) Query(ctx context.Context, req *connect.Request[routingv1.QueryRequest]) (*connect.Response[routingv1.QueryResponse], error) {
	resp, err := h.deps.Router.Query(ctx, req.Msg)
	if err != nil {
		connectErr := toConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("routing.Query(%q): %v", req.Msg.GetQuery(), err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(resp), nil
}

// Status is reserved for Phase 7 (federation status + per-provider health +
// classifier/reranker availability). It is intentionally Unimplemented until
// the metrics domain lands; the CLI manifest keeps it in `omitted[]` so no
// command binds to it yet.
func (h *connectHandler) Status(context.Context, *connect.Request[routingv1.StatusRequest]) (*connect.Response[routingv1.StatusResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented,
		errors.New("RoutingService.Status lands in Phase 7 (federation status + metrics)"))
}

// toConnectError translates routing-domain sentinels into Connect codes at the
// transport edge (the domain layer never imports connect). A caller mistake
// (ErrInvalidQuery) becomes InvalidArgument with its message; everything else
// is an opaque Internal so registry/transport internals never leak.
func toConnectError(err error) error {
	if err == nil {
		return nil
	}
	var invalid internalrouting.ErrInvalidQuery
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, invalid)
	}
	return connect.NewError(connect.CodeInternal, errors.New("internal error"))
}
