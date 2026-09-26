package conversations

import (
	"context"
	"errors"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/gorilla/mux"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/shared"

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
	return module.Module{Name: "conversations", Mount: func(r *mux.Router) {
		path, svc := connectv1.NewConversationsServiceHandler(h)
		r.PathPrefix(path).Handler(svc)
	}, Endpoints: Endpoints}
}

func (h *handler) Ask(ctx context.Context, req *connect.Request[v1.AskRequest]) (*connect.Response[v1.AskResponse], error) {
	subject, err := identity.Subject(ctx, req.Header(), h.verifier)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified owner identity is required"))
	}
	deadline, err := time.Parse(time.RFC3339, req.Msg.GetDeadline())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	id, notification, err := h.service.Ask(ctx, subject, req.Msg.GetQuestion(), req.Msg.GetAllowedAnswers(), deadline, req.Msg.GetSensitivityLabel(), req.Msg.GetIdempotencyKey())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&v1.AskResponse{AskId: id, Notification: &shared.Notification{Id: notification.ID, RequestedBy: notification.RequestedBy, Title: notification.Title, Body: notification.Body, Urgency: notification.Urgency, SensitivityLabel: notification.SensitivityLabel, IdempotencyKey: notification.IdempotencyKey, DedupeKey: notification.DedupeKey, State: notificationState(notification.State), Reason: notification.Reason, CreatedAt: notification.CreatedAt, UpdatedAt: notification.UpdatedAt}}), nil
}

func notificationState(state string) shared.NotificationState {
	switch state {
	case "pending":
		return shared.NotificationState_NOTIFICATION_STATE_PENDING
	case "held":
		return shared.NotificationState_NOTIFICATION_STATE_HELD
	case "delivered":
		return shared.NotificationState_NOTIFICATION_STATE_DELIVERED
	case "failed":
		return shared.NotificationState_NOTIFICATION_STATE_FAILED
	default:
		return shared.NotificationState_NOTIFICATION_STATE_UNSPECIFIED
	}
}

func (h *handler) Answer(ctx context.Context, req *connect.Request[v1.AnswerRequest]) (*connect.Response[v1.AnswerResponse], error) {
	actor, err := identity.Subject(ctx, req.Header(), h.verifier)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified owner identity is required"))
	}
	if err := h.service.Answer(ctx, req.Msg.GetAskId(), req.Msg.GetAnswer(), actor); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&v1.AnswerResponse{AskId: req.Msg.GetAskId(), Answer: req.Msg.GetAnswer(), AnsweredAt: time.Now().UTC().Format(time.RFC3339Nano)}), nil
}

func (h *handler) Wait(ctx context.Context, req *connect.Request[v1.WaitRequest]) (*connect.Response[v1.WaitResponse], error) {
	deadline, err := time.Parse(time.RFC3339, req.Msg.GetDeadline())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	state, answer, reason, err := h.service.Wait(ctx, req.Msg.GetAskId(), deadline)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.WaitResponse{AskId: req.Msg.GetAskId(), State: state, Answer: answer, Reason: reason}), nil
}

var Endpoints = []module.EndpointDescriptor{{ID: "conversations_ask", Path: connectv1.ConversationsServiceAskProcedure, Method: http.MethodPost, Summary: "Ask a question and persist its deadline", Category: "conversations"}, {ID: "conversations_answer", Path: connectv1.ConversationsServiceAnswerProcedure, Method: http.MethodPost, Summary: "Record an allowed answer", Category: "conversations"}, {ID: "conversations_wait", Path: connectv1.ConversationsServiceWaitProcedure, Method: http.MethodPost, Summary: "Block until an ask is answered or expires", Category: "conversations"}}
