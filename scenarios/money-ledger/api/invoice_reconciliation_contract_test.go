package main_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"

	"money-ledger/internal/testutil/ledgertest"
)

func invoiceRequest[T any](message *T, token string) *connect.Request[T] {
	request := connect.NewRequest(message)
	request.Header().Set("Authorization", "Bearer "+token)
	return request
}

// This is a fixture-only contract test through Money Ledger's production
// Connect server, authorization gate, and SQLite journal. It demonstrates the
// path Compute Manager must call; it does not fetch or post any real invoice.
func TestComputeManagerFixtureInvoiceIsExactlyOnceAndCorrectionIsReversal(t *testing.T) {
	ctx := context.Background()
	ledger := ledgertest.New(t)
	occurred := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	adapters, err := ledger.Ingest.ListAdapters(ctx, connect.NewRequest(&ingestpb.ListAdaptersRequest{}))
	require.NoError(t, err)
	var invoiceAdapter *ingestpb.Adapter
	for _, adapter := range adapters.Msg.GetAdapters() {
		if adapter.GetId() == "compute-manager-provider-invoices" {
			invoiceAdapter = adapter
			break
		}
	}
	require.NotNil(t, invoiceAdapter, "Money Ledger migrations seed the fixed invoice adapter")
	require.Equal(t, "Compute Manager provider invoices", invoiceAdapter.GetName())
	event := &sharedpb.MoneyEvent{
		ExternalId:  "compute-manager:digitalocean:invoice:fixture-invoice-01:line:fixture-line-01",
		AdapterId:   "compute-manager-provider-invoices",
		AccountId:   ledger.Account.Id,
		BookId:      ledger.Book.Id,
		AmountMinor: -1250,
		Currency:    "USD",
		OccurredAt:  timestamppb.New(occurred),
		FetchedAt:   timestamppb.New(occurred.Add(time.Minute)),
		Basis:       sharedpb.Basis_BASIS_AUTHORITATIVE,
		Description: "FIXTURE ONLY: Compute Manager provider invoice line; no live invoice",
		Category:    "test.compute_manager.provider_invoice",
	}
	cm := ledger.Identity.Service("compute-manager-money-ledger-invoice-writer", "money-ledger:compute-invoice")

	first, err := ledger.Ingest.IngestEvent(ctx, invoiceRequest(&ingestpb.IngestEventRequest{Event: event}, cm))
	require.NoError(t, err)
	require.NotNil(t, first.Msg.GetPosting())
	require.NotEmpty(t, first.Msg.GetPosting().GetId())
	require.False(t, first.Msg.GetDuplicate())
	require.Equal(t, event.GetExternalId(), first.Msg.GetPosting().GetEvent().GetExternalId())
	require.Equal(t, "succeeded", first.Msg.GetReceipt().GetStatus())

	retry, err := ledger.Ingest.IngestEvent(ctx, invoiceRequest(&ingestpb.IngestEventRequest{Event: event}, cm))
	require.NoError(t, err)
	require.True(t, retry.Msg.GetDuplicate())
	require.Equal(t, first.Msg.GetPosting().GetId(), retry.Msg.GetPosting().GetId())
	require.Equal(t, 1, ledger.Count(t, "postings"), "retry must not create a second expense")

	correction, err := ledger.Journal.ReversePosting(ctx, invoiceRequest(&ledgerpb.ReversePostingRequest{
		PostingId: first.Msg.GetPosting().GetId(),
		Reason:    "FIXTURE ONLY: corrected provider invoice line",
	}, cm))
	require.NoError(t, err)
	require.NotNil(t, correction.Msg.GetPosting())
	require.Equal(t, first.Msg.GetPosting().GetId(), correction.Msg.GetPosting().GetReversalOf())
	require.EqualValues(t, 1250, correction.Msg.GetPosting().GetEvent().GetAmountMinor())

	correctionRetry, err := ledger.Journal.ReversePosting(ctx, invoiceRequest(&ledgerpb.ReversePostingRequest{
		PostingId: first.Msg.GetPosting().GetId(),
		Reason:    "FIXTURE ONLY: corrected provider invoice line",
	}, cm))
	require.NoError(t, err)
	require.Equal(t, correction.Msg.GetPosting().GetId(), correctionRetry.Msg.GetPosting().GetId())
	require.Equal(t, 2, ledger.Count(t, "postings"), "a correction appends one linked reversal and never edits or duplicates")
}

// A signed token without the exact grant scope remains denied. Fixture signing
// does not create an actual principal or issue a credential.
func TestComputeManagerInvoiceWriterWithoutExactGrantScopeRemainsDenied(t *testing.T) {
	ctx := context.Background()
	ledger := ledgertest.New(t)
	service := ledger.Identity.Service("compute-manager-money-ledger-invoice-writer")
	event := &sharedpb.MoneyEvent{
		ExternalId: "compute-manager:digitalocean:invoice:fixture-invoice-closed:line:fixture-line-closed",
		AdapterId:  "compute-manager-provider-invoices", AccountId: ledger.Account.Id, BookId: ledger.Book.Id,
		AmountMinor: -100, Currency: "USD", OccurredAt: timestamppb.New(time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)),
		FetchedAt: timestamppb.New(time.Date(2026, 11, 23, 12, 1, 0, 0, time.UTC)), Basis: sharedpb.Basis_BASIS_AUTHORITATIVE,
		Description: "FIXTURE ONLY: denied before ledger write", Category: "test.compute_manager.provider_invoice",
	}
	_, err := ledger.Ingest.IngestEvent(ctx, invoiceRequest(&ingestpb.IngestEventRequest{Event: event}, service))
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("invoice writer result err=%v, want permission denied", err)
	}
	if ledger.Count(t, "postings") != 0 {
		t.Fatal("unauthorized Compute Manager principal created a posting")
	}
}
