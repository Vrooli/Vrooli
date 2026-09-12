package ledger

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func testStore(t *testing.T) (*Store, context.Context) {
	t.Helper()
	sqlDB := databasetest.NewSQLite(t)
	db := database.NewFromPrimary(sqlDB)
	ctx := context.Background()
	require.NoError(t, database.EnsureSchemas(ctx, sqlDB, database.SchemaProviderFunc(func() string { return (&Store{}).Schema() })))
	now := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	return NewStore(db, func() time.Time { return now }), ctx
}

func TestStoreIngestIsIdempotentAndReversalIsAppendOnly(t *testing.T) { // [REQ:JRNL-003] [REQ:JRNL-004] [REQ:CTR-004]
	s, ctx := testStore(t)
	b, err := s.CreateBook(ctx, "Operating", "USD")
	require.NoError(t, err)
	a, err := s.CreateAccount(ctx, b.Id, "Cash", ledgerpb.AccountKind_ASSET)
	require.NoError(t, err)
	e := &sharedpb.MoneyEvent{ExternalId: "sale-1", AdapterId: "manual", AccountId: a.Id, BookId: b.Id, AmountMinor: 1250, Currency: "USD", OccurredAt: timestamppb.Now(), Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED}
	p1, duplicate, err := s.Ingest(ctx, e, "operator")
	require.NoError(t, err)
	require.False(t, duplicate)
	p2, duplicate, err := s.Ingest(ctx, e, "operator")
	require.NoError(t, err)
	require.True(t, duplicate)
	require.Equal(t, p1.Id, p2.Id)
	reversal, err := s.Reverse(ctx, p1.Id, "wrong amount", "operator")
	require.NoError(t, err)
	require.EqualValues(t, -1250, reversal.Event.AmountMinor)
	require.Equal(t, p1.Id, reversal.ReversalOf)
	postings, err := s.ListPostings(ctx, a.Id, "", "", "", 100)
	require.NoError(t, err)
	require.Len(t, postings, 2)
	var reversalListed *sharedpb.Posting
	for _, posting := range postings {
		if posting.Id == reversal.Id {
			reversalListed = posting
		}
	}
	require.NotNil(t, reversalListed)
	require.Len(t, reversalListed.Audit, 1)
	require.Equal(t, "operator", reversalListed.Audit[0].Actor)
	require.NotEmpty(t, reversalListed.Audit[0].CreatedAt)
	require.Contains(t, reversalListed.Audit[0].Reason, "reversing entry")
	require.Equal(t, p1.Id, reversalListed.Audit[0].PriorValue)
	position, err := s.Position(ctx, b.Id)
	require.NoError(t, err)
	require.EqualValues(t, 0, position.CashMinor)
}

func TestStoreRejectsProjectedAndMissingOccurredAt(t *testing.T) { // [REQ:JRNL-003] [REQ:CTR-002]
	s, ctx := testStore(t)
	_, _, err := s.Ingest(ctx, &sharedpb.MoneyEvent{Basis: sharedpb.Basis_BASIS_PROJECTED}, "operator")
	require.ErrorContains(t, err, "basis")
	_, _, err = s.Ingest(ctx, &sharedpb.MoneyEvent{Basis: sharedpb.Basis_BASIS_AUTHORITATIVE}, "operator")
	require.ErrorContains(t, err, "occurred_at")
}

