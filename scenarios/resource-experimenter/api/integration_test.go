package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestExperimentWorkflow tests the complete experiment workflow
func TestExperimentWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("CompleteExperimentLifecycle", func(t *testing.T) {
		// Step 1: Create experiment
		req := ExperimentRequest{
			Name:           "Integration Test Experiment",
			Description:    "Testing complete workflow",
			Prompt:         "Integrate resource X into scenario Y",
			TargetScenario: "test-scenario",
			NewResource:    "test-resource",
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/experiments",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to create experiment: %v", err)
		}

		var exp Experiment
		assertJSONResponse(t, w, http.StatusOK, &exp)

		if exp.ID == "" {
			t.Fatal("Expected experiment ID to be set")
		}
		if exp.Status != "requested" {
			t.Errorf("Expected status 'requested', got '%s'", exp.Status)
		}

		// Step 2: Retrieve experiment
		w, err = makeHTTPRequest(env, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/experiments/%s", exp.ID),
			URLVars: map[string]string{"id": exp.ID},
		})
		if err != nil {
			t.Fatalf("Failed to retrieve experiment: %v", err)
		}

		var retrievedExp Experiment
		assertJSONResponse(t, w, http.StatusOK, &retrievedExp)

		if retrievedExp.ID != exp.ID {
			t.Errorf("Expected ID %s, got %s", exp.ID, retrievedExp.ID)
		}

		// Step 3: Update experiment status to running
		updates := map[string]interface{}{
			"status": "running",
		}

		w, err = makeHTTPRequest(env, HTTPTestRequest{
			Method:  "PUT",
			Path:    fmt.Sprintf("/api/experiments/%s", exp.ID),
			URLVars: map[string]string{"id": exp.ID},
			Body:    updates,
		})
		if err != nil {
			t.Fatalf("Failed to update experiment: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK, nil)

		// Step 4: Add experiment logs
		logID1 := uuid.New().String()
		logID2 := uuid.New().String()

		query := `INSERT INTO experiment_logs (id, experiment_id, step, prompt, response, success, started_at, completed_at)
		          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

		env.DB.Exec(query, logID1, exp.ID, "analyze-scenario", "Analyzing target scenario...", "Analysis complete", true, time.Now(), time.Now())
		env.DB.Exec(query, logID2, exp.ID, "integrate-resource", "Integrating new resource...", "Integration complete", true, time.Now(), time.Now())

		// Step 5: Retrieve logs
		w, err = makeHTTPRequest(env, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/experiments/%s/logs", exp.ID),
			URLVars: map[string]string{"id": exp.ID},
		})
		if err != nil {
			t.Fatalf("Failed to retrieve logs: %v", err)
		}

		var logs []ExperimentLog
		assertJSONResponse(t, w, http.StatusOK, &logs)

		if len(logs) != 2 {
			t.Errorf("Expected 2 logs, got %d", len(logs))
		}

		// Step 6: Update to completed
		updates = map[string]interface{}{
			"status": "completed",
		}

		w, err = makeHTTPRequest(env, HTTPTestRequest{
			Method:  "PUT",
			Path:    fmt.Sprintf("/api/experiments/%s", exp.ID),
			URLVars: map[string]string{"id": exp.ID},
			Body:    updates,
		})
		if err != nil {
			t.Fatalf("Failed to update to completed: %v", err)
		}

		// Step 7: List all experiments and verify our experiment is included
		w, err = makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/experiments",
		})
		if err != nil {
			t.Fatalf("Failed to list experiments: %v", err)
		}

		var allExperiments []Experiment
		assertJSONResponse(t, w, http.StatusOK, &allExperiments)

		found := false
		for _, e := range allExperiments {
			if e.ID == exp.ID {
				found = true
				break
			}
		}

		if !found {
			t.Error("Created experiment not found in list")
		}

		// Step 8: Delete experiment and logs
		w, err = makeHTTPRequest(env, HTTPTestRequest{
			Method:  "DELETE",
			Path:    fmt.Sprintf("/api/experiments/%s", exp.ID),
			URLVars: map[string]string{"id": exp.ID},
		})
		if err != nil {
			t.Fatalf("Failed to delete experiment: %v", err)
		}

		assertJSONResponse(t, w, http.StatusOK, nil)

		// Step 9: Verify deletion
		var count int
		env.DB.QueryRow("SELECT COUNT(*) FROM experiments WHERE id = $1", exp.ID).Scan(&count)
		if count != 0 {
			t.Error("Experiment not deleted from database")
		}

		// Verify logs cascade deleted
		env.DB.QueryRow("SELECT COUNT(*) FROM experiment_logs WHERE experiment_id = $1", exp.ID).Scan(&count)
		if count != 0 {
			t.Error("Experiment logs not cascade deleted")
		}
	})

	t.Run("TemplateUsageWorkflow", func(t *testing.T) {
		// Create template
		tmpl := createTestTemplate(t, env, "workflow-template")

		// List templates
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/templates",
		})
		if err != nil {
			t.Fatalf("Failed to list templates: %v", err)
		}

		var templates []ExperimentTemplate
		assertJSONResponse(t, w, http.StatusOK, &templates)

		found := false
		for _, t := range templates {
			if t.ID == tmpl.ID {
				found = true
				break
			}
		}

		if !found {
			t.Error("Created template not found in list")
		}

		// Increment usage count (simulating template usage)
		_, err = env.DB.Exec("UPDATE experiment_templates SET usage_count = usage_count + 1 WHERE id = $1", tmpl.ID)
		if err != nil {
			t.Fatalf("Failed to update usage count: %v", err)
		}

		// Verify update
		var usageCount int
		env.DB.QueryRow("SELECT usage_count FROM experiment_templates WHERE id = $1", tmpl.ID).Scan(&usageCount)
		if usageCount != 1 {
			t.Errorf("Expected usage_count 1, got %d", usageCount)
		}
	})

	t.Run("ScenarioDiscoveryWorkflow", func(t *testing.T) {
		// Create multiple scenarios with different characteristics
		scenario1 := createTestScenario(t, env, "simple-scenario")
		scenario2 := createTestScenario(t, env, "complex-scenario")

		// Update complexity levels
		env.DB.Exec("UPDATE available_scenarios SET complexity_level = 'low' WHERE id = $1", scenario1.ID)
		env.DB.Exec("UPDATE available_scenarios SET complexity_level = 'high' WHERE id = $1", scenario2.ID)

		// List all scenarios
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/scenarios",
		})
		if err != nil {
			t.Fatalf("Failed to list scenarios: %v", err)
		}

		var scenarios []AvailableScenario
		assertJSONResponse(t, w, http.StatusOK, &scenarios)

		if len(scenarios) < 2 {
			t.Errorf("Expected at least 2 scenarios, got %d", len(scenarios))
		}

		// Verify JSON arrays are properly populated
		for _, s := range scenarios {
			if s.ID == scenario1.ID || s.ID == scenario2.ID {
				if len(s.CurrentResources) == 0 {
					t.Error("Expected CurrentResources to be populated")
				}
				if len(s.ResourceCategories) == 0 {
					t.Error("Expected ResourceCategories to be populated")
				}
			}
		}
	})
}

// TestExperimentStatusTransitions tests valid and invalid status transitions
func TestExperimentStatusTransitions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("ValidStatusTransitions", func(t *testing.T) {
		exp := createTestExperiment(t, env, "status-transition-test")

		// requested -> running
		updates := map[string]interface{}{"status": "running"}
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "PUT",
			Path:    fmt.Sprintf("/api/experiments/%s", exp.ID),
			URLVars: map[string]string{"id": exp.ID},
			Body:    updates,
		})
		if err != nil {
			t.Fatalf("Failed to update: %v", err)
		}
		assertJSONResponse(t, w, http.StatusOK, nil)

		// running -> completed
		updates = map[string]interface{}{"status": "completed"}
		w, err = makeHTTPRequest(env, HTTPTestRequest{
			Method:  "PUT",
			Path:    fmt.Sprintf("/api/experiments/%s", exp.ID),
			URLVars: map[string]string{"id": exp.ID},
			Body:    updates,
		})
		if err != nil {
			t.Fatalf("Failed to update: %v", err)
		}
		assertJSONResponse(t, w, http.StatusOK, nil)

		// Verify final status
		var status string
		env.DB.QueryRow("SELECT status FROM experiments WHERE id = $1", exp.ID).Scan(&status)
		if status != "completed" {
			t.Errorf("Expected status 'completed', got '%s'", status)
		}
	})

	t.Run("FailedStatusTransition", func(t *testing.T) {
		exp := createTestExperiment(t, env, "failed-transition-test")

		// requested -> failed
		updates := map[string]interface{}{"status": "failed"}
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "PUT",
			Path:    fmt.Sprintf("/api/experiments/%s", exp.ID),
			URLVars: map[string]string{"id": exp.ID},
			Body:    updates,
		})
		if err != nil {
			t.Fatalf("Failed to update: %v", err)
		}
		assertJSONResponse(t, w, http.StatusOK, nil)

		// Verify status
		var status string
		env.DB.QueryRow("SELECT status FROM experiments WHERE id = $1", exp.ID).Scan(&status)
		if status != "failed" {
			t.Errorf("Expected status 'failed', got '%s'", status)
		}
	})
}

// TestMultipleExperimentsManagement tests managing multiple experiments
func TestMultipleExperimentsManagement(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("CreateMultipleExperiments", func(t *testing.T) {
		numExperiments := 10
		createdIDs := []string{}

		for i := 0; i < numExperiments; i++ {
			req := ExperimentRequest{
				Name:           fmt.Sprintf("Batch Experiment %d", i),
				Description:    "Batch test",
				Prompt:         "Test",
				TargetScenario: "test-scenario",
				NewResource:    "test-resource",
			}

			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/experiments",
				Body:   req,
			})
			if err != nil {
				t.Fatalf("Failed to create experiment %d: %v", i, err)
			}

			var exp Experiment
			assertJSONResponse(t, w, http.StatusOK, &exp)
			createdIDs = append(createdIDs, exp.ID)
		}

		// List and verify all created
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/experiments",
		})
		if err != nil {
			t.Fatalf("Failed to list experiments: %v", err)
		}

		var experiments []Experiment
		assertJSONResponse(t, w, http.StatusOK, &experiments)

		if len(experiments) < numExperiments {
			t.Errorf("Expected at least %d experiments, got %d", numExperiments, len(experiments))
		}

		// Verify all IDs present
		for _, id := range createdIDs {
			found := false
			for _, exp := range experiments {
				if exp.ID == id {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Experiment %s not found in list", id)
			}
		}
	})

	t.Run("FilterByStatus", func(t *testing.T) {
		// Clean up
		env.DB.Exec("TRUNCATE TABLE experiments CASCADE")

		// Create experiments with different statuses
		exp1 := createTestExperiment(t, env, "status-filter-1")
		exp2 := createTestExperiment(t, env, "status-filter-2")
		exp3 := createTestExperiment(t, env, "status-filter-3")

		// Update statuses and verify (cast string to UUID)
		result1, err1 := env.DB.Exec("UPDATE experiments SET status = 'completed' WHERE id = $1::uuid", exp1.ID)
		result2, err2 := env.DB.Exec("UPDATE experiments SET status = 'running' WHERE id = $1::uuid", exp2.ID)
		result3, err3 := env.DB.Exec("UPDATE experiments SET status = 'completed' WHERE id = $1::uuid", exp3.ID)

		// Verify updates succeeded
		if err1 != nil || err2 != nil || err3 != nil {
			t.Fatalf("Failed to update statuses: %v, %v, %v", err1, err2, err3)
		}

		rows1, _ := result1.RowsAffected()
		rows2, _ := result2.RowsAffected()
		rows3, _ := result3.RowsAffected()

		if rows1 != 1 || rows2 != 1 || rows3 != 1 {
			t.Fatalf("Update failed - rows affected: %d, %d, %d (expected 1, 1, 1)", rows1, rows2, rows3)
		}

		// Query for completed
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/experiments?status=completed",
		})
		if err != nil {
			t.Fatalf("Failed to query: %v", err)
		}

		var completed []Experiment
		assertJSONResponse(t, w, http.StatusOK, &completed)

		// Log what we got for debugging
		if len(completed) != 2 {
			t.Logf("Experiments returned: %d", len(completed))
			for i, exp := range completed {
				t.Logf("  [%d] ID: %s, Status: %s, Name: %s", i, exp.ID, exp.Status, exp.Name)
			}
			t.Errorf("Expected 2 completed experiments, got %d", len(completed))
		}

		// Verify only completed returned
		for _, exp := range completed {
			if exp.Status != "completed" {
				t.Errorf("Expected status 'completed', got '%s'", exp.Status)
			}
		}
	})

	t.Run("LimitResults", func(t *testing.T) {
		// Clean up and create test data
		env.DB.Exec("TRUNCATE TABLE experiments CASCADE")

		for i := 0; i < 15; i++ {
			createTestExperiment(t, env, fmt.Sprintf("limit-test-%d", i))
		}

		// Query with limit
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/experiments?limit=5",
		})
		if err != nil {
			t.Fatalf("Failed to query: %v", err)
		}

		var limited []Experiment
		assertJSONResponse(t, w, http.StatusOK, &limited)

		if len(limited) != 5 {
			t.Errorf("Expected 5 experiments with limit, got %d", len(limited))
		}
	})
}

// TestCascadingDeletes tests that related data is properly deleted
func TestCascadingDeletes(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("ExperimentWithLogs", func(t *testing.T) {
		exp := createTestExperiment(t, env, "cascade-test")

		// Create multiple logs
		for i := 0; i < 5; i++ {
			logID := uuid.New().String()
			query := `INSERT INTO experiment_logs (id, experiment_id, step, prompt, success, started_at)
			          VALUES ($1, $2, $3, $4, $5, NOW())`
			env.DB.Exec(query, logID, exp.ID, fmt.Sprintf("step-%d", i), "test", true)
		}

		// Verify logs created
		var count int
		env.DB.QueryRow("SELECT COUNT(*) FROM experiment_logs WHERE experiment_id = $1", exp.ID).Scan(&count)
		if count != 5 {
			t.Errorf("Expected 5 logs, found %d", count)
		}

		// Delete experiment
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "DELETE",
			Path:    fmt.Sprintf("/api/experiments/%s", exp.ID),
			URLVars: map[string]string{"id": exp.ID},
		})
		if err != nil {
			t.Fatalf("Failed to delete: %v", err)
		}
		assertJSONResponse(t, w, http.StatusOK, nil)

		// Verify logs cascade deleted
		env.DB.QueryRow("SELECT COUNT(*) FROM experiment_logs WHERE experiment_id = $1", exp.ID).Scan(&count)
		if count != 0 {
			t.Errorf("Expected 0 logs after cascade delete, found %d", count)
		}
	})
}

// TestDatabaseTransactions tests transaction handling
func TestDatabaseTransactions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("ConcurrentUpdates", func(t *testing.T) {
		exp := createTestExperiment(t, env, "concurrent-updates")

		// Attempt concurrent updates
		done := make(chan bool, 2)

		go func() {
			updates := map[string]interface{}{"status": "running"}
			makeHTTPRequest(env, HTTPTestRequest{
				Method:  "PUT",
				Path:    fmt.Sprintf("/api/experiments/%s", exp.ID),
				URLVars: map[string]string{"id": exp.ID},
				Body:    updates,
			})
			done <- true
		}()

		go func() {
			updates := map[string]interface{}{"status": "completed"}
			makeHTTPRequest(env, HTTPTestRequest{
				Method:  "PUT",
				Path:    fmt.Sprintf("/api/experiments/%s", exp.ID),
				URLVars: map[string]string{"id": exp.ID},
				Body:    updates,
			})
			done <- true
		}()

		<-done
		<-done

		// Verify one of the updates succeeded
		var status string
		env.DB.QueryRow("SELECT status FROM experiments WHERE id = $1", exp.ID).Scan(&status)

		if status != "running" && status != "completed" {
			t.Errorf("Expected status 'running' or 'completed', got '%s'", status)
		}
	})
}

// TestDataIntegrity tests data integrity constraints
func TestDataIntegrity(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("ExperimentIDUniqueness", func(t *testing.T) {
		exp := createTestExperiment(t, env, "unique-test")

		// Attempt to insert duplicate ID (should fail)
		query := `INSERT INTO experiments (id, name, description, prompt, target_scenario, new_resource, status, created_at, updated_at)
		          VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())`

		_, err := env.DB.Exec(query, exp.ID, "Duplicate", "Desc", "Prompt", "scenario", "resource", "requested")

		if err == nil {
			t.Error("Expected error for duplicate experiment ID, got nil")
		}
	})

	t.Run("LogForeignKeyConstraint", func(t *testing.T) {
		nonExistentID := uuid.New().String()

		// Attempt to create log for non-existent experiment (should fail)
		logID := uuid.New().String()
		query := `INSERT INTO experiment_logs (id, experiment_id, step, prompt, success, started_at)
		          VALUES ($1, $2, $3, $4, $5, NOW())`

		_, err := env.DB.Exec(query, logID, nonExistentID, "step", "prompt", true)

		if err == nil {
			t.Error("Expected foreign key constraint error, got nil")
		}
	})
}
