package bootstrap_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"audio-tools/internal/bootstrap"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/storage"
)

func TestOpenDB_MigratesQualificationEvidenceBeforeSchemaDriftCheck(t *testing.T) {
	// Storage is isolated by redirecting the class-root tree rather than by
	// naming a database file, so the legacy fixture must be seeded at exactly
	// the path this scenario's own identity resolves to.
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
	path, err := storage.SQLitePath(storage.SQLiteConfig{Scenario: "audio-tools"})
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))

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

	db, _, err := bootstrap.OpenDB(context.Background(), bootstrap.Env{})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	row := db.QueryRowContext(context.Background(), `SELECT model_id FROM qualification_evidence LIMIT 1`)
	var model string
	require.ErrorIs(t, row.Scan(&model), sql.ErrNoRows, "the migrated column must be queryable even with no rows")
}
