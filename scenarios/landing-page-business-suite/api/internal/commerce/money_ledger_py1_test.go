package commerce

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	ingestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest/ingest_v1connect"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
)

func TestMoneyLedgerPY1TestServiceWriterPostsScopedCheckoutAndSubscriptionInvoice(t *testing.T) {
	var gotAuth string
	mux := http.NewServeMux()
	path, handler := ingestconnect.NewIngestServiceHandler(moneyLedgerIngestStub{ingest: func(_ context.Context, request *connect.Request[ingestpb.IngestEventRequest]) (*connect.Response[ingestpb.IngestEventResponse], error) {
		gotAuth = request.Header().Get("Authorization")
		return connect.NewResponse(&ingestpb.IngestEventResponse{Posting: &sharedpb.Posting{Id: "posting-py1", Event: request.Msg.GetEvent()}, Receipt: &ingestpb.Receipt{Id: "receipt-py1", Status: "succeeded"}}), nil
	}})
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	writer, err := NewMoneyLedgerPY1TestServiceWriter(server.URL, moneyLedgerOrderToken("scoped-test-token"), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []*sharedpb.MoneyEvent{
		{ExternalId: "lpbs-py1-checkout:cs_test_123", AdapterId: "lpbs-test", BookId: "book-test", AccountId: "revenue-test", AmountMinor: 2900, Currency: "USD", Category: "test.py1.stripe_checkout", Description: "PY1 test Checkout cs_test_123 (checkout_success)"},
		{ExternalId: "lpbs-py1-invoice:in_test_123", AdapterId: "lpbs-test", BookId: "book-test", AccountId: "revenue-test", AmountMinor: 2900, Currency: "USD", Category: "test.py1.stripe_subscription_invoice", Description: "PY1 test subscription invoice in_test_123 (renewal; subscription_cycle)"},
	} {
		got, err := writer.IngestSandboxCapture(context.Background(), event)
		if err != nil || got.PostingID != "posting-py1" || got.ReceiptID != "receipt-py1" || got.ExternalID != event.GetExternalId() || gotAuth != "Bearer scoped-test-token" {
			t.Fatalf("writer receipt=%+v auth=%q err=%v", got, gotAuth, err)
		}
	}
}

func TestMoneyLedgerPY1TestServiceWriterRejectsOtherMoneyAndNonloopbackDestinations(t *testing.T) {
	if _, err := NewMoneyLedgerPY1TestServiceWriter("https://ledger.example", moneyLedgerOrderToken("token"), nil); err == nil {
		t.Fatal("non-loopback Money Ledger destination accepted")
	}
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("invalid MoneyEvent reached Ledger") }))
	defer server.Close()
	writer, err := NewMoneyLedgerPY1TestServiceWriter(server.URL, moneyLedgerOrderToken("token"), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []*sharedpb.MoneyEvent{
		{ExternalId: "service-order:1", AmountMinor: 2900, Currency: "USD", Category: "test.py1.stripe_checkout"},
		{ExternalId: "lpbs-py1-checkout:cs_live_123", AmountMinor: 2900, Currency: "USD", Category: "test.py1.stripe_checkout"},
		{ExternalId: "lpbs-py1-checkout:cs_test_123", AmountMinor: -2900, Currency: "USD", Category: "test.py1.stripe_checkout"},
		{ExternalId: "lpbs-py1-checkout:cs_test_123", AmountMinor: 2900, Currency: "EUR", Category: "test.py1.stripe_checkout"},
		{ExternalId: "lpbs-py1-invoice:in_test_123", AmountMinor: 2900, Currency: "USD", Category: "test.py1.stripe_checkout"},
	} {
		if _, err := writer.IngestSandboxCapture(context.Background(), event); err == nil {
			t.Fatalf("unsafe event accepted: %+v", event)
		}
	}
}
