package ledger

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	ingestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest/ingest_v1connect"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const treasuryAdapterID = "treasury"

type MoneyLedgerEmitter struct {
	client            ingestconnect.IngestServiceClient
	bookID, accountID string
}

func NewMoneyLedgerEmitter(baseURL, bookID, accountID string, client *http.Client) (*MoneyLedgerEmitter, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	bookID, accountID = strings.TrimSpace(bookID), strings.TrimSpace(accountID)
	if baseURL == "" || bookID == "" || accountID == "" {
		return nil, errors.New("MONEY_LEDGER_API_URL, TREASURY_LEDGER_BOOK_ID, and TREASURY_LEDGER_ACCOUNT_ID are required")
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &MoneyLedgerEmitter{client: ingestconnect.NewIngestServiceClient(client, baseURL), bookID: bookID, accountID: accountID}, nil
}

func (e *MoneyLedgerEmitter) Emit(ctx context.Context, value Emission) (bool, error) {
	if e == nil || e.client == nil {
		return false, errors.New("money-ledger client is required")
	}
	_, err := e.client.RegisterAdapter(ctx, connect.NewRequest(&ingestpb.RegisterAdapterRequest{Adapter: &ingestpb.Adapter{Id: treasuryAdapterID, Name: "Treasury settlements", Kind: ingestpb.AdapterKind_ADAPTER_KIND_AGGREGATOR, Enabled: true}}))
	if err != nil {
		return false, fmt.Errorf("register treasury adapter: %w", err)
	}
	basis := sharedpb.Basis_BASIS_AUTHORITATIVE
	if value.Basis == "operator_asserted" {
		basis = sharedpb.Basis_BASIS_OPERATOR_ASSERTED
	}
	response, err := e.client.IngestEvent(ctx, connect.NewRequest(&ingestpb.IngestEventRequest{Event: &sharedpb.MoneyEvent{
		Id: value.ID, ExternalId: value.ExternalID, AdapterId: treasuryAdapterID,
		AccountId: e.accountID, BookId: e.bookID, AmountMinor: value.AmountMinor,
		Currency: value.Currency, OccurredAt: timestamppb.New(value.OccurredAt), FetchedAt: timestamppb.New(value.FetchedAt),
		Basis: basis, Description: value.Description, Category: "agent_spend",
	}}))
	if err != nil {
		return false, fmt.Errorf("ingest treasury settlement: %w", err)
	}
	if response.Msg == nil || response.Msg.Posting == nil || response.Msg.Receipt == nil || response.Msg.Receipt.Status != "succeeded" {
		return false, errors.New("money-ledger returned an incomplete acceptance receipt")
	}
	return response.Msg.Duplicate, nil
}

var _ Emitter = (*MoneyLedgerEmitter)(nil)
