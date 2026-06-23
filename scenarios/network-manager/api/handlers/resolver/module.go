package resolver

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"network-manager/internal/module"
	domainresolver "network-manager/internal/resolver"

	resolverv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/resolver"
	resolverconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/resolver/resolver_v1connect"
)

type handler struct {
	service *domainresolver.Service
}

func Module(db domainresolver.SQLExecutor) module.Module {
	service := domainresolver.NewService(domainresolver.Config{
		Repo:   domainresolver.NewSQLiteRepository(db),
		Client: domainresolver.ConservativeAdGuardClient{},
	})
	path, h := resolverconnect.NewResolverServiceHandler(&handler{service: service})
	return module.Module{Name: "resolver", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
	}, Endpoints: Endpoints}
}

func Schema() string { return domainresolver.Schema() }

func (h *handler) GetResolverStatus(ctx context.Context, _ *connect.Request[resolverv1.GetResolverStatusRequest]) (*connect.Response[resolverv1.GetResolverStatusResponse], error) {
	status, err := h.service.Status(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&resolverv1.GetResolverStatusResponse{Status: toProtoStatus(status)}), nil
}

func (h *handler) ConfigureAdGuardHome(ctx context.Context, req *connect.Request[resolverv1.ConfigureAdGuardHomeRequest]) (*connect.Response[resolverv1.ConfigureAdGuardHomeResponse], error) {
	status, steps, err := h.service.ConfigureAdGuardHome(ctx, req.Msg.GetBaseUrl(), req.Msg.GetUsername(), req.Msg.GetTokenRef(), req.Msg.GetDryRun())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&resolverv1.ConfigureAdGuardHomeResponse{Status: toProtoStatus(status), NextSteps: steps}), nil
}

func (h *handler) UpdateUpstreams(ctx context.Context, req *connect.Request[resolverv1.UpdateUpstreamsRequest]) (*connect.Response[resolverv1.UpdateUpstreamsResponse], error) {
	status, changes, err := h.service.UpdateUpstreams(ctx, req.Msg.GetUpstreams(), req.Msg.GetDryRun())
	if err != nil {
		code := connect.CodeInvalidArgument
		if errors.Is(err, domainresolver.ErrClientUnsupported) {
			code = connect.CodeFailedPrecondition
		}
		return nil, connect.NewError(code, err)
	}
	return connect.NewResponse(&resolverv1.UpdateUpstreamsResponse{Status: toProtoStatus(status), Changes: changes}), nil
}

func (h *handler) CheckResolverHealth(ctx context.Context, _ *connect.Request[resolverv1.CheckResolverHealthRequest]) (*connect.Response[resolverv1.CheckResolverHealthResponse], error) {
	status, checks, err := h.service.Health(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&resolverv1.CheckResolverHealthResponse{Status: toProtoStatus(status), Checks: checks}), nil
}

func toProtoStatus(status domainresolver.Status) *resolverv1.ResolverStatus {
	return &resolverv1.ResolverStatus{
		Backend:          status.Backend,
		Status:           status.Status,
		BaseUrl:          status.BaseURL,
		Upstreams:        status.Upstreams,
		FilteringEnabled: status.FilteringEnabled,
		Warnings:         status.Warnings,
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
