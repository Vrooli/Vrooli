package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// StandardsStore provides persistent storage for standards violations
type StandardsStore struct {
	mu         sync.RWMutex
	violations map[string][]StandardsViolation // key is scenario name
	lastCheck  map[string]time.Time            // key is scenario name
	filePath   string                          // path to persistent storage file
}

// StandardsData represents the persistent data structure
type StandardsData struct {
	Violations map[string][]StandardsViolation `json:"violations"`
	LastCheck  map[string]time.Time            `json:"last_check"`
	LastUpdate time.Time                       `json:"last_update"`
}

var standardsStore = initStandardsStore()

func initStandardsStore() *StandardsStore {
	fmt.Fprintf(os.Stderr, "[INIT] Initializing standards store...\n")
	store := &StandardsStore{
		violations: make(map[string][]StandardsViolation),
		lastCheck:  make(map[string]time.Time),
	}

	// Try to enable persistence, but don't fail if we can't
	fmt.Fprintf(os.Stderr, "[INIT] Enabling standards store persistence...\n")
	store.enablePersistence()
	fmt.Fprintf(os.Stderr, "[INIT] Standards store initialized\n")

	return store
}

// enablePersistence attempts to enable file-based persistence
func (ss *StandardsStore) enablePersistence() {

	// Get Vrooli root directory
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		vrooliRoot = os.Getenv("HOME") + "/Vrooli"
	}

	// Create data directory if it doesn't exist
	dataDir := filepath.Join(vrooliRoot, ".vrooli", "data", "scenario-auditor")

	// Check if parent directory exists first
	parentDir := filepath.Join(vrooliRoot, ".vrooli", "data")
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		// Try to create parent directory structure
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			logger.Error(fmt.Sprintf("Failed to create parent data directory %s", parentDir), err)
			logger.Info("Standards store will operate in memory-only mode (no persistence)")
			return
		}
	}

	// Now create our specific directory
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		logger.Error(fmt.Sprintf("Failed to create scenario-auditor data directory %s", dataDir), err)
		logger.Info("Standards store will operate in memory-only mode (no persistence)")
		return
	}

	// Set the file path
	ss.filePath = filepath.Join(dataDir, "standards-violations.json")

	// Try to load existing data
	if err := ss.loadFromFile(); err != nil {
		logger.Error(fmt.Sprintf("Failed to load existing standards data from %s", ss.filePath), err)
		// Continue anyway - we can still save new data
	} else {
		count := 0
		for _, violations := range ss.violations {
			count += len(violations)
		}
		if count > 0 {
			logger.Info(fmt.Sprintf("Loaded %d existing standards violations from persistent storage", count))
		}
	}

	logger.Info(fmt.Sprintf("Standards store persistence enabled at: %s", ss.filePath))
}

// StoreViolations stores standards check results in memory and optionally to disk
func (ss *StandardsStore) StoreViolations(scenarioName string, violations []StandardsViolation) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	normalised := strings.TrimSpace(scenarioName)
	if normalised == "" {
		normalised = "all"
	}

	now := time.Now()

	if normalised == "all" {
		// Rebuild aggregated results when scanning everything so the fix pipeline
		// can access scenario-specific violations even if the scan was global.
		bucketed := make(map[string][]StandardsViolation)
		for _, violation := range violations {
			scenario := strings.TrimSpace(violation.ScenarioName)
			if scenario == "" || scenario == "unknown" {
				scenario = extractScenarioName(violation.FilePath)
			}
			scenario = strings.TrimSpace(scenario)
			if scenario == "" {
				scenario = "unknown"
			}
			bucketed[scenario] = append(bucketed[scenario], violation)
		}

		// Start fresh to avoid retaining stale results for scenarios that are now clean.
		ss.violations = make(map[string][]StandardsViolation, len(bucketed)+1)
		ss.lastCheck = make(map[string]time.Time, len(bucketed)+1)

		ss.violations["all"] = append([]StandardsViolation(nil), violations...)
		ss.lastCheck["all"] = now

		for scenario, list := range bucketed {
			ss.violations[scenario] = append([]StandardsViolation(nil), list...)
			ss.lastCheck[scenario] = now
		}
	} else {
		// Clear existing violations for this scenario before storing new ones
		ss.violations[normalised] = []StandardsViolation{}
		ss.violations[normalised] = append(ss.violations[normalised], violations...)
		ss.lastCheck[normalised] = now

		// Keep the aggregated "all" view in sync when a targeted scan runs.
		if len(violations) > 0 {
			// Ensure we don't accidentally append duplicates when multiple targeted scans run.
			rest := ss.violations["all"]
			var filtered []StandardsViolation
			for _, v := range rest {
				if strings.TrimSpace(v.ScenarioName) == normalised {
					continue
				}
				filtered = append(filtered, v)
			}
			ss.violations["all"] = append(filtered, append([]StandardsViolation(nil), violations...)...)
			ss.lastCheck["all"] = now
		} else {
			// Remove any stale entries for this scenario from the aggregated list
			var filtered []StandardsViolation
			for _, v := range ss.violations["all"] {
				if strings.TrimSpace(v.ScenarioName) == normalised {
					continue
				}
				filtered = append(filtered, v)
			}
			ss.violations["all"] = filtered
			if len(filtered) == 0 {
				delete(ss.lastCheck, "all")
			} else {
				ss.lastCheck["all"] = now
			}
		}
	}

	// Save to file if persistence is enabled
	if ss.filePath != "" {
		if err := ss.saveToFile(); err != nil {
			logger.Error("Failed to persist standards violations to disk", err)
			// Continue anyway - in-memory storage still works
		}
	}
}

