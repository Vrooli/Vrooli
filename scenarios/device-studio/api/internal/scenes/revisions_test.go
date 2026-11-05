package scenes

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
)

func TestSceneRevisionCASReplayHistoryAndCorruption(t *testing.T) {
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
	a := approvedPhoneStudyMetadata()
	first, err := service.SaveCandidate(ctx, a, 0)
	if err != nil || first.Revision != 1 || first.State != RevisionCandidate {
		t.Fatalf("save first scene: %+v %v", first, err)
	}
	replay, err := service.SaveCandidate(ctx, a, 0)
	if err != nil || replay.Revision != 1 || replay.SHA256 != first.SHA256 {
		t.Fatalf("exact target retry: %+v %v", replay, err)
	}
	b := approvedPhoneStudyMetadata()
	b.Environment.Strength = 0.25
	second, err := service.SaveCandidate(ctx, b, 1)
	if err != nil || second.Revision != 2 {
		t.Fatalf("save second scene: %+v %v", second, err)
	}
	if _, err := service.SaveCandidate(ctx, a, 1); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale head accepted: %v", err)
	}
	third, err := service.SaveCandidate(ctx, a, 2)
	if err != nil || third.Revision != 3 || third.SHA256 != first.SHA256 {
		t.Fatalf("A→B→A history: %+v %v", third, err)
	}
	rows, err := service.ListRevisions(ctx, a.SceneID, 0, 3)
	if err != nil || len(rows) != 3 || rows[2].Revision != 3 {
		t.Fatalf("list revisions: %+v %v", rows, err)
	}
	if _, err := db.Exec(`DROP TRIGGER scene_revisions_no_update`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE scene_revisions SET candidate_json=candidate_json || ' ' WHERE scene_id=? AND revision=1`, a.SceneID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetRevision(ctx, a.SceneID, 1); err == nil {
		t.Fatal("corrupt canonical scene bytes were returned")
	}
}

func TestConcurrentSceneWritersConvergeOrConflictWithoutLostHistory(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	if err := apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	repoA, err := NewSQLiteRevisionRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	repoB, err := NewSQLiteRevisionRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	svcA, _ := NewRevisionService(repoA, nil)
	svcB, _ := NewRevisionService(repoB, nil)
	a, b := approvedPhoneStudyMetadata(), approvedPhoneStudyMetadata()
	a.Environment.Strength = 0.3
	b.Environment.Strength = 0.4
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, item := range []struct {
		service   *RevisionService
		candidate SceneCandidate
	}{{svcA, a}, {svcB, b}} {
		wg.Add(1)
		go func(service *RevisionService, candidate SceneCandidate) {
			defer wg.Done()
			<-start
			_, err := service.SaveCandidate(ctx, candidate, 0)
			errs <- err
		}(item.service, item.candidate)
	}
	close(start)
	wg.Wait()
	close(errs)
	winners, conflicts := 0, 0
	for err := range errs {
		if err == nil {
			winners++
			continue
		}
		if errors.Is(err, ErrRevisionConflict) {
			conflicts++
			continue
		}
		t.Fatalf("unexpected concurrent scene write result: %v", err)
	}
	if winners != 1 || conflicts != 1 {
		t.Fatalf("concurrent different candidates winners=%d conflicts=%d", winners, conflicts)
	}
	if head, err := svcA.GetHead(ctx, "phone-table-study"); err != nil || head.Revision != 1 {
		t.Fatalf("head after competing save: %+v %v", head, err)
	}
}

func TestSceneRevisionRepositoryUsesRoutedTestPoolContext(t *testing.T) {
	ctx := context.Background()
	routed, err := apidb.Open(ctx, apidb.Config{Driver: apidb.DriverSQLite, DSN: filepath.Join(t.TempDir(), "scene-primary.db")})
	require.NoError(t, err)
	t.Cleanup(func() { _ = routed.Close() })
	require.NoError(t, apidb.EnsureSchemas(ctx, routed, apidb.SchemaProviderFunc(Schema)))
	routed.SetTestPoolInitializer(func(initCtx context.Context, pool *sql.DB) error {
		return apidb.EnsureSchemas(initCtx, pool, apidb.SchemaProviderFunc(Schema))
	})
	require.NoError(t, routed.InstallTestPool(ctx, filepath.Join(t.TempDir(), "scene-lease.db"), "scene-revisions", 0))
	repo, err := NewSQLiteRevisionRepository(routed)
	require.NoError(t, err)
	service, err := NewRevisionService(repo, nil)
	require.NoError(t, err)
	testCtx := apidb.WithTestMode(ctx)
	candidate := approvedPhoneStudyMetadata()
	saved, err := service.SaveCandidate(testCtx, candidate, 0)
	require.NoError(t, err)
	require.Equal(t, uint32(1), saved.Revision)
	got, err := service.GetRevision(testCtx, saved.SceneID, saved.Revision)
	require.NoError(t, err)
	require.Equal(t, saved.SHA256, got.SHA256)
	page, err := service.ListRevisions(testCtx, saved.SceneID, 0, 10)
	require.NoError(t, err)
	require.Len(t, page, 1)
	if _, err := service.GetHead(ctx, saved.SceneID); !errors.Is(err, ErrRevisionNotFound) {
		t.Fatalf("plain context should remain on primary pool, got %v", err)
	}
	var primaryRows int
	require.NoError(t, routed.QueryRowContext(ctx, "SELECT COUNT(*) FROM scene_revisions").Scan(&primaryRows))
	require.Equal(t, 0, primaryRows)
}
