package notifications

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"notification-hub/internal/hub"
	identity "notification-hub/internal/identity"
	"notification-hub/internal/module"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/notifications/notifications_v1connect"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/shared"
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
	return module.Module{Name: "notifications", Mount: func(r *mux.Router) {
		path, svc := connectv1.NewNotificationsServiceHandler(h)
		r.PathPrefix(path).Handler(svc)
	}, Endpoints: Endpoints}
}

func (h *handler) Send(ctx context.Context, req *connect.Request[v1.SendRequest]) (*connect.Response[v1.SendResponse], error) {
	subject, err := identity.Subject(ctx, req.Header(), h.verifier)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified owner identity is required"))
	}
	if req.Msg.GetSensitivityLabel() == "" || req.Msg.GetIdempotencyKey() == "" || req.Msg.GetBody() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("body, sensitivity_label, and idempotency_key are required"))
	}
	var scheduled time.Time
	if value := req.Msg.GetScheduledAt(); value != "" {
		scheduled, err = time.Parse(time.RFC3339, value)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("scheduled_at must be RFC3339"))
		}
	}
	n, err := h.service.Send(ctx, hub.SendInput{RequestedBy: subject, Title: req.Msg.GetTitle(), Body: req.Msg.GetBody(), Urgency: req.Msg.GetUrgency(), SensitivityLabel: req.Msg.GetSensitivityLabel(), IdempotencyKey: req.Msg.GetIdempotencyKey(), DedupeKey: req.Msg.GetDedupeKey(), ScheduledAt: scheduled, DigestWindow: time.Duration(req.Msg.GetDigestWindowSeconds()) * time.Second})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&v1.SendResponse{Notification: toProto(n)}), nil
}

type relayPayload struct {
	Title            string `json:"title"`
	Body             string `json:"body"`
	Urgency          string `json:"urgency"`
	SensitivityLabel string `json:"sensitivity_label"`
	IdempotencyKey   string `json:"idempotency_key"`
	DedupeKey        string `json:"dedupe_key"`
	DedupeWindowSecs int64  `json:"dedupe_window_seconds"`
}

// Relay is the bridge-safe ingress. The payload is one base64 token so the
// typed bridge argv path never has to carry shell-significant notification
// content. The remote node authenticates the request and resolves its local
// recipient projection before accepting the durable notification.
func (h *handler) Relay(ctx context.Context, req *connect.Request[v1.RelayRequest]) (*connect.Response[v1.RelayResponse], error) {
	subject, err := identity.Subject(ctx, req.Header(), h.verifier)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified owner identity is required"))
	}
	raw, err := base64.RawStdEncoding.DecodeString(req.Msg.GetPayloadBase64())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("payload_base64 is invalid"))
	}
	var payload relayPayload
	if err := json.Unmarshal(raw, &payload); err != nil || payload.Body == "" || payload.SensitivityLabel == "" || payload.IdempotencyKey == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("relay payload is invalid"))
	}
	n, err := h.service.Send(ctx, hub.SendInput{RequestedBy: subject, Title: payload.Title, Body: payload.Body, Urgency: payload.Urgency, SensitivityLabel: payload.SensitivityLabel, IdempotencyKey: payload.IdempotencyKey, DedupeKey: payload.DedupeKey, DedupeWindow: time.Duration(payload.DedupeWindowSecs) * time.Second})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&v1.RelayResponse{Notification: toProto(n)}), nil
}

func (h *handler) Get(ctx context.Context, req *connect.Request[v1.GetRequest]) (*connect.Response[v1.GetResponse], error) {
	if _, err := identity.Subject(ctx, req.Header(), h.verifier); err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified owner identity is required"))
	}
	n, receipts, err := h.service.Get(ctx, req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	out := &v1.GetResponse{Notification: toProto(n)}
	for _, receipt := range receipts {
		out.Receipts = append(out.Receipts, receiptProto(receipt))
	}
	return connect.NewResponse(out), nil
}

func (h *handler) List(ctx context.Context, req *connect.Request[v1.ListRequest]) (*connect.Response[v1.ListResponse], error) {
	if _, err := identity.Subject(ctx, req.Header(), h.verifier); err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified owner identity is required"))
	}
	items, err := h.service.List(ctx, int(req.Msg.GetLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListResponse{}
	for _, item := range items {
		out.Notifications = append(out.Notifications, toProto(item))
	}
	return connect.NewResponse(out), nil
}

func toProto(n hub.Notification) *shared.Notification {
	return &shared.Notification{Id: n.ID, RequestedBy: n.RequestedBy, Title: n.Title, Body: n.Body, Urgency: n.Urgency, SensitivityLabel: n.SensitivityLabel, IdempotencyKey: n.IdempotencyKey, DedupeKey: n.DedupeKey, State: StateProto(n.State), Reason: n.Reason, CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt}
}

func StateProto(state string) shared.NotificationState {
	switch state {
	case "pending":
		return shared.NotificationState_NOTIFICATION_STATE_PENDING
	case "held":
		return shared.NotificationState_NOTIFICATION_STATE_HELD
	case "routed":
		return shared.NotificationState_NOTIFICATION_STATE_ROUTED
	case "delivered":
		return shared.NotificationState_NOTIFICATION_STATE_DELIVERED
	case "failed":
		return shared.NotificationState_NOTIFICATION_STATE_FAILED
	case "unroutable":
		return shared.NotificationState_NOTIFICATION_STATE_UNROUTABLE
	case "suppressed":
		return shared.NotificationState_NOTIFICATION_STATE_SUPPRESSED
	default:
		return shared.NotificationState_NOTIFICATION_STATE_UNSPECIFIED
	}
}

func receiptProto(r hub.Receipt) *shared.DeliveryReceipt {
	return &shared.DeliveryReceipt{Id: r.ID, NotificationId: r.NotificationID, Channel: r.Channel, MachineId: r.MachineID, ProviderId: r.ProviderID, DeliveredAt: r.DeliveredAt}
}

var Endpoints = []module.EndpointDescriptor{{ID: "notifications_send", Path: connectv1.NotificationsServiceSendProcedure, Method: http.MethodPost, Summary: "Accept a notification and return its durable id", Category: "notifications"}, {ID: "notifications_relay", Path: connectv1.NotificationsServiceRelayProcedure, Method: http.MethodPost, Summary: "Accept a bridge-safe base64 notification payload", Category: "notifications"}, {ID: "notifications_get", Path: connectv1.NotificationsServiceGetProcedure, Method: http.MethodPost, Summary: "Read notification state and receipts", Category: "notifications"}, {ID: "notifications_list", Path: connectv1.NotificationsServiceListProcedure, Method: http.MethodPost, Summary: "List the delivery timeline", Category: "notifications"}}
