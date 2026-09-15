// Package dbx defines the narrow database surface web-console's SQL stores
// depend on.
//
// Every store takes a Handle rather than a concrete *sql.DB so production can
// pass a *database.RoutedDB — which routes each call to the leased test pool
// when the request context carries the X-Vrooli-Test-Mode marker — while tests
// keep passing a plain *sql.DB. Both satisfy Handle by construction.
//
// The interface is deliberately context-only: *database.RoutedDB has no
// ctx-free Exec/Query/QueryRow/Begin, because routing decisions are made per
// request from the context. A store that reaches for a ctx-free method is a
// store that cannot be isolated, so the type system rules it out.
package dbx

import (
	"context"
	"database/sql"
)

// Handle is the ctx-aware database surface shared by *sql.DB and
// *database.RoutedDB.
type Handle interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// Compile-time guarantee that the standard handle satisfies Handle. The
// RoutedDB side is asserted in api/main.go, where the import already exists.
var _ Handle = (*sql.DB)(nil)
