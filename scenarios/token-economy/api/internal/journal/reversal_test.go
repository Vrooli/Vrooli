package journal_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/schedule"

	"token-economy/internal/journal"
)

// [REQ:TKE-P0-010] A correction is a compensating event; the original remains
// readable and the balance reflects both records.
func TestReversalCompensatesWithoutRewritingOriginal(t *testing.T) {
	repo, db := newJournalRepository(t)
	seedTokenType(t, db, "chores")
	ctx := context.Background()
	base := time.Date(2026, 8, 19, 15, 0, 0, 0, time.UTC)
	original := event("wrong-credit", "chores", "child:sam", 6, journal.EventKindCredit, "grant:mistake", base)
	_, err := repo.Append(ctx, original)
	require.NoError(t, err)
	reversal := event("reverse-credit", "chores", "child:sam", 6, journal.EventKindReversal, original.ID, base.Add(time.Second))
	reversal.Reason = "correct mistaken weekly grant"
	_, err = repo.Append(ctx, reversal)
	require.NoError(t, err)

	events, err := repo.Read(ctx, "child:sam", "chores")
	require.NoError(t, err)
	require.Equal(t, []journal.Event{original, reversal}, events)
	balance, err := repo.BalanceAt(ctx, "child:sam", "chores")
	require.NoError(t, err)
	require.Zero(t, balance.Amount)
}

// [REQ:TKE-P0-010] The one minter reversal operation compensates mint, grant
// credit, and redemption debit events, replays its key, and refuses a second
// correction under a distinct key.
func TestReverseMintGrantAndRedemptionExactlyOnce(t *testing.T) {
	for _, originalKind := range []journal.EventKind{journal.EventKindMint, journal.EventKindCredit, journal.EventKindDebit} {
		t.Run(string(originalKind), func(t *testing.T) {
			repo, db := newJournalRepository(t)
			seedTokenType(t, db, "chores")
			base := time.Date(2026, 8, 19, 18, 0, 0, 0, time.UTC)
			original := event("original-"+string(originalKind), "chores", "child:sam", 4, originalKind, "source:"+string(originalKind), base)
			_, err := repo.Append(context.Background(), original)
			require.NoError(t, err)
			before, err := repo.BalanceAt(context.Background(), "child:sam", "chores")
			require.NoError(t, err)

			service := journal.NewService(repo, schedule.NewFake(base.Add(time.Second)))
			input := journal.ReverseInput{
				OriginalEventID: original.ID, Reason: "operator correction",
				IdempotencyKey: "reverse:" + string(originalKind), ActorIdentity: "parent:alex",
			}
			first, err := service.Reverse(context.Background(), input)
			require.NoError(t, err)
			second, err := service.Reverse(context.Background(), input)
			require.NoError(t, err)
			require.Equal(t, first, second)
			require.Equal(t, original.ID, first.CauseReference)
			require.Equal(t, "operator correction", first.Reason)

			after, err := repo.BalanceAt(context.Background(), "child:sam", "chores")
			require.NoError(t, err)
			require.Zero(t, after.Amount, "reversal must restore the balance before the original event (which was %d after it)", before.Amount)
			events, err := repo.Read(context.Background(), "child:sam", "chores")
			require.NoError(t, err)
			require.Equal(t, []journal.Event{original, first}, events)

			_, err = service.Reverse(context.Background(), journal.ReverseInput{
				OriginalEventID: original.ID, Reason: "second correction",
				IdempotencyKey: "other:" + string(originalKind), ActorIdentity: "parent:alex",
			})
			require.ErrorIs(t, err, journal.ErrEventAlreadyReversed)
		})
	}
}

func TestReverseRequiresReasonAndIdempotencyKey(t *testing.T) {
	repo, _ := newJournalRepository(t)
	service := journal.NewService(repo, schedule.NewFake(time.Now()))
	for _, input := range []journal.ReverseInput{
		{Reason: "reason", IdempotencyKey: "key"},
		{OriginalEventID: "event", IdempotencyKey: "key"},
		{OriginalEventID: "event", Reason: "reason"},
	} {
		_, err := service.Reverse(context.Background(), input)
		var invalid *journal.InvalidReversalError
		require.ErrorAs(t, err, &invalid)
	}
}
