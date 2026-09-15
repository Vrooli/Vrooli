package delivery

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	"github.com/gorilla/mux"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/delivery"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/delivery/delivery_v1connect"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/shared"

	notifications "notification-hub/handlers/notifications"
	"notification-hub/internal/hub"
	identity "notification-hub/internal/identity"
	"notification-hub/internal/module"
)

type handler struct {
	service  *hub.Service
	verifier identity.Verifier
}

func Module(service *hub.Service) module.Module {
	return ModuleWithVerifier(service, nil)
}

func ModuleWithVerifier(service *hub.Service, verifier identity.Verifier) module.Module {
	h := &handler{service: service, verifier: verifier}
	return module.Module{Name: "delivery", Mount: func(r *mux.Router) {
		path, svc := connectv1.NewDeliveryServiceHandler(h)
		r.PathPrefix(path).Handler(svc)
	}, Endpoints: Endpoints}
}

func (h *handler) GetTimeline(ctx context.Context, req *connect.Request[v1.GetTimelineRequest]) (*connect.Response[v1.GetTimelineResponse], error) {
	if _, err := identity.Subject(ctx, req.Header(), h.verifier); err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified owner identity is required"))
	}
	items, err := h.service.List(ctx, int(req.Msg.GetLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.GetTimelineResponse{}
	for _, n := range items {
		out.Notifications = append(out.Notifications, &shared.Notification{Id: n.ID, RequestedBy: n.RequestedBy, Title: n.Title, Body: n.Body, Urgency: n.Urgency, SensitivityLabel: n.SensitivityLabel, IdempotencyKey: n.IdempotencyKey, DedupeKey: n.DedupeKey, State: notifications.StateProto(n.State), Reason: n.Reason, CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt})
	}
	return connect.NewResponse(out), nil
}

func (h *handler) Deliver(ctx context.Context, req *connect.Request[v1.DeliverRequest]) (*connect.Response[v1.DeliverResponse], error) {
	if err := h.service.Process(ctx, req.Msg.GetId()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	_, receipts, err := h.service.Get(ctx, req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	out := &v1.DeliverResponse{}
	for _, r := range receipts {
		out.Receipts = append(out.Receipts, &shared.DeliveryReceipt{Id: r.ID, NotificationId: r.NotificationID, Channel: r.Channel, MachineId: r.MachineID, ProviderId: r.ProviderID, DeliveredAt: r.DeliveredAt})
	}
	return connect.NewResponse(out), nil
}

func (h *handler) GetAnalytics(ctx context.Context, req *connect.Request[v1.GetAnalyticsRequest]) (*connect.Response[v1.GetAnalyticsResponse], error) {
	if _, err := identity.Subject(ctx, req.Header(), h.verifier); err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified owner identity is required"))
	}
	analytics, err := h.service.Analytics(ctx, req.Msg.GetSince(), req.Msg.GetUntil())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	out := &v1.GetAnalyticsResponse{Since: analytics.Since, Until: analytics.Until, TotalNotifications: analytics.TotalNotifications}
	for _, channel := range analytics.Channels {
		out.Channels = append(out.Channels, &v1.AnalyticsChannel{Channel: channel.Channel, Delivered: channel.Delivered, Failed: channel.Failed, Attempts: channel.Attempts, FailureRate: channel.FailureRate, AverageLatencyMs: channel.AverageLatencyMillis})
	}
	return connect.NewResponse(out), nil
}

var Endpoints = []module.EndpointDescriptor{{ID: "delivery_timeline", Path: connectv1.DeliveryServiceGetTimelineProcedure, Method: http.MethodPost, Summary: "Read the delivery timeline", Category: "delivery"}, {ID: "delivery_deliver", Path: connectv1.DeliveryServiceDeliverProcedure, Method: http.MethodPost, Summary: "Deliver a durable notification", Category: "delivery"}, {ID: "delivery_analytics", Path: connectv1.DeliveryServiceGetAnalyticsProcedure, Method: http.MethodPost, Summary: "Report delivery counts, failure rates, and latency", Category: "delivery"}}
