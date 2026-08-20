package journal_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"

	"token-economy/internal/holders"
	"token-economy/internal/journal"
)

// [REQ:TKE-P0-006] The history repository applies subject ownership before it
// reads events. Foreign and absent holders expose the identical empty result.
// [REQ:TKE-P0-012] An owner receives every event in order and balances derived
// from that complete history.
func TestHolderHistoryRepositoryScopesAndProjectsCompleteHistory(t *testing.T) {
	events, db := newJournalRepository(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(holders.Schema)))
	holderRepository := holders.NewSQLiteRepository(db)
	createdAt := time.Date(2026, 8, 19, 13, 0, 0, 0, time.UTC)
	_, err := holderRepository.Create(context.Background(), holders.Holder{
		ID: "holder-sam", DisplayName: "Sam", AuthenticatorSubject: "auth:sam", CreatedAt: createdAt,
	})
	require.NoError(t, err)
	seedTokenType(t, db, "chores")

	first := event("grant-1", "chores", "holder-sam", 7, journal.EventKindCredit, "weekly chores", createdAt.Add(time.Minute))
	second := event("redeem-1", "chores", "holder-sam", 2, journal.EventKindDebit, "movie reward", createdAt.Add(2*time.Minute))
	_, err = events.Append(context.Background(), first)
	require.NoError(t, err)
	_, err = events.Append(context.Background(), second)
	require.NoError(t, err)

	repository := journal.NewHolderHistoryRepository(events, holderRepository)
	owned, err := repository.Read(context.Background(), "holder-sam", "auth:sam")
	require.NoError(t, err)
	require.Equal(t, []journal.Event{first, second}, owned.Events)
	require.Equal(t, []journal.Balance{{HolderID: "holder-sam", TokenTypeID: "chores", Amount: 5}}, owned.Balances)

	foreign, foreignErr := repository.Read(context.Background(), "holder-sam", "auth:lee")
	missing, missingErr := repository.Read(context.Background(), "holder-absent", "auth:lee")
	require.NoError(t, foreignErr)
	require.NoError(t, missingErr)
	require.Equal(t, missing, foreign)
	require.Empty(t, foreign.Events)
	require.Empty(t, foreign.Balances)
}
