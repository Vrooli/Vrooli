package motions

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"device-studio/internal/composition"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
)

func TestMotionRevisionCASReplayAndImmutableHistory(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	if err := apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	repo, err := NewSQLiteRevisionRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewRevisionService(repo, func() time.Time { return time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	a := baseCandidate()
	first, err := service.SaveCandidate(ctx, a, 0)
	if err != nil || first.Revision != 1 || first.State != RevisionCandidate {
		t.Fatalf("save first motion: %+v %v", first, err)
	}
	replay, err := service.SaveCandidate(ctx, a, 0)
	if err != nil || replay.Revision != 1 || replay.SHA256 != first.SHA256 {
		t.Fatalf("exact target retry: %+v %v", replay, err)
	}
	b := baseCandidate()
	b.DurationTicks = 20
	for i := range b.Tracks {
		b.Tracks[i].Keys[len(b.Tracks[i].Keys)-1].Tick = 20
	}
	b.Timebase = composition.Rational{Num: 24, Den: 1}
	second, err := service.SaveCandidate(ctx, b, 1)
	if err != nil || second.Revision != 2 {
		t.Fatalf("save second motion: %+v %v", second, err)
	}
	if _, err := service.SaveCandidate(ctx, a, 1); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale head accepted: %v", err)
	}
	third, err := service.SaveCandidate(ctx, a, 2)
	if err != nil || third.Revision != 3 || third.SHA256 != first.SHA256 {
		t.Fatalf("A→B→A history: %+v %v", third, err)
	}
	rows, err := service.ListRevisions(ctx, a.MotionID, 0, 3)
	if err != nil || len(rows) != 3 || rows[2].Revision != 3 {
		t.Fatalf("list revisions: %+v %v", rows, err)
	}
	if _, err := service.GetHead(ctx, a.MotionID); err != nil {
		t.Fatalf("get head: %v", err)
	}
	if _, err := db.Exec(`DROP TRIGGER motion_revisions_no_update`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE motion_revisions SET sha256=? WHERE motion_id=? AND revision=1`, strings.Repeat("0", 64), a.MotionID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetRevision(ctx, a.MotionID, 1); err == nil {
		t.Fatal("mismatched stored motion digest was returned")
	}
}

func TestMotionRevisionRepositoryUsesRoutedTestPoolContext(t *testing.T) {
	ctx := context.Background()
	routed, err := apidb.Open(ctx, apidb.Config{Driver: apidb.DriverSQLite, DSN: filepath.Join(t.TempDir(), "motion-primary.db")})
	require.NoError(t, err)
	t.Cleanup(func() { _ = routed.Close() })
	require.NoError(t, apidb.EnsureSchemas(ctx, routed, apidb.SchemaProviderFunc(Schema)))
	routed.SetTestPoolInitializer(func(initCtx context.Context, pool *sql.DB) error {
		return apidb.EnsureSchemas(initCtx, pool, apidb.SchemaProviderFunc(Schema))
	})
	require.NoError(t, routed.InstallTestPool(ctx, filepath.Join(t.TempDir(), "motion-lease.db"), "motion-revisions", 0))
	repo, err := NewSQLiteRevisionRepository(routed)
	require.NoError(t, err)
	service, err := NewRevisionService(repo, nil)
	require.NoError(t, err)
	testCtx := apidb.WithTestMode(ctx)
	candidate := baseCandidate()
	candidate.MotionID = "routed-motion"
	saved, err := service.SaveCandidate(testCtx, candidate, 0)
	require.NoError(t, err)
	require.Equal(t, uint32(1), saved.Revision)
	got, err := service.GetRevision(testCtx, saved.MotionID, saved.Revision)
	require.NoError(t, err)
	require.Equal(t, saved.SHA256, got.SHA256)
	page, err := service.ListRevisions(testCtx, saved.MotionID, 0, 10)
	require.NoError(t, err)
	require.Len(t, page, 1)
	if _, err := service.GetHead(ctx, saved.MotionID); !errors.Is(err, ErrRevisionNotFound) {
		t.Fatalf("plain context should remain on primary pool, got %v", err)
	}
	var primaryRows int
	require.NoError(t, routed.QueryRowContext(ctx, "SELECT COUNT(*) FROM motion_revisions").Scan(&primaryRows))
	require.Zero(t, primaryRows)
}
