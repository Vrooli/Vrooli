package commerce

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	ingestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest/ingest_v1connect"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
)

// MoneyLedgerTestFixtureWriter only accepts a loopback endpoint and a fixture
// identity previously issued by the isolated SQLite journey server.
type MoneyLedgerTestFixtureWriter struct {
	client   ingestconnect.IngestServiceClient
	identity moneyLedgerTestFixtureIdentity
}

type moneyLedgerTestFixtureIdentity struct {
	FixtureID        string `json:"fixture_id"`
	AdapterID        string `json:"adapter_id"`
	BookID           string `json:"book_id"`
	RevenueAccountID string `json:"revenue_account_id"`
}

func NewMoneyLedgerTestFixtureWriter(endpoint, fixtureID string) (*MoneyLedgerTestFixtureWriter, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "http" || !isLoopbackHost(parsed.Hostname()) {
		return nil, errors.New("Money Ledger test fixture endpoint must be a loopback HTTP URL")
	}
	if !strings.HasPrefix(fixtureID, "money-ledger-test-sqlite:") {
		return nil, errors.New("Money Ledger target is not an isolated SQLite journey fixture")
	}
	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Get(strings.TrimRight(endpoint, "/") + "/__journey/identity")
	if err != nil {
		return nil, errors.New("could not prove the Money Ledger isolated journey fixture identity")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("Money Ledger endpoint did not expose the isolated journey identity")
	}
	var identity moneyLedgerTestFixtureIdentity
	if err := json.NewDecoder(response.Body).Decode(&identity); err != nil || identity.FixtureID != fixtureID {
		return nil, errors.New("Money Ledger endpoint identity does not match the requested isolated fixture")
	}
	return &MoneyLedgerTestFixtureWriter{client: ingestconnect.NewIngestServiceClient(client, strings.TrimRight(endpoint, "/")), identity: identity}, nil
}

// ValidatePY1Target proves the configured ledger posting target belongs to
// the isolated fixture before any Stripe test Checkout session is created.
func (w *MoneyLedgerTestFixtureWriter) ValidatePY1Target(target PY1TestLedgerTarget) error {
	if w == nil || w.client == nil || strings.TrimSpace(w.identity.AdapterID) == "" || strings.TrimSpace(w.identity.BookID) == "" || strings.TrimSpace(w.identity.RevenueAccountID) == "" {
		return errors.New("Money Ledger fixture did not prove its adapter, book, and revenue account scope")
	}
	if target.AdapterID != w.identity.AdapterID || target.BookID != w.identity.BookID || target.RevenueAccountID != w.identity.RevenueAccountID {
		return errors.New("Money Ledger PY1 target does not match the isolated fixture scope")
	}
	return nil
}

// PY1Target returns the posting scope reported by the isolated fixture itself.
func (w *MoneyLedgerTestFixtureWriter) PY1Target() (PY1TestLedgerTarget, error) {
	if w == nil || w.client == nil || !strings.HasPrefix(w.identity.FixtureID, "money-ledger-test-sqlite:") {
		return PY1TestLedgerTarget{}, errors.New("Money Ledger fixture identity is not an isolated SQLite fixture")
	}
	target := PY1TestLedgerTarget{AdapterID: w.identity.AdapterID, BookID: w.identity.BookID, RevenueAccountID: w.identity.RevenueAccountID}
	if err := w.ValidatePY1Target(target); err != nil {
		return PY1TestLedgerTarget{}, err
	}
	return target, nil
}

func isLoopbackHost(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (w *MoneyLedgerTestFixtureWriter) IngestSandboxCapture(ctx context.Context, event *sharedpb.MoneyEvent) (ServiceOrderLedgerReceipt, error) {
	if w == nil || w.client == nil {
		return ServiceOrderLedgerReceipt{}, errors.New("Money Ledger test fixture writer is unavailable")
	}
	response, err := w.client.IngestEvent(ctx, connect.NewRequest(&ingestpb.IngestEventRequest{Event: event}))
	if err != nil {
		return ServiceOrderLedgerReceipt{}, err
	}
	if response.Msg.Posting == nil || response.Msg.Posting.Event == nil || response.Msg.Receipt == nil || response.Msg.Receipt.Status != "succeeded" {
		return ServiceOrderLedgerReceipt{}, errors.New("Money Ledger did not accept the sandbox event")
	}
	return ServiceOrderLedgerReceipt{PostingID: response.Msg.Posting.Id, ReceiptID: response.Msg.Receipt.Id, ExternalID: response.Msg.Posting.Event.ExternalId, Duplicate: response.Msg.Duplicate}, nil
}
