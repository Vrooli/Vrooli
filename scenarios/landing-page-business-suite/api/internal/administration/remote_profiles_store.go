package administration

import (
	"context"
	"database/sql"
)

// RemoteProfileStore is the context-aware persistence contract for remote
// profile configuration and encrypted remote sessions.
//
// seam: RemoteProfileStore keeps administration persistence independent of a
// concrete pool and preserves request-scoped test isolation.
type RemoteProfileStore interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}
