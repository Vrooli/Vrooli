package reconcile

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/askreader"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	ingestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest/ingest_v1connect"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger/ledger_v1connect"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/proto"
)

// contractLedger exercises the external generated Money Ledger Connect
// contract without importing or launching another scenario's source. It models
// only the idempotency and reversal semantics this producer requires; Money
// Ledger's policy, SQLite persistence, and handler behavior are covered by its
// own integration tests.
type contractLedger struct {
	ingestconnect.UnimplementedIngestServiceHandler
	ledgerconnect.UnimplementedJournalServiceHandler
	mu               sync.Mutex
	postsByExternal  map[string]*sharedpb.Posting
	eventsByExternal map[string]*sharedpb.MoneyEvent
	reversalsByID    map[string]*sharedpb.Posting
	count            int
}

func newContractLedger() *contractLedger {
	return &contractLedger{postsByExternal: make(map[string]*sharedpb.Posting), eventsByExternal: make(map[string]*sharedpb.MoneyEvent), reversalsByID: make(map[string]*sharedpb.Posting)}
}

func (l *contractLedger) IngestEvent(_ context.Context, req *connect.Request[ingestpb.IngestEventRequest]) (*connect.Response[ingestpb.IngestEventResponse], error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	event := req.Msg.GetEvent()
	if req.Header().Get("Authorization") != "Bearer fixture-scoped-token" || event.GetAdapterId() != InvoiceAdapterID || event.GetBookId() != "fixture-book" || event.GetAccountId() != "fixture-account" || event.GetExternalId() == "" || event.GetCurrency() != "USD" || event.GetBasis() != sharedpb.Basis_BASIS_AUTHORITATIVE || event.GetOccurredAt() == nil || event.GetFetchedAt() == nil {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("fixture Connect contract rejected identity or invoice target"))
	}
	if posting := l.postsByExternal[event.GetExternalId()]; posting != nil {
		if !proto.Equal(l.eventsByExternal[event.GetExternalId()], event) {
			return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("external invoice identity was reused with different facts"))
		}
		return connect.NewResponse(&ingestpb.IngestEventResponse{Posting: posting}), nil
	}
	l.count++
	posting := &sharedpb.Posting{Id: fmt.Sprintf("fixture-posting-%d", l.count)}
	l.postsByExternal[event.GetExternalId()] = posting
	l.eventsByExternal[event.GetExternalId()] = proto.Clone(event).(*sharedpb.MoneyEvent)
	return connect.NewResponse(&ingestpb.IngestEventResponse{Posting: posting}), nil
}

func (l *contractLedger) ReversePosting(_ context.Context, req *connect.Request[ledgerpb.ReversePostingRequest]) (*connect.Response[ledgerpb.ReversePostingResponse], error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if req.Header().Get("Authorization") != "Bearer fixture-scoped-token" {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("fixture Connect contract rejected identity"))
	}
	if reversal := l.reversalsByID[req.Msg.GetPostingId()]; reversal != nil {
		return connect.NewResponse(&ledgerpb.ReversePostingResponse{Posting: reversal}), nil
	}
	if _, ok := l.postsByExternalPosting(req.Msg.GetPostingId()); !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("original posting not found"))
	}
	l.count++
	reversal := &sharedpb.Posting{Id: fmt.Sprintf("fixture-posting-%d", l.count)}
	l.reversalsByID[req.Msg.GetPostingId()] = reversal
	return connect.NewResponse(&ledgerpb.ReversePostingResponse{Posting: reversal}), nil
}

func (l *contractLedger) postsByExternalPosting(postingID string) (*sharedpb.Posting, bool) {
	for _, posting := range l.postsByExternal {
		if posting.GetId() == postingID {
			return posting, true
		}
	}
	return nil, false
}

func (l *contractLedger) PostingCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.count
}

