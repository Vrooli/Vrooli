package storage

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"strings"
	"time"

	"test-genie/internal/storage/sqlitedb"
	"test-genie/internal/storage/sqliteutil"

	pq "github.com/lib/pq"
	// Register modernc.org/sqlite as the pure-Go "sqlite" driver.
	_ "modernc.org/sqlite"

	runtimepkg "test-genie/internal/app/runtime"
)

type dbOpener func(driver, dsn string) (*sql.DB, error)

// ImportConfig describes a one-time migration from the legacy Postgres store
// into Test Genie's embedded SQLite database.
type ImportConfig struct {
	SourceDSN string
	Target    string
	Force     bool
}

// ImportResult summarizes what was copied into SQLite.
type ImportResult struct {
	TargetPath          string
	SuiteRequestCount   int
	SuiteExecutionCount int
}

type importer struct {
	open          dbOpener
	applySchema   func(*sql.DB, bool) error
	resolveTarget func(string) (sqlitedb.Config, error)
}

func runImportPostgres(args []string) error {
	fs := flag.NewFlagSet("import-postgres", flag.ContinueOnError)
	source := fs.String("source", "", "Legacy Postgres DSN to import from")
	target := fs.String("target", "", "SQLite database path or file: DSN (defaults to Test Genie runtime path)")
	force := fs.Bool("force", false, "Overwrite existing SQLite operational rows before importing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*source) == "" {
		return fmt.Errorf("--source is required")
	}

	result, err := importPostgres(context.Background(), ImportConfig{
		SourceDSN: *source,
		Target:    *target,
		Force:     *force,
	})
	if err != nil {
		return err
	}

	fmt.Printf("target: %s\n", result.TargetPath)
	fmt.Printf("suite_requests: %d\n", result.SuiteRequestCount)
	fmt.Printf("suite_executions: %d\n", result.SuiteExecutionCount)
	return nil
}

func importPostgres(ctx context.Context, cfg ImportConfig) (ImportResult, error) {
	imp := importer{
		open:        sql.Open,
		applySchema: runtimepkg.ApplySchema,
		resolveTarget: func(raw string) (sqlitedb.Config, error) {
			if strings.TrimSpace(raw) != "" {
				return sqlitedb.ResolveExplicit(raw)
			}
			return sqlitedb.Resolve()
		},
	}
	return imp.run(ctx, cfg)
}

