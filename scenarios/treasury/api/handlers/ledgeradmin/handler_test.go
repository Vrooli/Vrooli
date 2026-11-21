package ledgeradmin

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/ledger"
	"treasury/internal/ledger"
	"treasury/internal/operatorauth"
)

type testAuthorizer struct{ err error }

func (a testAuthorizer) Authorize(context.Context, http.Header) (operatorauth.Identity, error) {
	return operatorauth.Identity{Subject: "operator"}, a.err
}

type testReader struct {
	externalID, status string
	limit              int
	values             []ledger.Emission
	queued             int
}

func (r *testReader) QueuedCount(context.Context) (int, error) { return r.queued, nil }

func (r *testReader) List(_ context.Context, externalID, status string, limit int) ([]ledger.Emission, error) {
	r.externalID, r.status, r.limit = externalID, status, limit
	return r.values, nil
}

func TestListEmissionsRequiresOperatorIdentity(t *testing.T) {
	handler := NewHandler(testAuthorizer{err: operatorauth.ErrRequired}, &testReader{}, Destinations{})
	_, err := handler.ListEmissions(context.Background(), connect.NewRequest(&ledgerpb.ListEmissionsRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestListEmissionsReturnsOccurrenceDeliveryEvidence(t *testing.T) {
	created := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	accepted := created.Add(time.Second)
	reader := &testReader{values: []ledger.Emission{{
		ID: "occurrence:ledger", SettlementID: "occurrence", ExternalID: "base:0xabc:4", AdapterID: "treasury-base",
		AccountID: "account", BookID: "book", AmountMinor: 1250, Currency: "USD", Basis: "authoritative",
		Status: ledger.StatusAccepted, Attempts: 2, OccurredAt: created, FetchedAt: created,
		CreatedAt: created, AcceptedAt: accepted,
	}}}
	request := connect.NewRequest(&ledgerpb.ListEmissionsRequest{ExternalId: "base:0xabc:4", Status: "accepted", Limit: 25})
	response, err := NewHandler(testAuthorizer{}, reader, Destinations{}).ListEmissions(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, "base:0xabc:4", reader.externalID)
	require.Equal(t, "accepted", reader.status)
	require.Equal(t, 25, reader.limit)
	require.Len(t, response.Msg.Emissions, 1)
	emission := response.Msg.Emissions[0]
	require.Equal(t, "occurrence:ledger", emission.Id)
	require.Equal(t, "base:0xabc:4", emission.ExternalId)
	require.Equal(t, "accepted", emission.Status)
	require.Equal(t, int32(2), emission.Attempts)
	require.Equal(t, "account", emission.AccountId)
	require.Equal(t, "book", emission.BookId)
	require.EqualValues(t, 1250, emission.AmountMinor)
}

func TestListEmissionsDoesNotMaskAuthorizationOutage(t *testing.T) {
	handler := NewHandler(testAuthorizer{err: errors.Join(operatorauth.ErrUnavailable, errors.New("token provider down"))}, &testReader{}, Destinations{})
	_, err := handler.ListEmissions(context.Background(), connect.NewRequest(&ledgerpb.ListEmissionsRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

type memoryDestination struct {
	value ledger.Destination
	set   bool
}

func (m *memoryDestination) Destination(context.Context) (ledger.Destination, error) {
	if !m.set {
		return ledger.Destination{}, ledger.ErrDestinationUnset
	}
	return m.value, nil
}

func (m *memoryDestination) Set(_ context.Context, d ledger.Destination) error {
	m.value, m.set = d, true
	return nil
}

type stubValidator struct {
	result ledger.Destination
	err    error
}

func (v stubValidator) Validate(context.Context, string, string) (ledger.Destination, error) {
	return v.result, v.err
}

var fixedNow = time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)

func destinationHandler(authErr error, memory *memoryDestination, validator ledger.DestinationValidator, queued int) *Handler {
	return NewHandler(testAuthorizer{err: authErr}, &testReader{queued: queued}, Destinations{Current: memory, Store: memory, Validator: validator, Now: func() time.Time { return fixedNow }})
}

// The destination is the operator's: no identity, no read and no change.
func TestDestinationRPCsRequireTheOperatorRealm(t *testing.T) {
	memory := &memoryDestination{}
	handler := destinationHandler(operatorauth.ErrDenied, memory, stubValidator{}, 0)
	_, err := handler.SetLedgerDestination(context.Background(), connect.NewRequest(&ledgerpb.SetLedgerDestinationRequest{BookId: "b", AccountId: "a"}))
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	_, err = handler.GetLedgerDestination(context.Background(), connect.NewRequest(&ledgerpb.GetLedgerDestinationRequest{}))
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	require.False(t, memory.set, "a refused caller changes nothing")
}

// Choosing is checked against Money Ledger first: a refusal is the operator's
// to fix (invalid argument) and stores nothing; an unreachable Money Ledger is an
// outage (unavailable) and stores nothing; no way to check refuses outright.
func TestSetLedgerDestinationStoresOnlyAChoiceMoneyLedgerAccepted(t *testing.T) {
	ctx := context.Background()
	request := func() *connect.Request[ledgerpb.SetLedgerDestinationRequest] {
		return connect.NewRequest(&ledgerpb.SetLedgerDestinationRequest{BookId: "b", AccountId: "a"})
	}
	for _, test := range []struct {
		name      string
		validator ledger.DestinationValidator
		code      connect.Code
	}{
		{"refused by Money Ledger", stubValidator{err: ledger.DestinationRefusal{Reason: "book is archived"}}, connect.CodeInvalidArgument},
		{"Money Ledger unreachable", stubValidator{err: errors.New("connection refused")}, connect.CodeUnavailable},
		{"no way to check", nil, connect.CodeFailedPrecondition},
	} {
		t.Run(test.name, func(t *testing.T) {
			memory := &memoryDestination{}
			_, err := destinationHandler(nil, memory, test.validator, 0).SetLedgerDestination(ctx, request())
			require.Equal(t, test.code, connect.CodeOf(err))
			require.False(t, memory.set, "an unchecked choice is never stored")
		})
	}
	t.Run("refusal reason reaches the operator", func(t *testing.T) {
		_, err := destinationHandler(nil, &memoryDestination{}, stubValidator{err: ledger.DestinationRefusal{Reason: "book is archived"}}, 0).SetLedgerDestination(ctx, request())
		require.ErrorContains(t, err, "book is archived")
	})
	t.Run("accepted", func(t *testing.T) {
		memory := &memoryDestination{}
		checked := ledger.Destination{BookID: "b", AccountID: "a", BookName: "Book", AccountName: "Cash", Currency: "USD"}
		response, err := destinationHandler(nil, memory, stubValidator{result: checked}, 0).SetLedgerDestination(ctx, request())
		require.NoError(t, err)
		d := response.Msg.Destination
		require.True(t, d.Configured)
		require.Equal(t, []string{"b", "a", "Book", "Cash", "USD", "operator"}, []string{d.BookId, d.AccountId, d.BookName, d.AccountName, d.Currency, d.Source})
		require.Equal(t, fixedNow, d.SetAt.AsTime())
		require.Equal(t, ledger.DestinationSourceOperator, memory.value.Source)
	})
}

// An unconfigured destination is stated in plain words next to the queue, so an
// empty or stuck queue is never the only signal that nothing is delivered.
func TestStatusAndListReportTheUnconfiguredDestination(t *testing.T) {
	ctx := context.Background()
	handler := destinationHandler(nil, &memoryDestination{}, nil, 3)
	got, err := handler.GetLedgerDestination(ctx, connect.NewRequest(&ledgerpb.GetLedgerDestinationRequest{}))
	require.NoError(t, err)
	require.False(t, got.Msg.Destination.Configured)
	require.Contains(t, got.Msg.Destination.UnconfiguredReason, "treasury ledger destination-set")
	require.EqualValues(t, 3, got.Msg.QueuedTotal)
	listed, err := handler.ListEmissions(ctx, connect.NewRequest(&ledgerpb.ListEmissionsRequest{Status: "accepted"}))
	require.NoError(t, err)
	require.False(t, listed.Msg.Destination.Configured)
	require.EqualValues(t, 3, listed.Msg.QueuedTotal, "the waiting total ignores the list filter")
}

func TestStatusReportsAConfiguredDestination(t *testing.T) {
	memory := &memoryDestination{set: true, value: ledger.Destination{BookID: "b", AccountID: "a", BookName: "Book", AccountName: "Cash", Currency: "USD", Source: ledger.DestinationSourceEnvironment}}
	got, err := destinationHandler(nil, memory, nil, 0).GetLedgerDestination(context.Background(), connect.NewRequest(&ledgerpb.GetLedgerDestinationRequest{}))
	require.NoError(t, err)
	d := got.Msg.Destination
	require.True(t, d.Configured)
	require.Empty(t, d.UnconfiguredReason)
	require.Equal(t, "environment", d.Source)
	require.Equal(t, "Cash", d.AccountName)
}
