package database

import (
	"context"
	"database/sql"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// TestRunExecutionModeRoundTrip verifies the interactive substrate's durable
// run-state additions (execution_mode + web_console_session_id) persist and
// read back through Create/Get/Update/List.
func TestRunExecutionModeRoundTrip(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	task := &domain.Task{
		ID:        uuid.New(),
		Title:     "Interactive parent task",
		ScopePath: "/test",
		Status:    domain.TaskStatusQueued,
	}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	run := &domain.Run{
		ID:                  uuid.New(),
		TaskID:              task.ID,
		Tag:                 "interactive-run",
		RunMode:             domain.RunModeInPlace,
		ExecutionMode:       domain.ExecutionModeInteractive,
		WebConsoleSessionID: "session-abc-123",
		TranscriptPath:      "/home/u/.claude/projects/-x/sess.jsonl",
		Status:              domain.RunStatusRunning,
		Phase:               domain.RunPhaseExecuting,
		ApprovalState:       domain.ApprovalStateNone,
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || got == nil {
		t.Fatalf("get run: %v (got=%v)", err, got)
	}
	if got.ExecutionMode != domain.ExecutionModeInteractive {
		t.Errorf("execution mode: got %q, want interactive", got.ExecutionMode)
	}
	if got.WebConsoleSessionID != "session-abc-123" {
		t.Errorf("web console session id: got %q", got.WebConsoleSessionID)
	}
	if got.TranscriptPath != run.TranscriptPath {
		t.Errorf("transcript path: got %q", got.TranscriptPath)
	}

	// List surfaces the mode + session id (pruned column set).
	runs, err := repos.Runs.List(ctx, repository.RunListFilter{ListFilter: repository.ListFilter{Limit: 10}})
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("list: got %d runs, want 1", len(runs))
	}
	if runs[0].ExecutionMode != domain.ExecutionModeInteractive {
		t.Errorf("list execution mode: got %q, want interactive", runs[0].ExecutionMode)
	}
	if runs[0].WebConsoleSessionID != "session-abc-123" {
		t.Errorf("list web console session id: got %q", runs[0].WebConsoleSessionID)
	}

	// Update flips it back to codec_pipe and clears the session id.
	got.ExecutionMode = domain.ExecutionModeCodecPipe
	got.WebConsoleSessionID = ""
	if err := repos.Runs.Update(ctx, got); err != nil {
		t.Fatalf("update run: %v", err)
	}
	after, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || after == nil {
		t.Fatalf("get after update: %v", err)
	}
	if after.ExecutionMode != domain.ExecutionModeCodecPipe {
		t.Errorf("after update execution mode: got %q, want codec_pipe", after.ExecutionMode)
	}
	if after.WebConsoleSessionID != "" {
		t.Errorf("after update session id: got %q, want empty", after.WebConsoleSessionID)
	}
}

// TestRunExecutionModeDefaultsToCodecPipe verifies a run created without an
// explicit ExecutionMode reads back as codec_pipe (the default), so existing
// callers are unaffected.
func TestRunExecutionModeDefaultsToCodecPipe(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	task := &domain.Task{ID: uuid.New(), Title: "t", ScopePath: "/test", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	run := &domain.Run{
		ID:            uuid.New(),
		TaskID:        task.ID,
		Tag:           "default-mode-run",
		RunMode:       domain.RunModeSandboxed,
		Status:        domain.RunStatusPending,
		Phase:         domain.RunPhaseQueued,
		ApprovalState: domain.ApprovalStateNone,
		// ExecutionMode left empty on purpose.
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}
	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || got == nil {
		t.Fatalf("get run: %v", err)
	}
	if got.ExecutionMode != domain.ExecutionModeCodecPipe {
		t.Errorf("default execution mode: got %q, want codec_pipe", got.ExecutionMode)
	}
}

func TestAttachedRunAllowsUnboundTaskAndPersistsHarnessIdentity(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	run := &domain.Run{
		ID:                uuid.New(),
		Tag:               "attached-run",
		RunMode:           domain.RunModeInPlace,
		ExecutionMode:     domain.ExecutionModeAttached,
		HarnessKind:       "claude-code",
		HarnessSessionID:  "session-123",
		Status:            domain.RunStatusRunning,
		Phase:             domain.RunPhaseExecuting,
		ApprovalState:     domain.ApprovalStateNone,
		IdentityTokenHash: "hash-only",
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("create attached run: %v", err)
	}

	got, err := repos.Runs.Get(ctx, run.ID)
	if err != nil || got == nil {
		t.Fatalf("get attached run: %v (got=%v)", err, got)
	}
	if got.TaskID != uuid.Nil {
		t.Fatalf("task id: got %s, want nil UUID for unbound run", got.TaskID)
	}
	if got.ExecutionMode != domain.ExecutionModeAttached || got.HarnessKind != "claude-code" || got.HarnessSessionID != "session-123" {
		t.Fatalf("attached metadata did not round-trip: mode=%q kind=%q session=%q", got.ExecutionMode, got.HarnessKind, got.HarnessSessionID)
	}

	var nullableTaskID sql.NullString
	if err := db.GetContext(ctx, &nullableTaskID, "SELECT task_id FROM runs WHERE id = ?", run.ID.String()); err != nil {
		t.Fatalf("read nullable task id: %v", err)
	}
	if nullableTaskID.Valid {
		t.Fatalf("database task_id = %q, want SQL NULL", nullableTaskID.String)
	}
}

func TestMigrateRunTaskIDNullabilityPreservesRowsIndexesAndTriggers(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := sqlx.Connect("sqlite", "file:"+t.TempDir()+"/nullable.db?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(1)
	if _, err := sqlDB.ExecContext(ctx, `
		CREATE TABLE tasks (id TEXT PRIMARY KEY);
		CREATE TABLE runs (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			tag TEXT,
			created_at TEXT,
			updated_at TEXT
		);
		CREATE TABLE run_checkpoints (run_id TEXT PRIMARY KEY REFERENCES runs(id) ON DELETE CASCADE);
		CREATE INDEX idx_runs_tag_test ON runs(tag);
		CREATE TRIGGER update_runs_updated_at_test AFTER UPDATE ON runs
		FOR EACH ROW BEGIN UPDATE runs SET updated_at = 'triggered' WHERE id = NEW.id; END;
		INSERT INTO tasks(id) VALUES ('task-1');
		INSERT INTO runs(id, task_id, tag, created_at, updated_at) VALUES ('run-1', 'task-1', 'before', 'created', 'original');
	`); err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}

	db := &DB{DB: sqlDB, log: logrus.New()}
	if err := db.migrateRunTaskIDNullability(ctx); err != nil {
		t.Fatalf("migrate task id nullability: %v", err)
	}

	var notNull int
	if err := sqlDB.GetContext(ctx, &notNull, "SELECT \"notnull\" FROM pragma_table_info('runs') WHERE name = 'task_id'"); err != nil {
		t.Fatalf("read task id nullability: %v", err)
	}
	if notNull != 0 {
		t.Fatalf("task_id notnull = %d, want 0", notNull)
	}
	var tag, updated string
	if err := sqlDB.QueryRowContext(ctx, "SELECT tag, updated_at FROM runs WHERE id = 'run-1'").Scan(&tag, &updated); err != nil {
		t.Fatalf("read preserved run: %v", err)
	}
	if tag != "before" || updated != "original" {
		t.Fatalf("preserved run = tag %q updated %q", tag, updated)
	}
	var indexCount, triggerCount int
	if err := sqlDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_runs_tag_test'").Scan(&indexCount); err != nil {
		t.Fatalf("check restored index: %v", err)
	}
	if err := sqlDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type = 'trigger' AND name = 'update_runs_updated_at_test'").Scan(&triggerCount); err != nil {
		t.Fatalf("check restored trigger: %v", err)
	}
	if indexCount != 1 || triggerCount != 1 {
		t.Fatalf("restored schema objects: index=%d trigger=%d", indexCount, triggerCount)
	}
	if _, err := sqlDB.ExecContext(ctx, "INSERT INTO runs(id, task_id, tag) VALUES ('run-2', NULL, 'attached')"); err != nil {
		t.Fatalf("insert unbound run after migration: %v", err)
	}
	if _, err := sqlDB.ExecContext(ctx, "UPDATE runs SET tag = 'after' WHERE id = 'run-1'"); err != nil {
		t.Fatalf("exercise restored trigger: %v", err)
	}
	if err := sqlDB.QueryRowContext(ctx, "SELECT updated_at FROM runs WHERE id = 'run-1'").Scan(&updated); err != nil {
		t.Fatalf("read triggered update: %v", err)
	}
	if updated != "triggered" {
		t.Fatalf("restored trigger updated_at = %q, want triggered", updated)
	}
}

