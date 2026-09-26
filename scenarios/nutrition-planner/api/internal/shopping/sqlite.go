package shopping

import (
	"context"
	"database/sql"
	"time"

	"github.com/vrooli/api-core/schedule"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clock}
}

func (r *sqliteRepository) Checked(ctx context.Context, workspaceID string) (map[string]bool, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT line_key,checked FROM shopping_checks WHERE workspace_id=?`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var key string
		var checked int
		if err := rows.Scan(&key, &checked); err != nil {
			return nil, err
		}
		out[key] = checked != 0
	}
	return out, rows.Err()
}

func (r *sqliteRepository) SetChecked(ctx context.Context, workspaceID, key string, checked bool) error {
	value := 0
	if checked {
		value = 1
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO shopping_checks(workspace_id,line_key,checked,updated_at) VALUES(?,?,?,?) ON CONFLICT(workspace_id,line_key) DO UPDATE SET checked=excluded.checked,updated_at=excluded.updated_at`, workspaceID, key, value, r.clock.Now().UTC().Format(time.RFC3339Nano))
	return err
}