// dropOnceTransport delivers a generated Connect request, lets the contract
// fixture commit, then drops the first selected response as a network loss.
type dropOnceTransport struct {
	base    http.RoundTripper
	mu      sync.Mutex
	dropped map[string]bool
}

func (t *dropOnceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	t.mu.Lock()
	shouldDrop := !t.dropped[req.URL.Path] && (req.URL.Path == "/vrooli.money_ledger.v1.ingest.IngestService/IngestEvent" || req.URL.Path == "/vrooli.money_ledger.v1.ledger.JournalService/ReversePosting")
	if shouldDrop {
		t.dropped[req.URL.Path] = true
	}
	t.mu.Unlock()
	if shouldDrop {
		_, _ = io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
		return nil, errors.New("fixture network lost response after Money Ledger committed request")
	}
	return response, nil
}

type integrationDecisionPublisher struct{ events []eventbus.DomainEvent }

func (p *integrationDecisionPublisher) PublishDomainEvent(_ context.Context, event eventbus.DomainEvent) error {
	p.events = append(p.events, event)
	return nil
}

// This test runs the producer's durable receipt and verified-human decision
// flow through Money Ledger's generated Connect contract. Money Ledger's own
// suite separately exercises its production handlers, policy gate and SQLite
// journal; this producer-contract test never imports that scenario module.
func TestFixtureProducerPostsIntoMoneyLedgerExactlyOnceAcrossLostResponsesAndCorrection(t *testing.T) {
	ctx := context.Background()
	ledger := newContractLedger()
	mux := http.NewServeMux()
	path, handler := ingestconnect.NewIngestServiceHandler(ledger)
	mux.Handle(path, handler)
	path, handler = ledgerconnect.NewJournalServiceHandler(ledger)
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	transport := &dropOnceTransport{base: http.DefaultTransport, dropped: make(map[string]bool)}
	httpClient := &http.Client{Transport: transport}
	destination := InvoiceDestination{BookID: "fixture-book", AccountID: "fixture-account"}
	client, err := NewConnectInvoiceLedger(server.URL, destination.BookID, destination.AccountID, httpClient, func(context.Context) (string, error) { return "fixture-scoped-token", nil })
	require.NoError(t, err)
	db := invoiceFixtureDB(t) // approved reservation/quote/admission values are synthetic fixtures.
	now := time.Date(2026, 11, 24, 1, 0, 0, 0, time.UTC)
	client.Now = func() time.Time { return now }
	reconciler := InvoiceReconciler{DB: db, Ledger: client, Destination: destination, Now: func() time.Time { return now }}
	original := InvoiceLine{Provider: "digitalocean", InvoiceID: "fixture-e2e-invoice-01", LineID: "fixture-e2e-line-01", ProviderInstanceID: "provider-vm-fixture", AmountMinor: -1250, Currency: "USD", Provenance: "FIXTURE ONLY: synthetic provider invoice; no live source call", OccurredAt: now.Add(-time.Hour)}
	originalID, err := reconciler.RecordFixtureOrSource(ctx, original)
	require.NoError(t, err)
	if _, err := reconciler.ConfirmAndPost(ctx, originalID); !errors.Is(err, ErrInvoiceConfirmation) {
		t.Fatalf("post before phone receipt err=%v", err)
	}
	require.Zero(t, ledger.PostingCount(), "no ledger effect before verified receipt")
	approveIntegrationFixture(t, ctx, reconciler, originalID, "fixture-e2e-ask-original")
	if _, err := reconciler.ConfirmAndPost(ctx, originalID); err == nil {
		t.Fatal("expected simulated lost IngestEvent response")
	}
	require.Equal(t, 1, ledger.PostingCount(), "first ingest committed before its response was dropped")
	originalPosting, err := reconciler.ConfirmAndPost(ctx, originalID)
	require.NoError(t, err)
	require.NotEmpty(t, originalPosting)
	require.Equal(t, 1, ledger.PostingCount(), "lost-response retry must resolve the original posting")

	corrected := InvoiceLine{Provider: "digitalocean", InvoiceID: "fixture-e2e-invoice-02", LineID: "fixture-e2e-adjustment-01", ProviderInstanceID: "provider-vm-fixture", AmountMinor: -1100, Currency: "USD", Provenance: "FIXTURE ONLY: synthetic corrected invoice line", OccurredAt: now}
	correctionID, err := reconciler.RecordCorrection(ctx, originalID, corrected)
	require.NoError(t, err)
	approveIntegrationFixture(t, ctx, reconciler, correctionID, "fixture-e2e-ask-correction")
	if _, err := reconciler.ConfirmCorrectionAndPost(ctx, correctionID); err == nil {
		t.Fatal("expected simulated lost ReversePosting response")
	}
	require.Equal(t, 2, ledger.PostingCount(), "reversal committed exactly once before its response was dropped")
	correctedPosting, err := reconciler.ConfirmCorrectionAndPost(ctx, correctionID)
	require.NoError(t, err)
	require.NotEmpty(t, correctedPosting)
	require.Equal(t, 3, ledger.PostingCount(), "original, one linked reversal, and corrected line")
	var state, reversalOf, reversalPosting, storedOriginal string
	require.NoError(t, db.QueryRow(`SELECT state,reverses_external_id,reversal_posting_id,ledger_posting_id FROM invoice_reconciliation_receipts WHERE external_id=?`, correctionID).Scan(&state, &reversalOf, &reversalPosting, &storedOriginal))
	require.Equal(t, "posted", state)
	require.Equal(t, originalID, reversalOf)
	require.NotEmpty(t, reversalPosting)
	require.Equal(t, correctedPosting, storedOriginal)
	var originalState string
	require.NoError(t, db.QueryRow(`SELECT state FROM invoice_reconciliation_receipts WHERE external_id=?`, originalID).Scan(&originalState))
	require.Equal(t, "reversed", originalState)
}

