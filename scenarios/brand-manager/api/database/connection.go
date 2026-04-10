// Package database handles SQLite connection setup and schema initialization.
// [REQ:BM-REQ-STORE-INIT]
// DOC: docs/concepts/ARCHITECTURE.md#storage-architecture
// DOC: docs/reference/configuration.md#sqlite-pragmas
// DOC: docs/internal/STORAGE_AUDIT.md
package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"brand-manager/config"

	"github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaFS embed.FS

// Connect opens a SQLite database using the provided Config and initializes the schema.
// It creates the parent directory if needed.
func Connect(cfg config.Config) (*sql.DB, error) {
	log.Printf("Opening SQLite database at %s", cfg.SQLitePath)

	// Ensure parent directory exists
	dir := filepath.Dir(cfg.SQLitePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create db directory %s: %w", dir, err)
	}

	// Use api-core/database.Connect with DSN from config.
	// modernc.org/sqlite registers as "sqlite" (not "sqlite3" which is the cgo driver).
	db, err := database.Connect(context.Background(), database.Config{
		Driver:       "sqlite",
		DSN:          cfg.DSN(),
		MaxOpenConns: cfg.MaxOpenConns,
		Logger:       log.Printf,
	})
	if err != nil {
		return nil, fmt.Errorf("connect sqlite: %w", err)
	}

	// Initialize schema
	if err := initSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return db, nil
}

// initSchema executes the embedded schema.sql to create tables idempotently.
func initSchema(db *sql.DB) error {
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("read embedded schema: %w", err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		return fmt.Errorf("execute schema: %w", err)
	}
	log.Println("Schema initialized successfully")
	return nil
}
