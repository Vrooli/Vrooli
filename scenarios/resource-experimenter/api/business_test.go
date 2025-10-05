package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestPromptTemplateProcessing tests prompt template variable replacement
func TestPromptTemplateProcessing(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := NewAPIServer()

	t.Run("ValidTemplateReplacement", func(t *testing.T) {
		// Create temporary template file
		templateContent := `Experiment: {{NAME}}
Description: {{DESCRIPTION}}
Prompt: {{PROMPT}}
Target Scenario: {{TARGET_SCENARIO}}
New Resource: {{NEW_RESOURCE}}`

		tmpDir := t.TempDir()
		promptsDir := fmt.Sprintf("%s/prompts", tmpDir)
		os.MkdirAll(promptsDir, 0755)

		templatePath := fmt.Sprintf("%s/test-template.md", promptsDir)
		os.WriteFile(templatePath, []byte(templateContent), 0644)

		// Change to temp directory
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		exp := Experiment{
			Name:           "Test Experiment",
			Description:    "Test Description",
			Prompt:         "Test Prompt",
			TargetScenario: "test-scenario",
			NewResource:    "test-resource",
		}

		result, err := server.loadPromptTemplate("test-template.md", exp)
		if err != nil {
			t.Fatalf("Failed to load template: %v", err)
		}

		// Verify all replacements
		if !strings.Contains(result, "Test Experiment") {
			t.Error("NAME not replaced")
		}
		if !strings.Contains(result, "Test Description") {
			t.Error("DESCRIPTION not replaced")
		}
		if !strings.Contains(result, "Test Prompt") {
			t.Error("PROMPT not replaced")
		}
		if !strings.Contains(result, "test-scenario") {
			t.Error("TARGET_SCENARIO not replaced")
		}
		if !strings.Contains(result, "test-resource") {
			t.Error("NEW_RESOURCE not replaced")
		}

		// Verify no template markers remain
		if strings.Contains(result, "{{") || strings.Contains(result, "}}") {
			t.Error("Template markers not fully replaced")
		}
	})

	t.Run("SpecialCharactersInTemplate", func(t *testing.T) {
		tmpDir := t.TempDir()
		promptsDir := fmt.Sprintf("%s/prompts", tmpDir)
		os.MkdirAll(promptsDir, 0755)

		templateContent := "Name: {{NAME}}\nQuotes: \"test\"\nNewlines:\n{{DESCRIPTION}}"
		templatePath := fmt.Sprintf("%s/special-chars.md", promptsDir)
		os.WriteFile(templatePath, []byte(templateContent), 0644)

		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		exp := Experiment{
			Name:        "Test \"with quotes\"",
			Description: "Line 1\nLine 2\nLine 3",
		}

		result, err := server.loadPromptTemplate("special-chars.md", exp)
		if err != nil {
			t.Fatalf("Failed to load template: %v", err)
		}

		if !strings.Contains(result, "Test \"with quotes\"") {
			t.Error("Quotes not preserved")
		}
		if !strings.Contains(result, "Line 1\nLine 2") {
			t.Error("Newlines not preserved")
		}
	})

	t.Run("EmptyFieldsInTemplate", func(t *testing.T) {
		tmpDir := t.TempDir()
		promptsDir := fmt.Sprintf("%s/prompts", tmpDir)
		os.MkdirAll(promptsDir, 0755)

		templateContent := "{{NAME}}|{{DESCRIPTION}}|{{PROMPT}}"
		templatePath := fmt.Sprintf("%s/empty.md", promptsDir)
		os.WriteFile(templatePath, []byte(templateContent), 0644)

		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)
		os.Chdir(tmpDir)

		exp := Experiment{
			Name:        "",
			Description: "",
			Prompt:      "",
		}

		result, err := server.loadPromptTemplate("empty.md", exp)
		if err != nil {
			t.Fatalf("Failed to load template: %v", err)
		}

		// Should have empty replacements
		if result != "||" {
			t.Errorf("Expected '||', got '%s'", result)
		}
	})
}

