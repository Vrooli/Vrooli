package ledger

import (
	"testing"

	"github.com/stretchr/testify/require"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestIngestAuditFailureDoesNotLeaveAnAcceptedPosting(t *testing.T) {
	s, ctx := testStore(t)
	book, err := s.CreateBook(ctx, "Atomic acceptance fixture", "USD")
	require.NoError(t, err)
	account, err := s.CreateAccount(ctx, book.Id, "Cash", ledgerpb.AccountKind_ASSET)
	require.NoError(t, err)
	event := &sharedpb.MoneyEvent{ExternalId: "audit-failure-fixture", AdapterId: "fixture", AccountId: account.Id, BookId: book.Id, AmountMinor: 1250, Currency: "USD", OccurredAt: timestamppb.Now(), Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED}
	_, err = s.db.ExecContext(ctx, `CREATE TRIGGER refuse_original_audit BEFORE INSERT ON ledger_audit WHEN NEW.reason='ingested money event' BEGIN SELECT RAISE(ABORT,'injected original audit failure'); END`)
	require.NoError(t, err)
	_, _, err = s.Ingest(ctx, event, "test")
	require.ErrorContains(t, err, "injected original audit failure")
	var count int
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM postings WHERE external_id=?`, event.ExternalId).Scan(&count))
	require.Zero(t, count, "a failed acceptance must not leave an unaudited posting")
	_, err = s.db.ExecContext(ctx, `DROP TRIGGER refuse_original_audit`)
	require.NoError(t, err)
	accepted, duplicate, err := s.Ingest(ctx, event, "test")
	require.NoError(t, err)
	require.False(t, duplicate)
	require.Len(t, accepted.Audit, 1)
	again, duplicate, err := s.Ingest(ctx, event, "test")
	require.NoError(t, err)
	require.True(t, duplicate)
	require.Equal(t, accepted.Id, again.Id)
	require.Len(t, again.Audit, 1)
}
