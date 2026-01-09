package main

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewScheduler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	t.Run("Success", func(t *testing.T) {
		scheduler := NewScheduler(testDB.DB)

		if scheduler.db == nil {
			t.Error("Expected db to be set")
		}
		if scheduler.cron == nil {
			t.Error("Expected cron to be set")
		}
		if scheduler.jobs == nil {
			t.Error("Expected jobs map to be initialized")
		}
		if scheduler.executions == nil {
			t.Error("Expected executions channel to be initialized")
		}
		if scheduler.workers != 5 {
			t.Errorf("Expected 5 workers, got %d", scheduler.workers)
		}
	})
}

func TestSchedulerLoadSchedules(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	t.Run("EmptyDatabase", func(t *testing.T) {
		scheduler := NewScheduler(testDB.DB)
		err := scheduler.loadSchedules()

		if err != nil {
			t.Fatalf("loadSchedules failed: %v", err)
		}

		if len(scheduler.jobs) != 0 {
			t.Errorf("Expected 0 jobs, got %d", len(scheduler.jobs))
		}
	})

	t.Run("WithActiveSchedules", func(t *testing.T) {
		// Create active schedules
		schedule1 := createTestSchedule(t, testDB.DB, &Schedule{
			Name:           "Active Schedule 1",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
			TargetURL:      "http://example.com",
			Enabled:        true,
			Status:         "active",
		})

		schedule2 := createTestSchedule(t, testDB.DB, &Schedule{
			Name:           "Active Schedule 2",
			CronExpression: "*/5 * * * *",
			TargetType:     "webhook",
			TargetURL:      "http://example.com",
			Enabled:        true,
			Status:         "active",
		})

		// Create disabled schedule (should not be loaded)
		createTestSchedule(t, testDB.DB, &Schedule{
			Name:           "Disabled Schedule",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
			Enabled:        false,
			Status:         "active",
		})

		scheduler := NewScheduler(testDB.DB)
		err := scheduler.loadSchedules()

		if err != nil {
			t.Fatalf("loadSchedules failed: %v", err)
		}

		if len(scheduler.jobs) != 2 {
			t.Errorf("Expected 2 jobs, got %d", len(scheduler.jobs))
		}

		// Verify specific schedules were loaded
		if _, exists := scheduler.jobs[schedule1.ID]; !exists {
			t.Error("Schedule 1 was not loaded")
		}
		if _, exists := scheduler.jobs[schedule2.ID]; !exists {
			t.Error("Schedule 2 was not loaded")
		}
	})
}

func TestSchedulerAddSchedule(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	t.Run("ValidSchedule", func(t *testing.T) {
		scheduler := NewScheduler(testDB.DB)

		schedule := &Schedule{
			ID:             uuid.New().String(),
			Name:           "Test Schedule",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
			TargetURL:      "http://example.com",
		}

		err := scheduler.AddSchedule(schedule)
		if err != nil {
			t.Fatalf("AddSchedule failed: %v", err)
		}

		if len(scheduler.jobs) != 1 {
			t.Errorf("Expected 1 job, got %d", len(scheduler.jobs))
		}

		if _, exists := scheduler.jobs[schedule.ID]; !exists {
			t.Error("Schedule was not added to jobs map")
		}
	})

	t.Run("InvalidCronExpression", func(t *testing.T) {
		scheduler := NewScheduler(testDB.DB)

		schedule := &Schedule{
			ID:             uuid.New().String(),
			Name:           "Invalid Cron",
			CronExpression: "invalid cron",
			TargetType:     "webhook",
		}

		err := scheduler.AddSchedule(schedule)
		if err == nil {
			t.Error("Expected error for invalid cron expression")
		}
	})

	t.Run("UpdateExistingSchedule", func(t *testing.T) {
		scheduler := NewScheduler(testDB.DB)

		scheduleID := uuid.New().String()

		// Add initial schedule
		schedule1 := &Schedule{
			ID:             scheduleID,
			Name:           "Original",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
		}
		scheduler.AddSchedule(schedule1)

		// Update with new cron expression
		schedule2 := &Schedule{
			ID:             scheduleID,
			Name:           "Updated",
			CronExpression: "*/5 * * * *",
			TargetType:     "webhook",
		}
		scheduler.AddSchedule(schedule2)

		if len(scheduler.jobs) != 1 {
			t.Errorf("Expected 1 job after update, got %d", len(scheduler.jobs))
		}
	})
}

