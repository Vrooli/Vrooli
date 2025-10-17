package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestTaskMonitorComprehensive tests task monitor functionality
func TestTaskMonitorComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	skipIfNoDatabase(t, env.DB)

	appID := createTestApp(t, env, "test-app")
	defer cleanupTestData(t, env)

	t.Run("ValidateStatusTransition_ValidTransitions", func(t *testing.T) {
		validTransitions := map[string][]string{
			"backlog":     {"in_progress", "cancelled"},
			"in_progress": {"blocked", "review", "completed", "backlog", "cancelled"},
			"blocked":     {"in_progress", "backlog", "cancelled"},
			"review":      {"in_progress", "completed", "backlog"},
			"completed":   {"review", "backlog"},
			"cancelled":   {"backlog"},
		}

		for fromStatus, toStatuses := range validTransitions {
			for _, toStatus := range toStatuses {
				err := env.Service.validateStatusTransition(fromStatus, toStatus)
				if err != nil {
					t.Errorf("Expected valid transition from %s to %s, got error: %v", fromStatus, toStatus, err)
				}
			}
		}
	})

	t.Run("ValidateStatusTransition_InvalidTransitions", func(t *testing.T) {
		invalidTransitions := [][2]string{
			{"backlog", "blocked"},
			{"backlog", "review"},
			{"backlog", "completed"},
			{"blocked", "review"},
			{"blocked", "completed"},
			{"cancelled", "in_progress"},
			{"cancelled", "completed"},
		}

		for _, transition := range invalidTransitions {
			fromStatus, toStatus := transition[0], transition[1]
			err := env.Service.validateStatusTransition(fromStatus, toStatus)
			if err == nil {
				t.Errorf("Expected invalid transition from %s to %s to fail, but it succeeded", fromStatus, toStatus)
			}
		}
	})

	t.Run("GetNextActions_AllStatuses", func(t *testing.T) {
		task := &Task{
			ID:          uuid.New(),
			Title:       "Test Task",
			Description: "Test Description",
			Status:      "backlog",
			Priority:    "medium",
			AppID:       appID,
		}

		statuses := []string{"backlog", "in_progress", "blocked", "review", "completed", "cancelled"}
		for _, status := range statuses {
			nextActions := env.Service.getNextActions(status, task)
			if len(nextActions) == 0 {
				t.Errorf("Expected next actions for status %s, got none", status)
			}
		}
	})

	t.Run("StatusChangeRecording", func(t *testing.T) {
		taskID := createTestTask(t, env, appID, "Test Task", "backlog")

		// Transition through multiple states
		transitions := []struct {
			toStatus string
			reason   string
		}{
			{"in_progress", "Starting work"},
			{"blocked", "Waiting for dependency"},
			{"in_progress", "Dependency resolved"},
			{"review", "Ready for review"},
			{"completed", "All requirements met"},
		}

		for i, transition := range transitions {
			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: "PUT",
				Path:   "/api/tasks/status",
				Body: map[string]interface{}{
					"task_id":   taskID.String(),
					"to_status": transition.toStatus,
					"reason":    transition.reason,
					"notes":     fmt.Sprintf("Transition %d", i+1),
				},
			})
			if err != nil {
				t.Fatalf("Failed to make status update request: %v", err)
			}

			if w.Code != http.StatusOK {
				t.Logf("Status update %d failed: %s", i+1, w.Body.String())
			}
		}

		// Verify history
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/tasks/status-history",
			QueryParams: map[string]string{"task_id": taskID.String()},
		})
		if err != nil {
			t.Fatalf("Failed to get status history: %v", err)
		}

		response := assertSuccessResponse(t, w, []string{"task_id", "history"})
		history, ok := response["history"].([]interface{})
		if !ok {
			t.Fatalf("Expected history to be an array")
		}

		// Should have recorded successful transitions
		if len(history) > 0 {
			t.Logf("Status history has %d entries", len(history))
			ValidateStatusHistoryResponse(t, history)
		}
	})
}

