package database

import (
	"github.com/vrooli/api-core/storage"
)

// DefaultDSN resolves flow-verifier's own database and returns its DSN.
//
// The path derives from this scenario's identity, so the CLI subcommands that
// call this and the API process that calls it always open the SAME file —
// which they did not when both read whatever SQLITE_PATH happened to hold.
func DefaultDSN() (string, error) {
	return storage.SQLiteDSN(storage.SQLiteConfig{Scenario: "flow-verifier"})
}

// FileDSN wraps an explicit filesystem path in the canonical SQLite DSN. Tests
// and one-off diagnostics pass a path here as an argument rather than exporting
// one into the environment.
func FileDSN(path string) (string, error) {
	return storage.SQLiteDSNAt(path, storage.SQLiteTuning{})
}
