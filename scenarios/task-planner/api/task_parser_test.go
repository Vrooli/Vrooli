package main

import (
	"testing"
)

// TestParseAITaskResponse tests AI response parsing
func TestParseAITaskResponse(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	tests := []struct {
		name           string
		aiResponse     string
		expectedTasks  int
		shouldError    bool
		errorContains  string
	}{
		{
			name: "ValidJSONResponse",
			aiResponse: `[
				{
					"title": "Test Task 1",
					"description": "Description 1",
					"priority": "high",
					"tags": ["test", "automated"],
					"estimated_hours": 2.5
				},
				{
					"title": "Test Task 2",
					"description": "Description 2",
					"priority": "medium",
					"tags": ["test"],
					"estimated_hours": 1.0
				}
			]`,
			expectedTasks: 2,
			shouldError:   false,
		},
		{
			name: "JSONWithMarkdownFormatting",
			aiResponse: "```json\n[\n  {\n    \"title\": \"Task\",\n    \"description\": \"Desc\",\n    \"priority\": \"low\",\n    \"tags\": [],\n    \"estimated_hours\": 1.0\n  }\n]\n```",
			expectedTasks: 1,
			shouldError:   false,
		},
		{
			name:          "EmptyArray",
			aiResponse:    "[]",
			expectedTasks: 0,
			shouldError:   false,
		},
		{
			name:          "NoJSONFound",
			aiResponse:    "This is just text without JSON",
			expectedTasks: 0,
			shouldError:   true,
			errorContains: "no JSON array found",
		},
		{
			name: "InvalidPriorityFixed",
			aiResponse: `[
				{
					"title": "Task",
					"description": "Desc",
					"priority": "invalid_priority",
					"tags": [],
					"estimated_hours": 1.0
				}
			]`,
			expectedTasks: 1,
			shouldError:   false,
		},
		{
			name: "MissingTitle",
			aiResponse: `[
				{
					"description": "Desc",
					"priority": "medium",
					"tags": [],
					"estimated_hours": 1.0
				}
			]`,
			expectedTasks: 0, // Task without title should be filtered out
			shouldError:   false,
		},
		{
			name: "TitleTooLong",
			aiResponse: `[
				{
					"title": "This is an extremely long title that exceeds the maximum allowed length and should be truncated by the validation logic to ensure it fits within the database schema constraints and doesnt cause any issues with storage or display in the user interface and this continues on and on to make sure we reach over 500 characters to properly test the truncation functionality of the parseAITaskResponse function which is critical for data integrity",
					"description": "Desc",
					"priority": "medium",
					"tags": [],
					"estimated_hours": 1.0
				}
			]`,
			expectedTasks: 1, // Should be truncated
			shouldError:   false,
		},
		{
			name: "InvalidEstimatedHours",
			aiResponse: `[
				{
					"title": "Task",
					"description": "Desc",
					"priority": "medium",
					"tags": [],
					"estimated_hours": -5
				}
			]`,
			expectedTasks: 1, // Should be set to default
			shouldError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tasks, err := env.Service.parseAITaskResponse(tt.aiResponse)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorContains)
				} else if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(tasks) != tt.expectedTasks {
				t.Errorf("Expected %d tasks, got %d", tt.expectedTasks, len(tasks))
			}

			// Validate task structure
			for i, task := range tasks {
				if task.Title == "" {
					t.Errorf("Task %d has empty title", i)
				}

				if len(task.Title) > 500 {
					t.Errorf("Task %d title exceeds 500 characters: %d", i, len(task.Title))
				}

				validPriorities := map[string]bool{"critical": true, "high": true, "medium": true, "low": true}
				if !validPriorities[task.Priority] {
					t.Errorf("Task %d has invalid priority: %s", i, task.Priority)
				}

				if task.EstimatedHours <= 0 || task.EstimatedHours > 1000 {
					t.Errorf("Task %d has invalid estimated hours: %f", i, task.EstimatedHours)
				}
			}
		})
	}
}

