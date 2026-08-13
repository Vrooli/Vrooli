package retention

import (
	"context"
	"database/sql"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	internalbindings "program-runtime/internal/bindings"
	internalprograms "program-runtime/internal/programs"
	"program-runtime/internal/sessions"
	internaltelemetry "program-runtime/internal/telemetry"
)

func newRetentionDB(t *testing.T) *sql.DB {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(sessions.Schema),
		apidb.SchemaProviderFunc(internalprograms.Schema),
		apidb.SchemaProviderFunc(internalbindings.Schema),
		apidb.SchemaProviderFunc(internaltelemetry.Schema),
	))
	return d
}

func TestRetentionDeletesOnlyRowsPastDeclaredWindows(t *testing.T) {
	ctx := context.Background()
	d := newRetentionDB(t)
	old := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	_, err := d.ExecContext(ctx, `INSERT INTO programs (id, session_id, source, provenance, status, created_at) VALUES (?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?)`, "old", "s", "x", "1", "failed", old.Format(time.RFC3339Nano), "new", "s", "x", "1", "failed", now.Format(time.RFC3339Nano))
	require.NoError(t, err)
	_, err = d.ExecContext(ctx, `INSERT INTO refusals (session_id, binding_id, reason, occurred_at) VALUES (?, ?, ?, ?), (?, ?, ?, ?)`, "s", "b", "old", old.Format(time.RFC3339Nano), "s", "b", "new", now.Format(time.RFC3339Nano))
	require.NoError(t, err)
	_, err = d.ExecContext(ctx, `INSERT INTO event_outbox (event_id, kind, occurred_at, payload, next_attempt_at, state) VALUES (?, ?, ?, ?, ?, 'delivered')`, "event_old", 1, old.Format(time.RFC3339Nano), `{}`, now.Format(time.RFC3339Nano))
	require.NoError(t, err)

	w := New(Options{DB: d, Clock: func() time.Time { return now }, ProgramWindow: 24 * time.Hour, RefusalWindow: 24 * time.Hour, ReclaimWindow: 24 * time.Hour, Logger: log.Default()})
	result, err := w.RunOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), result.ProgramsDeleted)
	require.Equal(t, int64(1), result.RefusalsDeleted)
	require.Equal(t, int64(1), result.TelemetryDeleted)
	var count int
	require.NoError(t, d.QueryRowContext(ctx, `SELECT COUNT(*) FROM programs`).Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, d.QueryRowContext(ctx, `SELECT COUNT(*) FROM refusals`).Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, d.QueryRowContext(ctx, `SELECT COUNT(*) FROM event_outbox`).Scan(&count))
	require.Equal(t, 0, count)
}

func TestRetentionUsesAnIndependentDatabaseHandle(t *testing.T) {
	serving := newRetentionDB(t)
	retentionDB := newRetentionDB(t)
	require.NotSame(t, serving, retentionDB)
	w := New(Options{DB: retentionDB, Interval: time.Hour})
	require.NotNil(t, w)
}
