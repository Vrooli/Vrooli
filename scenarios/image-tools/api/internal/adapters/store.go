package adapters

import (
	"context"
	"database/sql"
	"fmt"
)

// SQLExecutor is the narrow database surface the adapter stores depend on. Both
// *sql.DB (unit tests) and *database.RoutedDB (production) satisfy it, so
// production wiring participates in per-request routing without the test fixture
// wrapping its handle. Mirrors models.SQLExecutor.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Store persists the runtime enabled-state overlay over the seed catalog. The
// seed stays read-only; this store records ONLY explicit operator overrides.
type Store struct {
	db SQLExecutor
}

// NewStore constructs the adapter-state store over db.
func NewStore(db SQLExecutor) *Store { return &Store{db: db} }

const upsertAdapterStateSQL = `
INSERT INTO adapter_state (id, enabled) VALUES (?, ?)
ON CONFLICT(id) DO UPDATE SET enabled = excluded.enabled`

// SetEnabled persists an explicit enabled override for adapter id.
func (s *Store) SetEnabled(ctx context.Context, id string, enabled bool) error {
	if id == "" {
		return fmt.Errorf("adapters: adapter id is required")
	}
	if _, err := s.db.ExecContext(ctx, upsertAdapterStateSQL, id, boolToInt(enabled)); err != nil {
		return fmt.Errorf("adapters: set enabled %q: %w", id, err)
	}
	return nil
}

// LoadOverlay returns the explicit enabled overrides keyed by adapter id. An
// adapter absent from the map uses its seed default.
func (s *Store) LoadOverlay(ctx context.Context) (map[string]bool, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, enabled FROM adapter_state")
	if err != nil {
		return nil, fmt.Errorf("adapters: load overlay: %w", err)
	}
	defer rows.Close()

	overlay := make(map[string]bool)
	for rows.Next() {
		var (
			id string
			en int
		)
		if err := rows.Scan(&id, &en); err != nil {
			return nil, fmt.Errorf("adapters: scan overlay row: %w", err)
		}
		overlay[id] = en != 0
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("adapters: iterate overlay: %w", err)
	}
	return overlay, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// EffectiveEnabled reports whether adapter a is enabled given the overlay (an
// explicit override wins; otherwise the seed default).
func EffectiveEnabled(a Adapter, overlay map[string]bool) bool {
	if v, ok := overlay[a.ID]; ok {
		return v
	}
	return a.Enabled
}
