package review

import (
	"context"
	"database/sql"
)

// queryPort is the narrow SQL capability needed by read-only review sources.
// Keeping it as a port avoids coupling a projector to the process-wide pool.
type queryPort interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// queueStore is the SQL capability used by the durable review queue. Both the
// production pool and test transactions satisfy this contract.
type queueStore interface {
	queryPort
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}