func approveIntegrationFixture(t *testing.T, ctx context.Context, reconciler InvoiceReconciler, externalID, askID string) {
	t.Helper()
	publisher := &integrationDecisionPublisher{}
	eventID, err := reconciler.RequestInvoiceApproval(ctx, externalID, publisher)
	require.NoError(t, err)
	require.Len(t, publisher.events, 1)
	event := publisher.events[0]
	correlationRaw, ok := event.Payload["correlation"].(map[string]any)
	require.True(t, ok)
	correlation := make(map[string]string, len(correlationRaw))
	for key, value := range correlationRaw {
		correlation[key] = value.(string)
	}
	task := askreader.Ask{ID: askID, State: "answered", Answer: "approve", AnsweredBy: "fixture-verified-human", AnsweredByIdentityKind: "verified_human", AnsweredByIdentitySource: "fixture-human-identity-provider", AnsweredAt: "2026-11-24T01:00:00Z", ResolvedAt: "2026-11-24T01:00:00Z", Source: "compute-manager", SourceEventType: InvoiceApprovalEventType, RequesterRef: eventID, Correlation: correlation}
	resolution := operatordecision.DecisionResolution{AskID: askID, State: operatordecision.StateAnswered, Answer: "approve", AnsweredBy: task.AnsweredBy, AnsweredByIdentityKind: task.AnsweredByIdentityKind, AnsweredByIdentitySource: task.AnsweredByIdentitySource, AnsweredAt: task.AnsweredAt, ResolvedAt: task.ResolvedAt, Source: task.Source, SourceEventType: task.SourceEventType, RequesterRef: eventID, Correlation: correlation}
	reader := fixedInvoiceAskReader{ask: task}
	handled, err := (InvoiceApprovalEffect{DB: reconciler.DB, Reader: reader, Now: func() time.Time { return time.Date(2026, 11, 24, 1, 0, 0, 0, time.UTC) }}).Apply(ctx, resolution)
	require.NoError(t, err)
	require.True(t, handled)
}
