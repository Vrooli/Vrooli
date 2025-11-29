// Package history provides score history storage and trend analysis
// [REQ:SCS-HIST-001] Score history storage
// [REQ:SCS-HIST-002] SQLite database
package history

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps SQLite database connection for score history
type DB struct {
	conn   *sql.DB
	dbPath string
	mu     sync.RWMutex
}

// NewDB creates or opens the SQLite database for score history
// [REQ:SCS-HIST-002] SQLite database initialization
func NewDB(dataDir string) (*DB, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "scores.db")
	conn, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	conn.SetMaxOpenConns(1) // SQLite doesn't support multiple writers
	conn.SetMaxIdleConns(1)

	db := &DB{
		conn:   conn,
		dbPath: dbPath,
	}

	// Run migrations
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// migrate creates the database schema
func (db *DB) migrate() error {
	schema := `
	-- Score snapshots table
	-- [REQ:SCS-HIST-001] Store score snapshots over time
	CREATE TABLE IF NOT EXISTS score_snapshots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		scenario TEXT NOT NULL,
		score INTEGER NOT NULL,
		classification TEXT NOT NULL,
		breakdown JSON NOT NULL,
		config_snapshot JSON,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Index for efficient queries by scenario
	CREATE INDEX IF NOT EXISTS idx_snapshots_scenario ON score_snapshots(scenario);

	-- Index for efficient time-based queries
	CREATE INDEX IF NOT EXISTS idx_snapshots_created_at ON score_snapshots(created_at);

	-- Compound index for scenario + time queries
	CREATE INDEX IF NOT EXISTS idx_snapshots_scenario_time ON score_snapshots(scenario, created_at DESC);
	`

	_, err := db.conn.Exec(schema)
	return err
}

// Close closes the database connection
func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// Path returns the database file path
func (db *DB) Path() string {
	return db.dbPath
}

// Ping verifies database connectivity
func (db *DB) Ping() error {
	return db.conn.Ping()
}