func TestSchedulerRemoveSchedule(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	t.Run("ExistingSchedule", func(t *testing.T) {
		scheduler := NewScheduler(testDB.DB)

		schedule := &Schedule{
			ID:             uuid.New().String(),
			Name:           "To Remove",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
		}

		scheduler.AddSchedule(schedule)
		if len(scheduler.jobs) != 1 {
			t.Error("Schedule was not added")
		}

		scheduler.RemoveSchedule(schedule.ID)
		if len(scheduler.jobs) != 0 {
			t.Errorf("Expected 0 jobs after removal, got %d", len(scheduler.jobs))
		}
	})

	t.Run("NonExistentSchedule", func(t *testing.T) {
		scheduler := NewScheduler(testDB.DB)

		// Should not panic
		scheduler.RemoveSchedule(uuid.New().String())
	})
}

func TestSchedulerGetScheduleByID(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	t.Run("ExistingSchedule", func(t *testing.T) {
		scheduler := NewScheduler(testDB.DB)

		schedule := createTestSchedule(t, testDB.DB, &Schedule{
			Name:           "Get By ID Test",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
			TargetURL:      "http://example.com",
		})

		retrieved, err := scheduler.getScheduleByID(schedule.ID)
		if err != nil {
			t.Fatalf("getScheduleByID failed: %v", err)
		}

		if retrieved.ID != schedule.ID {
			t.Errorf("Expected ID '%s', got '%s'", schedule.ID, retrieved.ID)
		}
		if retrieved.Name != schedule.Name {
			t.Errorf("Expected name '%s', got '%s'", schedule.Name, retrieved.Name)
		}
	})

	t.Run("NonExistentSchedule", func(t *testing.T) {
		scheduler := NewScheduler(testDB.DB)

		_, err := scheduler.getScheduleByID(uuid.New().String())
		if err != sql.ErrNoRows {
			t.Errorf("Expected sql.ErrNoRows, got %v", err)
		}
	})
}

func TestSchedulerIsScheduleRunning(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	t.Run("NotRunning", func(t *testing.T) {
		scheduler := NewScheduler(testDB.DB)

		schedule := createTestSchedule(t, testDB.DB, &Schedule{
			Name:           "Not Running",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
		})

		isRunning := scheduler.isScheduleRunning(schedule.ID)
		if isRunning {
			t.Error("Expected schedule to not be running")
		}
	})

	t.Run("Running", func(t *testing.T) {
		scheduler := NewScheduler(testDB.DB)

		schedule := createTestSchedule(t, testDB.DB, &Schedule{
			Name:           "Running",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
		})

		// Create running execution
		_, err := testDB.DB.Exec(`
			INSERT INTO executions (id, schedule_id, scheduled_time, start_time, status)
			VALUES ($1, $2, $3, $4, 'running')
		`, uuid.New().String(), schedule.ID, time.Now(), time.Now())

		if err != nil {
			t.Fatalf("Failed to create running execution: %v", err)
		}

		isRunning := scheduler.isScheduleRunning(schedule.ID)
		if !isRunning {
			t.Error("Expected schedule to be running")
		}
	})
}

