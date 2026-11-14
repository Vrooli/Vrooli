package grants_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/api-core/schedule"

	"token-economy/internal/grants"
	"token-economy/internal/journal"
	"token-economy/internal/mints"
)

func newGrantStore(t *testing.T) (*sql.DB, grants.Repository, journal.Repository, mints.Repository) {
	t.Helper()
	db := dbtest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), db,
		database.SchemaProviderFunc(mints.Schema), database.SchemaProviderFunc(journal.Schema),
		database.SchemaProviderFunc(grants.Schema),
	))
	appendCredit := func(ctx context.Context, tx *sql.Tx, credit grants.Credit) error {
		_, err := journal.NewTransactionalAppender(tx).Append(ctx, journal.Event{
			ID: credit.ID, TokenTypeID: credit.TokenTypeID, HolderID: credit.HolderID,
			Amount: credit.Amount, Kind: journal.EventKindCredit,
			CauseReference: credit.CauseReference, ActorIdentity: credit.ActorIdentity, CreatedAt: credit.CreatedAt,
		})
		return err
	}
	return db, grants.NewSQLiteRepository(db, appendCredit), journal.NewSQLiteRepository(db), mints.NewSQLiteRepository(db)
}

// [REQ:TKE-P0-011] Agent and operator grants share the grant path but remain
// distinguishable in the immutable journal row.
func TestGrantJournalCreditDistinguishesAgentFromOperator(t *testing.T) {
	now := time.Date(2026, 8, 19, 19, 0, 0, 0, time.UTC)
	_, grantRepository, journalRepository, tokenTypes := newGrantStore(t)
	createTokenType(t, tokenTypes, "chores", now)
	service := grants.NewService(grantRepository, readTokenType(tokenTypes), grants.NewRuleEvaluator(), schedule.NewFake(now))

	agentContext := provenance.NewContext(context.Background(), provenance.Provenance{
		Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified,
		Subject: "agent:chore-bot", ProfileKey: "household/chore-bot", RunID: "run-agent-grant",
	})
	operatorContext := provenance.NewContext(context.Background(), provenance.Provenance{
		Actor: provenance.ActorOperator, VerificationStatus: provenance.VerificationAbsent, Subject: "parent:alex",
	})
	for index, item := range []struct {
		ctx context.Context
		key string
	}{
		{ctx: agentContext, key: "agent-grant"},
		{ctx: operatorContext, key: "operator-grant"},
	} {
		_, err := service.Create(item.ctx, grants.CreateInput{
			TokenTypeID: "chores", GrantSourceID: item.key, Authorizer: "parent:alex",
			HolderID: "child:sam", AmountMinor: int64(index + 1), ExpiresAt: now.Add(time.Hour),
			IdempotencyKey: item.key, ActorIdentity: "parent:alex",
		})
		require.NoError(t, err)
	}

	events, err := journalRepository.Read(context.Background(), "child:sam", "chores")
	require.NoError(t, err)
	require.Len(t, events, 2)
	byKind := map[string]journal.Event{}
	for _, event := range events {
		byKind[event.ActorKind] = event
	}
	agentEvent := byKind[journal.ActorKindAgent]
	require.Equal(t, journal.VerificationVerified, agentEvent.ActorVerificationStatus)
	require.Equal(t, "agent:chore-bot", agentEvent.ActorIdentity)
	require.Equal(t, "run-agent-grant", agentEvent.ActorRunID)
	operatorEvent := byKind[journal.ActorKindOperator]
	require.Equal(t, journal.VerificationAbsent, operatorEvent.ActorVerificationStatus)
	require.Equal(t, "parent:alex", operatorEvent.ActorIdentity)
	require.Empty(t, operatorEvent.ActorRunID)
}

func createTokenType(t *testing.T, repository mints.Repository, id string, at time.Time) {
	t.Helper()
	_, err := repository.Create(context.Background(), mints.TokenType{
		ID: id, Name: id, Symbol: "T", Color: "#2563eb", SupplyPolicy: mints.SupplyPolicyUnbounded,
		Authority: mints.MinterAuthority{TokenTypeID: id, Subject: "parent:alex"}, CreatedAt: at,
	})
	require.NoError(t, err)
}

func readTokenType(repository mints.Repository) grants.TokenTypeReader {
	return grants.TokenTypeReaderFunc(func(ctx context.Context, id string) (grants.TokenTypeState, error) {
		value, err := repository.Get(ctx, id)
		if err != nil {
			return grants.TokenTypeState{}, err
		}
		return grants.TokenTypeState{ID: value.ID, Retired: value.Retired}, nil
	})
}

