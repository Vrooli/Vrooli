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
	client ingestconnect.IngestServiceClient
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
	var identity struct {
		FixtureID string `json:"fixture_id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&identity); err != nil || identity.FixtureID != fixtureID {
		return nil, errors.New("Money Ledger endpoint identity does not match the requested isolated fixture")
	}
	return &MoneyLedgerTestFixtureWriter{client: ingestconnect.NewIngestServiceClient(client, strings.TrimRight(endpoint, "/"))}, nil
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
