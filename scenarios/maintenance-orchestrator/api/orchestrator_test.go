package main

import (
	"testing"
	"time"
)

func TestNewOrchestrator(t *testing.T) {
	orch := NewOrchestrator()

	if orch == nil {
		t.Fatal("NewOrchestrator() returned nil")
	}

	if orch.scenarios == nil {
		t.Error("scenarios map not initialized")
	}

	if orch.presets == nil {
		t.Error("presets map not initialized")
	}

	if orch.activityLog == nil {
		t.Error("activityLog not initialized")
	}
}

func TestAddScenario(t *testing.T) {
	orch := NewOrchestrator()
	scenario := &MaintenanceScenario{
		ID:          "test-scenario",
		Name:        "test-scenario",
		DisplayName: "Test Scenario",
		Description: "A test scenario",
		IsActive:    false,
		Tags:        []string{"test", "maintenance"},
	}

	orch.AddScenario(scenario)

	retrieved, exists := orch.GetScenario("test-scenario")
	if !exists {
		t.Error("Scenario not found after adding")
	}

	if retrieved.ID != "test-scenario" {
		t.Errorf("Expected ID 'test-scenario', got '%s'", retrieved.ID)
	}
}

func TestAddScenarioThreadSafety(t *testing.T) {
	orch := NewOrchestrator()

	// Test concurrent additions
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			scenario := &MaintenanceScenario{
				ID:   string(rune('a' + id)),
				Name: string(rune('a' + id)),
			}
			orch.AddScenario(scenario)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	scenarios := orch.GetScenarios()
	if len(scenarios) != 10 {
		t.Errorf("Expected 10 scenarios, got %d", len(scenarios))
	}
}

func TestActivateDeactivateScenario(t *testing.T) {
	orch := NewOrchestrator()
	scenario := &MaintenanceScenario{
		ID:       "test-scenario",
		Name:     "test-scenario",
		IsActive: false,
	}

	orch.AddScenario(scenario)

	// Test activation
	if !orch.ActivateScenario("test-scenario") {
		t.Error("Failed to activate scenario")
	}

	retrieved, _ := orch.GetScenario("test-scenario")
	if !retrieved.IsActive {
		t.Error("Scenario should be active after activation")
	}

	// Verify LastActive timestamp was set
	if retrieved.LastActive == nil {
		t.Error("LastActive timestamp should be set")
	}

	// Test deactivation
	if !orch.DeactivateScenario("test-scenario") {
		t.Error("Failed to deactivate scenario")
	}

	retrieved, _ = orch.GetScenario("test-scenario")
	if retrieved.IsActive {
		t.Error("Scenario should be inactive after deactivation")
	}
}

func TestActivateNonexistentScenario(t *testing.T) {
	orch := NewOrchestrator()

	if orch.ActivateScenario("nonexistent") {
		t.Error("Should not be able to activate nonexistent scenario")
	}
}

func TestDeactivateNonexistentScenario(t *testing.T) {
	orch := NewOrchestrator()

	if orch.DeactivateScenario("nonexistent") {
		t.Error("Should not be able to deactivate nonexistent scenario")
	}
}

func TestGetScenario(t *testing.T) {
	orch := NewOrchestrator()
	scenario := &MaintenanceScenario{
		ID:   "test",
		Name: "test",
	}
	orch.AddScenario(scenario)

	t.Run("ExistingScenario", func(t *testing.T) {
		retrieved, exists := orch.GetScenario("test")
		if !exists {
			t.Error("Scenario should exist")
		}
		if retrieved.ID != "test" {
			t.Errorf("Expected ID 'test', got '%s'", retrieved.ID)
		}
	})

	t.Run("NonExistentScenario", func(t *testing.T) {
		_, exists := orch.GetScenario("nonexistent")
		if exists {
			t.Error("Scenario should not exist")
		}
	})
}

func TestGetScenarios(t *testing.T) {
	orch := NewOrchestrator()

	// Add multiple scenarios
	for i := 0; i < 5; i++ {
		scenario := &MaintenanceScenario{
			ID:       string(rune('a' + i)),
			Name:     string(rune('a' + i)),
			IsActive: false,
		}
		orch.AddScenario(scenario)
	}

	scenarios := orch.GetScenarios()
	if len(scenarios) != 5 {
		t.Errorf("Expected 5 scenarios, got %d", len(scenarios))
	}
}

