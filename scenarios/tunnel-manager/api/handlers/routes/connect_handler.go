package routes

import (
	"context"
	"log"

	"tunnel-manager/internal/authz"
	"tunnel-manager/internal/routes"

	"connectrpc.com/connect"

	routesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/routes"
)

// Deps wires the seams the Connect routes handler needs.
type Deps struct {
	Service    routes.Service
	Logger     *log.Logger
	Authorizer authz.Authorizer
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	if d.Authorizer == nil {
		d.Authorizer = authz.AllowLocalOperator()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ListRoutes(ctx context.Context, req *connect.Request[routesv1.ListRoutesRequest]) (*connect.Response[routesv1.ListRoutesResponse], error) {
	results, err := h.deps.Service.List(ctx, tierFromProto(req.Msg.Tier))
	if err != nil {
		h.deps.Logger.Printf("routes.ListRoutes: %v", err)
		return nil, routes.ToConnectError(err)
	}
	resp := &routesv1.ListRoutesResponse{Routes: make([]*routesv1.Route, 0, len(results))}
	for _, r := range results {
		resp.Routes = append(resp.Routes, domainToProto(r))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetRoute(ctx context.Context, req *connect.Request[routesv1.GetRouteRequest]) (*connect.Response[routesv1.GetRouteResponse], error) {
	got, err := h.deps.Service.Get(ctx, req.Msg.Id)
	if err != nil {
		connectErr := routes.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("routes.GetRoute(%q): %v", req.Msg.Id, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&routesv1.GetRouteResponse{Route: domainToProto(got)}), nil
}

func (h *connectHandler) CreateRoute(ctx context.Context, req *connect.Request[routesv1.CreateRouteRequest]) (*connect.Response[routesv1.CreateRouteResponse], error) {
	if err := h.deps.Authorizer.Authorize(ctx, authz.OperationRoutesCreate, req.Header()); err != nil {
		return nil, authz.ToConnectError(err)
	}
	in := routes.CreateInput{
		Subdomain:  req.Msg.Subdomain,
		Scenario:   req.Msg.Scenario,
		Domain:     req.Msg.Domain,
		LocalPort:  int(req.Msg.LocalPort),
		Tier:       tierFromProto(req.Msg.Tier),
		LeaseID:    req.Msg.LeaseId,
		HealthPath: req.Msg.HealthPath,
	}
	if req.Msg.Enabled != nil {
		in.Enabled = req.Msg.Enabled
	}
	created, err := h.deps.Service.Create(ctx, in)
	if err != nil {
		connectErr := routes.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("routes.CreateRoute: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&routesv1.CreateRouteResponse{Route: domainToProto(created)}), nil
}

func (h *connectHandler) UpdateRoute(ctx context.Context, req *connect.Request[routesv1.UpdateRouteRequest]) (*connect.Response[routesv1.UpdateRouteResponse], error) {
	if err := h.deps.Authorizer.Authorize(ctx, authz.OperationRoutesUpdate, req.Header()); err != nil {
		return nil, authz.ToConnectError(err)
	}
	in := routes.UpdateInput{
		Subdomain:  req.Msg.Subdomain,
		Scenario:   req.Msg.Scenario,
		Domain:     req.Msg.Domain,
		LocalPort:  int(req.Msg.LocalPort),
		Tier:       tierFromProto(req.Msg.Tier),
		HealthPath: req.Msg.HealthPath,
	}
	if req.Msg.Enabled != nil {
		in.Enabled = req.Msg.Enabled
	}
	updated, err := h.deps.Service.Update(ctx, req.Msg.Id, in)
	if err != nil {
		connectErr := routes.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("routes.UpdateRoute(%q): %v", req.Msg.Id, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&routesv1.UpdateRouteResponse{Route: domainToProto(updated)}), nil
}

func (h *connectHandler) DeleteRoute(ctx context.Context, req *connect.Request[routesv1.DeleteRouteRequest]) (*connect.Response[routesv1.DeleteRouteResponse], error) {
	if err := h.deps.Authorizer.Authorize(ctx, authz.OperationRoutesDelete, req.Header()); err != nil {
		return nil, authz.ToConnectError(err)
	}
	deleted, err := h.deps.Service.Delete(ctx, req.Msg.Id)
	if err != nil {
		h.deps.Logger.Printf("routes.DeleteRoute(%q): %v", req.Msg.Id, err)
		return nil, routes.ToConnectError(err)
	}
	return connect.NewResponse(&routesv1.DeleteRouteResponse{Deleted: deleted}), nil
}