func TestTransferIsPairedAndAudited(t *testing.T) { // [REQ:JRNL-002] [REQ:JRNL-004]
	s, ctx := testStore(t)
	b, err := s.CreateBook(ctx, "Operating", "USD")
	require.NoError(t, err)
	from, err := s.CreateAccount(ctx, b.Id, "Checking", ledgerpb.AccountKind_ASSET)
	require.NoError(t, err)
	to, err := s.CreateAccount(ctx, b.Id, "Reserve", ledgerpb.AccountKind_ASSET)
	require.NoError(t, err)
	posts, err := s.Transfer(ctx, from.Id, to.Id, 250, "USD", "move-1", "reserve transfer", timestamppb.New(time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)), "operator")
	require.NoError(t, err)
	require.Len(t, posts, 2)
	var audits int
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ledger_audit WHERE entity_type='posting'`).Scan(&audits))
	require.Equal(t, 2, audits)
}

func TestStatementWithoutBoundsDoesNotInventAnEmptyOpeningBalance(t *testing.T) { // [REQ:JRNL-005] [REQ:POS-003]
	s, ctx := testStore(t)
	b, err := s.CreateBook(ctx, "Operating", "USD")
	require.NoError(t, err)
	a, err := s.CreateAccount(ctx, b.Id, "Cash", ledgerpb.AccountKind_ASSET)
	require.NoError(t, err)
	_, _, err = s.Ingest(ctx, &sharedpb.MoneyEvent{ExternalId: "opening", AdapterId: "manual", AccountId: a.Id, BookId: b.Id, AmountMinor: 500, Currency: "USD", OccurredAt: timestamppb.New(time.Date(2025, 12, 31, 10, 0, 0, 0, time.UTC)), Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED}, "operator")
	require.NoError(t, err)
	statement, err := s.Statement(ctx, b.Id, "", "")
	require.NoError(t, err)
	require.EqualValues(t, 0, statement.OpeningCashMinor)
	require.EqualValues(t, 500, statement.ClosingCashMinor)
	require.EqualValues(t, 500, statement.AssetsMinor)
}

func TestAccountsRemainScopedToTheirBook(t *testing.T) { // [REQ:JRNL-001]
	s, ctx := testStore(t)
	first, err := s.CreateBook(ctx, "Operating", "USD")
	require.NoError(t, err)
	second, err := s.CreateBook(ctx, "Personal", "USD")
	require.NoError(t, err)
	firstAccount, err := s.CreateAccount(ctx, first.Id, "Operating cash", ledgerpb.AccountKind_ASSET)
	require.NoError(t, err)
	secondAccount, err := s.CreateAccount(ctx, second.Id, "Personal cash", ledgerpb.AccountKind_ASSET)
	require.NoError(t, err)

	accounts, err := s.ListAccounts(ctx, first.Id)
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	require.Equal(t, firstAccount.Id, accounts[0].Id)
	require.NotEqual(t, secondAccount.Id, accounts[0].Id)

	_, _, err = s.Ingest(ctx, &sharedpb.MoneyEvent{
		ExternalId: "wrong-book", AdapterId: "manual", AccountId: firstAccount.Id, BookId: second.Id,
		AmountMinor: 1, Currency: "USD", OccurredAt: timestamppb.Now(), Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED,
	}, "operator")
	require.ErrorContains(t, err, "different book")
}

func TestPositionChangesForBackdatedPostingAndReportsBasis(t *testing.T) { // [REQ:JRNL-005] [REQ:POS-001]
	s, ctx := testStore(t)
	b, err := s.CreateBook(ctx, "Operating", "USD")
	require.NoError(t, err)
	a, err := s.CreateAccount(ctx, b.Id, "Cash", ledgerpb.AccountKind_ASSET)
	require.NoError(t, err)
	postAt := func(id string, amount int64, when time.Time) {
		t.Helper()
		_, _, ingestErr := s.Ingest(ctx, &sharedpb.MoneyEvent{
			ExternalId: id, AdapterId: "manual", AccountId: a.Id, BookId: b.Id,
			AmountMinor: amount, Currency: "USD", OccurredAt: timestamppb.New(when), Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED,
		}, "operator")
		require.NoError(t, ingestErr)
	}
	postAt("current", 100, time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC))
	position, err := s.Position(ctx, b.Id)
	require.NoError(t, err)
	require.EqualValues(t, 100, position.CashMinor)
	require.Len(t, position.Inputs, 1)
	require.Equal(t, sharedpb.Basis_BASIS_OPERATOR_ASSERTED, position.Inputs[0].Basis)
	postAt("backdated", 50, time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC))
	position, err = s.Position(ctx, b.Id)
	require.NoError(t, err)
	require.EqualValues(t, 150, position.CashMinor)
}

func TestGoalVerdictRequiresItsDeclaredSustainWindow(t *testing.T) { // [REQ:POS-002]
	s, ctx := testStore(t)
	b, err := s.CreateBook(ctx, "Operating", "USD")
	require.NoError(t, err)
	a, err := s.CreateAccount(ctx, b.Id, "Revenue", ledgerpb.AccountKind_REVENUE)
	require.NoError(t, err)
	for i, when := range []time.Time{
		time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
		time.Date(2025, 12, 1, 10, 0, 0, 0, time.UTC),
	} {
		_, _, err = s.Ingest(ctx, &sharedpb.MoneyEvent{
			ExternalId: "revenue-" + string(rune('a'+i)), AdapterId: "manual", AccountId: a.Id, BookId: b.Id,
			AmountMinor: 100, Currency: "USD", OccurredAt: timestamppb.New(when), Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED,
		}, "operator")
		require.NoError(t, err)
	}
	_, err = s.DeclareGoal(ctx, b.Id, &ledgerpb.Goal{Name: "two-month revenue", Metric: "revenue", Comparator: ">=", ThresholdMinor: 100, SustainPeriods: 2})
	require.NoError(t, err)
	verdicts, err := s.ListGoals(ctx, b.Id, false)
	require.NoError(t, err)
	require.Len(t, verdicts, 1)
	require.True(t, verdicts[0].Met)
	require.EqualValues(t, 2, verdicts[0].SustainedPeriods)
}

func TestUnavailableGoalDoesNotBecomeApassingZero(t *testing.T) { // [REQ:POS-002]
	s, ctx := testStore(t)
	b, err := s.CreateBook(ctx, "Operating", "USD")
	require.NoError(t, err)
	a, err := s.CreateAccount(ctx, b.Id, "Cash", ledgerpb.AccountKind_ASSET)
	require.NoError(t, err)
	_, _, err = s.Ingest(ctx, &sharedpb.MoneyEvent{
		ExternalId:  "cash-only",
		AdapterId:   "manual",
		AccountId:   a.Id,
		BookId:      b.Id,
		AmountMinor: 1000,
		Currency:    "USD",
		OccurredAt:  timestamppb.New(time.Now().UTC()),
		FetchedAt:   timestamppb.New(time.Now().UTC()),
		Basis:       sharedpb.Basis_BASIS_OPERATOR_ASSERTED,
	}, "operator")
	require.NoError(t, err)
	_, err = s.DeclareGoal(ctx, b.Id, &ledgerpb.Goal{Name: "runway-floor", Metric: "runway", Comparator: ">=", ThresholdMinor: 3, SustainPeriods: 1})
	require.NoError(t, err)
	_, err = s.DeclareGoal(ctx, b.Id, &ledgerpb.Goal{Name: "default-alive", Metric: "revenue", Comparator: ">=", ComparandMetric: "burn", SustainPeriods: 3, BufferMultiple: 1.25})
	require.NoError(t, err)
	verdicts, err := s.ListGoals(ctx, b.Id, false)
	require.NoError(t, err)
	require.Len(t, verdicts, 2)
	for _, verdict := range verdicts {
		require.False(t, verdict.Met)
		require.Zero(t, verdict.SustainedPeriods)
		require.Contains(t, verdict.Explanation, "UNKNOWN:")
	}
}

func TestGoalSupportsRatioAndMetricComparandWithExplicitUnits(t *testing.T) { // [REQ:POS-002]
	s, ctx := testStore(t)
	b, err := s.CreateBook(ctx, "Operating", "USD")
	require.NoError(t, err)
	ratio, err := s.DeclareGoal(ctx, b.Id, &ledgerpb.Goal{Name: "services capacity", Metric: "services_capacity", Comparator: "<=", ThresholdRatio: 0.3, SustainPeriods: 3, SustainPeriodUnit: ledgerpb.SustainPeriodUnit_WEEK, BufferMultiple: 1})
	require.NoError(t, err)
	require.Equal(t, ledgerpb.SustainPeriodUnit_WEEK, ratio.SustainPeriodUnit)
	comparand, err := s.DeclareGoal(ctx, b.Id, &ledgerpb.Goal{Name: "services trap", Metric: "services_revenue", Comparator: ">", ComparandMetric: "subscription_revenue", SustainPeriods: 2, SustainPeriodUnit: ledgerpb.SustainPeriodUnit_MONTH, BufferMultiple: 1})
	require.NoError(t, err)
	require.Equal(t, "subscription_revenue", comparand.ComparandMetric)
	verdicts, err := s.ListGoals(ctx, b.Id, false)
	require.NoError(t, err)
	require.Len(t, verdicts, 2)
}

func TestJournalWriteBoundaryHasOnePostingWriter(t *testing.T) { // [REQ:CTR-001] [REQ:CTR-005]
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	entries, err := os.ReadDir(filepath.Dir(file))
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || entry.Name() == "store.go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		contents, readErr := os.ReadFile(filepath.Join(filepath.Dir(file), entry.Name()))
		require.NoError(t, readErr)
		require.NotContains(t, string(contents), "INSERT INTO postings", "only ledger.Store may write postings")
	}
}

func TestArchivedBookIsExcludedFromPositionAndStillReadableById(t *testing.T) {
	s, ctx := testStore(t)
	b, err := s.CreateBook(ctx, "Drill", "USD")
	require.NoError(t, err)
	a, err := s.CreateAccount(ctx, b.Id, "Cash", ledgerpb.AccountKind_ASSET)
	require.NoError(t, err)
	p, _, err := s.Ingest(ctx, &sharedpb.MoneyEvent{ExternalId: "archive-proof", AdapterId: "manual", AccountId: a.Id, BookId: b.Id, AmountMinor: 100, Currency: "USD", OccurredAt: timestamppb.Now(), Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED}, "operator")
	require.NoError(t, err)
	_, err = s.ArchiveBook(ctx, b.Id, "operator")
	require.NoError(t, err)
	books, err := s.ListBooks(ctx, false)
	require.NoError(t, err)
	require.Empty(t, books)
	_, err = s.Position(ctx, b.Id)
	require.ErrorContains(t, err, "archived")
	postings, err := s.ListPostings(ctx, "", b.Id, "", "", 100)
	require.NoError(t, err)
	require.Len(t, postings, 1)
	require.Equal(t, p.Id, postings[0].Id)
}

func TestAccountKindRefusesUndeclaredValueAndListsAccepted(t *testing.T) {
	s, ctx := testStore(t)
	b, err := s.CreateBook(ctx, "Operating", "USD")
	require.NoError(t, err)
	_, err = s.CreateAccount(ctx, b.Id, "Savings", ledgerpb.AccountKind(99))
	require.Error(t, err)
	require.Contains(t, err.Error(), "ASSET, LIABILITY, REVENUE, EXPENSE, EQUITY")
}

func TestKindNormalizationNeverGuessesAnUnknownKind(t *testing.T) {
	s, ctx := testStore(t)
	b, err := s.CreateBook(ctx, "Operating", "USD")
	require.NoError(t, err)
	_, err = s.db.ExecContext(ctx, `INSERT INTO accounts(id,book_id,name,kind,created_at) VALUES(?,?,?,?,?)`, "legacy-asset", b.Id, "Legacy asset", "asset", time.Now().UTC().Format(time.RFC3339Nano))
	require.NoError(t, err)
	_, err = s.db.ExecContext(ctx, `INSERT INTO accounts(id,book_id,name,kind,created_at) VALUES(?,?,?,?,?)`, "legacy-unknown", b.Id, "Legacy unknown", "savings", time.Now().UTC().Format(time.RFC3339Nano))
	require.NoError(t, err)
	require.NoError(t, s.EnsureMigrations(ctx))
	var normalized, unknown string
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT kind FROM accounts WHERE id=?`, "legacy-asset").Scan(&normalized))
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT kind FROM accounts WHERE id=?`, "legacy-unknown").Scan(&unknown))
	require.Equal(t, "ASSET", normalized)
	require.Equal(t, "SAVINGS", unknown)
	var findings int
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM operator_input_findings WHERE path=?`, "account.kind:legacy-unknown").Scan(&findings))
	require.Equal(t, 1, findings)
}

