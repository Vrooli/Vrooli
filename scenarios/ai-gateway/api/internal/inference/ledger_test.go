package inference

import (
	"context"
	"database/sql"
	"errors"
	"testing"

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
