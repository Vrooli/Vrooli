package testsqlite

import (
	"testing"
)

func TestOpenInitializesSchema(t *testing.T) {
	db := Open(t)

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'suite_executions'`).Scan(&count); err != nil {
		t.Fatalf("count sqlite tables: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected execution schema table to exist, got %d", count)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'suite_execution_phases'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("suite_execution_phases table count = %d, want 1", count)
	}
}
