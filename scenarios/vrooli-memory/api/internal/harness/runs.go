package harness

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type ImportRunStatus string

const (
	ImportRunQueued              ImportRunStatus = "queued"
	ImportRunRunning             ImportRunStatus = "running"
	ImportRunCompleted           ImportRunStatus = "completed"
	ImportRunCompletedWithErrors ImportRunStatus = "completed_with_errors"
	ImportRunFailed              ImportRunStatus = "failed"
)

type (
	ImportRun struct {
		ID, Runtime, SourceRoot                                                   string
		Status                                                                    ImportRunStatus
		TotalSources, ProcessedSources, ImportedCount, ExistingCount, FailedCount int
		CurrentPath, ErrorMessage                                                 string
		StartedAt, CompletedAt, UpdatedAt                                         time.Time
	}
	runStore struct{ db *sql.DB }
)

func newRunStore(db *sql.DB) *runStore { return &runStore{db: db} }

func (s *runStore) cursor(ctx context.Context, runtime, root string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT cursor_path FROM harness_import_cursors WHERE runtime=? AND source_root=?`, runtime, root).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (s *runStore) setCursor(ctx context.Context, runtime, root, path string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO harness_import_cursors(runtime,source_root,cursor_path,updated_at) VALUES(?,?,?,?) ON CONFLICT(runtime) DO UPDATE SET source_root=excluded.source_root,cursor_path=excluded.cursor_path,updated_at=excluded.updated_at`, runtime, root, path, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (s *runStore) create(ctx context.Context, runtime, root string, total int) (ImportRun, error) {
	now := time.Now().UTC()
	r := ImportRun{ID: uuid.NewString(), Runtime: runtime, SourceRoot: root, Status: ImportRunQueued, TotalSources: total, StartedAt: now, UpdatedAt: now}
	_, err := s.db.ExecContext(ctx, `INSERT INTO harness_import_runs (id,runtime,source_root,status,total_sources,started_at,updated_at) VALUES (?,?,?,?,?,?,?)`, r.ID, r.Runtime, r.SourceRoot, r.Status, r.TotalSources, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	return r, err
}

func (s *runStore) running(ctx context.Context, id string) error {
	return s.update(ctx, id, ImportRunRunning, 0, 0, 0, 0, "", "")
}

func (s *runStore) progress(ctx context.Context, id string, processed, imported, existing, failed int, path string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE harness_import_runs SET processed_sources=?,imported_count=?,existing_count=?,failed_count=?,current_path=?,updated_at=? WHERE id=?`, processed, imported, existing, failed, path, time.Now().UTC().Format(time.RFC3339Nano), id)
	return err
}

func (s *runStore) finish(ctx context.Context, id string, status ImportRunStatus, processed, imported, existing, failed int, msg string) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `UPDATE harness_import_runs SET status=?,processed_sources=?,imported_count=?,existing_count=?,failed_count=?,error_message=?,completed_at=?,updated_at=? WHERE id=?`, status, processed, imported, existing, failed, msg, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), id)
	return err
}

func (s *runStore) update(ctx context.Context, id string, status ImportRunStatus, processed, imported, existing, failed int, path, msg string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE harness_import_runs SET status=?,updated_at=? WHERE id=?`, status, time.Now().UTC().Format(time.RFC3339Nano), id)
	return err
}

func (s *runStore) get(ctx context.Context, id string) (ImportRun, error) {
	return s.one(ctx, `WHERE id=?`, id)
}

func (s *runStore) latest(ctx context.Context, runtime string) (ImportRun, error) {
	return s.one(ctx, `WHERE runtime=? ORDER BY updated_at DESC LIMIT 1`, runtime)
}

func (s *runStore) one(ctx context.Context, tail string, args ...any) (ImportRun, error) {
	var r ImportRun
	var started, completed, updated string
	err := s.db.QueryRowContext(ctx, `SELECT id,runtime,source_root,status,total_sources,processed_sources,imported_count,existing_count,failed_count,current_path,error_message,started_at,completed_at,updated_at FROM harness_import_runs `+tail, args...).Scan(&r.ID, &r.Runtime, &r.SourceRoot, &r.Status, &r.TotalSources, &r.ProcessedSources, &r.ImportedCount, &r.ExistingCount, &r.FailedCount, &r.CurrentPath, &r.ErrorMessage, &started, &completed, &updated)
	if err != nil {
		return r, err
	}
	r.StartedAt, _ = time.Parse(time.RFC3339Nano, started)
	if completed != "" {
		r.CompletedAt, _ = time.Parse(time.RFC3339Nano, completed)
	}
	r.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return r, nil
}
