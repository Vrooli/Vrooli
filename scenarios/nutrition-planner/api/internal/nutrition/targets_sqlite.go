package nutrition

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
	"nutrition-planner/internal/decimalx"
)

type TargetSQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type targetSQLiteRepository struct {
	db    TargetSQLExecutor
	clock schedule.Clock
}

func NewSQLiteTargetRepository(db TargetSQLExecutor, clock schedule.Clock) TargetRepository {
	return &targetSQLiteRepository{db: db, clock: clock}
}

func (r *targetSQLiteRepository) Create(ctx context.Context, t Target) (Target, error) {
	if err := ValidateTarget(t); err != nil {
		return Target{}, err
	}
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	if t.Revision == 0 {
		t.Revision = 1
	}
	if t.Provenance == "" {
		t.Provenance = "user_assertion"
	}
	if !t.Active {
		t.Active = true
	}
	var effectiveTo any
	if t.EffectiveTo != nil {
		effectiveTo = t.EffectiveTo.UTC().Format(time.RFC3339Nano)
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO nutrition_targets(id,revision,workspace_id,nutrient_id,lower_bound,upper_bound,period,scope,enforcement,provenance,effective_from,effective_to,active) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, t.ID, t.Revision, t.WorkspaceID, t.NutrientID, t.Lower.String(), t.Upper.String(), t.Period, t.Scope, t.Enforcement, t.Provenance, t.EffectiveFrom.UTC().Format(time.RFC3339Nano), effectiveTo, boolInt(t.Active))
	if err != nil {
		return Target{}, fmt.Errorf("insert nutrition target: %w", err)
	}
	return t, nil
}

func (r *targetSQLiteRepository) List(ctx context.Context, workspaceID string) ([]Target, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,revision,workspace_id,nutrient_id,lower_bound,upper_bound,period,scope,enforcement,provenance,effective_from,effective_to,active FROM nutrition_targets WHERE workspace_id=? ORDER BY effective_from DESC,id,revision DESC`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Target
	for rows.Next() {
		v, err := scanTarget(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *targetSQLiteRepository) Get(ctx context.Context, id, workspaceID string, revision int64) (Target, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id,revision,workspace_id,nutrient_id,lower_bound,upper_bound,period,scope,enforcement,provenance,effective_from,effective_to,active FROM nutrition_targets WHERE id=? AND revision=? AND workspace_id=?`, id, revision, workspaceID)
	v, err := scanTarget(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Target{}, TargetNotFound{id, revision}
	}
	if err != nil {
		return Target{}, err
	}
	return v, nil
}

func scanTarget(s interface{ Scan(...any) error }) (Target, error) {
	var t Target
	var lower, upper, from string
	var to sql.NullString
	var active int
	if err := s.Scan(&t.ID, &t.Revision, &t.WorkspaceID, &t.NutrientID, &lower, &upper, &t.Period, &t.Scope, &t.Enforcement, &t.Provenance, &from, &to, &active); err != nil {
		return Target{}, err
	}
	var err error
	t.Lower, err = decimalx.Parse(lower)
	if err != nil {
		return Target{}, err
	}
	t.Upper, err = decimalx.Parse(upper)
	if err != nil {
		return Target{}, err
	}
	t.EffectiveFrom, err = time.Parse(time.RFC3339Nano, from)
	if err != nil {
		return Target{}, err
	}
	if to.Valid {
		end, e := time.Parse(time.RFC3339Nano, to.String)
		if e != nil {
			return Target{}, e
		}
		t.EffectiveTo = &end
	}
	t.Active = active != 0
	return t, nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
