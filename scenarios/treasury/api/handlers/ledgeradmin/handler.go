package ledgeradmin

import (
	"context"
	"errors"
	"net/http"
	"time"

	"connectrpc.com/connect"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/ledger/ledger_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"treasury/internal/ledger"
	"treasury/internal/operatorauth"
)

type EmissionReader interface {
	List(context.Context, string, string, int) ([]ledger.Emission, error)
	QueuedCount(context.Context) (int, error)
}

// Destinations is everything the operator's destination choice needs: the
// current choice (stored, else the rehearsal seed), the store that records a
// new one, and the check against Money Ledger. Validator is nil when the Money
// Ledger origin could not be resolved, which refuses a change rather than
// storing an unchecked one.
type Destinations struct {
	Current   ledger.DestinationSource
	Store     DestinationWriter
	Validator ledger.DestinationValidator
	Now       func() time.Time
}

type DestinationWriter interface {
	Set(context.Context, ledger.Destination) error
}

type Handler struct {
	authorizer   operatorauth.Authorizer
	reader       EmissionReader
	destinations Destinations
}

func NewHandler(authorizer operatorauth.Authorizer, reader EmissionReader, destinations Destinations) *Handler {
	if destinations.Now == nil {
		destinations.Now = time.Now
	}
	return &Handler{authorizer: authorizer, reader: reader, destinations: destinations}
}

func (h *Handler) authorize(ctx context.Context, header http.Header) error {
	if h == nil || h.authorizer == nil {
		return connect.NewError(connect.CodeFailedPrecondition, operatorauth.ErrUnavailable)
	}
	if _, err := h.authorizer.Authorize(ctx, header); err != nil {
		switch {
		case errors.Is(err, operatorauth.ErrRequired):
			return connect.NewError(connect.CodeUnauthenticated, err)
		case errors.Is(err, operatorauth.ErrDenied):
			return connect.NewError(connect.CodePermissionDenied, err)
		default:
			return connect.NewError(connect.CodeFailedPrecondition, err)
		}
	}
	return nil
}

// destinationState reports where settlements go, or why they cannot go anywhere.
func (h *Handler) destinationState(ctx context.Context) (*ledgerpb.LedgerDestination, error) {
	if h.destinations.Current == nil {
		return &ledgerpb.LedgerDestination{UnconfiguredReason: "the settlement destination is unavailable"}, nil
	}
	d, err := h.destinations.Current.Destination(ctx)
	switch {
	case errors.Is(err, ledger.ErrDestinationUnset):
		return &ledgerpb.LedgerDestination{UnconfiguredReason: err.Error()}, nil
	case err != nil:
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return destinationMessage(d), nil
}

func destinationMessage(d ledger.Destination) *ledgerpb.LedgerDestination {
	return &ledgerpb.LedgerDestination{
		Configured: true, BookId: d.BookID, AccountId: d.AccountID, BookName: d.BookName,
		AccountName: d.AccountName, Currency: d.Currency, Source: d.Source, SetAt: optionalTimestamp(d.SetAt),
	}
}

func (h *Handler) GetLedgerDestination(ctx context.Context, request *connect.Request[ledgerpb.GetLedgerDestinationRequest]) (*connect.Response[ledgerpb.GetLedgerDestinationResponse], error) {
	if err := h.authorize(ctx, request.Header()); err != nil {
		return nil, err
	}
	state, err := h.destinationState(ctx)
	if err != nil {
		return nil, err
	}
	queued, err := h.queued(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&ledgerpb.GetLedgerDestinationResponse{Destination: state, QueuedTotal: queued}), nil
}

func (h *Handler) SetLedgerDestination(ctx context.Context, request *connect.Request[ledgerpb.SetLedgerDestinationRequest]) (*connect.Response[ledgerpb.SetLedgerDestinationResponse], error) {
	if err := h.authorize(ctx, request.Header()); err != nil {
		return nil, err
	}
	if h.destinations.Validator == nil || h.destinations.Store == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("the Money Ledger origin could not be resolved (is money-ledger running?), so the choice cannot be checked; nothing was stored"))
	}
	checked, err := h.destinations.Validator.Validate(ctx, request.Msg.GetBookId(), request.Msg.GetAccountId())
	if err != nil {
		var refusal ledger.DestinationRefusal
		if errors.As(err, &refusal) {
			return nil, connect.NewError(connect.CodeInvalidArgument, refusal)
		}
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	checked.Source, checked.SetAt = ledger.DestinationSourceOperator, h.destinations.Now().UTC()
	if err := h.destinations.Store.Set(ctx, checked); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&ledgerpb.SetLedgerDestinationResponse{Destination: destinationMessage(checked)}), nil
}

func (h *Handler) queued(ctx context.Context) (int32, error) {
	if h.reader == nil {
		return 0, connect.NewError(connect.CodeFailedPrecondition, errors.New("ledger emission readback unavailable"))
	}
	n, err := h.reader.QueuedCount(ctx)
	if err != nil {
		return 0, connect.NewError(connect.CodeInternal, err)
	}
	return int32(n), nil
}

func (h *Handler) ListEmissions(ctx context.Context, request *connect.Request[ledgerpb.ListEmissionsRequest]) (*connect.Response[ledgerpb.ListEmissionsResponse], error) {
	if err := h.authorize(ctx, request.Header()); err != nil {
		return nil, err
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
	state, err := h.destinationState(ctx)
	if err != nil {
		return nil, err
	}
	queued, err := h.queued(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&ledgerpb.ListEmissionsResponse{Emissions: out, Destination: state, QueuedTotal: queued}), nil
}

func optionalTimestamp(value time.Time) *timestamppb.Timestamp {
	if value.IsZero() {
		return nil
	}
	return timestamppb.New(value)
}

var _ ledgerconnect.LedgerServiceHandler = (*Handler)(nil)