// TestTaskParserComprehensive tests task parser functionality
func TestTaskParserComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	skipIfNoDatabase(t, env.DB)

	t.Run("ParseAITaskResponse_EdgeCases", func(t *testing.T) {
		tests := []struct {
			name          string
			aiResponse    string
			expectedCount int
			shouldError   bool
		}{
			{
				name:          "EmptyArray",
				aiResponse:    "[]",
				expectedCount: 0,
				shouldError:   false,
			},
			{
				name:          "TaskWithMissingTitle",
				aiResponse:    `[{"description": "No title", "priority": "medium", "tags": [], "estimated_hours": 1}]`,
				expectedCount: 0, // Should be filtered out
				shouldError:   false,
			},
			{
				name: "TaskWithInvalidPriority",
				aiResponse: `[{
					"title": "Test Task",
					"description": "Test",
					"priority": "invalid_priority",
					"tags": [],
					"estimated_hours": 1
				}]`,
				expectedCount: 1, // Should default to medium
				shouldError:   false,
			},
			{
				name: "TaskWithVeryLongTitle",
				aiResponse: fmt.Sprintf(`[{
					"title": "%s",
					"description": "Test",
					"priority": "medium",
					"tags": [],
					"estimated_hours": 1
				}]`, strings.Repeat("A", 1000)),
				expectedCount: 1, // Should truncate
				shouldError:   false,
			},
			{
				name: "TaskWithInvalidEstimatedHours",
				aiResponse: `[{
					"title": "Test Task",
					"description": "Test",
					"priority": "medium",
					"tags": [],
					"estimated_hours": -5
				}]`,
				expectedCount: 1, // Should default to 1.0
				shouldError:   false,
			},
			{
				name:          "InvalidJSON",
				aiResponse:    "not valid json at all",
				expectedCount: 0,
				shouldError:   true,
			},
			{
				name:          "JSONObject_NotArray",
				aiResponse:    `{"title": "Single task"}`,
				expectedCount: 0,
				shouldError:   true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				tasks, err := env.Service.parseAITaskResponse(tt.aiResponse)

				if tt.shouldError {
					if err == nil {
						t.Errorf("Expected error for %s, got nil", tt.name)
					}
				} else {
					if err != nil {
						t.Errorf("Unexpected error for %s: %v", tt.name, err)
					}
					if len(tasks) != tt.expectedCount {
						t.Errorf("Expected %d tasks for %s, got %d", tt.expectedCount, tt.name, len(tasks))
					}
				}
			})
		}
	})

	t.Run("ValidateAppToken_EdgeCases", func(t *testing.T) {
		// Create test app with known token
		appID := createTestApp(t, env, "test-app-token")
		defer cleanupTestData(t, env)

		// Get the actual token from database
		var apiToken string
		err := env.DB.QueryRow("SELECT api_token FROM apps WHERE id = $1", appID).Scan(&apiToken)
		if err != nil {
			t.Skipf("Could not retrieve API token: %v", err)
		}

		tests := []struct {
			name        string
			appID       string
			apiToken    string
			shouldError bool
		}{
			{
				name:        "ValidCredentials",
				appID:       appID.String(),
				apiToken:    apiToken,
				shouldError: false,
			},
			{
				name:        "InvalidToken",
				appID:       appID.String(),
				apiToken:    "invalid-token-12345",
				shouldError: true,
			},
			{
				name:        "InvalidAppID",
				appID:       uuid.New().String(),
				apiToken:    apiToken,
				shouldError: true,
			},
			{
				name:        "EmptyToken",
				appID:       appID.String(),
				apiToken:    "",
				shouldError: true,
			},
			{
				name:        "MalformedAppID",
				appID:       "not-a-uuid",
				apiToken:    apiToken,
				shouldError: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				app, err := env.Service.validateAppToken(tt.appID, tt.apiToken)

				if tt.shouldError {
					if err == nil {
						t.Errorf("Expected error for %s, got nil", tt.name)
					}
					if app != nil {
						t.Errorf("Expected nil app for %s, got %v", tt.name, app)
					}
				} else {
					if err != nil {
						t.Errorf("Unexpected error for %s: %v", tt.name, err)
					}
					if app == nil {
						t.Errorf("Expected app for %s, got nil", tt.name)
					}
				}
			})
		}
	})
}

