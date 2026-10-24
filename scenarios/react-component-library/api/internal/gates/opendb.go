package gates

import (
	"context"
	"os"

	"github.com/vrooli/api-core/database"
	"react-component-library/internal/gatedb"
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
	return gatedb.Open(ctx, path)
}

// removeGateDB deletes a throwaway gate database AND its write-ahead sidecars.
//
// Unlinking only the ".db" file is not enough once a database has been opened
// in WAL mode: SQLite leaves "-wal" and "-shm" beside it, and reopening the
// path then finds a write-ahead log with no database to belong to and fails
// with SQLITE_CANTOPEN. The gates create and discard these fixtures
// repeatedly, so the sidecars have to go with them.
func removeGateDB(path string) error {
	for _, suffix := range []string{"", "-wal", "-shm"} {
		if err := os.Remove(path + suffix); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
