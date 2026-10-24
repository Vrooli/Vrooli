package sqlutil

import (
	"context"
	"database/sql"
)

// DB is the database surface shared by production's RoutedDB and the plain
// *sql.DB used by focused tests. Keeping repositories on this seam ensures
// requests marked for test mode use Test Genie's isolated pool.
type DB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}
