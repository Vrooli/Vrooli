// Package store defines the storage seams the API depends on and the
// schema-bootstrap entry point invoked from main.go.
//
// Production wires *sql.DB opened against modernc.org/sqlite (which
// already satisfies Pinger via its PingContext method). Tests wire
// testutil/mocks.FakePinger or testutil/db.NewSQLite for real-handle
// repository tests.
//
// As scenarios grow beyond the health endpoint, additional interfaces
// (TaskStore, UserStore, etc.) live alongside Pinger here. Repository
// implementations live in their own packages; this package only owns
// the contracts and schema bootstrap (schema.sql + EnsureSchema).
package store

import (
	"context"
	"database/sql"
)

// Pinger is the minimum surface the health endpoint needs to verify the
// database is reachable. *sql.DB satisfies it directly.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// Compile-time guarantee that *sql.DB satisfies Pinger; production
// wires the real handle without an adapter.
var _ Pinger = (*sql.DB)(nil)