// TestMigrateRunColumnsAddsMissingColumns builds a runs table WITHOUT the
// interactive/result columns, seeds a row, then runs the additive migration and
// asserts the columns appear with their defaults while the existing row's data
// is preserved (migrate-never-recreate).
func TestMigrateRunColumnsAddsMissingColumns(t *testing.T) {
	ctx := context.Background()
	sqlDB, err := sqlx.Connect("sqlite", "file:"+t.TempDir()+"/legacy.db?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer sqlDB.Close()

	// Minimal legacy runs table missing the interactive columns.
	if _, err := sqlDB.ExecContext(ctx, `CREATE TABLE runs (
		id TEXT PRIMARY KEY,
		tag TEXT,
		run_mode TEXT DEFAULT 'sandboxed'
	)`); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	if _, err := sqlDB.ExecContext(ctx, `INSERT INTO runs (id, tag, run_mode) VALUES ('r1', 'legacy-tag', 'in_place')`); err != nil {
		t.Fatalf("seed row: %v", err)
	}

	db := &DB{DB: sqlDB, log: logrus.New()}
	db.log.SetLevel(logrus.PanicLevel)

	// First migration adds the columns.
	if err := db.migrateRunColumns(ctx); err != nil {
		t.Fatalf("migrate (first pass): %v", err)
	}
	// Second migration is a no-op (idempotent) — must not error.
	if err := db.migrateRunColumns(ctx); err != nil {
		t.Fatalf("migrate (second pass): %v", err)
	}

	cols, err := db.tableColumns(ctx, "runs")
	if err != nil {
		t.Fatalf("table columns: %v", err)
	}
	for _, want := range []string{"execution_mode", "web_console_session_id", "run_result"} {
		if _, ok := cols[want]; !ok {
			t.Errorf("expected column %q after migration", want)
		}
	}

	// Existing row preserved; new column took its default.
	var tag, execMode string
	if err := sqlDB.QueryRowContext(ctx, `SELECT tag, execution_mode FROM runs WHERE id = 'r1'`).Scan(&tag, &execMode); err != nil {
		t.Fatalf("read migrated row: %v", err)
	}
	if tag != "legacy-tag" {
		t.Errorf("preserved tag: got %q, want legacy-tag", tag)
	}
	if execMode != "codec_pipe" {
		t.Errorf("default execution_mode: got %q, want codec_pipe", execMode)
	}
	var runResult sql.NullString
	if err := sqlDB.QueryRowContext(ctx, `SELECT run_result FROM runs WHERE id = 'r1'`).Scan(&runResult); err != nil {
		t.Fatalf("read historical run_result: %v", err)
	}
	if runResult.Valid {
		t.Errorf("historical run_result should remain NULL, got %q", runResult.String)
	}
}
