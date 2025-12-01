package main

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	pq "github.com/lib/pq"
)

func TestScenarioDirectoryRepositoryList(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewScenarioDirectoryRepository(db)
	now := time.Now().UTC()
	execID := uuid.New()

	rows := sqlmock.NewRows([]string{
		"scenario_name",
		"pending_requests",
		"total_requests",
		"last_request_at",
		"priority",
		"status",
		"notes",
		"coverage_target",
		"requested_types",
		"total_executions",
		"last_execution_at",
		"id",
		"preset_used",
		"success",
		"phases",
		"completed_at",
	}).AddRow(
		"ecosystem-manager",
		2,
		5,
		now,
		"high",
		"queued",
		"Cover OT-P0",
		int64(95),
		pq.StringArray{"unit", "integration"},
		3,
		now,
		execID.String(),
		sql.NullString{String: "quick", Valid: true},
		sql.NullBool{Bool: true, Valid: true},
		[]byte(`[{"name":"structure","status":"passed","durationSeconds":1}]`),
		sql.NullTime{Time: now.Add(-time.Hour), Valid: true},
	)

	mock.ExpectQuery("WITH\\s+scenario_names").
		WillReturnRows(rows)

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
	if summary.LastExecutionPhaseSummary == nil || summary.LastExecutionPhaseSummary.Total != 1 {
		t.Fatalf("expected phase summary to be calculated: %#v", summary.LastExecutionPhaseSummary)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestScenarioDirectoryRepositoryGet(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewScenarioDirectoryRepository(db)
	rows := sqlmock.NewRows([]string{
		"scenario_name",
		"pending_requests",
		"total_requests",
		"last_request_at",
		"priority",
		"status",
		"notes",
		"coverage_target",
		"requested_types",
		"total_executions",
		"last_execution_at",
		"id",
		"preset_used",
		"success",
		"phases",
		"completed_at",
	}).AddRow(
		"ecosystem-manager",
		0,
		2,
		sql.NullTime{Valid: false},
		sql.NullString{Valid: false},
		sql.NullString{Valid: false},
		sql.NullString{Valid: false},
		sql.NullInt64{Valid: false},
		pq.StringArray{},
		0,
		sql.NullTime{Valid: false},
		nil,
		sql.NullString{String: "", Valid: false},
		sql.NullBool{Valid: false},
		nil,
		sql.NullTime{Valid: false},
	)

	mock.ExpectQuery("WITH\\s+scenario_names").
		WithArgs("ecosystem-manager").
		WillReturnRows(rows)

	summary, err := repo.Get(context.Background(), "ecosystem-manager")
	if err != nil {
		t.Fatalf("expected get to succeed: %v", err)
	}
	if summary == nil || summary.ScenarioName != "ecosystem-manager" {
		t.Fatalf("unexpected summary: %#v", summary)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
