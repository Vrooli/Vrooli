package main

import (
	"testing"
)

func TestInitializeDefaultPresets(t *testing.T) {
	orch := NewOrchestrator()

	// Should start with no presets
	if len(orch.presets) != 0 {
		t.Errorf("Expected 0 presets before initialization, got %d", len(orch.presets))
	}

	initializeDefaultPresets(orch)

	// Should have presets after initialization
	if len(orch.presets) == 0 {
		t.Error("Expected presets after initialization")
	}

	// Verify specific presets exist based on actual implementation
	expectedPresets := []string{
		"full",
		"emergency",
		"security",
		"self-improvement",
		"performance",
		"quality",
		"analytics",
	}

	for _, presetID := range expectedPresets {
		if _, exists := orch.presets[presetID]; !exists {
			t.Errorf("Expected preset '%s' to exist", presetID)
		}
	}

	// Verify preset structure
	fullPreset, exists := orch.presets["full"]
	if !exists {
		t.Fatal("Expected 'full' preset to exist")
	}
	if fullPreset.Name == "" {
		t.Error("Preset name should not be empty")
	}
	if fullPreset.Description == "" {
		t.Error("Preset description should not be empty")
	}
	if fullPreset.IsDefault == false {
		t.Error("full preset should be marked as default")
	}
	if fullPreset.IsActive == true {
		t.Error("Presets should start inactive")
	}
	if fullPreset.Pattern != "*" {
		t.Error("full preset should have wildcard pattern")
	}
}

func TestPresetHasCorrectTags(t *testing.T) {
	orch := NewOrchestrator()
	initializeDefaultPresets(orch)

	securityPreset, exists := orch.presets["security"]
	if !exists {
		t.Fatal("security preset should exist")
	}

	// Should have security tag
	if len(securityPreset.Tags) == 0 {
		t.Error("security preset should have tags")
	}

	hasSecurityTag := false
	for _, tag := range securityPreset.Tags {
		if tag == "security" {
			hasSecurityTag = true
			break
		}
	}

	if !hasSecurityTag {
		t.Error("security preset should have 'security' tag")
	}
}

func TestPresetStatesInitialized(t *testing.T) {
	orch := NewOrchestrator()
	initializeDefaultPresets(orch)

	for presetID, preset := range orch.presets {
		if preset.States == nil {
			t.Errorf("Preset '%s' should have States map initialized", presetID)
		}
	}
}
