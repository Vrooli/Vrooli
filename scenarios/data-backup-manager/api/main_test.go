package main

import (
	"net/http"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	// Set lifecycle env var for tests
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
	os.Exit(m.Run())
}

// TestHealthEndpoint tests the health check endpoint
func TestHealthEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := createTestRouter()

	t.Run("Success", func(t *testing.T) {
		req := makeHTTPRequest("GET", "/health", nil)
		rr := executeRequest(router, req)

		response := assertJSONResponse(t, rr, http.StatusOK, func(r map[string]interface{}) bool {
			status, ok := r["status"].(string)
			if !ok || status != "healthy" {
				t.Errorf("Expected status 'healthy', got %v", r["status"])
				return false
			}

			// Validate resources map exists
			resources, ok := r["resources"].(map[string]interface{})
			if !ok {
				t.Error("Expected resources map")
				return false
			}

			// Check for expected resources
			for _, resource := range []string{"postgres", "minio", "n8n"} {
				if _, exists := resources[resource]; !exists {
					t.Errorf("Expected resource %s in health check", resource)
					return false
				}
			}

			return true
		})

		if response != nil {
			// Validate version field
			if _, ok := response["version"].(string); !ok {
				t.Error("Expected version field in response")
			}

			// Validate timestamp
			if _, ok := response["timestamp"].(string); !ok {
				t.Error("Expected timestamp field in response")
			}
		}
	})
}

// TestBackupCreate tests backup creation endpoint
func TestBackupCreate(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := createTestRouter()

	t.Run("SuccessFullBackup", func(t *testing.T) {
		reqBody := BackupCreateRequest{
			Type:          "full",
			Targets:       []string{"postgres"},
			Description:   "Test full backup",
			RetentionDays: 7,
		}

		req := makeHTTPRequest("POST", "/api/v1/backup/create", reqBody)
		rr := executeRequest(router, req)

		response := assertJSONResponse(t, rr, http.StatusCreated, func(r map[string]interface{}) bool {
			if _, ok := r["job_id"].(string); !ok {
				t.Error("Expected job_id in response")
				return false
			}

			if status, ok := r["status"].(string); !ok || status != "pending" {
				t.Errorf("Expected status 'pending', got %v", r["status"])
				return false
			}

			return true
		})

		if response != nil {
			// Validate targets match request
			targets, ok := response["targets"].([]interface{})
			if !ok || len(targets) != 1 {
				t.Error("Expected targets array with one element")
			}
		}
	})

	t.Run("SuccessIncrementalBackup", func(t *testing.T) {
		reqBody := BackupCreateRequest{
			Type:    "incremental",
			Targets: []string{"files", "postgres"},
		}

		req := makeHTTPRequest("POST", "/api/v1/backup/create", reqBody)
		rr := executeRequest(router, req)

		assertJSONResponse(t, rr, http.StatusCreated, func(r map[string]interface{}) bool {
			jobID, ok := r["job_id"].(string)
			if !ok || jobID == "" {
				t.Error("Expected non-empty job_id")
				return false
			}
			return true
		})
	})

	t.Run("SuccessDifferentialBackup", func(t *testing.T) {
		reqBody := BackupCreateRequest{
			Type:    "differential",
			Targets: []string{"minio"},
		}

		req := makeHTTPRequest("POST", "/api/v1/backup/create", reqBody)
		rr := executeRequest(router, req)

		assertJSONResponse(t, rr, http.StatusCreated, nil)
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		patterns := NewTestScenarioBuilder().
			AddInvalidBackupType("/api/v1/backup/create").
			AddEmptyTargets("/api/v1/backup/create").
			Build()

		ExecuteErrorPatterns(t, router, patterns)
	})

	t.Run("EdgeCases", func(t *testing.T) {
		// Test with multiple targets
		reqBody := BackupCreateRequest{
			Type:    "full",
			Targets: []string{"postgres", "files", "minio"},
		}

		req := makeHTTPRequest("POST", "/api/v1/backup/create", reqBody)
		rr := executeRequest(router, req)

		assertJSONResponse(t, rr, http.StatusCreated, func(r map[string]interface{}) bool {
			targets, ok := r["targets"].([]interface{})
			return ok && len(targets) == 3
		})
	})
}

