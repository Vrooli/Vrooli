package models

import (
	"context"
	"fmt"
)

// OpDefaultStore persists per-operation default-model overrides (the settings
// surface, IMG-P0-007). The seed's default_for is the baseline; a row here pins a
// different enabled model as the default for one operation.
type OpDefaultStore struct {
	db SQLExecutor
}

// NewOpDefaultStore constructs the op-default store over db.
func NewOpDefaultStore(db SQLExecutor) *OpDefaultStore { return &OpDefaultStore{db: db} }

// Set pins model id as the default for operation. An empty id clears the pin.
func (s *OpDefaultStore) Set(ctx context.Context, operation, id string) error {
	if operation == "" {
		return fmt.Errorf("models: operation is required")
	}
	if id == "" {
		return s.Clear(ctx, operation)
	}
	if _, err := s.db.ExecContext(ctx,
		"INSERT INTO op_default (operation, model_id) VALUES (?, ?) ON CONFLICT(operation) DO UPDATE SET model_id=excluded.model_id",
		operation, id); err != nil {
		return fmt.Errorf("models: set op default %q: %w", operation, err)
	}
	return nil
}

// Clear removes the default-model pin for operation.
func (s *OpDefaultStore) Clear(ctx context.Context, operation string) error {
	if _, err := s.db.ExecContext(ctx, "DELETE FROM op_default WHERE operation = ?", operation); err != nil {
		return fmt.Errorf("models: clear op default %q: %w", operation, err)
	}
	return nil
}

// Load returns the pinned default-model overrides keyed by operation.
func (s *OpDefaultStore) Load(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT operation, model_id FROM op_default")
	if err != nil {
		return nil, fmt.Errorf("models: load op defaults: %w", err)
	}
	defer rows.Close()

	out := make(map[string]string)
	for rows.Next() {
		var op, id string
		if err := rows.Scan(&op, &id); err != nil {
			return nil, fmt.Errorf("models: scan op default: %w", err)
		}
		out[op] = id
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("models: iterate op defaults: %w", err)
	}
	return out, nil
}

// Get returns the pinned default for operation ("" when none).
func (s *OpDefaultStore) Get(ctx context.Context, operation string) (string, error) {
	all, err := s.Load(ctx)
	if err != nil {
		return "", err
	}
	return all[operation], nil
}
