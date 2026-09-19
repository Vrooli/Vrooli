package database_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	coredb "github.com/vrooli/api-core/database"
	basdb "github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/domain"
	"github.com/vrooli/browser-automation-studio/services/recording/persistence"
)

func TestNewConnectionRoutesRecordingWritesToLeasedTestPool(t *testing.T) {
	primaryPath := filepath.Join(t.TempDir(), "primary.db")
	testPath := filepath.Join(t.TempDir(), "test.db")
	t.Setenv("BAS_SQLITE_PATH", primaryPath)

	db, err := basdb.NewConnection(logrus.New())
	if err != nil {
		t.Fatalf("open routed BAS database: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Routed.ClearTestPool("routing-test"); err != nil {
			t.Errorf("clear test pool: %v", err)
		}
		if err := db.Close(); err != nil {
			t.Errorf("close primary database: %v", err)
		}
	})

	testDSN := fmt.Sprintf("file:%s", testPath)
	if err := db.Routed.InstallTestPool(context.Background(), testDSN, "routing-test", time.Minute); err != nil {
		t.Fatalf("install test pool: %v", err)
	}

	repo := persistence.NewSQLiteRepository(db.Routed, logrus.New())
	ctx := coredb.WithTestMode(context.Background())
	if err := repo.CreateSession(ctx, &domain.RecordingSession{
		ID:             "routed-session",
		Status:         domain.SessionStatusActive,
		ViewportWidth:  1280,
		ViewportHeight: 720,
		CreatedAt:      time.Now().UTC(),
	}); err != nil {
		t.Fatalf("create recording session through routed pool: %v", err)
	}

	var primaryCount, testCount int
	if err := db.RawDB().QueryRowContext(context.Background(), "SELECT COUNT(*) FROM recording_sessions").Scan(&primaryCount); err != nil {
		t.Fatalf("count primary recording sessions: %v", err)
	}
	if err := db.Routed.QueryRowContext(ctx, "SELECT COUNT(*) FROM recording_sessions").Scan(&testCount); err != nil {
		t.Fatalf("count test recording sessions: %v", err)
	}
	if primaryCount != 0 || testCount != 1 {
		t.Fatalf("expected write only in leased test pool, got primary=%d test=%d", primaryCount, testCount)
	}

	// The scenario's central index repository uses the sqlx-shaped facade on
	// database.DB. This assertion protects against a future fallback to the
	// embedded primary sqlx pool.
	indexRepo := basdb.NewRepository(db, logrus.New())
	if err := indexRepo.CreateProject(ctx, &basdb.ProjectIndex{Name: "leased-project", FolderPath: "/leased-project"}); err != nil {
		t.Fatalf("create project through routed repository: %v", err)
	}
	if err := db.RawDB().QueryRowContext(context.Background(), "SELECT COUNT(*) FROM projects").Scan(&primaryCount); err != nil {
		t.Fatalf("count primary projects: %v", err)
	}
	if err := db.Routed.QueryRowContext(ctx, "SELECT COUNT(*) FROM projects").Scan(&testCount); err != nil {
		t.Fatalf("count test projects: %v", err)
	}
	if primaryCount != 0 || testCount != 1 {
		t.Fatalf("expected repository write only in leased test pool, got primary=%d test=%d", primaryCount, testCount)
	}

	// Routed reads must retain sqlx's struct and slice mapping semantics. This
	// protects GetContext/SelectContext from silently scanning via the primary
	// pool or treating a struct destination as a slice.
	project, err := indexRepo.GetProjectByName(ctx, "leased-project")
	if err != nil {
		t.Fatalf("get routed project by name: %v", err)
	}
	if project.FolderPath != "/leased-project" {
		t.Fatalf("routed project folder = %q", project.FolderPath)
	}
	projects, err := indexRepo.ListProjects(ctx, 10, 0)
	if err != nil {
		t.Fatalf("list routed projects: %v", err)
	}
	if len(projects) != 1 || projects[0].Name != "leased-project" {
		t.Fatalf("unexpected routed projects: %+v", projects)
	}
}
