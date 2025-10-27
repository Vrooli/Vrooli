package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ProtectedScenariosStore manages a persistent list of scenarios that should be
// excluded by default from rule testing and issue reporting operations.
// This allows users to protect critical scenarios (like ecosystem-manager,
// app-issue-tracker) from being modified during bulk operations.
type ProtectedScenariosStore struct {
	mu               sync.RWMutex
	protectedSet     map[string]bool // scenario name -> true
	filePath         string
	persistenceReady bool
}

type protectedScenariosData struct {
	Scenarios  []string  `json:"scenarios"`
	LastUpdate time.Time `json:"last_update"`
}

var protectedScenariosStore = initProtectedScenariosStore()

func initProtectedScenariosStore() *ProtectedScenariosStore {
	fmt.Fprintf(os.Stderr, "[INIT] Initializing protected scenarios store...\n")
	store := &ProtectedScenariosStore{
		protectedSet: make(map[string]bool),
	}
	store.enablePersistence()
	return store
}

func (ps *ProtectedScenariosStore) enablePersistence() {
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		vrooliRoot = filepath.Join(os.Getenv("HOME"), "Vrooli")
	}

	parentDir := filepath.Join(vrooliRoot, ".vrooli", "data")
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			logger.Error(fmt.Sprintf("Failed to create parent data directory %s", parentDir), err)
			logger.Info("Protected scenarios store will operate in memory-only mode (no persistence)")
			return
		}
	}

	dataDir := filepath.Join(parentDir, "scenario-auditor")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		logger.Error(fmt.Sprintf("Failed to create scenario-auditor data directory %s", dataDir), err)
		logger.Info("Protected scenarios store will operate in memory-only mode (no persistence)")
		return
	}

	ps.filePath = filepath.Join(dataDir, "protected-scenarios.json")
	ps.persistenceReady = true

	if err := ps.loadFromFile(); err != nil {
		logger.Error(fmt.Sprintf("Failed to load existing protected scenarios from %s", ps.filePath), err)
	} else {
		if len(ps.protectedSet) > 0 {
			logger.Info(fmt.Sprintf("Loaded %d protected scenarios", len(ps.protectedSet)))
		}
	}

	logger.Info(fmt.Sprintf("Protected scenarios persistence enabled at: %s", ps.filePath))
}

func (ps *ProtectedScenariosStore) loadFromFile() error {
	if !ps.persistenceReady || ps.filePath == "" {
		return nil
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	data, err := os.ReadFile(ps.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet - that's okay for first run
			return nil
		}
		return err
	}

	var stored protectedScenariosData
	if err := json.Unmarshal(data, &stored); err != nil {
		return err
	}

	// Rebuild the set from the stored list
	ps.protectedSet = make(map[string]bool)
	for _, scenario := range stored.Scenarios {
		if scenario != "" {
			ps.protectedSet[scenario] = true
		}
	}

	return nil
}

func (ps *ProtectedScenariosStore) saveToFileLocked() error {
	if !ps.persistenceReady || ps.filePath == "" {
		return nil
	}

	// Convert set to sorted list for consistent JSON output
	scenarios := make([]string, 0, len(ps.protectedSet))
	for scenario := range ps.protectedSet {
		scenarios = append(scenarios, scenario)
	}

	payload := protectedScenariosData{
		Scenarios:  scenarios,
		LastUpdate: time.Now(),
	}

	bytes, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := ps.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, bytes, 0644); err != nil {
		return err
	}

	return os.Rename(tmpPath, ps.filePath)
}

// GetProtectedScenarios returns the current list of protected scenarios
func (ps *ProtectedScenariosStore) GetProtectedScenarios() []string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	scenarios := make([]string, 0, len(ps.protectedSet))
	for scenario := range ps.protectedSet {
		scenarios = append(scenarios, scenario)
	}
	return scenarios
}

// IsProtected checks if a scenario is in the protected list
func (ps *ProtectedScenariosStore) IsProtected(scenarioName string) bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	return ps.protectedSet[scenarioName]
}

// SetProtectedScenarios replaces the entire protected scenarios list
func (ps *ProtectedScenariosStore) SetProtectedScenarios(scenarios []string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Rebuild the set
	ps.protectedSet = make(map[string]bool)
	for _, scenario := range scenarios {
		if scenario != "" {
			ps.protectedSet[scenario] = true
		}
	}

	return ps.saveToFileLocked()
}

// AddProtectedScenario adds a single scenario to the protected list
func (ps *ProtectedScenariosStore) AddProtectedScenario(scenarioName string) error {
	if scenarioName == "" {
		return fmt.Errorf("scenario name is required")
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.protectedSet[scenarioName] = true
	return ps.saveToFileLocked()
}

// RemoveProtectedScenario removes a single scenario from the protected list
func (ps *ProtectedScenariosStore) RemoveProtectedScenario(scenarioName string) error {
	if scenarioName == "" {
		return fmt.Errorf("scenario name is required")
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	delete(ps.protectedSet, scenarioName)
	return ps.saveToFileLocked()
}
