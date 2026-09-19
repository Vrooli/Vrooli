package redemption_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"

	"token-economy/internal/catalog"
	"token-economy/internal/grants"
	"token-economy/internal/holders"
	"token-economy/internal/journal"
	"token-economy/internal/mints"
	"token-economy/internal/redemption"
)

type fixture struct {
	db      *sql.DB
	service redemption.Service
	catalog catalog.Service
	journal interface {
		Read(context.Context, string, string) ([]journal.Event, error)
		BalanceAt(context.Context, string, string) (journal.Balance, error)
	}
	grantID string
	now     time.Time
}

func newFixture(t *testing.T, inject redemption.FailureInjector, relay redemption.Relay) fixture {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	db := dbtest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db,
		database.SchemaProviderFunc(mints.Schema),
		database.SchemaProviderFunc(catalog.Schema),
		database.SchemaProviderFunc(holders.Schema),
		database.SchemaProviderFunc(journal.Schema),
		database.SchemaProviderFunc(grants.Schema),
		database.SchemaProviderFunc(redemption.Schema),
	))
	mintRepository := mints.NewSQLiteRepository(db)
	_, err := mintRepository.Create(ctx, mints.TokenType{
		ID: "chores", Name: "Chores", Symbol: "CT", Color: "#2563eb",
		SupplyPolicy: mints.SupplyPolicyUnbounded,
		Authority:    mints.MinterAuthority{TokenTypeID: "chores", Subject: "parent:alex"},
		CreatedAt:    now,
	})
	require.NoError(t, err)
	holderRepository := holders.NewSQLiteRepository(db)
	_, err = holderRepository.Create(ctx, holders.Holder{ID: "holder:sam", DisplayName: "Sam", AuthenticatorSubject: "auth:sam", CreatedAt: now})
	require.NoError(t, err)
	journalRepository := journal.NewSQLiteRepository(db)
	grantRepository := grants.NewSQLiteRepository(db, func(ctx context.Context, tx *sql.Tx, credit grants.Credit) error {
		_, appendErr := journal.NewTransactionalAppender(tx).Append(ctx, journal.Event{
			ID: credit.ID, TokenTypeID: credit.TokenTypeID, HolderID: credit.HolderID,
			Amount: credit.Amount, Kind: journal.EventKindCredit, CauseReference: credit.CauseReference,
			ActorIdentity: credit.ActorIdentity, CreatedAt: credit.CreatedAt,
		})
		return appendErr
	})
	grantService := grants.NewService(grantRepository, grants.TokenTypeReaderFunc(func(context.Context, string) (grants.TokenTypeState, error) {
		return grants.TokenTypeState{ID: "chores"}, nil
	}), grants.NewRuleEvaluator(), schedule.NewFake(now))
	grant, err := grantService.Create(ctx, grants.CreateInput{
		TokenTypeID: "chores", GrantSourceID: "weekly", Authorizer: "parent:alex",
		HolderID: "holder:sam", AmountMinor: 10, ExpiresAt: now.Add(24 * time.Hour),
		IdempotencyKey: "grant:sam", ActorIdentity: "parent:alex",
	})
	require.NoError(t, err)
	catalogService := catalog.NewService(catalog.NewSQLiteRepository(db), catalog.TokenTypeReaderFunc(func(context.Context, string) (catalog.TokenTypeState, error) {
		return catalog.TokenTypeState{ID: "chores"}, nil
	}), schedule.NewFake(now))
	reserve := func(ctx context.Context, tx *sql.Tx, id string, at time.Time) (redemption.CatalogEntry, error) {
		entry, reserveErr := catalog.ReserveInventory(ctx, tx, id, at)
		return redemption.CatalogEntry{ID: entry.ID, TokenTypeID: entry.TokenTypeID, CostAmount: entry.CostAmount, ApprovalPosture: redemption.ApprovalPosture(entry.ApprovalPosture)}, reserveErr
	}
	balance := func(ctx context.Context, tx *sql.Tx, holderID, tokenTypeID string) (int64, error) {
		value, balanceErr := journal.BalanceInTransaction(ctx, tx, holderID, tokenTypeID)
		return value.Amount, balanceErr
	}
	appendDebit := func(ctx context.Context, tx *sql.Tx, debit redemption.Debit) error {
		_, appendErr := journal.NewTransactionalAppender(tx).Append(ctx, journal.Event{
			ID: debit.ID, TokenTypeID: debit.TokenTypeID, HolderID: debit.HolderID,
			Amount: debit.Amount, Kind: journal.EventKindDebit, CauseReference: debit.CauseReference,
			ActorIdentity: debit.ActorIdentity, CreatedAt: debit.CreatedAt,
		})
		return appendErr
	}
	var repository redemption.Repository
	if inject == nil {
		repository = redemption.NewSQLiteRepository(db, reserve, catalog.ReleaseInventory, balance, appendDebit)
	} else {
		repository = redemption.NewSQLiteRepositoryWithFailureInjector(db, reserve, catalog.ReleaseInventory, balance, appendDebit, inject)
	}
	service := redemption.NewService(
		repository,
		redemption.HolderReaderFunc(func(ctx context.Context, subject string) (redemption.Holder, error) {
			holder, holderErr := holderRepository.GetBySubject(ctx, subject)
			return redemption.Holder{ID: holder.ID}, holderErr
		}),
		redemption.CatalogReaderFunc(func(ctx context.Context, id string) (redemption.CatalogEntry, error) {
			entry, entryErr := catalogService.RequireAvailable(ctx, id)
			return redemption.CatalogEntry{ID: entry.ID, TokenTypeID: entry.TokenTypeID, CostAmount: entry.CostAmount, ApprovalPosture: redemption.ApprovalPosture(entry.ApprovalPosture)}, entryErr
		}),
		redemption.GrantEvaluatorFunc(func(ctx context.Context, id, scope string, evidence []string, available, requested int64, at time.Time) (redemption.GrantEvaluation, error) {
			decision, decisionErr := grantService.EvaluateRedemption(ctx, id, grants.EvaluationRequest{CatalogScope: scope, Evidence: evidence, AvailableBalance: available, RequestedAmount: requested, Now: at})
			return redemption.GrantEvaluation{Allowed: decision.Allowed, Reason: decision.Reason}, decisionErr
		}),
		relay,
		schedule.NewFake(now),
	)
	return fixture{db: db, service: service, catalog: catalogService, journal: journalRepository, grantID: grant.ID, now: now}
}

