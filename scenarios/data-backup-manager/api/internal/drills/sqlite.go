package drills

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type sqliteRepository struct{ db SQLExecutor }

func NewSQLiteRepository(db SQLExecutor) Repository { return &sqliteRepository{db: db} }

const drillTime = time.RFC3339Nano

func (s *sqliteRepository) Create(ctx context.Context, d Drill) (Drill, error) {
	if d.Status != StatusRequested && d.Status != StatusFailed {
		return Drill{}, fmt.Errorf("new drill must begin requested or failed, got %q", d.Status)
	}
	if d.ID == "" {
		d.ID = uuid.NewString()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO recovery_drills
 (id,plan_id,target_id,destination_id,snapshot_id,restore_id,status,scheduled,idempotency_key,error,next_action,requested_at,started_at,finished_at)
 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, d.ID, d.PlanID, d.TargetID, d.DestinationID, d.SnapshotID, d.RestoreID, string(d.Status), boolInt(d.Scheduled), d.IdempotencyKey, d.Error, d.NextAction, formatTime(d.RequestedAt), formatTime(d.StartedAt), formatTime(d.FinishedAt))
	if err != nil {
		return Drill{}, fmt.Errorf("insert drill: %w", err)
	}
	return d, nil
}

func (s *sqliteRepository) Start(ctx context.Context, id string, startedAt time.Time) error {
	return s.persistTransition(ctx, id, EventStart, func(from, next Status) (sql.Result, error) {
		return s.db.ExecContext(ctx, `UPDATE recovery_drills SET status=?, started_at=? WHERE id=? AND status=?`, string(next), formatTime(startedAt), id, string(from))
	})
}

func (s *sqliteRepository) AttachRestore(ctx context.Context, id, restoreID string) error {
	if restoreID == "" {
		return errors.New("restore ID is required")
	}
	var status, existingRestoreID string
	err := s.db.QueryRowContext(ctx, `SELECT status, restore_id FROM recovery_drills WHERE id=?`, id).Scan(&status, &existingRestoreID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound{ID: id}
	}
	if err != nil {
		return fmt.Errorf("read recovery drill before linking restore: %w", err)
	}
	if Status(status) != StatusRunning || existingRestoreID != "" {
		return fmt.Errorf("cannot link restore to drill %q from status %q with existing restore %q", id, status, existingRestoreID)
	}
	result, err := s.db.ExecContext(ctx, `UPDATE recovery_drills SET restore_id=? WHERE id=? AND status=? AND restore_id=''`, restoreID, id, string(StatusRunning))
	return requireDrillWrite(result, err, id, "attach restore")
}

func (s *sqliteRepository) Finish(ctx context.Context, id string, event Event, errMsg, nextAction string, finishedAt time.Time) error {
	return s.persistTransition(ctx, id, event, func(from, next Status) (sql.Result, error) {
		return s.db.ExecContext(ctx, `UPDATE recovery_drills SET status=?, error=?, next_action=?, finished_at=? WHERE id=? AND status=?`, string(next), errMsg, nextAction, formatTime(finishedAt), id, string(from))
	})
}

