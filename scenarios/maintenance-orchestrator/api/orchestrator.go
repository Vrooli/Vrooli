package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type Orchestrator struct {
	scenarios   map[string]*MaintenanceScenario
	presets     map[string]*Preset
	mu          sync.RWMutex
	activityLog []ActivityEntry
}

func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		scenarios:   make(map[string]*MaintenanceScenario),
		presets:     make(map[string]*Preset),
		activityLog: make([]ActivityEntry, 0),
	}
}

func (o *Orchestrator) AddScenario(scenario *MaintenanceScenario) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.scenarios[scenario.ID] = scenario
}

func (o *Orchestrator) GetScenarios() []*MaintenanceScenario {
	o.mu.RLock()
	defer o.mu.RUnlock()

	scenarios := make([]*MaintenanceScenario, 0, len(o.scenarios))
	for _, scenario := range o.scenarios {
		scenarios = append(scenarios, scenario)
	}
	return scenarios
}

func (o *Orchestrator) GetScenario(id string) (*MaintenanceScenario, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	scenario, exists := o.scenarios[id]
	return scenario, exists
}

func (o *Orchestrator) ActivateScenario(id string) bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	scenario, exists := o.scenarios[id]
	if !exists {
		return false
	}

	scenario.IsActive = true
	now := time.Now()
	scenario.LastActive = &now

	o.activityLog = append(o.activityLog, ActivityEntry{
		Timestamp: now,
		Action:    "activate",
		Scenario:  id,
		Success:   true,
	})

	return true
}

// UpdateResourceUsage updates resource usage metrics for a scenario
func (o *Orchestrator) UpdateResourceUsage(id string, usage map[string]float64) {
	o.mu.Lock()
	defer o.mu.Unlock()

	scenario, exists := o.scenarios[id]
	if exists {
		scenario.ResourceUsage = usage
	}
}

func (o *Orchestrator) DeactivateScenario(id string) bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	scenario, exists := o.scenarios[id]
	if !exists {
		return false
	}

	scenario.IsActive = false

	o.activityLog = append(o.activityLog, ActivityEntry{
		Timestamp: time.Now(),
		Action:    "deactivate",
		Scenario:  id,
		Success:   true,
	})

	return true
}

func (o *Orchestrator) GetPresets() []*Preset {
	o.mu.RLock()
	defer o.mu.RUnlock()

	presets := make([]*Preset, 0, len(o.presets))
	for _, preset := range o.presets {
		presets = append(presets, preset)
	}
	return presets
}

func (o *Orchestrator) GetPreset(id string) (*Preset, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	preset, exists := o.presets[id]
	return preset, exists
}

func (o *Orchestrator) ApplyPreset(presetID string) ([]string, []string, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()

	preset, exists := o.presets[presetID]
	if !exists {
		return nil, nil, false
	}

	// Toggle preset active state
	preset.IsActive = !preset.IsActive

	// Apply the union of all active presets to scenarios
	activated, deactivated := o.applyPresetUnion()

	action := "activate-preset"
	if !preset.IsActive {
		action = "deactivate-preset"
	}

	o.activityLog = append(o.activityLog, ActivityEntry{
		Timestamp: time.Now(),
		Action:    action,
		Preset:    presetID,
		Success:   true,
		Message: fmt.Sprintf("Preset %s, Scenarios changed: %d activated, %d deactivated",
			map[bool]string{true: "activated", false: "deactivated"}[preset.IsActive],
			len(activated), len(deactivated)),
	})

	return activated, deactivated, true
}

func (o *Orchestrator) StopAll() []string {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Deactivate all presets first
	for _, preset := range o.presets {
		preset.IsActive = false
	}

	// Deactivate all scenarios
	deactivated := make([]string, 0)
	for scenarioID, scenario := range o.scenarios {
		if scenario.IsActive {
			scenario.IsActive = false
			deactivated = append(deactivated, scenarioID)
		}
	}

	o.activityLog = append(o.activityLog, ActivityEntry{
		Timestamp: time.Now(),
		Action:    "stop-all",
		Success:   true,
		Message:   fmt.Sprintf("Deactivated all presets and %d scenarios", len(deactivated)),
	})

	return deactivated
}

func (o *Orchestrator) GetStatus() (int, int, []ActivityEntry) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	activeCount := 0
	for _, scenario := range o.scenarios {
		if scenario.IsActive {
			activeCount++
		}
	}

	recentActivity := o.activityLog
	if len(recentActivity) > 10 {
		recentActivity = o.activityLog[len(o.activityLog)-10:]
	}

	return len(o.scenarios), activeCount, recentActivity
}

