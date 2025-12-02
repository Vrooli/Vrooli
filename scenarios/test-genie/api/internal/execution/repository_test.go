package execution

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"

	"test-genie/internal/orchestrator/phases"
)

func TestSuiteExecutionRepositoryCreate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewSuiteExecutionRepository(db)
	now := time.Now()
	reqID := uuid.New()
	execID := uuid.New()
	record := &SuiteExecutionRecord{
		ID:             execID,
		SuiteRequestID: &reqID,
		ScenarioName:   "demo",
		PresetUsed:     "quick",
		Success:        true,
		Phases: []phases.ExecutionResult{
			{Name: "structure", Status: "passed", DurationSeconds: 1},
		},
		StartedAt:   now.Add(-time.Minute),
		CompletedAt: now,
	}

	mock.ExpectExec("INSERT INTO suite_executions").
		WithArgs(
			execID,
			reqID,
			record.ScenarioName,
			sql.NullString{String: "quick", Valid: true},
			record.Success,
			sqlmock.AnyArg(),
			record.StartedAt,
			record.CompletedAt,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.Create(context.Background(), record); err != nil {
		t.Fatalf("expected create to succeed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSuiteExecutionRepositoryListRecent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewSuiteExecutionRepository(db)
	now := time.Now().UTC()
	rows := sqlmock.NewRows([]string{
		"id",
		"suite_request_id",
		"scenario_name",
		"preset_used",
		"success",
		"phases",
		"started_at",
		"completed_at",
	}).AddRow(
		"11111111-1111-1111-1111-111111111111",
		"22222222-2222-2222-2222-222222222222",
		"demo",
		sql.NullString{String: "quick", Valid: true},
		true,
		[]byte(`[{"name":"structure","status":"passed","durationSeconds":1}]`),
		now.Add(-time.Minute),
		now,
	)

	mock.ExpectQuery("SELECT\\s+id,\\s+suite_request_id").
		WithArgs("demo", 5, 0).
		WillReturnRows(rows)

	results, err := repo.ListRecent(context.Background(), "demo", 5, 0)
	if err != nil {
		t.Fatalf("expected list to succeed: %v", err)
	}
	if len(results) != 1 || results[0].ScenarioName != "demo" {
		t.Fatalf("unexpected list response: %#v", results)
	}
	if len(results[0].Phases) != 1 {
		t.Fatalf("expected phases to be unmarshaled: %#v", results[0])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSuiteExecutionRepositoryGetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewSuiteExecutionRepository(db)
	now := time.Now().UTC()
	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	rows := sqlmock.NewRows([]string{
		"id",
		"suite_request_id",
		"scenario_name",
		"preset_used",
		"success",
		"phases",
		"started_at",
		"completed_at",
	}).AddRow(
		id.String(),
		nil,
		"demo",
		sql.NullString{String: "", Valid: false},
		false,
		[]byte(`[{"name":"structure","status":"failed","durationSeconds":2}]`),
		now.Add(-time.Minute),
		now,
	)

	mock.ExpectQuery("SELECT\\s+id,\\s+suite_request_id").
		WithArgs(id).
		WillReturnRows(rows)

	record, err := repo.GetByID(context.Background(), id)
	if err != nil {
		t.Fatalf("expected get to succeed: %v", err)
	}
	if record == nil || record.ID != id {
		t.Fatalf("unexpected record: %#v", record)
	}
	if record.Success {
		t.Fatalf("expected success=false")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
