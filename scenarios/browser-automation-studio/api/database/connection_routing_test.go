package database_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
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

	repo := persistence.NewSQLiteRepository(db.Routed)
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

	// A journal append must work on the production schema, including a freshly
	// leased test pool. A private repository-test schema cannot prove this.
	entry := &persistence.UnifiedTimelineEntry{
		ID: uuid.New(), Type: persistence.TimelineEntryTypeAction,
		Timestamp: time.Now().UTC(), SessionID: "routed-session", PageID: uuid.New(),
		Sequence: 1, Action: &domain.RecordingAction{ID: uuid.New(), ActionType: "input", Payload: map[string]interface{}{"text": "preserved"}},
	}
	if _, err := repo.AppendTimelineEntry(ctx, entry); err != nil {
		t.Fatalf("append routed journal entry using production schema: %v", err)
	}
	restored, err := repo.GetTimelineEntry(ctx, entry.ID)
	if err != nil || restored == nil || restored.Action == nil || restored.Action.Payload["text"] != "preserved" {
		t.Fatalf("read routed journal entry: entry=%+v err=%v", restored, err)
	}
	if err := db.RawDB().QueryRowContext(context.Background(), "SELECT COUNT(*) FROM timeline_entries").Scan(&primaryCount); err != nil {
		t.Fatalf("count primary journal: %v", err)
	}
	if primaryCount != 0 {
		t.Fatalf("leased journal write escaped into primary: %d entries", primaryCount)
	}
	// Reapplying declarative startup must preserve committed journal data.
	if err := basdb.ApplySchemaRegistry(ctx, db.Routed, ""); err != nil {
		t.Fatalf("repeat leased schema bootstrap: %v", err)
	}
	restored, err = repo.GetTimelineEntry(ctx, entry.ID)
	if err != nil || restored == nil || restored.Action.Payload["text"] != "preserved" {
		t.Fatalf("journal changed across schema bootstrap: entry=%+v err=%v", restored, err)
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
	workflow := &basdb.WorkflowIndex{ID: uuid.New(), Name: "leased-workflow", FolderPath: "/leased-workflow", Version: 1}
	if err := indexRepo.CreateWorkflow(ctx, workflow); err != nil {
		t.Fatal(err)
	}
	execution := &basdb.ExecutionIndex{ID: uuid.New(), WorkflowID: workflow.ID, Status: basdb.ExecutionStatusRunning, StartedAt: time.Now().UTC()}
	if err := indexRepo.CreateExecution(ctx, execution); err != nil {
		t.Fatal(err)
	}
	rows, total, err := indexRepo.ListExecutions(ctx, basdb.ExecutionQuery{Status: basdb.ExecutionStatusRunning, Limit: 1})
	if err != nil || total != 1 || len(rows) != 1 || rows[0].ID != execution.ID {
		t.Fatalf("routed execution page: rows=%v total=%d err=%v", rows, total, err)
	}
	rows, total, err = indexRepo.ListExecutions(context.Background(), basdb.ExecutionQuery{})
	if err != nil || total != 0 || len(rows) != 0 {
		t.Fatalf("execution query escaped test pool: rows=%v total=%d err=%v", rows, total, err)
	}

}
