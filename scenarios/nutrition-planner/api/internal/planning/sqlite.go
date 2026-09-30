package planning

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/vrooli/api-core/schedule"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type Repository interface {
	Apply(context.Context, string, int64, string) (int64, error)
	Get(context.Context, string) (int64, string, error)
}
type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clock}
}

type ErrStaleInputs struct {
	WorkspaceID      string
	Expected, Actual int64
}

func (e ErrStaleInputs) Error() string {
	return fmt.Sprintf("plan inputs are stale for workspace %q: expected revision %d, actual %d", e.WorkspaceID, e.Expected, e.Actual)
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (int64, string, error) {
	var revision int64
	var plan string
	err := r.db.QueryRowContext(ctx, `SELECT revision,plan_json FROM plans WHERE workspace_id=?`, id).Scan(&revision, &plan)
	if err == sql.ErrNoRows {
		return 0, "", nil
	}
	return revision, plan, err
}

func (r *sqliteRepository) Apply(ctx context.Context, id string, expected int64, plan string) (int64, error) {
	next := expected + 1
	result, err := r.db.ExecContext(ctx, `INSERT INTO plans(workspace_id,revision,plan_json,updated_at) SELECT ?,?,?,? WHERE ?=0 ON CONFLICT(workspace_id) DO UPDATE SET revision=excluded.revision,plan_json=excluded.plan_json,updated_at=excluded.updated_at WHERE plans.revision=?`, id, next, plan, r.clock.Now().UTC().Format(time.RFC3339Nano), expected, expected)
	if err != nil {
		return 0, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if rows == 0 {
		actual, _, getErr := r.Get(ctx, id)
		if getErr != nil {
			return 0, getErr
		}
		return 0, ErrStaleInputs{id, expected, actual}
	}
	return next, nil
}
