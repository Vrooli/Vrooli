package discovery

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/api-core/schedule"
)

// SQLExecutor is the narrow database surface the sqlite dismissal store depends
// on. Both *sql.DB (tests) and *database.RoutedDB (production) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

const dismissalTimeFormat = time.RFC3339Nano

const (
	insertDismissalSQL = `
INSERT INTO discovery_dismissals (id, kind, dismissed_at)
VALUES (?, ?, ?)
ON CONFLICT(id) DO NOTHING
`
	selectDismissalSQL = `SELECT 1 FROM discovery_dismissals WHERE id = ?`
)

type sqliteDismissalStore struct {
	db    SQLExecutor
	clock schedule.Clock
}

// NewSQLiteDismissalStore constructs the production DismissalStore.
func NewSQLiteDismissalStore(db SQLExecutor, clk schedule.Clock) DismissalStore {
	return &sqliteDismissalStore{db: db, clock: clk}
}

// Compile-time guarantee.
var _ DismissalStore = (*sqliteDismissalStore)(nil)

func (s *sqliteDismissalStore) IsDismissed(ctx context.Context, id string) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx, selectDismissalSQL, id).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query dismissal %q: %w", id, err)
	}
	return true, nil
}

func (s *sqliteDismissalStore) Dismiss(ctx context.Context, id, kind string) error {
	now := s.clock.Now().UTC().Format(dismissalTimeFormat)
	if _, err := s.db.ExecContext(ctx, insertDismissalSQL, id, kind, now); err != nil {
		return fmt.Errorf("insert dismissal %q: %w", id, err)
	}
	return nil
}