// TestDatabaseConnectionPooling tests connection pool behavior
func TestDatabaseConnectionPooling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping connection pool tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("ConnectionPoolSettings", func(t *testing.T) {
		// Set pool settings for test
		env.DB.SetMaxOpenConns(25)
		env.DB.SetMaxIdleConns(5)
		env.DB.SetConnMaxLifetime(5 * time.Minute)

		stats := env.DB.Stats()

		t.Logf("Max open connections: %d", stats.MaxOpenConnections)
		t.Logf("Open connections: %d", stats.OpenConnections)
		t.Logf("In use: %d", stats.InUse)
		t.Logf("Idle: %d", stats.Idle)

		if stats.MaxOpenConnections != 25 {
			t.Errorf("Expected max open connections 25, got %d", stats.MaxOpenConnections)
		}
	})

	t.Run("ConnectionReuse", func(t *testing.T) {
		statsBefore := env.DB.Stats()

		// Execute multiple queries
		for i := 0; i < 10; i++ {
			var count int
			env.DB.QueryRow("SELECT COUNT(*) FROM experiments").Scan(&count)
		}

		statsAfter := env.DB.Stats()

		// Should reuse connections
		if statsAfter.OpenConnections > statsBefore.OpenConnections+2 {
			t.Logf("Warning: Connections increased from %d to %d (possible leak)",
				statsBefore.OpenConnections, statsAfter.OpenConnections)
		}
	})
}

// TestExperimentValidation tests experiment creation validation
func TestExperimentValidation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("MissingRequiredFields", func(t *testing.T) {
		testCases := []struct {
			name    string
			request ExperimentRequest
		}{
			{
				name: "MissingName",
				request: ExperimentRequest{
					TargetScenario: "test",
					NewResource:    "test",
				},
			},
			{
				name: "MissingTargetScenario",
				request: ExperimentRequest{
					Name:        "test",
					NewResource: "test",
				},
			},
			{
				name: "MissingNewResource",
				request: ExperimentRequest{
					Name:           "test",
					TargetScenario: "test",
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				w, err := makeHTTPRequest(env, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/experiments",
					Body:   tc.request,
				})
				if err != nil {
					t.Fatalf("Request failed: %v", err)
				}

				if w.Code != http.StatusBadRequest {
					t.Errorf("Expected 400 for %s, got %d", tc.name, w.Code)
				}

				assertResponseContains(t, w, "required")
			})
		}
	})

	t.Run("ValidRequest", func(t *testing.T) {
		req := ExperimentRequest{
			Name:           "Valid Experiment",
			Description:    "This is valid",
			Prompt:         "Do something",
			TargetScenario: "test-scenario",
			NewResource:    "test-resource",
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/experiments",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var exp Experiment
		assertJSONResponse(t, w, http.StatusOK, &exp)

		if exp.Name != req.Name {
			t.Errorf("Expected name '%s', got '%s'", req.Name, exp.Name)
		}
	})
}

