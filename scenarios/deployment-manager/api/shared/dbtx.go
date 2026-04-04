package shared

import (
	"context"
	"database/sql"
)

// DBTX is the common interface satisfied by both *sql.DB and *sql.Tx,
// allowing repository methods to run inside or outside a transaction.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}
