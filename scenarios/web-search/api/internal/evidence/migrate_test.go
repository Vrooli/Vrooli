package evidence_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	testdb "github.com/vrooli/api-core/databasetest"

	"web-search/internal/evidence"
)

type failOnceExecDB struct {
	db interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
	}
	failure error
}

func (d *failOnceExecDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if d.failure != nil {
		err := d.failure
		d.failure = nil
		return nil, err
	}
	return d.db.ExecContext(ctx, query, args...)
}

func TestMigrateAddsProducerExecutionIdentityToExistingReceipts(t *testing.T) {
	db := testdb.NewSQLite(t)
	_, err := db.ExecContext(context.Background(), `CREATE TABLE evidence_receipts (receipt_id TEXT PRIMARY KEY, observation_id TEXT NOT NULL UNIQUE, url TEXT NOT NULL, retrieved_at TEXT NOT NULL, content_hash TEXT NOT NULL, artifact_id TEXT NOT NULL, extraction_revision TEXT NOT NULL, retention TEXT NOT NULL DEFAULT '', failure_code TEXT NOT NULL DEFAULT '')`)
	require.NoError(t, err)
	require.NoError(t, evidence.Migrate(context.Background(), db))
	// A lifecycle restart can invoke the additive migration again. SQLite's
	// duplicate-column response is treated as the idempotent success path.
	require.NoError(t, evidence.Migrate(context.Background(), db))
	_, err = db.ExecContext(context.Background(), `INSERT INTO evidence_receipts (receipt_id, observation_id, producer_execution_id, url, retrieved_at, content_hash, artifact_id, extraction_revision) VALUES ('r', 'o', 'exec', 'https://example.test', '2026-09-06T00:00:00Z', 'hash', 'artifact', 'extract-v1')`)
	require.NoError(t, err)
}

func TestMigrateResumesAfterInterruptedBoundary(t *testing.T) {
	db := testdb.NewSQLite(t)
	_, err := db.ExecContext(context.Background(), `CREATE TABLE evidence_receipts (receipt_id TEXT PRIMARY KEY, observation_id TEXT NOT NULL UNIQUE, url TEXT NOT NULL, retrieved_at TEXT NOT NULL, content_hash TEXT NOT NULL, artifact_id TEXT NOT NULL, extraction_revision TEXT NOT NULL, retention TEXT NOT NULL DEFAULT '', failure_code TEXT NOT NULL DEFAULT '')`)
	require.NoError(t, err)

	fault := &failOnceExecDB{db: db, failure: errors.New("simulated migration interruption")}
	require.Error(t, evidence.Migrate(context.Background(), fault))
	// A restart or retry must be able to resume the additive migration after a
	// failure before the schema change is committed.
	require.NoError(t, evidence.Migrate(context.Background(), fault))
	require.NoError(t, evidence.Migrate(context.Background(), fault))

	_, err = db.ExecContext(context.Background(), `INSERT INTO evidence_receipts (receipt_id, observation_id, producer_execution_id, url, retrieved_at, content_hash, artifact_id, extraction_revision) VALUES ('r-resume', 'o-resume', 'exec-resume', 'https://example.test', '2026-09-06T00:00:00Z', 'hash', 'artifact', 'extract-v1')`)
	require.NoError(t, err)
}
