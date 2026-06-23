package adapters

import (
	"context"
	"runtime"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"network-manager/internal/module"

	adaptersv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/adapters"
	adaptersconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/adapters/adapters_v1connect"
)

type handler struct{}

func Module() module.Module {
	path, h := adaptersconnect.NewAdapterServiceHandler(&handler{})
	return module.Module{Name: "adapters", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
	}, Endpoints: Endpoints}
}

func Schema() string { return "" }

func (h *handler) ListCapabilities(context.Context, *connect.Request[adaptersv1.ListCapabilitiesRequest]) (*connect.Response[adaptersv1.ListCapabilitiesResponse], error) {
	return connect.NewResponse(&adaptersv1.ListCapabilitiesResponse{Capabilities: capabilities()}), nil
}

func (h *handler) ExplainUnsupportedAction(_ context.Context, req *connect.Request[adaptersv1.ExplainUnsupportedActionRequest]) (*connect.Response[adaptersv1.ExplainUnsupportedActionResponse], error) {
	action := req.Msg.GetAction()
	if action == "" {
		action = "router_write"
	}
	return connect.NewResponse(&adaptersv1.ExplainUnsupportedActionResponse{Capability: &adaptersv1.Capability{Adapter: "manual", Action: action, Supported: false, Reason: "No concrete adapter is configured yet."}, ManualSteps: []string{"Use read-only diagnostics until a supported adapter is implemented."}}), nil
}

func (h *handler) GetPlatformSummary(context.Context, *connect.Request[adaptersv1.GetPlatformSummaryRequest]) (*connect.Response[adaptersv1.GetPlatformSummaryResponse], error) {
	return connect.NewResponse(&adaptersv1.GetPlatformSummaryResponse{Summary: &adaptersv1.PlatformSummary{Os: runtime.GOOS, Arch: runtime.GOARCH, Profile: "auto", Notes: []string{"Platform detection is wired; host network adapter is not implemented yet."}}}), nil
}

func capabilities() []*adaptersv1.Capability {
	return []*adaptersv1.Capability{
		{Adapter: "host-" + runtime.GOOS, Action: "read_network_status", Supported: true, Reason: "Runtime OS detection is available."},
		{Adapter: "adguard-home", Action: "manage_dns_filtering", Supported: false, RollbackSupported: true, Reason: "Resolver adapter is planned but not configured."},
		{Adapter: "manual-router", Action: "router_dns_enforcement", Supported: false, Reason: "P0 does not perform unsupported router writes."},
	}
}

var Endpoints = []module.EndpointDescriptor{
	connectEndpoint("adapters_capabilities", adaptersconnect.AdapterServiceListCapabilitiesProcedure, "List adapter capabilities"),
	connectEndpoint("adapters_explain_unsupported", adaptersconnect.AdapterServiceExplainUnsupportedActionProcedure, "Explain unsupported adapter action"),
	connectEndpoint("adapters_platform", adaptersconnect.AdapterServiceGetPlatformSummaryProcedure, "Get platform summary"),
}

func connectEndpoint(id, path, summary string) module.EndpointDescriptor {
	return module.EndpointDescriptor{ID: id, Path: path, Method: "POST", Summary: summary, Category: "adapters", Request: &module.Schema{Type: "object", Properties: map[string]string{}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"capabilities": "array<Capability>"}}}
}
