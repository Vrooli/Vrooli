package ledger_test

import (
	"context"
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
}

func (s *ingestContract) RegisterAdapter(_ context.Context, request *connect.Request[ingestpb.RegisterAdapterRequest]) (*connect.Response[ingestpb.RegisterAdapterResponse], error) {
	s.adapter = request.Msg.Adapter
	return connect.NewResponse(&ingestpb.RegisterAdapterResponse{Adapter: request.Msg.Adapter}), nil
}

func (s *ingestContract) IngestEvent(_ context.Context, request *connect.Request[ingestpb.IngestEventRequest]) (*connect.Response[ingestpb.IngestEventResponse], error) {
	s.event = request.Msg.Event
	return connect.NewResponse(&ingestpb.IngestEventResponse{Posting: &sharedpb.Posting{Id: "posting-1", Event: request.Msg.Event}, Duplicate: true, Receipt: &ingestpb.Receipt{Id: "receipt-1", Status: "succeeded"}}), nil
}

// [REQ:TRS-P0-008] Treasury speaks Money Ledger's generated inbound contract,
// preserves operator basis, uses configured routing, and treats a duplicate
// acceptance as success.
func TestMoneyLedgerEmitterContract(t *testing.T) {
	contract := &ingestContract{}
	_, handler := ingestconnect.NewIngestServiceHandler(contract)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	emitter, err := ledger.NewMoneyLedgerEmitter(server.URL, "book-1", "expense-1", server.Client())
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
}