// TestBackupStatus tests backup status endpoint
func TestBackupStatus(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := createTestRouter()

	t.Run("Success", func(t *testing.T) {
		req := makeHTTPRequest("GET", "/api/v1/backup/status", nil)
		rr := executeRequest(router, req)

		response := assertJSONResponse(t, rr, http.StatusOK, func(r map[string]interface{}) bool {
			if status, ok := r["system_status"].(string); !ok || status == "" {
				t.Error("Expected system_status field")
				return false
			}

			if _, ok := r["storage_usage"].(map[string]interface{}); !ok {
				t.Error("Expected storage_usage object")
				return false
			}

			if _, ok := r["resource_health"].(map[string]interface{}); !ok {
				t.Error("Expected resource_health object")
				return false
			}

			return true
		})

		if response != nil {
			// Validate storage usage structure
			storageUsage := response["storage_usage"].(map[string]interface{})
			for _, field := range []string{"used_gb", "available_gb", "compression_ratio"} {
				if _, ok := storageUsage[field]; !ok {
					t.Errorf("Expected %s in storage_usage", field)
				}
			}
		}
	})
}

// TestBackupList tests backup list endpoint
func TestBackupList(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := createTestRouter()

	t.Run("Success", func(t *testing.T) {
		req := makeHTTPRequest("GET", "/api/v1/backup/list", nil)
		rr := executeRequest(router, req)

		response := assertJSONResponse(t, rr, http.StatusOK, func(r map[string]interface{}) bool {
			if _, ok := r["backups"].([]interface{}); !ok {
				t.Error("Expected backups array")
				return false
			}

			if _, ok := r["total"].(float64); !ok {
				t.Error("Expected total count")
				return false
			}

			return true
		})

		if response != nil {
			backups := response["backups"].([]interface{})
			if len(backups) > 0 {
				// Validate backup structure
				backup := backups[0].(map[string]interface{})
				requiredFields := []string{"id", "type", "status", "storage_path"}
				for _, field := range requiredFields {
					if _, ok := backup[field]; !ok {
						t.Errorf("Expected field %s in backup object", field)
					}
				}
			}
		}
	})
}

// TestBackupVerify tests backup verification endpoint
func TestBackupVerify(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := createTestRouter()

	t.Run("Success", func(t *testing.T) {
		backupID := "backup-1234567890"
		req := makeHTTPRequest("POST", "/api/v1/backup/verify/"+backupID, nil)
		rr := executeRequest(router, req)

		response := assertJSONResponse(t, rr, http.StatusOK, func(r map[string]interface{}) bool {
			if id, ok := r["backup_id"].(string); !ok || id != backupID {
				t.Errorf("Expected backup_id %s, got %v", backupID, r["backup_id"])
				return false
			}

			if _, ok := r["verified"].(bool); !ok {
				t.Error("Expected verified boolean field")
				return false
			}

			return true
		})

		if response != nil {
			// Validate verification fields
			for _, field := range []string{"checksum_match", "size_match", "verified_at"} {
				if _, ok := response[field]; !ok {
					t.Errorf("Expected field %s in verification response", field)
				}
			}
		}
	})
}