func (i importer) run(ctx context.Context, cfg ImportConfig) (ImportResult, error) {
	if strings.TrimSpace(cfg.SourceDSN) == "" {
		return ImportResult{}, fmt.Errorf("source DSN is required")
	}

	targetCfg, err := i.resolveTarget(cfg.Target)
	if err != nil {
		return ImportResult{}, err
	}

	sourceDB, err := i.open("postgres", cfg.SourceDSN)
	if err != nil {
		return ImportResult{}, fmt.Errorf("open source postgres: %w", err)
	}
	defer sourceDB.Close()

	targetDB, err := i.open("sqlite", targetCfg.DSN)
	if err != nil {
		return ImportResult{}, fmt.Errorf("open target sqlite: %w", err)
	}
	defer targetDB.Close()

	if err := sourceDB.PingContext(ctx); err != nil {
		return ImportResult{}, fmt.Errorf("ping source postgres: %w", err)
	}
	if err := targetDB.PingContext(ctx); err != nil {
		return ImportResult{}, fmt.Errorf("ping target sqlite: %w", err)
	}
	if err := i.applySchema(targetDB, false); err != nil {
		return ImportResult{}, fmt.Errorf("prepare sqlite schema: %w", err)
	}

	requests, err := loadSuiteRequests(ctx, sourceDB)
	if err != nil {
		return ImportResult{}, err
	}
	executions, err := loadSuiteExecutions(ctx, sourceDB)
	if err != nil {
		return ImportResult{}, err
	}

	tx, err := targetDB.BeginTx(ctx, nil)
	if err != nil {
		return ImportResult{}, fmt.Errorf("begin sqlite transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := prepareTarget(ctx, tx, cfg.Force); err != nil {
		return ImportResult{}, err
	}
	if err := insertSuiteRequests(ctx, tx, requests); err != nil {
		return ImportResult{}, err
	}
	if err := insertSuiteExecutions(ctx, tx, executions); err != nil {
		return ImportResult{}, err
	}

	if err := tx.Commit(); err != nil {
		return ImportResult{}, fmt.Errorf("commit sqlite import: %w", err)
	}

	return ImportResult{
		TargetPath:          targetCfg.Path,
		SuiteRequestCount:   len(requests),
		SuiteExecutionCount: len(executions),
	}, nil
}

type suiteRequestRow struct {
	ID                string
	ScenarioName      string
	RequestedTypes    []string
	CoverageTarget    int
	Priority          string
	Status            string
	Notes             sql.NullString
	DelegationIssueID sql.NullString
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type suiteExecutionRow struct {
	ID                  string
	SuiteRequestID      sql.NullString
	ScenarioName        string
	PresetUsed          sql.NullString
	RequestedPreset     sql.NullString
	RequestedPhases     []string
	RequestedSkipPhases []string
	PlannedPhases       []string
	FailFast            bool
	Success             bool
	Phases              []byte
	StartedAt           time.Time
	CompletedAt         time.Time
}

func loadSuiteRequests(ctx context.Context, db *sql.DB) ([]suiteRequestRow, error) {
	const q = `
SELECT
	id::text,
	scenario_name,
	requested_types,
	coverage_target,
	priority,
	status,
	notes,
	delegation_issue_id,
	created_at,
	updated_at
FROM suite_requests
ORDER BY created_at ASC
`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query suite_requests from postgres: %w", err)
	}
	defer rows.Close()

	var results []suiteRequestRow
	for rows.Next() {
		var row suiteRequestRow
		var requestedTypes pq.StringArray
		if err := rows.Scan(
			&row.ID,
			&row.ScenarioName,
			&requestedTypes,
			&row.CoverageTarget,
			&row.Priority,
			&row.Status,
			&row.Notes,
			&row.DelegationIssueID,
			&row.CreatedAt,
			&row.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan suite_request row: %w", err)
		}
		row.RequestedTypes = append([]string(nil), requestedTypes...)
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate suite_requests: %w", err)
	}
	return results, nil
}

func loadSuiteExecutions(ctx context.Context, db *sql.DB) ([]suiteExecutionRow, error) {
	const q = `
SELECT
	id::text,
	suite_request_id::text,
	scenario_name,
	preset_used,
	requested_preset,
	requested_phases,
	requested_skip_phases,
	planned_phases,
	fail_fast,
	success,
	phases::text,
	started_at,
	completed_at
FROM suite_executions
ORDER BY completed_at ASC
`
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query suite_executions from postgres: %w", err)
	}
	defer rows.Close()

	var results []suiteExecutionRow
	for rows.Next() {
		var row suiteExecutionRow
		var requestedPhases pq.StringArray
		var requestedSkipPhases pq.StringArray
		var plannedPhases pq.StringArray
		if err := rows.Scan(
			&row.ID,
			&row.SuiteRequestID,
			&row.ScenarioName,
			&row.PresetUsed,
			&row.RequestedPreset,
			&requestedPhases,
			&requestedSkipPhases,
			&plannedPhases,
			&row.FailFast,
			&row.Success,
			&row.Phases,
			&row.StartedAt,
			&row.CompletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan suite_execution row: %w", err)
		}
		row.RequestedPhases = append([]string(nil), requestedPhases...)
		row.RequestedSkipPhases = append([]string(nil), requestedSkipPhases...)
		row.PlannedPhases = append([]string(nil), plannedPhases...)
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate suite_executions: %w", err)
	}
	return results, nil
}

