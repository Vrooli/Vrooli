package maintenance

import (
	"context"
	"database/sql"
	"fmt"
)

// Schema is owned next to its interpreter and installed through EnsureSchemas.
func Schema() string {
	return `
CREATE TABLE IF NOT EXISTS maintenance_admission (
 singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
 closed INTEGER NOT NULL CHECK (closed IN (0,1)),
 revision INTEGER NOT NULL,
 owner TEXT NOT NULL,
 reason TEXT NOT NULL
);
INSERT INTO maintenance_admission(singleton,closed,revision,owner,reason)
 VALUES(1,0,0,'','') ON CONFLICT(singleton) DO NOTHING;
`
}

type Database interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Repository struct{ db Database }

func NewRepository(db Database) *Repository { return &Repository{db: db} }

func (r *Repository) Load(ctx context.Context) (State, error) {
	var state State
	err := r.db.QueryRowContext(ctx, `SELECT closed,revision,owner,reason FROM maintenance_admission WHERE singleton=1`).Scan(&state.Closed, &state.Revision, &state.Owner, &state.Reason)
	if err != nil {
		return State{}, fmt.Errorf("load maintenance fence: %w", err)
	}
	return state, nil
}

func (r *Repository) Save(ctx context.Context, state State, revision int64) error {
	result, err := r.db.ExecContext(ctx, `UPDATE maintenance_admission SET closed=?,revision=?,owner=?,reason=? WHERE singleton=1 AND revision=?`, state.Closed, state.Revision, state.Owner, state.Reason, revision)
	if err != nil {
		return fmt.Errorf("persist maintenance fence: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrConflict
	}
	return nil
}
