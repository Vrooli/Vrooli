package redemption

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	domain "token-economy/internal/redemption"

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

func (h *connectHandler) RequestRedemption(ctx context.Context, subject string, req *connect.Request[accessv1.RequestRedemptionRequest]) (*connect.Response[accessv1.RequestRedemptionResponse], error) {
	if req.Msg.Redemption == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("redemption is required"))
	}
	value, err := h.service.Request(ctx, domain.RequestInput{
		AuthenticatedSubject: subject,
		CatalogEntryID:       req.Msg.Redemption.CatalogEntryId,
		GrantID:              req.Msg.Redemption.GrantId,
		IdempotencyKey:       req.Msg.IdempotencyKey,
		Evidence:             req.Msg.Evidence,
	})
	if err != nil {
		return nil, h.mapError("RequestRedemption", err)
	}
	return connect.NewResponse(&accessv1.RequestRedemptionResponse{Redemption: redemptionToProto(value)}), nil
}

func (h *connectHandler) ListPendingRedemptions(ctx context.Context, _ *connect.Request[accessv1.ListPendingRedemptionsRequest]) (*connect.Response[accessv1.ListPendingRedemptionsResponse], error) {
	values, err := h.service.ListPending(ctx)
	if err != nil {
		return nil, h.mapError("ListPendingRedemptions", err)
	}
	out := &accessv1.ListPendingRedemptionsResponse{Redemptions: make([]*accessv1.Redemption, 0, len(values))}
	for _, value := range values {
		out.Redemptions = append(out.Redemptions, redemptionToProto(value))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ListHolderRedemptions(ctx context.Context, subject string) ([]*accessv1.Redemption, error) {
	values, err := h.service.ListForSubject(ctx, subject)
	if err != nil {
		return nil, h.mapError("ListHolderRedemptions", err)
	}
	out := make([]*accessv1.Redemption, 0, len(values))
	for _, value := range values {
		out = append(out, redemptionToProto(value))
	}
	return out, nil
}

func (h *connectHandler) ApproveRedemption(ctx context.Context, subject string, req *connect.Request[accessv1.ApproveRedemptionRequest]) (*connect.Response[accessv1.ApproveRedemptionResponse], error) {
	value, err := h.service.Approve(ctx, domain.DecisionInput{RedemptionID: req.Msg.RedemptionId, DeciderSubject: subject, Reason: req.Msg.Reason, IdempotencyKey: req.Msg.IdempotencyKey})
	if err != nil {
		return nil, h.mapError("ApproveRedemption", err)
	}
	return connect.NewResponse(&accessv1.ApproveRedemptionResponse{Redemption: redemptionToProto(value)}), nil
}

func (h *connectHandler) DenyRedemption(ctx context.Context, subject string, req *connect.Request[accessv1.DenyRedemptionRequest]) (*connect.Response[accessv1.DenyRedemptionResponse], error) {
	value, err := h.service.Deny(ctx, domain.DecisionInput{RedemptionID: req.Msg.RedemptionId, DeciderSubject: subject, Reason: req.Msg.Reason, IdempotencyKey: req.Msg.IdempotencyKey})
	if err != nil {
		return nil, h.mapError("DenyRedemption", err)
	}
	return connect.NewResponse(&accessv1.DenyRedemptionResponse{Redemption: redemptionToProto(value)}), nil
}

func (h *connectHandler) mapError(operation string, err error) error {
	var invalid *domain.InvalidRedemptionError
	switch {
	case errors.As(err, &invalid):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, domain.ErrRedemptionNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, domain.ErrInsufficientBalance), errors.Is(err, domain.ErrGrantRefused), errors.Is(err, domain.ErrRedemptionConflict), errors.Is(err, domain.ErrCatalogChanged):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		h.logger.Printf("redemption.%s: %v", operation, err)
		return connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}
}

func redemptionToProto(value domain.Redemption) *accessv1.Redemption {
	out := &accessv1.Redemption{
		Id: value.ID, CatalogEntryId: value.CatalogEntryID, HolderId: value.HolderID,
		TokenTypeId: value.TokenTypeID, GrantId: value.GrantID, Amount: value.Amount,
		IdempotencyKey: value.IdempotencyKey, State: stateToProto(value.State),
		DeciderSubject: value.DeciderSubject, DecisionReason: value.DecisionReason,
		RequestedAt: timestamppb.New(value.RequestedAt),
	}
	if value.DecidedAt != nil {
		out.DecidedAt = timestamppb.New(*value.DecidedAt)
	}
	if value.SettledAt != nil {
		out.SettledAt = timestamppb.New(*value.SettledAt)
	}
	return out
}

func stateToProto(value domain.State) accessv1.RedemptionState {
	switch value {
	case domain.StatePendingApproval:
		return accessv1.RedemptionState_REDEMPTION_STATE_PENDING_APPROVAL
	case domain.StateSettled:
		return accessv1.RedemptionState_REDEMPTION_STATE_SETTLED
	case domain.StateDenied:
		return accessv1.RedemptionState_REDEMPTION_STATE_DENIED
	default:
		return accessv1.RedemptionState_REDEMPTION_STATE_UNSPECIFIED
	}
}
