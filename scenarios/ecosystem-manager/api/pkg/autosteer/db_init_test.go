package autosteer

import (
	"testing"
)

// TestGetTableCounts verifies the startup diagnostic over the SQLite controller
// schema applied by SetupTestDatabase (via database.EnsureSchemas).
func TestGetTableCounts(t *testing.T) {
	pg, cleanup := SetupTestDatabase(t)
	defer cleanup()

	counts, err := GetTableCounts(pg.db)
	if err != nil {
		t.Fatalf("GetTableCounts() error = %v", err)
	}

	expectedTables := []string{
		"profile_executions",
		"profile_execution_state",
		"decision_trace",
	}

	for _, table := range expectedTables {
		count, exists := counts[table]
		if !exists {
			t.Errorf("Table %s not found in counts", table)
		}
		if count != 0 {
			t.Errorf("freshly created table %s should have 0 rows, got %d", table, count)
		}
	}
}
