package commerce

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/servicetoken"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	ingestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest/ingest_v1connect"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
)

const (
	MoneyLedgerServiceOrderPrincipal = "landing-page-business-suite-money-ledger-writer"
	MoneyLedgerServiceOrderAudience  = "scenario-authenticator:money-ledger"
	MoneyLedgerServiceOrderScope     = "money-ledger:write"
)

type MoneyLedgerServiceOrderTokenSource interface {
	Token(context.Context) (string, error)
}

// MoneyLedgerTestServiceWriter posts explicitly test-only LPBS captures to an
// isolated loopback Money Ledger service. Production/non-loopback destinations
// are rejected, and Treasury x402 captures must continue to use Treasury's
// accepted seller receipt rather than this writer.
type MoneyLedgerTestServiceWriter struct {
	client ingestconnect.IngestServiceClient
	tokens MoneyLedgerServiceOrderTokenSource
}

func NewMoneyLedgerServiceOrderTokens(credentials servicetoken.CredentialSource, authenticatorURL string, client *http.Client, options ...servicetoken.Option) *servicetoken.Tokens {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	} else {
		clone := *client
		client = &clone
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	exchange := servicetoken.AuthenticatorExchanger(client, func(context.Context) (string, error) {
		if strings.TrimSpace(authenticatorURL) == "" {
			return "", errors.New("Scenario Authenticator URL is unavailable")
		}
		return strings.TrimRight(strings.TrimSpace(authenticatorURL), "/"), nil
	})
	configured := []servicetoken.Option{servicetoken.WithExchanger(exchange)}
	configured = append(configured, options...)
	return servicetoken.New(servicetoken.Config{
		Name:       "LPBS Money Ledger test-order writer",
		Credential: credentials,
		Expect:     servicetoken.ExpectIdentity(MoneyLedgerServiceOrderPrincipal, MoneyLedgerServiceOrderAudience, MoneyLedgerServiceOrderScope),
		IssueHint:  "the operator configures and issues the LPBS Money Ledger service-order principal",
	}, configured...)
}

func NewMoneyLedgerTestServiceWriter(endpoint string, tokens MoneyLedgerServiceOrderTokenSource, client *http.Client) (*MoneyLedgerTestServiceWriter, error) {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return nil, errors.New("Money Ledger test service endpoint must be an origin URL")
	}
	if parsed.Scheme != "http" || !isLoopbackHost(parsed.Hostname()) {
		return nil, errors.New("Money Ledger test service endpoint must use loopback HTTP")
	}
	if tokens == nil {
		return nil, errors.New("Money Ledger service-order token source is required")
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	} else {
		clone := *client
		client = &clone
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &MoneyLedgerTestServiceWriter{client: ingestconnect.NewIngestServiceClient(client, strings.TrimRight(parsed.String(), "/")), tokens: tokens}, nil
}

func (w *MoneyLedgerTestServiceWriter) IngestSandboxCapture(ctx context.Context, event *sharedpb.MoneyEvent) (ServiceOrderLedgerReceipt, error) {
	if w == nil || w.client == nil || w.tokens == nil {
		return ServiceOrderLedgerReceipt{}, errors.New("Money Ledger test service writer is unavailable")
	}
	if event == nil || !strings.HasPrefix(event.GetExternalId(), "service-order:") || !strings.HasPrefix(event.GetCategory(), "test.service_order.") || event.GetAmountMinor() <= 0 || event.GetCurrency() != "USD" {
		return ServiceOrderLedgerReceipt{}, errors.New("Money Ledger service-order writer accepts only positive USD test capture events")
	}
	token, err := w.tokens.Token(ctx)
	if err != nil {
		return ServiceOrderLedgerReceipt{}, err
	}
	if strings.TrimSpace(token) == "" {
		return ServiceOrderLedgerReceipt{}, errors.New("Money Ledger service-order token is empty")
	}
	request := connect.NewRequest(&ingestpb.IngestEventRequest{Event: event})
	request.Header().Set("Authorization", "Bearer "+token)
	response, err := w.client.IngestEvent(ctx, request)
	if err != nil {
		return ServiceOrderLedgerReceipt{}, err
	}
	if response.Msg.GetPosting() == nil || response.Msg.GetPosting().GetEvent() == nil || response.Msg.GetReceipt() == nil || response.Msg.GetReceipt().GetStatus() != "succeeded" || strings.TrimSpace(response.Msg.GetPosting().GetId()) == "" || strings.TrimSpace(response.Msg.GetReceipt().GetId()) == "" {
		return ServiceOrderLedgerReceipt{}, errors.New("Money Ledger did not accept the LPBS test capture")
	}
	posting := response.Msg.GetPosting()
	if posting.GetEvent().GetExternalId() != event.GetExternalId() || posting.GetEvent().GetAdapterId() != event.GetAdapterId() || posting.GetEvent().GetBookId() != event.GetBookId() || posting.GetEvent().GetAccountId() != event.GetAccountId() || posting.GetEvent().GetAmountMinor() != event.GetAmountMinor() || posting.GetEvent().GetCurrency() != event.GetCurrency() || posting.GetEvent().GetCategory() != event.GetCategory() {
		return ServiceOrderLedgerReceipt{}, errors.New("Money Ledger returned a posting for a different service-order capture")
	}
	return ServiceOrderLedgerReceipt{PostingID: posting.GetId(), ReceiptID: response.Msg.GetReceipt().GetId(), ExternalID: posting.GetEvent().GetExternalId(), Duplicate: response.Msg.GetDuplicate()}, nil
}
