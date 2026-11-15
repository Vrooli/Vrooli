package ledger_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	"google.golang.org/protobuf/types/known/timestamppb"

	"money-ledger/internal/testutil/authfixture"
	"money-ledger/internal/testutil/ledgertest"
)

func as[T any](msg *T, token string) *connect.Request[T] {
	request := connect.NewRequest(msg)
	request.Header().Set("Authorization", "Bearer "+token)
	return request
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
