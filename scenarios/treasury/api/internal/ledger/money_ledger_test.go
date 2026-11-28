package ledger_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	ingestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest/ingest_v1connect"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"

	"treasury/internal/ledger"
)

type ingestContract struct {
	ingestconnect.UnimplementedIngestServiceHandler
	adapter *ingestpb.Adapter
	event   *sharedpb.MoneyEvent
	// authorization is what each call presented, in order.
	authorization []string
}

func (s *ingestContract) RegisterAdapter(_ context.Context, request *connect.Request[ingestpb.RegisterAdapterRequest]) (*connect.Response[ingestpb.RegisterAdapterResponse], error) {
	s.authorization = append(s.authorization, request.Header().Get("Authorization"))
	s.adapter = request.Msg.Adapter
	return connect.NewResponse(&ingestpb.RegisterAdapterResponse{Adapter: request.Msg.Adapter}), nil
}

func (s *ingestContract) IngestEvent(_ context.Context, request *connect.Request[ingestpb.IngestEventRequest]) (*connect.Response[ingestpb.IngestEventResponse], error) {
	s.authorization = append(s.authorization, request.Header().Get("Authorization"))
	s.event = request.Msg.Event
	return connect.NewResponse(&ingestpb.IngestEventResponse{Posting: &sharedpb.Posting{Id: "posting-1", Event: request.Msg.Event}, Duplicate: true, Receipt: &ingestpb.Receipt{Id: "receipt-1", Status: "succeeded"}}), nil
}

type staticToken string

func (t staticToken) Token(context.Context) (string, error) { return string(t), nil }

type failingToken struct{}

func (failingToken) Token(context.Context) (string, error) {
	return "", errors.New("credential authority down")
}

type fixedDestination struct {
	value ledger.Destination
	err   error
}

func (f fixedDestination) Destination(context.Context) (ledger.Destination, error) {
	return f.value, f.err
}

var usdDestination = ledger.Destination{BookID: "book-1", AccountID: "expense-1", Currency: "USD", Source: ledger.DestinationSourceOperator}

// [REQ:TRS-P0-008] Treasury speaks Money Ledger's generated inbound contract,
// preserves operator basis, posts to the operator's chosen destination as its own
// service principal, and treats a duplicate acceptance as success.
func TestMoneyLedgerEmitterContract(t *testing.T) {
	contract := &ingestContract{}
	_, handler := ingestconnect.NewIngestServiceHandler(contract)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	emitter, err := ledger.NewMoneyLedgerEmitter(server.URL, fixedDestination{value: usdDestination}, staticToken("treasury-bearer"), server.Client())
	require.NoError(t, err)
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	duplicate, err := emitter.Emit(context.Background(), ledger.Emission{ID: "settlement-1:ledger", ExternalID: "treasury:settlement-1", AmountMinor: -250, Currency: "USD", Basis: "operator_asserted", OccurredAt: now, FetchedAt: now, Description: "Treasury settlement settlement-1"})
	require.NoError(t, err)
	require.True(t, duplicate)
	require.Equal(t, "treasury", contract.adapter.Id)
	require.Equal(t, ingestpb.AdapterKind_ADAPTER_KIND_AGGREGATOR, contract.adapter.Kind)
	require.Equal(t, "treasury:settlement-1", contract.event.ExternalId)
	require.Equal(t, "book-1", contract.event.BookId)
	require.Equal(t, "expense-1", contract.event.AccountId)
	require.EqualValues(t, -250, contract.event.AmountMinor)
	require.Equal(t, sharedpb.Basis_BASIS_OPERATOR_ASSERTED, contract.event.Basis)
	require.Equal(t, []string{"Bearer treasury-bearer", "Bearer treasury-bearer"}, contract.authorization, "both journal writes present Treasury's money-ledger bearer")
	accepted, err := emitter.EmitWithReceipt(context.Background(), ledger.Emission{ID: "settlement-2:ledger", ExternalID: "treasury:settlement-2", AmountMinor: -100, Currency: "USD", Basis: "authoritative", OccurredAt: now, FetchedAt: now})
	require.NoError(t, err)
	require.Equal(t, ledger.Acceptance{Duplicate: true, PostingID: "posting-1", ReceiptID: "receipt-1"}, accepted)
}

