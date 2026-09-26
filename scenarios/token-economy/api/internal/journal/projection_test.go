package journal_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"token-economy/internal/journal"
)

// [REQ:TKE-P0-004] A corrupted or missing cache never overrides event truth,
// and a full rebuild reproduces every replay-derived balance.
func TestProjectionRebuildEqualsEventReplayForEveryHolderAndType(t *testing.T) {
	repo, db := newJournalRepository(t)
	seedTokenType(t, db, "chores")
	seedTokenType(t, db, "reading")
	ctx := context.Background()
	base := time.Date(2026, 8, 19, 14, 0, 0, 0, time.UTC)
	events := []journal.Event{
		event("sam-credit", "chores", "child:sam", 12, journal.EventKindCredit, "grant:sam", base),
		event("lee-credit", "chores", "child:lee", 8, journal.EventKindCredit, "grant:lee", base.Add(time.Second)),
		event("sam-debit", "chores", "child:sam", 5, journal.EventKindDebit, "redemption:sam", base.Add(2*time.Second)),
		event("sam-reading", "reading", "child:sam", 4, journal.EventKindMint, "mint:reading", base.Add(3*time.Second)),
		event("lee-expiry", "chores", "child:lee", 3, journal.EventKindExpiry, "grant:lee", base.Add(4*time.Second)),
	}
	for _, value := range events {
		_, err := repo.Append(ctx, value)
		require.NoError(t, err)
	}

	cases := []struct {
		holderID, tokenTypeID string
		amount                int64
	}{
		{holderID: "child:sam", tokenTypeID: "chores", amount: 7},
		{holderID: "child:lee", tokenTypeID: "chores", amount: 5},
		{holderID: "child:sam", tokenTypeID: "reading", amount: 4},
	}
	for _, testCase := range cases {
		balance, balanceErr := repo.BalanceAt(ctx, testCase.holderID, testCase.tokenTypeID)
		require.NoError(t, balanceErr)
		require.Equal(t, testCase.amount, balance.Amount)
	}

	_, err := db.ExecContext(ctx, `UPDATE balance_projections SET amount = 999`)
	require.NoError(t, err)
	truth, err := repo.BalanceAt(ctx, "child:sam", "chores")
	require.NoError(t, err)
	require.Equal(t, int64(7), truth.Amount, "event replay must win over a corrupt cache")

	require.NoError(t, repo.Rebuild(ctx))
	rows, err := db.QueryContext(ctx, `SELECT holder_id, token_type_id, amount FROM balance_projections`)
	require.NoError(t, err)
	defer rows.Close()
	got := make(map[string]int64)
	want := make(map[string]int64, len(cases))
	for _, testCase := range cases {
		want[testCase.holderID+"/"+testCase.tokenTypeID] = testCase.amount
	}
	for rows.Next() {
		var holderID, tokenTypeID string
		var amount int64
		require.NoError(t, rows.Scan(&holderID, &tokenTypeID, &amount))
		got[holderID+"/"+tokenTypeID] = amount
	}
	require.NoError(t, rows.Err())
	require.Equal(t, want, got)
}
