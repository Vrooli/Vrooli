package ledger_test

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

	"money-ledger/internal/access"
	"money-ledger/internal/testutil/authfixture"
	"money-ledger/internal/testutil/ledgertest"
)

func as[T any](msg *T, token string) *connect.Request[T] {
	request := connect.NewRequest(msg)
	request.Header().Set("Authorization", "Bearer "+token)
	return request
}

func TestComputeManagerCanReverseOnlyItsOwnOriginalInvoicePosting(t *testing.T) {
	ctx := context.Background()
	l := ledgertest.New(t)
	owner := l.Identity.Owner()
	cm := l.Identity.Service(authfixture.ComputeManagerInvoiceWriter, authfixture.ComputeInvoiceScope)
	_, err := l.Ingest.RegisterAdapter(ctx, as(&ingestpb.RegisterAdapterRequest{Adapter: &ingestpb.Adapter{Id: access.ComputeManagerInvoiceAdapter, Name: "Compute Manager provider invoices", Kind: ingestpb.AdapterKind_ADAPTER_KIND_AGGREGATOR, Enabled: true}}, owner))
	require.NoError(t, err)
	_, err = l.Ingest.RegisterAdapter(ctx, as(adapterRequestForLedgerTest("owner-expense"), owner))
	require.NoError(t, err)
	invoice := &sharedpb.MoneyEvent{ExternalId: "fixture-invoice-line", AdapterId: access.ComputeManagerInvoiceAdapter, AccountId: l.Account.Id, BookId: l.Book.Id, AmountMinor: -250, Currency: "USD", OccurredAt: timestamppb.New(time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC)), FetchedAt: timestamppb.New(time.Date(2026, 11, 23, 0, 1, 0, 0, time.UTC)), Basis: sharedpb.Basis_BASIS_AUTHORITATIVE, Description: "FIXTURE ONLY", Category: "compute.provider_invoice"}
	posted, err := l.Ingest.IngestEvent(ctx, as(&ingestpb.IngestEventRequest{Event: invoice}, cm))
	require.NoError(t, err)
	firstReversal, err := l.Journal.ReversePosting(ctx, as(&ledgerpb.ReversePostingRequest{PostingId: posted.Msg.Posting.Id, Reason: "fixture correction"}, cm))
	require.NoError(t, err)
	_, err = l.Journal.ReversePosting(ctx, as(&ledgerpb.ReversePostingRequest{PostingId: firstReversal.Msg.Posting.Id, Reason: "must refuse reversal of reversal"}, cm))
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	manual := &sharedpb.MoneyEvent{ExternalId: "owner-expense-line", AdapterId: "owner-expense", AccountId: l.Account.Id, BookId: l.Book.Id, AmountMinor: -50, Currency: "USD", OccurredAt: invoice.OccurredAt, FetchedAt: invoice.FetchedAt, Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED, Description: "owner", Category: "manual"}
	owned, err := l.Ingest.IngestEvent(ctx, as(&ingestpb.IngestEventRequest{Event: manual}, owner))
	require.NoError(t, err)
	_, err = l.Journal.ReversePosting(ctx, as(&ledgerpb.ReversePostingRequest{PostingId: owned.Msg.Posting.Id, Reason: "must refuse other adapter"}, cm))
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
}

func adapterRequestForLedgerTest(id string) *ingestpb.RegisterAdapterRequest {
	return &ingestpb.RegisterAdapterRequest{Adapter: &ingestpb.Adapter{Id: id, Name: id, Kind: ingestpb.AdapterKind_ADAPTER_KIND_AGGREGATOR, Enabled: true}}
}

// The owner's corrections and transfers are journaled and audited in the
// owner's name, and no service can make them.
func TestOwnerCorrectionsAreAttributedAndServicesCannotMakeThem(t *testing.T) {
	ctx := context.Background()
	l := ledgertest.New(t)
	owner := l.Identity.Owner()
	treasury := l.Identity.Service(authfixture.TreasuryWriter, authfixture.WriteScope)
	savings, err := l.Books.CreateAccount(ctx, as(&ledgerpb.CreateAccountRequest{BookId: l.Book.Id, Name: "Savings", AccountKind: ledgerpb.AccountKind_ASSET}, owner))
	require.NoError(t, err)
	transfer := &ledgerpb.TransferRequest{FromAccountId: l.Account.Id, ToAccountId: savings.Msg.Account.Id, AmountMinor: 5000, Currency: "USD", ExternalId: "move-1", OccurredAt: timestamppb.New(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))}
	_, err = l.Journal.Transfer(ctx, as(transfer, treasury))
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), "a service never moves the owner's money between accounts")
	moved, err := l.Journal.Transfer(ctx, as(transfer, owner))
	require.NoError(t, err)
	require.Len(t, moved.Msg.Postings, 2)
	for _, posting := range moved.Msg.Postings {
		require.Equal(t, "operator", posting.Actor)
	}
	_, err = l.Journal.ReversePosting(ctx, as(&ledgerpb.ReversePostingRequest{PostingId: moved.Msg.Postings[0].Id, Reason: "fixture"}, treasury))
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), "reversal is the owner's decision alone")
	reversed, err := l.Journal.ReversePosting(ctx, as(&ledgerpb.ReversePostingRequest{PostingId: moved.Msg.Postings[0].Id, Reason: "entered twice"}, owner))
	require.NoError(t, err)
	require.Equal(t, "operator", reversed.Msg.Posting.Actor)
	_, err = l.Books.ArchiveBook(ctx, as(&ledgerpb.ArchiveBookRequest{BookId: l.Book.Id}, owner))
	require.NoError(t, err)
	var auditActor string
	require.NoError(t, l.DB.QueryRowContext(ctx, `SELECT actor FROM ledger_audit WHERE entity_type='book' AND entity_id=?`, l.Book.Id).Scan(&auditActor))
	require.Equal(t, "operator", auditActor)
}

// A goal declaration without a goal is a bad request, not a server failure.
func TestDeclareGoalWithoutAGoalIsInvalid(t *testing.T) {
	l := ledgertest.New(t)
	_, err := l.Position.DeclareGoal(context.Background(), as(&ledgerpb.DeclareGoalRequest{}, l.Identity.Owner()))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
