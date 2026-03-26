// Package sqlite implements SQLite storage for domain entities.
package sqlite

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite" // SQLite driver
)

//go:embed schema.sql
var schemaSQL string

// NewDB opens a SQLite database at the storage-resolver path for DTV
// and initializes the schema.
func NewDB() (*sql.DB, error) {
	dbPath, err := resolveDBPath()
	if err != nil {
		return nil, fmt.Errorf("resolving db path: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}

	dsn := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=synchronous(NORMAL)",
		dbPath,
	)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := initSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// NewDBAt opens a SQLite database at a specific path (useful for tests).
func NewDBAt(dbPath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}

	dsn := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=synchronous(NORMAL)",
		dbPath,
	)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := initSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func resolveDBPath() (string, error) {
	r, err := storage.NewResolver(storage.ResolverConfig{})
	if err != nil {
		return "", fmt.Errorf("creating storage resolver: %w", err)
	}

	return r.Path(
		storage.Options{ScenarioID: "development-toolchain-validator"},
		storage.ClassData,
		"dtv.db",
	)
}

func initSchema(db *sql.DB) error {
	if _, err := db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("initializing schema: %w", err)
	}
	return nil
}