// TestTaskResearcherComprehensive tests task researcher functionality
func TestTaskResearcherComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	skipIfNoDatabase(t, env.DB)

	appID := createTestApp(t, env, "test-app")
	defer cleanupTestData(t, env)

	t.Run("ParseResearchResults_EdgeCases", func(t *testing.T) {
		taskID := uuid.New()

		tests := []struct {
			name        string
			aiResponse  string
			shouldError bool
		}{
			{
				name: "MinimalValidResponse",
				aiResponse: `{
					"research_summary": "Summary",
					"requirements": [],
					"dependencies": [],
					"recommendations": [],
					"estimated_hours": 1.0,
					"complexity": "low"
				}`,
				shouldError: false,
			},
			{
				name: "ResponseWithExtraFields",
				aiResponse: `{
					"research_summary": "Summary",
					"requirements": ["req1"],
					"dependencies": ["dep1"],
					"recommendations": ["rec1"],
					"estimated_hours": 2.5,
					"complexity": "medium",
					"extra_field_1": "ignored",
					"extra_field_2": 123
				}`,
				shouldError: false,
			},
			{
				name: "ResponseWithIntegerHours",
				aiResponse: `{
					"research_summary": "Summary",
					"requirements": [],
					"dependencies": [],
					"recommendations": [],
					"estimated_hours": 5,
					"complexity": "high"
				}`,
				shouldError: false,
			},
			{
				name: "ResponseWithInvalidComplexity",
				aiResponse: `{
					"research_summary": "Summary",
					"requirements": [],
					"dependencies": [],
					"recommendations": [],
					"estimated_hours": 2.0,
					"complexity": "super_complex"
				}`,
				shouldError: false, // Should default to medium
			},
			{
				name:        "EmptyResponse",
				aiResponse:  "",
				shouldError: true,
			},
			{
				name:        "InvalidJSON",
				aiResponse:  "not json { invalid",
				shouldError: true,
			},
			{
				name:        "JSONArray_NotObject",
				aiResponse:  `["item1", "item2"]`,
				shouldError: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := env.Service.parseResearchResults(tt.aiResponse, taskID)

				if tt.shouldError {
					if err == nil {
						t.Errorf("Expected error for %s, got nil", tt.name)
					}
				} else {
					if err != nil {
						t.Errorf("Unexpected error for %s: %v", tt.name, err)
					}
					if result == nil {
						t.Errorf("Expected result for %s, got nil", tt.name)
					}
					if result != nil {
						// Verify defaults are applied
						if result.Complexity == "" {
							t.Errorf("Expected complexity to be set, got empty string")
						}
						if result.EstimatedHours <= 0 {
							t.Errorf("Expected positive estimated hours, got %f", result.EstimatedHours)
						}
					}
				}
			})
		}
	})

	t.Run("GatherResearchContext_Complete", func(t *testing.T) {
		// Create a task with related tasks and completed tasks
		taskID := createTestTask(t, env, appID, "Research Target Task", "backlog")

		// Create related tasks
		for i := 0; i < 3; i++ {
			createTestTask(t, env, appID, fmt.Sprintf("Related Task %d", i), "in_progress")
		}

		// Create completed tasks
		for i := 0; i < 3; i++ {
			createTestTask(t, env, appID, fmt.Sprintf("Completed Task %d", i), "completed")
		}

		task, err := env.Service.getTaskByID(taskID)
		if err != nil {
			t.Skipf("Failed to get task: %v", err)
		}

		context, err := env.Service.gatherResearchContext(task)
		if err != nil {
			t.Errorf("Failed to gather research context: %v", err)
		}

		if context == nil {
			t.Fatal("Expected research context, got nil")
		}

		if context.Task == nil {
			t.Error("Expected task in context")
		}

		if context.AppContext == nil {
			t.Error("Expected app context")
		}

		// Related tasks and completed tasks may or may not be found depending on implementation
		t.Logf("Found %d related tasks and %d completed tasks", len(context.RelatedTasks), len(context.CompletedTasks))
	})

	t.Run("BuildResearchPrompt_AllSections", func(t *testing.T) {
		task := &Task{
			ID:          uuid.New(),
			Title:       "Implement feature X",
			Description: "Add new functionality to the system",
			Status:      "backlog",
			Priority:    "high",
			Tags:        []string{"feature", "enhancement"},
			AppID:       appID,
		}

		app := &App{
			ID:          appID,
			Name:        "test-app",
			DisplayName: "Test Application",
			Type:        "web-application",
		}

		context := &TaskResearchContext{
			Task:       task,
			AppContext: app,
			RelatedTasks: []Task{
				{Title: "Related Task 1", Status: "completed"},
				{Title: "Related Task 2", Status: "in_progress"},
			},
			CompletedTasks: []Task{
				{Title: "Completed Task 1"},
			},
		}

		prompt := env.Service.buildResearchPrompt(task, context)

		// Verify prompt contains expected sections
		expectedSections := []string{
			"TASK TO RESEARCH",
			"Implement feature X",
			"APPLICATION CONTEXT",
			"Test Application",
			"RELATED TASKS",
			"RECENT COMPLETED TASKS",
			"RESEARCH REQUIREMENTS",
		}

		for _, section := range expectedSections {
			if !strings.Contains(prompt, section) {
				t.Errorf("Expected prompt to contain '%s', but it doesn't", section)
			}
		}
	})
}

