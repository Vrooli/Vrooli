package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// TestDiscoverScenarios tests the scenario discovery function
func TestDiscoverScenarios(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Create temporary scenarios directory structure
	tempDir, err := ioutil.TempDir("", "discover-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Save original working directory
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWD)

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	t.Run("NoScenariosDirectory", func(t *testing.T) {
		orch := NewOrchestrator()
		discoverScenarios(orch, logger)

		scenarios := orch.GetScenarios()
		// Should handle missing directory gracefully
		t.Logf("Discovered %d scenarios with no scenarios directory", len(scenarios))
	})

	t.Run("EmptyScenariosDirectory", func(t *testing.T) {
		// Create scenarios directory
		if err := os.MkdirAll("scenarios", 0755); err != nil {
			t.Fatalf("Failed to create scenarios dir: %v", err)
		}

		orch := NewOrchestrator()
		discoverScenarios(orch, logger)

		scenarios := orch.GetScenarios()
		if len(scenarios) != 0 {
			t.Errorf("Expected 0 scenarios in empty directory, got %d", len(scenarios))
		}
	})

	t.Run("WithMaintenanceTaggedScenarios", func(t *testing.T) {
		// Create scenarios directory structure
		if err := os.MkdirAll("scenarios", 0755); err != nil {
			t.Fatalf("Failed to create scenarios dir: %v", err)
		}

		// Create a test scenario with maintenance tag
		scenarioDir := filepath.Join("scenarios", "test-maint-scenario")
		if err := os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0755); err != nil {
			t.Fatalf("Failed to create scenario dir: %v", err)
		}

		// Create service.json with maintenance tag
		serviceJSON := `{
  "service": {
    "name": "test-maint-scenario",
    "tags": ["maintenance", "test"]
  },
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999"
    }
  }
}`
		serviceJSONPath := filepath.Join(scenarioDir, ".vrooli", "service.json")
		if err := ioutil.WriteFile(serviceJSONPath, []byte(serviceJSON), 0644); err != nil {
			t.Fatalf("Failed to write service.json: %v", err)
		}

		orch := NewOrchestrator()
		discoverScenarios(orch, logger)

		scenarios := orch.GetScenarios()
		if len(scenarios) != 1 {
			t.Errorf("Expected 1 scenario, got %d", len(scenarios))
		}

		if len(scenarios) > 0 {
			scenario := scenarios[0]
			if scenario.Name != "test-maint-scenario" {
				t.Errorf("Expected scenario name 'test-maint-scenario', got '%s'", scenario.Name)
			}

			// Check tags
			hasMaintenanceTag := false
			for _, tag := range scenario.Tags {
				if tag == "maintenance" {
					hasMaintenanceTag = true
					break
				}
			}
			if !hasMaintenanceTag {
				t.Error("Expected scenario to have 'maintenance' tag")
			}

			// Scenario should start inactive
			if scenario.IsActive {
				t.Error("New scenario should start inactive")
			}
		}
	})

	t.Run("WithNonMaintenanceScenarios", func(t *testing.T) {
		// Clear scenarios
		if err := os.RemoveAll("scenarios"); err != nil {
			t.Fatalf("Failed to remove scenarios: %v", err)
		}
		if err := os.MkdirAll("scenarios", 0755); err != nil {
			t.Fatalf("Failed to create scenarios dir: %v", err)
		}

		// Create a scenario without maintenance tag
		scenarioDir := filepath.Join("scenarios", "test-normal-scenario")
		if err := os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0755); err != nil {
			t.Fatalf("Failed to create scenario dir: %v", err)
		}

		serviceJSON := `{
  "service": {
    "name": "test-normal-scenario",
    "tags": ["normal", "test"]
  },
  "ports": {
    "api": {
      "env_var": "API_PORT"
    }
  }
}`
		serviceJSONPath := filepath.Join(scenarioDir, ".vrooli", "service.json")
		if err := ioutil.WriteFile(serviceJSONPath, []byte(serviceJSON), 0644); err != nil {
			t.Fatalf("Failed to write service.json: %v", err)
		}

		orch := NewOrchestrator()
		discoverScenarios(orch, logger)

		scenarios := orch.GetScenarios()
		// Should not discover scenarios without maintenance tag
		if len(scenarios) != 0 {
			t.Errorf("Expected 0 maintenance scenarios, got %d", len(scenarios))
		}
	})

	t.Run("MultipleMaintenanceScenarios", func(t *testing.T) {
		// Clear scenarios
		if err := os.RemoveAll("scenarios"); err != nil {
			t.Fatalf("Failed to remove scenarios: %v", err)
		}
		if err := os.MkdirAll("scenarios", 0755); err != nil {
			t.Fatalf("Failed to create scenarios dir: %v", err)
		}

		// Create multiple scenarios with maintenance tag
		for i := 1; i <= 3; i++ {
			scenarioDir := filepath.Join("scenarios", "test-maint-scenario-"+string(rune('0'+i)))
			if err := os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0755); err != nil {
				t.Fatalf("Failed to create scenario dir: %v", err)
			}

			serviceJSON := `{
  "service": {
    "name": "test-maint-scenario-` + string(rune('0'+i)) + `",
    "tags": ["maintenance"]
  },
  "ports": {
    "api": {
      "env_var": "API_PORT"
    }
  }
}`
			serviceJSONPath := filepath.Join(scenarioDir, ".vrooli", "service.json")
			if err := ioutil.WriteFile(serviceJSONPath, []byte(serviceJSON), 0644); err != nil {
				t.Fatalf("Failed to write service.json: %v", err)
			}
		}

		orch := NewOrchestrator()
		discoverScenarios(orch, logger)

		scenarios := orch.GetScenarios()
		if len(scenarios) != 3 {
			t.Errorf("Expected 3 scenarios, got %d", len(scenarios))
		}
	})

	t.Run("InvalidServiceJSON", func(t *testing.T) {
		// Clear scenarios
		if err := os.RemoveAll("scenarios"); err != nil {
			t.Fatalf("Failed to remove scenarios: %v", err)
		}
		if err := os.MkdirAll("scenarios", 0755); err != nil {
			t.Fatalf("Failed to create scenarios dir: %v", err)
		}

		// Create scenario with invalid JSON
		scenarioDir := filepath.Join("scenarios", "test-invalid-scenario")
		if err := os.MkdirAll(filepath.Join(scenarioDir, ".vrooli"), 0755); err != nil {
			t.Fatalf("Failed to create scenario dir: %v", err)
		}

		serviceJSON := `{"invalid": json}`
		serviceJSONPath := filepath.Join(scenarioDir, ".vrooli", "service.json")
		if err := ioutil.WriteFile(serviceJSONPath, []byte(serviceJSON), 0644); err != nil {
			t.Fatalf("Failed to write service.json: %v", err)
		}

		orch := NewOrchestrator()
		discoverScenarios(orch, logger)

		scenarios := orch.GetScenarios()
		// Should skip invalid scenarios
		if len(scenarios) != 0 {
			t.Errorf("Expected 0 scenarios (invalid JSON should be skipped), got %d", len(scenarios))
		}
	})

	t.Run("MissingServiceJSON", func(t *testing.T) {
		// Clear scenarios
		if err := os.RemoveAll("scenarios"); err != nil {
			t.Fatalf("Failed to remove scenarios: %v", err)
		}
		if err := os.MkdirAll("scenarios", 0755); err != nil {
			t.Fatalf("Failed to create scenarios dir: %v", err)
		}

		// Create scenario directory without service.json
		scenarioDir := filepath.Join("scenarios", "test-missing-json")
		if err := os.MkdirAll(scenarioDir, 0755); err != nil {
			t.Fatalf("Failed to create scenario dir: %v", err)
		}

		orch := NewOrchestrator()
		discoverScenarios(orch, logger)

		scenarios := orch.GetScenarios()
		// Should skip scenarios without service.json
		if len(scenarios) != 0 {
			t.Errorf("Expected 0 scenarios (missing service.json), got %d", len(scenarios))
		}
	})
}
