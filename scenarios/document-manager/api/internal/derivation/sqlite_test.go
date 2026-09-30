package derivation

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestSQLiteStoreIsAppendOnly(t *testing.T) {
	db, err := sql.Open("sqlite", "file:derivation-test?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.ExecContext(context.Background(), Schema())
	require.NoError(t, err)
	store := NewSQLiteStore(db)
	version, err := store.NextVersion(context.Background(), "sha256-x")
	require.NoError(t, err)
	require.Equal(t, 1, version)
	require.NoError(t, store.Append(context.Background(), Result{DocumentHash: "sha256-x", Version: version, State: 1}))
	version, err = store.NextVersion(context.Background(), "sha256-x")
	require.NoError(t, err)
	require.Equal(t, 2, version)
	var count int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM derivation_versions`).Scan(&count))
	require.Equal(t, 1, count)
}
