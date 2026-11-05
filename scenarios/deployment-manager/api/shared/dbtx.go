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

// RoutedDBTX is the repository-facing database seam. Both *sql.DB and
// api-core's *database.RoutedDB satisfy it, so repositories cannot capture a
// concrete production pool and test-mode requests remain routable.
type RoutedDBTX interface {
	DBTX
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	Conn(context.Context) (*sql.Conn, error)
}