// GetViolations retrieves violations from memory
func (ss *StandardsStore) GetViolations(scenarioName string) []StandardsViolation {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	if scenarioName == "" || scenarioName == "all" {
		if aggregate, exists := ss.violations["all"]; exists {
			return append([]StandardsViolation(nil), aggregate...)
		}

		var allViolations []StandardsViolation
		for scenario, violations := range ss.violations {
			if scenario == "all" {
				continue
			}
			allViolations = append(allViolations, violations...)
		}
		return allViolations
	}

	if violations, exists := ss.violations[scenarioName]; exists {
		return append([]StandardsViolation(nil), violations...)
	}

	return nil
}

// ListScenarios returns the list of scenarios that currently have stored violations.
func (ss *StandardsStore) ListScenarios() []string {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	if len(ss.violations) == 0 {
		return nil
	}

	names := make([]string, 0, len(ss.violations))
	for scenario := range ss.violations {
		if scenario == "all" {
			continue
		}
		names = append(names, scenario)
	}
	sort.Strings(names)
	return names
}

// GetAllViolations returns all stored violations
func (ss *StandardsStore) GetAllViolations() []StandardsViolation {
	return ss.GetViolations("")
}

// GetLastCheckTime returns when a scenario was last checked
func (ss *StandardsStore) GetLastCheckTime(scenarioName string) *time.Time {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	if t, exists := ss.lastCheck[scenarioName]; exists {
		return &t
	}
	return nil
}

// ClearViolations clears violations for a scenario
func (ss *StandardsStore) ClearViolations(scenarioName string) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	delete(ss.violations, scenarioName)
	delete(ss.lastCheck, scenarioName)

	// Save to file if persistence is enabled
	if ss.filePath != "" {
		if err := ss.saveToFile(); err != nil {
			logger.Error("Failed to persist cleared violations to disk", err)
		}
	}
}

// GetStats returns violation statistics
func (ss *StandardsStore) GetStats() map[string]any {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	stats := map[string]any{
		"total":                     0,
		"critical":                  0,
		"high":                      0,
		"medium":                    0,
		"low":                       0,
		"scenarios_with_violations": 0,
	}

	scenariosWithViolations := make(map[string]bool)

	for scenario, violations := range ss.violations {
		if len(violations) > 0 {
			scenariosWithViolations[scenario] = true
		}
		for _, violation := range violations {
			stats["total"] = stats["total"].(int) + 1
			switch violation.Severity {
			case "critical":
				stats["critical"] = stats["critical"].(int) + 1
			case "high":
				stats["high"] = stats["high"].(int) + 1
			case "medium":
				stats["medium"] = stats["medium"].(int) + 1
			case "low":
				stats["low"] = stats["low"].(int) + 1
			}
		}
	}

	stats["scenarios_with_violations"] = len(scenariosWithViolations)
	return stats
}

// saveToFile persists the current data to disk
func (ss *StandardsStore) saveToFile() error {
	// Skip persistence if no file path (in-memory mode)
	if ss.filePath == "" {
		return nil
	}

	data := StandardsData{
		Violations: ss.violations,
		LastCheck:  ss.lastCheck,
		LastUpdate: time.Now(),
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal standards data: %w", err)
	}

	if err := os.WriteFile(ss.filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write standards data to %s: %w", ss.filePath, err)
	}

	return nil
}

// loadFromFile loads persisted data from disk
func (ss *StandardsStore) loadFromFile() error {
	// Skip loading if no file path (in-memory mode)
	if ss.filePath == "" {
		return nil
	}

	data, err := os.ReadFile(ss.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, that's okay
			return nil
		}
		return fmt.Errorf("failed to read standards data from %s: %w", ss.filePath, err)
	}

	var standardsData StandardsData
	if err := json.Unmarshal(data, &standardsData); err != nil {
		return fmt.Errorf("failed to unmarshal standards data from %s: %w", ss.filePath, err)
	}

	// Load the data
	ss.violations = standardsData.Violations
	ss.lastCheck = standardsData.LastCheck

	// Initialize maps if they're nil
	if ss.violations == nil {
		ss.violations = make(map[string][]StandardsViolation)
	}
	if ss.lastCheck == nil {
		ss.lastCheck = make(map[string]time.Time)
	}

	return nil
}
