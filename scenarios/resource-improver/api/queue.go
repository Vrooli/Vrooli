package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// QueueItemYAML represents the YAML structure of queue items
type QueueItemYAML struct {
	ID          string            `yaml:"id"`
	Title       string            `yaml:"title"`
	Description string            `yaml:"description"`
	Type        string            `yaml:"type"`
	Target      string            `yaml:"target"`
	Priority    string            `yaml:"priority"`
	Estimates   PriorityEstimates `yaml:"priority_estimates"`
	Requirements []string         `yaml:"requirements"`
	ValidationCriteria []string   `yaml:"validation_criteria"`
	CrossScenario CrossScenarioInfo `yaml:"cross_scenario"`
	MemoryContext string           `yaml:"memory_context,omitempty"`
	Metadata    QueueMetadata     `yaml:"metadata"`
}

type PriorityEstimates struct {
	Impact       int     `yaml:"impact"`
	Urgency      string  `yaml:"urgency"`
	SuccessProb  float64 `yaml:"success_prob"`
	ResourceCost string  `yaml:"resource_cost"`
}

type CrossScenarioInfo struct {
	AffectedScenarios []string `yaml:"affected_scenarios"`
	SharedResources   []string `yaml:"shared_resources"`
	BreakingChanges   bool     `yaml:"breaking_changes"`
}

type QueueMetadata struct {
	CreatedBy       string    `yaml:"created_by"`
	CreatedAt       time.Time `yaml:"created_at"`
	CooldownUntil   time.Time `yaml:"cooldown_until"`
	AttemptCount    int       `yaml:"attempt_count"`
	LastAttempt     *time.Time `yaml:"last_attempt,omitempty"`
	FailureReasons  []string  `yaml:"failure_reasons"`
}

// Load queue item from YAML file
func loadQueueItem(filepath string) (*QueueItem, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	
	var yamlItem QueueItemYAML
	if err := yaml.Unmarshal(data, &yamlItem); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %v", err)
	}
	
	// Convert to internal structure
	item := &QueueItem{
		ID:          yamlItem.ID,
		Title:       yamlItem.Title,
		Description: yamlItem.Description,
		Type:        yamlItem.Type,
		Target:      yamlItem.Target,
		Priority:    yamlItem.Priority,
		Estimates: map[string]interface{}{
			"impact":       yamlItem.Estimates.Impact,
			"urgency":      yamlItem.Estimates.Urgency,
			"success_prob": yamlItem.Estimates.SuccessProb,
			"resource_cost": yamlItem.Estimates.ResourceCost,
		},
		CreatedBy:    yamlItem.Metadata.CreatedBy,
		CreatedAt:    yamlItem.Metadata.CreatedAt,
		AttemptCount: yamlItem.Metadata.AttemptCount,
	}
	
	return item, nil
}

// Save queue item to YAML file
func saveQueueItem(item *QueueItem, status string) error {
	dir := filepath.Join(queueDir, status)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	// Convert to YAML structure
	yamlItem := QueueItemYAML{
		ID:          item.ID,
		Title:       item.Title,
		Description: item.Description,
		Type:        item.Type,
		Target:      item.Target,
		Priority:    item.Priority,
		Estimates: PriorityEstimates{
			Impact:       getIntFromEstimates(item.Estimates, "impact", 5),
			Urgency:      getStringFromEstimates(item.Estimates, "urgency", "medium"),
			SuccessProb:  getFloatFromEstimates(item.Estimates, "success_prob", 0.7),
			ResourceCost: getStringFromEstimates(item.Estimates, "resource_cost", "moderate"),
		},
		Requirements:       []string{"Auto-generated improvement task"},
		ValidationCriteria: []string{"Task completed successfully", "All tests pass"},
		CrossScenario: CrossScenarioInfo{
			AffectedScenarios: []string{},
			SharedResources:   []string{},
			BreakingChanges:   false,
		},
		Metadata: QueueMetadata{
			CreatedBy:      item.CreatedBy,
			CreatedAt:      item.CreatedAt,
			CooldownUntil:  time.Now().Add(1 * time.Hour), // 1 hour default cooldown
			AttemptCount:   item.AttemptCount,
			FailureReasons: []string{},
		},
	}
	
	data, err := yaml.Marshal(yamlItem)
	if err != nil {
		return err
	}
	
	filename := fmt.Sprintf("%s.yaml", item.ID)
	filepath := filepath.Join(dir, filename)
	
	return os.WriteFile(filepath, data, 0644)
}

