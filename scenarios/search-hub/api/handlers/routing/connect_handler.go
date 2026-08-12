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
	Status(ctx context.Context) (*routingv1.StatusResponse, error)
	RepromoteProvider(ctx context.Context, providerID string) error
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
		Repromote(context.Context, *connect.Request[routingv1.RepromoteRequest]) (*connect.Response[routingv1.RepromoteResponse], error)
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

func (h *connectHandler) Repromote(ctx context.Context, req *connect.Request[routingv1.RepromoteRequest]) (*connect.Response[routingv1.RepromoteResponse], error) {
	if err := h.deps.Router.RepromoteProvider(ctx, req.Msg.GetProviderId()); err != nil {
		connectErr := toConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("routing.Repromote(%q): %v", req.Msg.GetProviderId(), err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&routingv1.RepromoteResponse{
		ProviderId: req.Msg.GetProviderId(),
		Reset_:     true,
		Message:    "graded-empty demotion evidence cleared; automatic routing may probe the provider",
	}), nil
}

// Status reports federation health (Phase 7): per-provider reachability plus
// classifier/reranker availability. It delegates to the router, which never
// fails on an individual provider — a registry read failure is the only error
// path (an opaque Internal so store internals never leak).
func (h *connectHandler) Status(ctx context.Context, _ *connect.Request[routingv1.StatusRequest]) (*connect.Response[routingv1.StatusResponse], error) {
	resp, err := h.deps.Router.Status(ctx)
	if err != nil {
		h.deps.Logger.Printf("routing.Status: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}
	return connect.NewResponse(resp), nil
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