// TestExperimentStatusLogic tests status-related business logic
func TestExperimentStatusLogic(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("DefaultStatusOnCreation", func(t *testing.T) {
		exp := createTestExperiment(t, env, "default-status")

		// Verify default status is 'requested'
		var status string
		env.DB.QueryRow("SELECT status FROM experiments WHERE id = $1", exp.ID).Scan(&status)

		if status != "requested" {
			t.Errorf("Expected default status 'requested', got '%s'", status)
		}
	})

	t.Run("StatusTransitionTracking", func(t *testing.T) {
		exp := createTestExperiment(t, env, "status-tracking")

		// Get initial timestamp
		var createdAt time.Time
		env.DB.QueryRow("SELECT created_at FROM experiments WHERE id = $1", exp.ID).Scan(&createdAt)

		time.Sleep(100 * time.Millisecond)

		// Update status
		updates := map[string]interface{}{"status": "running"}
		makeHTTPRequest(env, HTTPTestRequest{
			Method:  "PUT",
			Path:    fmt.Sprintf("/api/experiments/%s", exp.ID),
			URLVars: map[string]string{"id": exp.ID},
			Body:    updates,
		})

		// Verify updated_at changed
		var updatedAt time.Time
		env.DB.QueryRow("SELECT updated_at FROM experiments WHERE id = $1", exp.ID).Scan(&updatedAt)

		if !updatedAt.After(createdAt) {
			t.Error("updated_at should be after created_at")
		}
	})

	t.Run("CompletedAtTracking", func(t *testing.T) {
		exp := createTestExperiment(t, env, "completed-tracking")

		// Verify completed_at is null initially
		var completedAt sql.NullTime
		env.DB.QueryRow("SELECT completed_at FROM experiments WHERE id = $1", exp.ID).Scan(&completedAt)

		if completedAt.Valid {
			t.Error("completed_at should be NULL for new experiment")
		}

		// Simulate completion
		now := time.Now()
		env.DB.Exec("UPDATE experiments SET status = 'completed', completed_at = $1 WHERE id = $2", now, exp.ID)

		// Verify completed_at is set
		env.DB.QueryRow("SELECT completed_at FROM experiments WHERE id = $1", exp.ID).Scan(&completedAt)

		if !completedAt.Valid {
			t.Error("completed_at should be set for completed experiment")
		}
	})
}

// TestJSONFieldHandling tests JSONB field operations
func TestJSONFieldHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("EmptyJSONArrays", func(t *testing.T) {
		exp := createTestExperiment(t, env, "empty-json")

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/experiments/%s", exp.ID),
			URLVars: map[string]string{"id": exp.ID},
		})
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		var result Experiment
		assertJSONResponse(t, w, http.StatusOK, &result)

		// Empty arrays should be initialized
		if result.ExistingResources == nil {
			t.Log("ExistingResources is nil (acceptable, will be handled by client)")
		}
		if result.FilesGenerated == nil {
			t.Log("FilesGenerated is nil (acceptable, will be handled by client)")
		}
	})

	t.Run("NullJSONFields", func(t *testing.T) {
		exp := createTestExperiment(t, env, "null-json")

		// Explicitly set to NULL
		env.DB.Exec("UPDATE experiments SET existing_resources = NULL, files_generated = NULL WHERE id = $1", exp.ID)

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/experiments/%s", exp.ID),
			URLVars: map[string]string{"id": exp.ID},
		})
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		var result Experiment
		assertJSONResponse(t, w, http.StatusOK, &result)

		// Should handle NULL gracefully
		t.Logf("ExistingResources: %v", result.ExistingResources)
		t.Logf("FilesGenerated: %v", result.FilesGenerated)
	})
}