func TestGetStatus(t *testing.T) {
	orch := NewOrchestrator()

	// Add scenarios with mixed active states
	for i := 0; i < 5; i++ {
		scenario := &MaintenanceScenario{
			ID:       string(rune('a' + i)),
			Name:     string(rune('a' + i)),
			IsActive: i < 2, // First 2 are active
		}
		orch.AddScenario(scenario)
	}

	total, active, _ := orch.GetStatus()

	if total != 5 {
		t.Errorf("Expected 5 total scenarios, got %d", total)
	}

	if active != 2 {
		t.Errorf("Expected 2 active scenarios, got %d", active)
	}
}

func TestStopAll(t *testing.T) {
	orch := NewOrchestrator()

	// Add some active scenarios
	for i := 0; i < 3; i++ {
		scenario := &MaintenanceScenario{
			ID:       string(rune('a' + i)),
			Name:     string(rune('a' + i)),
			IsActive: true,
		}
		orch.AddScenario(scenario)
	}

	deactivated := orch.StopAll()

	if len(deactivated) != 3 {
		t.Errorf("Expected 3 deactivated scenarios, got %d", len(deactivated))
	}

	// Verify all are inactive
	_, active, _ := orch.GetStatus()
	if active != 0 {
		t.Errorf("Expected 0 active scenarios after StopAll, got %d", active)
	}
}

func TestStopAllWithPresets(t *testing.T) {
	orch := NewOrchestrator()

	// Add scenarios and presets
	scenario := &MaintenanceScenario{
		ID:       "test",
		Name:     "test",
		IsActive: true,
	}
	orch.AddScenario(scenario)

	preset := &Preset{
		ID:       "preset1",
		Name:     "Test Preset",
		IsActive: true,
		States:   map[string]bool{"test": true},
	}
	orch.presets["preset1"] = preset

	// Stop all
	orch.StopAll()

	// Verify preset is deactivated
	if preset.IsActive {
		t.Error("Preset should be deactivated")
	}
}