func (o *Orchestrator) UpdatePresetStates() {
	o.mu.Lock()
	defer o.mu.Unlock()

	for _, preset := range o.presets {
		for scenarioID, scenario := range o.scenarios {
			if preset.Pattern == "*" {
				preset.States[scenarioID] = true
			} else if len(preset.Tags) > 0 {
				for _, presetTag := range preset.Tags {
					for _, scenarioTag := range scenario.Tags {
						if presetTag == scenarioTag {
							preset.States[scenarioID] = true
							break
						}
					}
				}
			}
		}
	}
}

// applyPresetUnion calculates the union of all active presets and applies the result to scenarios
// This method assumes the mutex is already locked
func (o *Orchestrator) applyPresetUnion() ([]string, []string) {
	// Calculate which scenarios should be active based on the union of all active presets
	shouldBeActive := make(map[string]bool)

	for _, preset := range o.presets {
		if preset.IsActive {
			for scenarioID, active := range preset.States {
				if active {
					shouldBeActive[scenarioID] = true
				}
			}
		}
	}

	activated := make([]string, 0)
	deactivated := make([]string, 0)

	// Update scenarios based on union
	for scenarioID, scenario := range o.scenarios {
		if shouldBeActive[scenarioID] && !scenario.IsActive {
			scenario.IsActive = true
			now := time.Now()
			scenario.LastActive = &now
			activated = append(activated, scenarioID)
		} else if !shouldBeActive[scenarioID] && scenario.IsActive {
			scenario.IsActive = false
			deactivated = append(deactivated, scenarioID)
		}
	}

	return activated, deactivated
}

// GetActivePresets returns a list of currently active presets
func (o *Orchestrator) GetActivePresets() []*Preset {
	o.mu.RLock()
	defer o.mu.RUnlock()

	activePresets := make([]*Preset, 0)
	for _, preset := range o.presets {
		if preset.IsActive {
			activePresets = append(activePresets, preset)
		}
	}
	return activePresets
}

// CreatePreset adds a new custom preset based on provided parameters
func (o *Orchestrator) CreatePreset(name, description string, states map[string]bool, tags []string) (*Preset, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Validate name is unique
	for _, existing := range o.presets {
		if existing.Name == name {
			return nil, fmt.Errorf("preset with name '%s' already exists", name)
		}
	}

	// Generate ID from name (lowercase, replace spaces with dashes)
	id := strings.ToLower(strings.ReplaceAll(name, " ", "-"))

	// Validate states reference real scenarios
	for scenarioID := range states {
		if _, exists := o.scenarios[scenarioID]; !exists {
			return nil, fmt.Errorf("scenario '%s' not found", scenarioID)
		}
	}

	preset := &Preset{
		ID:          id,
		Name:        name,
		Description: description,
		States:      states,
		Tags:        tags,
		IsDefault:   false,
		IsActive:    false,
	}

	o.presets[id] = preset

	o.activityLog = append(o.activityLog, ActivityEntry{
		Timestamp: time.Now(),
		Action:    "create-preset",
		Preset:    name,
		Success:   true,
		Message:   fmt.Sprintf("Created custom preset '%s' with %d scenarios", name, len(states)),
	})

	return preset, nil
}

// CreatePresetFromCurrentState creates a preset capturing the current state of all scenarios
func (o *Orchestrator) CreatePresetFromCurrentState(name, description string) (*Preset, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Validate name is unique
	for _, existing := range o.presets {
		if existing.Name == name {
			return nil, fmt.Errorf("preset with name '%s' already exists", name)
		}
	}

	// Generate ID from name
	id := strings.ToLower(strings.ReplaceAll(name, " ", "-"))

	// Capture current state
	states := make(map[string]bool)
	for scenarioID, scenario := range o.scenarios {
		states[scenarioID] = scenario.IsActive
	}

	preset := &Preset{
		ID:          id,
		Name:        name,
		Description: description,
		States:      states,
		IsDefault:   false,
		IsActive:    false,
	}

	o.presets[id] = preset

	o.activityLog = append(o.activityLog, ActivityEntry{
		Timestamp: time.Now(),
		Action:    "create-preset-from-state",
		Preset:    name,
		Success:   true,
		Message:   fmt.Sprintf("Created preset '%s' from current state (%d scenarios)", name, len(states)),
	})

	return preset, nil
}

// IsPresetActive checks if a preset is currently active
func (o *Orchestrator) IsPresetActive(presetID string) bool {
	o.mu.RLock()
	defer o.mu.RUnlock()

	preset, exists := o.presets[presetID]
	if !exists {
		return false
	}
	return preset.IsActive
}