// [REQ:TKE-P0-002] Idempotent grant creation writes exactly one grant and one
// credit, and the resulting holder balance is derived from that event.
func TestCreateGrantCommitsExactlyOneJournalCredit(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	db, grantRepository, journalRepository, tokenTypes := newGrantStore(t)
	createTokenType(t, tokenTypes, "chores", now)
	service := grants.NewService(grantRepository, readTokenType(tokenTypes), grants.NewRuleEvaluator(), schedule.NewFake(now))
	input := grants.CreateInput{
		TokenTypeID: "chores", GrantSourceID: "weekly", Authorizer: "parent:alex",
		HolderID: "child:sam", AmountMinor: 9, ExpiresAt: now.Add(24 * time.Hour),
		IdempotencyKey: "weekly:sam:1",
	}
	first, err := service.Create(context.Background(), input)
	require.NoError(t, err)
	second, err := service.Create(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, first, second)

	events, err := journalRepository.Read(context.Background(), "child:sam", "chores")
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "grant:"+first.ID, events[0].CauseReference)
	balance, err := journal.NewSQLiteRepository(db).BalanceAt(context.Background(), "child:sam", "chores")
	require.NoError(t, err)
	require.Equal(t, int64(9), balance.Amount)
}

// [REQ:TKE-P0-002] A journal failure rolls back the grant row, proving both
// writes share one transaction.
func TestGrantRollsBackWhenJournalAppendFails(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	_, grantRepository, journalRepository, tokenTypes := newGrantStore(t)
	createTokenType(t, tokenTypes, "chores", now)
	_, err := journalRepository.Append(context.Background(), journal.Event{
		ID: "duplicate-event", TokenTypeID: "chores", HolderID: "child:sam", Amount: 1,
		Kind: journal.EventKindCredit, CauseReference: "seed", CreatedAt: now,
	})
	require.NoError(t, err)
	grant := grants.Grant{
		ID: "grant-rollback", TokenTypeID: "chores", GrantSourceID: "weekly", Authorizer: "parent:alex",
		HolderID: "child:sam", AmountMinor: 5, ExpiresAt: now.Add(time.Hour), IssuedAt: now,
		Status: grants.GrantStatusLive, IdempotencyKey: "rollback-key", Rules: []grants.GrantRule{},
	}
	_, err = grantRepository.Create(context.Background(), grant, grants.Credit{
		ID: "duplicate-event", TokenTypeID: "chores", HolderID: "child:sam", Amount: 5,
		CauseReference: "grant:" + grant.ID, CreatedAt: now,
	})
	require.Error(t, err)
	_, getErr := grantRepository.Get(context.Background(), grant.ID)
	require.True(t, errors.Is(getErr, grants.ErrGrantNotFound))
}

func TestListAndRevokeGrantAreTypedAndIdempotent(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	_, repository, _, tokenTypes := newGrantStore(t)
	createTokenType(t, tokenTypes, "chores", now)
	service := grants.NewService(repository, readTokenType(tokenTypes), grants.NewRuleEvaluator(), schedule.NewFake(now))
	created, err := service.Create(context.Background(), grants.CreateInput{
		TokenTypeID: "chores", GrantSourceID: "weekly", Authorizer: "parent:alex",
		HolderID: "child:sam", AmountMinor: 9, ExpiresAt: now.Add(24 * time.Hour),
		IdempotencyKey: "grant-list-revoke",
	})
	require.NoError(t, err)

	live, err := service.List(context.Background(), "child:sam", "chores", false)
	require.NoError(t, err)
	require.Len(t, live, 1)
	require.Equal(t, created.ID, live[0].ID)

	revoked, err := service.Revoke(context.Background(), grants.RevokeInput{
		ID: created.ID, Reason: "issued in error", IdempotencyKey: "revoke-once",
	})
	require.NoError(t, err)
	require.Equal(t, grants.GrantStatusRevoked, revoked.Status)
	require.NotNil(t, revoked.CancelledAt)

	replayed, err := service.Revoke(context.Background(), grants.RevokeInput{
		ID: "ignored-on-replay", Reason: "changed", IdempotencyKey: "revoke-once",
	})
	require.NoError(t, err)
	require.Equal(t, revoked, replayed)
	live, err = service.List(context.Background(), "child:sam", "chores", false)
	require.NoError(t, err)
	require.Empty(t, live)
	all, err := service.List(context.Background(), "child:sam", "chores", true)
	require.NoError(t, err)
	require.Equal(t, []grants.Grant{revoked}, all)
}
