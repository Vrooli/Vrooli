package routine

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

func (r *sqliteRepository) Create(ctx context.Context, t Template) (Template, error) {
	if err := Validate(t); err != nil {
		return Template{}, err
	}
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	if t.Revision == 0 {
		t.Revision = 1
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = r.clock.Now().UTC()
	}
	now := t.CreatedAt.Format(time.RFC3339Nano)
	if _, err := r.db.ExecContext(ctx, `INSERT INTO routine_templates(id,workspace_id,current_revision,created_at,updated_at) VALUES(?,?,?,?,?)`, t.ID, t.WorkspaceID, t.Revision, now, now); err != nil {
		return Template{}, fmt.Errorf("insert routine template: %w", err)
	}
	if err := r.insertRevision(ctx, t); err != nil {
		return Template{}, err
	}
	return t, nil
}

func (r *sqliteRepository) insertRevision(ctx context.Context, t Template) error {
	days, _ := json.Marshal(t.Weekdays)
	_, err := r.db.ExecContext(ctx, `INSERT INTO routine_template_revisions(template_id,revision,workspace_id,slot_name,recipe_id,quantity,weekdays_json,start_date,end_date,mode,active,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, t.ID, t.Revision, t.WorkspaceID, t.SlotName, t.RecipeID, t.Quantity.String(), string(days), t.StartDate, t.EndDate, t.Mode, boolInt(t.Active), t.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("insert routine revision: %w", err)
	}
	return nil
}

func (r *sqliteRepository) List(ctx context.Context, workspaceID string) ([]Template, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT t.id,v.revision,v.workspace_id,v.slot_name,v.recipe_id,v.quantity,v.weekdays_json,v.start_date,v.end_date,v.mode,v.active,v.created_at FROM routine_templates t JOIN routine_template_revisions v ON v.template_id=t.id AND v.revision=t.current_revision WHERE t.workspace_id=? ORDER BY t.updated_at DESC,t.id`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Template
	for rows.Next() {
		v, e := scan(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) Get(ctx context.Context, id, workspaceID string, revision int64) (Template, error) {
	row := r.db.QueryRowContext(ctx, `SELECT template_id,revision,workspace_id,slot_name,recipe_id,quantity,weekdays_json,start_date,end_date,mode,active,created_at FROM routine_template_revisions WHERE template_id=? AND revision=? AND workspace_id=?`, id, revision, workspaceID)
	v, err := scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Template{}, ErrNotFound{id, revision}
	}
	if err != nil {
		return Template{}, err
	}
	return v, nil
}

func (r *sqliteRepository) Update(ctx context.Context, in UpdateInput) (Template, error) {
	current, err := r.Get(ctx, in.ID, in.WorkspaceID, in.ExpectedRevision)
	if err != nil {
		return Template{}, err
	}
	var actual int64
	if err = r.db.QueryRowContext(ctx, `SELECT current_revision FROM routine_templates WHERE id=? AND workspace_id=?`, in.ID, in.WorkspaceID).Scan(&actual); err != nil {
		return Template{}, err
	}
	if actual != in.ExpectedRevision {
		return Template{}, ErrConflict{in.ID, in.ExpectedRevision, actual}
	}
	next := current
	next.Revision++
	next.SlotName = in.SlotName
	next.RecipeID = in.RecipeID
	next.Quantity = in.Quantity
	next.Weekdays = in.Weekdays
	next.StartDate = in.StartDate
	next.EndDate = in.EndDate
	next.Mode = in.Mode
	next.Active = in.Active
	next.CreatedAt = r.clock.Now().UTC()
	if err = Validate(next); err != nil {
		return Template{}, err
	}
	if err = r.insertRevision(ctx, next); err != nil {
		return Template{}, err
	}
	if _, err = r.db.ExecContext(ctx, `UPDATE routine_templates SET current_revision=?,updated_at=? WHERE id=? AND workspace_id=? AND current_revision=?`, next.Revision, next.CreatedAt.Format(time.RFC3339Nano), next.ID, next.WorkspaceID, in.ExpectedRevision); err != nil {
		return Template{}, err
	}
	return next, nil
}

func scan(s interface{ Scan(...any) error }) (Template, error) {
	var t Template
	var quantity, days, created string
	var active int
	if err := s.Scan(&t.ID, &t.Revision, &t.WorkspaceID, &t.SlotName, &t.RecipeID, &quantity, &days, &t.StartDate, &t.EndDate, &t.Mode, &active, &created); err != nil {
		return Template{}, err
	}
	var err error
	t.Quantity, err = decimalx.Parse(quantity)
	if err != nil {
		return Template{}, err
	}
	if err = json.Unmarshal([]byte(days), &t.Weekdays); err != nil {
		return Template{}, err
	}
	t.Active = active != 0
	t.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	return t, err
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