// TestRestoreCreate tests restore creation endpoint
func TestRestoreCreate(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := createTestRouter()

	t.Run("SuccessWithBackupJobID", func(t *testing.T) {
		reqBody := RestoreCreateRequest{
			BackupJobID:         "backup-1234567890",
			Targets:             []string{"postgres"},
			VerifyBeforeRestore: true,
		}

		req := makeHTTPRequest("POST", "/api/v1/restore/create", reqBody)
		rr := executeRequest(router, req)

		response := assertJSONResponse(t, rr, http.StatusCreated, func(r map[string]interface{}) bool {
			if _, ok := r["restore_id"].(string); !ok {
				t.Error("Expected restore_id in response")
				return false
			}

			if status, ok := r["status"].(string); !ok || status != "pending" {
				t.Errorf("Expected status 'pending', got %v", r["status"])
				return false
			}

			return true
		})

		if response != nil {
			// Validate estimated duration
			if _, ok := response["estimated_duration"].(string); !ok {
				t.Error("Expected estimated_duration field")
			}
		}
	})

	t.Run("SuccessWithRestorePointID", func(t *testing.T) {
		reqBody := RestoreCreateRequest{
			RestorePointID: "restore-point-123",
			Targets:        []string{"files"},
			Destination:    "/tmp/restore",
		}

		req := makeHTTPRequest("POST", "/api/v1/restore/create", reqBody)
		rr := executeRequest(router, req)

		assertJSONResponse(t, rr, http.StatusCreated, nil)
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		patterns := NewTestScenarioBuilder().
			AddEmptyTargets("/api/v1/restore/create").
			AddMissingRestoreID("/api/v1/restore/create").
			Build()

		ExecuteErrorPatterns(t, router, patterns)
	})
}

// TestRestoreStatus tests restore status endpoint
func TestRestoreStatus(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := createTestRouter()

	t.Run("Success", func(t *testing.T) {
		restoreID := "restore-1234567890"
		req := makeHTTPRequest("GET", "/api/v1/restore/status/"+restoreID, nil)
		rr := executeRequest(router, req)

		response := assertJSONResponse(t, rr, http.StatusOK, func(r map[string]interface{}) bool {
			if id, ok := r["restore_id"].(string); !ok || id != restoreID {
				t.Errorf("Expected restore_id %s, got %v", restoreID, r["restore_id"])
				return false
			}

			if _, ok := r["status"].(string); !ok {
				t.Error("Expected status field")
				return false
			}

			return true
		})

		if response != nil {
			// Validate progress field
			if _, ok := response["progress"].(float64); !ok {
				t.Error("Expected progress field")
			}
		}
	})
}

// TestScheduleEndpoints tests schedule CRUD operations
func TestScheduleEndpoints(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := createTestRouter()

	t.Run("ListSchedules", func(t *testing.T) {
		req := makeHTTPRequest("GET", "/api/v1/schedules", nil)
		rr := executeRequest(router, req)

		response := assertJSONResponse(t, rr, http.StatusOK, func(r map[string]interface{}) bool {
			if _, ok := r["schedules"].([]interface{}); !ok {
				t.Error("Expected schedules array")
				return false
			}
			return true
		})

		if response != nil {
			schedules := response["schedules"].([]interface{})
			if len(schedules) > 0 {
				schedule := schedules[0].(map[string]interface{})
				requiredFields := []string{"id", "name", "cron_expression", "backup_type", "targets"}
				for _, field := range requiredFields {
					if _, ok := schedule[field]; !ok {
						t.Errorf("Expected field %s in schedule", field)
					}
				}
			}
		}
	})

	t.Run("CreateSchedule", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name":            "Test Schedule",
			"cron_expression": "0 2 * * *",
			"backup_type":     "full",
			"targets":         []string{"postgres"},
			"retention_days":  7,
		}

		req := makeHTTPRequest("POST", "/api/v1/schedules", reqBody)
		rr := executeRequest(router, req)

		response := assertJSONResponse(t, rr, http.StatusCreated, func(r map[string]interface{}) bool {
			if _, ok := r["id"].(string); !ok {
				t.Error("Expected id in response")
				return false
			}

			if enabled, ok := r["enabled"].(bool); !ok || !enabled {
				t.Error("Expected schedule to be enabled by default")
				return false
			}

			return true
		})

		if response != nil {
			// Validate created_at timestamp
			if _, ok := response["created_at"].(string); !ok {
				t.Error("Expected created_at timestamp")
			}
		}
	})

	t.Run("UpdateSchedule", func(t *testing.T) {
		scheduleID := "schedule-123"
		reqBody := map[string]interface{}{
			"cron_expression": "0 3 * * *",
			"enabled":         false,
		}

		req := makeHTTPRequest("PUT", "/api/v1/schedules/"+scheduleID, reqBody)
		rr := executeRequest(router, req)

		response := assertJSONResponse(t, rr, http.StatusOK, func(r map[string]interface{}) bool {
			if id, ok := r["id"].(string); !ok || id != scheduleID {
				t.Errorf("Expected id %s, got %v", scheduleID, r["id"])
				return false
			}
			return true
		})

		if response != nil {
			// Validate updated_at timestamp
			if _, ok := response["updated_at"].(string); !ok {
				t.Error("Expected updated_at timestamp")
			}
		}
	})

	t.Run("DeleteSchedule", func(t *testing.T) {
		scheduleID := "schedule-123"
		req := makeHTTPRequest("DELETE", "/api/v1/schedules/"+scheduleID, nil)
		rr := executeRequest(router, req)

		if rr.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", rr.Code)
		}
	})
}

