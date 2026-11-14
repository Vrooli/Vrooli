package homeintegration

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	domainhome "network-manager/internal/homeintegration"
	"network-manager/internal/module"

	homev1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/home_integration"
	homeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/home_integration/home_integration_v1connect"
)

type handler struct {
	service *domainhome.Service
}

func Module(db domainhome.SQLExecutor) module.Module {
	service := domainhome.NewService(domainhome.Config{Repo: domainhome.NewSQLiteRepository(db)})
	path, h := homeconnect.NewHomeIntegrationServiceHandler(&handler{service: service})
	return module.Module{Name: "home_integration", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
	}, Endpoints: Endpoints}
}

func Schema() string { return domainhome.Schema() }

func (h *handler) ListActions(ctx context.Context, _ *connect.Request[homev1.ListActionsRequest]) (*connect.Response[homev1.ListActionsResponse], error) {
	actions := h.service.ListActions(ctx)
	out := make([]*homev1.HomeAction, 0, len(actions))
	for _, action := range actions {
		out = append(out, &homev1.HomeAction{
			Name:             action.Name,
			Description:      action.Description,
			Effect:           action.Effect,
			ApprovalRequired: action.ApprovalRequired,
		})
	}
	return connect.NewResponse(&homev1.ListActionsResponse{Actions: out}), nil
}

func (h *handler) InvokeAction(ctx context.Context, req *connect.Request[homev1.InvokeActionRequest]) (*connect.Response[homev1.InvokeActionResponse], error) {
	invocation, evt, err := h.service.InvokeAction(ctx, req.Msg.GetName(), req.Msg.GetParams(), req.Msg.GetApproved())
	if err != nil {
		if errors.Is(err, domainhome.ErrUnknownAction) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&homev1.InvokeActionResponse{Status: invocation.Status, Message: invocation.Message, Event: toProtoEvent(evt)}), nil
}

func (h *handler) ListRecentEvents(ctx context.Context, _ *connect.Request[homev1.ListRecentEventsRequest]) (*connect.Response[homev1.ListRecentEventsResponse], error) {
	events, err := h.service.ListRecentEvents(ctx, 25)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	out := make([]*homev1.HomeEvent, 0, len(events))
	for _, evt := range events {
		out = append(out, toProtoEvent(evt))
	}
	return connect.NewResponse(&homev1.ListRecentEventsResponse{Events: out}), nil
}

func toProtoEvent(evt domainhome.Event) *homev1.HomeEvent {
	return &homev1.HomeEvent{
		Id:         evt.ID,
		Type:       evt.Type,
		Summary:    evt.Summary,
		OccurredAt: evt.OccurredAt.UTC().Format(time.RFC3339),
	}
}

var Endpoints = []module.EndpointDescriptor{
	connectEndpoint("home_actions_list", homeconnect.HomeIntegrationServiceListActionsProcedure, "List Home Automation actions"),
	connectEndpoint("home_action_invoke", homeconnect.HomeIntegrationServiceInvokeActionProcedure, "Invoke Home Automation action"),
	connectEndpoint("home_events_list", homeconnect.HomeIntegrationServiceListRecentEventsProcedure, "List recent Home Automation events"),
}

func connectEndpoint(id, path, summary string) module.EndpointDescriptor {
	return module.EndpointDescriptor{ID: id, Path: path, Method: "POST", Summary: summary, Category: "home_integration", Request: &module.Schema{Type: "object", Properties: map[string]string{}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"events": "array<HomeEvent>"}}}
}
