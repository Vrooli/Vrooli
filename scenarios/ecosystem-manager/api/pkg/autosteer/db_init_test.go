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
			"auto_steer_profiles",
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

	t.Run("counts increase after adding profile", func(t *testing.T) {
		// Get initial counts
		initialCounts, err := GetTableCounts(pg.db)
		if err != nil {
			t.Fatalf("GetTableCounts() error = %v", err)
		}

		// Create a test profile
		profileService := NewProfileService(pg.db)
		profile := CreateTestProfile(t, "Count Test", ModeProgress, 10)
		if err := profileService.CreateProfile(profile); err != nil {
			t.Fatalf("Failed to create profile: %v", err)
		}

		// Get new counts
		newCounts, err := GetTableCounts(pg.db)
		if err != nil {
			t.Fatalf("GetTableCounts() after insert error = %v", err)
		}

		// Verify profile count increased
		if newCounts["auto_steer_profiles"] != initialCounts["auto_steer_profiles"]+1 {
			t.Errorf("Expected profile count to increase by 1, got %d -> %d",
				initialCounts["auto_steer_profiles"],
				newCounts["auto_steer_profiles"])
		}

		t.Logf("Profile count correctly increased from %d to %d",
			initialCounts["auto_steer_profiles"],
			newCounts["auto_steer_profiles"])
	})
}
