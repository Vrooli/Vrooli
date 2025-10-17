package main

import (
	"database/sql"
	"os"
	"testing"
)

func TestInitializeDatabase(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		testDB := setupTestDatabase(t)
		defer testDB.Cleanup()

		// Initialize should succeed
		err := InitializeDatabase(testDB.DB)
		if err != nil {
			t.Fatalf("InitializeDatabase failed: %v", err)
		}

		// Verify tables exist
		tables := []string{
			"schedules",
			"executions",
			"cron_presets",
		}

		for _, table := range tables {
			var exists bool
			err := testDB.DB.QueryRow(`
				SELECT EXISTS (
					SELECT FROM information_schema.tables
					WHERE table_schema = 'public'
					AND table_name = $1
				)
			`, table).Scan(&exists)

			if err != nil {
				t.Fatalf("Failed to check if table %s exists: %v", table, err)
			}

			if !exists {
				t.Errorf("Expected table %s to exist", table)
			}
		}
	})

	t.Run("IdempotentInitialization", func(t *testing.T) {
		testDB := setupTestDatabase(t)
		defer testDB.Cleanup()

		// First initialization
		err := InitializeDatabase(testDB.DB)
		if err != nil {
			t.Fatalf("First InitializeDatabase failed: %v", err)
		}

		// Second initialization should not fail
		err = InitializeDatabase(testDB.DB)
		if err != nil {
			t.Fatalf("Second InitializeDatabase failed: %v", err)
		}
	})

	t.Run("VerifySchedulesTableStructure", func(t *testing.T) {
		testDB := setupTestDatabase(t)
		defer testDB.Cleanup()

		err := InitializeDatabase(testDB.DB)
		if err != nil {
			t.Fatalf("InitializeDatabase failed: %v", err)
		}

		// Verify required columns exist
		requiredColumns := []string{
			"id",
			"name",
			"cron_expression",
			"timezone",
			"target_type",
			"target_url",
			"enabled",
			"status",
			"created_at",
			"updated_at",
		}

		for _, column := range requiredColumns {
			var exists bool
			err := testDB.DB.QueryRow(`
				SELECT EXISTS (
					SELECT FROM information_schema.columns
					WHERE table_name = 'schedules'
					AND column_name = $1
				)
			`, column).Scan(&exists)

			if err != nil {
				t.Fatalf("Failed to check if column %s exists: %v", column, err)
			}

			if !exists {
				t.Errorf("Expected column %s to exist in schedules table", column)
			}
		}
	})

	t.Run("VerifyExecutionsTableStructure", func(t *testing.T) {
		testDB := setupTestDatabase(t)
		defer testDB.Cleanup()

		err := InitializeDatabase(testDB.DB)
		if err != nil {
			t.Fatalf("InitializeDatabase failed: %v", err)
		}

		// Verify required columns exist
		requiredColumns := []string{
			"id",
			"schedule_id",
			"scheduled_time",
			"start_time",
			"end_time",
			"status",
			"duration_ms",
			"response_code",
			"error_message",
		}

		for _, column := range requiredColumns {
			var exists bool
			err := testDB.DB.QueryRow(`
				SELECT EXISTS (
					SELECT FROM information_schema.columns
					WHERE table_name = 'executions'
					AND column_name = $1
				)
			`, column).Scan(&exists)

			if err != nil {
				t.Fatalf("Failed to check if column %s exists: %v", column, err)
			}

			if !exists {
				t.Errorf("Expected column %s to exist in executions table", column)
			}
		}
	})

	t.Run("VerifyIndexes", func(t *testing.T) {
		testDB := setupTestDatabase(t)
		defer testDB.Cleanup()

		err := InitializeDatabase(testDB.DB)
		if err != nil {
			t.Fatalf("InitializeDatabase failed: %v", err)
		}

		// Verify key indexes exist
		expectedIndexes := []string{
			"idx_schedules_enabled",
			"idx_schedules_next_execution",
			"idx_executions_schedule_id",
			"idx_executions_status",
		}

		for _, indexName := range expectedIndexes {
			var exists bool
			err := testDB.DB.QueryRow(`
				SELECT EXISTS (
					SELECT FROM pg_indexes
					WHERE indexname = $1
				)
			`, indexName).Scan(&exists)

			if err != nil {
				t.Fatalf("Failed to check if index %s exists: %v", indexName, err)
			}

			if !exists {
				t.Errorf("Expected index %s to exist", indexName)
			}
		}
	})

	t.Run("VerifyDefaultCronPresets", func(t *testing.T) {
		testDB := setupTestDatabase(t)
		defer testDB.Cleanup()

		err := InitializeDatabase(testDB.DB)
		if err != nil {
			t.Fatalf("InitializeDatabase failed: %v", err)
		}

		// Verify some default presets exist
		var count int
		err = testDB.DB.QueryRow("SELECT COUNT(*) FROM cron_presets WHERE is_system = true").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count cron presets: %v", err)
		}

		if count == 0 {
			t.Error("Expected at least some default cron presets")
		}

		// Verify specific preset exists
		var exists bool
		err = testDB.DB.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM cron_presets
				WHERE name = 'Every minute'
				AND expression = '* * * * *'
			)
		`).Scan(&exists)

		if err != nil {
			t.Fatalf("Failed to check for 'Every minute' preset: %v", err)
		}

		if !exists {
			t.Error("Expected 'Every minute' preset to exist")
		}
	})

	t.Run("VerifyForeignKeyConstraints", func(t *testing.T) {
		testDB := setupTestDatabase(t)
		defer testDB.Cleanup()

		err := InitializeDatabase(testDB.DB)
		if err != nil {
			t.Fatalf("InitializeDatabase failed: %v", err)
		}

		// Verify foreign key constraint on executions
		var exists bool
		err = testDB.DB.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM information_schema.table_constraints
				WHERE constraint_type = 'FOREIGN KEY'
				AND table_name = 'executions'
			)
		`).Scan(&exists)

		if err != nil {
			t.Fatalf("Failed to check foreign key constraints: %v", err)
		}

		if !exists {
			t.Error("Expected foreign key constraint on executions table")
		}
	})

	t.Run("VerifyCascadeDelete", func(t *testing.T) {
		testDB := setupTestDatabase(t)
		defer testDB.Cleanup()

		err := InitializeDatabase(testDB.DB)
		if err != nil {
			t.Fatalf("InitializeDatabase failed: %v", err)
		}

		// Create a schedule
		schedule := createTestSchedule(t, testDB.DB, &Schedule{
			Name:           "Cascade Test",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
		})

		// Create executions
		createTestExecution(t, testDB.DB, schedule.ID, "success")
		createTestExecution(t, testDB.DB, schedule.ID, "failed")

		// Verify executions exist
		var executionCount int
		testDB.DB.QueryRow("SELECT COUNT(*) FROM executions WHERE schedule_id = $1", schedule.ID).Scan(&executionCount)
		if executionCount != 2 {
			t.Errorf("Expected 2 executions, got %d", executionCount)
		}

		// Delete schedule
		_, err = testDB.DB.Exec("DELETE FROM schedules WHERE id = $1", schedule.ID)
		if err != nil {
			t.Fatalf("Failed to delete schedule: %v", err)
		}

		// Verify executions were cascade deleted
		testDB.DB.QueryRow("SELECT COUNT(*) FROM executions WHERE schedule_id = $1", schedule.ID).Scan(&executionCount)
		if executionCount != 0 {
			t.Errorf("Expected executions to be cascade deleted, but found %d", executionCount)
		}
	})
}

