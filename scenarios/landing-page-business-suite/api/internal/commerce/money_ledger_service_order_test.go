package commerce

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/servicetoken"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	ingestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest/ingest_v1connect"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
)

type moneyLedgerOrderToken string

func (t moneyLedgerOrderToken) Token(context.Context) (string, error) { return string(t), nil }

func TestMoneyLedgerServiceOrderTokensRequireDedicatedWriterIdentity(t *testing.T) {
	const credential = "lpbs-test-only-money-ledger-writer-credential"
	seenCredential := ""
	tokens := NewMoneyLedgerServiceOrderTokens(func(context.Context) (string, error) {
		return credential, nil
	}, "http://127.0.0.1:1", nil, servicetoken.WithExchanger(func(_ context.Context, gotCredential string) (servicetoken.Exchanged, error) {
		seenCredential = gotCredential
		return servicetoken.Exchanged{
			AccessToken: "lpbs-test-only-money-ledger-writer-token", ExpiresAt: time.Now().Add(time.Minute),
			Subject: MoneyLedgerServiceOrderPrincipal, Audience: MoneyLedgerServiceOrderAudience,
			Scopes: []string{MoneyLedgerServiceOrderScope},
		}, nil
	}))
	got, err := tokens.Token(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if seenCredential != credential || got != "lpbs-test-only-money-ledger-writer-token" {
		t.Fatalf("writer identity exchange credential=%q token=%q", seenCredential, got)
	}
}

func TestMoneyLedgerTestServiceWriterPostsScopedTestCaptureAndLinksAcceptance(t *testing.T) {
	var gotAuth string
	mux := http.NewServeMux()
	path, handler := ingestconnect.NewIngestServiceHandler(moneyLedgerIngestStub{ingest: func(ctx context.Context, request *connect.Request[ingestpb.IngestEventRequest]) (*connect.Response[ingestpb.IngestEventResponse], error) {
		gotAuth = request.Header().Get("Authorization")
		return connect.NewResponse(&ingestpb.IngestEventResponse{
			Posting: &sharedpb.Posting{Id: "posting-1", Event: request.Msg.Event},
			Receipt: &ingestpb.Receipt{Id: "receipt-1", Status: "succeeded"},
		}), nil
	}})
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	writer, err := NewMoneyLedgerTestServiceWriter(server.URL, moneyLedgerOrderToken("scoped-test-token"), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	event := &sharedpb.MoneyEvent{ExternalId: "service-order:order-1:capture:test", AdapterId: "lpbs-test", BookId: "cf2-test", AccountId: "service-revenue", AmountMinor: 4900, Currency: "USD", Category: "test.service_order.sandbox"}
	got, err := writer.IngestSandboxCapture(context.Background(), event)
	if err != nil {
		t.Fatal(err)
	}
	if got != (ServiceOrderLedgerReceipt{PostingID: "posting-1", ReceiptID: "receipt-1", ExternalID: event.ExternalId}) {
		t.Fatalf("receipt = %+v", got)
	}
	if gotAuth != "Bearer scoped-test-token" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
}

func TestMoneyLedgerTestServiceWriterRejectsUnsafeOriginsAndEvents(t *testing.T) {
	if _, err := NewMoneyLedgerTestServiceWriter("https://ledger.example", moneyLedgerOrderToken("token"), http.DefaultClient); err == nil {
		t.Fatal("non-loopback service URL was accepted")
	}
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("invalid event reached ledger") }))
	defer server.Close()
	writer, err := NewMoneyLedgerTestServiceWriter(server.URL, moneyLedgerOrderToken("token"), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = writer.IngestSandboxCapture(context.Background(), &sharedpb.MoneyEvent{ExternalId: "unrelated", AdapterId: "test", BookId: "test", AccountId: "test", AmountMinor: 4900, Currency: "USD", Category: "test.service_order.sandbox"})
	if err == nil || !strings.Contains(err.Error(), "only positive USD test capture events") {
		t.Fatalf("invalid event error = %v", err)
	}
}

func TestMoneyLedgerTestServiceWriterRejectsIncompleteOrChangedAcceptance(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*ingestpb.IngestEventResponse)
	}{
		{name: "empty IDs", mutate: func(response *ingestpb.IngestEventResponse) {
			response.Posting.Id = ""
			response.Receipt.Id = ""
		}},
		{name: "changed amount", mutate: func(response *ingestpb.IngestEventResponse) {
			response.Posting.Event.AmountMinor++
		}},
		{name: "changed category", mutate: func(response *ingestpb.IngestEventResponse) {
			response.Posting.Event.Category = "production.service_order"
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mux := http.NewServeMux()
			path, handler := ingestconnect.NewIngestServiceHandler(moneyLedgerIngestStub{ingest: func(_ context.Context, request *connect.Request[ingestpb.IngestEventRequest]) (*connect.Response[ingestpb.IngestEventResponse], error) {
				response := &ingestpb.IngestEventResponse{
					Posting: &sharedpb.Posting{Id: "posting-1", Event: request.Msg.Event},
					Receipt: &ingestpb.Receipt{Id: "receipt-1", Status: "succeeded"},
				}
				tc.mutate(response)
				return connect.NewResponse(response), nil
			}})
			mux.Handle(path, handler)
			server := httptest.NewServer(mux)
			defer server.Close()
			writer, err := NewMoneyLedgerTestServiceWriter(server.URL, moneyLedgerOrderToken("token"), server.Client())
			if err != nil {
				t.Fatal(err)
			}
			event := &sharedpb.MoneyEvent{ExternalId: "service-order:order-2:capture:test", AdapterId: "lpbs-test", BookId: "cf2-test", AccountId: "service-revenue", AmountMinor: 4900, Currency: "USD", Category: "test.service_order.sandbox"}
			if _, err := writer.IngestSandboxCapture(context.Background(), event); err == nil {
				t.Fatal("changed or incomplete Money Ledger response was accepted")
			}
		})
	}
}

type moneyLedgerIngestStub struct {
	ingestconnect.UnimplementedIngestServiceHandler
	ingest func(context.Context, *connect.Request[ingestpb.IngestEventRequest]) (*connect.Response[ingestpb.IngestEventResponse], error)
}

func (s moneyLedgerIngestStub) IngestEvent(ctx context.Context, request *connect.Request[ingestpb.IngestEventRequest]) (*connect.Response[ingestpb.IngestEventResponse], error) {
	return s.ingest(ctx, request)
}
