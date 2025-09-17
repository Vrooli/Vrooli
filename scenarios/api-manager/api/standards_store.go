package main

import (
	"sync"
	"time"
)

// StandardsStore provides in-memory storage for standards violations
type StandardsStore struct {
	mu         sync.RWMutex
	violations map[string][]StandardsViolation // key is scenario name
	lastCheck  map[string]time.Time            // key is scenario name
}

var standardsStore = &StandardsStore{
	violations: make(map[string][]StandardsViolation),
	lastCheck:  make(map[string]time.Time),
}

// StoreViolations stores standards check results in memory
func (ss *StandardsStore) StoreViolations(scenarioName string, violations []StandardsViolation) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	
	// Clear existing violations for this scenario
	ss.violations[scenarioName] = []StandardsViolation{}
	
	// Store new violations
	ss.violations[scenarioName] = append(ss.violations[scenarioName], violations...)
	ss.lastCheck[scenarioName] = time.Now()
}

// GetViolations retrieves violations from memory
func (ss *StandardsStore) GetViolations(scenarioName string) []StandardsViolation {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	
	if scenarioName == "" || scenarioName == "all" {
		// Return all violations
		var allViolations []StandardsViolation
		for _, violations := range ss.violations {
			allViolations = append(allViolations, violations...)
		}
		return allViolations
	}
	
	return ss.violations[scenarioName]
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
}

// GetStats returns violation statistics
func (ss *StandardsStore) GetStats() map[string]interface{} {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	
	stats := map[string]interface{}{
		"total":    0,
		"critical": 0,
		"high":     0,
		"medium":   0,
		"low":      0,
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