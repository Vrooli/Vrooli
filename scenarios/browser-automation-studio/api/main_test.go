package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
)

// TestDatabaseModels tests the database models
func TestDatabaseModels(t *testing.T) {
	t.Run("JSONMapValue", func(t *testing.T) {
		m := database.JSONMap{
			"key": "value",
			"nested": map[string]interface{}{
				"inner": "data",
			},
		}

		val, err := m.Value()
		if err != nil {
			t.Fatalf("Failed to marshal JSONMap: %v", err)
		}

		if val == nil {
			t.Error("Expected non-nil value")
		}
	})

	t.Run("JSONMapScan", func(t *testing.T) {
		var m database.JSONMap
		jsonData := []byte(`{"key":"value","number":42}`)

		err := m.Scan(jsonData)
		if err != nil {
			t.Fatalf("Failed to scan JSONMap: %v", err)
		}

		if m["key"] != "value" {
			t.Errorf("Expected key=value, got %v", m["key"])
		}
	})

	t.Run("ProjectStructure", func(t *testing.T) {
		project := &database.Project{
			ID:          uuid.New(),
			Name:        "Test Project",
			Description: "A test project",
			FolderPath:  "/test/project",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if project.ID == uuid.Nil {
			t.Error("Expected valid UUID")
		}

		if project.Name == "" {
			t.Error("Expected non-empty name")
		}
	})

	t.Run("WorkflowStructure", func(t *testing.T) {
		workflow := &database.Workflow{
			ID:         uuid.New(),
			Name:       "Test Workflow",
			FolderPath: "/test",
			FlowDefinition: database.JSONMap{
				"nodes": []interface{}{},
				"edges": []interface{}{},
			},
			Tags:    []string{"test", "automation"},
			Version: 1,
		}

		if len(workflow.Tags) != 2 {
			t.Errorf("Expected 2 tags, got %d", len(workflow.Tags))
		}

		if workflow.Version != 1 {
			t.Errorf("Expected version 1, got %d", workflow.Version)
		}
	})

	t.Run("ExecutionStructure", func(t *testing.T) {
		execution := &database.Execution{
			ID:         uuid.New(),
			WorkflowID: uuid.New(),
			Status:     "pending",
			StartedAt:  time.Now(),
		}

		if execution.Status != "pending" {
			t.Errorf("Expected status pending, got %s", execution.Status)
		}

		if execution.StartedAt.IsZero() {
			t.Error("Expected non-zero start time")
		}
	})
}

// TestJSONSerialization tests JSON serialization of models
func TestJSONSerialization(t *testing.T) {
	t.Run("ProjectToJSON", func(t *testing.T) {
		project := &database.Project{
			ID:          uuid.New(),
			Name:        "JSON Test",
			Description: "Testing JSON serialization",
			FolderPath:  "/test",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		data, err := json.Marshal(project)
		if err != nil {
			t.Fatalf("Failed to marshal project: %v", err)
		}

		var decoded database.Project
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal project: %v", err)
		}

		if decoded.Name != project.Name {
			t.Errorf("Expected name %s, got %s", project.Name, decoded.Name)
		}
	})

	t.Run("WorkflowToJSON", func(t *testing.T) {
		workflow := &database.Workflow{
			ID:         uuid.New(),
			Name:       "JSON Workflow",
			FolderPath: "/test",
			FlowDefinition: database.JSONMap{
				"version": "1.0",
				"nodes":   []interface{}{},
			},
			Tags:    []string{"test"},
			Version: 1,
		}

		data, err := json.Marshal(workflow)
		if err != nil {
			t.Fatalf("Failed to marshal workflow: %v", err)
		}

		if len(data) == 0 {
			t.Error("Expected non-empty JSON data")
		}
	})
}

// TestErrorTypes tests error handling
func TestErrorTypes(t *testing.T) {
	t.Run("ErrNotFound", func(t *testing.T) {
		err := database.ErrNotFound
		if err == nil {
			t.Error("Expected non-nil error")
		}

		if err.Error() != "not found" {
			t.Errorf("Expected 'not found', got %s", err.Error())
		}
	})
}

// TestContextOperations tests context-aware operations
func TestContextOperations(t *testing.T) {
	t.Run("ContextWithTimeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		select {
		case <-ctx.Done():
			t.Error("Context should not be done yet")
		case <-time.After(100 * time.Millisecond):
			// Success - context still active
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		cancel()

		select {
		case <-ctx.Done():
			// Success - context cancelled
		case <-time.After(100 * time.Millisecond):
			t.Error("Context should be cancelled")
		}
	})
}