// TestComplianceEndpoints tests compliance-related endpoints
func TestComplianceEndpoints(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := createTestRouter()

	t.Run("ComplianceReport", func(t *testing.T) {
		req := makeHTTPRequest("GET", "/api/v1/compliance/report", nil)
		rr := executeRequest(router, req)

		response := assertJSONResponse(t, rr, http.StatusOK, func(r map[string]interface{}) bool {
			requiredFields := []string{"total_resources", "compliant", "non_compliant", "compliance_score"}
			for _, field := range requiredFields {
				if _, ok := r[field]; !ok {
					t.Errorf("Expected field %s in compliance report", field)
					return false
				}
			}
			return true
		})

		if response != nil {
			// Validate issues array
			if issues, ok := response["issues"].([]interface{}); ok {
				if len(issues) > 0 {
					issue := issues[0].(map[string]interface{})
					for _, field := range []string{"id", "severity", "title"} {
						if _, ok := issue[field]; !ok {
							t.Errorf("Expected field %s in issue", field)
						}
					}
				}
			}
		}
	})

	t.Run("ComplianceScan", func(t *testing.T) {
		req := makeHTTPRequest("POST", "/api/v1/compliance/scan", nil)
		rr := executeRequest(router, req)

		response := assertJSONResponse(t, rr, http.StatusAccepted, func(r map[string]interface{}) bool {
			if _, ok := r["scan_id"].(string); !ok {
				t.Error("Expected scan_id in response")
				return false
			}

			if status, ok := r["status"].(string); !ok || status != "started" {
				t.Errorf("Expected status 'started', got %v", r["status"])
				return false
			}

			return true
		})

		if response != nil {
			// Validate estimated time
			if _, ok := response["estimated"].(string); !ok {
				t.Error("Expected estimated field")
			}
		}
	})

	t.Run("ComplianceFix", func(t *testing.T) {
		issueID := "issue-001"
		req := makeHTTPRequest("POST", "/api/v1/compliance/issue/"+issueID+"/fix", nil)
		rr := executeRequest(router, req)

		response := assertJSONResponse(t, rr, http.StatusOK, func(r map[string]interface{}) bool {
			if id, ok := r["issue_id"].(string); !ok || id != issueID {
				t.Errorf("Expected issue_id %s, got %v", issueID, r["issue_id"])
				return false
			}

			if status, ok := r["status"].(string); !ok || status != "fixing" {
				t.Errorf("Expected status 'fixing', got %v", r["status"])
				return false
			}

			return true
		})

		if response != nil {
			// Validate action field
			if _, ok := response["action"].(string); !ok {
				t.Error("Expected action field")
			}
		}
	})
}

