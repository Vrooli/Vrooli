package ledger

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapptest"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	ingestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest/ingest_v1connect"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger/ledger_v1connect"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
)

type booksServiceStub struct {
	ledgerconnect.BooksServiceHandler
	request *ledgerpb.ListBooksRequest
	books   *ledgerpb.ListBooksResponse
	err     error
}

func (s *booksServiceStub) ListBooks(_ context.Context, req *connect.Request[ledgerpb.ListBooksRequest]) (*connect.Response[ledgerpb.ListBooksResponse], error) {
	s.request = req.Msg
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.books), nil
}

type journalServiceStub struct {
	ledgerconnect.JournalServiceHandler
	request       *ledgerpb.TransferRequest
	authorization string
	response      *ledgerpb.TransferResponse
}

type ingestServiceStub struct {
	ingestconnect.IngestServiceHandler
	request  *ingestpb.PreviewCryptoObservationRequest
	response *ingestpb.PreviewCryptoObservationResponse
}

func (s *journalServiceStub) Transfer(_ context.Context, req *connect.Request[ledgerpb.TransferRequest]) (*connect.Response[ledgerpb.TransferResponse], error) {
	s.request = req.Msg
	s.authorization = req.Header().Get("Authorization")
	return connect.NewResponse(s.response), nil
}

func (s *ingestServiceStub) PreviewCryptoObservation(_ context.Context, req *connect.Request[ingestpb.PreviewCryptoObservationRequest]) (*connect.Response[ingestpb.PreviewCryptoObservationResponse], error) {
	s.request = req.Msg
	return connect.NewResponse(s.response), nil
}

