package gates

import (
	"context"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
)

// openGateDB opens a gate's SQLite database at an explicit path.
//
// Gates read and write databases they are handed — a calibration source, a
// scratch target, the scenario's own catalog — so the path arrives as an
// argument rather than from the scenario's identity. What this helper removes
// is the eleven separately hand-assembled DSNs that used to sit inline at each
// call site, each with a slightly different pragma set: some set
// foreign_keys(ON) and some did not, so the same database enforced its foreign
// keys or ignored them depending on which gate opened it.
//
// The connection pool is capped at one because SQLite serializes writes.
func openGateDB(ctx context.Context, path string) (*database.RoutedDB, error) {
	dsn, err := storage.SQLiteDSNAt(path, storage.SQLiteTuning{})
	if err != nil {
		return nil, err
	}
	return database.Open(ctx, database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
}
