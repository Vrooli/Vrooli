package dbschema

import (
	"context"
	"database/sql"
)

// DB is the small database surface shared by GCT repositories. It is
// implemented by both *sql.DB and api-core's *database.RoutedDB, allowing
// request-scoped queries to select the Test Genie lease when test mode is
// active without coupling domain packages to the routing implementation.
type DB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	PingContext(context.Context) error
	Close() error
}
