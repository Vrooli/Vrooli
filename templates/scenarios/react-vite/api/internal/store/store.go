// Package store owns the generic database infrastructure shared
// across domains: the connection-level Pinger seam consumed by the
// health endpoint, and the schema-bootstrap entry point invoked from
// main.go (EnsureSchema + schema.sql).
//
// Domain-scoped persistence lives in internal/<domain>/ packages
// (e.g., internal/notes/{repository,sqlite}.go) per the canonical
// Vrooli pattern. New domains do NOT add Repository interfaces here
// — they own their own package.
//
// Production wires *sql.DB opened against modernc.org/sqlite (which
// already satisfies Pinger via its PingContext method). Tests wire
// testutil/mocks.FakePinger or testutil/db.NewSQLite for real-handle
// repository tests.
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
