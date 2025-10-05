package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHealthCheck(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)
	defer cleanupTestData(t, app.DB)

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var response map[string]interface{}
		assertJSONResponse(t, rr, http.StatusOK, &response)

		if response["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got '%v'", response["status"])
		}
		if response["service"] != "workflow-scheduler" {
			t.Errorf("Expected service 'workflow-scheduler', got '%v'", response["service"])
		}
	})
}

func TestDatabaseStatus(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)
	defer cleanupTestData(t, app.DB)

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/system/db-status",
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var response map[string]interface{}
		assertJSONResponse(t, rr, http.StatusOK, &response)

		if response["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got '%v'", response["status"])
		}
	})
}

func TestGetSchedules(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)
	defer cleanupTestData(t, app.DB)

	t.Run("EmptyList", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/schedules",
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var schedules []Schedule
		assertJSONResponse(t, rr, http.StatusOK, &schedules)

		if len(schedules) != 0 {
			t.Errorf("Expected empty list, got %d schedules", len(schedules))
		}
	})

	t.Run("WithSchedules", func(t *testing.T) {
		// Create test schedules
		schedule1 := createTestSchedule(t, app.DB, &Schedule{
			Name:           "Test Schedule 1",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
			TargetURL:      "http://example.com",
			Enabled:        true,
		})

		schedule2 := createTestSchedule(t, app.DB, &Schedule{
			Name:           "Test Schedule 2",
			CronExpression: "*/5 * * * *",
			TargetType:     "webhook",
			TargetURL:      "http://example.com",
			Enabled:        false,
		})

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/schedules",
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var schedules []Schedule
		assertJSONResponse(t, rr, http.StatusOK, &schedules)

		if len(schedules) != 2 {
			t.Errorf("Expected 2 schedules, got %d", len(schedules))
		}

		// Verify schedule data
		foundSchedule1 := false
		foundSchedule2 := false
		for _, s := range schedules {
			if s.ID == schedule1.ID {
				foundSchedule1 = true
				if s.Name != "Test Schedule 1" {
					t.Errorf("Schedule 1 name mismatch")
				}
			}
			if s.ID == schedule2.ID {
				foundSchedule2 = true
				if s.Name != "Test Schedule 2" {
					t.Errorf("Schedule 2 name mismatch")
				}
			}
		}

		if !foundSchedule1 || !foundSchedule2 {
			t.Error("Not all schedules were returned")
		}
	})
}