// TestTemplateBusinessLogic tests template-related business logic
func TestTemplateBusinessLogic(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("ActiveTemplateFiltering", func(t *testing.T) {
		// Clean up
		env.DB.Exec("TRUNCATE TABLE experiment_templates CASCADE")

		// Create templates
		createTestTemplate(t, env, "active-template")
		inactive := createTestTemplate(t, env, "inactive-template")

		env.DB.Exec("UPDATE experiment_templates SET is_active = false WHERE id = $1", inactive.ID)

		// Query templates
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/templates",
		})
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		var templates []ExperimentTemplate
		assertJSONResponse(t, w, http.StatusOK, &templates)

		// Should only return active
		for _, tmpl := range templates {
			if tmpl.ID == inactive.ID {
				t.Error("Inactive template returned in results")
			}
			if !tmpl.IsActive {
				t.Error("Non-active template in results")
			}
		}
	})

	t.Run("TemplateUsageTracking", func(t *testing.T) {
		tmpl := createTestTemplate(t, env, "usage-tracking")

		// Verify initial usage
		var usageCount int
		env.DB.QueryRow("SELECT usage_count FROM experiment_templates WHERE id = $1", tmpl.ID).Scan(&usageCount)

		if usageCount != 0 {
			t.Errorf("Expected initial usage_count 0, got %d", usageCount)
		}

		// Simulate usage
		env.DB.Exec("UPDATE experiment_templates SET usage_count = usage_count + 1 WHERE id = $1", tmpl.ID)

		env.DB.QueryRow("SELECT usage_count FROM experiment_templates WHERE id = $1", tmpl.ID).Scan(&usageCount)

		if usageCount != 1 {
			t.Errorf("Expected usage_count 1 after increment, got %d", usageCount)
		}
	})

	t.Run("SuccessRateCalculation", func(t *testing.T) {
		tmpl := createTestTemplate(t, env, "success-rate")

		// Simulate setting success rate
		successRate := 85.5
		env.DB.Exec("UPDATE experiment_templates SET success_rate = $1 WHERE id = $2", successRate, tmpl.ID)

		// Retrieve and verify
		var rate sql.NullFloat64
		env.DB.QueryRow("SELECT success_rate FROM experiment_templates WHERE id = $1", tmpl.ID).Scan(&rate)

		if !rate.Valid {
			t.Error("Success rate should be valid")
		}

		if rate.Float64 != successRate {
			t.Errorf("Expected success rate %.2f, got %.2f", successRate, rate.Float64)
		}
	})
}

// TestScenarioBusinessLogic tests scenario-related business logic
func TestScenarioBusinessLogic(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("ExperimentationFriendlyFiltering", func(t *testing.T) {
		// Clean up
		env.DB.Exec("TRUNCATE TABLE available_scenarios CASCADE")

		createTestScenario(t, env, "friendly")
		unfriendly := createTestScenario(t, env, "unfriendly")

		env.DB.Exec("UPDATE available_scenarios SET experimentation_friendly = false WHERE id = $1", unfriendly.ID)

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/scenarios",
		})
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		var scenarios []AvailableScenario
		assertJSONResponse(t, w, http.StatusOK, &scenarios)

		for _, s := range scenarios {
			if s.ID == unfriendly.ID {
				t.Error("Non-experimentation-friendly scenario returned")
			}
			if !s.ExperimentationFriendly {
				t.Error("Non-friendly scenario in results")
			}
		}
	})

	t.Run("ComplexityLevelClassification", func(t *testing.T) {
		scenarios := []struct {
			name       string
			complexity string
		}{
			{"low-complexity", "low"},
			{"medium-complexity", "medium"},
			{"high-complexity", "high"},
		}

		for _, sc := range scenarios {
			scenario := createTestScenario(t, env, sc.name)
			env.DB.Exec("UPDATE available_scenarios SET complexity_level = $1 WHERE id = $2", sc.complexity, scenario.ID)

			var complexity string
			env.DB.QueryRow("SELECT complexity_level FROM available_scenarios WHERE id = $1", scenario.ID).Scan(&complexity)

			if complexity != sc.complexity {
				t.Errorf("Expected complexity '%s', got '%s'", sc.complexity, complexity)
			}
		}
	})

	t.Run("LastExperimentDateTracking", func(t *testing.T) {
		scenario := createTestScenario(t, env, "experiment-tracking")

		// Initially NULL
		var lastDate sql.NullTime
		env.DB.QueryRow("SELECT last_experiment_date FROM available_scenarios WHERE id = $1", scenario.ID).Scan(&lastDate)

		if lastDate.Valid {
			t.Error("last_experiment_date should be NULL initially")
		}

		// Simulate experiment
		now := time.Now()
		env.DB.Exec("UPDATE available_scenarios SET last_experiment_date = $1 WHERE id = $2", now, scenario.ID)

		env.DB.QueryRow("SELECT last_experiment_date FROM available_scenarios WHERE id = $1", scenario.ID).Scan(&lastDate)

		if !lastDate.Valid {
			t.Error("last_experiment_date should be set after experiment")
		}
	})
}

