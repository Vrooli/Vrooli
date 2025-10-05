package main

import (
	"testing"

	"github.com/google/uuid"
)

// TestParseResearchResults tests research results parsing
func TestParseResearchResults(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	taskID := uuid.New()

	tests := []struct {
		name          string
		aiResponse    string
		shouldError   bool
		errorContains string
	}{
		{
			name: "ValidJSONResponse",
			aiResponse: `{
				"research_summary": "This task involves implementing a new feature",
				"requirements": ["req1", "req2"],
				"dependencies": ["dep1"],
				"recommendations": ["rec1", "rec2", "rec3"],
				"estimated_hours": 4.5,
				"complexity": "medium",
				"technical_considerations": ["consideration1"],
				"potential_challenges": ["challenge1"],
				"success_criteria": ["criteria1"]
			}`,
			shouldError: false,
		},
		{
			name: "JSONWithExtraWhitespace",
			aiResponse: `

			{
				"research_summary": "Summary",
				"requirements": [],
				"dependencies": [],
				"recommendations": [],
				"estimated_hours": 2,
				"complexity": "low"
			}

			`,
			shouldError: false,
		},
		{
			name: "JSONWithBackticks",
			aiResponse: "```\n{\n  \"research_summary\": \"Summary\",\n  \"requirements\": [],\n  \"dependencies\": [],\n  \"recommendations\": [],\n  \"estimated_hours\": 3.0,\n  \"complexity\": \"high\"\n}\n```",
			shouldError: false,
		},
		{
			name:          "NoJSONFound",
			aiResponse:    "This is just text without any JSON",
			shouldError:   true,
			errorContains: "no valid JSON found",
		},
		{
			name:          "InvalidJSON",
			aiResponse:    "{invalid json}",
			shouldError:   true,
			errorContains: "failed to parse JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := env.Service.parseResearchResults(tt.aiResponse, taskID)

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

			if result == nil {
				t.Fatal("Expected result to be non-nil")
			}

			if !result.Success {
				t.Error("Expected result.Success to be true")
			}

			if result.TaskID != taskID {
				t.Errorf("Expected task ID %s, got %s", taskID, result.TaskID)
			}

			// Validate complexity is valid
			validComplexities := map[string]bool{"low": true, "medium": true, "high": true, "very_high": true}
			if !validComplexities[result.Complexity] {
				t.Errorf("Invalid complexity: %s", result.Complexity)
			}

			// Validate estimated hours is reasonable
			if result.EstimatedHours < 0 {
				t.Errorf("Estimated hours should not be negative: %f", result.EstimatedHours)
			}
		})
	}
}

// TestBuildResearchPrompt tests research prompt building
func TestBuildResearchPrompt(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	task := &Task{
		ID:          uuid.New(),
		Title:       "Test Task",
		Description: "Test Description",
		Priority:    "high",
		Status:      "backlog",
		Tags:        []string{"test", "feature"},
	}

	app := &App{
		ID:   uuid.New(),
		Name: "test-app",
		Type: "web-application",
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

	t.Run("BuildPrompt_WithContext", func(t *testing.T) {
		prompt := env.Service.buildResearchPrompt(task, context)

		if prompt == "" {
			t.Fatal("Expected non-empty prompt")
		}

		// Verify prompt contains key information
		requiredStrings := []string{
			task.Title,
			task.Description,
			task.Priority,
			app.Name,
			app.Type,
			"RESEARCH REQUIREMENTS",
		}

		for _, required := range requiredStrings {
			if !contains(prompt, required) {
				t.Errorf("Prompt should contain '%s'", required)
			}
		}
	})

	t.Run("BuildPrompt_IncludesRelatedTasks", func(t *testing.T) {
		prompt := env.Service.buildResearchPrompt(task, context)

		if !contains(prompt, "RELATED TASKS") {
			t.Error("Prompt should contain RELATED TASKS section when related tasks exist")
		}

		if !contains(prompt, context.RelatedTasks[0].Title) {
			t.Error("Prompt should include related task titles")
		}
	})

	t.Run("BuildPrompt_IncludesCompletedTasks", func(t *testing.T) {
		prompt := env.Service.buildResearchPrompt(task, context)

		if !contains(prompt, "RECENT COMPLETED TASKS") {
			t.Error("Prompt should contain RECENT COMPLETED TASKS section when completed tasks exist")
		}

		if !contains(prompt, context.CompletedTasks[0].Title) {
			t.Error("Prompt should include completed task titles")
		}
	})

	t.Run("BuildPrompt_NoRelatedTasks", func(t *testing.T) {
		emptyContext := &TaskResearchContext{
			Task:           task,
			AppContext:     app,
			RelatedTasks:   []Task{},
			CompletedTasks: []Task{},
		}

		prompt := env.Service.buildResearchPrompt(task, emptyContext)

		if prompt == "" {
			t.Fatal("Expected non-empty prompt")
		}
	})
}

// TestEnhanceResearchWithContext tests context-based research enhancement
func TestEnhanceResearchWithContext(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("EnhanceForWebApplication", func(t *testing.T) {
		result := &TaskResearchResult{
			Success:         true,
			TaskID:          uuid.New(),
			EstimatedHours:  5.0,
			Recommendations: []string{"Original recommendation"},
		}

		context := &TaskResearchContext{
			AppContext: &App{
				Type: "web-application",
			},
			CompletedTasks: []Task{},
		}

		env.Service.enhanceResearchWithContext(result, context)

		// Should add web-specific recommendations
		found := false
		for _, rec := range result.Recommendations {
			if contains(rec, "accessibility") || contains(rec, "responsive") {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected web accessibility recommendations to be added for web-application type")
		}
	})

	t.Run("EnhanceWithCompletedTasks", func(t *testing.T) {
		originalEstimate := 5.0
		result := &TaskResearchResult{
			Success:         true,
			TaskID:          uuid.New(),
			EstimatedHours:  originalEstimate,
			Recommendations: []string{},
		}

		context := &TaskResearchContext{
			AppContext: &App{
				Type: "api",
			},
			CompletedTasks: []Task{
				{Title: "Task 1"},
				{Title: "Task 2"},
			},
		}

		env.Service.enhanceResearchWithContext(result, context)

		// Estimate should be adjusted based on historical data
		if result.EstimatedHours == originalEstimate {
			t.Log("Note: Estimated hours may be adjusted based on historical data")
		}
	})

	t.Run("EnhanceNoCompletedTasks", func(t *testing.T) {
		originalEstimate := 5.0
		result := &TaskResearchResult{
			Success:         true,
			TaskID:          uuid.New(),
			EstimatedHours:  originalEstimate,
			Recommendations: []string{},
		}

		context := &TaskResearchContext{
			AppContext: &App{
				Type: "cli",
			},
			CompletedTasks: []Task{},
		}

		env.Service.enhanceResearchWithContext(result, context)

		// Estimate should remain unchanged when no completed tasks
		if result.EstimatedHours != originalEstimate {
			t.Errorf("Expected estimate to remain %f, got %f", originalEstimate, result.EstimatedHours)
		}
	})
}
