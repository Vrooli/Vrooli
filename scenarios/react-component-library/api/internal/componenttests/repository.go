package componenttests

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

type Repository interface {
	Save(context.Context, Report) error
	Get(context.Context, string) (Report, error)
	List(context.Context, string, string, int) ([]Report, error)
}

type SQLiteRepository struct{ db *sql.DB }

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository { return &SQLiteRepository{db: db} }

func (r *SQLiteRepository) Save(ctx context.Context, report Report) error {
	results, err := json.Marshal(struct {
		Results   []Result   `json:"results"`
		Artifacts []Artifact `json:"artifacts"`
	}{Results: report.Results, Artifacts: report.Artifacts})
	if err != nil {
		return fmt.Errorf("encode report results: %w", err)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	created := report.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
	if _, err = tx.ExecContext(ctx, `INSERT INTO component_test_reports (id, component_id, root_library_id, root_version, include_closure, created_at, verdict, results_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, report.ID, report.RootComponentID, report.RootLibraryID, report.RootVersion, report.IncludeClosure, created, report.Verdict, string(results)); err != nil {
		return err
	}
	if err = updateRollup(ctx, tx, report); err != nil {
		return err
	}
	if err = retainVersionReports(ctx, tx, report.RootComponentID, report.RootLibraryID, report.RootVersion); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SQLiteRepository) Get(ctx context.Context, id string) (Report, error) {
	rows, err := r.query(ctx, `WHERE id = ?`, id, 1)
	if err != nil {
		return Report{}, err
	}
	if len(rows) == 0 {
		return Report{}, sql.ErrNoRows
	}
	return rows[0], nil
}

func (r *SQLiteRepository) List(ctx context.Context, componentID, version string, limit int) ([]Report, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	where := []string{}
	args := []any{}
	if componentID != "" {
		where = append(where, "component_id = ?")
		args = append(args, componentID)
	}
	if version != "" {
		where = append(where, "root_version = ?")
		args = append(args, version)
	}
	if len(where) == 0 {
		return r.query(ctx, "", nil, limit)
	}
	return r.query(ctx, "WHERE "+strings.Join(where, " AND "), args, limit)
}

func updateRollup(ctx context.Context, tx *sql.Tx, report Report) error {
	passed, failed, blocked := 0, 0, 0
	switch report.Verdict {
	case VerdictPassed:
		passed = 1
	case VerdictFailed:
		failed = 1
	case VerdictBlocked:
		blocked = 1
	}
	_, err := tx.ExecContext(ctx, `
INSERT INTO component_version_test_rollup
  (library_id, version, runs_total, runs_passed, runs_failed, runs_blocked, first_pass_report_id, first_fail_report_id, last_run_at)
VALUES (?, ?, 1, ?, ?, ?, CASE WHEN ? = 1 THEN ? ELSE '' END, CASE WHEN ? = 1 THEN ? ELSE '' END, ?)
ON CONFLICT(library_id, version) DO UPDATE SET
  runs_total = runs_total + 1,
  runs_passed = runs_passed + excluded.runs_passed,
  runs_failed = runs_failed + excluded.runs_failed,
  runs_blocked = runs_blocked + excluded.runs_blocked,
  first_pass_report_id = CASE WHEN component_version_test_rollup.first_pass_report_id = '' AND excluded.first_pass_report_id <> '' THEN excluded.first_pass_report_id ELSE component_version_test_rollup.first_pass_report_id END,
  first_fail_report_id = CASE WHEN component_version_test_rollup.first_fail_report_id = '' AND excluded.first_fail_report_id <> '' THEN excluded.first_fail_report_id ELSE component_version_test_rollup.first_fail_report_id END,
  last_run_at = CASE WHEN excluded.last_run_at > component_version_test_rollup.last_run_at THEN excluded.last_run_at ELSE component_version_test_rollup.last_run_at END`,
		report.RootLibraryID, report.RootVersion, passed, failed, blocked, passed, report.ID, failed, report.ID, report.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"))
	return err
}

func retainVersionReports(ctx context.Context, tx *sql.Tx, componentID, libraryID, version string) error {
	_, err := tx.ExecContext(ctx, `
WITH ranked AS (
  SELECT id, ROW_NUMBER() OVER (ORDER BY created_at DESC, id DESC) AS rank
  FROM component_test_reports WHERE component_id = ? AND root_version = ?
), pinned AS (
  SELECT first_pass_report_id AS id FROM component_version_test_rollup WHERE library_id = ? AND version = ? AND first_pass_report_id <> ''
  UNION ALL
  SELECT first_fail_report_id FROM component_version_test_rollup WHERE library_id = ? AND version = ? AND first_fail_report_id <> ''
)
DELETE FROM component_test_reports
WHERE id IN (SELECT id FROM ranked WHERE rank > 5)
		AND id NOT IN (SELECT id FROM pinned)`, componentID, version, libraryID, version, libraryID, version)
	return err
}

func (r *SQLiteRepository) query(ctx context.Context, where string, arg any, limit int) ([]Report, error) {
	query := `SELECT id, component_id, root_library_id, root_version, include_closure, created_at, verdict, results_json FROM component_test_reports ` + where + ` ORDER BY created_at DESC LIMIT ?`
	args := []any{}
	if arg != nil {
		if many, ok := arg.([]any); ok {
			args = append(args, many...)
		} else {
			args = append(args, arg)
		}
	}
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Report{}
	for rows.Next() {
		var report Report
		var created string
		var results string
		if err := rows.Scan(&report.ID, &report.RootComponentID, &report.RootLibraryID, &report.RootVersion, &report.IncludeClosure, &created, &report.Verdict, &results); err != nil {
			return nil, err
		}
		if err := report.CreatedAt.UnmarshalText([]byte(created)); err != nil {
			return nil, err
		}
		persisted, err := decodePersistedResults(results)
		if err != nil {
			return nil, err
		}
		report.Results, report.Artifacts = persisted.Results, persisted.Artifacts
		out = append(out, report)
	}
	return out, rows.Err()
}

// decodePersistedResults preserves execution evidence created before artifacts
// were added to the report envelope. Those reports are durable user history,
// not invalid records, so they are normalized on read with an empty artifact
// collection instead of making the entire history unavailable.
func decodePersistedResults(raw string) (struct {
	Results   []Result   `json:"results"`
	Artifacts []Artifact `json:"artifacts"`
}, error,
) {
	var persisted struct {
		Results   []Result   `json:"results"`
		Artifacts []Artifact `json:"artifacts"`
	}
	if err := json.Unmarshal([]byte(raw), &persisted); err == nil {
		return persisted, nil
	}
	if !strings.HasPrefix(strings.TrimSpace(raw), "[") {
		return persisted, fmt.Errorf("decode report results: %w", json.Unmarshal([]byte(raw), &persisted))
	}
	if err := json.Unmarshal([]byte(raw), &persisted.Results); err != nil {
		return persisted, fmt.Errorf("decode legacy report results: %w", err)
	}
	return persisted, nil
}
