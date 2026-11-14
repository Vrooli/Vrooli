package inference

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAttemptStateTransitionMatrix(t *testing.T) {
	states := []AttemptState{
		AttemptDispatching, AttemptPreDispatchFailed, AttemptUncertain, AttemptPending,
		AttemptSettled, AttemptReleased, AttemptOutputObserved,
	}
	allowed := map[[2]AttemptState]bool{
		{AttemptDispatching, AttemptPreDispatchFailed}: true,
		{AttemptDispatching, AttemptUncertain}:         true,
		{AttemptDispatching, AttemptPending}:           true,
		{AttemptDispatching, AttemptSettled}:           true,
		{AttemptDispatching, AttemptReleased}:          true,
		{AttemptDispatching, AttemptOutputObserved}:    true,
		{AttemptUncertain, AttemptUncertain}:           true,
		{AttemptUncertain, AttemptPending}:             true,
		{AttemptUncertain, AttemptSettled}:             true,
		{AttemptUncertain, AttemptReleased}:            true,
		{AttemptUncertain, AttemptOutputObserved}:      true,
		{AttemptPending, AttemptUncertain}:             true,
		{AttemptPending, AttemptPending}:               true,
		{AttemptPending, AttemptSettled}:               true,
		{AttemptPending, AttemptReleased}:              true,
		{AttemptPending, AttemptOutputObserved}:        true,
	}
	for _, from := range states {
		for _, to := range states {
			got := from.CanTransition(to)
			want := allowed[[2]AttemptState{from, to}]
			require.Equalf(t, want, got, "%s -> %s", from, to)
		}
	}
	require.False(t, AttemptState("unknown").CanTransition(AttemptPending))
	require.False(t, AttemptDispatching.CanTransition("unknown"))
}

func TestSQLAttemptLedgerRejectsTerminalRevivalAndUnknownState(t *testing.T) {
	ctx := context.Background()
	ledger := newTestAttemptLedger(t)
	const root = "f8551f34287c4ec49d4725a106e265e4"
	const digest = "attempt-state-test"
	const generation = "generation-one"
	record, err := ledger.Begin(ctx, root, digest, generation, 0, 0, "hosted", "metered")
	require.NoError(t, err)
	require.NoError(t, ledger.Set(ctx, root, record.Ordinal, AttemptUncertain, "pending", nil))
	credits := int64(7)
	require.NoError(t, ledger.Set(ctx, root, record.Ordinal, AttemptSettled, "settled", &credits))
	require.Error(t, ledger.Set(ctx, root, record.Ordinal, AttemptUncertain, "pending", nil), "settled accounting is terminal")
	require.Error(t, ledger.Set(ctx, root, record.Ordinal, AttemptState("unknown"), "", nil))

	snapshot, err := ledger.Lookup(ctx, root, digest)
	require.NoError(t, err)
	require.NotNil(t, snapshot.Attempt)
	require.Equal(t, AttemptSettled, snapshot.Attempt.State)
	require.Equal(t, int64(7), *snapshot.Attempt.ChargedCredits)
}

func TestSQLAttemptLedgerRejectsUnknownPersistedState(t *testing.T) {
	ctx := context.Background()
	ledger := newTestAttemptLedger(t)
	const root = "0c1e7d87086f4ffea48c76130cf31acb"
	record, err := ledger.Begin(ctx, root, "unknown-state-test", "generation-one", 0, 0, "hosted", "metered")
	require.NoError(t, err)
	_, err = ledger.db.ExecContext(ctx, `UPDATE ai_gateway_inference_attempts SET state=$1 WHERE root_operation_id=$2 AND attempt_ordinal=$3`, "unknown", root, record.Ordinal)
	require.NoError(t, err)
	_, err = ledger.Lookup(ctx, root, "unknown-state-test")
	require.ErrorContains(t, err, "unknown state")
}
