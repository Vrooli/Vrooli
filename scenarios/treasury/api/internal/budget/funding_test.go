package budget_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/storage"
	"treasury/internal/book"
	"treasury/internal/budget"
)

type fundingUsage struct{ used int64 }

func (u *fundingUsage) BudgetUsage(context.Context, string, time.Time, time.Time) (budget.Usage, error) {
	return budget.Usage{TotalMinor: u.used, PeriodMinor: u.used}, nil
}

// Receipt funding is an independent limit inside the operator policy. Replays
// and inclusion-only revisions cannot increase it; a removal can reduce it
// below spent value without granting negative or overflowed headroom.
func TestAcceptedReceiptFundingIsExactIdempotentAndSurvivesRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "treasury.sqlite")
	dsn, err := storage.SQLiteDSNAt(path, storage.SQLiteTuning{TxLock: "immediate"})
	require.NoError(t, err)
	open := func() *sql.DB {
		handle, err := sql.Open("sqlite", dsn)
		require.NoError(t, err)
		handle.SetMaxOpenConns(1)
		t.Cleanup(func() { _ = handle.Close() })
		return handle
	}
	db := open()
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(book.Schema), database.SchemaProviderFunc(budget.Schema)))
	now := time.Date(2026, 11, 8, 20, 0, 0, 0, time.UTC)
	clock := schedule.NewFake(now)
	_, err = book.NewService(book.NewSQLiteRepository(db), clock).Create(ctx, book.CreateInput{ID: "research", Name: "Research", BeneficiaryIdentity: "operator:local"})
	require.NoError(t, err)
	repository := budget.NewSQLiteRepository(db)
	usage := &fundingUsage{}
	service := budget.NewService(repository, clock, usage)
	_, err = service.Create(ctx, budget.Budget{ID: "run", BookID: "research", Currency: "USD", TotalCapMinor: 1000, PeriodicCapMinor: 1000, PerTransactionCapMinor: 1000, Period: time.Hour})
	require.NoError(t, err)
	require.NoError(t, repository.BindFunding(ctx, "run", "eip155:8453", "base-native-usdc", 6, now))
	assertHeadroom := func(atomic string, credited, available, receipts int64) {
		t.Helper()
		funding, err := repository.Funding(ctx, "run")
		require.NoError(t, err)
		require.Equal(t, atomic, funding.AtomicAmount)
		require.Equal(t, receipts, funding.ReceiptCount)
		headroom, err := service.Headroom(ctx, "run")
		require.NoError(t, err)
		require.True(t, headroom.FundingRequired)
		require.Equal(t, credited, headroom.FundingCreditedMinor)
		require.Equal(t, available, headroom.AvailableMinor)
		require.EqualValues(t, 1000, headroom.TotalCapMinor, "funding must not rewrite policy")
	}
	assertHeadroom("0", 0, 0, 0)
	credit := budget.FundingCredit{LedgerReceiptID: "receipt-1", PostingID: "posting-1", SourceCheckID: "check-1", BudgetID: "run", AtomicDelta: "9007199254740993000001", ValuationDeltaMinor: 701, ValuationCurrency: "USD", AcceptedAt: now}
	require.NoError(t, repository.CreditAccepted(ctx, credit))
	require.NoError(t, repository.CreditAccepted(ctx, credit))
	assertHeadroom("9007199254740993000001", 701, 701, 1)
	changed := credit
	changed.ValuationDeltaMinor++
	require.ErrorIs(t, repository.CreditAccepted(ctx, changed), budget.ErrInvalid)
	changed = credit
	changed.LedgerReceiptID, changed.PostingID, changed.SourceCheckID = "receipt-2", "posting-2", "check-2"
	changed.AtomicDelta, changed.ValuationDeltaMinor = "0", 0
	require.NoError(t, repository.CreditAccepted(ctx, changed))
	assertHeadroom("9007199254740993000001", 701, 701, 2)
	usage.used = 500
	assertHeadroom("9007199254740993000001", 701, 201, 2)
	removed := credit
	removed.LedgerReceiptID, removed.PostingID, removed.SourceCheckID = "receipt-3", "posting-3", "check-3"
	removed.AtomicDelta, removed.ValuationDeltaMinor = "-9007199254740993000001", -701
	require.NoError(t, repository.CreditAccepted(ctx, removed))
	assertHeadroom("0", 0, 0, 3)
	require.NoError(t, db.Close())
	db = open()
	repository = budget.NewSQLiteRepository(db)
	service = budget.NewService(repository, clock, usage)
	assertHeadroom("0", 0, 0, 3)
	require.NoError(t, repository.CreditAccepted(ctx, removed))
	assertHeadroom("0", 0, 0, 3)
}

func TestFundingRejectsAssetRebindingWrongCurrencyAndNoncanonicalQuantity(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(book.Schema), database.SchemaProviderFunc(budget.Schema)))
	now := time.Now().UTC()
	_, err := book.NewService(book.NewSQLiteRepository(db), schedule.NewFake(now)).Create(ctx, book.CreateInput{ID: "b", Name: "Book", BeneficiaryIdentity: "operator"})
	require.NoError(t, err)
	r := budget.NewSQLiteRepository(db)
	_, err = budget.NewService(r, schedule.NewFake(now)).Create(ctx, budget.Budget{ID: "budget", BookID: "b", Currency: "USD", TotalCapMinor: 100, PeriodicCapMinor: 100, PerTransactionCapMinor: 100, Period: time.Hour})
	require.NoError(t, err)
	require.NoError(t, r.BindFunding(ctx, "budget", "base", "USDC", 6, now))
	require.NoError(t, r.BindFunding(ctx, "budget", "base", "USDC", 6, now))
	require.ErrorIs(t, r.BindFunding(ctx, "budget", "ethereum", "USDC", 6, now), budget.ErrInvalid)
	credit := budget.FundingCredit{LedgerReceiptID: "r", PostingID: "p", SourceCheckID: "s", BudgetID: "budget", AtomicDelta: "1000001", ValuationDeltaMinor: 100, ValuationCurrency: "EUR", AcceptedAt: now}
	require.ErrorIs(t, r.CreditAccepted(ctx, credit), budget.ErrInvalid)
	credit.ValuationCurrency = "USD"
	for _, invalid := range []string{"1.1", "1e6", "+1", "01", "-0"} {
		credit.AtomicDelta = invalid
		require.ErrorIs(t, r.CreditAccepted(ctx, credit), budget.ErrInvalid)
	}
	funding, err := r.Funding(ctx, "budget")
	require.NoError(t, err)
	require.Zero(t, funding.ReceiptCount)
}