func (s *sqliteRepository) persistTransition(ctx context.Context, id string, event Event, write func(Status, Status) (sql.Result, error)) error {
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT status FROM recovery_drills WHERE id=?`, id).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound{ID: id}
	}
	if err != nil {
		return fmt.Errorf("read recovery drill before transition: %w", err)
	}
	from := Status(raw)
	next, ok := Transition(from, event)
	if !ok {
		return fmt.Errorf("invalid recovery-drill transition %q + %q", from, event)
	}
	result, err := write(from, next)
	if err != nil {
		return fmt.Errorf("persist recovery-drill transition: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check recovery-drill transition: %w", err)
	}
	if rows != 1 {
		return fmt.Errorf("recovery drill %q changed after state %q was read; event %q was not applied", id, from, event)
	}
	return nil
}

func requireDrillWrite(result sql.Result, err error, id, action string) error {
	if err != nil {
		return fmt.Errorf("%s recovery drill %q: %w", action, id, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check %s recovery drill write: %w", action, err)
	}
	if rows != 1 {
		return fmt.Errorf("recovery drill %q changed while attempting to %s", id, action)
	}
	return nil
}

func (s *sqliteRepository) Get(ctx context.Context, id string) (Drill, error) {
	d, err := scanDrill(s.db.QueryRowContext(ctx, `SELECT id,plan_id,target_id,destination_id,snapshot_id,restore_id,status,scheduled,idempotency_key,error,next_action,requested_at,started_at,finished_at FROM recovery_drills WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Drill{}, ErrNotFound{ID: id}
	}
	if err != nil {
		return Drill{}, fmt.Errorf("get drill: %w", err)
	}
	return d, nil
}

func (s *sqliteRepository) List(ctx context.Context, planID, targetID string, limit int) ([]Drill, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,plan_id,target_id,destination_id,snapshot_id,restore_id,status,scheduled,idempotency_key,error,next_action,requested_at,started_at,finished_at FROM recovery_drills WHERE (?='' OR plan_id=?) AND (?='' OR target_id=?) ORDER BY requested_at DESC LIMIT ?`, planID, planID, targetID, targetID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Drill
	for rows.Next() {
		d, err := scanDrill(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *sqliteRepository) FindByIdempotency(ctx context.Context, key string) (Drill, bool, error) {
	if key == "" {
		return Drill{}, false, nil
	}
	d, err := scanDrill(s.db.QueryRowContext(ctx, `SELECT id,plan_id,target_id,destination_id,snapshot_id,restore_id,status,scheduled,idempotency_key,error,next_action,requested_at,started_at,finished_at FROM recovery_drills WHERE idempotency_key=?`, key))
	if errors.Is(err, sql.ErrNoRows) {
		return Drill{}, false, nil
	}
	if err != nil {
		return Drill{}, false, err
	}
	return d, true, nil
}

func (s *sqliteRepository) LatestForUnit(ctx context.Context, planID, targetID, destinationID string) (Drill, bool, error) {
	d, err := scanDrill(s.db.QueryRowContext(ctx, `SELECT id,plan_id,target_id,destination_id,snapshot_id,restore_id,status,scheduled,idempotency_key,error,next_action,requested_at,started_at,finished_at FROM recovery_drills WHERE plan_id=? AND target_id=? AND destination_id=? ORDER BY requested_at DESC LIMIT 1`, planID, targetID, destinationID))
	if errors.Is(err, sql.ErrNoRows) {
		return Drill{}, false, nil
	}
	if err != nil {
		return Drill{}, false, err
	}
	return d, true, nil
}

type scanner interface{ Scan(...any) error }

func scanDrill(sc scanner) (Drill, error) {
	var d Drill
	var scheduled int
	var status, requested, started, finished string
	if err := sc.Scan(&d.ID, &d.PlanID, &d.TargetID, &d.DestinationID, &d.SnapshotID, &d.RestoreID, &status, &scheduled, &d.IdempotencyKey, &d.Error, &d.NextAction, &requested, &started, &finished); err != nil {
		return Drill{}, err
	}
	d.Status = Status(status)
	if !ValidStatus(d.Status) {
		return Drill{}, fmt.Errorf("recovery drill %q has unknown status %q", d.ID, d.Status)
	}
	d.Scheduled = scheduled != 0
	var err error
	if d.RequestedAt, err = parseTime(requested); err != nil {
		return Drill{}, err
	}
	if d.StartedAt, err = parseTime(started); err != nil {
		return Drill{}, err
	}
	if d.FinishedAt, err = parseTime(finished); err != nil {
		return Drill{}, err
	}
	return d, nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(drillTime)
}

func parseTime(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, nil
	}
	return time.Parse(drillTime, raw)
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
