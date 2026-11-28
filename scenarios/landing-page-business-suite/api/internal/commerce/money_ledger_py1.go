package commerce

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	ingestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest/ingest_v1connect"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
)

// MoneyLedgerPY1TestServiceWriter is restricted to positive, test-only Stripe
// Checkout captures and a loopback Money Ledger test endpoint.
type MoneyLedgerPY1TestServiceWriter struct {
	client ingestconnect.IngestServiceClient
	tokens MoneyLedgerServiceOrderTokenSource
}

func NewMoneyLedgerPY1TestServiceWriter(endpoint string, tokens MoneyLedgerServiceOrderTokenSource, client *http.Client) (*MoneyLedgerPY1TestServiceWriter, error) {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") || parsed.Scheme != "http" || !isLoopbackHost(parsed.Hostname()) {
		return nil, errors.New("Money Ledger PY1 test endpoint must use loopback HTTP")
	}
	if tokens == nil {
		return nil, errors.New("Money Ledger PY1 test writer token source is required")
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	} else {
		clone := *client
		client = &clone
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &MoneyLedgerPY1TestServiceWriter{client: ingestconnect.NewIngestServiceClient(client, strings.TrimRight(parsed.String(), "/")), tokens: tokens}, nil
}

func (w *MoneyLedgerPY1TestServiceWriter) IngestSandboxCapture(ctx context.Context, event *sharedpb.MoneyEvent) (ServiceOrderLedgerReceipt, error) {
	if w == nil || w.client == nil || w.tokens == nil {
		return ServiceOrderLedgerReceipt{}, errors.New("Money Ledger PY1 test writer is unavailable")
	}
	checkout := event != nil && strings.HasPrefix(event.GetExternalId(), "lpbs-py1-checkout:cs_test_") && event.GetCategory() == "test.py1.stripe_checkout" && strings.HasPrefix(event.GetDescription(), "PY1 test Checkout cs_test_")
	invoice := event != nil && strings.HasPrefix(event.GetExternalId(), "lpbs-py1-invoice:in_") && event.GetCategory() == "test.py1.stripe_subscription_invoice" && strings.HasPrefix(event.GetDescription(), "PY1 test subscription invoice in_")
	if event == nil || (!checkout && !invoice) || event.GetAmountMinor() <= 0 || event.GetCurrency() != "USD" {
		return ServiceOrderLedgerReceipt{}, errors.New("Money Ledger PY1 test writer accepts only positive USD Stripe test Checkout or subscription invoices")
	}
	token, err := w.tokens.Token(ctx)
	if err != nil || strings.TrimSpace(token) == "" {
		return ServiceOrderLedgerReceipt{}, errors.New("Money Ledger PY1 test service token is unavailable")
	}
	request := connect.NewRequest(&ingestpb.IngestEventRequest{Event: event})
	request.Header().Set("Authorization", "Bearer "+token)
	response, err := w.client.IngestEvent(ctx, request)
	if err != nil {
		return ServiceOrderLedgerReceipt{}, err
	}
	posting := response.Msg.GetPosting()
	if posting == nil || posting.GetEvent() == nil || response.Msg.GetReceipt() == nil || response.Msg.GetReceipt().GetStatus() != "succeeded" || strings.TrimSpace(posting.GetId()) == "" || strings.TrimSpace(response.Msg.GetReceipt().GetId()) == "" {
		return ServiceOrderLedgerReceipt{}, errors.New("Money Ledger did not accept the PY1 test economic capture")
	}
	actual := posting.GetEvent()
	if actual.GetExternalId() != event.GetExternalId() || actual.GetAdapterId() != event.GetAdapterId() || actual.GetBookId() != event.GetBookId() || actual.GetAccountId() != event.GetAccountId() || actual.GetAmountMinor() != event.GetAmountMinor() || actual.GetCurrency() != event.GetCurrency() || actual.GetCategory() != event.GetCategory() {
		return ServiceOrderLedgerReceipt{}, fmt.Errorf("Money Ledger returned a posting for a different PY1 test economic capture")
	}
	return ServiceOrderLedgerReceipt{PostingID: posting.GetId(), ReceiptID: response.Msg.GetReceipt().GetId(), ExternalID: actual.GetExternalId(), Duplicate: response.Msg.GetDuplicate()}, nil
}
