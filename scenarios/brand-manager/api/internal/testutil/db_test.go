package testutil

import (
	"testing"
)

// [REQ:BM-REQ-STORE-INIT] Verify test database setup helper works correctly.

func TestSetupTestDB(t *testing.T) {
	db := SetupTestDB(t)

	// Verify the database is usable
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM brands").Scan(&count)
	if err != nil {
		t.Fatalf("query brands table: %v", err)
	}
	if count != 0 {
		t.Errorf("expected empty brands table, got %d rows", count)
	}
}

func TestSetupTestDB_SchemaPresent(t *testing.T) {
	db := SetupTestDB(t)

	// Verify all expected tables exist
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	defer rows.Close()

	tables := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		tables[name] = true
	}

	for _, expected := range []string{"brands", "brand_versions", "assignments", "assets"} {
		if !tables[expected] {
			t.Errorf("missing table: %s", expected)
		}
	}
}
