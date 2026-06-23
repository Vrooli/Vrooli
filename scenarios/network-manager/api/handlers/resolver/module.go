package resolver

import (
	"context"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"network-manager/internal/module"

	resolverv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/resolver"
	resolverconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/resolver/resolver_v1connect"
)

type handler struct{}

func Module() module.Module {
	path, h := resolverconnect.NewResolverServiceHandler(&handler{})
	return module.Module{Name: "resolver", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
	}, Endpoints: Endpoints}
}

func Schema() string { return "" }

func (h *handler) GetResolverStatus(context.Context, *connect.Request[resolverv1.GetResolverStatusRequest]) (*connect.Response[resolverv1.GetResolverStatusResponse], error) {
	return connect.NewResponse(&resolverv1.GetResolverStatusResponse{Status: status()}), nil
}

func (h *handler) ConfigureAdGuardHome(context.Context, *connect.Request[resolverv1.ConfigureAdGuardHomeRequest]) (*connect.Response[resolverv1.ConfigureAdGuardHomeResponse], error) {
	return connect.NewResponse(&resolverv1.ConfigureAdGuardHomeResponse{Status: status(), NextSteps: []string{"Implement AdGuard Home adapter/resource before applying resolver config."}}), nil
}

func (h *handler) UpdateUpstreams(context.Context, *connect.Request[resolverv1.UpdateUpstreamsRequest]) (*connect.Response[resolverv1.UpdateUpstreamsResponse], error) {
	return connect.NewResponse(&resolverv1.UpdateUpstreamsResponse{Status: status(), Changes: []string{"Preview only; no upstreams changed."}}), nil
}

func (h *handler) CheckResolverHealth(context.Context, *connect.Request[resolverv1.CheckResolverHealthRequest]) (*connect.Response[resolverv1.CheckResolverHealthResponse], error) {
	return connect.NewResponse(&resolverv1.CheckResolverHealthResponse{Status: status(), Checks: []string{"AdGuard Home adapter not configured yet."}}), nil
}

func status() *resolverv1.ResolverStatus {
	return &resolverv1.ResolverStatus{
		Backend:   "adguard-home",
		Status:    "not_configured",
		Upstreams: []string{},
		Warnings:  []string{"Managed resolver adapter is scaffolded but not connected."},
	}
}

var Endpoints = []module.EndpointDescriptor{
	connectEndpoint("resolver_status", resolverconnect.ResolverServiceGetResolverStatusProcedure, "Get resolver status"),
	connectEndpoint("resolver_configure_adguard", resolverconnect.ResolverServiceConfigureAdGuardHomeProcedure, "Configure AdGuard Home"),
	connectEndpoint("resolver_update_upstreams", resolverconnect.ResolverServiceUpdateUpstreamsProcedure, "Update resolver upstreams"),
	connectEndpoint("resolver_health", resolverconnect.ResolverServiceCheckResolverHealthProcedure, "Check resolver health"),
}

func connectEndpoint(id, path, summary string) module.EndpointDescriptor {
	return module.EndpointDescriptor{ID: id, Path: path, Method: "POST", Summary: summary, Category: "resolver", Request: &module.Schema{Type: "object", Properties: map[string]string{}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"status": "proto response"}}}
}
