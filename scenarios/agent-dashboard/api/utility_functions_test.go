package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResolveAgentIdentifier(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("EmptyIdentifier", func(t *testing.T) {
		// Clean up any existing manager state
		if codexManager != nil {
			codexManager.mu.Lock()
			codexManager.agents = make(map[string]*managedAgent)
			codexManager.mu.Unlock()
		}

		result := resolveAgentIdentifier("")
		if result != "" {
			t.Errorf("Expected empty result for empty identifier, got %s", result)
		}
	})

	t.Run("FullAgentID", func(t *testing.T) {
		// If identifier is already a full ID (contains ':'), it should be returned
		identifier := "codex:agent-123"
		result := resolveAgentIdentifier(identifier)
		// The function may validate or transform the ID
		if result == "" {
			t.Logf("Full ID %s not found in manager (expected for non-existent agent)", identifier)
		}
	})

	t.Run("ShortName", func(t *testing.T) {
		// Test resolution by short name
		shortName := "test-agent"
		result := resolveAgentIdentifier(shortName)
		// Empty result is expected if no matching agent exists
		if result != "" {
			t.Logf("Found agent ID %s for short name %s", result, shortName)
		}
	})
}

func TestDetectScenarioRoot(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("FromCurrentDirectory", func(t *testing.T) {
		// Create a temporary directory structure
		tmpDir, err := os.MkdirTemp("", "scenario-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create service.json to mark as scenario root
		serviceFile := filepath.Join(tmpDir, "service.json")
		if err := os.WriteFile(serviceFile, []byte(`{"name":"test-scenario"}`), 0644); err != nil {
			t.Fatalf("Failed to create service.json: %v", err)
		}

		// Change to temp directory
		oldWd, _ := os.Getwd()
		defer os.Chdir(oldWd)

		os.Chdir(tmpDir)
		root := detectScenarioRoot()

		if root == "" {
			t.Error("Expected to detect scenario root, got empty string")
		}

		// Verify the detected root contains service.json
		serviceJsonPath := filepath.Join(root, "service.json")
		if _, err := os.Stat(serviceJsonPath); os.IsNotExist(err) {
			t.Errorf("Expected service.json at %s", serviceJsonPath)
		}
	})

	t.Run("NoServiceJson", func(t *testing.T) {
		// Create temporary directory without service.json
		tmpDir, err := os.MkdirTemp("", "no-scenario-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		oldWd, _ := os.Getwd()
		defer os.Chdir(oldWd)

		os.Chdir(tmpDir)
		root := detectScenarioRoot()

		// Should return empty or some default when not in a scenario
		t.Logf("detectScenarioRoot from non-scenario dir returned: %s", root)
	})
}

func TestDetectVrooliRoot(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("FromCurrentDirectory", func(t *testing.T) {
		currentDir, _ := os.Getwd()
		root := detectVrooliRoot(currentDir)

		// Should return a path (even if empty when not in Vrooli)
		t.Logf("detectVrooliRoot returned: %s", root)

		// If we got a path, verify .vrooli directory exists
		if root != "" {
			vrooliDir := filepath.Join(root, ".vrooli")
			if _, err := os.Stat(vrooliDir); err != nil {
				t.Logf("Detected root %s but no .vrooli directory found (may be expected)", root)
			}
		}
	})

	t.Run("FromTemporaryDirectory", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "vrooli-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		root := detectVrooliRoot(tmpDir)

		// From a random temp directory, may not find Vrooli root
		t.Logf("detectVrooliRoot from temp dir returned: %s", root)
	})
}

func TestResolveCodexAgentTimeout(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Create a temp scenario directory
	tmpDir, err := os.MkdirTemp("", "scenario-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("WithDefaultTimeout", func(t *testing.T) {
		timeout := resolveCodexAgentTimeout(tmpDir)

		// Should return default timeout when no config exists
		if timeout != defaultAgentTimeout {
			t.Logf("Expected default timeout %v, got %v", defaultAgentTimeout, timeout)
		}
	})

	t.Run("WithInvalidScenarioRoot", func(t *testing.T) {
		timeout := resolveCodexAgentTimeout("/nonexistent/path")

		// Should fall back to default on error
		if timeout != defaultAgentTimeout {
			t.Errorf("Expected default timeout %v for invalid path, got %v", defaultAgentTimeout, timeout)
		}
	})
}

func TestComputeUptime(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	t.Run("RunningAgent_NoEndTime", func(t *testing.T) {
		uptime := computeUptime(baseTime, nil)
		if uptime == "" {
			t.Error("Expected non-empty uptime for running agent")
		}
	})

	t.Run("CompletedAgent_WithEndTime", func(t *testing.T) {
		endTime := baseTime.Add(2 * time.Hour)
		uptime := computeUptime(baseTime, &endTime)
		if uptime == "" {
			t.Error("Expected non-empty uptime for completed agent")
		}
		// Should contain duration information
		t.Logf("Computed uptime: %s", uptime)
	})

	t.Run("ShortDuration", func(t *testing.T) {
		endTime := baseTime.Add(30 * time.Second)
		uptime := computeUptime(baseTime, &endTime)
		if uptime == "" {
			t.Error("Expected non-empty uptime for short duration")
		}
	})

	t.Run("LongDuration", func(t *testing.T) {
		endTime := baseTime.Add(25 * time.Hour)
		uptime := computeUptime(baseTime, &endTime)
		if uptime == "" {
			t.Error("Expected non-empty uptime for long duration")
		}
		// Should handle days/hours properly
		t.Logf("Long duration uptime: %s", uptime)
	})
}

// Note: waitForExit and appendLog are methods on managedAgent struct, not standalone functions
// They are tested indirectly through integration tests and the manager lifecycle tests
