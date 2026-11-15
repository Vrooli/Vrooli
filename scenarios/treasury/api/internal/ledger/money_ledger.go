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

// MoneyLedgerEmitter posts settlements to the operator's chosen Money Ledger
// destination as Treasury's own service principal. The destination is read on
// every delivery, so choosing or changing it needs no restart; until one is
// chosen each delivery fails with ErrDestinationUnset and the settlement stays
// queued.
type MoneyLedgerEmitter struct {
	client      ingestconnect.IngestServiceClient
	destination DestinationSource
	tokens      TokenSource
}

func NewMoneyLedgerEmitter(baseURL string, destination DestinationSource, tokens TokenSource, client *http.Client) (*MoneyLedgerEmitter, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, errors.New("the Money Ledger origin could not be resolved (is money-ledger running?)")
	}
	if destination == nil || tokens == nil {
		return nil, errors.New("a settlement destination and Treasury's money-ledger service identity are required")
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &MoneyLedgerEmitter{client: ingestconnect.NewIngestServiceClient(client, baseURL), destination: destination, tokens: tokens}, nil
}

// authorize presents Treasury's money-ledger principal; Money Ledger accepts a
// journal write only from a verified identity.
func (e *MoneyLedgerEmitter) authorize(ctx context.Context, header http.Header) error {
	token, err := e.tokens.Token(ctx)
	if err != nil || strings.TrimSpace(token) == "" {
		return errors.New("Treasury service identity is unavailable")
	}
	header.Set("Authorization", "Bearer "+token)
	return nil
}

func (e *MoneyLedgerEmitter) Emit(ctx context.Context, value Emission) (bool, error) {
	receipt, err := e.EmitWithReceipt(ctx, value)
	return receipt.Duplicate, err
}

func (e *MoneyLedgerEmitter) EmitWithReceipt(ctx context.Context, value Emission) (Acceptance, error) {
	if e == nil || e.client == nil {
		return Acceptance{}, errors.New("money-ledger client is required")
	}
	destination, err := e.destination.Destination(ctx)
	if err != nil {
		return Acceptance{}, err
	}
	if destination.Currency != "" && !strings.EqualFold(value.Currency, destination.Currency) {
		return Acceptance{}, fmt.Errorf("settlement currency %s does not match the destination book currency %s", value.Currency, destination.Currency)
	}
	register := connect.NewRequest(&ingestpb.RegisterAdapterRequest{Adapter: &ingestpb.Adapter{Id: treasuryAdapterID, Name: "Treasury settlements", Kind: ingestpb.AdapterKind_ADAPTER_KIND_AGGREGATOR, Enabled: true}})
	if err := e.authorize(ctx, register.Header()); err != nil {
		return Acceptance{}, err
	}
	if _, err = e.client.RegisterAdapter(ctx, register); err != nil {
		return Acceptance{}, fmt.Errorf("register treasury adapter: %w", err)
	}
	basis := sharedpb.Basis_BASIS_AUTHORITATIVE
	if value.Basis == "operator_asserted" {
		basis = sharedpb.Basis_BASIS_OPERATOR_ASSERTED
	}
	ingest := connect.NewRequest(&ingestpb.IngestEventRequest{Event: &sharedpb.MoneyEvent{
		Id: value.ID, ExternalId: value.ExternalID, AdapterId: treasuryAdapterID,
		AccountId: destination.AccountID, BookId: destination.BookID, AmountMinor: value.AmountMinor,
		Currency: value.Currency, OccurredAt: timestamppb.New(value.OccurredAt), FetchedAt: timestamppb.New(value.FetchedAt),
		Basis: basis, Description: value.Description, Category: "agent_spend",
	}})
	if err := e.authorize(ctx, ingest.Header()); err != nil {
		return Acceptance{}, err
	}
	response, err := e.client.IngestEvent(ctx, ingest)
	if err != nil {
		return Acceptance{}, fmt.Errorf("ingest treasury settlement: %w", err)
	}
	if response.Msg == nil || response.Msg.Posting == nil || response.Msg.Posting.Event == nil || response.Msg.Posting.Event.ExternalId != value.ExternalID || response.Msg.Receipt == nil || response.Msg.Receipt.Status != "succeeded" || strings.TrimSpace(response.Msg.Posting.Id) == "" || strings.TrimSpace(response.Msg.Receipt.Id) == "" {
		return Acceptance{}, errors.New("money-ledger returned an incomplete or mismatched acceptance receipt")
	}
	return Acceptance{Duplicate: response.Msg.Duplicate, PostingID: response.Msg.Posting.Id, ReceiptID: response.Msg.Receipt.Id}, nil
}

var (
	_ Emitter        = (*MoneyLedgerEmitter)(nil)
	_ ReceiptEmitter = (*MoneyLedgerEmitter)(nil)
)
