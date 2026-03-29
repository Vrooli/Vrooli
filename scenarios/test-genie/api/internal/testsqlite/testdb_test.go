package testsqlite

import (
	"testing"
)

func TestOpenInitializesSchema(t *testing.T) {
	db := Open(t)

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name IN ('suite_requests', 'suite_executions')`).Scan(&count); err != nil {
		t.Fatalf("count sqlite tables: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected both schema tables to exist, got %d", count)
	}
}

func TestOpenWithSeedLoadsPreviewRow(t *testing.T) {
	db := OpenWithSeed(t)

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM suite_requests WHERE id = ?`, "00000000-0000-0000-0000-000000000001").Scan(&count); err != nil {
		t.Fatalf("count seeded suite request: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected seeded suite request, got %d", count)
	}
}
