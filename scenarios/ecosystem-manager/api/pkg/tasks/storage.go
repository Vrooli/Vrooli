package tasks

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Storage handles file-based task persistence
// DESIGN DECISION: File-based task storage is intentional and provides several benefits:
// 1. Manual task editing - developers can directly edit .yaml files for debugging/testing
// 2. Git version control - task history and changes are tracked in source control
// 3. Multi-scenario collaboration - other scenarios can create/modify tasks via file system
// 4. Transparency - task state is always visible and inspectable without special tools
// 5. Atomic operations - file renames provide atomic status transitions
type Storage struct {
	QueueDir string
}

// NewStorage creates a new storage instance
func NewStorage(queueDir string) *Storage {
	return &Storage{
		QueueDir: queueDir,
	}
}

// convertToStringMap converts map[interface{}]interface{} to map[string]interface{}
// This is needed because YAML unmarshals to interface{} keys but JSON needs string keys
func convertToStringMap(m interface{}) interface{} {
	switch v := m.(type) {
	case map[interface{}]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			keyStr, ok := key.(string)
			if !ok {
				keyStr = fmt.Sprintf("%v", key)
			}
			result[keyStr] = convertToStringMap(value)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, value := range v {
			result[i] = convertToStringMap(value)
		}
		return result
	default:
		return v
	}
}

// GetQueueItems retrieves all tasks with the specified status
func (s *Storage) GetQueueItems(status string) ([]TaskItem, error) {
	queuePath := filepath.Join(s.QueueDir, status)
	
	// Debug logging
	log.Printf("Looking for tasks in: %s", queuePath)
	
	files, err := filepath.Glob(filepath.Join(queuePath, "*.yaml"))
	if err != nil {
		log.Printf("Error globbing files: %v", err)
		return nil, err
	}
	
	log.Printf("Found %d files in %s", len(files), status)
	
	var items []TaskItem
	for _, file := range files {
		log.Printf("Reading file: %s", file)
		data, err := os.ReadFile(file)
		if err != nil {
			log.Printf("Error reading file %s: %v", file, err)
			continue
		}
		
		var item TaskItem
		if err := yaml.Unmarshal(data, &item); err != nil {
			log.Printf("Error unmarshaling YAML from %s: %v", file, err)
			continue
		}
		
		// Convert interface{} maps to string maps for JSON compatibility
		if item.Requirements != nil {
			converted := convertToStringMap(item.Requirements)
			if m, ok := converted.(map[string]interface{}); ok {
				item.Requirements = m
			}
		}
		if item.Results != nil {
			converted := convertToStringMap(item.Results)
			if m, ok := converted.(map[string]interface{}); ok {
				item.Results = m
			}
		}
		// AssignedResources should be fine as map[string]bool with yaml.v3
		
		log.Printf("Successfully loaded task: %s", item.ID)
		items = append(items, item)
	}
	
	log.Printf("Returning %d items for status %s", len(items), status)
	return items, nil
}

// SaveQueueItem saves a task to the specified status directory
func (s *Storage) SaveQueueItem(item TaskItem, status string) error {
	queuePath := filepath.Join(s.QueueDir, status)
	filename := fmt.Sprintf("%s.yaml", item.ID)
	filePath := filepath.Join(queuePath, filename)
	
	data, err := yaml.Marshal(item)
	if err != nil {
		return err
	}
	
	return os.WriteFile(filePath, data, 0644)
}

// MoveQueueItem moves a task from one status to another (atomic rename)
func (s *Storage) MoveQueueItem(itemID, fromStatus, toStatus string) error {
	fromPath := filepath.Join(s.QueueDir, fromStatus, fmt.Sprintf("%s.yaml", itemID))
	toPath := filepath.Join(s.QueueDir, toStatus, fmt.Sprintf("%s.yaml", itemID))
	
	return os.Rename(fromPath, toPath)
}

// MoveTask moves a task between queue states and updates timestamps
func (s *Storage) MoveTask(taskID, fromStatus, toStatus string) error {
	// Read the task
	fromPath := filepath.Join(s.QueueDir, fromStatus, fmt.Sprintf("%s.yaml", taskID))
	data, err := os.ReadFile(fromPath)
	if err != nil {
		return fmt.Errorf("failed to read task from %s: %v", fromPath, err)
	}
	
	var task TaskItem
	if err := yaml.Unmarshal(data, &task); err != nil {
		return fmt.Errorf("failed to unmarshal task: %v", err)
	}
	
	// Update timestamps based on status change
	now := time.Now().Format(time.RFC3339)
	task.UpdatedAt = now
	
	switch toStatus {
	case "in-progress":
		task.StartedAt = now
		task.Status = "in-progress"
		task.CurrentPhase = "initialization"
	case "completed":
		task.CompletedAt = now
		task.Status = "completed"
		task.ProgressPercent = 100
	case "failed":
		task.CompletedAt = now
		task.Status = "failed"
	}
	
	// CRITICAL: Remove from old location FIRST to prevent duplicates
	// If this fails, we abort the whole operation
	if err := os.Remove(fromPath); err != nil {
		return fmt.Errorf("failed to remove task from %s: %v", fromStatus, err)
	}
	log.Printf("Deleted task %s from %s", taskID, fromStatus)
	
	// Now save to new location
	if err := s.SaveQueueItem(task, toStatus); err != nil {
		// Try to restore to old location since we already deleted it
		log.Printf("ERROR: Failed to save task %s to %s, attempting restore to %s", taskID, toStatus, fromStatus)
		if restoreErr := s.SaveQueueItem(task, fromStatus); restoreErr != nil {
			log.Printf("CRITICAL: Failed to restore task %s to %s: %v", taskID, fromStatus, restoreErr)
			return fmt.Errorf("failed to save task to %s and failed to restore: save error: %v, restore error: %v", toStatus, err, restoreErr)
		}
		return fmt.Errorf("failed to save task to %s (restored to %s): %v", toStatus, fromStatus, err)
	}
	
	log.Printf("Successfully moved task %s from %s to %s", taskID, fromStatus, toStatus)
	return nil
}

// GetTaskByID finds a task by ID across all queue statuses
func (s *Storage) GetTaskByID(taskID string) (*TaskItem, string, error) {
	statuses := []string{"pending", "in-progress", "review", "completed", "failed"}
	
	for _, status := range statuses {
		filePath := filepath.Join(s.QueueDir, status, fmt.Sprintf("%s.yaml", taskID))
		if _, err := os.Stat(filePath); err == nil {
			data, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}
			
			var task TaskItem
			if err := yaml.Unmarshal(data, &task); err != nil {
				continue
			}
			
			return &task, status, nil
		}
	}
	
	return nil, "", fmt.Errorf("task not found: %s", taskID)
}

// DeleteTask removes a task file from the appropriate status directory
func (s *Storage) DeleteTask(taskID string) (string, error) {
	statuses := []string{"pending", "in-progress", "review", "completed", "failed"}
	
	for _, status := range statuses {
		filePath := filepath.Join(s.QueueDir, status, fmt.Sprintf("%s.yaml", taskID))
		if _, err := os.Stat(filePath); err == nil {
			// Delete the file
			if err := os.Remove(filePath); err != nil {
				return status, fmt.Errorf("failed to delete task: %v", err)
			}
			
			log.Printf("Deleted task %s from %s", taskID, status)
			return status, nil
		}
	}
	
	return "", fmt.Errorf("task not found: %s", taskID)
}

// Utility functions
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func ContainsTask(slice []TaskItem, item TaskItem) bool {
	for _, t := range slice {
		if t.ID == item.ID {
			return true
		}
	}
	return false
}