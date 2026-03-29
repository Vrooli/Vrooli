package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	pq "github.com/lib/pq"
	_ "modernc.org/sqlite"

	runtimepkg "test-genie/internal/app/runtime"
	"test-genie/internal/storage/sqlitedb"
)

func TestImporterRun_MigratesLegacyPostgresRows(t *testing.T) {
	sourceDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer sourceDB.Close()

	now := time.Now().UTC()
	mock.ExpectQuery("SELECT\\s+id::text,\\s+scenario_name,\\s+requested_types").
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"scenario_name",
			"requested_types",
			"coverage_target",
			"priority",
			"status",
			"notes",
			"delegation_issue_id",
			"created_at",
			"updated_at",
		}).AddRow(
			"11111111-1111-1111-1111-111111111111",
			"demo",
			pq.StringArray{"unit", "integration"},
			95,
			"normal",
			"queued",
			"imported",
			nil,
			now.Add(-time.Minute),
			now,
		))

	mock.ExpectQuery("SELECT\\s+id::text,\\s+suite_request_id::text,\\s+scenario_name").
		WillReturnRows(sqlmock.NewRows([]string{
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
			"22222222-2222-2222-2222-222222222222",
			"11111111-1111-1111-1111-111111111111",
			"demo",
			"quick",
			"quick",
			pq.StringArray{"structure", "unit"},
			pq.StringArray{"performance"},
			pq.StringArray{"structure", "unit"},
			true,
			true,
			[]byte(`[{"name":"structure","status":"passed","durationSeconds":1}]`),
			now.Add(-2*time.Minute),
			now,
		))

	targetPath := t.TempDir() + "/imported.db"
	imp := importer{
		open: func(driver, dsn string) (*sql.DB, error) {
			if driver == "postgres" {
				return sourceDB, nil
			}
			return sql.Open("sqlite", dsn)
		},
		applySchema: runtimepkg.ApplySchema,
		resolveTarget: func(string) (sqlitedb.Config, error) {
			return sqlitedb.ResolveExplicit(targetPath)
		},
	}

	result, err := imp.run(context.Background(), ImportConfig{
		SourceDSN: "postgres://legacy",
	})
	if err != nil {
		t.Fatalf("importer.run: %v", err)
	}
	if result.SuiteRequestCount != 1 || result.SuiteExecutionCount != 1 {
		t.Fatalf("unexpected import counts: %#v", result)
	}

	targetDB, err := sql.Open("sqlite", sqlitedb.BuildDSN(targetPath))
	if err != nil {
		t.Fatalf("open target sqlite: %v", err)
	}
	defer targetDB.Close()

	var requestCount int
	if err := targetDB.QueryRow(`SELECT COUNT(*) FROM suite_requests`).Scan(&requestCount); err != nil {
		t.Fatalf("count suite_requests: %v", err)
	}
	if requestCount != 1 {
		t.Fatalf("expected 1 request row, got %d", requestCount)
	}

	var executionCount int
	if err := targetDB.QueryRow(`SELECT COUNT(*) FROM suite_executions`).Scan(&executionCount); err != nil {
		t.Fatalf("count suite_executions: %v", err)
	}
	if executionCount != 1 {
		t.Fatalf("expected 1 execution row, got %d", executionCount)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet postgres expectations: %v", err)
	}
}

func TestImporterRun_RejectsNonEmptyTargetWithoutForce(t *testing.T) {
	sourceDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer sourceDB.Close()

	mock.ExpectQuery("SELECT\\s+id::text,\\s+scenario_name,\\s+requested_types").
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"scenario_name",
			"requested_types",
			"coverage_target",
			"priority",
			"status",
			"notes",
			"delegation_issue_id",
			"created_at",
			"updated_at",
		}))
	mock.ExpectQuery("SELECT\\s+id::text,\\s+suite_request_id::text,\\s+scenario_name").
		WillReturnRows(sqlmock.NewRows([]string{
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
		}))

	targetPath := t.TempDir() + "/existing.db"
	targetDB, err := sql.Open("sqlite", sqlitedb.BuildDSN(targetPath))
	if err != nil {
		t.Fatalf("open target sqlite: %v", err)
	}
	if err := runtimepkg.ApplySchema(targetDB, false); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	if _, err := targetDB.Exec(`
INSERT INTO suite_requests (
	id, scenario_name, requested_types, coverage_target, priority, status, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`,
		"existing",
		"demo",
		`["unit"]`,
		95,
		"normal",
		"queued",
		time.Now().UTC().Format(time.RFC3339Nano),
		time.Now().UTC().Format(time.RFC3339Nano),
	); err != nil {
		t.Fatalf("seed target sqlite: %v", err)
	}
	targetDB.Close()

	imp := importer{
		open: func(driver, dsn string) (*sql.DB, error) {
			if driver == "postgres" {
				return sourceDB, nil
			}
			return sql.Open("sqlite", dsn)
		},
		applySchema: runtimepkg.ApplySchema,
		resolveTarget: func(string) (sqlitedb.Config, error) {
			return sqlitedb.ResolveExplicit(targetPath)
		},
	}

	if _, err := imp.run(context.Background(), ImportConfig{SourceDSN: "postgres://legacy"}); err == nil {
		t.Fatal("expected importer to reject non-empty target without --force")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet postgres expectations: %v", err)
	}
}
