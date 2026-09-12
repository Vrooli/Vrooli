package registry

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

const overrideKey = "behavior_override"

type SQLStore interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type Store struct {
	db SQLStore
}

func NewStore(db SQLStore) *Store {
	return &Store{db: db}
}

func (s *Store) Override(ctx context.Context) (Override, error) {
	if s == nil || s.db == nil {
		return OverrideAuto, nil
	}
	var value string
	err := s.db.QueryRowContext(ctx, "SELECT value FROM integration_settings WHERE key = ?", overrideKey).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return OverrideAuto, nil
	}
	if err != nil {
		return OverrideAuto, fmt.Errorf("read integration override: %w", err)
	}
	return normalizeOverride(value), nil
}

func (s *Store) SetOverride(ctx context.Context, value Override, now time.Time) error {
	if s == nil || s.db == nil {
		return nil
	}
	normalized := normalizeOverride(string(value))
	_, err := s.db.ExecContext(ctx, `
INSERT INTO integration_settings (key, value, updated_at)
VALUES (?, ?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		overrideKey, string(normalized), now.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("persist integration override: %w", err)
	}
	return nil
}

func normalizeOverride(value string) Override {
	switch Override(strings.TrimSpace(value)) {
	case OverrideForceOff:
		return OverrideForceOff
	case OverrideForcePassive:
		return OverrideForcePassive
	default:
		return OverrideAuto
	}
}
