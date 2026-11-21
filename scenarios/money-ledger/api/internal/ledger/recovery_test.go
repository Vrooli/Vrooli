package ledger

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func correctionFixture(t *testing.T, amount int64) (*Store, context.Context, *sharedpb.Posting) {
	t.Helper()
	s, ctx := testStore(t)
	b, err := s.CreateBook(ctx, "Isolated recovery fixture", "USD")
	require.NoError(t, err)
	a, err := s.CreateAccount(ctx, b.Id, "Cash", ledgerpb.AccountKind_ASSET)
	require.NoError(t, err)
	p, _, err := s.Ingest(ctx, &sharedpb.MoneyEvent{ExternalId: "fixture-deposit", AdapterId: "fixture", AccountId: a.Id, BookId: b.Id, AmountMinor: amount, Currency: "USD", OccurredAt: timestamppb.Now(), Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED}, "test-operator")
	require.NoError(t, err)
	return s, ctx, p
}

func TestReversalRetryReturnsAcceptedCorrectionAfterFreshHandle(t *testing.T) {
	s, ctx, p := correctionFixture(t, 1250)
	first, err := s.Reverse(ctx, p.Id, "fixture removal", "operator")
	require.NoError(t, err)
	fresh := NewStore(s.db, s.now)
	again, err := fresh.Reverse(ctx, p.Id, "retry after lost response", "operator")
	require.NoError(t, err)
	require.Equal(t, first.Id, again.Id)
	require.Equal(t, first.Event.ExternalId, again.Event.ExternalId)
	rows, err := fresh.ListPostings(ctx, p.Event.AccountId, p.Event.BookId, "", "", 100)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	var count int
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ledger_audit WHERE entity_id=?`, first.Id).Scan(&count))
	require.Equal(t, 1, count)
	position, err := fresh.Position(ctx, p.Event.BookId)
	require.NoError(t, err)
	require.EqualValues(t, 0, position.CashMinor)
}

func TestReversalRejectsUnrepresentableNegation(t *testing.T) {
	s, ctx, p := correctionFixture(t, math.MinInt64)
	_, err := s.Reverse(ctx, p.Id, "cannot negate minimum int64", "operator")
	require.Error(t, err)
	var count int
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM postings WHERE reversal_of=?`, p.Id).Scan(&count))
	require.Zero(t, count)
}

func TestReversalAuditFailureRollsBackBeforeRetry(t *testing.T) {
	s, ctx, p := correctionFixture(t, 1250)
	_, err := s.db.ExecContext(ctx, `CREATE TRIGGER refuse_reversal_audit BEFORE INSERT ON ledger_audit WHEN NEW.reason LIKE 'reversing entry:%' BEGIN SELECT RAISE(ABORT,'injected audit failure'); END`)
	require.NoError(t, err)
	_, err = s.Reverse(ctx, p.Id, "injected failure", "operator")
	require.ErrorContains(t, err, "injected audit failure")
	var count int
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM postings WHERE reversal_of=?`, p.Id).Scan(&count))
	require.Zero(t, count)
	_, err = s.db.ExecContext(ctx, `DROP TRIGGER refuse_reversal_audit`)
	require.NoError(t, err)
	correction, err := s.Reverse(ctx, p.Id, "safe retry after rollback", "operator")
	require.NoError(t, err)
	require.EqualValues(t, -1250, correction.Event.AmountMinor)
	require.Len(t, correction.Audit, 1)
}
