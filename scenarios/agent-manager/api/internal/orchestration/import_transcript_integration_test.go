package orchestration_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	runnercore "agent-manager/internal/adapters/runner/core"
	"agent-manager/internal/domain"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/testutil"
)

// TestImportTranscriptPersistsAgainstSQLite exercises the real repositories,
// not a fake RunRepository. In particular, it proves imported runs satisfy the
// non-null runs.task_id foreign key through the durable synthetic task.
func TestImportTranscriptPersistsAgainstSQLite(t *testing.T) {
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	registry := runner.NewRegistry()
	if err := registry.Register(runnercore.NewRunner(codecs.NewCodexForTest(), nil, nil)); err != nil {
		t.Fatalf("register codex runner: %v", err)
	}
	svc := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs,
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(registry),
		orchestration.WithRunStateRoot(t.TempDir()),
		orchestration.WithConfig(orchestration.OrchestratorConfig{DefaultTimeout: time.Minute, MaxConcurrentRuns: 1}),
		orchestration.WithInvocationReadModel(repos.InvocationReadModel),
	)
	path := filepath.Join(t.TempDir(), "codex.jsonl")
	body := "{\"type\":\"thread.started\",\"thread_id\":\"import-test\"}\n{\"type\":\"turn.completed\"}\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	request := orchestration.ImportTranscriptRequest{Path: path, RunnerType: domain.RunnerTypeCodex, Label: "sqlite", SourceHarness: "resource:codex/sessions", SourceSessionID: "import-test"}
	run, err := svc.ImportTranscript(context.Background(), request)
	if err != nil {
		t.Fatalf("import transcript against SQLite: %v", err)
	}
	if run.TaskID.String() == "00000000-0000-0000-0000-000000000000" || run.ExecutionMode != domain.ExecutionModeImported {
		t.Fatalf("imported run = %+v", run)
	}
	task, err := svc.GetTask(context.Background(), run.TaskID)
	if err != nil || task.Title != "Imported external transcripts" {
		t.Fatalf("synthetic task = %+v, %v", task, err)
	}
	persisted, err := svc.GetRun(context.Background(), run.ID)
	if err != nil || persisted == nil || persisted.TaskID != run.TaskID {
		t.Fatalf("persisted run = %+v, %v", persisted, err)
	}
	if persisted.ImportSourceHarness != request.SourceHarness || persisted.ImportSourceSessionID != request.SourceSessionID || persisted.ImportedAt == nil {
		t.Fatalf("import provenance = %+v", persisted)
	}
	again, err := svc.ImportTranscript(context.Background(), request)
	if err != nil {
		t.Fatalf("repeat import: %v", err)
	}
	if again.ID != run.ID {
		t.Fatalf("repeat import created %s, want existing %s", again.ID, run.ID)
	}
}

func TestImportTranscriptCapturesOptionalGoalOutcome(t *testing.T) {
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	registry := runner.NewRegistry()
	if err := registry.Register(runnercore.NewRunner(codecs.NewCodexForTest(), nil, nil)); err != nil {
		t.Fatal(err)
	}
	goalHome := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(goalHome, "goals_1.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(`CREATE TABLE thread_goals (thread_id TEXT PRIMARY KEY, goal_id TEXT NOT NULL, objective TEXT NOT NULL, status TEXT NOT NULL, token_budget INTEGER, tokens_used INTEGER NOT NULL, time_used_seconds INTEGER NOT NULL); INSERT INTO thread_goals VALUES ('goal-thread','goal-1','ship','blocked',1000,900,60)`); err != nil {
		t.Fatal(err)
	}
	svc := orchestration.New(repos.Profiles, repos.Tasks, repos.Runs, orchestration.WithEvents(eventStore), orchestration.WithRunners(registry), orchestration.WithRunStateRoot(t.TempDir()), orchestration.WithInvocationReadModel(repos.InvocationReadModel))
	path := filepath.Join(t.TempDir(), "goal.jsonl")
	if err := os.WriteFile(path, []byte("{\"type\":\"thread.started\",\"thread_id\":\"goal-thread\"}\n{\"type\":\"turn.completed\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run, err := svc.ImportTranscript(context.Background(), orchestration.ImportTranscriptRequest{Path: path, RunnerType: domain.RunnerTypeCodex, GoalSessionHome: goalHome})
	if err != nil {
		t.Fatal(err)
	}
	store := repos.InvocationReadModel.(invocationreadmodel.GoalOutcomeStore)
	goal, err := store.GoalOutcome(context.Background(), run.ID.String())
	if err != nil || goal == nil || goal.Status != "blocked" || goal.TokensUsed != 900 {
		t.Fatalf("goal=%#v err=%v", goal, err)
	}
}
