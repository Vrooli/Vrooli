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

// producer admits a direct ask withdrawal from a verified agent run or the
// verified owner. It never admits an answer or a dismissal: those use owner.
func (h *handler) producer(ctx context.Context, headers http.Header) (identity.Producer, error) {
	producer, err := identity.ProducerFor(ctx, headers, h.verifier, h.service.ResolveRecipient(ctx))
	if err != nil {
		return identity.Producer{}, connect.NewError(connect.CodeUnauthenticated, errors.New("a verified agent run or verified owner identity is required"))
	}
	return producer, nil
}

func (h *handler) directProducer(ctx context.Context, headers http.Header) (identity.Producer, error) {
	producer, err := identity.ProducerForDirect(ctx, headers, h.verifier, h.service.ResolveRecipient(ctx))
	if err != nil {
		return identity.Producer{}, connect.NewError(connect.CodeUnauthenticated, errors.New("a verified agent run or verified owner identity is required"))
	}
	return producer, nil
}

func (h *handler) owner(ctx context.Context, headers http.Header) (string, error) {
	subject, err := identity.Subject(ctx, headers, h.verifier)
	if err != nil {
		return "", connect.NewError(connect.CodeUnauthenticated, errors.New("verified owner identity is required"))
	}
	return subject, nil
}

func (h *handler) Ask(ctx context.Context, req *connect.Request[v1.AskRequest]) (*connect.Response[v1.AskResponse], error) {
	producer, err := h.directProducer(ctx, req.Header())
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
	result, err := h.service.AskDecision(ctx, hub.AskSpec{
		Recipient:               producer.Recipient,
		Question:                req.Msg.GetQuestion(),
		Options:                 options,
		Recommended:             req.Msg.GetRecommended(),
		RecommendationReason:    req.Msg.GetRecommendationReason(),
		DefaultAnswer:           req.Msg.GetDefaultAnswer(),
		Reversible:              req.Msg.GetReversible(),
		Urgency:                 req.Msg.GetUrgency(),
		ContextURL:              req.Msg.GetContextUrl(),
		BigPicture:              req.Msg.GetBigPicture(),
		TechnicalDetails:        req.Msg.GetTechnicalDetails(),
		ProviderDecisionContext: providerDecisionContextFromProto(req.Msg.GetProviderDecisionContext()),
		Deadline:                deadline,
		DeadlineSpecified:       true,
		SensitivityLabel:        req.Msg.GetSensitivityLabel(),
		IdempotencyKey:          req.Msg.GetIdempotencyKey(),
		SubjectKey:              req.Msg.GetSubjectKey(),
		ReassertWithin:          time.Duration(req.Msg.GetReassertWithinSeconds()) * time.Second,
	}, producer.Key)
	if err != nil {
		return nil, connectError(err)
	}
	ask, notification := result.Ask, result.Notification
	return connect.NewResponse(&v1.AskResponse{AskId: ask.ID, Outcome: result.Outcome, Notification: &shared.Notification{Id: notification.ID, RequestedBy: notification.RequestedBy, Title: notification.Title, Body: notification.Body, Urgency: notification.Urgency, SensitivityLabel: notification.SensitivityLabel, IdempotencyKey: notification.IdempotencyKey, DedupeKey: notification.DedupeKey, State: notificationState(notification.State), Reason: notification.Reason, CreatedAt: notification.CreatedAt, UpdatedAt: notification.UpdatedAt}}), nil
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
	// Attribute the answer to the ACTUAL verified human (the Cloudflare Access
	// subject/email) while keeping the recipient-only authorization check keyed on
	// the recipient subject (actor). AnsweredBy is "" on the bearer/cookie path,
	// where AnswerAsk falls back to actor — preserving prior behavior.
	answerIdentityKind, answerIdentitySource := identity.AnswerIdentity(ctx)
	ask, err := h.service.AnswerAsk(ctx, hub.AnswerInput{
		AskID:                    req.Msg.GetAskId(),
		Answer:                   req.Msg.GetAnswer(),
		Actor:                    actor,
		AnsweredBy:               identity.AnsweredBy(ctx),
		AnsweredByIdentityKind:   answerIdentityKind,
		AnsweredByIdentitySource: answerIdentitySource,
		Note:                     req.Msg.GetNote(),
	})
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&v1.AnswerResponse{AskId: ask.ID, Answer: ask.Answer, AnsweredAt: ask.AnsweredAt, Late: ask.Late}), nil
}

