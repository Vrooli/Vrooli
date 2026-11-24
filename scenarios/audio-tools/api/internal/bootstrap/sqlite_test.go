package bootstrap_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"audio-tools/internal/bootstrap"
	"audio-tools/internal/modules"

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

	row := db.QueryRowContext(context.Background(), `SELECT model_id, provider_tier FROM qualification_evidence LIMIT 1`)
	var model, providerTier string
	require.ErrorIs(t, row.Scan(&model, &providerTier), sql.ErrNoRows, "the migrated columns must be queryable even with no rows")
}

func TestOpenDB_UsesVariantSchemas(t *testing.T) {
	for _, tc := range []struct {
		name       string
		lab        bool
		wantCorpus bool
		wantExp    bool
	}{
		{name: "product"},
		{name: "lab", lab: true, wantCorpus: true, wantExp: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
			db, _, err := bootstrap.OpenDB(context.Background(), bootstrap.Env{EnableLabAPI: tc.lab})
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, db.Close()) })

			for table, want := range map[string]bool{
				"corpus_clips":           tc.wantCorpus && modules.IsLabBuild(),
				"experiments":            tc.wantExp && modules.IsLabBuild(),
				"qualification_evidence": true,
			} {
				var found bool
				err := db.QueryRowContext(context.Background(), `SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name=?)`, table).Scan(&found)
				require.NoError(t, err)
				require.Equalf(t, want, found, "table %s for %s variant", table, tc.name)
			}
		})
	}
}

func TestOpenDB_ProductVariantPreservesExistingLabTables(t *testing.T) {
	if !modules.IsLabBuild() {
		t.Skip("lab schema registration is only compiled into the audio_lab build")
	}
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
	labDB, _, err := bootstrap.OpenDB(context.Background(), bootstrap.Env{EnableLabAPI: true})
	require.NoError(t, err)
	_, err = labDB.Primary().ExecContext(context.Background(), `INSERT INTO corpus_clips (id, blob_key, created_at) VALUES ('clip-preserved', 'blob-preserved', '2026-11-24T00:00:00Z')`)
	require.NoError(t, err)
	_, err = labDB.Primary().ExecContext(context.Background(), `INSERT INTO experiments (id, status, created_at) VALUES ('experiment-preserved', 'queued', '2026-11-24T00:00:00Z')`)
	require.NoError(t, err)
	_, err = labDB.Primary().ExecContext(context.Background(), `INSERT INTO experiment_runs (id, experiment_id, strategy, created_at) VALUES ('run-preserved', 'experiment-preserved', 'baseline', '2026-11-24T00:00:00Z')`)
	require.NoError(t, err)
	require.NoError(t, labDB.Close())

	productDB, _, err := bootstrap.OpenDB(context.Background(), bootstrap.Env{EnableLabAPI: false})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, productDB.Close()) })
	for _, table := range []string{"corpus_clips", "experiments", "experiment_runs"} {
		var found bool
		err := productDB.QueryRowContext(context.Background(), `SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name=?)`, table).Scan(&found)
		require.NoError(t, err)
		require.Truef(t, found, "switching to product mode must preserve existing lab table %s", table)
	}
	var clipID, blobKey, experimentID, runID string
	require.NoError(t, productDB.QueryRowContext(context.Background(), `SELECT id, blob_key FROM corpus_clips WHERE id='clip-preserved'`).Scan(&clipID, &blobKey))
	require.Equal(t, "clip-preserved", clipID)
	require.Equal(t, "blob-preserved", blobKey)
	require.NoError(t, productDB.QueryRowContext(context.Background(), `SELECT e.id, r.id FROM experiments e JOIN experiment_runs r ON r.experiment_id=e.id WHERE e.id='experiment-preserved'`).Scan(&experimentID, &runID))
	require.Equal(t, "experiment-preserved", experimentID)
	require.Equal(t, "run-preserved", runID)
}
