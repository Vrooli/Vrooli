// Package database owns scenario-wide database infrastructure: the
// connection-level Pinger seam consumed by the health endpoint, and the
// system-schema home for cross-cutting bits that don't belong to any
// one domain (postgres extensions, custom types, cross-domain views).
//
// Domain-scoped persistence lives in internal/<domain>/ packages
// (e.g., internal/journal/{repository,sqlite,schema}.{go,sql}) per the
// canonical Vrooli pattern. New domains do NOT add Repository
// interfaces or schema fragments here — they own their own package and
// register through internal/modules.
//
// Production wires *database.RoutedDB (from packages/api-core/database)
// which satisfies Pinger via its PingContext method. *sql.DB also
// satisfies it; tests wire testutil/mocks.FakePinger or
// testutil/db.NewSQLite for real-handle repository tests.
package database

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