func (h *handler) Dismiss(ctx context.Context, req *connect.Request[v1.DismissRequest]) (*connect.Response[v1.DismissResponse], error) {
	actor, err := h.owner(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	ask, err := h.service.DismissAsk(ctx, req.Msg.GetAskId(), actor, req.Msg.GetNote())
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&v1.DismissResponse{AskId: ask.ID, State: ask.State}), nil
}

func (h *handler) Withdraw(ctx context.Context, req *connect.Request[v1.WithdrawRequest]) (*connect.Response[v1.WithdrawResponse], error) {
	requester, err := h.producer(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	var ask hub.AskRecord
	switch {
	case req.Msg.GetSubjectKey() != "" && req.Msg.GetAskId() != "":
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("give ask_id or subject_key, not both"))
	case req.Msg.GetSubjectKey() != "":
		ask, err = h.service.WithdrawDirectSubject(ctx, requester.Recipient, req.Msg.GetSubjectKey(), req.Msg.GetReason())
	default:
		ask, err = h.service.WithdrawDirectAsk(ctx, req.Msg.GetAskId(), requester.Recipient, req.Msg.GetReason())
	}
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&v1.WithdrawResponse{AskId: ask.ID, State: ask.State}), nil
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
	recipient := h.service.ResolveRecipient(ctx)
	if recipient == "" {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("ask not found"))
	}
	ask, err := h.service.GetAskForRecipient(ctx, req.Msg.GetAskId(), recipient)
	if err != nil {
		return nil, connectError(err)
	}
	revisions, err := h.service.ListAskRevisions(ctx, ask.ID)
	if err != nil {
		return nil, connectError(err)
	}
	out := h.toProto(ask)
	for _, revision := range revisions {
		options := make([]*v1.AskOption, 0, len(revision.Options))
		for _, option := range revision.Options {
			options = append(options, &v1.AskOption{Key: option.Key, Label: option.Label})
		}
		out.Revisions = append(out.Revisions, &v1.AskRevision{
			Revision: int32(revision.Revision), Title: revision.Title, Question: revision.Question,
			RecommendationReason: revision.RecommendationReason, ContextUrl: revision.ContextURL,
			BigPicture: revision.BigPicture, TechnicalDetails: revision.TechnicalDetails, Options: options,
			Deadline: revision.Deadline, RevisedAt: revision.RevisedAt,
		})
	}
	return connect.NewResponse(&v1.GetAskResponse{Ask: out}), nil
}

func (h *handler) ListAsks(ctx context.Context, req *connect.Request[v1.ListAsksRequest]) (*connect.Response[v1.ListAsksResponse], error) {
	recipient := h.service.ResolveRecipient(ctx)
	if recipient == "" {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("recipient not found"))
	}
	page, err := h.service.ListAsksPage(ctx, recipient, hub.AskListOptions{
		OpenOnly: req.Msg.GetOpenOnly(), Limit: int(req.Msg.GetLimit()),
		Cursor: req.Msg.GetCursor(), Query: req.Msg.GetQuery(), StateFilter: req.Msg.GetStateFilter(),
	})
	if err != nil {
		return nil, connectError(err)
	}
	out := make([]*v1.Ask, 0, len(page.Asks))
	for _, ask := range page.Asks {
		out = append(out, h.toProto(ask))
	}
	return connect.NewResponse(&v1.ListAsksResponse{Asks: out, NextCursor: page.NextCursor}), nil
}

func (h *handler) toProto(ask hub.AskRecord) *v1.Ask {
	options := make([]*v1.AskOption, 0, len(ask.Options)+1)
	for _, option := range ask.Options {
		options = append(options, &v1.AskOption{Key: option.Key, Label: option.Label})
	}
	if !containsAskOption(ask.Options, "other") {
		options = append(options, &v1.AskOption{Key: "other", Label: "Other — I have a different answer"})
	}
	return &v1.Ask{
		Id: ask.ID, NotificationId: ask.NotificationID, Question: ask.Question, Title: ask.Title, Options: options,
		Recommended: ask.Recommended, RecommendationReason: ask.RecommendationReason, DefaultAnswer: ask.DefaultAnswer,
		Reversible: ask.Reversible, Urgency: ask.Urgency, ContextUrl: ask.ContextURL,
		BigPicture: ask.BigPicture, TechnicalDetails: ask.TechnicalDetails, Deadline: ask.Deadline,
		State: ask.State, Reason: ask.Reason, Answer: ask.Answer, AnswerLabel: ask.AnswerLabel, Note: ask.Note,
		AnsweredBy: ask.AnsweredBy, AnsweredAt: ask.AnsweredAt, Late: ask.Late, DefaultAppliedAt: ask.DefaultAppliedAt,
		AnsweredByIdentityKind: ask.AnsweredByIdentityKind, AnsweredByIdentitySource: ask.AnsweredByIdentitySource,
		FirstDeliveredAt: ask.FirstDeliveredAt, DefaultEligibleAt: h.service.DefaultEligibleAt(ask),
		Source: ask.Source, SourceEventType: ask.SourceEventType, RequesterRef: ask.RequesterRef,
		ResolvedAt: ask.ResolvedAt, CreatedAt: ask.CreatedAt, UpdatedAt: ask.UpdatedAt,
		ProviderDecisionContext: providerDecisionContextToProto(ask.ProviderDecisionContext),
		SubjectKey:              ask.SubjectKey, LastAssertedAt: ask.LastAssertedAt,
		ReassertWithinSeconds: int32(ask.ReassertWithinSeconds), RevisionCount: int32(ask.RevisionCount),
		SupersededByAskId: ask.SupersededByAskID,
	}
}