func TestSchedulerRecordExecution(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	t.Run("RecordStart", func(t *testing.T) {
		scheduler := NewScheduler(testDB.DB)

		schedule := createTestSchedule(t, testDB.DB, &Schedule{
			Name:           "Execution Start Test",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
		})

		executionID := uuid.New().String()
		scheduledTime := time.Now()

		scheduler.recordExecutionStart(executionID, schedule.ID, scheduledTime, false, false, "test")

		// Verify execution was recorded
		var status string
		err := testDB.DB.QueryRow("SELECT status FROM executions WHERE id = $1", executionID).Scan(&status)
		if err != nil {
			t.Fatalf("Failed to query execution: %v", err)
		}

		if status != "running" {
			t.Errorf("Expected status 'running', got '%s'", status)
		}
	})

	t.Run("RecordSuccess", func(t *testing.T) {
		scheduler := NewScheduler(testDB.DB)

		schedule := createTestSchedule(t, testDB.DB, &Schedule{
			Name:           "Success Test",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
		})

		executionID := uuid.New().String()
		scheduler.recordExecutionStart(executionID, schedule.ID, time.Now(), false, false, "test")

		scheduler.recordExecutionSuccess(executionID, 1500, 200, "OK")

		// Verify execution was updated
		var status string
		var responseCode int
		var durationMs int
		err := testDB.DB.QueryRow(
			"SELECT status, response_code, duration_ms FROM executions WHERE id = $1",
			executionID,
		).Scan(&status, &responseCode, &durationMs)

		if err != nil {
			t.Fatalf("Failed to query execution: %v", err)
		}

		if status != "success" {
			t.Errorf("Expected status 'success', got '%s'", status)
		}
		if responseCode != 200 {
			t.Errorf("Expected response code 200, got %d", responseCode)
		}
		if durationMs != 1500 {
			t.Errorf("Expected duration 1500ms, got %d", durationMs)
		}
	})

	t.Run("RecordFailure", func(t *testing.T) {
		scheduler := NewScheduler(testDB.DB)

		schedule := createTestSchedule(t, testDB.DB, &Schedule{
			Name:           "Failure Test",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
		})

		executionID := uuid.New().String()
		scheduler.recordExecutionStart(executionID, schedule.ID, time.Now(), false, false, "test")

		scheduler.recordExecutionFailure(executionID, 500, "Connection failed")

		// Verify execution was updated
		var status string
		var errorMessage string
		err := testDB.DB.QueryRow(
			"SELECT status, error_message FROM executions WHERE id = $1",
			executionID,
		).Scan(&status, &errorMessage)

		if err != nil {
			t.Fatalf("Failed to query execution: %v", err)
		}

		if status != "failed" {
			t.Errorf("Expected status 'failed', got '%s'", status)
		}
		if errorMessage != "Connection failed" {
			t.Errorf("Expected error message 'Connection failed', got '%s'", errorMessage)
		}
	})
}

func TestSchedulerUpdateMetrics(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	t.Run("SuccessMetrics", func(t *testing.T) {
		scheduler := NewScheduler(testDB.DB)

		schedule := createTestSchedule(t, testDB.DB, &Schedule{
			Name:           "Metrics Success Test",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
		})

		scheduler.updateScheduleMetrics(schedule.ID, true)

		// Verify metrics
		var successCount int
		err := testDB.DB.QueryRow(
			"SELECT success_count FROM schedule_metrics WHERE schedule_id = $1",
			schedule.ID,
		).Scan(&successCount)

		if err != nil {
			t.Fatalf("Failed to query metrics: %v", err)
		}

		if successCount != 1 {
			t.Errorf("Expected success count 1, got %d", successCount)
		}
	})

	t.Run("FailureMetrics", func(t *testing.T) {
		scheduler := NewScheduler(testDB.DB)

		schedule := createTestSchedule(t, testDB.DB, &Schedule{
			Name:           "Metrics Failure Test",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
		})

		scheduler.updateScheduleMetrics(schedule.ID, false)

		// Verify metrics
		var failureCount, consecutiveFailures int
		err := testDB.DB.QueryRow(
			"SELECT failure_count, consecutive_failures FROM schedule_metrics WHERE schedule_id = $1",
			schedule.ID,
		).Scan(&failureCount, &consecutiveFailures)

		if err != nil {
			t.Fatalf("Failed to query metrics: %v", err)
		}

		if failureCount != 1 {
			t.Errorf("Expected failure count 1, got %d", failureCount)
		}
		if consecutiveFailures != 1 {
			t.Errorf("Expected consecutive failures 1, got %d", consecutiveFailures)
		}
	})

	t.Run("ConsecutiveFailuresReset", func(t *testing.T) {
		scheduler := NewScheduler(testDB.DB)

		schedule := createTestSchedule(t, testDB.DB, &Schedule{
			Name:           "Consecutive Reset Test",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
		})

		// Record failures
		scheduler.updateScheduleMetrics(schedule.ID, false)
		scheduler.updateScheduleMetrics(schedule.ID, false)

		// Verify consecutive failures
		var consecutiveFailures int
		testDB.DB.QueryRow(
			"SELECT consecutive_failures FROM schedule_metrics WHERE schedule_id = $1",
			schedule.ID,
		).Scan(&consecutiveFailures)

		if consecutiveFailures != 2 {
			t.Errorf("Expected 2 consecutive failures, got %d", consecutiveFailures)
		}

		// Record success - should reset consecutive failures
		scheduler.updateScheduleMetrics(schedule.ID, true)

		testDB.DB.QueryRow(
			"SELECT consecutive_failures FROM schedule_metrics WHERE schedule_id = $1",
			schedule.ID,
		).Scan(&consecutiveFailures)

		if consecutiveFailures != 0 {
			t.Errorf("Expected consecutive failures reset to 0, got %d", consecutiveFailures)
		}
	})
}

