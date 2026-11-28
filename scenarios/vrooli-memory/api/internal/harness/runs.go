package harness

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
)

type ImportRunStatus string

const (
	ImportRunQueued              ImportRunStatus = "queued"
	ImportRunRunning             ImportRunStatus = "running"
	ImportRunCompleted           ImportRunStatus = "completed"
	ImportRunCompletedWithErrors ImportRunStatus = "completed_with_errors"
	ImportRunFailed              ImportRunStatus = "failed"
)

// importRunTransition is the sole owner of the durable import run state graph.
// Unknown persisted or requested states fail closed.
func importRunTransition(from, to ImportRunStatus) bool {
	switch from {
	case ImportRunQueued:
		return to == ImportRunRunning || to == ImportRunFailed
	case ImportRunRunning:
		return to == ImportRunCompleted || to == ImportRunCompletedWithErrors || to == ImportRunFailed
	default:
		return false
	}
}

func requireRunTransition(from, to ImportRunStatus) error {
	if !importRunTransition(from, to) {
		return fmt.Errorf("invalid import run transition %q -> %q", from, to)
	}
	return nil
}

func requireOneRunUpdated(result sql.Result, err error, id string) error {
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("import run %q is missing or no longer in the expected state", id)
	}
	return nil
}

type (
	ImportRun struct {
		ID, Runtime, SourceRoot                                                   string
		Status                                                                    ImportRunStatus
		TotalSources, ProcessedSources, ImportedCount, ExistingCount, FailedCount int
		CurrentPath, ErrorMessage                                                 string
		StartedAt, CompletedAt, UpdatedAt                                         time.Time
	}
	runStore struct {
		db    *sql.DB
		clock schedule.Clock
	}
)

func newRunStore(db *sql.DB, clk schedule.Clock) *runStore {
	if clk == nil {
		clk = schedule.System()
	}
	return &runStore{db: db, clock: clk}
}

func (s *runStore) cursor(ctx context.Context, runtime, root string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT cursor_path FROM harness_import_cursors WHERE runtime=? AND source_root=?`, runtime, root).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (s *runStore) setCursor(ctx context.Context, runtime, root, path string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO harness_import_cursors(runtime,source_root,cursor_path,updated_at) VALUES(?,?,?,?) ON CONFLICT(runtime) DO UPDATE SET source_root=excluded.source_root,cursor_path=excluded.cursor_path,updated_at=excluded.updated_at`, runtime, root, path, s.clock.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (s *runStore) create(ctx context.Context, runtime, root string, total int) (ImportRun, error) {
	now := s.clock.Now().UTC()
	r := ImportRun{ID: uuid.NewString(), Runtime: runtime, SourceRoot: root, Status: ImportRunQueued, TotalSources: total, StartedAt: now, UpdatedAt: now}
	_, err := s.db.ExecContext(ctx, `INSERT INTO harness_import_runs (id,runtime,source_root,status,total_sources,started_at,updated_at) VALUES (?,?,?,?,?,?,?)`, r.ID, r.Runtime, r.SourceRoot, r.Status, r.TotalSources, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	return r, err
}

func (s *runStore) running(ctx context.Context, id string) error {
	if err := requireRunTransition(ImportRunQueued, ImportRunRunning); err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx, `UPDATE harness_import_runs SET status=?,updated_at=? WHERE id=? AND status=?`, ImportRunRunning, s.clock.Now().UTC().Format(time.RFC3339Nano), id, ImportRunQueued)
	return requireOneRunUpdated(res, err, id)
}

func (s *runStore) progress(ctx context.Context, id string, processed, imported, existing, failed int, path string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE harness_import_runs SET processed_sources=?,imported_count=?,existing_count=?,failed_count=?,current_path=?,updated_at=? WHERE id=? AND status=?`, processed, imported, existing, failed, path, s.clock.Now().UTC().Format(time.RFC3339Nano), id, ImportRunRunning)
	return requireOneRunUpdated(res, err, id)
}

func (s *runStore) finish(ctx context.Context, id string, status ImportRunStatus, processed, imported, existing, failed int, msg string) error {
	if err := requireRunTransition(ImportRunRunning, status); err != nil {
		return err
	}
	now := s.clock.Now().UTC()
	res, err := s.db.ExecContext(ctx, `UPDATE harness_import_runs SET status=?,processed_sources=?,imported_count=?,existing_count=?,failed_count=?,error_message=?,completed_at=?,updated_at=? WHERE id=? AND status=?`, status, processed, imported, existing, failed, msg, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), id, ImportRunRunning)
	return requireOneRunUpdated(res, err, id)
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
