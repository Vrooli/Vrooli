package bootstrap_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"audio-tools/internal/bootstrap"
	"github.com/stretchr/testify/require"
)

func TestOpenDB_MigratesQualificationEvidenceBeforeSchemaDriftCheck(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	legacy, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	_, err = legacy.Exec(`CREATE TABLE qualification_evidence (
		id TEXT PRIMARY KEY,
		engine_id TEXT NOT NULL,
		strategy TEXT NOT NULL,
		policy_profile TEXT NOT NULL DEFAULT '',
		kind TEXT NOT NULL,
		fault_profile TEXT NOT NULL DEFAULT '',
		passed INTEGER NOT NULL,
		artifact_ref TEXT NOT NULL,
		notes TEXT NOT NULL DEFAULT '',
		machine_json TEXT NOT NULL DEFAULT '{}',
		observed_at TEXT NOT NULL
	)`)
	require.NoError(t, err)
	require.NoError(t, legacy.Close())

	db, _, err := bootstrap.OpenDB(context.Background(), bootstrap.Env{SqlitePath: path})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	row := db.QueryRowContext(context.Background(), `SELECT model_id FROM qualification_evidence LIMIT 1`)
	var model string
	require.ErrorIs(t, row.Scan(&model), sql.ErrNoRows, "the migrated column must be queryable even with no rows")
}
