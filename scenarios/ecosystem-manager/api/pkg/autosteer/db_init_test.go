package autosteer

import (
	"testing"
)

func TestEnsureTablesExist(t *testing.T) {
	pg, cleanup := SetupTestDatabase(t)
	if pg == nil {
		return
	}
	defer cleanup()

	t.Run("tables exist after schema creation", func(t *testing.T) {
		err := EnsureTablesExist(pg.db)
		if err != nil {
			t.Errorf("EnsureTablesExist() failed: %v", err)
		}
	})

	t.Run("get table counts", func(t *testing.T) {
		counts, err := GetTableCounts(pg.db)
		if err != nil {
			t.Fatalf("GetTableCounts() error = %v", err)
		}

		expectedTables := []string{
			"profile_executions",
			"profile_execution_state",
		}

		for _, table := range expectedTables {
			count, exists := counts[table]
			if !exists {
				t.Errorf("Table %s not found in counts", table)
			}

			// Initially should be 0 records
			if count < 0 {
				t.Errorf("Table %s has negative count: %d", table, count)
			}

			t.Logf("Table %s has %d records", table, count)
		}
	})
}
