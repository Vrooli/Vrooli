package models

import (
	"context"
	"database/sql"
	"fmt"
)

// SQLExecutor is the narrow database surface the model-state store depends on.
// Both *sql.DB (unit tests) and *database.RoutedDB (production) satisfy it, so
// production wiring participates in per-request routing without the test fixture
// wrapping its handle (seam-discovery §4).
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

// NewStore constructs the model-state store over db.
func NewStore(db SQLExecutor) *Store { return &Store{db: db} }

const upsertModelStateSQL = `
INSERT INTO model_state (id, enabled) VALUES (?, ?)
ON CONFLICT(id) DO UPDATE SET enabled = excluded.enabled`

// SetEnabled persists an explicit enabled override for model id.
func (s *Store) SetEnabled(ctx context.Context, id string, enabled bool) error {
	if id == "" {
		return fmt.Errorf("models: model id is required")
	}
	if _, err := s.db.ExecContext(ctx, upsertModelStateSQL, id, boolToInt(enabled)); err != nil {
		return fmt.Errorf("models: set enabled %q: %w", id, err)
	}
	return nil
}

// LoadOverlay returns the explicit enabled overrides keyed by model id. A model
// absent from the map uses its seed default.
func (s *Store) LoadOverlay(ctx context.Context) (map[string]bool, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, enabled FROM model_state")
	if err != nil {
		return nil, fmt.Errorf("models: load overlay: %w", err)
	}
	defer rows.Close()

	overlay := make(map[string]bool)
	for rows.Next() {
		var (
			id string
			en int
		)
		if err := rows.Scan(&id, &en); err != nil {
			return nil, fmt.Errorf("models: scan overlay row: %w", err)
		}
		overlay[id] = en != 0
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("models: iterate overlay: %w", err)
	}
	return overlay, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// EnabledWithOverlay returns an EnabledFunc that reports a model's effective
// enabled state: an explicit overlay override when present, else the seed
// default. Unknown ids report false. Pass the result to Registry.Select and use
// EffectiveEnabled for the wire shape so the two never disagree.
func (r *Registry) EnabledWithOverlay(overlay map[string]bool) EnabledFunc {
	return func(id string) bool {
		if v, ok := overlay[id]; ok {
			return v
		}
		if m, ok := r.byID[id]; ok {
			return m.Enabled
		}
		return false
	}
}

// EffectiveEnabled reports whether model m is enabled given the overlay (an
// explicit override wins; otherwise the seed default).
func EffectiveEnabled(m Model, overlay map[string]bool) bool {
	if v, ok := overlay[m.ID]; ok {
		return v
	}
	return m.Enabled
}