// TestVisitedTrackerIntegration tests visited tracker integration
func TestVisitedTrackerIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := createTestRouter()

	t.Run("RecordVisit", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"type": "scenario",
			"name": "test-scenario",
			"path": "/scenarios/test-scenario",
		}

		req := makeHTTPRequest("POST", "/api/v1/visited/record", reqBody)
		rr := executeRequest(router, req)

		response := assertJSONResponse(t, rr, http.StatusCreated, func(r map[string]interface{}) bool {
			if success, ok := r["success"].(bool); !ok || !success {
				t.Error("Expected success to be true")
				return false
			}
			return true
		})

		if response != nil {
			// Validate data field contains original record
			if data, ok := response["data"].(map[string]interface{}); ok {
				if _, ok := data["visited_at"].(string); !ok {
					t.Error("Expected visited_at timestamp in data")
				}
			}
		}
	})

	t.Run("GetNextUnvisited", func(t *testing.T) {
		req := makeHTTPRequest("GET", "/api/v1/visited/next", nil)
		rr := executeRequest(router, req)

		response := assertJSONResponse(t, rr, http.StatusOK, func(r map[string]interface{}) bool {
			requiredFields := []string{"type", "name", "path"}
			for _, field := range requiredFields {
				if _, ok := r[field]; !ok {
					t.Errorf("Expected field %s", field)
					return false
				}
			}
			return true
		})

		if response != nil {
			// Validate priority and reason fields
			if _, ok := response["priority"].(string); !ok {
				t.Error("Expected priority field")
			}
			if _, ok := response["reason"].(string); !ok {
				t.Error("Expected reason field")
			}
		}
	})
}

// TestMaintenanceOrchestratorIntegration tests maintenance orchestrator integration
func TestMaintenanceOrchestratorIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := createTestRouter()

	t.Run("MaintenanceStatus", func(t *testing.T) {
		req := makeHTTPRequest("GET", "/api/v1/maintenance/status", nil)
		rr := executeRequest(router, req)

		response := assertJSONResponse(t, rr, http.StatusOK, func(r map[string]interface{}) bool {
			if serviceName, ok := r["service_name"].(string); !ok || serviceName != "data-backup-manager" {
				t.Error("Expected service_name to be data-backup-manager")
				return false
			}

			if _, ok := r["agent_enabled"].(bool); !ok {
				t.Error("Expected agent_enabled field")
				return false
			}

			return true
		})

		if response != nil {
			// Validate available tasks
			if tasks, ok := response["available_tasks"].([]interface{}); ok {
				if len(tasks) == 0 {
					t.Error("Expected available_tasks to not be empty")
				}
			}

			// Validate dependencies
			if deps, ok := response["dependencies"].(map[string]interface{}); ok {
				if len(deps) == 0 {
					t.Error("Expected dependencies to not be empty")
				}
			}
		}
	})

	t.Run("ExecuteMaintenanceTask", func(t *testing.T) {
		taskTypes := []string{"backup_postgres", "compliance_scan", "cleanup_old_backups"}

		for _, taskType := range taskTypes {
			t.Run(taskType, func(t *testing.T) {
				reqBody := map[string]interface{}{
					"task_type": taskType,
				}

				req := makeHTTPRequest("POST", "/api/v1/maintenance/task", reqBody)
				rr := executeRequest(router, req)

				response := assertJSONResponse(t, rr, http.StatusAccepted, func(r map[string]interface{}) bool {
					if _, ok := r["task_id"].(string); !ok {
						t.Error("Expected task_id in response")
						return false
					}

					if tt, ok := r["task_type"].(string); !ok || tt != taskType {
						t.Errorf("Expected task_type %s, got %v", taskType, r["task_type"])
						return false
					}

					return true
				})

				if response != nil {
					// Validate status and estimated time
					if status, ok := response["status"].(string); !ok || status != "running" {
						t.Errorf("Expected status 'running', got %v", response["status"])
					}
					if _, ok := response["estimated"].(string); !ok {
						t.Error("Expected estimated field")
					}
				}
			})
		}
	})

	t.Run("ToggleMaintenanceAgent", func(t *testing.T) {
		// Test enabling
		reqBody := map[string]interface{}{
			"enabled": true,
		}

		req := makeHTTPRequest("POST", "/api/v1/maintenance/agent/toggle", reqBody)
		rr := executeRequest(router, req)

		response := assertJSONResponse(t, rr, http.StatusOK, func(r map[string]interface{}) bool {
			if enabled, ok := r["agent_enabled"].(bool); !ok || !enabled {
				t.Error("Expected agent_enabled to be true")
				return false
			}

			if _, ok := r["next_scheduled"]; !ok {
				t.Error("Expected next_scheduled when enabled")
				return false
			}

			return true
		})

		if response != nil {
			// Validate schedule field when enabled
			if _, ok := response["schedule"].(string); !ok {
				t.Error("Expected schedule field when enabled")
			}
		}

		// Test disabling
		reqBody = map[string]interface{}{
			"enabled": false,
		}

		req = makeHTTPRequest("POST", "/api/v1/maintenance/agent/toggle", reqBody)
		rr = executeRequest(router, req)

		assertJSONResponse(t, rr, http.StatusOK, func(r map[string]interface{}) bool {
			if enabled, ok := r["agent_enabled"].(bool); !ok || enabled {
				t.Error("Expected agent_enabled to be false")
				return false
			}
			return true
		})
	})
}