func TestCreateSchedule(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)
	defer cleanupTestData(t, app.DB)

	t.Run("Success", func(t *testing.T) {
		scheduleData := map[string]interface{}{
			"name":            "New Schedule",
			"description":     "Test schedule",
			"cron_expression": "0 9 * * *",
			"timezone":        "UTC",
			"target_type":     "webhook",
			"target_url":      "http://example.com/webhook",
			"target_method":   "POST",
			"enabled":         true,
			"max_retries":     3,
			"retry_strategy":  "exponential",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/schedules",
			Body:   scheduleData,
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var schedule Schedule
		assertJSONResponse(t, rr, http.StatusCreated, &schedule)

		if schedule.ID == "" {
			t.Error("Expected schedule ID to be generated")
		}
		if schedule.Name != "New Schedule" {
			t.Errorf("Expected name 'New Schedule', got '%s'", schedule.Name)
		}
		if schedule.CronExpression != "0 9 * * *" {
			t.Errorf("Expected cron '0 9 * * *', got '%s'", schedule.CronExpression)
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		RunErrorScenarios(t, app, CreateScheduleErrors())
	})

	t.Run("InvalidCronExpression", func(t *testing.T) {
		scheduleData := map[string]interface{}{
			"name":            "Invalid Cron",
			"cron_expression": "invalid cron",
			"target_type":     "webhook",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/schedules",
			Body:   scheduleData,
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, rr, http.StatusBadRequest, "Invalid cron expression")
	})

	t.Run("DefaultValues", func(t *testing.T) {
		scheduleData := map[string]interface{}{
			"name":            "Minimal Schedule",
			"cron_expression": "0 * * * *",
			"target_type":     "webhook",
			"target_url":      "http://example.com",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/schedules",
			Body:   scheduleData,
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var schedule Schedule
		assertJSONResponse(t, rr, http.StatusCreated, &schedule)

		if schedule.Timezone != "UTC" {
			t.Errorf("Expected default timezone 'UTC', got '%s'", schedule.Timezone)
		}
		if schedule.TargetMethod != "POST" {
			t.Errorf("Expected default method 'POST', got '%s'", schedule.TargetMethod)
		}
		if schedule.Status != "active" {
			t.Errorf("Expected default status 'active', got '%s'", schedule.Status)
		}
		if schedule.Priority != 5 {
			t.Errorf("Expected default priority 5, got %d", schedule.Priority)
		}
	})
}

func TestGetSchedule(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)
	defer cleanupTestData(t, app.DB)

	t.Run("Success", func(t *testing.T) {
		// Create test schedule
		schedule := createTestSchedule(t, app.DB, &Schedule{
			Name:           "Get Test Schedule",
			CronExpression: "0 12 * * *",
			TargetType:     "webhook",
			TargetURL:      "http://example.com",
		})

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/schedules/" + schedule.ID,
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var retrieved Schedule
		assertJSONResponse(t, rr, http.StatusOK, &retrieved)

		if retrieved.ID != schedule.ID {
			t.Errorf("Expected ID '%s', got '%s'", schedule.ID, retrieved.ID)
		}
		if retrieved.Name != schedule.Name {
			t.Errorf("Expected name '%s', got '%s'", schedule.Name, retrieved.Name)
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		RunErrorScenarios(t, app, GetScheduleErrors())
	})
}

func TestUpdateSchedule(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)
	defer cleanupTestData(t, app.DB)

	t.Run("Success", func(t *testing.T) {
		// Create test schedule
		schedule := createTestSchedule(t, app.DB, &Schedule{
			Name:           "Original Name",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
			TargetURL:      "http://example.com",
		})

		updateData := map[string]interface{}{
			"name":            "Updated Name",
			"cron_expression": "*/15 * * * *",
			"target_type":     "webhook",
			"target_url":      "http://updated.com",
		}

		req := HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/schedules/" + schedule.ID,
			Body:   updateData,
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var updated Schedule
		assertJSONResponse(t, rr, http.StatusOK, &updated)

		if updated.Name != "Updated Name" {
			t.Errorf("Expected updated name 'Updated Name', got '%s'", updated.Name)
		}
		if updated.CronExpression != "*/15 * * * *" {
			t.Errorf("Expected updated cron '*/15 * * * *', got '%s'", updated.CronExpression)
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		RunErrorScenarios(t, app, UpdateScheduleErrors())
	})
}

func TestDeleteSchedule(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)
	defer cleanupTestData(t, app.DB)

	t.Run("Success", func(t *testing.T) {
		// Create test schedule
		schedule := createTestSchedule(t, app.DB, &Schedule{
			Name:           "To Delete",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
		})

		req := HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/schedules/" + schedule.ID,
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var response map[string]string
		assertJSONResponse(t, rr, http.StatusOK, &response)

		if response["result"] != "success" {
			t.Errorf("Expected result 'success', got '%s'", response["result"])
		}

		// Verify schedule is deleted
		var count int
		app.DB.QueryRow("SELECT COUNT(*) FROM schedules WHERE id = $1", schedule.ID).Scan(&count)
		if count != 0 {
			t.Error("Schedule was not deleted")
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		RunErrorScenarios(t, app, DeleteScheduleErrors())
	})
}

func TestEnableDisableSchedule(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)
	defer cleanupTestData(t, app.DB)

	t.Run("EnableSchedule", func(t *testing.T) {
		schedule := createTestSchedule(t, app.DB, &Schedule{
			Name:           "Test Enable",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
			Enabled:        false,
		})

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/schedules/" + schedule.ID + "/enable",
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var response map[string]interface{}
		assertJSONResponse(t, rr, http.StatusOK, &response)

		if response["enabled"] != true {
			t.Error("Expected enabled to be true")
		}

		// Verify in database
		var enabled bool
		app.DB.QueryRow("SELECT enabled FROM schedules WHERE id = $1", schedule.ID).Scan(&enabled)
		if !enabled {
			t.Error("Schedule was not enabled in database")
		}
	})

	t.Run("DisableSchedule", func(t *testing.T) {
		schedule := createTestSchedule(t, app.DB, &Schedule{
			Name:           "Test Disable",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
			Enabled:        true,
		})

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/schedules/" + schedule.ID + "/disable",
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var response map[string]interface{}
		assertJSONResponse(t, rr, http.StatusOK, &response)

		if response["enabled"] != false {
			t.Error("Expected enabled to be false")
		}

		// Verify in database
		var enabled bool
		app.DB.QueryRow("SELECT enabled FROM schedules WHERE id = $1", schedule.ID).Scan(&enabled)
		if enabled {
			t.Error("Schedule was not disabled in database")
		}
	})
}

func TestTriggerSchedule(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)
	defer cleanupTestData(t, app.DB)

	t.Run("Success", func(t *testing.T) {
		schedule := createTestSchedule(t, app.DB, &Schedule{
			Name:           "Test Trigger",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
			TargetURL:      "http://example.com",
		})

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/schedules/" + schedule.ID + "/trigger",
			Headers: map[string]string{
				"X-User": "test-user",
			},
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var response map[string]string
		assertJSONResponse(t, rr, http.StatusOK, &response)

		if response["result"] != "success" {
			t.Errorf("Expected result 'success', got '%s'", response["result"])
		}
		if response["execution_id"] == "" {
			t.Error("Expected execution_id to be set")
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		RunErrorScenarios(t, app, TriggerScheduleErrors())
	})
}

func TestGetExecutions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)
	defer cleanupTestData(t, app.DB)

	t.Run("EmptyList", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/executions",
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var executions []map[string]interface{}
		assertJSONResponse(t, rr, http.StatusOK, &executions)

		if len(executions) != 0 {
			t.Errorf("Expected empty list, got %d executions", len(executions))
		}
	})

	t.Run("WithExecutions", func(t *testing.T) {
		// Create test schedule and executions
		schedule := createTestSchedule(t, app.DB, &Schedule{
			Name:           "Execution Test",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
		})

		createTestExecution(t, app.DB, schedule.ID, "success")
		createTestExecution(t, app.DB, schedule.ID, "failed")

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/executions",
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var executions []map[string]interface{}
		assertJSONResponse(t, rr, http.StatusOK, &executions)

		if len(executions) != 2 {
			t.Errorf("Expected 2 executions, got %d", len(executions))
		}
	})
}

