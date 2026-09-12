package investigations

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type SQLDB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type SQLiteRepository struct{ db SQLDB }

func NewSQLiteRepository(db SQLDB) *SQLiteRepository { return &SQLiteRepository{db: db} }

func utcText(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func (r *SQLiteRepository) SaveRun(ctx context.Context, run Run) error {
	if run.StartedAt.IsZero() || run.CompletedAt.IsZero() {
		return fmt.Errorf("investigation run timestamps are required")
	}
	_, err := r.db.ExecContext(ctx, `INSERT OR REPLACE INTO investigation_runs
        (id, entry_id, execution_mode, status, skip_reason, exit_code, timed_out, started_at, completed_at, duration_seconds, host_os, host_arch, result_json, stderr_tail, anomaly_id)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID, run.EntryID, run.ExecutionMode, run.Status, run.SkipReason, run.ExitCode, boolInt(run.TimedOut), utcText(run.StartedAt), utcText(run.CompletedAt), run.DurationSeconds, run.HostOS, run.HostArch, run.ResultJSON, run.StderrTail, run.AnomalyID)
	if err != nil {
		return fmt.Errorf("save investigation run: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, "DELETE FROM investigation_findings WHERE run_id = ?", run.ID); err != nil {
		return fmt.Errorf("replace investigation findings: %w", err)
	}
	for _, finding := range run.Findings {
		if _, err := r.db.ExecContext(ctx, `INSERT INTO investigation_findings (run_id, severity, code, summary, detail_json) VALUES (?, ?, ?, ?, ?)`, run.ID, finding.Severity, finding.Code, finding.Summary, finding.DetailJSON); err != nil {
			return fmt.Errorf("save investigation finding: %w", err)
		}
	}
	return nil
}

func (r *SQLiteRepository) GetRun(ctx context.Context, id string) (Run, error) {
	var run Run
	var started, completed string
	var timedOut int
	err := r.db.QueryRowContext(ctx, `SELECT id, entry_id, execution_mode, status, skip_reason, exit_code, timed_out, started_at, completed_at, duration_seconds, host_os, host_arch, result_json, stderr_tail, anomaly_id FROM investigation_runs WHERE id = ?`, id).
		Scan(&run.ID, &run.EntryID, &run.ExecutionMode, &run.Status, &run.SkipReason, &run.ExitCode, &timedOut, &started, &completed, &run.DurationSeconds, &run.HostOS, &run.HostArch, &run.ResultJSON, &run.StderrTail, &run.AnomalyID)
	if err != nil {
		return Run{}, err
	}
	run.TimedOut = timedOut != 0
	run.StartedAt, err = parseUTC(started)
	if err != nil {
		return Run{}, err
	}
	run.CompletedAt, err = parseUTC(completed)
	if err != nil {
		return Run{}, err
	}
	run.Findings, err = r.findings(ctx, run.ID)
	return run, err
}

func (r *SQLiteRepository) ListRuns(ctx context.Context, entryID string, since time.Time, limit int) ([]Run, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	query := `SELECT id, entry_id, execution_mode, status, skip_reason, exit_code, timed_out, started_at, completed_at, duration_seconds, host_os, host_arch, result_json, stderr_tail, anomaly_id FROM investigation_runs WHERE 1=1`
	args := make([]any, 0, 3)
	if strings.TrimSpace(entryID) != "" {
		query += " AND entry_id = ?"
		args = append(args, entryID)
	}
	if !since.IsZero() {
		query += " AND started_at >= ?"
		args = append(args, utcText(since))
	}
	query += " ORDER BY started_at DESC LIMIT ?"
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]Run, 0)
	for rows.Next() {
		var run Run
		var started, completed string
		var timedOut int
		if err := rows.Scan(&run.ID, &run.EntryID, &run.ExecutionMode, &run.Status, &run.SkipReason, &run.ExitCode, &timedOut, &started, &completed, &run.DurationSeconds, &run.HostOS, &run.HostArch, &run.ResultJSON, &run.StderrTail, &run.AnomalyID); err != nil {
			return nil, err
		}
		run.TimedOut = timedOut != 0
		run.StartedAt, err = parseUTC(started)
		if err != nil {
			return nil, err
		}
		run.CompletedAt, err = parseUTC(completed)
		if err != nil {
			return nil, err
		}
		result = append(result, run)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range result {
		result[i].Findings, err = r.findings(ctx, result[i].ID)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (r *SQLiteRepository) findings(ctx context.Context, runID string) ([]Finding, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT severity, code, summary, detail_json FROM investigation_findings WHERE run_id = ? ORDER BY id", runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	findings := make([]Finding, 0)
	for rows.Next() {
		var f Finding
		if err := rows.Scan(&f.Severity, &f.Code, &f.Summary, &f.DetailJSON); err != nil {
			return nil, err
		}
		findings = append(findings, f)
	}
	return findings, rows.Err()
}

func (r *SQLiteRepository) PruneRunsBefore(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.ExecContext(ctx, "DELETE FROM investigation_runs WHERE completed_at < ?", utcText(before))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *SQLiteRepository) CountRunsBefore(ctx context.Context, before time.Time) (int64, error) {
	var count int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM investigation_runs WHERE completed_at < ?", utcText(before)).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func parseUTC(value string) (time.Time, error) {
	formats := []string{time.RFC3339Nano, "2006-01-02T15:04:05.999999999-0700", "2006-01-02T15:04:05-0700"}
	for _, format := range formats {
		if parsed, err := time.Parse(format, value); err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("parse investigation timestamp %q", value)
}
func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func tail(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[len(value)-limit:]
}
