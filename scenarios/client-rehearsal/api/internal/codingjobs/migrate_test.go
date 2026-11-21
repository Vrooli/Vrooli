package codingjobs_test

import (
	"context"
	"testing"
	"time"

	"client-rehearsal/internal/codingjobs"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
)

func TestEnsureReplayKeySchemaUpgradesExistingCodingJobsTable(t *testing.T) {
	db := dbtest.NewSQLite(t)
	_, err := db.ExecContext(context.Background(), `CREATE TABLE coding_jobs (
		id TEXT PRIMARY KEY, case_id TEXT NOT NULL, project TEXT NOT NULL, title TEXT NOT NULL,
		task TEXT NOT NULL, state TEXT NOT NULL, source_seal_ref TEXT NOT NULL,
		solver_run_ref TEXT NOT NULL DEFAULT '', evaluator_receipt_ref TEXT NOT NULL DEFAULT '',
		evaluator_receipt_sha256 TEXT NOT NULL DEFAULT '', score TEXT NOT NULL DEFAULT '',
		revision INTEGER NOT NULL DEFAULT 1, events_json TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL
	)`)
	require.NoError(t, err)
	require.NoError(t, codingjobs.EnsureReplayKeySchema(context.Background(), db))
	require.NoError(t, codingjobs.EnsureReplayKeySchema(context.Background(), db), "schema upgrade must be repeatable")

	repo := codingjobs.NewSQLiteRepository(db)
	service := codingjobs.NewService(repo, func() time.Time { return time.Now().UTC() })
	job, err := service.Create(context.Background(), codingjobs.CreateInput{ReplayKey: "case:retry", CaseID: "case", Project: "org/repo", Title: "Fix", Task: "Repair", SourceSealRef: "seal:source"})
	require.NoError(t, err)
	loaded, err := repo.GetByReplayKey(context.Background(), "case:retry")
	require.NoError(t, err)
	require.Equal(t, job.ID, loaded.ID)

	// The production schema provider remains safe to run after the upgrade.
	require.NoError(t, database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(codingjobs.Schema)))
}