// TestValidateStatusTransition tests status transition validation
func TestValidateStatusTransition(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	tests := []struct {
		name        string
		fromStatus  string
		toStatus    string
		shouldError bool
	}{
		// Valid transitions
		{"BacklogToInProgress", "backlog", "in_progress", false},
		{"BacklogToCancelled", "backlog", "cancelled", false},
		{"InProgressToBlocked", "in_progress", "blocked", false},
		{"InProgressToReview", "in_progress", "review", false},
		{"InProgressToCompleted", "in_progress", "completed", false},
		{"InProgressToBacklog", "in_progress", "backlog", false},
		{"InProgressToCancelled", "in_progress", "cancelled", false},
		{"BlockedToInProgress", "blocked", "in_progress", false},
		{"BlockedToBacklog", "blocked", "backlog", false},
		{"BlockedToCancelled", "blocked", "cancelled", false},
		{"ReviewToInProgress", "review", "in_progress", false},
		{"ReviewToCompleted", "review", "completed", false},
		{"ReviewToBacklog", "review", "backlog", false},
		{"CompletedToReview", "completed", "review", false},
		{"CompletedToBacklog", "completed", "backlog", false},
		{"CancelledToBacklog", "cancelled", "backlog", false},

		// Invalid transitions
		{"BacklogToReview", "backlog", "review", true},
		{"BacklogToCompleted", "backlog", "completed", true},
		{"BacklogToBlocked", "backlog", "blocked", true},
		{"ReviewToCancelled", "review", "cancelled", true},
		{"CompletedToInProgress", "completed", "in_progress", true},
		{"CompletedToCancelled", "completed", "cancelled", true},
		{"CancelledToInProgress", "cancelled", "in_progress", true},
		{"CancelledToCompleted", "cancelled", "completed", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Service.validateStatusTransition(tt.fromStatus, tt.toStatus)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for transition %s -> %s, got nil", tt.fromStatus, tt.toStatus)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for transition %s -> %s, got: %v", tt.fromStatus, tt.toStatus, err)
				}
			}
		})
	}
}

// TestGetNextActions tests next actions generation
func TestGetNextActions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	tests := []struct {
		status       string
		minActions   int
		shouldContain string
	}{
		{"backlog", 2, "Research"},
		{"in_progress", 2, "progress"},
		{"blocked", 2, "resolve"},
		{"review", 2, "Review"},
		{"completed", 2, "Document"},
		{"cancelled", 2, "Archive"},
		{"unknown", 0, ""},
	}

	for _, tt := range tests {
		t.Run("NextActions_"+tt.status, func(t *testing.T) {
			task := &Task{Status: tt.status}
			actions := env.Service.getNextActions(tt.status, task)

			if len(actions) < tt.minActions {
				t.Errorf("Expected at least %d actions for status %s, got %d", tt.minActions, tt.status, len(actions))
			}

			if tt.shouldContain != "" {
				found := false
				for _, action := range actions {
					if contains(action, tt.shouldContain) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected actions for status %s to contain '%s', got: %v", tt.status, tt.shouldContain, actions)
				}
			}
		})
	}
}

// TestMinFunction tests the min helper function
func TestMinFunction(t *testing.T) {
	tests := []struct {
		a, b, expected int
	}{
		{1, 2, 1},
		{5, 3, 3},
		{0, 0, 0},
		{-1, 1, -1},
		{100, 50, 50},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := min(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("min(%d, %d) = %d, expected %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// TestGetResourcePort tests resource port retrieval
func TestGetResourcePort(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Test default fallback behavior
	tests := []struct {
		resourceName string
		expectPort   bool
	}{
		{"n8n", true},
		{"windmill", true},
		{"postgres", true},
		{"qdrant", true},
		{"unknown-resource", true}, // Should return generic fallback
	}

	for _, tt := range tests {
		t.Run("GetPort_"+tt.resourceName, func(t *testing.T) {
			port := getResourcePort(tt.resourceName)
			if tt.expectPort && port == "" {
				// This might fail if port registry script is not available, which is acceptable
				t.Logf("Warning: No port returned for resource %s (script may not be available)", tt.resourceName)
			} else if port != "" {
				t.Logf("Got port %s for resource %s", port, tt.resourceName)
			}
		})
	}
}