func TestGetScheduleExecutions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)
	defer cleanupTestData(t, app.DB)

	t.Run("Success", func(t *testing.T) {
		schedule := createTestSchedule(t, app.DB, &Schedule{
			Name:           "Schedule Executions Test",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
		})

		createTestExecution(t, app.DB, schedule.ID, "success")
		createTestExecution(t, app.DB, schedule.ID, "success")

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/schedules/" + schedule.ID + "/executions",
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var executions []Execution
		assertJSONResponse(t, rr, http.StatusOK, &executions)

		if len(executions) != 2 {
			t.Errorf("Expected 2 executions, got %d", len(executions))
		}

		for _, exec := range executions {
			if exec.ScheduleID != schedule.ID {
				t.Errorf("Expected schedule_id '%s', got '%s'", schedule.ID, exec.ScheduleID)
			}
		}
	})
}

func TestGetExecution(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)
	defer cleanupTestData(t, app.DB)

	t.Run("Success", func(t *testing.T) {
		schedule := createTestSchedule(t, app.DB, &Schedule{
			Name:           "Get Execution Test",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
		})

		executionID := createTestExecution(t, app.DB, schedule.ID, "success")

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/executions/" + executionID,
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var execution Execution
		assertJSONResponse(t, rr, http.StatusOK, &execution)

		if execution.ID != executionID {
			t.Errorf("Expected ID '%s', got '%s'", executionID, execution.ID)
		}
		if execution.Status != "success" {
			t.Errorf("Expected status 'success', got '%s'", execution.Status)
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		RunErrorScenarios(t, app, GetExecutionErrors())
	})
}

