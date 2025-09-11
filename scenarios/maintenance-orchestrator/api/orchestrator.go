package main

import (
	"fmt"
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
	
	activated := make([]string, 0)
	deactivated := make([]string, 0)
	
	for scenarioID, shouldBeActive := range preset.States {
		scenario, exists := o.scenarios[scenarioID]
		if !exists {
			continue
		}
		
		if shouldBeActive && !scenario.IsActive {
			scenario.IsActive = true
			now := time.Now()
			scenario.LastActive = &now
			activated = append(activated, scenarioID)
		} else if !shouldBeActive && scenario.IsActive {
			scenario.IsActive = false
			deactivated = append(deactivated, scenarioID)
		}
	}
	
	o.activityLog = append(o.activityLog, ActivityEntry{
		Timestamp: time.Now(),
		Action:    "apply-preset",
		Preset:    presetID,
		Success:   true,
		Message:   fmt.Sprintf("Activated: %d, Deactivated: %d", len(activated), len(deactivated)),
	})
	
	return activated, deactivated, true
}

func (o *Orchestrator) StopAll() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	
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
		Message:   fmt.Sprintf("Deactivated %d scenarios", len(deactivated)),
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