func TestActivityLog(t *testing.T) {
	orch := NewOrchestrator()
	scenario := &MaintenanceScenario{
		ID:       "test-scenario",
		Name:     "test-scenario",
		IsActive: false,
	}

	orch.AddScenario(scenario)

	// Perform an action that logs activity
	orch.ActivateScenario("test-scenario")

	_, _, activity := orch.GetStatus()

	if len(activity) == 0 {
		t.Error("Expected activity log to contain entries")
	}

	if activity[0].Action != "activate" {
		t.Errorf("Expected action 'activate', got '%s'", activity[0].Action)
	}

	// Verify activity entry fields
	entry := activity[0]
	if entry.Scenario != "test-scenario" {
		t.Errorf("Expected scenario 'test-scenario', got '%s'", entry.Scenario)
	}
	if !entry.Success {
		t.Error("Expected success to be true")
	}
	if entry.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

func TestActivityLogMultipleActions(t *testing.T) {
	orch := NewOrchestrator()
	scenario := &MaintenanceScenario{
		ID:   "test",
		Name: "test",
	}
	orch.AddScenario(scenario)

	// Perform multiple actions
	orch.ActivateScenario("test")
	time.Sleep(1 * time.Millisecond)
	orch.DeactivateScenario("test")
	time.Sleep(1 * time.Millisecond)
	orch.ActivateScenario("test")

	_, _, activity := orch.GetStatus()

	if len(activity) != 3 {
		t.Errorf("Expected 3 activity entries, got %d", len(activity))
	}
}

func TestActivityLogLimit(t *testing.T) {
	orch := NewOrchestrator()
	scenario := &MaintenanceScenario{
		ID:   "test",
		Name: "test",
	}
	orch.AddScenario(scenario)

	// Add more than 10 activities
	for i := 0; i < 15; i++ {
		if i%2 == 0 {
			orch.ActivateScenario("test")
		} else {
			orch.DeactivateScenario("test")
		}
	}

	_, _, activity := orch.GetStatus()

	// GetStatus returns max 10 recent activities
	if len(activity) > 10 {
		t.Errorf("Expected at most 10 activity entries, got %d", len(activity))
	}
}

func TestGetPresets(t *testing.T) {
	orch := NewOrchestrator()

	preset1 := &Preset{ID: "p1", Name: "Preset 1", States: make(map[string]bool)}
	preset2 := &Preset{ID: "p2", Name: "Preset 2", States: make(map[string]bool)}

	orch.presets["p1"] = preset1
	orch.presets["p2"] = preset2

	presets := orch.GetPresets()

	if len(presets) != 2 {
		t.Errorf("Expected 2 presets, got %d", len(presets))
	}
}

func TestGetPreset(t *testing.T) {
	orch := NewOrchestrator()

	preset := &Preset{ID: "test", Name: "Test Preset", States: make(map[string]bool)}
	orch.presets["test"] = preset

	t.Run("ExistingPreset", func(t *testing.T) {
		retrieved, exists := orch.GetPreset("test")
		if !exists {
			t.Error("Preset should exist")
		}
		if retrieved.ID != "test" {
			t.Errorf("Expected ID 'test', got '%s'", retrieved.ID)
		}
	})

	t.Run("NonExistentPreset", func(t *testing.T) {
		_, exists := orch.GetPreset("nonexistent")
		if exists {
			t.Error("Preset should not exist")
		}
	})
}

func TestApplyPreset(t *testing.T) {
	orch := NewOrchestrator()

	// Add scenarios
	for i := 0; i < 3; i++ {
		scenario := &MaintenanceScenario{
			ID:       string(rune('a' + i)),
			Name:     string(rune('a' + i)),
			IsActive: false,
		}
		orch.AddScenario(scenario)
	}

	// Add preset
	preset := &Preset{
		ID:   "preset1",
		Name: "Test Preset",
		States: map[string]bool{
			"a": true,
			"b": true,
		},
		IsActive: false,
	}
	orch.presets["preset1"] = preset

	t.Run("ActivatePreset", func(t *testing.T) {
		activated, deactivated, success := orch.ApplyPreset("preset1")

		if !success {
			t.Error("ApplyPreset should succeed")
		}

		if len(activated) != 2 {
			t.Errorf("Expected 2 activated scenarios, got %d", len(activated))
		}

		if len(deactivated) != 0 {
			t.Errorf("Expected 0 deactivated scenarios, got %d", len(deactivated))
		}

		// Verify preset is active
		if !preset.IsActive {
			t.Error("Preset should be active")
		}

		// Verify scenarios are active
		scenarioA, _ := orch.GetScenario("a")
		scenarioB, _ := orch.GetScenario("b")
		scenarioC, _ := orch.GetScenario("c")

		if !scenarioA.IsActive {
			t.Error("Scenario 'a' should be active")
		}
		if !scenarioB.IsActive {
			t.Error("Scenario 'b' should be active")
		}
		if scenarioC.IsActive {
			t.Error("Scenario 'c' should be inactive")
		}
	})

	t.Run("DeactivatePreset", func(t *testing.T) {
		// Apply again to deactivate
		activated, deactivated, success := orch.ApplyPreset("preset1")

		if !success {
			t.Error("ApplyPreset should succeed")
		}

		if len(activated) != 0 {
			t.Errorf("Expected 0 activated scenarios, got %d", len(activated))
		}

		if len(deactivated) != 2 {
			t.Errorf("Expected 2 deactivated scenarios, got %d", len(deactivated))
		}

		// Verify preset is inactive
		if preset.IsActive {
			t.Error("Preset should be inactive")
		}
	})

	t.Run("NonExistentPreset", func(t *testing.T) {
		_, _, success := orch.ApplyPreset("nonexistent")

		if success {
			t.Error("ApplyPreset should fail for nonexistent preset")
		}
	})
}

func TestApplyPresetUnion(t *testing.T) {
	orch := NewOrchestrator()

	// Add scenarios
	scenarios := []string{"a", "b", "c", "d"}
	for _, id := range scenarios {
		orch.AddScenario(&MaintenanceScenario{
			ID:       id,
			Name:     id,
			IsActive: false,
		})
	}

	// Add presets
	preset1 := &Preset{
		ID:       "preset1",
		Name:     "Preset 1",
		States:   map[string]bool{"a": true, "b": true},
		IsActive: false,
	}
	preset2 := &Preset{
		ID:       "preset2",
		Name:     "Preset 2",
		States:   map[string]bool{"b": true, "c": true},
		IsActive: false,
	}
	orch.presets["preset1"] = preset1
	orch.presets["preset2"] = preset2

	// Apply preset1
	orch.ApplyPreset("preset1")

	// Verify only a and b are active
	scenarioA, _ := orch.GetScenario("a")
	scenarioB, _ := orch.GetScenario("b")
	scenarioC, _ := orch.GetScenario("c")
	scenarioD, _ := orch.GetScenario("d")

	if !scenarioA.IsActive || !scenarioB.IsActive {
		t.Error("Scenarios a and b should be active")
	}
	if scenarioC.IsActive || scenarioD.IsActive {
		t.Error("Scenarios c and d should be inactive")
	}

	// Apply preset2 (union should activate a, b, c)
	orch.ApplyPreset("preset2")

	// Re-fetch scenarios
	scenarioA, _ = orch.GetScenario("a")
	scenarioB, _ = orch.GetScenario("b")
	scenarioC, _ = orch.GetScenario("c")
	scenarioD, _ = orch.GetScenario("d")

	if !scenarioA.IsActive || !scenarioB.IsActive || !scenarioC.IsActive {
		t.Error("Scenarios a, b, and c should be active (union of both presets)")
	}
	if scenarioD.IsActive {
		t.Error("Scenario d should be inactive")
	}
}

func TestGetActivePresets(t *testing.T) {
	orch := NewOrchestrator()

	preset1 := &Preset{ID: "p1", Name: "Preset 1", IsActive: true, States: make(map[string]bool)}
	preset2 := &Preset{ID: "p2", Name: "Preset 2", IsActive: false, States: make(map[string]bool)}
	preset3 := &Preset{ID: "p3", Name: "Preset 3", IsActive: true, States: make(map[string]bool)}

	orch.presets["p1"] = preset1
	orch.presets["p2"] = preset2
	orch.presets["p3"] = preset3

	activePresets := orch.GetActivePresets()

	if len(activePresets) != 2 {
		t.Errorf("Expected 2 active presets, got %d", len(activePresets))
	}
}

func TestIsPresetActive(t *testing.T) {
	orch := NewOrchestrator()

	preset := &Preset{ID: "test", Name: "Test", IsActive: true, States: make(map[string]bool)}
	orch.presets["test"] = preset

	t.Run("ActivePreset", func(t *testing.T) {
		if !orch.IsPresetActive("test") {
			t.Error("Preset should be active")
		}
	})

	t.Run("InactivePreset", func(t *testing.T) {
		preset.IsActive = false
		if orch.IsPresetActive("test") {
			t.Error("Preset should be inactive")
		}
	})

	t.Run("NonExistentPreset", func(t *testing.T) {
		if orch.IsPresetActive("nonexistent") {
			t.Error("Non-existent preset should return false")
		}
	})
}

func TestUpdatePresetStates(t *testing.T) {
	orch := NewOrchestrator()

	// Add scenarios
	scenario1 := &MaintenanceScenario{
		ID:   "s1",
		Name: "s1",
		Tags: []string{"maintenance", "backup"},
	}
	scenario2 := &MaintenanceScenario{
		ID:   "s2",
		Name: "s2",
		Tags: []string{"security"},
	}
	scenario3 := &MaintenanceScenario{
		ID:   "s3",
		Name: "s3",
		Tags: []string{"maintenance"},
	}

	orch.AddScenario(scenario1)
	orch.AddScenario(scenario2)
	orch.AddScenario(scenario3)

	// Add preset with tag filter
	preset := &Preset{
		ID:      "preset1",
		Name:    "Maintenance Preset",
		Tags:    []string{"maintenance"},
		States:  make(map[string]bool),
		Pattern: "",
	}
	orch.presets["preset1"] = preset

	// Update preset states based on tags
	orch.UpdatePresetStates()

	// Verify scenarios with matching tags were added to preset
	if !preset.States["s1"] {
		t.Error("Scenario s1 should be in preset (has maintenance tag)")
	}
	if preset.States["s2"] {
		t.Error("Scenario s2 should not be in preset (no maintenance tag)")
	}
	if !preset.States["s3"] {
		t.Error("Scenario s3 should be in preset (has maintenance tag)")
	}
}

func TestUpdatePresetStatesWithWildcard(t *testing.T) {
	orch := NewOrchestrator()

	// Add scenarios
	orch.AddScenario(&MaintenanceScenario{ID: "s1", Name: "s1"})
	orch.AddScenario(&MaintenanceScenario{ID: "s2", Name: "s2"})

	// Add preset with wildcard pattern
	preset := &Preset{
		ID:      "all",
		Name:    "All Scenarios",
		Pattern: "*",
		States:  make(map[string]bool),
	}
	orch.presets["all"] = preset

	// Update preset states
	orch.UpdatePresetStates()

	// Verify all scenarios were added
	if !preset.States["s1"] || !preset.States["s2"] {
		t.Error("All scenarios should be in wildcard preset")
	}
}
