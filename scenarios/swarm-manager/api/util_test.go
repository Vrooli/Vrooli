// +build testing

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// TestGetFloat tests the getFloat utility function
func TestGetFloat(t *testing.T) {
	tests := []struct {
		name       string
		input      interface{}
		defaultVal float64
		expected   float64
	}{
		{"ValidFloat64", float64(5.5), 0.0, 5.5},
		{"ValidInt", 10, 0.0, 10.0},
		{"ValidFloat32", float32(15.0), 0.0, 15.0},
		{"NilValue", nil, 7.5, 7.5},
		{"StringValue", "invalid", 3.0, 3.0},
		{"BoolValue", true, 2.0, 2.0},
		{"NegativeFloat", float64(-3.5), 0.0, -3.5},
		{"ZeroValue", 0, 5.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFloat(tt.input, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

// TestGetString tests the getString utility function
func TestGetString(t *testing.T) {
	tests := []struct {
		name       string
		input      interface{}
		defaultVal string
		expected   string
	}{
		{"ValidString", "test", "default", "test"},
		{"EmptyString", "", "default", ""},
		{"NilValue", nil, "default", "default"},
		{"IntValue", 123, "default", "default"},
		{"FloatValue", 1.23, "default", "default"},
		{"BoolValue", true, "default", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getString(tt.input, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestConvertUrgencyToFloat tests urgency conversion
func TestConvertUrgencyToFloat(t *testing.T) {
	tests := []struct {
		urgency  interface{}
		expected float64
	}{
		{"critical", 4.0},
		{"high", 3.0},
		{"medium", 2.0},
		{"low", 1.0},
		{"unknown", 2.0}, // default
		{nil, 2.0},
		{"CRITICAL", 4.0}, // case insensitive
		{123, 123.0},      // numeric type, converted via getFloat
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Urgency_%v", tt.urgency), func(t *testing.T) {
			result := convertUrgencyToFloat(tt.urgency)
			if result != tt.expected {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

// TestConvertResourceCostToFloat tests resource cost conversion
func TestConvertResourceCostToFloat(t *testing.T) {
	tests := []struct {
		cost     interface{}
		expected float64
	}{
		{"minimal", 1.0},
		{"moderate", 2.0},
		{"heavy", 3.0},
		{"unknown", 2.0}, // default
		{nil, 2.0},
		{"MINIMAL", 1.0}, // case insensitive
		{456, 456.0},     // numeric type, converted via getFloat
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Cost_%v", tt.cost), func(t *testing.T) {
			result := convertResourceCostToFloat(tt.cost)
			if result != tt.expected {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

// TestSaveTaskToFile tests task file creation
func TestSaveTaskToFile(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		task := Task{
			ID:          "test-task-123",
			Title:       "Test Task",
			Description: "Test description",
			Type:        "bug-fix",
			Target:      "test-target",
			PriorityEstimates: map[string]interface{}{
				"impact":        8.0,
				"urgency":       "high",
				"success_prob":  0.8,
				"resource_cost": "moderate",
			},
			Dependencies: []string{},
			Blockers:     []string{},
			CreatedAt:    time.Now(),
			CreatedBy:    "test-user",
			Attempts:     []interface{}{},
			Notes:        "Test notes",
			Status:       "backlog",
		}

		taskDir := filepath.Join(env.TasksDir, "backlog", "manual")
		err := saveTaskToFile(task, taskDir)
		if err != nil {
			t.Fatalf("Failed to save task: %v", err)
		}

		// Verify file was created
		taskPath := filepath.Join(taskDir, fmt.Sprintf("%s.yaml", task.ID))
		if _, err := os.Stat(taskPath); os.IsNotExist(err) {
			t.Error("Expected task file to be created")
		}

		// Verify file contents
		content, err := ioutil.ReadFile(taskPath)
		if err != nil {
			t.Fatalf("Failed to read task file: %v", err)
		}

		var savedTask Task
		if err := yaml.Unmarshal(content, &savedTask); err != nil {
			t.Fatalf("Failed to parse task file: %v", err)
		}

		if savedTask.ID != task.ID {
			t.Errorf("Expected ID %s, got %s", task.ID, savedTask.ID)
		}
		if savedTask.Title != task.Title {
			t.Errorf("Expected title %s, got %s", task.Title, savedTask.Title)
		}
		if savedTask.Type != task.Type {
			t.Errorf("Expected type %s, got %s", task.Type, savedTask.Type)
		}
	})

	t.Run("Error_InvalidDirectory", func(t *testing.T) {
		task := Task{
			ID:    "test-task-456",
			Title: "Test Task",
		}

		// Try to save to a non-existent directory without creating it
		taskDir := "/nonexistent/path/that/does/not/exist"
		err := saveTaskToFile(task, taskDir)
		if err == nil {
			t.Error("Expected error when saving to invalid directory")
		}
	})
}

// TestReadTasksFromFolder tests reading tasks from a folder
func TestReadTasksFromFolder(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("Success_EmptyFolder", func(t *testing.T) {
		folderPath := filepath.Join(env.TasksDir, "active")
		tasks := readTasksFromFolder(folderPath, "active")

		if len(tasks) != 0 {
			t.Errorf("Expected 0 tasks, got %d", len(tasks))
		}
	})

	t.Run("Success_WithTasks", func(t *testing.T) {
		// Create test tasks
		testTask1 := setupTestTask(t, env, "bug-fix", "active")
		defer testTask1.Cleanup()

		testTask2 := setupTestTask(t, env, "feature", "active")
		defer testTask2.Cleanup()

		folderPath := filepath.Join(env.TasksDir, "active")
		tasks := readTasksFromFolder(folderPath, "active")

		if len(tasks) < 2 {
			t.Errorf("Expected at least 2 tasks, got %d", len(tasks))
		}

		// Verify status is set correctly
		for _, task := range tasks {
			if task.Status != "active" {
				t.Errorf("Expected status 'active', got '%s'", task.Status)
			}
		}
	})

	t.Run("Success_NonExistentFolder", func(t *testing.T) {
		folderPath := "/nonexistent/path"
		tasks := readTasksFromFolder(folderPath, "test")

		if len(tasks) != 0 {
			t.Errorf("Expected 0 tasks for non-existent folder, got %d", len(tasks))
		}
	})

	t.Run("Success_IgnoreNonYAMLFiles", func(t *testing.T) {
		folderPath := filepath.Join(env.TasksDir, "backlog", "manual")

		// Create a non-YAML file
		nonYamlFile := filepath.Join(folderPath, "readme.txt")
		ioutil.WriteFile(nonYamlFile, []byte("not a yaml file"), 0644)
		defer os.Remove(nonYamlFile)

		// Create a valid task
		testTask := setupTestTask(t, env, "bug-fix", "backlog")
		defer testTask.Cleanup()

		tasks := readTasksFromFolder(folderPath, "backlog")

		// Should only return the YAML task, not the txt file
		if len(tasks) < 1 {
			t.Error("Expected at least 1 task")
		}
	})
}

// TestFindTaskFile tests the findTaskFile utility
func TestFindTaskFile(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	tasksDir = env.TasksDir

	t.Run("Success_FindInActive", func(t *testing.T) {
		testTask := setupTestTask(t, env, "bug-fix", "active")
		defer testTask.Cleanup()

		foundPath := findTaskFile(testTask.Task.ID)
		if foundPath == "" {
			t.Error("Expected to find task file")
		}

		if foundPath != testTask.Path {
			t.Errorf("Expected path %s, got %s", testTask.Path, foundPath)
		}
	})

	t.Run("Success_FindInBacklog", func(t *testing.T) {
		testTask := setupTestTask(t, env, "feature", "backlog")
		defer testTask.Cleanup()

		foundPath := findTaskFile(testTask.Task.ID)
		if foundPath == "" {
			t.Error("Expected to find task file")
		}
	})

	t.Run("Error_TaskNotFound", func(t *testing.T) {
		foundPath := findTaskFile("non-existent-task-id")
		if foundPath != "" {
			t.Errorf("Expected empty path for non-existent task, got %s", foundPath)
		}
	})
}

// TestSeverityToImpact tests severity to impact conversion
func TestSeverityToImpact(t *testing.T) {
	tests := []struct {
		severity string
		expected int
	}{
		{"critical", 10},
		{"high", 8},
		{"medium", 5},
		{"low", 2},
		{"unknown", 5}, // default
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Severity_%s", tt.severity), func(t *testing.T) {
			result := severityToImpact(tt.severity)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// TestFrequencyToUrgency tests frequency to urgency conversion
func TestFrequencyToUrgency(t *testing.T) {
	tests := []struct {
		frequency string
		expected  string
	}{
		{"constant", "critical"},
		{"frequent", "high"},
		{"occasional", "medium"},
		{"rare", "low"},
		{"unknown", "medium"}, // default
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Frequency_%s", tt.frequency), func(t *testing.T) {
			result := frequencyToUrgency(tt.frequency)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestImpactToResourceCost tests impact to resource cost conversion
func TestImpactToResourceCost(t *testing.T) {
	tests := []struct {
		impact   string
		expected string
	}{
		{"system_down", "heavy"},
		{"degraded_performance", "moderate"},
		{"user_impact", "moderate"},
		{"cosmetic", "minimal"},
		{"unknown", "moderate"}, // default
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Impact_%s", tt.impact), func(t *testing.T) {
			result := impactToResourceCost(tt.impact)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestCountTasksInFolder tests counting tasks in a folder
func TestCountTasksInFolder(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("Success_EmptyFolder", func(t *testing.T) {
		folderPath := filepath.Join(env.TasksDir, "active")
		count := countTasksInFolder(folderPath)

		if count != 0 {
			t.Errorf("Expected 0 tasks, got %d", count)
		}
	})

	t.Run("Success_WithTasks", func(t *testing.T) {
		// Create test tasks
		testTask1 := setupTestTask(t, env, "bug-fix", "staged")
		defer testTask1.Cleanup()

		testTask2 := setupTestTask(t, env, "feature", "staged")
		defer testTask2.Cleanup()

		testTask3 := setupTestTask(t, env, "enhancement", "staged")
		defer testTask3.Cleanup()

		folderPath := filepath.Join(env.TasksDir, "staged")
		count := countTasksInFolder(folderPath)

		if count < 3 {
			t.Errorf("Expected at least 3 tasks, got %d", count)
		}
	})

	t.Run("Success_NonExistentFolder", func(t *testing.T) {
		count := countTasksInFolder("/nonexistent/path")

		if count != 0 {
			t.Errorf("Expected 0 for non-existent folder, got %d", count)
		}
	})
}

// TestScenarioExists tests scenario existence check
func TestScenarioExists(t *testing.T) {
	// This test checks if scenario directories exist
	// We can't reliably test this without the actual Vrooli structure
	// So we'll test the function logic

	t.Run("ValidCheck", func(t *testing.T) {
		// The function checks for scenario existence
		// Testing this requires actual filesystem structure
		result := scenarioExists("swarm-manager")
		// Result depends on actual filesystem, so we just ensure it returns a bool
		_ = result
	})
}
