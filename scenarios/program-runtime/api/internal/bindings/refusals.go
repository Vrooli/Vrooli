package bindings

import (
	"context"
	"fmt"
	"time"

	"program-runtime/internal/sessions"
)

// RefusalRecorder is the narrow seam used by the private binding bridge. It
// keeps governance failures durable without coupling the bridge to SQLite.
type RefusalRecorder interface {
	RecordRefusal(context.Context, string, string, string, time.Time) error
}

type UnresolvedRecorder interface {
	RecordUnresolved(context.Context, string, string, time.Time) error
}

type refusalRepository struct{ db sessions.SQLExecutor }

// NewRefusalRepository creates a repository over the caller-owned database
// handle. The handle is normally the serving RoutedDB primary connection.
func NewRefusalRepository(db sessions.SQLExecutor) RefusalRecorder {
	return &refusalRepository{db: db}
}

func (r *refusalRepository) RecordRefusal(ctx context.Context, sessionID, bindingID, reason string, occurredAt time.Time) error {
	if _, err := r.db.ExecContext(ctx, `INSERT INTO refusals (session_id, binding_id, reason, occurred_at) VALUES (?, ?, ?, ?)`, sessionID, bindingID, reason, occurredAt.UTC().Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("record binding refusal: %w", err)
	}
	return nil
}

func (r *refusalRepository) RecordUnresolved(ctx context.Context, sessionID, attemptedName string, occurredAt time.Time) error {
	if _, err := r.db.ExecContext(ctx, `INSERT INTO unresolved_binding_attempts (session_id, attempted_name, occurred_at) VALUES (?, ?, ?)`, sessionID, attemptedName, occurredAt.UTC().Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("record unresolved binding attempt: %w", err)
	}
	return nil
}
