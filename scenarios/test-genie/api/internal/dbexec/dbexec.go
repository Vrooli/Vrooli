// Package dbexec defines the narrow database-execution seam that test-genie's
// repositories and schema appliers depend on, instead of capturing a raw
// *sql.DB. Both the production handle (*database.RoutedDB) and the test handle
// (*sql.DB) satisfy it, so:
//
//   - production wires the RoutedDB, letting test-genie install a runtime test
//     pool on the live process without a restart (the routed test-DB path); and
//   - tests keep passing a plain *sql.DB fixture from internal/testsqlite.
//
// Capturing this interface (rather than *sql.DB) is what keeps test-genie
// clear of storage-health's SQL_DB_HANDLE_CAPTURE finding and thus eligible
// for the in-place routed e2e path.
package dbexec

import (
	"context"
	"database/sql"
)

// Executor is the subset of database methods test-genie persistence needs. It
// is deliberately narrow: the four context-aware methods every repository and
// the schema applier use. *sql.DB and *database.RoutedDB both satisfy it.
//
// seam: Executor is the database-handle seam. Production wires
// *database.RoutedDB (per-request routable); tests wire *sql.DB fixtures.
type Executor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}
