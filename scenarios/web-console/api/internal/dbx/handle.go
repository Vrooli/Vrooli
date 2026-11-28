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

// Querier is the read-only half of Handle. *database.RoutedReader satisfies
// it: reads routed the same way as the RoutedDB, served by its read pool.
type Querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// ReadWrite is a Handle whose standalone queries go to Reads while statements
// and transactions go to Writes. A store built on it keeps its single-writer
// discipline and lets its SELECTs run concurrently on a read pool.
//
// Only stores whose QueryContext/QueryRowContext calls are pure reads may use
// it: an INSERT ... RETURNING issued as a query would reach the read pool,
// which refuses it. Reads inside a transaction stay on the transaction's
// writer connection, so read-then-write logic belongs in a transaction.
type ReadWrite struct {
	Writes Handle
	Reads  Querier
}

func (h ReadWrite) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return h.Reads.QueryContext(ctx, query, args...)
}

func (h ReadWrite) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return h.Reads.QueryRowContext(ctx, query, args...)
}

func (h ReadWrite) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return h.Writes.ExecContext(ctx, query, args...)
}

func (h ReadWrite) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return h.Writes.BeginTx(ctx, opts)
}

var _ Handle = ReadWrite{}

// txBeginner is a read side that can hold one snapshot across statements, as
// *database.RoutedReader and *sql.DB can.
type txBeginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// ReadSnapshot runs read with every query inside one read-only transaction, so
// a result assembled from several statements describes one committed state
// rather than a mix of states that never existed together. A ReadWrite serves
// it from its read pool, which leaves the writer free however long the read
// takes; any other Handle serves it from its own connections.
func ReadSnapshot(ctx context.Context, h Handle, read func(Querier) error) error {
	begin := h.BeginTx
	if rw, ok := h.(ReadWrite); ok {
		if reads, ok := rw.Reads.(txBeginner); ok {
			begin = reads.BeginTx
		}
	}
	tx, err := begin(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // a read-only snapshot has nothing to commit
	return read(tx)
}
