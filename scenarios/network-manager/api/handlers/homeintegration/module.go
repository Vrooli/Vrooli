package homeintegration

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"network-manager/internal/module"

	homev1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/home_integration"
	homeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/home_integration/home_integration_v1connect"
)

type handler struct{}

func Module() module.Module {
	path, h := homeconnect.NewHomeIntegrationServiceHandler(&handler{})
	return module.Module{Name: "home_integration", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
	}, Endpoints: Endpoints}
}

func Schema() string { return "" }

func (h *handler) ListActions(context.Context, *connect.Request[homev1.ListActionsRequest]) (*connect.Response[homev1.ListActionsResponse], error) {
	return connect.NewResponse(&homev1.ListActionsResponse{Actions: []*homev1.HomeAction{
		{Name: "network.health.run", Description: "Run a read-only network health snapshot.", Effect: "read", ApprovalRequired: false},
		{Name: "network.filtering.pause", Description: "Pause filtering for a device or group after approval.", Effect: "write", ApprovalRequired: true},
	}}), nil
}

func (h *handler) InvokeAction(_ context.Context, req *connect.Request[homev1.InvokeActionRequest]) (*connect.Response[homev1.InvokeActionResponse], error) {
	name := req.Msg.GetName()
	if name == "" {
		name = "network.health.run"
	}
	return connect.NewResponse(&homev1.InvokeActionResponse{Status: "preview", Message: "Home Automation action contract is wired; no network mutation performed.", Event: event("event-preview", name)}), nil
}

func (h *handler) ListRecentEvents(context.Context, *connect.Request[homev1.ListRecentEventsRequest]) (*connect.Response[homev1.ListRecentEventsResponse], error) {
	return connect.NewResponse(&homev1.ListRecentEventsResponse{Events: []*homev1.HomeEvent{event("event-preview", "network.manager.ready")}}), nil
}

func event(id, eventType string) *homev1.HomeEvent {
	return &homev1.HomeEvent{Id: id, Type: eventType, Summary: "Network Manager integration event scaffold.", OccurredAt: time.Now().UTC().Format(time.RFC3339)}
}

var Endpoints = []module.EndpointDescriptor{
	connectEndpoint("home_actions_list", homeconnect.HomeIntegrationServiceListActionsProcedure, "List Home Automation actions"),
	connectEndpoint("home_action_invoke", homeconnect.HomeIntegrationServiceInvokeActionProcedure, "Invoke Home Automation action"),
	connectEndpoint("home_events_list", homeconnect.HomeIntegrationServiceListRecentEventsProcedure, "List recent Home Automation events"),
}

func connectEndpoint(id, path, summary string) module.EndpointDescriptor {
	return module.EndpointDescriptor{ID: id, Path: path, Method: "POST", Summary: summary, Category: "home_integration", Request: &module.Schema{Type: "object", Properties: map[string]string{}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"events": "array<HomeEvent>"}}}
}
