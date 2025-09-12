package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Settings represents the system monitor settings
type Settings struct {
	// Monitor activation status
	Active bool `json:"active"`
	
	// Monitoring intervals (in seconds)
	MetricCollectionInterval int `json:"metric_collection_interval"`
	AnomalyDetectionInterval int `json:"anomaly_detection_interval"`
	ThresholdCheckInterval   int `json:"threshold_check_interval"`
	
	// Investigation settings
	CooldownPeriodSeconds int `json:"cooldown_period_seconds"`
	
	// System thresholds
	CPUThreshold    float64 `json:"cpu_threshold"`
	MemoryThreshold float64 `json:"memory_threshold"`
	DiskThreshold   float64 `json:"disk_threshold"`
}

// SettingsManager manages system monitor settings with thread safety
type SettingsManager struct {
	settings     Settings
	mutex        sync.RWMutex
	configPath   string
	onActiveChanged func(active bool) // Callback for when active status changes
}

// Default settings (always start inactive for safety)
var defaultSettings = Settings{
	Active:                   false, // ALWAYS start inactive for safety
	MetricCollectionInterval: 10,    // 10 seconds
	AnomalyDetectionInterval: 30,    // 30 seconds  
	ThresholdCheckInterval:   20,    // 20 seconds
	CooldownPeriodSeconds:    300,   // 5 minutes
	CPUThreshold:             85.0,  // 85%
	MemoryThreshold:          90.0,  // 90%
	DiskThreshold:            85.0,  // 85%
}

// NewSettingsManager creates a new settings manager
func NewSettingsManager() *SettingsManager {
	// Determine config file path
	configPath := filepath.Join(os.Getenv("VROOLI_ROOT"), "scenarios/system-monitor/initialization/configuration/system-monitor-settings.json")
	if configPath == "scenarios/system-monitor/initialization/configuration/system-monitor-settings.json" {
		configPath = filepath.Join(os.Getenv("HOME"), "Vrooli/scenarios/system-monitor/initialization/configuration/system-monitor-settings.json")
	}

	sm := &SettingsManager{
		settings:   defaultSettings,
		configPath: configPath,
	}
	
	// Try to load existing settings, but if it fails, use defaults
	if err := sm.loadFromFile(); err != nil {
		// Log the error but continue with defaults
		fmt.Printf("Warning: Could not load settings from file (%v), using defaults\n", err)
		// Save defaults to create the config file
		sm.saveToFile()
	}
	
	return sm
}

// GetSettings returns a copy of current settings (thread-safe)
func (sm *SettingsManager) GetSettings() Settings {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.settings
}

// UpdateSettings updates the settings and saves to file
func (sm *SettingsManager) UpdateSettings(newSettings Settings) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	// Check if active status is changing
	oldActive := sm.settings.Active
	newActive := newSettings.Active
	
	// Update settings
	sm.settings = newSettings
	
	// Save to file
	if err := sm.saveToFile(); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}
	
	// Call callback if active status changed
	if oldActive != newActive && sm.onActiveChanged != nil {
		// Call callback outside of mutex to prevent deadlock
		go sm.onActiveChanged(newActive)
	}
	
	return nil
}

// IsActive returns whether the monitor is currently active
func (sm *SettingsManager) IsActive() bool {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.settings.Active
}

// SetActive updates just the active status
func (sm *SettingsManager) SetActive(active bool) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	
	oldActive := sm.settings.Active
	sm.settings.Active = active
	
	// Save to file
	if err := sm.saveToFile(); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}
	
	// Call callback if status changed
	if oldActive != active && sm.onActiveChanged != nil {
		// Call callback outside of mutex to prevent deadlock
		go sm.onActiveChanged(active)
	}
	
	return nil
}

// ResetSettings resets to default settings (inactive)
func (sm *SettingsManager) ResetSettings() error {
	return sm.UpdateSettings(defaultSettings)
}

// SetActiveChangedCallback sets the callback function for when active status changes
func (sm *SettingsManager) SetActiveChangedCallback(callback func(active bool)) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.onActiveChanged = callback
}

// loadFromFile loads settings from JSON file
func (sm *SettingsManager) loadFromFile() error {
	data, err := ioutil.ReadFile(sm.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	
	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}
	
	sm.settings = settings
	return nil
}

// saveToFile saves current settings to JSON file
func (sm *SettingsManager) saveToFile() error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(sm.configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Create config with metadata
	config := map[string]interface{}{
		"version": "1.0.0",
		"metadata": map[string]interface{}{
			"last_modified":   time.Now().Format(time.RFC3339),
			"config_version":  "1.0.0",
			"description":     "System Monitor settings including active/inactive status",
		},
		"settings": sm.settings,
	}
	
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	if err := ioutil.WriteFile(sm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// GetMaintenanceState returns the maintenance state for external systems
func (sm *SettingsManager) GetMaintenanceState() string {
	if sm.IsActive() {
		return "active"
	}
	return "inactive"
}

// SetMaintenanceState sets the maintenance state (for external systems like maintenance-orchestrator)
func (sm *SettingsManager) SetMaintenanceState(state string) error {
	active := state == "active"
	return sm.SetActive(active)
}