func TestGoalReparentingPreservesSustainMetadata(t *testing.T) {
	s, ctx := testStore(t)
	from, err := s.CreateBook(ctx, "Drill", "USD")
	require.NoError(t, err)
	to, err := s.CreateBook(ctx, "Production", "USD")
	require.NoError(t, err)
	original, err := s.DeclareGoal(ctx, from.Id, &ledgerpb.Goal{Name: "default-alive", Metric: "revenue", Comparator: ">=", ThresholdMinor: 100, SustainPeriods: 3, BufferMultiple: 1.25, SustainPeriodUnit: ledgerpb.SustainPeriodUnit_MONTH})
	require.NoError(t, err)
	reparented, err := s.ReparentGoal(ctx, original.Id, to.Id, "operator")
	require.NoError(t, err)
	require.Equal(t, to.Id, reparented.BookId)
	require.Equal(t, original.Name, reparented.Name)
	require.Equal(t, original.Metric, reparented.Metric)
	require.Equal(t, original.Comparator, reparented.Comparator)
	require.Equal(t, original.ThresholdMinor, reparented.ThresholdMinor)
	require.Equal(t, original.SustainPeriods, reparented.SustainPeriods)
	require.Equal(t, original.BufferMultiple, reparented.BufferMultiple)
	require.Equal(t, original.SustainPeriodUnit, reparented.SustainPeriodUnit)
	var audits int
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ledger_audit WHERE entity_type='goal' AND entity_id=?`, original.Id).Scan(&audits))
	require.Equal(t, 1, audits)
}

// Position excludes archived books; an unfiltered journal read must agree with
// it, or the journal page shows retired drill data as the live ledger.
func TestUnfilteredJournalReadExcludesArchivedBooks(t *testing.T) {
	s, ctx := testStore(t)
	live, err := s.CreateBook(ctx, "Monetization Operating", "USD")
	require.NoError(t, err)
	liveAccount, err := s.CreateAccount(ctx, live.Id, "Operating Cash", ledgerpb.AccountKind_ASSET)
	require.NoError(t, err)
	drill, err := s.CreateBook(ctx, "Drill", "USD")
	require.NoError(t, err)
	drillAccount, err := s.CreateAccount(ctx, drill.Id, "Drill Cash", ledgerpb.AccountKind_ASSET)
	require.NoError(t, err)

	ingest := func(account, book, external string, amount int64) {
		_, _, err := s.Ingest(ctx, &sharedpb.MoneyEvent{
			ExternalId: external, AdapterId: "manual", AccountId: account, BookId: book,
			AmountMinor: amount, Currency: "USD", OccurredAt: timestamppb.Now(), FetchedAt: timestamppb.Now(),
			Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED,
		}, "operator")
		require.NoError(t, err)
	}
	ingest(liveAccount.Id, live.Id, "live-1", 100000)
	ingest(drillAccount.Id, drill.Id, "drill-1", 999)

	_, err = s.ArchiveBook(ctx, drill.Id, "operator")
	require.NoError(t, err)

	unfiltered, err := s.ListPostings(ctx, "", "", "", "", 100)
	require.NoError(t, err)
	require.Len(t, unfiltered, 1, "only the live book's postings are returned")
	require.Equal(t, live.Id, unfiltered[0].Event.BookId)

	// An explicitly named archived book stays readable for deliberate history.
	explicit, err := s.ListPostings(ctx, "", drill.Id, "", "", 100)
	require.NoError(t, err)
	require.Len(t, explicit, 1)
}
