package reconcile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	ingestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest/ingest_v1connect"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger/ledger_v1connect"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
)

type invoiceIngestServer struct {
	ingestconnect.UnimplementedIngestServiceHandler
	t     *testing.T
	token string
	event *sharedpb.MoneyEvent
}

func (s *invoiceIngestServer) IngestEvent(_ context.Context, req *connect.Request[ingestpb.IngestEventRequest]) (*connect.Response[ingestpb.IngestEventResponse], error) {
	s.t.Helper()
	require.Equal(s.t, "Bearer fixture-ledger-token", req.Header().Get("Authorization"))
	s.token = req.Header().Get("Authorization")
	s.event = req.Msg.GetEvent()
	return connect.NewResponse(&ingestpb.IngestEventResponse{Posting: &sharedpb.Posting{Id: "fixture-ledger-posting"}}), nil
}

type invoiceJournalServer struct {
	ledgerconnect.UnimplementedJournalServiceHandler
	t     *testing.T
	token string
	req   *ledgerpb.ReversePostingRequest
}

func (s *invoiceJournalServer) ReversePosting(_ context.Context, req *connect.Request[ledgerpb.ReversePostingRequest]) (*connect.Response[ledgerpb.ReversePostingResponse], error) {
	s.t.Helper()
	require.Equal(s.t, "Bearer fixture-ledger-token", req.Header().Get("Authorization"))
	s.token = req.Header().Get("Authorization")
	s.req = req.Msg
	return connect.NewResponse(&ledgerpb.ReversePostingResponse{Posting: &sharedpb.Posting{Id: "fixture-ledger-reversal"}}), nil
}

func TestGeneratedConnectInvoiceAdapterSendsTypedEventAndReversal(t *testing.T) {
	ctx := context.Background()
	ingest := &invoiceIngestServer{t: t}
	journal := &invoiceJournalServer{t: t}
	mux := http.NewServeMux()
	p, h := ingestconnect.NewIngestServiceHandler(ingest)
	mux.Handle(p, h)
	p, h = ledgerconnect.NewJournalServiceHandler(journal)
	mux.Handle(p, h)
	server := httptest.NewServer(mux)
	defer server.Close()
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	client, err := NewConnectInvoiceLedger(server.URL, "book-fixture", "account-fixture", server.Client(), func(context.Context) (string, error) { return "fixture-ledger-token", nil })
	require.NoError(t, err)
	client.Now = func() time.Time { return now }
	line := InvoiceLine{Provider: "digitalocean", InvoiceID: "fixture-invoice", LineID: "fixture-line", ProviderInstanceID: "fixture-vm", AmountMinor: -1250, Currency: "usd", Provenance: "FIXTURE ONLY", OccurredAt: now.Add(-time.Hour)}
	postingID, err := client.PostInvoice(ctx, InvoicePosting{Line: line, ExternalID: line.ExternalID(), Destination: InvoiceDestination{BookID: "book-fixture", AccountID: "account-fixture"}})
	require.NoError(t, err)
	require.Equal(t, "fixture-ledger-posting", postingID)
	require.Equal(t, "Bearer fixture-ledger-token", ingest.token)
	require.Equal(t, InvoiceAdapterID, ingest.event.GetAdapterId())
	require.Equal(t, line.ExternalID(), ingest.event.GetExternalId())
	require.Equal(t, "book-fixture", ingest.event.GetBookId())
	require.Equal(t, "account-fixture", ingest.event.GetAccountId())
	require.EqualValues(t, -1250, ingest.event.GetAmountMinor())
	require.Equal(t, "USD", ingest.event.GetCurrency())
	require.Equal(t, sharedpb.Basis_BASIS_AUTHORITATIVE, ingest.event.GetBasis())
	require.True(t, ingest.event.GetOccurredAt().AsTime().Equal(line.OccurredAt))
	require.True(t, ingest.event.GetFetchedAt().AsTime().Equal(now))
	reversal, err := client.ReverseInvoice(ctx, postingID, "FIXTURE ONLY correction")
	require.NoError(t, err)
	require.Equal(t, "fixture-ledger-reversal", reversal)
	require.Equal(t, postingID, journal.req.GetPostingId())
	require.Equal(t, "FIXTURE ONLY correction", journal.req.GetReason())
	require.Equal(t, "Bearer fixture-ledger-token", journal.token)
	_, err = client.PostInvoice(ctx, InvoicePosting{Line: line, ExternalID: line.ExternalID(), Destination: InvoiceDestination{BookID: "book-fixture", AccountID: "other-account"}})
	require.ErrorContains(t, err, "does not match the configured Money Ledger target")
}

func TestGeneratedConnectInvoiceAdapterFailsClosedWithoutCredential(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	client, err := NewConnectInvoiceLedger(server.URL, "book", "account", server.Client(), func(context.Context) (string, error) { return "", nil })
	require.NoError(t, err)
	_, err = client.PostInvoice(context.Background(), InvoicePosting{Line: InvoiceLine{Provider: "digitalocean", InvoiceID: "f", LineID: "l", AmountMinor: -1, Currency: "USD", OccurredAt: time.Now()}, ExternalID: "external", Destination: InvoiceDestination{BookID: "book", AccountID: "account"}})
	require.ErrorContains(t, err, "credential is unavailable")
}
