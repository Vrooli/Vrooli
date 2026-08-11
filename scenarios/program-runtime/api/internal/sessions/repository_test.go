package sessions

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	"program-runtime/internal/testutil/db"
)

func newSessionTestDB(t *testing.T) *sql.DB {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)))
	return d
}

func TestSQLiteRepositoryPersistsSessionAndGrants(t *testing.T) { // [REQ:PRT-P2-003]
	ctx := context.Background()
	d := newSessionTestDB(t)
	repo := NewRepository(d)
	want := &Session{
		ID:               "sess_persisted",
		Name:             "investigation",
		State:            "running",
		CreatedAt:        time.Date(2026, 8, 11, 12, 0, 0, 123, time.UTC),
		LastActivityAt:   time.Date(2026, 8, 11, 12, 1, 0, 456, time.UTC),
		SandboxWorkspace: "/workspace/investigation",
		Grants:           grantSet([]string{"network:internal", "filesystem:workspace"}),
	}
	require.NoError(t, repo.Create(ctx, want))

	// Rebuild the repository around the same durable handle to model a
	// process restart; no Manager or in-memory kernel state is reused.
	restarted := NewRepository(d)
	got, err := restarted.Get(ctx, want.ID)
	require.NoError(t, err)
	require.Equal(t, want.ID, got.ID)
	require.Equal(t, want.Name, got.Name)
	require.Equal(t, want.SandboxWorkspace, got.SandboxWorkspace)
	require.Equal(t, want.CreatedAt, got.CreatedAt)
	require.Equal(t, want.LastActivityAt, got.LastActivityAt)
	require.Equal(t, want.Grants, got.Grants)

	has, err := restarted.HasGrant(ctx, want.ID, "network:internal")
	require.NoError(t, err)
	require.True(t, has)
}

func TestSQLiteRepositoryRecordsReclamationReason(t *testing.T) { // [REQ:PRT-P1-005]
	ctx := context.Background()
	d := newSessionTestDB(t)
	repo := NewRepository(d)
	now := time.Date(2026, 8, 11, 13, 0, 0, 0, time.UTC)
	require.NoError(t, repo.Create(ctx, &Session{
		ID:             "sess_reclaim",
		State:          "running",
		CreatedAt:      now.Add(-time.Hour),
		LastActivityAt: now.Add(-time.Hour),
		Grants:         map[string]struct{}{},
	}))
	require.NoError(t, repo.Reclaim(ctx, "sess_reclaim", "idle timeout exceeded", now))
	_, err := repo.Get(ctx, "sess_reclaim")
	require.ErrorIs(t, err, ErrNotFound)

	var reason, reclaimedAt string
	require.NoError(t, d.QueryRowContext(ctx, `SELECT reason, reclaimed_at FROM reclamation_reasons WHERE session_id = ?`, "sess_reclaim").Scan(&reason, &reclaimedAt))
	require.Equal(t, "idle timeout exceeded", reason)
	require.Equal(t, formatTime(now), reclaimedAt)
}

func TestSQLiteRepositoryReclaimMissingSessionDoesNotWriteReason(t *testing.T) {
	ctx := context.Background()
	d := newSessionTestDB(t)
	repo := NewRepository(d)
	require.ErrorIs(t, repo.Reclaim(ctx, "missing", "not real", time.Now().UTC()), ErrNotFound)
	var count int
	require.NoError(t, d.QueryRowContext(ctx, `SELECT COUNT(*) FROM reclamation_reasons`).Scan(&count))
	require.Equal(t, 0, count)
}

func TestSQLiteRepositoryRejectsMalformedTimestamp(t *testing.T) {
	ctx := context.Background()
	d := newSessionTestDB(t)
	_, err := d.ExecContext(ctx, `INSERT INTO sessions (id, state, created_at, last_activity_at) VALUES (?, ?, ?, ?)`, "sess_bad_time", "running", "not-a-time", "not-a-time")
	require.NoError(t, err)
	_, err = NewRepository(d).Get(ctx, "sess_bad_time")
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrNotFound))
}
