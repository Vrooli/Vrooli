package holders

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	domain "token-economy/internal/holders"

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

func (h *connectHandler) CreateHolder(ctx context.Context, req *connect.Request[accessv1.CreateHolderRequest]) (*connect.Response[accessv1.CreateHolderResponse], error) {
	holder, err := h.service.Add(ctx, domain.AddInput{
		DisplayName: req.Msg.DisplayName, AuthenticatorSubject: req.Msg.AuthenticatorSubject,
		IdempotencyKey: req.Msg.IdempotencyKey,
	})
	if err != nil {
		return nil, h.mapError("CreateHolder", err)
	}
	return connect.NewResponse(&accessv1.CreateHolderResponse{Holder: holderToProto(holder)}), nil
}

func (h *connectHandler) GetHolder(ctx context.Context, req *connect.Request[accessv1.GetHolderRequest]) (*connect.Response[accessv1.GetHolderResponse], error) {
	holder, err := h.service.Get(ctx, req.Msg.Id)
	if err != nil {
		return nil, h.mapError("GetHolder", err)
	}
	return connect.NewResponse(&accessv1.GetHolderResponse{Holder: holderToProto(holder)}), nil
}

func (h *connectHandler) ListHolders(ctx context.Context, _ *connect.Request[accessv1.ListHoldersRequest]) (*connect.Response[accessv1.ListHoldersResponse], error) {
	holders, err := h.service.List(ctx)
	if err != nil {
		return nil, h.mapError("ListHolders", err)
	}
	out := &accessv1.ListHoldersResponse{Holders: make([]*accessv1.Holder, 0, len(holders))}
	for _, holder := range holders {
		out.Holders = append(out.Holders, holderToProto(holder))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ViewEconomy(ctx context.Context, authenticatedSubject string, _ *connect.Request[accessv1.ViewEconomyRequest]) (*connect.Response[accessv1.ViewEconomyResponse], error) {
	view, err := h.service.View(ctx, authenticatedSubject)
	if err != nil {
		return nil, h.mapError("ViewEconomy", err)
	}
	response := &accessv1.ViewEconomyResponse{
		Holder:   holderToProto(view.Holder),
		Events:   make([]*accessv1.Event, 0, len(view.History.Events)),
		Balances: make([]*accessv1.Balance, 0, len(view.History.Balances)),
	}
	for _, event := range view.History.Events {
		response.Events = append(response.Events, &accessv1.Event{
			Id: event.ID, TokenTypeId: event.TokenTypeID, Amount: event.Amount,
			Kind: eventKindToProto(event.Kind), Reason: event.Reason,
			CreatedAt: timestamppb.New(event.CreatedAt), ActorIdentity: event.ActorIdentity,
			CauseReference: event.CauseReference, ActorKind: actorKindToProto(event.ActorKind),
			ActorVerificationStatus: verificationStatusToProto(event.ActorVerificationStatus),
			ActorRunId:              event.ActorRunID,
		})
	}
	for _, balance := range view.History.Balances {
		response.Balances = append(response.Balances, &accessv1.Balance{TokenTypeId: balance.TokenTypeID, Amount: balance.Amount})
	}
	return connect.NewResponse(response), nil
}

func holderToProto(holder domain.Holder) *accessv1.Holder {
	return &accessv1.Holder{
		Id: holder.ID, DisplayName: holder.DisplayName,
		AuthenticatorSubject: holder.AuthenticatorSubject, CreatedAt: timestamppb.New(holder.CreatedAt),
	}
}

func actorKindToProto(kind string) accessv1.ActorKind {
	switch kind {
	case "agent":
		return accessv1.ActorKind_ACTOR_KIND_AGENT
	case "operator":
		return accessv1.ActorKind_ACTOR_KIND_OPERATOR
	default:
		return accessv1.ActorKind_ACTOR_KIND_UNSPECIFIED
	}
}

func verificationStatusToProto(status string) accessv1.VerificationStatus {
	switch status {
	case "verified":
		return accessv1.VerificationStatus_VERIFICATION_STATUS_VERIFIED
	case "unavailable":
		return accessv1.VerificationStatus_VERIFICATION_STATUS_UNAVAILABLE
	case "invalid":
		return accessv1.VerificationStatus_VERIFICATION_STATUS_INVALID
	case "absent":
		return accessv1.VerificationStatus_VERIFICATION_STATUS_ABSENT
	default:
		return accessv1.VerificationStatus_VERIFICATION_STATUS_UNSPECIFIED
	}
}

func (h *connectHandler) mapError(operation string, err error) error {
	switch {
	case errors.Is(err, domain.ErrHolderNotFound):
		return connect.NewError(connect.CodeNotFound, errors.New("holder not found"))
	case errors.Is(err, domain.ErrInvalidHolder):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		h.logger.Printf("holders.%s: %v", operation, err)
		return connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}
}

func eventKindToProto(kind string) accessv1.EventKind {
	switch kind {
	case "mint":
		return accessv1.EventKind_EVENT_KIND_MINT
	case "credit":
		return accessv1.EventKind_EVENT_KIND_CREDIT
	case "debit":
		return accessv1.EventKind_EVENT_KIND_DEBIT
	case "reversal":
		return accessv1.EventKind_EVENT_KIND_REVERSAL
	case "expiry":
		return accessv1.EventKind_EVENT_KIND_EXPIRY
	default:
		return accessv1.EventKind_EVENT_KIND_UNSPECIFIED
	}
}
