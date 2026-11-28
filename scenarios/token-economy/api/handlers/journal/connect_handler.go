package journal

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	domain "token-economy/internal/journal"

	accessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access"
)

type connectHandler struct {
	service domain.Service
	logger  *log.Logger
}

func NewConnectHandler(service domain.Service, logger *log.Logger) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &connectHandler{service: service, logger: logger}
}

func (h *connectHandler) ListJournalEvents(ctx context.Context, req *connect.Request[accessv1.ListJournalEventsRequest]) (*connect.Response[accessv1.ListJournalEventsResponse], error) {
	events, err := h.service.Events(ctx, req.Msg.HolderId, req.Msg.TokenTypeId)
	if err != nil {
		return nil, h.mapReadError("ListJournalEvents", err)
	}
	return connect.NewResponse(&accessv1.ListJournalEventsResponse{Events: eventsToProto(events)}), nil
}

func (h *connectHandler) ShowBalance(ctx context.Context, req *connect.Request[accessv1.ShowBalanceRequest]) (*connect.Response[accessv1.ShowBalanceResponse], error) {
	balance, err := h.service.Balance(ctx, req.Msg.HolderId, req.Msg.TokenTypeId)
	if err != nil {
		return nil, h.mapReadError("ShowBalance", err)
	}
	return connect.NewResponse(&accessv1.ShowBalanceResponse{Balance: &accessv1.Balance{TokenTypeId: balance.TokenTypeID, Amount: balance.Amount}}), nil
}

func (h *connectHandler) ExportJournal(ctx context.Context, req *connect.Request[accessv1.ExportJournalRequest]) (*connect.Response[accessv1.ExportJournalResponse], error) {
	events, err := h.service.Events(ctx, req.Msg.HolderId, req.Msg.TokenTypeId)
	if err != nil {
		return nil, h.mapReadError("ExportJournal", err)
	}
	return connect.NewResponse(&accessv1.ExportJournalResponse{Events: eventsToProto(events)}), nil
}

func (h *connectHandler) ReverseEvent(ctx context.Context, subject string, req *connect.Request[accessv1.ReverseEventRequest]) (*connect.Response[accessv1.ReverseEventResponse], error) {
	value, err := h.service.Reverse(ctx, domain.ReverseInput{
		OriginalEventID: req.Msg.OriginalEventId,
		Reason:          req.Msg.Reason,
		IdempotencyKey:  req.Msg.IdempotencyKey,
		ActorIdentity:   subject,
	})
	if err != nil {
		var invalid *domain.InvalidReversalError
		switch {
		case errors.As(err, &invalid):
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case errors.Is(err, domain.ErrEventNotFound):
			return nil, connect.NewError(connect.CodeNotFound, err)
		case errors.Is(err, domain.ErrEventAlreadyReversed):
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		default:
			h.logger.Printf("journal.ReverseEvent: %v", err)
			return nil, connect.NewError(connect.CodeInternal, errors.New("internal error"))
		}
	}
	return connect.NewResponse(&accessv1.ReverseEventResponse{Reversal: eventToProto(value)}), nil
}

func (h *connectHandler) mapReadError(operation string, err error) error {
	if errors.Is(err, domain.ErrInvalidJournalEvent) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	h.logger.Printf("journal.%s: %v", operation, err)
	return connect.NewError(connect.CodeInternal, errors.New("internal error"))
}

func eventsToProto(events []domain.Event) []*accessv1.Event {
	out := make([]*accessv1.Event, 0, len(events))
	for _, event := range events {
		out = append(out, eventToProto(event))
	}
	return out
}

func eventToProto(value domain.Event) *accessv1.Event {
	return &accessv1.Event{
		Id: value.ID, TokenTypeId: value.TokenTypeID, Amount: value.Amount,
		Kind: accessv1.EventKind(eventKindToProto(value.Kind)), Reason: value.Reason,
		CreatedAt: timestamppb.New(value.CreatedAt), ActorIdentity: value.ActorIdentity,
		CauseReference: value.CauseReference, ActorKind: actorKindToProto(value.ActorKind),
		ActorVerificationStatus: verificationStatusToProto(value.ActorVerificationStatus),
		ActorRunId:              value.ActorRunID,
	}
}

func eventKindToProto(value domain.EventKind) int32 {
	switch value {
	case domain.EventKindMint:
		return int32(accessv1.EventKind_EVENT_KIND_MINT)
	case domain.EventKindCredit:
		return int32(accessv1.EventKind_EVENT_KIND_CREDIT)
	case domain.EventKindDebit:
		return int32(accessv1.EventKind_EVENT_KIND_DEBIT)
	case domain.EventKindReversal:
		return int32(accessv1.EventKind_EVENT_KIND_REVERSAL)
	case domain.EventKindExpiry:
		return int32(accessv1.EventKind_EVENT_KIND_EXPIRY)
	default:
		return int32(accessv1.EventKind_EVENT_KIND_UNSPECIFIED)
	}
}

func actorKindToProto(value string) accessv1.ActorKind {
	switch value {
	case domain.ActorKindAgent:
		return accessv1.ActorKind_ACTOR_KIND_AGENT
	case domain.ActorKindOperator:
		return accessv1.ActorKind_ACTOR_KIND_OPERATOR
	default:
		return accessv1.ActorKind_ACTOR_KIND_UNSPECIFIED
	}
}

func verificationStatusToProto(value string) accessv1.VerificationStatus {
	switch value {
	case domain.VerificationVerified:
		return accessv1.VerificationStatus_VERIFICATION_STATUS_VERIFIED
	case domain.VerificationUnavailable:
		return accessv1.VerificationStatus_VERIFICATION_STATUS_UNAVAILABLE
	case domain.VerificationInvalid:
		return accessv1.VerificationStatus_VERIFICATION_STATUS_INVALID
	case domain.VerificationAbsent:
		return accessv1.VerificationStatus_VERIFICATION_STATUS_ABSENT
	default:
		return accessv1.VerificationStatus_VERIFICATION_STATUS_UNSPECIFIED
	}
}