func (f fixture) addEntry(t *testing.T, id string, cost int64, posture catalog.ApprovalPosture, quantity *int64) {
	t.Helper()
	_, err := f.catalog.Create(context.Background(), catalog.Input{
		ID: id, TokenTypeID: "chores", Title: id, Description: id,
		CostAmount: cost, ApprovalPosture: posture,
		Availability: catalog.Availability{RemainingQuantity: quantity},
	}, "catalog:"+id)
	require.NoError(t, err)
}

func (f fixture) request(entryID, key string) redemption.RequestInput {
	return redemption.RequestInput{AuthenticatedSubject: "auth:sam", CatalogEntryID: entryID, GrantID: f.grantID, IdempotencyKey: key}
}

// [REQ:TKE-P0-009] A repeated caller key returns the first committed outcome,
// while a distinct key produces an independent debit.
func TestImmediateSettlementIsExactlyOnce(t *testing.T) {
	f := newFixture(t, nil, nil)
	f.addEntry(t, "movie", 4, catalog.ApprovalPostureImmediate, nil)
	first, err := f.service.Request(context.Background(), f.request("movie", "redeem:movie:1"))
	require.NoError(t, err)
	require.Equal(t, redemption.StateSettled, first.State)
	replayed, err := f.service.Request(context.Background(), f.request("movie", "redeem:movie:1"))
	require.NoError(t, err)
	require.Equal(t, first.ID, replayed.ID)
	second, err := f.service.Request(context.Background(), f.request("movie", "redeem:movie:2"))
	require.NoError(t, err)
	require.NotEqual(t, first.ID, second.ID)
	balance, err := f.journal.BalanceAt(context.Background(), "holder:sam", "chores")
	require.NoError(t, err)
	require.Equal(t, int64(2), balance.Amount)
	events, err := f.journal.Read(context.Background(), "holder:sam", "chores")
	require.NoError(t, err)
	require.Len(t, events, 3)
}