func containsAskOption(options []hub.AskOption, key string) bool {
	for _, option := range options {
		if option.Key == key {
			return true
		}
	}
	return false
}

func providerDecisionContextFromProto(value *v1.ProviderDecisionContext) *hub.ProviderDecisionContext {
	if value == nil {
		return nil
	}
	scope := ""
	switch value.GetScope() {
	case v1.ProviderDecisionContext_SCOPE_PROVIDER_RULE:
		scope = hub.ProviderDecisionScopeRule
	case v1.ProviderDecisionContext_SCOPE_PROVIDER_AGREEMENT:
		scope = hub.ProviderDecisionScopeAgreement
	}
	return &hub.ProviderDecisionContext{SchemaVersion: value.GetSchemaVersion(), CanonicalTermsSHA256: value.GetCanonicalTermsSha256(), Scope: scope}
}

func providerDecisionContextToProto(value *hub.ProviderDecisionContext) *v1.ProviderDecisionContext {
	if value == nil {
		return nil
	}
	scope := v1.ProviderDecisionContext_SCOPE_UNSPECIFIED
	switch value.Scope {
	case hub.ProviderDecisionScopeRule:
		scope = v1.ProviderDecisionContext_SCOPE_PROVIDER_RULE
	case hub.ProviderDecisionScopeAgreement:
		scope = v1.ProviderDecisionContext_SCOPE_PROVIDER_AGREEMENT
	}
	return &v1.ProviderDecisionContext{SchemaVersion: value.SchemaVersion, CanonicalTermsSha256: value.CanonicalTermsSHA256, Scope: scope}
}

func connectError(err error) error {
	switch {
	case errors.Is(err, hub.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, hub.ErrAskResolved):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, hub.ErrAskIdentityConflict):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, hub.ErrInvalidArgument):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "conversations_list_native_consents", Path: connectv1.ConversationsServiceListNativeConsentsProcedure, Method: http.MethodPost, Summary: "Read source-owned native consent without approving or creating an ask", Category: "conversations"},
	{ID: "conversations_ask", Path: connectv1.ConversationsServiceAskProcedure, Method: http.MethodPost, Summary: "Ask a question and persist its deadline", Category: "conversations"},
	{ID: "conversations_answer", Path: connectv1.ConversationsServiceAnswerProcedure, Method: http.MethodPost, Summary: "Record an allowed answer", Category: "conversations"},
	{ID: "conversations_dismiss", Path: connectv1.ConversationsServiceDismissProcedure, Method: http.MethodPost, Summary: "Dismiss an ask without answering", Category: "conversations"},
	{ID: "conversations_withdraw", Path: connectv1.ConversationsServiceWithdrawProcedure, Method: http.MethodPost, Summary: "Withdraw a direct ask you created", Category: "conversations"},
	{ID: "conversations_wait", Path: connectv1.ConversationsServiceWaitProcedure, Method: http.MethodPost, Summary: "Block until an ask is resolved or the caller deadline passes", Category: "conversations"},
	{ID: "conversations_get_ask", Path: connectv1.ConversationsServiceGetAskProcedure, Method: http.MethodPost, Summary: "Read one ask without changing it", Category: "conversations"},
	{ID: "conversations_list_asks", Path: connectv1.ConversationsServiceListAsksProcedure, Method: http.MethodPost, Summary: "List the owner's asks, optionally only open ones", Category: "conversations"},
}
