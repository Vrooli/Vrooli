package main

import (
	"testing"
)

// TestUpdateResourceUsage tests the UpdateResourceUsage function
func TestUpdateResourceUsage(t *testing.T) {
	o := NewOrchestrator()

	// Add a test scenario
	o.AddScenario(&MaintenanceScenario{
		ID:          "test-scenario",
		DisplayName: "Test Scenario",
		Description: "Test description",
		Tags:        []string{"test"},
	})

	// Test updating resource usage for existing scenario
	usage := map[string]float64{
		"cpu_percent":    25.5,
		"memory_percent": 15.2,
	}
	o.UpdateResourceUsage("test-scenario", usage)

	// Verify the usage was updated
	scenario, exists := o.GetScenario("test-scenario")
	if !exists || scenario == nil {
		t.Fatal("Expected scenario to exist")
	}

	if scenario.ResourceUsage == nil {
		t.Fatal("Expected resource usage to be set")
	}

	if cpu, ok := scenario.ResourceUsage["cpu_percent"]; !ok || cpu != 25.5 {
		t.Errorf("Expected CPU usage 25.5, got %v", cpu)
	}

	if mem, ok := scenario.ResourceUsage["memory_percent"]; !ok || mem != 15.2 {
		t.Errorf("Expected memory usage 15.2, got %v", mem)
	}
}

// TestUpdateResourceUsage_NonExistentScenario tests updating resource usage for non-existent scenario
func TestUpdateResourceUsage_NonExistentScenario(t *testing.T) {
	o := NewOrchestrator()

	// Test updating resource usage for non-existent scenario (should not panic)
	usage := map[string]float64{
		"cpu_percent":    25.5,
		"memory_percent": 15.2,
	}
	o.UpdateResourceUsage("non-existent", usage)

	// Verify scenario still doesn't exist
	scenario, exists := o.GetScenario("non-existent")
	if exists || scenario != nil {
		t.Error("Expected scenario to not exist")
	}
}

// TestUpdateResourceUsage_EmptyUsage tests updating with empty usage map
func TestUpdateResourceUsage_EmptyUsage(t *testing.T) {
	o := NewOrchestrator()

	// Add a test scenario
	o.AddScenario(&MaintenanceScenario{
		ID:          "test-scenario",
		DisplayName: "Test Scenario",
		Description: "Test description",
		Tags:        []string{"test"},
	})

	// Test updating with empty usage
	usage := map[string]float64{}
	o.UpdateResourceUsage("test-scenario", usage)

	// Verify the usage was updated (even if empty)
	scenario, exists := o.GetScenario("test-scenario")
	if !exists || scenario == nil {
		t.Fatal("Expected scenario to exist")
	}

	if scenario.ResourceUsage == nil {
		t.Fatal("Expected resource usage to be set (even if empty)")
	}

	if len(scenario.ResourceUsage) != 0 {
		t.Errorf("Expected empty resource usage, got %v", scenario.ResourceUsage)
	}
}

// TestUpdateResourceUsage_NilUsage tests updating with nil usage map
func TestUpdateResourceUsage_NilUsage(t *testing.T) {
	o := NewOrchestrator()

	// Add a test scenario with initial usage
	o.AddScenario(&MaintenanceScenario{
		ID:          "test-scenario",
		DisplayName: "Test Scenario",
		Description: "Test description",
		Tags:        []string{"test"},
		ResourceUsage: map[string]float64{
			"cpu_percent": 10.0,
		},
	})

	// Test updating with nil usage
	o.UpdateResourceUsage("test-scenario", nil)

	// Verify the usage was updated to nil
	scenario, exists := o.GetScenario("test-scenario")
	if !exists || scenario == nil {
		t.Fatal("Expected scenario to exist")
	}

	if scenario.ResourceUsage != nil {
		t.Errorf("Expected resource usage to be nil, got %v", scenario.ResourceUsage)
	}
}

// TestGetScenarioResourceUsage tests the getScenarioResourceUsage function
func TestGetScenarioResourceUsage(t *testing.T) {
	// Test with port 0 (should return nil immediately)
	usage := getScenarioResourceUsage("test-scenario", 0)
	if usage != nil {
		t.Errorf("Expected nil usage for port 0, got %v", usage)
	}
}

// TestGetScenarioResourceUsage_InvalidPort tests with an unlikely port
func TestGetScenarioResourceUsage_InvalidPort(t *testing.T) {
	// Test with a port that's unlikely to be in use
	usage := getScenarioResourceUsage("test-scenario", 65535)
	// Should return nil when no process is found (expected behavior)
	if usage != nil {
		t.Logf("Got usage data for port 65535: %v (port may actually be in use)", usage)
	}
}

// TestGetScenarioResourceUsage_NegativePort tests with negative port
func TestGetScenarioResourceUsage_NegativePort(t *testing.T) {
	// Test with negative port (invalid, should handle gracefully)
	usage := getScenarioResourceUsage("test-scenario", -1)
	// Should return nil for invalid port (command will fail)
	if usage != nil {
		t.Errorf("Expected nil usage for negative port, got %v", usage)
	}
}