// TestUUIDOperations tests UUID handling
func TestUUIDOperations(t *testing.T) {
	t.Run("GenerateUUID", func(t *testing.T) {
		id1 := uuid.New()
		id2 := uuid.New()

		if id1 == id2 {
			t.Error("Expected different UUIDs")
		}

		if id1 == uuid.Nil {
			t.Error("Expected non-nil UUID")
		}
	})

	t.Run("UUIDToString", func(t *testing.T) {
		id := uuid.New()
		str := id.String()

		if len(str) != 36 {
			t.Errorf("Expected UUID string length 36, got %d", len(str))
		}
	})

	t.Run("ParseUUID", func(t *testing.T) {
		original := uuid.New()
		str := original.String()

		parsed, err := uuid.Parse(str)
		if err != nil {
			t.Fatalf("Failed to parse UUID: %v", err)
		}

		if parsed != original {
			t.Error("Parsed UUID doesn't match original")
		}
	})
}

// TestLogging tests logger functionality
func TestLogging(t *testing.T) {
	t.Run("CreateLogger", func(t *testing.T) {
		log := logrus.New()
		log.SetOutput(ioutil.Discard)

		if log == nil {
			t.Fatal("Expected non-nil logger")
		}

		log.Info("Test message")
		log.WithField("key", "value").Debug("Debug message")
	})

	t.Run("LogLevels", func(t *testing.T) {
		log := logrus.New()
		log.SetOutput(ioutil.Discard)

		log.SetLevel(logrus.DebugLevel)
		if log.Level != logrus.DebugLevel {
			t.Error("Expected debug level")
		}

		log.SetLevel(logrus.InfoLevel)
		if log.Level != logrus.InfoLevel {
			t.Error("Expected info level")
		}
	})
}

// TestEnvironmentVariables tests environment variable handling
func TestEnvironmentVariables(t *testing.T) {
	t.Run("SetAndGetEnv", func(t *testing.T) {
		key := "TEST_VAR"
		value := "test_value"

		os.Setenv(key, value)
		defer os.Unsetenv(key)

		if got := os.Getenv(key); got != value {
			t.Errorf("Expected %s, got %s", value, got)
		}
	})

	t.Run("UnsetEnv", func(t *testing.T) {
		key := "TEST_VAR_2"
		os.Setenv(key, "value")
		os.Unsetenv(key)

		if got := os.Getenv(key); got != "" {
			t.Errorf("Expected empty string, got %s", got)
		}
	})
}

// TestTimeOperations tests time-related functionality
func TestTimeOperations(t *testing.T) {
	t.Run("TimeNow", func(t *testing.T) {
		now := time.Now()
		if now.IsZero() {
			t.Error("Expected non-zero time")
		}
	})

	t.Run("TimeDuration", func(t *testing.T) {
		start := time.Now()
		time.Sleep(10 * time.Millisecond)
		duration := time.Since(start)

		if duration < 10*time.Millisecond {
			t.Error("Expected duration >= 10ms")
		}
	})

	t.Run("TimeFormatting", func(t *testing.T) {
		now := time.Now()
		formatted := now.Format(time.RFC3339)

		if formatted == "" {
			t.Error("Expected non-empty formatted time")
		}
	})
}
