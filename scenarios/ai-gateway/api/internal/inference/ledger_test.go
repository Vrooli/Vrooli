package inference

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	dbtest "github.com/vrooli/api-core/databasetest"
)

func TestSQLAttemptLedgerPersistsAcrossRestartOrdersRepairAndRejectsChangedDigest(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	_, err := db.ExecContext(ctx, AttemptSchema())
	require.NoError(t, err)
	var sequence int
	var name, path string
	require.NoError(t, db.QueryRowContext(ctx, `PRAGMA database_list`).Scan(&sequence, &name, &path))
	root := "4e4e5be93eae40108d011c8a2ff391d4"
	digest := "canonical-request-sha256"
	generation := "opaque-device-key-generation"
	first, err := NewSQLAttemptLedger(db).Begin(ctx, root, digest, generation, 0, 0, "lpbs", "metered")
	require.NoError(t, err)
	require.True(t, first.Created)
	require.NoError(t, NewSQLAttemptLedger(db).Set(ctx, root, first.Ordinal, AttemptUncertain, "pending", nil))

	// A new process/ledger instance reopens the same file-backed SQLite database.
	require.NoError(t, db.Close())
	reopened, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	reopened.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = reopened.Close() })
	ledger := NewSQLAttemptLedger(reopened)
	snapshot, err := ledger.Lookup(ctx, root, digest)
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	require.Equal(t, generation, snapshot.Generation)
	require.NotNil(t, snapshot.Attempt)
	require.Equal(t, first.Key, snapshot.Attempt.Key)
	require.Equal(t, AttemptUncertain, snapshot.Attempt.State)

	repair, err := ledger.Begin(ctx, root, digest, generation, 1, 0, "lpbs", "metered")
	require.NoError(t, err)
	require.True(t, repair.Created)
	require.Greater(t, repair.Ordinal, first.Ordinal)
	require.NotEqual(t, first.Key, repair.Key, "schema repair gets its own stable child idempotency key")

	_, err = ledger.Lookup(ctx, root, "changed-request-sha256")
	require.ErrorIs(t, err, ErrRootConflict)
	_, err = ledger.Begin(ctx, root, digest, "rotated-device-key-generation", 1, 0, "lpbs", "metered")
	require.ErrorIs(t, err, ErrRootConflict)
	_, err = ledger.Lookup(ctx, root, "")
	require.True(t, errors.Is(err, ErrRootConflict))
}

func TestPruneExpiredRetainsLedgerForLPBSReceiptValidityAndKeepsUncertainRoots(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	_, err := db.ExecContext(ctx, AttemptSchema())
	require.NoError(t, err)
	ledger := NewSQLAttemptLedger(db)
	now := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)

	begin := func(root string) AttemptRecord {
		record, beginErr := ledger.Begin(ctx, root, "digest-"+root, "generation", 0, 0, "lpbs", "metered")
		require.NoError(t, beginErr)
		return record
	}
	expired := begin("expired-terminal-root-000000000000")
	require.NoError(t, ledger.Set(ctx, "expired-terminal-root-000000000000", expired.Ordinal, AttemptSettled, "settled", nil))
	fresh := begin("fresh-terminal-root-000000000000000")
	require.NoError(t, ledger.Set(ctx, "fresh-terminal-root-000000000000000", fresh.Ordinal, AttemptSettled, "settled", nil))
	uncertainRoot := "expired-uncertain-root-0000000000000"
	uncertain := begin(uncertainRoot)
	require.NoError(t, ledger.Set(ctx, uncertainRoot, uncertain.Ordinal, AttemptUncertain, "pending", nil))

	old := now.Add(-HostedAttemptRetention - time.Minute).UTC().Format("2006-01-02 15:04:05")
	_, err = db.ExecContext(ctx, `UPDATE ai_gateway_inference_attempts SET updated_at=$1 WHERE root_operation_id IN ($2,$3)`, old, "expired-terminal-root-000000000000", uncertainRoot)
	require.NoError(t, err)
	removed, err := ledger.PruneExpired(ctx, now)
	require.NoError(t, err)
	require.EqualValues(t, 1, removed)
	require.Greater(t, HostedAttemptRetention, 30*24*time.Hour)

	snapshot, err := ledger.Lookup(ctx, "expired-terminal-root-000000000000", "digest-expired-terminal-root-000000000000")
	require.NoError(t, err)
	require.Nil(t, snapshot, "a terminal record is pruned only after the LPBS receipt-validity window")
	snapshot, err = ledger.Lookup(ctx, "fresh-terminal-root-000000000000000", "digest-fresh-terminal-root-000000000000000")
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	snapshot, err = ledger.Lookup(ctx, uncertainRoot, "digest-"+uncertainRoot)
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	require.Equal(t, AttemptUncertain, snapshot.Attempt.State, "uncertain attempts must never be pruned")
}
