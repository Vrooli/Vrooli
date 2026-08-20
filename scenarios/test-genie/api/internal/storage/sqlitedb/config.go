// Package sqlitedb resolves where Test Genie's own run ledger lives.
//
// Test Genie is both a scenario with a database and the process that starts
// other scenarios, which makes it the worst possible place to trust an
// environment variable for a database path. It previously accepted
// TEST_GENIE_SQLITE_PATH, then SQLITE_PATH, then SQLITE_DB, all ahead of its
// own identity. The generic pair is what let a supervisor's inherited
// environment redirect this ledger into another scenario's file: 146 Test Genie
// runs were recorded inside vrooli-autoheal's 9.35 GB database, behind that
// database's single writer lock.
//
// Resolution now reads no database-path variable at all. The path is a function
// of this scenario's own identity, resolved by the one owned seam in
// api-core/storage. A caller that needs an explicit file — a test, a cutover,
// an isolated run — passes it as an argument to ResolveExplicit.
package sqlitedb

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/storage"
)

// scenarioID is this scenario naming itself. It is the identity the database
// path derives from, and the reason no environment variable can redirect it.
const scenarioID = "test-genie"

// databaseFile is the ledger's file name within the resolved data class.
const databaseFile = "test-genie.db"

// tuning states Test Genie's two deviations from the fleet defaults in typed
// form. The ledger is read far more often than it is written — every cost
// query, every cache lookup, every run listing — so it takes a 4 KiB page size
// and a 256 MiB memory map.
var tuning = storage.SQLiteTuning{
	PageSizeBytes: 4096,
	MMapSizeBytes: 268435456,
}

// Config captures the resolved SQLite location for runtime consumers.
type Config struct {
	Path string
	DSN  string
}

// Resolve returns the file path and DSN for Test Genie's own runtime storage.
func Resolve() (Config, error) {
	path, err := storage.SQLitePath(storage.SQLiteConfig{
		Scenario: scenarioID,
		Filename: databaseFile,
	})
	if err != nil {
		return Config{}, fmt.Errorf("resolve test-genie database path: %w", err)
	}
	return ResolveExplicit(path)
}

// ResolveExplicit resolves a SQLite path or DSN supplied directly by a caller.
//
// Passing the location as an argument is the deliberate alternative to reading
// it from the environment: an argument cannot leak into a child process.
func ResolveExplicit(raw string) (Config, error) {
	if strings.HasPrefix(raw, "file:") {
		path, err := extractPathFromDSN(raw)
		if err != nil {
			return Config{}, err
		}
		if err := ensureDir(path); err != nil {
			return Config{}, err
		}
		return Config{Path: path, DSN: raw}, nil
	}

	path, err := filepath.Abs(raw)
	if err != nil {
		return Config{}, fmt.Errorf("resolve sqlite path: %w", err)
	}
	dsn, err := storage.SQLiteDSNAt(path, tuning)
	if err != nil {
		return Config{}, fmt.Errorf("build sqlite dsn: %w", err)
	}
	return Config{Path: path, DSN: dsn}, nil
}

// BuildDSN applies Test Genie's pragmas to an explicit path.
//
// It reports an error rather than returning a best-effort string, because a
// silently malformed DSN opens a DIFFERENT database instead of failing.
func BuildDSN(path string) (string, error) {
	return storage.SQLiteDSNAt(path, tuning)
}

func extractPathFromDSN(dsn string) (string, error) {
	trimmed := strings.TrimPrefix(dsn, "file:")
	parsed, err := url.Parse("file:" + trimmed)
	if err == nil && parsed.Path != "" {
		path, decodeErr := url.PathUnescape(parsed.Path)
		if decodeErr == nil {
			if abs, absErr := filepath.Abs(path); absErr == nil {
				return abs, nil
			}
			return path, nil
		}
	}

	path := trimmed
	if idx := strings.Index(path, "?"); idx >= 0 {
		path = path[:idx]
	}
	if path == "" {
		return "", fmt.Errorf("sqlite DSN must include a file path")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve sqlite path: %w", err)
	}
	return abs, nil
}

func ensureDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("prepare sqlite directory %s: %w", dir, err)
	}
	return nil
}
