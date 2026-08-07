package maintenance

import (
	"context"
	"database/sql"
	"embed"
	"time"
)

//go:embed schema.sql
var schema embed.FS

func Schema() string {
	b, _ := schema.ReadFile("schema.sql")
	return string(b)
}

type SQLiteStore struct{ db *sql.DB }

func NewSQLiteStore(db *sql.DB) *SQLiteStore { return &SQLiteStore{db: db} }

func (s *SQLiteStore) Begin(ctx context.Context, run Run) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO maintenance_runs(id,started_at,completed_at) VALUES(?,?,?)`, run.ID, run.StartedAt.Format(time.RFC3339Nano), "")
	return err
}

func (s *SQLiteStore) PutOutcome(ctx context.Context, runID string, o Outcome) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO maintenance_outcomes(run_id,runtime,import_status,import_error,projection_status,projection_error,started_at,completed_at) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(run_id,runtime) DO UPDATE SET import_status=excluded.import_status,import_error=excluded.import_error,projection_status=excluded.projection_status,projection_error=excluded.projection_error,completed_at=excluded.completed_at`, runID, o.Runtime, o.ImportStatus, o.ImportError, o.ProjectionStatus, o.ProjectionError, o.StartedAt.Format(time.RFC3339Nano), formatTime(o.CompletedAt))
	return err
}

func (s *SQLiteStore) PutCompaction(ctx context.Context, runID string, c Compaction) error {
	_, err := s.db.ExecContext(ctx, `UPDATE maintenance_runs SET compaction_status=?,compaction_error=?,compacted_count=?,frontier_before=?,frontier_after=?,frontier_target=? WHERE id=?`,
		c.Status, c.Error, c.Compacted, c.FrontierBefore, c.FrontierAfter, c.Target, runID)
	return err
}

func (s *SQLiteStore) Complete(ctx context.Context, runID string, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE maintenance_runs SET completed_at=? WHERE id=?`, at.Format(time.RFC3339Nano), runID)
	return err
}

func (s *SQLiteStore) Latest(ctx context.Context) (Run, error) {
	var run Run
	var started, completed string
	err := s.db.QueryRowContext(ctx, `SELECT id,started_at,completed_at,compaction_status,compaction_error,compacted_count,frontier_before,frontier_after,frontier_target FROM maintenance_runs ORDER BY started_at DESC,id DESC LIMIT 1`).
		Scan(&run.ID, &started, &completed, &run.Compaction.Status, &run.Compaction.Error, &run.Compaction.Compacted, &run.Compaction.FrontierBefore, &run.Compaction.FrontierAfter, &run.Compaction.Target)
	if err != nil {
		return run, err
	}
	run.StartedAt, _ = time.Parse(time.RFC3339Nano, started)
	run.CompletedAt, _ = time.Parse(time.RFC3339Nano, completed)
	rows, err := s.db.QueryContext(ctx, `SELECT runtime,import_status,import_error,projection_status,projection_error,started_at,completed_at FROM maintenance_outcomes WHERE run_id=? ORDER BY runtime`, run.ID)
	if err != nil {
		return run, err
	}
	defer rows.Close()
	for rows.Next() {
		var o Outcome
		var start, end string
		if err := rows.Scan(&o.Runtime, &o.ImportStatus, &o.ImportError, &o.ProjectionStatus, &o.ProjectionError, &start, &end); err != nil {
			return run, err
		}
		o.StartedAt, _ = time.Parse(time.RFC3339Nano, start)
		if end != "" {
			o.CompletedAt, _ = time.Parse(time.RFC3339Nano, end)
		}
		run.Outcomes = append(run.Outcomes, o)
	}
	return run, rows.Err()
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}
