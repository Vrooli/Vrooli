package credits

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// createTestDB creates an in-memory SQLite database for testing.
func createTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	db.SetMaxOpenConns(1)

	// Create required tables
	schema := `
		CREATE TABLE credit_usage (
			id TEXT PRIMARY KEY,
			user_identity TEXT NOT NULL,
			billing_month TEXT NOT NULL,
			total_credits_used INTEGER DEFAULT 0,
			total_operations INTEGER DEFAULT 0,
			credits_by_operation TEXT DEFAULT '{}',
			operations_by_type TEXT DEFAULT '{}',
			last_operation_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_identity, billing_month)
		);

		CREATE TABLE operation_log (
			id TEXT PRIMARY KEY,
			user_identity TEXT NOT NULL,
			operation_type TEXT NOT NULL,
			credits_charged INTEGER DEFAULT 0,
			success INTEGER DEFAULT 1,
			metadata TEXT DEFAULT '{}',
			error_message TEXT,
			duration_ms INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE monetization_usage_outbox (
			operation_id TEXT PRIMARY KEY,
			user_identity TEXT NOT NULL,
			payload TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			attempts INTEGER NOT NULL DEFAULT 0,
			next_attempt_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_error TEXT,
			delivered_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`

	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	return db
}