func prepareTarget(ctx context.Context, tx *sql.Tx, force bool) error {
	requestCount, err := countRows(ctx, tx, "suite_requests")
	if err != nil {
		return err
	}
	executionCount, err := countRows(ctx, tx, "suite_executions")
	if err != nil {
		return err
	}
	if requestCount == 0 && executionCount == 0 {
		return nil
	}
	if !force {
		return fmt.Errorf("target sqlite database already contains operational data; rerun with --force to overwrite it")
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM suite_executions`); err != nil {
		return fmt.Errorf("clear suite_executions: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM suite_requests`); err != nil {
		return fmt.Errorf("clear suite_requests: %w", err)
	}
	return nil
}

func countRows(ctx context.Context, tx *sql.Tx, table string) (int, error) {
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, table)
	var count int
	if err := tx.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("count %s: %w", table, err)
	}
	return count, nil
}

func insertSuiteRequests(ctx context.Context, tx *sql.Tx, rows []suiteRequestRow) error {
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO suite_requests (
	id,
	scenario_name,
	requested_types,
	coverage_target,
	priority,
	status,
	notes,
	delegation_issue_id,
	created_at,
	updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`)
	if err != nil {
		return fmt.Errorf("prepare sqlite suite_request insert: %w", err)
	}
	defer stmt.Close()

	for _, row := range rows {
		requestedTypes, err := sqliteutil.MarshalStringSlice(row.RequestedTypes)
		if err != nil {
			return err
		}

		var note any
		if row.Notes.Valid {
			note = row.Notes.String
		}
		var delegation any
		if row.DelegationIssueID.Valid {
			delegation = row.DelegationIssueID.String
		}

		if _, err := stmt.ExecContext(
			ctx,
			row.ID,
			row.ScenarioName,
			requestedTypes,
			row.CoverageTarget,
			row.Priority,
			row.Status,
			note,
			delegation,
			sqliteutil.FormatTimestamp(row.CreatedAt),
			sqliteutil.FormatTimestamp(row.UpdatedAt),
		); err != nil {
			return fmt.Errorf("insert suite_request %s: %w", row.ID, err)
		}
	}
	return nil
}

func insertSuiteExecutions(ctx context.Context, tx *sql.Tx, rows []suiteExecutionRow) error {
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO suite_executions (
	id,
	suite_request_id,
	scenario_name,
	preset_used,
	requested_preset,
	requested_phases,
	requested_skip_phases,
	planned_phases,
	fail_fast,
	success,
	phases,
	started_at,
	completed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`)
	if err != nil {
		return fmt.Errorf("prepare sqlite suite_execution insert: %w", err)
	}
	defer stmt.Close()

	for _, row := range rows {
		requestedPhases, err := sqliteutil.MarshalStringSlice(row.RequestedPhases)
		if err != nil {
			return err
		}
		requestedSkipPhases, err := sqliteutil.MarshalStringSlice(row.RequestedSkipPhases)
		if err != nil {
			return err
		}
		plannedPhases, err := sqliteutil.MarshalStringSlice(row.PlannedPhases)
		if err != nil {
			return err
		}

		var suiteRequestID any
		if row.SuiteRequestID.Valid {
			suiteRequestID = row.SuiteRequestID.String
		}
		var presetUsed any
		if row.PresetUsed.Valid {
			presetUsed = row.PresetUsed.String
		}
		var requestedPreset any
		if row.RequestedPreset.Valid {
			requestedPreset = row.RequestedPreset.String
		}

		if _, err := stmt.ExecContext(
			ctx,
			row.ID,
			suiteRequestID,
			row.ScenarioName,
			presetUsed,
			requestedPreset,
			requestedPhases,
			requestedSkipPhases,
			plannedPhases,
			boolToInt(row.FailFast),
			boolToInt(row.Success),
			string(row.Phases),
			sqliteutil.FormatTimestamp(row.StartedAt),
			sqliteutil.FormatTimestamp(row.CompletedAt),
		); err != nil {
			return fmt.Errorf("insert suite_execution %s: %w", row.ID, err)
		}
	}
	return nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