// TestEdgeCases tests edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := createTestRouter()

	t.Run("EmptyDescription", func(t *testing.T) {
		reqBody := BackupCreateRequest{
			Type:        "full",
			Targets:     []string{"postgres"},
			Description: "",
		}

		req := makeHTTPRequest("POST", "/api/v1/backup/create", reqBody)
		rr := executeRequest(router, req)

		// Should succeed with empty description
		assertJSONResponse(t, rr, http.StatusCreated, nil)
	})

	t.Run("ZeroRetentionDays", func(t *testing.T) {
		reqBody := BackupCreateRequest{
			Type:          "incremental",
			Targets:       []string{"files"},
			RetentionDays: 0,
		}

		req := makeHTTPRequest("POST", "/api/v1/backup/create", reqBody)
		rr := executeRequest(router, req)

		// Should succeed with zero retention
		assertJSONResponse(t, rr, http.StatusCreated, nil)
	})

	t.Run("LargeRetentionDays", func(t *testing.T) {
		reqBody := BackupCreateRequest{
			Type:          "full",
			Targets:       []string{"postgres"},
			RetentionDays: 365,
		}

		req := makeHTTPRequest("POST", "/api/v1/backup/create", reqBody)
		rr := executeRequest(router, req)

		assertJSONResponse(t, rr, http.StatusCreated, nil)
	})

	t.Run("UnknownBackupTarget", func(t *testing.T) {
		reqBody := BackupCreateRequest{
			Type:    "full",
			Targets: []string{"unknown-target"},
		}

		req := makeHTTPRequest("POST", "/api/v1/backup/create", reqBody)
		rr := executeRequest(router, req)

		// Should still succeed (target validation happens during execution)
		assertJSONResponse(t, rr, http.StatusCreated, nil)
	})

	t.Run("BothRestoreIDsProvided", func(t *testing.T) {
		reqBody := RestoreCreateRequest{
			RestorePointID: "restore-point-123",
			BackupJobID:    "backup-123",
			Targets:        []string{"postgres"},
		}

		req := makeHTTPRequest("POST", "/api/v1/restore/create", reqBody)
		rr := executeRequest(router, req)

		// Should succeed - backup_job_id takes precedence
		assertJSONResponse(t, rr, http.StatusCreated, nil)
	})
}

// TestConcurrentRequests tests handling of concurrent requests
func TestConcurrentRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := createTestRouter()

	t.Run("ConcurrentBackupCreation", func(t *testing.T) {
		numRequests := 10
		done := make(chan bool, numRequests)

		for i := 0; i < numRequests; i++ {
			go func(index int) {
				reqBody := BackupCreateRequest{
					Type:        "incremental",
					Targets:     []string{"postgres"},
					Description: time.Now().String(),
				}

				req := makeHTTPRequest("POST", "/api/v1/backup/create", reqBody)
				rr := executeRequest(router, req)

				if rr.Code != http.StatusCreated {
					t.Errorf("Request %d failed with status %d", index, rr.Code)
				}

				done <- true
			}(i)
		}

		// Wait for all requests to complete
		timeout := time.After(5 * time.Second)
		for i := 0; i < numRequests; i++ {
			select {
			case <-done:
				// Request completed
			case <-timeout:
				t.Fatal("Timeout waiting for concurrent requests")
			}
		}
	})
}
