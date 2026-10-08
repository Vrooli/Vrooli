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

func (h *handler) owner(ctx context.Context, headers http.Header) (string, error) {
	subject, err := identity.Subject(ctx, headers, h.verifier)
	if err != nil {
		return "", connect.NewError(connect.CodeUnauthenticated, errors.New("verified owner identity is required"))
	}
	return subject, nil
}

func (h *handler) Ask(ctx context.Context, req *connect.Request[v1.AskRequest]) (*connect.Response[v1.AskResponse], error) {
	subject, err := h.owner(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	deadline, err := time.Parse(time.RFC3339, req.Msg.GetDeadline())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	options := make([]hub.AskOption, 0, len(req.Msg.GetOptions())+len(req.Msg.GetAllowedAnswers()))
	for _, option := range req.Msg.GetOptions() {
		options = append(options, hub.AskOption{Key: option.GetKey(), Label: option.GetLabel()})
	}
	if len(options) == 0 {
		for _, key := range req.Msg.GetAllowedAnswers() {
			options = append(options, hub.AskOption{Key: key, Label: key})
		}
	}
	ask, notification, err := h.service.AskWithSpec(ctx, hub.AskSpec{
		Recipient:            subject,
		Question:             req.Msg.GetQuestion(),
		Options:              options,
		Recommended:          req.Msg.GetRecommended(),
		RecommendationReason: req.Msg.GetRecommendationReason(),
		DefaultAnswer:        req.Msg.GetDefaultAnswer(),
		Reversible:           req.Msg.GetReversible(),
		Urgency:              req.Msg.GetUrgency(),
		ContextURL:           req.Msg.GetContextUrl(),
		Deadline:             deadline,
		SensitivityLabel:     req.Msg.GetSensitivityLabel(),
		IdempotencyKey:       req.Msg.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&v1.AskResponse{AskId: ask.ID, Notification: &shared.Notification{Id: notification.ID, RequestedBy: notification.RequestedBy, Title: notification.Title, Body: notification.Body, Urgency: notification.Urgency, SensitivityLabel: notification.SensitivityLabel, IdempotencyKey: notification.IdempotencyKey, DedupeKey: notification.DedupeKey, State: notificationState(notification.State), Reason: notification.Reason, CreatedAt: notification.CreatedAt, UpdatedAt: notification.UpdatedAt}}), nil
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
	actor, err := h.owner(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	ask, err := h.service.AnswerAsk(ctx, hub.AnswerInput{AskID: req.Msg.GetAskId(), Answer: req.Msg.GetAnswer(), Actor: actor, Note: req.Msg.GetNote()})
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&v1.AnswerResponse{AskId: ask.ID, Answer: ask.Answer, AnsweredAt: ask.AnsweredAt, Late: ask.Late}), nil
}

// Wait stays readable without an owner token: autoheal's approval check
// reads it, and a read never changes an ask.
func (h *handler) Wait(ctx context.Context, req *connect.Request[v1.WaitRequest]) (*connect.Response[v1.WaitResponse], error) {
	deadline, err := time.Parse(time.RFC3339, req.Msg.GetDeadline())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	state, answer, reason, err := h.service.Wait(ctx, req.Msg.GetAskId(), deadline)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&v1.WaitResponse{AskId: req.Msg.GetAskId(), State: state, Answer: answer, Reason: reason}), nil
}

func (h *handler) GetAsk(ctx context.Context, req *connect.Request[v1.GetAskRequest]) (*connect.Response[v1.GetAskResponse], error) {
	subject, err := h.owner(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	ask, err := h.service.GetAsk(ctx, req.Msg.GetAskId())
	if err != nil {
		return nil, connectError(err)
	}
	if ask.Recipient != subject {
		return nil, connect.NewError(connect.CodeNotFound, hub.ErrNotFound)
	}
	return connect.NewResponse(&v1.GetAskResponse{Ask: h.toProto(ask)}), nil
}

func (h *handler) ListAsks(ctx context.Context, req *connect.Request[v1.ListAsksRequest]) (*connect.Response[v1.ListAsksResponse], error) {
	subject, err := h.owner(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	asks, err := h.service.ListAsks(ctx, subject, req.Msg.GetOpenOnly(), int(req.Msg.GetLimit()))
	if err != nil {
		return nil, connectError(err)
	}
	out := make([]*v1.Ask, 0, len(asks))
	for _, ask := range asks {
		out = append(out, h.toProto(ask))
	}
	return connect.NewResponse(&v1.ListAsksResponse{Asks: out}), nil
}

func (h *handler) toProto(ask hub.AskRecord) *v1.Ask {
	options := make([]*v1.AskOption, 0, len(ask.Options))
	for _, option := range ask.Options {
		options = append(options, &v1.AskOption{Key: option.Key, Label: option.Label})
	}
	return &v1.Ask{
		Id: ask.ID, NotificationId: ask.NotificationID, Question: ask.Question, Options: options,
		Recommended: ask.Recommended, RecommendationReason: ask.RecommendationReason, DefaultAnswer: ask.DefaultAnswer,
		Reversible: ask.Reversible, Urgency: ask.Urgency, ContextUrl: ask.ContextURL, Deadline: ask.Deadline,
		State: ask.State, Reason: ask.Reason, Answer: ask.Answer, AnswerLabel: ask.AnswerLabel, Note: ask.Note,
		AnsweredBy: ask.AnsweredBy, AnsweredAt: ask.AnsweredAt, Late: ask.Late, DefaultAppliedAt: ask.DefaultAppliedAt,
		FirstDeliveredAt: ask.FirstDeliveredAt, DefaultEligibleAt: h.service.DefaultEligibleAt(ask),
		Source: ask.Source, SourceEventType: ask.SourceEventType, RequesterRef: ask.RequesterRef,
		ResolvedAt: ask.ResolvedAt, CreatedAt: ask.CreatedAt, UpdatedAt: ask.UpdatedAt,
	}
}

func connectError(err error) error {
	switch {
	case errors.Is(err, hub.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, hub.ErrAskResolved):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, hub.ErrInvalidArgument):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "conversations_ask", Path: connectv1.ConversationsServiceAskProcedure, Method: http.MethodPost, Summary: "Ask a question and persist its deadline", Category: "conversations"},
	{ID: "conversations_answer", Path: connectv1.ConversationsServiceAnswerProcedure, Method: http.MethodPost, Summary: "Record an allowed answer", Category: "conversations"},
	{ID: "conversations_wait", Path: connectv1.ConversationsServiceWaitProcedure, Method: http.MethodPost, Summary: "Block until an ask is resolved or the caller deadline passes", Category: "conversations"},
	{ID: "conversations_get_ask", Path: connectv1.ConversationsServiceGetAskProcedure, Method: http.MethodPost, Summary: "Read one ask without changing it", Category: "conversations"},
	{ID: "conversations_list_asks", Path: connectv1.ConversationsServiceListAsksProcedure, Method: http.MethodPost, Summary: "List the owner's asks, optionally only open ones", Category: "conversations"},
}
