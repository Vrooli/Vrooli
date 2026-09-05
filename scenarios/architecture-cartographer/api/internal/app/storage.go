package app

import (
	"github.com/vrooli/api-core/storage"
)

// SQLiteDSN returns the DSN for architecture-cartographer's own database.
//
// The path comes from the scenario's identity rather than the environment, so a
// supervisor that restarts this scenario cannot redirect it at another
// scenario's file. See packages/api-core/storage/sqlite.go.
func SQLiteDSN() (string, error) {
	return storage.SQLiteDSN(storage.SQLiteConfig{Scenario: "architecture-cartographer"})
}
