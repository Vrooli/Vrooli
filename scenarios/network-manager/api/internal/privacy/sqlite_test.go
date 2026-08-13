package privacy

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	testdb "github.com/vrooli/api-core/databasetest"
	localdb "network-manager/internal/database"
	"network-manager/internal/snapshot"
)

func newPrivacyDB(t *testing.T) (*sql.DB, *sqliteRepository) {
	t.Helper()
	db := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(snapshot.Schema),
		apidb.SchemaProviderFunc(Schema),
	))
	return db, NewSQLiteRepository(db).(*sqliteRepository)
}

func TestSQLiteRepositorySavesSettings(t *testing.T) {
	// [REQ:NM-P0-008] Privacy settings are persisted instead of being
	// scaffold-only responses.
	_, repo := newPrivacyDB(t)
	now := time.Date(2026, 6, 23, 21, 0, 0, 0, time.UTC)

	_, err := repo.SaveRetention(context.Background(), RetentionSettings{
		QueryLogDays:   3,
		SnapshotDays:   60,
		ExperimentDays: 90,
		Profile:        ProfileHomeExtended,
		UpdatedAt:      now,
	})
	require.NoError(t, err)

	retention, err := repo.GetRetention(context.Background())
	require.NoError(t, err)
	require.Equal(t, int32(3), retention.QueryLogDays)
	require.Equal(t, ProfileHomeExtended, retention.Profile)

	_, err = repo.SaveVisibility(context.Background(), VisibilitySettings{
		ShowQueryDomains:  false,
		ShowDeviceHistory: true,
		HouseholdMode:     true,
		Notes:             []string{"operator enabled device history"},
		UpdatedAt:         now,
	})
	require.NoError(t, err)
	visibility, err := repo.GetVisibility(context.Background())
	require.NoError(t, err)
	require.True(t, visibility.ShowDeviceHistory)
	require.Equal(t, []string{"operator enabled device history"}, visibility.Notes)
}

func TestSQLiteRepositorySweepPrunesExpiredNonBaselineSnapshots(t *testing.T) {
	// [REQ:NM-P0-008] Retention sweeps remove expired non-baseline snapshots
	// while keeping the baseline anchor for before/after comparison.
	db, repo := newPrivacyDB(t)
	now := time.Date(2026, 6, 23, 21, 0, 0, 0, time.UTC)
	insertSnapshot(t, db, "baseline-old", "baseline", now.AddDate(0, 0, -90))
	insertSnapshot(t, db, "complete-old", "complete", now.AddDate(0, 0, -90))
	insertSnapshot(t, db, "complete-new", "complete", now.AddDate(0, 0, -3))

	result, err := repo.Sweep(context.Background(), RetentionSettings{
		QueryLogDays:   0,
		SnapshotDays:   30,
		ExperimentDays: 30,
		Profile:        ProfileHomeMinimal,
	}, now)
	require.NoError(t, err)
	require.Equal(t, 1, result.SnapshotsDeleted)
	require.Equal(t, now.AddDate(0, 0, -30), result.SnapshotCutoff)

	requireSnapshotExists(t, db, "baseline-old")
	requireSnapshotMissing(t, db, "complete-old")
	requireSnapshotExists(t, db, "complete-new")
}

func TestSQLiteRepositorySchemaIsIdempotent(t *testing.T) {
	db := testdb.NewSQLite(t)
	for range 2 {
		require.NoError(t, apidb.EnsureSchemas(context.Background(), db,
			apidb.SchemaProviderFunc(localdb.SystemSchema),
			apidb.SchemaProviderFunc(Schema),
		))
	}
}

func insertSnapshot(t *testing.T, db *sql.DB, id, status string, createdAt time.Time) {
	t.Helper()
	_, err := db.ExecContext(context.Background(), `
INSERT INTO network_snapshots (id, status, profile, summary, findings_json, created_at)
VALUES (?, ?, 'home', 'summary', '[]', ?)
`, id, status, createdAt.UTC().Format(snapshot.TimeFormat))
	require.NoError(t, err)
}

func requireSnapshotExists(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	var count int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM network_snapshots WHERE id = ?`, id).Scan(&count))
	require.Equal(t, 1, count)
}

func requireSnapshotMissing(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	var count int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM network_snapshots WHERE id = ?`, id).Scan(&count))
	require.Zero(t, count)
}
