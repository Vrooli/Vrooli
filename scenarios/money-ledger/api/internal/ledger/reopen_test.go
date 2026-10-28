package ledger

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func reopenStore(t *testing.T, path string) (*Store, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(3000)&_pragma=foreign_keys(1)")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(func() string { return (&Store{}).Schema() })))
	return NewStore(database.NewFromPrimary(db), func() time.Time { return time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC) }), db
}

func fileCorrection(t *testing.T, s *Store) *sharedpb.Posting {
	t.Helper()
	ctx := context.Background()
	b, err := s.CreateBook(ctx, "Reopen fixture", "USD")
	require.NoError(t, err)
	a, err := s.CreateAccount(ctx, b.Id, "Cash", ledgerpb.AccountKind_ASSET)
	require.NoError(t, err)
	p, _, err := s.Ingest(ctx, &sharedpb.MoneyEvent{ExternalId: "fixture", AdapterId: "file-recovery", AccountId: a.Id, BookId: b.Id, AmountMinor: 1250, Currency: "USD", OccurredAt: timestamppb.Now(), Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED}, "test")
	require.NoError(t, err)
	return p
}

func TestReversalLostReplyRecoversAfterDatabaseCloseAndReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.sqlite")
	s, db := reopenStore(t, path)
	p := fileCorrection(t, s)
	ctx := context.Background()
	accepted, err := s.Reverse(ctx, p.Id, "removed fixture occurrence", "test")
	require.NoError(t, err)
	require.NoError(t, db.Close())
	fresh, _ := reopenStore(t, path)
	retry, err := fresh.Reverse(ctx, p.Id, "lost reply retry", "test")
	require.NoError(t, err)
	require.Equal(t, accepted.Id, retry.Id)
	require.EqualValues(t, -1250, retry.Event.AmountMinor)
	require.Len(t, retry.Audit, 1)
	position, err := fresh.Position(ctx, p.Event.BookId)
	require.NoError(t, err)
	require.Zero(t, position.CashMinor)
}

func TestConcurrentReversalsAcrossIndependentConnectionsHaveOneCorrection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "concurrent.sqlite")
	a, _ := reopenStore(t, path)
	p := fileCorrection(t, a)
	b, _ := reopenStore(t, path)
	ctx := context.Background()
	start := make(chan struct{})
	var wg sync.WaitGroup
	ids := make(chan string, 2)
	errs := make(chan error, 2)
	for _, s := range []*Store{a, b} {
		wg.Add(1)
		go func(s *Store) {
			defer wg.Done()
			<-start
			r, err := s.Reverse(ctx, p.Id, "concurrent fixture removal", "test")
			errs <- err
			if r != nil {
				ids <- r.Id
			}
		}(s)
	}
	close(start)
	wg.Wait()
	close(errs)
	close(ids)
	for err := range errs {
		require.NoError(t, err)
	}
	var same string
	for id := range ids {
		if same == "" {
			same = id
		}
		require.Equal(t, same, id)
	}
	var count int
	require.NoError(t, a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM postings WHERE reversal_of=?`, p.Id).Scan(&count))
	require.Equal(t, 1, count)
}

func TestOriginalAcceptanceLostReplyRecoversAfterDatabaseCloseAndReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "original-reopen.sqlite")
	s, db := reopenStore(t, path)
	accepted := fileCorrection(t, s)
	event := proto.Clone(accepted.Event).(*sharedpb.MoneyEvent)
	require.NoError(t, db.Close())
	fresh, _ := reopenStore(t, path)
	replayed, duplicate, err := fresh.Ingest(context.Background(), event, "lost-reply retry")
	require.NoError(t, err)
	require.True(t, duplicate)
	require.Equal(t, accepted.Id, replayed.Id)
	require.EqualValues(t, 1250, replayed.Event.AmountMinor)
	require.Len(t, replayed.Audit, 1)
	position, err := fresh.Position(context.Background(), accepted.Event.BookId)
	require.NoError(t, err)
	require.EqualValues(t, 1250, position.CashMinor)
}

func TestConcurrentOriginalAcceptanceHasOnePostingAndAudit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "original-concurrent.sqlite")
	a, _ := reopenStore(t, path)
	fixture := fileCorrection(t, a)
	b, _ := reopenStore(t, path)
	event := proto.Clone(fixture.Event).(*sharedpb.MoneyEvent)
	event.ExternalId = "concurrent-original"
	ctx := context.Background()
	start := make(chan struct{})
	type outcome struct {
		posting   *sharedpb.Posting
		duplicate bool
		err       error
	}
	results := make(chan outcome, 2)
	var wg sync.WaitGroup
	for _, s := range []*Store{a, b} {
		wg.Add(1)
		go func(s *Store) {
			defer wg.Done()
			<-start
			p, duplicate, err := s.Ingest(ctx, proto.Clone(event).(*sharedpb.MoneyEvent), "concurrent acceptance")
			results <- outcome{p, duplicate, err}
		}(s)
	}
	close(start)
	wg.Wait()
	close(results)
	var canonical string
	duplicates := 0
	for result := range results {
		require.NoError(t, result.err)
		require.NotNil(t, result.posting)
		require.Len(t, result.posting.Audit, 1)
		require.EqualValues(t, 1250, result.posting.Event.AmountMinor)
		if canonical == "" {
			canonical = result.posting.Id
		}
		require.Equal(t, canonical, result.posting.Id)
		if result.duplicate {
			duplicates++
		}
	}
	require.Equal(t, 1, duplicates)
	var postings, audits int
	require.NoError(t, a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM postings WHERE adapter_id=? AND external_id=?`, event.AdapterId, event.ExternalId).Scan(&postings))
	require.NoError(t, a.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ledger_audit WHERE entity_type='posting' AND entity_id=?`, canonical).Scan(&audits))
	require.Equal(t, 1, postings)
	require.Equal(t, 1, audits)
}
