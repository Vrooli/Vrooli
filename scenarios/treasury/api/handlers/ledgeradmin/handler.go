package ledgeradmin

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/ledger"
	"google.golang.org/protobuf/types/known/timestamppb"
	"treasury/internal/ledger"
	"treasury/internal/operatorauth"
)

type EmissionReader interface {
	List(context.Context, string, string, int) ([]ledger.Emission, error)
}

type Handler struct {
	authorizer operatorauth.Authorizer
	reader     EmissionReader
}

func NewHandler(authorizer operatorauth.Authorizer, reader EmissionReader) *Handler {
	return &Handler{authorizer: authorizer, reader: reader}
}

func (h *Handler) ListEmissions(ctx context.Context, request *connect.Request[ledgerpb.ListEmissionsRequest]) (*connect.Response[ledgerpb.ListEmissionsResponse], error) {
	if h == nil || h.authorizer == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, operatorauth.ErrUnavailable)
	}
	if _, err := h.authorizer.Authorize(ctx, request.Header()); err != nil {
		switch {
		case errors.Is(err, operatorauth.ErrRequired):
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		case errors.Is(err, operatorauth.ErrDenied):
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		default:
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
	}
	if h.reader == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("ledger emission readback unavailable"))
	}
	values, err := h.reader.List(ctx, request.Msg.GetExternalId(), request.Msg.GetStatus(), int(request.Msg.GetLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	out := make([]*ledgerpb.LedgerEmission, 0, len(values))
	for _, value := range values {
		out = append(out, &ledgerpb.LedgerEmission{
			Id: value.ID, ChargeId: value.SettlementID, SettlementId: value.SettlementID,
			ExternalId: value.ExternalID, AdapterId: value.AdapterID, AccountId: value.AccountID,
			BookId: value.BookID, AmountMinor: value.AmountMinor, Currency: value.Currency,
			Basis: value.Basis, Status: string(value.Status), Attempts: int32(value.Attempts), LastError: value.LastError,
			OccurredAt: timestamppb.New(value.OccurredAt), FetchedAt: timestamppb.New(value.FetchedAt),
			CreatedAt: timestamppb.New(value.CreatedAt), AcceptedAt: optionalTimestamp(value.AcceptedAt),
		})
	}
	return connect.NewResponse(&ledgerpb.ListEmissionsResponse{Emissions: out}), nil
}

func optionalTimestamp(value time.Time) *timestamppb.Timestamp {
	if value.IsZero() {
		return nil
	}
	return timestamppb.New(value)
}

var _ interface {
	ListEmissions(context.Context, *connect.Request[ledgerpb.ListEmissionsRequest]) (*connect.Response[ledgerpb.ListEmissionsResponse], error)
} = (*Handler)(nil)