// Move queue item between directories
func moveQueueItem(id, from, to string) error {
	fromDir := filepath.Join(queueDir, from)
	toDir := filepath.Join(queueDir, to)
	
	// Find the file in the source directory
	files, err := filepath.Glob(filepath.Join(fromDir, fmt.Sprintf("%s*.yaml", id)))
	if err != nil {
		return err
	}
	
	if len(files) == 0 {
		return fmt.Errorf("queue item %s not found in %s", id, from)
	}
	
	// Ensure target directory exists
	if err := os.MkdirAll(toDir, 0755); err != nil {
		return err
	}
	
	// Move the file
	sourceFile := files[0]
	filename := filepath.Base(sourceFile)
	targetFile := filepath.Join(toDir, filename)
	
	// Read, write, and delete (to handle cross-filesystem moves)
	data, err := os.ReadFile(sourceFile)
	if err != nil {
		return err
	}
	
	if err := os.WriteFile(targetFile, data, 0644); err != nil {
		return err
	}
	
	return os.Remove(sourceFile)
}

// Count queue items in a status directory
func countQueueItems(status string) int {
	dir := filepath.Join(queueDir, status)
	files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return 0
	}
	return len(files)
}

// Calculate priority score using the formula from the prompt
func calculatePriorityScore(item *QueueItem) float64 {
	impact := getFloatFromEstimates(item.Estimates, "impact", 5.0)
	urgency := urgencyToScore(getStringFromEstimates(item.Estimates, "urgency", "medium"))
	successProb := getFloatFromEstimates(item.Estimates, "success_prob", 0.7)
	resourceCost := resourceCostToScore(getStringFromEstimates(item.Estimates, "resource_cost", "moderate"))
	
	// priority = (impact * 2 + urgency * 1.5) * success_prob / resource_cost
	score := (impact*2 + urgency*1.5) * successProb / resourceCost
	
	return score
}

// Helper functions for type conversion
func getIntFromEstimates(estimates map[string]interface{}, key string, defaultValue int) int {
	if val, ok := estimates[key]; ok {
		if intVal, ok := val.(int); ok {
			return intVal
		}
		if floatVal, ok := val.(float64); ok {
			return int(floatVal)
		}
		if strVal, ok := val.(string); ok {
			if intVal, err := strconv.Atoi(strVal); err == nil {
				return intVal
			}
		}
	}
	return defaultValue
}

func getFloatFromEstimates(estimates map[string]interface{}, key string, defaultValue float64) float64 {
	if val, ok := estimates[key]; ok {
		if floatVal, ok := val.(float64); ok {
			return floatVal
		}
		if intVal, ok := val.(int); ok {
			return float64(intVal)
		}
		if strVal, ok := val.(string); ok {
			if floatVal, err := strconv.ParseFloat(strVal, 64); err == nil {
				return floatVal
			}
		}
	}
	return defaultValue
}

func getStringFromEstimates(estimates map[string]interface{}, key string, defaultValue string) string {
	if val, ok := estimates[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return defaultValue
}

// Convert urgency string to numeric score
func urgencyToScore(urgency string) float64 {
	switch strings.ToLower(urgency) {
	case "critical":
		return 10.0
	case "high":
		return 7.0
	case "medium":
		return 5.0
	case "low":
		return 2.0
	default:
		return 5.0
	}
}

// Convert resource cost string to numeric score (higher cost = lower multiplier)
func resourceCostToScore(cost string) float64 {
	switch strings.ToLower(cost) {
	case "minimal":
		return 1.0
	case "moderate":
		return 2.0
	case "heavy":
		return 4.0
	default:
		return 2.0
	}
}

// Log queue events for monitoring
func logQueueEvent(eventType, itemID, detail string) {
	eventsFile := filepath.Join(queueDir, "events.ndjson")
	
	event := map[string]interface{}{
		"type":   eventType,
		"ts":     time.Now().Format(time.RFC3339),
		"item":   itemID,
		"detail": detail,
	}
	
	// Append to events file
	f, err := os.OpenFile(eventsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return // Ignore logging errors
	}
	defer f.Close()
	
	if data, err := json.Marshal(event); err == nil {
		f.Write(data)
		f.WriteString("\n")
	}
}

// Check if item is in cooldown period
func isInCooldown(item *QueueItem) bool {
	// Load the full YAML to check cooldown
	files, err := filepath.Glob(filepath.Join(queueDir, "pending", fmt.Sprintf("%s*.yaml", item.ID)))
	if err != nil || len(files) == 0 {
		return false
	}
	
	var yamlItem QueueItemYAML
	data, err := os.ReadFile(files[0])
	if err != nil {
		return false
	}
	
	if err := yaml.Unmarshal(data, &yamlItem); err != nil {
		return false
	}
	
	return time.Now().Before(yamlItem.Metadata.CooldownUntil)
}

// Copy file (utility function)
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()
	
	_, err = io.Copy(destFile, sourceFile)
	return err
}