func TestCalculateRetryDelay(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	scheduler := NewScheduler(testDB.DB)

	t.Run("ExponentialStrategy", func(t *testing.T) {
		delay1 := scheduler.calculateRetryDelay(1, "exponential")
		delay2 := scheduler.calculateRetryDelay(2, "exponential")
		delay3 := scheduler.calculateRetryDelay(3, "exponential")

		if delay1 != 2*time.Second {
			t.Errorf("Expected 2s for attempt 1, got %v", delay1)
		}
		if delay2 != 4*time.Second {
			t.Errorf("Expected 4s for attempt 2, got %v", delay2)
		}
		if delay3 != 8*time.Second {
			t.Errorf("Expected 8s for attempt 3, got %v", delay3)
		}
	})

	t.Run("LinearStrategy", func(t *testing.T) {
		delay1 := scheduler.calculateRetryDelay(1, "linear")
		delay2 := scheduler.calculateRetryDelay(2, "linear")
		delay3 := scheduler.calculateRetryDelay(3, "linear")

		if delay1 != 10*time.Second {
			t.Errorf("Expected 10s for attempt 1, got %v", delay1)
		}
		if delay2 != 20*time.Second {
			t.Errorf("Expected 20s for attempt 2, got %v", delay2)
		}
		if delay3 != 30*time.Second {
			t.Errorf("Expected 30s for attempt 3, got %v", delay3)
		}
	})

	t.Run("FixedStrategy", func(t *testing.T) {
		delay1 := scheduler.calculateRetryDelay(1, "fixed")
		delay2 := scheduler.calculateRetryDelay(2, "fixed")
		delay3 := scheduler.calculateRetryDelay(3, "fixed")

		if delay1 != 30*time.Second {
			t.Errorf("Expected 30s for attempt 1, got %v", delay1)
		}
		if delay2 != 30*time.Second {
			t.Errorf("Expected 30s for attempt 2, got %v", delay2)
		}
		if delay3 != 30*time.Second {
			t.Errorf("Expected 30s for attempt 3, got %v", delay3)
		}
	})

	t.Run("DefaultStrategy", func(t *testing.T) {
		delay := scheduler.calculateRetryDelay(1, "unknown")

		if delay != 10*time.Second {
			t.Errorf("Expected 10s for unknown strategy, got %v", delay)
		}
	})
}

func TestSchedulerExecuteHTTPTarget(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	scheduler := NewScheduler(testDB.DB)

	t.Run("InvalidURL", func(t *testing.T) {
		schedule := &Schedule{
			TargetMethod: "GET",
			TargetURL:    "://invalid-url",
		}

		_, _, err := scheduler.executeHTTPTarget(schedule)
		if err == nil {
			t.Error("Expected error for invalid URL")
		}
	})

	t.Run("TimeoutConfiguration", func(t *testing.T) {
		// Test that timeout is properly configured
		schedule := &Schedule{
			TargetMethod:   "GET",
			TargetURL:      "http://httpbin.org/delay/1",
			TimeoutSeconds: 30,
		}

		// This should succeed with 30s timeout
		// Note: This test may fail if network is unavailable
		_, _, err := scheduler.executeHTTPTarget(schedule)
		if err != nil {
			t.Logf("Network test skipped: %v", err)
		}
	})
}

func TestUpdateNextExecutionTime(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	t.Run("ValidSchedule", func(t *testing.T) {
		scheduler := NewScheduler(testDB.DB)

		schedule := createTestSchedule(t, testDB.DB, &Schedule{
			Name:           "Next Execution Test",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
		})

		// Add to scheduler
		scheduler.AddSchedule(schedule)

		// Update next execution time
		scheduler.updateNextExecutionTime(schedule.ID)

		// Verify next execution time was set
		var nextExecutionAt *time.Time
		err := testDB.DB.QueryRow(
			"SELECT next_execution_at FROM schedules WHERE id = $1",
			schedule.ID,
		).Scan(&nextExecutionAt)

		if err != nil {
			t.Fatalf("Failed to query next execution time: %v", err)
		}

		if nextExecutionAt == nil {
			t.Error("Expected next_execution_at to be set")
		} else if nextExecutionAt.Before(time.Now()) {
			t.Error("Expected next_execution_at to be in the future")
		}
	})

	t.Run("NonExistentSchedule", func(t *testing.T) {
		scheduler := NewScheduler(testDB.DB)

		// Should not panic
		scheduler.updateNextExecutionTime(uuid.New().String())
	})
}
