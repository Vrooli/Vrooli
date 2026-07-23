package testutil

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestGetSchemaDir_ContainsSchemaFile(t *testing.T) {
	schemaPath := filepath.Join(getSchemaDir(), "schema.sql")
	info, err := os.Stat(schemaPath)
	if err != nil {
		t.Fatalf("expected schema file at %s: %v", schemaPath, err)
	}
	if info.IsDir() {
		t.Fatalf("expected schema path to be file, got directory: %s", schemaPath)
	}
}

func TestSetupTestDBAndRepositoriesProvideAnIsolatedUsableSchema(t *testing.T) {
	db, cleanup := SetupTestDB(t)
	t.Cleanup(cleanup)
	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("ping test database: %v", err)
	}
	repos, events, reposCleanup := SetupTestReposWithDB(t, db)
	t.Cleanup(reposCleanup)
	task := &domain.Task{ID: uuid.New(), Title: "harness", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(context.Background(), task); err != nil {
		t.Fatalf("create through shared repository graph: %v", err)
	}
	runID := uuid.New()
	if err := repos.Runs.Create(context.Background(), &domain.Run{ID: runID, TaskID: task.ID, Status: domain.RunStatusPending, Phase: domain.RunPhaseQueued}); err != nil {
		t.Fatalf("create run for event foreign key: %v", err)
	}
	if err := events.Append(context.Background(), runID, domain.NewLogEvent(runID, "info", "harness event")); err != nil {
		t.Fatalf("append through shared event store: %v", err)
	}

	otherRepos, otherEvents, otherCleanup := SetupTestRepos(t)
	t.Cleanup(otherCleanup)
	if otherRepos == nil || otherEvents == nil {
		t.Fatal("independent repository harness was not constructed")
	}
}
