package feedback

import (
	"context"
	"database/sql"
	"time"

	"github.com/vrooli/api-core/schedule"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clock}
}
func (r *sqliteRepository) Record(ctx context.Context, in Feedback) (Feedback, error) {
	in.RecordedAt = r.clock.Now().UTC()
	_, err := r.db.ExecContext(ctx, `INSERT INTO meal_feedback(workspace_id,date,recipe_id,portion,minutes,recorded_at) VALUES(?,?,?,?,?,?) ON CONFLICT(workspace_id,date) DO UPDATE SET recipe_id=excluded.recipe_id,portion=excluded.portion,minutes=excluded.minutes,recorded_at=excluded.recorded_at`, in.WorkspaceID, in.Date, in.RecipeID, in.Portion, in.Minutes, in.RecordedAt.Format(time.RFC3339Nano))
	return in, err
}
func (r *sqliteRepository) Get(ctx context.Context, workspaceID, date string) (Feedback, error) {
	var out Feedback
	var recorded string
	err := r.db.QueryRowContext(ctx, `SELECT workspace_id,date,recipe_id,portion,minutes,recorded_at FROM meal_feedback WHERE workspace_id=? AND date=?`, workspaceID, date).Scan(&out.WorkspaceID, &out.Date, &out.RecipeID, &out.Portion, &out.Minutes, &recorded)
	if err == sql.ErrNoRows {
		return Feedback{}, ErrNotFound{workspaceID, date}
	}
	if err != nil {
		return Feedback{}, err
	}
	out.RecordedAt, err = time.Parse(time.RFC3339Nano, recorded)
	return out, err
}
func (r *sqliteRepository) Undo(ctx context.Context, workspaceID, date string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM meal_feedback WHERE workspace_id=? AND date=?`, workspaceID, date)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return ErrNotFound{workspaceID, date}
	}
	return err
}