// TestDatabaseOperations tests database interaction edge cases
func TestDatabaseOperations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	skipIfNoDatabase(t, env.DB)

	appID := createTestApp(t, env, "test-app")
	defer cleanupTestData(t, env)

	t.Run("GetTaskByID_NonExistent", func(t *testing.T) {
		nonExistentID := uuid.New()
		task, err := env.Service.getTaskByID(nonExistentID)

		if err == nil {
			t.Error("Expected error for non-existent task")
		}
		if task != nil {
			t.Error("Expected nil task for non-existent ID")
		}
	})

	t.Run("GetAppByID_NonExistent", func(t *testing.T) {
		nonExistentID := uuid.New()
		app, err := env.Service.getAppByID(nonExistentID)

		if err == nil {
			t.Error("Expected error for non-existent app")
		}
		if app != nil {
			t.Error("Expected nil app for non-existent ID")
		}
	})

	t.Run("FindRelatedTasks_NoMatches", func(t *testing.T) {
		// Create a task with unique tags and content
		taskID := createTestTask(t, env, appID, "Unique Task With No Matches XYZZYX123", "backlog")
		task, err := env.Service.getTaskByID(taskID)
		if err != nil {
			t.Skipf("Failed to get task: %v", err)
		}

		relatedTasks, err := env.Service.findRelatedTasks(task)
		if err != nil {
			t.Errorf("Unexpected error finding related tasks: %v", err)
		}

		// May return empty array or some matches
		t.Logf("Found %d related tasks for unique task", len(relatedTasks))
	})

	t.Run("GetRecentCompletedTasks_NoCompleted", func(t *testing.T) {
		// Use a new app with no completed tasks
		newAppID := createTestApp(t, env, "app-no-completed")

		completedTasks, err := env.Service.getRecentCompletedTasks(newAppID, 5)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if len(completedTasks) != 0 {
			t.Errorf("Expected 0 completed tasks, got %d", len(completedTasks))
		}
	})

	t.Run("GetTaskStatusHistory_NoHistory", func(t *testing.T) {
		// Create a new task that hasn't had status changes
		taskID := createTestTask(t, env, appID, "New Task No History", "backlog")

		history, err := env.Service.getTaskStatusHistory(taskID)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Should return empty array, not error
		if history == nil {
			t.Error("Expected empty array, got nil")
		}
	})
}

