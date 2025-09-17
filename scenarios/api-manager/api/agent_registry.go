package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Agent represents a running agent in the registry
type Agent struct {
	ID        string    `json:"id"`
	PID       int       `json:"pid"`
	Status    string    `json:"status"`    // "running", "stopped", "crashed"
	StartTime time.Time `json:"start_time"`
	LastSeen  time.Time `json:"last_seen"`
	Command   string    `json:"command"`
	Type      string    `json:"type,omitempty"`      // "claude-fix", "security-scan", etc.
	Scenario  string    `json:"scenario,omitempty"`  // Associated scenario
	LogFile   string    `json:"log_file,omitempty"`  // Path to log file
}

// AgentRegistry manages the agent registry
type AgentRegistry struct {
	Agents map[string]*Agent `json:"agents"`
	mutex  sync.RWMutex
}

var agentRegistry *AgentRegistry
var registryOnce sync.Once

// GetAgentRegistry returns the singleton agent registry
func GetAgentRegistry() *AgentRegistry {
	registryOnce.Do(func() {
		agentRegistry = &AgentRegistry{
			Agents: make(map[string]*Agent),
		}
		agentRegistry.loadFromFile()
	})
	return agentRegistry
}

// getRegistryFilePath returns the path to the agent registry file
func (ar *AgentRegistry) getRegistryFilePath() string {
	appRoot := os.Getenv("APP_ROOT")
	if appRoot == "" {
		appRoot = os.Getenv("VROOLI_ROOT")
		if appRoot == "" {
			appRoot = filepath.Join(os.Getenv("HOME"), "Vrooli")
		}
	}
	
	// Ensure .vrooli directory exists
	vrooliDir := filepath.Join(appRoot, ".vrooli")
	os.MkdirAll(vrooliDir, 0755)
	
	return filepath.Join(vrooliDir, "api-manager-agents.json")
}

// loadFromFile loads the agent registry from disk
func (ar *AgentRegistry) loadFromFile() error {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()
	
	filePath := ar.getRegistryFilePath()
	
	// If file doesn't exist, start with empty registry
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read registry file: %w", err)
	}
	
	var registryData struct {
		Agents map[string]*Agent `json:"agents"`
	}
	
	if err := json.Unmarshal(data, &registryData); err != nil {
		return fmt.Errorf("failed to parse registry file: %w", err)
	}
	
	ar.Agents = registryData.Agents
	if ar.Agents == nil {
		ar.Agents = make(map[string]*Agent)
	}
	
	// Clean up dead agents on load
	ar.cleanupDeadAgents()
	
	return nil
}

// saveToFile saves the agent registry to disk
func (ar *AgentRegistry) saveToFile() error {
	filePath := ar.getRegistryFilePath()
	
	data, err := json.MarshalIndent(ar, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}
	
	return os.WriteFile(filePath, data, 0644)
}

// RegisterAgent adds a new agent to the registry
func (ar *AgentRegistry) RegisterAgent(agent *Agent) error {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()
	
	if agent.ID == "" {
		return fmt.Errorf("agent ID cannot be empty")
	}
	
	// Set default values
	if agent.Status == "" {
		agent.Status = "running"
	}
	if agent.StartTime.IsZero() {
		agent.StartTime = time.Now()
	}
	agent.LastSeen = time.Now()
	
	ar.Agents[agent.ID] = agent
	
	return ar.saveToFile()
}

// UpdateAgent updates an existing agent's information
func (ar *AgentRegistry) UpdateAgent(agentID string, updates *Agent) error {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()
	
	agent, exists := ar.Agents[agentID]
	if !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}
	
	// Update fields if provided
	if updates.Status != "" {
		agent.Status = updates.Status
	}
	if updates.PID != 0 {
		agent.PID = updates.PID
	}
	if updates.Command != "" {
		agent.Command = updates.Command
	}
	if updates.LogFile != "" {
		agent.LogFile = updates.LogFile
	}
	
	agent.LastSeen = time.Now()
	
	return ar.saveToFile()
}

// Heartbeat updates the last seen time for an agent
func (ar *AgentRegistry) Heartbeat(agentID string) error {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()
	
	agent, exists := ar.Agents[agentID]
	if !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}
	
	agent.LastSeen = time.Now()
	
	return ar.saveToFile()
}

// RemoveAgent removes an agent from the registry
func (ar *AgentRegistry) RemoveAgent(agentID string) error {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()
	
	delete(ar.Agents, agentID)
	
	return ar.saveToFile()
}

// GetAgent retrieves a specific agent
func (ar *AgentRegistry) GetAgent(agentID string) (*Agent, bool) {
	ar.mutex.RLock()
	defer ar.mutex.RUnlock()
	
	agent, exists := ar.Agents[agentID]
	return agent, exists
}

// ListAgents returns all agents
func (ar *AgentRegistry) ListAgents() map[string]*Agent {
	ar.mutex.RLock()
	defer ar.mutex.RUnlock()
	
	// Return a copy to avoid race conditions
	result := make(map[string]*Agent)
	for id, agent := range ar.Agents {
		agentCopy := *agent
		result[id] = &agentCopy
	}
	
	return result
}

// cleanupDeadAgents removes agents whose processes are no longer running
func (ar *AgentRegistry) cleanupDeadAgents() {
	for agentID, agent := range ar.Agents {
		if agent.PID != 0 {
			// Check if process is still running
			if !isProcessRunning(agent.PID) {
				agent.Status = "stopped"
			}
		}
		
		// Remove agents that haven't been seen for more than 1 hour
		if time.Since(agent.LastSeen) > time.Hour {
			delete(ar.Agents, agentID)
		}
	}
}

// isProcessRunning checks if a process with the given PID is running
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	
	// Check if /proc/<pid> exists
	procPath := fmt.Sprintf("/proc/%d", pid)
	_, err := os.Stat(procPath)
	return err == nil
}

// StopAgent attempts to stop an agent
func (ar *AgentRegistry) StopAgent(agentID string) error {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()
	
	agent, exists := ar.Agents[agentID]
	if !exists {
		return fmt.Errorf("agent %s not found", agentID)
	}
	
	// If agent has a PID, try to kill it
	if agent.PID > 0 {
		process, err := os.FindProcess(agent.PID)
		if err != nil {
			// Process doesn't exist, mark as stopped
			agent.Status = "stopped"
		} else {
			// Try to terminate gracefully first
			if err := process.Signal(os.Interrupt); err != nil {
				// If that fails, force kill
				if err := process.Kill(); err != nil {
					return fmt.Errorf("failed to stop agent process: %w", err)
				}
			}
			agent.Status = "stopped"
		}
	} else {
		// No PID, just mark as stopped
		agent.Status = "stopped"
	}
	
	return ar.saveToFile()
}

// PeriodicCleanup should be called periodically to clean up dead agents
func (ar *AgentRegistry) PeriodicCleanup() {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()
	
	ar.cleanupDeadAgents()
	ar.saveToFile()
}