func TestCreateMinimalSchema(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		// Create a fresh database connection
		dbURL := getTestDatabaseURL(t)
		db, err := sql.Open("postgres", dbURL)
		if err != nil {
			t.Fatalf("Failed to connect to test database: %v", err)
		}
		defer db.Close()

		// Drop tables if they exist
		db.Exec("DROP TABLE IF EXISTS executions CASCADE")
		db.Exec("DROP TABLE IF EXISTS schedules CASCADE")
		db.Exec("DROP TABLE IF EXISTS cron_presets CASCADE")
		db.Exec("DROP TABLE IF EXISTS audit_log CASCADE")
		db.Exec("DROP TABLE IF EXISTS schedule_metrics CASCADE")

		// Create minimal schema
		err = createMinimalSchema(db)
		if err != nil {
			t.Fatalf("createMinimalSchema failed: %v", err)
		}

		// Verify tables were created
		var exists bool
		err = db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables
				WHERE table_schema = 'public'
				AND table_name = 'schedules'
			)
		`).Scan(&exists)

		if err != nil {
			t.Fatalf("Failed to check if schedules table exists: %v", err)
		}

		if !exists {
			t.Error("Expected schedules table to be created")
		}
	})
}

func TestDatabaseSchemaValidation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	t.Run("InsertScheduleWithAllFields", func(t *testing.T) {
		// Test that we can insert a schedule with all fields
		schedule := createTestSchedule(t, testDB.DB, &Schedule{
			Name:            "Full Fields Test",
			Description:     "Test with all fields",
			CronExpression:  "0 * * * *",
			Timezone:        "America/New_York",
			TargetType:      "webhook",
			TargetURL:       "http://example.com",
			TargetMethod:    "POST",
			Status:          "active",
			Enabled:         true,
			OverlapPolicy:   "skip",
			MaxRetries:      5,
			RetryStrategy:   "exponential",
			TimeoutSeconds:  60,
			CatchUpMissed:   true,
			Priority:        10,
			Owner:           "test-user",
			Team:            "test-team",
		})

		// Verify it was inserted
		var name string
		err := testDB.DB.QueryRow("SELECT name FROM schedules WHERE id = $1", schedule.ID).Scan(&name)
		if err != nil {
			t.Fatalf("Failed to query inserted schedule: %v", err)
		}

		if name != "Full Fields Test" {
			t.Errorf("Expected name 'Full Fields Test', got '%s'", name)
		}
	})

	t.Run("InsertExecutionWithAllFields", func(t *testing.T) {
		schedule := createTestSchedule(t, testDB.DB, &Schedule{
			Name:           "Execution Fields Test",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
		})

		// Insert execution with all fields
		executionID := createTestExecution(t, testDB.DB, schedule.ID, "success")

		// Verify it was inserted
		var status string
		err := testDB.DB.QueryRow("SELECT status FROM executions WHERE id = $1", executionID).Scan(&status)
		if err != nil {
			t.Fatalf("Failed to query inserted execution: %v", err)
		}

		if status != "success" {
			t.Errorf("Expected status 'success', got '%s'", status)
		}
	})
}

// Helper function to get test database URL
func getTestDatabaseURL(t *testing.T) string {
	t.Helper()

	dbURL := os.Getenv("TEST_POSTGRES_URL")
	if dbURL == "" {
		t.Skip("TEST_POSTGRES_URL not set, skipping database tests")
	}

	return dbURL
}
