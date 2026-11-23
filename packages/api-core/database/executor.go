package database

import (
	"context"
	"database/sql"
)

// SQLExecutor is the context-aware query and mutation surface used by domain
// repositories. The schema helpers remain smaller so their callers can use
// partial database fakes.
type SQLExecutor interface {
	SchemaExecer
	SchemaQuerier
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// SQLTxExecutor is the SQLExecutor surface for repositories that perform
// operations inside database transactions.
type SQLTxExecutor interface {
	SQLExecutor
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

var (
	_ SQLExecutor   = (*sql.DB)(nil)
	_ SQLExecutor   = (*RoutedDB)(nil)
	_ SQLTxExecutor = (*sql.DB)(nil)
	_ SQLTxExecutor = (*RoutedDB)(nil)
)