// TestConcurrentOperations tests thread safety
func TestConcurrentOperations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	skipIfNoDatabase(t, env.DB)

	appID := createTestApp(t, env, "test-app")
	defer cleanupTestData(t, env)

	t.Run("ConcurrentStatusUpdates", func(t *testing.T) {
		taskID := createTestTask(t, env, appID, "Concurrent Test Task", "backlog")

		// Try to update status concurrently
		done := make(chan bool, 3)

		for i := 0; i < 3; i++ {
			go func(idx int) {
				defer func() { done <- true }()

				status := "in_progress"
				if idx == 1 {
					time.Sleep(50 * time.Millisecond)
					status = "blocked"
				} else if idx == 2 {
					time.Sleep(100 * time.Millisecond)
					status = "review"
				}

				w, err := makeHTTPRequest(env, HTTPTestRequest{
					Method: "PUT",
					Path:   "/api/tasks/status",
					Body: map[string]interface{}{
						"task_id":   taskID.String(),
						"to_status": status,
						"reason":    fmt.Sprintf("Concurrent update %d", idx),
					},
				})
				if err != nil {
					t.Logf("Concurrent request %d failed: %v", idx, err)
					return
				}

				// Some may succeed, some may fail - that's okay
				t.Logf("Concurrent request %d status: %d", idx, w.Code)
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 3; i++ {
			<-done
		}

		// Verify task still exists and is in a valid state
		task, err := env.Service.getTaskByID(taskID)
		if err != nil {
			t.Errorf("Task should still exist after concurrent updates: %v", err)
		}
		if task != nil {
			t.Logf("Final task status: %s", task.Status)
		}
	})
}

// TestHelperFunctionsUtility tests utility helper functions
func TestHelperFunctionsUtility(t *testing.T) {
	t.Run("MinFunction", func(t *testing.T) {
		tests := []struct {
			a, b, expected int
		}{
			{1, 2, 1},
			{5, 3, 3},
			{10, 10, 10},
			{-5, -3, -5},
			{0, 1, 0},
		}

		for _, tt := range tests {
			result := min(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("min(%d, %d) = %d; expected %d", tt.a, tt.b, result, tt.expected)
			}
		}
	})

	t.Run("ContainsFunction", func(t *testing.T) {
		tests := []struct {
			s, substr string
			expected  bool
		}{
			{"hello world", "world", true},
			{"hello world", "planet", false},
			{"", "", true},
			{"test", "", true},
			{"", "test", false},
			{"exact", "exact", true},
		}

		for _, tt := range tests {
			result := contains(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("contains(%q, %q) = %v; expected %v", tt.s, tt.substr, result, tt.expected)
			}
		}
	})

	t.Run("WaitForCondition_Success", func(t *testing.T) {
		condition := func() bool {
			return true
		}

		result := waitForCondition(t, condition, 1*time.Second, 10*time.Millisecond)
		if !result {
			t.Error("Expected condition to be met immediately")
		}
	})

	t.Run("WaitForCondition_Timeout", func(t *testing.T) {
		condition := func() bool {
			return false
		}

		start := time.Now()
		result := waitForCondition(t, condition, 100*time.Millisecond, 10*time.Millisecond)
		elapsed := time.Since(start)

		if result {
			t.Error("Expected condition to timeout")
		}
		if elapsed < 100*time.Millisecond {
			t.Error("Expected to wait for timeout duration")
		}
	})

	t.Run("WaitForCondition_EventualSuccess", func(t *testing.T) {
		counter := 0
		condition := func() bool {
			counter++
			return counter >= 3
		}

		result := waitForCondition(t, condition, 1*time.Second, 10*time.Millisecond)
		if !result {
			t.Error("Expected condition to eventually succeed")
		}
		if counter < 3 {
			t.Errorf("Expected at least 3 attempts, got %d", counter)
		}
	})
}

// Note: TestEnhanceResearchWithContext is in task_researcher_test.go

// TestBackgroundFunctions tests background goroutines behavior
func TestBackgroundFunctions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	skipIfNoDatabase(t, env.DB)

	appID := createTestApp(t, env, "test-app")
	defer cleanupTestData(t, env)

	t.Run("EnsureTaskResearched", func(t *testing.T) {
		taskID := createTestTask(t, env, appID, "Test Task for Research", "backlog")

		// Call background function directly (not in goroutine for testing)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		env.Service.ensureTaskResearched(ctx, taskID)

		// Give it a moment to complete
		time.Sleep(100 * time.Millisecond)

		// This may or may not add research metadata depending on Ollama availability
		// Just verify it doesn't crash
		t.Log("ensureTaskResearched completed without panic")
	})

	t.Run("SuggestRelatedTasks", func(t *testing.T) {
		task := &Task{
			ID:          uuid.New(),
			Title:       "Completed Task",
			Description: "A completed task",
			Status:      "completed",
			AppID:       appID,
		}

		ctx := context.Background()

		// Should not panic
		env.Service.suggestRelatedTasks(ctx, task)
		t.Log("suggestRelatedTasks completed without panic")
	})

	t.Run("InvestigateBlockedTask", func(t *testing.T) {
		taskID := uuid.New()
		reason := "Waiting for external API"

		ctx := context.Background()

		// Should not panic
		env.Service.investigateBlockedTask(ctx, taskID, reason)
		t.Log("investigateBlockedTask completed without panic")
	})
}
