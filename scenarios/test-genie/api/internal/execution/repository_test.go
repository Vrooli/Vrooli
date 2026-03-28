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
		ID:                  execID,
		SuiteRequestID:      &reqID,
		ScenarioName:        "demo",
		PresetUsed:          "quick",
		RequestedPreset:     "quick",
		RequestedPhases:     []string{"structure", "unit"},
		RequestedSkipPhases: []string{"performance"},
		PlannedPhases:       []string{"structure", "unit"},
		FailFast:            true,
		Success:             true,
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
			sql.NullString{String: "quick", Valid: true},
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			record.FailFast,
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
		"requested_preset",
		"requested_phases",
		"requested_skip_phases",
		"planned_phases",
		"fail_fast",
		"success",
		"phases",
		"started_at",
		"completed_at",
	}).AddRow(
		"11111111-1111-1111-1111-111111111111",
		"22222222-2222-2222-2222-222222222222",
		"demo",
		sql.NullString{String: "quick", Valid: true},
		sql.NullString{String: "quick", Valid: true},
		`{"structure","unit"}`,
		`{"performance"}`,
		`{"structure","unit"}`,
		true,
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
	if results[0].RequestedPreset != "quick" || !results[0].FailFast {
		t.Fatalf("expected execution metadata to round-trip: %#v", results[0])
	}
	if len(results[0].PlannedPhases) != 2 || results[0].PlannedPhases[1] != "unit" {
		t.Fatalf("expected planned phases to round-trip: %#v", results[0])
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
		"requested_preset",
		"requested_phases",
		"requested_skip_phases",
		"planned_phases",
		"fail_fast",
		"success",
		"phases",
		"started_at",
		"completed_at",
	}).AddRow(
		id.String(),
		nil,
		"demo",
		sql.NullString{String: "", Valid: false},
		sql.NullString{String: "", Valid: false},
		`{"structure"}`,
		`{"performance"}`,
		`{"structure","integration"}`,
		false,
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
	if len(record.RequestedPhases) != 1 || record.RequestedPhases[0] != "structure" {
		t.Fatalf("expected requested phases to round-trip: %#v", record)
	}
	if len(record.RequestedSkipPhases) != 1 || record.RequestedSkipPhases[0] != "performance" {
		t.Fatalf("expected requested skip phases to round-trip: %#v", record)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestSuiteExecutionRepositoryListPhaseSamples(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewSuiteExecutionRepository(db)
	now := time.Now().UTC()
	rows := sqlmock.NewRows([]string{
		"scenario_name",
		"phase_name",
		"status",
		"duration_seconds",
		"completed_at",
	}).AddRow(
		"demo",
		"unit",
		"passed",
		42,
		now,
	)

	mock.ExpectQuery("SELECT\\s+scenario_name,\\s+LOWER\\(TRIM\\(phase->>'name'\\)\\)").
		WithArgs(sqlmock.AnyArg(), now.Add(-time.Hour), 100).
		WillReturnRows(rows)

	samples, err := repo.ListPhaseSamples(context.Background(), []string{"unit", "unit"}, now.Add(-time.Hour), 100)
	if err != nil {
		t.Fatalf("expected list phase samples to succeed: %v", err)
	}
	if len(samples) != 1 {
		t.Fatalf("expected one sample, got %#v", samples)
	}
	if samples[0].ScenarioName != "demo" || samples[0].PhaseName != "unit" || samples[0].DurationSeconds != 42 {
		t.Fatalf("unexpected sample: %#v", samples[0])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