// [REQ:TKE-P0-009] The holder-facing economy view can reload its own
// redemption states from durable storage instead of relying on UI memory.
func TestListForSubjectReturnsDurableHolderRedemptions(t *testing.T) {
	f := newFixture(t, nil, nil)
	f.addEntry(t, "day-trip", 4, catalog.ApprovalPostureRequiresApproval, nil)
	requested, err := f.service.Request(context.Background(), f.request("day-trip", "redeem:trip:list"))
	require.NoError(t, err)

	values, err := f.service.ListForSubject(context.Background(), "auth:sam")
	require.NoError(t, err)
	require.Len(t, values, 1)
	require.Equal(t, requested.ID, values[0].ID)
	require.Equal(t, redemption.StatePendingApproval, values[0].State)

	_, err = f.service.ListForSubject(context.Background(), "")
	require.Error(t, err)
}

// [REQ:TKE-P0-009] An induced failure immediately before the journal debit
// rolls back the redemption, reservation, and catalog stock together.
func TestSettlementFailureRollsBackEveryWrite(t *testing.T) {
	f := newFixture(t, func(stage string) error {
		if stage == "before_journal_append" {
			return errors.New("injected journal boundary failure")
		}
		return nil
	}, nil)
	quantity := int64(1)
	f.addEntry(t, "trip", 4, catalog.ApprovalPostureImmediate, &quantity)
	_, err := f.service.Request(context.Background(), f.request("trip", "redeem:trip"))
	require.ErrorContains(t, err, "injected journal boundary failure")
	var redemptions, reservations int
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM redemptions`).Scan(&redemptions))
	require.NoError(t, f.db.QueryRow(`SELECT COUNT(*) FROM reservations`).Scan(&reservations))
	require.Zero(t, redemptions)
	require.Zero(t, reservations)
	entry, err := f.catalog.Get(context.Background(), "trip")
	require.NoError(t, err)
	require.Equal(t, int64(1), *entry.Availability.RemainingQuantity)
	balance, err := f.journal.BalanceAt(context.Background(), "holder:sam", "chores")
	require.NoError(t, err)
	require.Equal(t, int64(10), balance.Amount)
}

type failingRelay struct{ calls int }

func (r *failingRelay) Pending(context.Context, redemption.Redemption) error {
	r.calls++
	return errors.New("notification-hub unavailable")
}

// [REQ:TKE-P0-013] Pending approval reserves spendable balance, denial
// releases it and inventory, and approval settles with the optional relay down.
func TestApprovalQueueWorksWithoutNotificationHub(t *testing.T) {
	relay := &failingRelay{}
	f := newFixture(t, nil, relay)
	quantity := int64(1)
	f.addEntry(t, "day-trip", 7, catalog.ApprovalPostureRequiresApproval, &quantity)
	f.addEntry(t, "book", 7, catalog.ApprovalPostureRequiresApproval, nil)
	first, err := f.service.Request(context.Background(), f.request("day-trip", "redeem:trip"))
	require.NoError(t, err)
	require.Equal(t, redemption.StatePendingApproval, first.State)
	require.Equal(t, 1, relay.calls)
	_, err = f.service.Request(context.Background(), f.request("book", "redeem:book:blocked"))
	require.ErrorIs(t, err, redemption.ErrGrantRefused)
	denied, err := f.service.Deny(context.Background(), redemption.DecisionInput{RedemptionID: first.ID, DeciderSubject: "parent:alex", Reason: "not this week", IdempotencyKey: "deny:trip"})
	require.NoError(t, err)
	require.Equal(t, redemption.StateDenied, denied.State)
	entry, err := f.catalog.Get(context.Background(), "day-trip")
	require.NoError(t, err)
	require.Equal(t, int64(1), *entry.Availability.RemainingQuantity)
	second, err := f.service.Request(context.Background(), f.request("book", "redeem:book"))
	require.NoError(t, err)
	approved, err := f.service.Approve(context.Background(), redemption.DecisionInput{RedemptionID: second.ID, DeciderSubject: "parent:alex", Reason: "earned it", IdempotencyKey: "approve:book"})
	require.NoError(t, err)
	require.Equal(t, redemption.StateSettled, approved.State)
	replayed, err := f.service.Approve(context.Background(), redemption.DecisionInput{RedemptionID: second.ID, DeciderSubject: "changed", Reason: "changed", IdempotencyKey: "approve:book"})
	require.NoError(t, err)
	require.Equal(t, approved, replayed)
	pending, err := f.service.ListPending(context.Background())
	require.NoError(t, err)
	require.Empty(t, pending)
	balance, err := f.journal.BalanceAt(context.Background(), "holder:sam", "chores")
	require.NoError(t, err)
	require.Equal(t, int64(3), balance.Amount)
}
