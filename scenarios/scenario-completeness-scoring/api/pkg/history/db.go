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

	if _, err := db.conn.Exec(schema); err != nil {
		return err
	}

	// Migration v2: Add source and tags columns for ecosystem-manager correlation
	// These enable filtering history by source system and arbitrary tags
	migrations := []string{
		// Add source column (e.g., "ecosystem-manager", "cli", "ci")
		`ALTER TABLE score_snapshots ADD COLUMN source TEXT`,
		// Add tags column (JSON array of strings, e.g., ["task:abc123", "iteration:5"])
		`ALTER TABLE score_snapshots ADD COLUMN tags JSON`,
		// Index for filtering by source
		`CREATE INDEX IF NOT EXISTS idx_snapshots_source ON score_snapshots(source)`,
		// Compound index for source + scenario queries
		`CREATE INDEX IF NOT EXISTS idx_snapshots_scenario_source ON score_snapshots(scenario, source)`,
	}

	for _, migration := range migrations {
		// SQLite returns error if column already exists, which is fine
		_, err := db.conn.Exec(migration)
		if err != nil && !isColumnExistsError(err) && !isIndexExistsError(err) {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

// isColumnExistsError checks if error is due to column already existing
func isColumnExistsError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "duplicate column name") || contains(errStr, "already exists")
}

// isIndexExistsError checks if error is due to index already existing
func isIndexExistsError(err error) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), "index") && contains(err.Error(), "already exists")
}

// contains checks if s contains substr (simple helper to avoid strings import)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
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