func TestRegisterGroupsReturnsBothManifestBoundCommandGroups(t *testing.T) {
	books := &booksServiceStub{books: &ledgerpb.ListBooksResponse{Books: []*ledgerpb.Book{{Id: "operating", Name: "Operating", Currency: "USD"}}}}
	booksPath, booksHandler := ledgerconnect.NewBooksServiceHandler(books)
	mux := http.NewServeMux()
	mux.Handle(booksPath, booksHandler)
	core := cliapptest.NewTestApp(t, mux)
	groups, err := RegisterGroups(core, cliapptest.ReadManifest(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 || groups[0].Name != "ledger" || groups[1].Name != "ingest" {
		t.Fatalf("groups = %#v, want ledger and ingest", groups)
	}
	out, err := cliapptest.RunCommand(t, cliapptest.FindCommand(t, groups, "ledger", "books-list"), core, "--include-archived", "true")
	if err != nil {
		t.Fatalf("books-list: %v\n%s", err, out)
	}
	if books.request == nil || !books.request.IncludeArchived {
		t.Fatalf("ListBooks request = %+v, want archived books included", books.request)
	}
	if !strings.Contains(out, "Found 1 book(s).") || !strings.Contains(out, "operating — Operating (USD)") {
		t.Fatalf("books-list output = %q", out)
	}
}

func TestBooksListPreservesConnectErrors(t *testing.T) {
	books := &booksServiceStub{err: connect.NewError(connect.CodeUnavailable, errors.New("ledger store is offline"))}
	booksPath, booksHandler := ledgerconnect.NewBooksServiceHandler(books)
	mux := http.NewServeMux()
	mux.Handle(booksPath, booksHandler)
	core := cliapptest.NewTestApp(t, mux)
	groups, err := RegisterGroups(core, cliapptest.ReadManifest(t))
	if err != nil {
		t.Fatal(err)
	}
	out, err := cliapptest.RunCommand(t, cliapptest.FindCommand(t, groups, "ledger", "books-list"), core)
	if err == nil || !strings.Contains(err.Error(), "unavailable") || !strings.Contains(err.Error(), "ledger store is offline") {
		t.Fatalf("books-list error = %v, want the Connect unavailable cause; output=%q", err, out)
	}
}

func TestTransferSendsOwnerAuthorizedRequestAndReportsPairedPostings(t *testing.T) {
	withOwnerSeams(t, func(context.Context) error { return nil }, func(context.Context) (string, error) { return "owner-session", nil })
	journal := &journalServiceStub{response: &ledgerpb.TransferResponse{Postings: []*sharedpb.Posting{{Id: "out"}, {Id: "in"}}}}
	journalPath, journalHandler := ledgerconnect.NewJournalServiceHandler(journal)
	mux := http.NewServeMux()
	mux.Handle(journalPath, journalHandler)
	core := cliapptest.NewTestApp(t, mux)
	groups, err := RegisterGroups(core, cliapptest.ReadManifest(t))
	if err != nil {
		t.Fatal(err)
	}
	out, err := cliapptest.RunCommand(t, cliapptest.FindCommand(t, groups, "ledger", "journal-transfer"), core,
		"--from-account-id", "cash", "--to-account-id", "reserve", "--amount-minor", "1250",
		"--currency", "USD", "--external-id", "transfer-1", "--description", "monthly move", "--local-owner")
	if err != nil {
		t.Fatalf("journal-transfer: %v\n%s", err, out)
	}
	got := journal.request
	if got == nil || got.FromAccountId != "cash" || got.ToAccountId != "reserve" || got.AmountMinor != 1250 || got.Currency != "USD" || got.ExternalId != "transfer-1" || got.Description != "monthly move" || got.OccurredAt == nil {
		t.Fatalf("Transfer request = %+v, want the exact two-account transfer", got)
	}
	if journal.authorization != "Bearer owner-session" {
		t.Fatalf("Transfer authorization = %q, want the owner session", journal.authorization)
	}
	if !strings.Contains(out, "Created 2 paired transfer postings.") {
		t.Fatalf("journal-transfer output = %q", out)
	}
}

func TestCryptoPreviewParsesObservationAndKeepsCreditWithheld(t *testing.T) {
	service := &ingestServiceStub{response: &ingestpb.PreviewCryptoObservationResponse{
		Network: "eip155:8453", AtomicAmount: "1000000", PostingState: "withheld", BudgetState: "uncredited",
		Holds: []string{"wallet_ownership_unverified", "authorized_valuation_absent"},
	}}
	ingestPath, ingestHandler := ingestconnect.NewIngestServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(ingestPath, ingestHandler)
	core := cliapptest.NewTestApp(t, mux)
	groups, err := RegisterGroups(core, cliapptest.ReadManifest(t))
	if err != nil {
		t.Fatal(err)
	}
	observation := `{"network":"eip155:8453","assetId":"eip155:8453/slip44:60","atomicAmount":"1000000","decimals":6,"state":"CRYPTO_OBSERVATION_STATE_FINALIZED","legKind":"CRYPTO_LEG_KIND_DEPOSIT","sourceReference":"fixture:held-out","transactionHash":"0xabc","blockHash":"0xdef","address":"0x123"}`
	out, err := cliapptest.RunCommand(t, cliapptest.FindCommand(t, groups, "ingest", "crypto-preview"), core, "--observation", observation)
	if err != nil {
		t.Fatalf("crypto-preview: %v\n%s", err, out)
	}
	if got := service.request; got == nil || got.Network != "eip155:8453" || got.AssetId != "eip155:8453/slip44:60" || got.AtomicAmount != "1000000" || got.Decimals != 6 || got.SourceReference != "fixture:held-out" || got.Address != "0x123" {
		t.Fatalf("PreviewCryptoObservation request = %+v, want supplied observation", got)
	}
	for _, want := range []string{"Crypto observation preview; journal posting withheld and budget uncredited.", "wallet_ownership_unverified", "authorized_valuation_absent"} {
		if !strings.Contains(out, want) {
			t.Errorf("crypto-preview output lacks %q:\n%s", want, out)
		}
	}
}
