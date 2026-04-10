// Package domain contains the core business entities for the lifestyle dashboard.
package domain

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestInitSchema_Success verifies schema initialization works.
// [REQ:LD-EVENT-STORAGE] Schema creates events table.
// [REQ:LD-DOMAIN-REGISTER] Schema creates domains table.
func TestInitSchema_Success(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("InitSchema failed: %v", err)
	}

	// Verify events table exists
	var name string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='events'").Scan(&name)
	if err != nil {
		t.Errorf("Events table not created: %v", err)
	}

	// Verify domains table exists
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='domains'").Scan(&name)
	if err != nil {
		t.Errorf("Domains table not created: %v", err)
	}
}

// TestInitSchema_Idempotent verifies schema is idempotent.
// [REQ:LD-EVENT-STORAGE] Schema is idempotent.
func TestInitSchema_Idempotent(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Run multiple times
	for i := 0; i < 3; i++ {
		if err := InitSchema(db); err != nil {
			t.Fatalf("InitSchema failed on iteration %d: %v", i, err)
		}
	}

	// Verify tables still exist
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('events', 'domains')").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count tables: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 tables, got %d", count)
	}
}

// TestInitSchema_CreatesIndexes verifies indexes are created.
// [REQ:LD-QUERY-FILTER] Schema creates query indexes.
func TestInitSchema_CreatesIndexes(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("InitSchema failed: %v", err)
	}

	// Verify key indexes exist
	expectedIndexes := []string{
		"idx_events_domain_timestamp",
		"idx_events_timestamp",
		"idx_events_type",
		"idx_events_hypothesis",
		"idx_domains_status",
	}

	for _, indexName := range expectedIndexes {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='index' AND name=?", indexName).Scan(&name)
		if err != nil {
			t.Errorf("Index %s not created: %v", indexName, err)
		}
	}
}

// TestInitSchema_EventsTableColumns verifies events table has correct columns.
// [REQ:LD-EVENT-SCHEMA] Events table structure.
func TestInitSchema_EventsTableColumns(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("InitSchema failed: %v", err)
	}

	// Insert a test row to verify column compatibility
	_, err = db.Exec(`
		INSERT INTO events (id, timestamp, domain, event_type, payload, is_intervention, hypothesis_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, "test-id", "2026-03-10T12:00:00Z", "test", "test.event", "{}", 0, nil, "2026-03-10T12:00:00Z")
	if err != nil {
		t.Fatalf("Failed to insert test event: %v", err)
	}

	// Verify row was inserted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM events").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count events: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 event, got %d", count)
	}
}

// TestInitSchema_DomainsTableColumns verifies domains table has correct columns.
// [REQ:LD-DOMAIN-REGISTER] Domains table structure.
func TestInitSchema_DomainsTableColumns(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	if err := InitSchema(db); err != nil {
		t.Fatalf("InitSchema failed: %v", err)
	}

	// Insert a test row to verify column compatibility
	_, err = db.Exec(`
		INSERT INTO domains (name, display_name, description, capabilities, status, health_url, last_health_at, registered_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "test", "Test Domain", "Description", "[]", "active", "", nil, "2026-03-10T12:00:00Z", "2026-03-10T12:00:00Z")
	if err != nil {
		t.Fatalf("Failed to insert test domain: %v", err)
	}

	// Verify row was inserted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM domains").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count domains: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 domain, got %d", count)
	}
}
