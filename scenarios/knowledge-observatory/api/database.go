package main

// DOC: docs/concepts/ARCHITECTURE.md#api-runtime

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"

	_ "modernc.org/sqlite" // pure-Go, CGO-free SQLite driver
)

// scenarioSlug names this scenario. It is the fallback passed to
// storage.ScenarioNamespace, which prefers the variant-scoped namespace the
// lifecycle injects — that is what keeps a shadow instance off live's files.
const scenarioSlug = "knowledge-observatory"

// databaseFileName is the SQLite database inside the resolved data directory.
const databaseFileName = "knowledge-observatory.db"

// storageLayout is everything the process needs from the storage resolver: the
// per-class roots for the routed file seam, and the database path.
type storageLayout struct {
	Paths        storage.Paths
	DatabasePath string
}

// resolveStorageLayout asks api-core/storage where this scenario's files live.
//
// The scenario ID is the variant-aware namespace, never the bare slug, so a
// Baseline-Modes shadow addresses a different directory than live.
func resolveStorageLayout() (storageLayout, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return storageLayout{}, fmt.Errorf("create storage resolver: %w", err)
	}

	scenarioID, err := storage.ScenarioNamespace(scenarioSlug)
	if err != nil {
		return storageLayout{}, fmt.Errorf("resolve storage namespace: %w", err)
	}
	opts := storage.Options{ScenarioID: scenarioID}

	paths, err := resolver.Resolve(opts)
	if err != nil {
		return storageLayout{}, fmt.Errorf("resolve storage roots: %w", err)
	}

	dbPath, err := resolver.Path(opts, storage.ClassData, databaseFileName)
	if err != nil {
		return storageLayout{}, fmt.Errorf("resolve database path: %w", err)
	}

	return storageLayout{Paths: paths, DatabasePath: dbPath}, nil
}

// openScenarioDatabase opens the SQLite database at layout.DatabasePath.
//
// MaxOpenConns is 1 on purpose: SQLite admits a single writer, and a larger
// pool converts contention into SQLITE_BUSY errors rather than throughput.
// The pragmas are the standard durable-but-fast configuration — WAL for
// concurrent readers, a busy timeout so a contended write waits instead of
// failing, and NORMAL synchronous, which is safe under WAL.
func openScenarioDatabase(ctx context.Context, layout storageLayout) (*database.RoutedDB, error) {
	if err := os.MkdirAll(filepath.Dir(layout.DatabasePath), 0o755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	db, err := database.Open(ctx, database.Config{
		Driver:       database.DriverSQLite,
		DSN:          layout.DatabasePath,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	primary := db.Primary()
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
	} {
		if _, err := primary.ExecContext(ctx, pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("apply %s: %w", pragma, err)
		}
	}

	log.Printf("SQLite database opened at %s", layout.DatabasePath)
	return db, nil
}
