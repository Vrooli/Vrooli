package physicalruntime

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"device-studio/internal/devices"
	"device-studio/internal/jobs"
	"device-studio/internal/library"
	"device-studio/internal/motions"
	"device-studio/internal/projects"
	"device-studio/internal/routedblobstore"
	"device-studio/internal/scenes"
	"device-studio/internal/shots"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

type starterTestFixture struct {
	ctx            context.Context
	owners         Owners
	db             *database.RoutedDB
	roots          *filerouting.RoutedRoots
	projectService projects.Service
	media          projects.SourceMediaService
}

// newStarterTestFixture creates one isolated primary/test route pair and the
// stable project/media owners used by physical starter integration tests. The
// caller supplies its lease identity and keeps device-specific expectations
// and any additional services beside the behavior under test.
func newStarterTestFixture(t *testing.T, ctx context.Context, lease string) starterTestFixture {
	t.Helper()
	root := t.TempDir()
	providers := []database.SchemaProvider{
		database.SchemaProviderFunc(library.Schema), database.SchemaProviderFunc(devices.Schema),
		database.SchemaProviderFunc(scenes.Schema), database.SchemaProviderFunc(motions.Schema),
		database.SchemaProviderFunc(projects.Schema), database.SchemaProviderFunc(shots.Schema), database.SchemaProviderFunc(jobs.Schema),
	}
	db, err := database.Open(ctx, database.Config{Driver: database.DriverSQLite, DSN: filepath.Join(root, "primary.db"), MaxOpenConns: 1, MaxIdleConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.EnsureSchemas(ctx, db.Primary(), providers...); err != nil {
		t.Fatal(err)
	}
	db.SetTestPoolInitializer(func(initCtx context.Context, pool *sql.DB) error {
		return database.EnsureSchemas(initCtx, pool, providers...)
	})
	roots := filerouting.New(storage.Paths{DataDir: filepath.Join(root, "primary-data")})
	poolInstalled, rootsInstalled := false, false
	t.Cleanup(func() {
		if rootsInstalled {
			if err := roots.ClearTestRoots(lease); err != nil {
				t.Errorf("clear starter test file route %q: %v", lease, err)
			}
		}
		if poolInstalled {
			if err := db.ClearTestPool(lease); err != nil {
				t.Errorf("clear starter test database route %q: %v", lease, err)
			}
		}
	})
	if err := db.InstallTestPool(ctx, filepath.Join(root, "leased.db"), lease, time.Minute); err != nil {
		t.Fatal(err)
	}
	poolInstalled = true
	if err := roots.InstallTestRoots(storage.Paths{DataDir: filepath.Join(root, "leased-data")}, lease, time.Minute); err != nil {
		t.Fatal(err)
	}
	rootsInstalled = true
	repo := projects.NewSQLiteRepository(db, nil)
	projectService := projects.NewService(repo, nil)
	mediaStore, err := routedblobstore.New(roots, "media")
	if err != nil {
		t.Fatal(err)
	}
	media := projects.NewSourceMediaService(projects.SourceMediaDeps{Projects: projectService, Repository: repo.(projects.SourceMediaRepository), Store: mediaStore})
	owners, err := New(db, roots, media)
	if err != nil {
		t.Fatal(err)
	}
	return starterTestFixture{ctx: database.WithTestMode(ctx), owners: owners, db: db, roots: roots, projectService: projectService, media: media}
}