func TestGetScheduleMetrics(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)
	defer cleanupTestData(t, app.DB)

	t.Run("NoMetrics", func(t *testing.T) {
		schedule := createTestSchedule(t, app.DB, &Schedule{
			Name:           "Metrics Test",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
		})

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/schedules/" + schedule.ID + "/metrics",
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var metrics ScheduleMetrics
		assertJSONResponse(t, rr, http.StatusOK, &metrics)

		if metrics.TotalExecutions != 0 {
			t.Errorf("Expected 0 total executions, got %d", metrics.TotalExecutions)
		}
		if metrics.HealthScore != 100 {
			t.Errorf("Expected health score 100, got %d", metrics.HealthScore)
		}
	})
}

func TestValidateCron(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)
	defer cleanupTestData(t, app.DB)

	t.Run("ValidExpression", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/cron/validate",
			QueryParams: map[string]string{
				"expression": "0 9 * * *",
			},
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var response map[string]interface{}
		assertJSONResponse(t, rr, http.StatusOK, &response)

		if response["valid"] != true {
			t.Error("Expected valid to be true")
		}
		if response["expression"] != "0 9 * * *" {
			t.Errorf("Expected expression '0 9 * * *', got '%v'", response["expression"])
		}
	})

	t.Run("InvalidExpression", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/cron/validate",
			QueryParams: map[string]string{
				"expression": "invalid",
			},
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var response map[string]interface{}
		assertJSONResponse(t, rr, http.StatusOK, &response)

		if response["valid"] != false {
			t.Error("Expected valid to be false")
		}
	})

	t.Run("SpecialExpressions", func(t *testing.T) {
		specialExpressions := []string{"@hourly", "@daily", "@weekly", "@monthly", "@yearly"}

		for _, expr := range specialExpressions {
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/api/cron/validate",
				QueryParams: map[string]string{
					"expression": expr,
				},
			}

			rr, err := makeHTTPRequest(app.Router, req)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}

			var response map[string]interface{}
			assertJSONResponse(t, rr, http.StatusOK, &response)

			if response["valid"] != true {
				t.Errorf("Expected '%s' to be valid", expr)
			}
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		RunErrorScenarios(t, app, ValidateCronErrors())
	})
}

func TestGetCronPresets(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)
	defer cleanupTestData(t, app.DB)

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/cron/presets",
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var presets []CronPreset
		assertJSONResponse(t, rr, http.StatusOK, &presets)

		// Should have at least some presets from schema
		if len(presets) == 0 {
			t.Error("Expected at least some cron presets")
		}
	})
}

func TestGetTimezones(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)
	defer cleanupTestData(t, app.DB)

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/timezones",
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var timezones []map[string]string
		assertJSONResponse(t, rr, http.StatusOK, &timezones)

		if len(timezones) == 0 {
			t.Error("Expected at least some timezones")
		}

		// Check for UTC
		hasUTC := false
		for _, tz := range timezones {
			if tz["value"] == "UTC" {
				hasUTC = true
				break
			}
		}
		if !hasUTC {
			t.Error("Expected UTC to be in timezone list")
		}
	})
}

