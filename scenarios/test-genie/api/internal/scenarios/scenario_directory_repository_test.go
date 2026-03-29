package scenarios

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"

	"test-genie/internal/storage/sqliteutil"
	"test-genie/internal/testsqlite"
)

func TestScenarioDirectoryRepositoryList(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewScenarioDirectoryRepository(db)
	now := time.Now().UTC()
	repo.clock = func() time.Time { return now }
	repo.queueActiveWindow = 24 * time.Hour
	execID := uuid.New()

	insertSuiteRequest(t, db,
		"req-1",
		"ecosystem-manager",
		`["unit","integration"]`,
		95,
		"high",
		"queued",
		sql.NullString{String: "Cover OT-P0", Valid: true},
		now,
		now,
	)
	insertSuiteRequest(t, db,
		"req-2",
		"ecosystem-manager",
		`["unit"]`,
		90,
		"normal",
		"delegated",
		sql.NullString{},
		now.Add(-time.Minute),
		now.Add(-time.Minute),
	)

	insertExecution(t, db,
		execID.String(),
		"ecosystem-manager",
		sql.NullString{String: "quick", Valid: true},
		1,
		`[{"name":"structure","status":"passed","durationSeconds":1}]`,
		now,
	)
	insertExecution(t, db,
		uuid.NewString(),
		"ecosystem-manager",
		sql.NullString{String: "", Valid: false},
		0,
		`[{"name":"unit","status":"failed","durationSeconds":2}]`,
		now.Add(-time.Hour),
	)

	summaries, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("expected list to succeed: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected one summary: %#v", summaries)
	}
	summary := summaries[0]
	if summary.ScenarioName != "ecosystem-manager" {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	if summary.PendingRequests != 2 || summary.TotalRequests != 2 {
		t.Fatalf("unexpected queue counts: %#v", summary)
	}
	if summary.LastExecutionPhaseSummary == nil || summary.LastExecutionPhaseSummary.Total != 1 {
		t.Fatalf("expected phase summary to be calculated: %#v", summary.LastExecutionPhaseSummary)
	}
	if summary.LastExecutionID == nil || *summary.LastExecutionID != execID {
		t.Fatalf("expected execution id %s, got %#v", execID, summary.LastExecutionID)
	}
	if summary.LastFailureAt == nil || !summary.LastFailureAt.Equal(now.Add(-time.Hour)) {
		t.Fatalf("expected failure timestamp to be preserved: %#v", summary.LastFailureAt)
	}
}

func TestScenarioDirectoryRepositoryGet(t *testing.T) {
	db := testsqlite.Open(t)
	repo := NewScenarioDirectoryRepository(db)
	now := time.Now().UTC()
	repo.clock = func() time.Time { return now }
	repo.queueActiveWindow = 24 * time.Hour

	insertSuiteRequest(t, db,
		"req-1",
		"ecosystem-manager",
		`[]`,
		95,
		"normal",
		"completed",
		sql.NullString{},
		now,
		now,
	)

	summary, err := repo.Get(context.Background(), "ecosystem-manager")
	if err != nil {
		t.Fatalf("expected get to succeed: %v", err)
	}
	if summary == nil || summary.ScenarioName != "ecosystem-manager" {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	if summary.TotalRequests != 1 {
		t.Fatalf("expected total requests to be populated: %#v", summary)
	}
}

func insertSuiteRequest(t *testing.T, db *sql.DB, id, scenarioName, requestedTypes string, coverageTarget int, priority, status string, notes sql.NullString, createdAt, updatedAt time.Time) {
	t.Helper()
	var note any
	if notes.Valid {
		note = notes.String
	}
	if _, err := db.Exec(`
INSERT INTO suite_requests (
	id, scenario_name, requested_types, coverage_target, priority, status, notes, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
`,
		id,
		scenarioName,
		requestedTypes,
		coverageTarget,
		priority,
		status,
		note,
		sqliteutil.FormatTimestamp(createdAt),
		sqliteutil.FormatTimestamp(updatedAt),
	); err != nil {
		t.Fatalf("insert suite request %s: %v", id, err)
	}
}

func insertExecution(t *testing.T, db *sql.DB, id, scenarioName string, preset sql.NullString, success int, phases string, completedAt time.Time) {
	t.Helper()
	var presetValue any
	if preset.Valid {
		presetValue = preset.String
	}
	if _, err := db.Exec(`
INSERT INTO suite_executions (
	id, scenario_name, preset_used, requested_phases, requested_skip_phases, planned_phases,
	fail_fast, success, phases, started_at, completed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`,
		id,
		scenarioName,
		presetValue,
		`[]`,
		`[]`,
		`[]`,
		0,
		success,
		phases,
		sqliteutil.FormatTimestamp(completedAt.Add(-time.Minute)),
		sqliteutil.FormatTimestamp(completedAt),
	); err != nil {
		t.Fatalf("insert execution %s: %v", id, err)
	}
}
