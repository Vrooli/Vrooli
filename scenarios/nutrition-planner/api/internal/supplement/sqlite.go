package supplement

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
	"nutrition-planner/internal/decimalx"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clock}
}

func (r *sqliteRepository) Create(ctx context.Context, s Schedule) (Schedule, error) {
	if err := Validate(s); err != nil {
		return Schedule{}, err
	}
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	if s.Revision == 0 {
		s.Revision = 1
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = r.clock.Now().UTC()
	}
	now := s.CreatedAt.Format(time.RFC3339Nano)
	if _, err := r.db.ExecContext(ctx, `INSERT INTO supplement_schedules(id,workspace_id,current_revision,created_at,updated_at) VALUES(?,?,?,?,?)`, s.ID, s.WorkspaceID, s.Revision, now, now); err != nil {
		return Schedule{}, fmt.Errorf("insert supplement schedule: %w", err)
	}
	if err := r.insertRevision(ctx, s); err != nil {
		return Schedule{}, err
	}
	return s, nil
}

func (r *sqliteRepository) insertRevision(ctx context.Context, s Schedule) error {
	weekdays, _ := json.Marshal(s.Weekdays)
	_, err := r.db.ExecContext(ctx, `INSERT INTO supplement_schedule_revisions(schedule_id,revision,workspace_id,product_revision_id,dose,dose_unit,weekdays_json,start_date,end_date,paused,confirmed,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, s.ID, s.Revision, s.WorkspaceID, s.ProductRevisionID, s.Dose.String(), s.DoseUnit, string(weekdays), s.StartDate, s.EndDate, boolInt(s.Paused), boolInt(s.Confirmed), s.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("insert supplement revision: %w", err)
	}
	return nil
}

func (r *sqliteRepository) List(ctx context.Context, workspaceID string) ([]Schedule, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT s.id,v.revision,v.workspace_id,v.product_revision_id,v.dose,v.dose_unit,v.weekdays_json,v.start_date,v.end_date,v.paused,v.confirmed,v.created_at FROM supplement_schedules s JOIN supplement_schedule_revisions v ON v.schedule_id=s.id AND v.revision=s.current_revision WHERE s.workspace_id=? ORDER BY s.updated_at DESC,s.id`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Schedule
	for rows.Next() {
		v, e := scan(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) Get(ctx context.Context, id, workspaceID string, revision int64) (Schedule, error) {
	row := r.db.QueryRowContext(ctx, `SELECT schedule_id,revision,workspace_id,product_revision_id,dose,dose_unit,weekdays_json,start_date,end_date,paused,confirmed,created_at FROM supplement_schedule_revisions WHERE schedule_id=? AND revision=? AND workspace_id=?`, id, revision, workspaceID)
	v, err := scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Schedule{}, ErrNotFound{id, revision}
	}
	if err != nil {
		return Schedule{}, err
	}
	return v, nil
}

func (r *sqliteRepository) Update(ctx context.Context, in UpdateInput) (Schedule, error) {
	current, err := r.Get(ctx, in.ID, in.WorkspaceID, in.ExpectedRevision)
	if err != nil {
		return Schedule{}, err
	}
	var actual int64
	if err = r.db.QueryRowContext(ctx, `SELECT current_revision FROM supplement_schedules WHERE id=? AND workspace_id=?`, in.ID, in.WorkspaceID).Scan(&actual); err != nil {
		return Schedule{}, err
	}
	if actual != in.ExpectedRevision {
		return Schedule{}, ErrConflict{in.ID, in.ExpectedRevision, actual}
	}
	next := current
	next.Revision++
	next.Dose = in.Dose
	next.DoseUnit = in.DoseUnit
	next.StartDate = in.StartDate
	next.EndDate = in.EndDate
	next.Weekdays = in.Weekdays
	next.Paused = in.Paused
	next.Confirmed = in.Confirmed
	next.CreatedAt = r.clock.Now().UTC()
	if err = Validate(next); err != nil {
		return Schedule{}, err
	}
	if err = r.insertRevision(ctx, next); err != nil {
		return Schedule{}, err
	}
	if _, err = r.db.ExecContext(ctx, `UPDATE supplement_schedules SET current_revision=?,updated_at=? WHERE id=? AND workspace_id=? AND current_revision=?`, next.Revision, next.CreatedAt.Format(time.RFC3339Nano), next.ID, next.WorkspaceID, in.ExpectedRevision); err != nil {
		return Schedule{}, err
	}
	return next, nil
}

func scan(s interface{ Scan(...any) error }) (Schedule, error) {
	var v Schedule
	var dose, weekdays, created string
	var paused, confirmed int
	if err := s.Scan(&v.ID, &v.Revision, &v.WorkspaceID, &v.ProductRevisionID, &dose, &v.DoseUnit, &weekdays, &v.StartDate, &v.EndDate, &paused, &confirmed, &created); err != nil {
		return Schedule{}, err
	}
	var err error
	v.Dose, err = decimalx.Parse(dose)
	if err != nil {
		return Schedule{}, err
	}
	if err = json.Unmarshal([]byte(weekdays), &v.Weekdays); err != nil {
		return Schedule{}, err
	}
	v.Paused = paused != 0
	v.Confirmed = confirmed != 0
	v.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	return v, err
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