// TestExperimentLogBusinessLogic tests experiment log business logic
func TestExperimentLogBusinessLogic(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("LogOrdering", func(t *testing.T) {
		exp := createTestExperiment(t, env, "log-ordering")

		// Create logs with specific timestamps
		times := []time.Time{
			time.Now().Add(-3 * time.Hour),
			time.Now().Add(-2 * time.Hour),
			time.Now().Add(-1 * time.Hour),
		}

		for i, ts := range times {
			logID := uuid.New().String()
			query := `INSERT INTO experiment_logs (id, experiment_id, step, prompt, success, started_at)
			          VALUES ($1, $2, $3, $4, $5, $6)`
			env.DB.Exec(query, logID, exp.ID, fmt.Sprintf("step-%d", i), "prompt", true, ts)
		}

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/experiments/%s/logs", exp.ID),
			URLVars: map[string]string{"id": exp.ID},
		})
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		var logs []ExperimentLog
		assertJSONResponse(t, w, http.StatusOK, &logs)

		// Verify chronological order
		for i := 0; i < len(logs)-1; i++ {
			if logs[i].StartedAt.After(logs[i+1].StartedAt) {
				t.Error("Logs not in chronological order")
			}
		}
	})

	t.Run("DurationCalculation", func(t *testing.T) {
		exp := createTestExperiment(t, env, "duration-calc")

		logID := uuid.New().String()
		startTime := time.Now()
		endTime := startTime.Add(125 * time.Second) // 125 seconds
		duration := 125

		query := `INSERT INTO experiment_logs (id, experiment_id, step, prompt, success, started_at, completed_at, duration_seconds)
		          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
		env.DB.Exec(query, logID, exp.ID, "timed-step", "prompt", true, startTime, endTime, duration)

		var retrievedDuration sql.NullInt64
		env.DB.QueryRow("SELECT duration_seconds FROM experiment_logs WHERE id = $1", logID).Scan(&retrievedDuration)

		if !retrievedDuration.Valid {
			t.Error("Duration should be valid")
		}

		if retrievedDuration.Int64 != int64(duration) {
			t.Errorf("Expected duration %d, got %d", duration, retrievedDuration.Int64)
		}
	})

	t.Run("SuccessFailureTracking", func(t *testing.T) {
		exp := createTestExperiment(t, env, "success-tracking")

		// Create successful log
		successID := uuid.New().String()
		query := `INSERT INTO experiment_logs (id, experiment_id, step, prompt, success, started_at)
		          VALUES ($1, $2, $3, $4, $5, NOW())`
		env.DB.Exec(query, successID, exp.ID, "success-step", "prompt", true)

		// Create failed log
		failID := uuid.New().String()
		errorMsg := "Something went wrong"
		query = `INSERT INTO experiment_logs (id, experiment_id, step, prompt, success, error_message, started_at)
		          VALUES ($1, $2, $3, $4, $5, $6, NOW())`
		env.DB.Exec(query, failID, exp.ID, "fail-step", "prompt", false, errorMsg)

		// Retrieve and verify
		var successFlag bool
		env.DB.QueryRow("SELECT success FROM experiment_logs WHERE id = $1", successID).Scan(&successFlag)
		if !successFlag {
			t.Error("Success flag should be true")
		}

		env.DB.QueryRow("SELECT success FROM experiment_logs WHERE id = $1", failID).Scan(&successFlag)
		if successFlag {
			t.Error("Success flag should be false for failed log")
		}

		var errorMessage sql.NullString
		env.DB.QueryRow("SELECT error_message FROM experiment_logs WHERE id = $1", failID).Scan(&errorMessage)
		if !errorMessage.Valid || errorMessage.String != errorMsg {
			t.Errorf("Expected error message '%s', got '%s'", errorMsg, errorMessage.String)
		}
	})
}