// Until the operator chooses a destination, nothing is sent: each delivery
// fails with the reason and the settlement stays queued. The destination is read
// on every delivery, so choosing it later needs no restart.
func TestMoneyLedgerEmitterWaitsForTheOperatorsDestination(t *testing.T) {
	contract := &ingestContract{}
	_, handler := ingestconnect.NewIngestServiceHandler(contract)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	var current fixedDestination = fixedDestination{err: ledger.ErrDestinationUnset}
	source := destinationFunc(func(ctx context.Context) (ledger.Destination, error) { return current.Destination(ctx) })
	emitter, err := ledger.NewMoneyLedgerEmitter(server.URL, source, staticToken("bearer"), server.Client())
	require.NoError(t, err)
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	emission := ledger.Emission{ID: "s:ledger", ExternalID: "treasury:s", AmountMinor: -100, Currency: "USD", Basis: "authoritative", OccurredAt: now, FetchedAt: now}
	_, err = emitter.Emit(context.Background(), emission)
	require.ErrorIs(t, err, ledger.ErrDestinationUnset)
	require.Empty(t, contract.authorization, "nothing is sent before a destination is chosen")
	current = fixedDestination{value: usdDestination}
	_, err = emitter.Emit(context.Background(), emission)
	require.NoError(t, err)
	require.Equal(t, "book-1", contract.event.BookId)
}

type destinationFunc func(context.Context) (ledger.Destination, error)

func (f destinationFunc) Destination(ctx context.Context) (ledger.Destination, error) { return f(ctx) }

// A settlement in another currency than the destination book is refused rather
// than posted as mismatched money, and without Treasury's bearer nothing is sent.
func TestMoneyLedgerEmitterRefusesWhatItCannotDeliverSafely(t *testing.T) {
	contract := &ingestContract{}
	_, handler := ingestconnect.NewIngestServiceHandler(contract)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	eur := ledger.Emission{ID: "s:ledger", ExternalID: "treasury:s", AmountMinor: -100, Currency: "EUR", OccurredAt: now, FetchedAt: now}
	emitter, err := ledger.NewMoneyLedgerEmitter(server.URL, fixedDestination{value: usdDestination}, staticToken("bearer"), server.Client())
	require.NoError(t, err)
	_, err = emitter.Emit(context.Background(), eur)
	require.ErrorContains(t, err, "does not match the destination book currency USD")
	noIdentity, err := ledger.NewMoneyLedgerEmitter(server.URL, fixedDestination{value: usdDestination}, failingToken{}, server.Client())
	require.NoError(t, err)
	eur.Currency = "USD"
	_, err = noIdentity.Emit(context.Background(), eur)
	require.ErrorContains(t, err, "service identity is unavailable")
	require.NotContains(t, err.Error(), "credential authority down", "the credential source's reason is not leaked")
	require.Empty(t, contract.authorization)
}

// The origin is resolved by service discovery, so it is missing only when Money
// Ledger is not running; the destination and identity are required collaborators.
func TestNewMoneyLedgerEmitterNamesWhatIsMissing(t *testing.T) {
	_, err := ledger.NewMoneyLedgerEmitter("  ", fixedDestination{}, staticToken("t"), nil)
	require.ErrorContains(t, err, "origin could not be resolved")
	require.ErrorContains(t, err, "money-ledger running")
	_, err = ledger.NewMoneyLedgerEmitter("http://127.0.0.1:1", nil, staticToken("t"), nil)
	require.ErrorContains(t, err, "destination")
	_, err = ledger.NewMoneyLedgerEmitter("http://127.0.0.1:1", fixedDestination{}, nil, nil)
	require.ErrorContains(t, err, "service identity")
}