func TestGetDashboardStats(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)
	defer cleanupTestData(t, app.DB)

	t.Run("Success", func(t *testing.T) {
		// Create test data
		schedule := createTestSchedule(t, app.DB, &Schedule{
			Name:           "Dashboard Test",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
			Enabled:        true,
		})

		createTestExecution(t, app.DB, schedule.ID, "success")
		createTestExecution(t, app.DB, schedule.ID, "failed")

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/dashboard/stats",
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var stats map[string]interface{}
		assertJSONResponse(t, rr, http.StatusOK, &stats)

		// Verify stats contain expected fields
		if _, ok := stats["total_schedules"]; !ok {
			t.Error("Expected total_schedules in response")
		}
		if _, ok := stats["total_executions"]; !ok {
			t.Error("Expected total_executions in response")
		}
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("isValidCron", func(t *testing.T) {
		validExpressions := []string{
			"0 * * * *",
			"*/5 * * * *",
			"0 9 * * *",
			"@hourly",
			"@daily",
		}

		for _, expr := range validExpressions {
			if !isValidCron(expr) {
				t.Errorf("Expected '%s' to be valid", expr)
			}
		}

		invalidExpressions := []string{
			"invalid",
			"",
			"* * * *", // missing field
		}

		for _, expr := range invalidExpressions {
			if isValidCron(expr) {
				t.Errorf("Expected '%s' to be invalid", expr)
			}
		}
	})

	t.Run("calculateNextRuns", func(t *testing.T) {
		runs := calculateNextRuns("0 * * * *", "UTC", 5)

		if len(runs) != 5 {
			t.Errorf("Expected 5 runs, got %d", len(runs))
		}

		// Verify runs are in the future and sequential
		now := time.Now()
		for i, run := range runs {
			if !run.After(now) {
				t.Errorf("Run %d is not in the future", i)
			}
			if i > 0 && !run.After(runs[i-1]) {
				t.Errorf("Run %d is not after run %d", i, i-1)
			}
		}
	})
}

func TestServeDocs(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)
	defer cleanupTestData(t, app.DB)

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/docs",
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		contentType := rr.Header().Get("Content-Type")
		if contentType != "text/html" {
			t.Errorf("Expected Content-Type 'text/html', got '%s'", contentType)
		}

		body := rr.Body.String()
		if body == "" {
			t.Error("Expected non-empty documentation")
		}
	})
}

func TestGetNextRuns(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)
	defer cleanupTestData(t, app.DB)

	t.Run("Success", func(t *testing.T) {
		schedule := createTestSchedule(t, app.DB, &Schedule{
			Name:           "Next Runs Test",
			CronExpression: "0 * * * *",
			Timezone:       "UTC",
			TargetType:     "webhook",
		})

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/schedules/" + schedule.ID + "/next-runs",
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var response map[string]interface{}
		assertJSONResponse(t, rr, http.StatusOK, &response)

		if response["schedule_id"] != schedule.ID {
			t.Errorf("Expected schedule_id '%s', got '%v'", schedule.ID, response["schedule_id"])
		}

		nextRuns, ok := response["next_runs"].([]interface{})
		if !ok {
			t.Fatal("Expected next_runs to be an array")
		}

		if len(nextRuns) != 5 {
			t.Errorf("Expected 5 next runs, got %d", len(nextRuns))
		}
	})

	t.Run("NonExistentSchedule", func(t *testing.T) {
		nonExistentID := uuid.New().String()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/schedules/" + nonExistentID + "/next-runs",
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, rr, http.StatusNotFound, "Schedule not found")
	})
}

func TestRetryExecution(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	app := setupTestApp(t)
	defer cleanupTestData(t, app.DB)

	t.Run("Success", func(t *testing.T) {
		schedule := createTestSchedule(t, app.DB, &Schedule{
			Name:           "Retry Test",
			CronExpression: "0 * * * *",
			TargetType:     "webhook",
		})

		executionID := createTestExecution(t, app.DB, schedule.ID, "failed")

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/executions/" + executionID + "/retry",
		}

		rr, err := makeHTTPRequest(app.Router, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var response map[string]string
		assertJSONResponse(t, rr, http.StatusOK, &response)

		if response["result"] != "success" {
			t.Errorf("Expected result 'success', got '%s'", response["result"])
		}

		// Verify attempt count was incremented
		var attemptCount int
		app.DB.QueryRow("SELECT attempt_count FROM executions WHERE id = $1", executionID).Scan(&attemptCount)
		if attemptCount != 2 {
			t.Errorf("Expected attempt_count 2, got %d", attemptCount)
		}
	})
}
