// +build testing

package main

import (
	"testing"
)

func TestSchemaExists(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Check if users table exists and has correct columns
	var count int
	err := env.DB.QueryRow("SELECT COUNT(*) FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'scenario_id'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query schema: %v", err)
	}

	if count == 0 {
		t.Error("users.scenario_id column does not exist")
	}

	t.Logf("Schema check passed: users.scenario_id exists (count: %d)", count)
}
