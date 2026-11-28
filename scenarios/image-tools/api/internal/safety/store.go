package safety

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// SQLExecutor is the narrow database surface the consent log depends on
// (satisfied by both *sql.DB in tests and *database.RoutedDB in production).
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// ConsentLog persists consent affirmations for high-weight ops (audit). A row is
// written only when an affirmation unlocked something: the public tier's consent
// control, or a route that leaves this machine. A local run on the local tier
// records nothing.
type ConsentLog struct {
	db SQLExecutor
}

// NewConsentLog constructs the consent log over db.
func NewConsentLog(db SQLExecutor) *ConsentLog { return &ConsentLog{db: db} }

// EnsureSchema adds columns an install created before they existed. SQLite has
// no ADD COLUMN IF NOT EXISTS, and CREATE TABLE IF NOT EXISTS leaves an old table
// as it was, so a duplicate-column error is the steady state rather than a
// failure (the same rule the jobs store uses).
func (l *ConsentLog) EnsureSchema(ctx context.Context) error {
	if l == nil || l.db == nil {
		return nil
	}
	_, err := l.db.ExecContext(ctx, `ALTER TABLE consent_log ADD COLUMN route TEXT NOT NULL DEFAULT ''`)
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return fmt.Errorf("consent_log: add route column: %w", err)
	}
	return nil
}

// Record appends one consent-affirmation row. Best-effort audit: a write failure
// is returned so the caller can log it, but it should NOT fail the user's op (an
// audit-log hiccup must not block a consented, allowed edit).
func (l *ConsentLog) Record(ctx context.Context, op string, weight Weight, tier Tier, route string) error {
	if l == nil || l.db == nil {
		return nil
	}
	_, err := l.db.ExecContext(ctx,
		`INSERT INTO consent_log (operation, weight, tier, route) VALUES (?, ?, ?, ?)`,
		op, string(weight), string(tier), route)
	if err != nil {
		return fmt.Errorf("consent_log insert: %w", err)
	}
	return nil
}

// Count returns the number of recorded consent affirmations (for tests + audit).
func (l *ConsentLog) Count(ctx context.Context) (int, error) {
	if l == nil || l.db == nil {
		return 0, nil
	}
	var n int
	if err := l.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM consent_log`).Scan(&n); err != nil {
		return 0, fmt.Errorf("consent_log count: %w", err)
	}
	return n, nil
}
