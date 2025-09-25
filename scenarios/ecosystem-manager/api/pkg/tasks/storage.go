package tasks

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ecosystem-manager/api/pkg/systemlog"
	"gopkg.in/yaml.v3"
)

var timestampSuffixPattern = regexp.MustCompile(`-(\d{6}|\d{8}|\d{4}-\d{2}-\d{2})$`)

var activeTaskStatuses = []string{"pending", "in-progress", "review"}

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

func hasLegacyTaskFields(raw map[string]interface{}) bool {
	legacyKeys := []string{
		"impact_score",
		"requirements",
		"assigned_resources",
		"progress_percentage",
		"estimated_completion",
	}
	for _, key := range legacyKeys {
		if _, exists := raw[key]; exists {
			return true
		}
	}
	return false
}

func (s *Storage) normalizeTaskItem(item *TaskItem, status string, raw map[string]interface{}) bool {
	changed := false

	normalizedTargets, canonicalTarget := NormalizeTargets(item.Target, item.Targets)
	if !EqualStringSlices(item.Targets, normalizedTargets) {
		item.Targets = normalizedTargets
		changed = true
	}

	if item.Target != canonicalTarget {
		item.Target = canonicalTarget
		changed = true
	}

	if item.Status == "" && status != "" {
		item.Status = status
		changed = true
	}

	if item.Status == "completed" {
		if item.CompletionCount < 1 {
			item.CompletionCount = 1
			changed = true
		}
		if item.LastCompletedAt == "" && item.CompletedAt != "" {
			item.LastCompletedAt = item.CompletedAt
			changed = true
		}
	} else {
		if item.CompletionCount < 0 {
			item.CompletionCount = 0
			changed = true
		}
	}

	if raw != nil && hasLegacyTaskFields(raw) {
		changed = true
	}

	// Ensure streak counters are non-negative
	if item.ConsecutiveCompletionClaims < 0 {
		item.ConsecutiveCompletionClaims = 0
		changed = true
	}
	if item.ConsecutiveFailures < 0 {
		item.ConsecutiveFailures = 0
		changed = true
	}

	// Default processor auto requeue to true if not explicitly set
	if raw != nil {
		if _, exists := raw["processor_auto_requeue"]; !exists {
			if !item.ProcessorAutoRequeue {
				item.ProcessorAutoRequeue = true
				changed = true
			}
		}
	} else if !item.ProcessorAutoRequeue {
		// If raw metadata missing (e.g., crafted programmatically) default to true
		item.ProcessorAutoRequeue = true
		changed = true
	}

	return changed
}

// FindActiveTargetTask returns an active task (pending/in-progress/review) for the same type/operation/target if it exists.
func (s *Storage) FindActiveTargetTask(taskType, operation, target string) (*TaskItem, string, error) {
	normalizedTarget := strings.ToLower(strings.TrimSpace(target))
	if normalizedTarget == "" {
		return nil, "", nil
	}

	for _, status := range activeTaskStatuses {
		items, err := s.GetQueueItems(status)
		if err != nil {
			return nil, "", err
		}

		for i := range items {
			item := items[i]
			if item.Type != taskType || item.Operation != operation {
				continue
			}

			normalizedTargets, _ := NormalizeTargets(item.Target, item.Targets)
			for _, candidate := range normalizedTargets {
				if strings.ToLower(candidate) == normalizedTarget {
					copy := item
					return &copy, status, nil
				}
			}
		}
	}

	return nil, "", nil
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

		var raw map[string]interface{}
		if err := yaml.Unmarshal(data, &raw); err != nil {
			log.Printf("Warning: unable to parse raw YAML for %s: %v", file, err)
		}

		var item TaskItem
		if err := yaml.Unmarshal(data, &item); err != nil {
			log.Printf("Error unmarshaling YAML from %s: %v", file, err)
			continue
		}

		// Convert interface{} maps to string maps for JSON compatibility
		if item.Results != nil {
			converted := convertToStringMap(item.Results)
			if m, ok := converted.(map[string]interface{}); ok {
				item.Results = m
			}
		}

		if s.normalizeTaskItem(&item, status, raw) {
			if err := s.SaveQueueItem(item, status); err != nil {
				log.Printf("Warning: failed to rewrite sanitized task %s: %v", item.ID, err)
			} else {
				log.Printf("Sanitized legacy task metadata for %s", item.ID)
			}
		}

		log.Printf("Successfully loaded task: %s", item.ID)
		items = append(items, item)
	}

	log.Printf("Returning %d items for status %s", len(items), status)
	return items, nil
}

// SaveQueueItem saves a task to the specified status directory using atomic write
func (s *Storage) SaveQueueItem(item TaskItem, status string) error {
	queuePath := filepath.Join(s.QueueDir, status)
	filename := fmt.Sprintf("%s.yaml", item.ID)
	filePath := filepath.Join(queuePath, filename)

	data, err := yaml.Marshal(item)
	if err != nil {
		return err
	}

	// Use atomic write to prevent partial writes or corruption
	return s.atomicWriteFile(filePath, data, 0644)
}

// findTaskFile returns the path and contents for a task file within a specific status directory.
func (s *Storage) findTaskFile(status, taskID string) (string, []byte, error) {
	queuePath := filepath.Join(s.QueueDir, status)

	candidate := filepath.Join(queuePath, fmt.Sprintf("%s.yaml", taskID))
	if data, err := os.ReadFile(candidate); err == nil {
		return candidate, data, nil
	}

	// Handle cases where filename omits timestamp suffixes
	taskIDPrefix := taskID
	if matches := timestampSuffixPattern.FindStringSubmatch(taskID); len(matches) > 0 {
		taskIDPrefix = taskID[:len(taskID)-len(matches[0])]
	}

	entries, err := os.ReadDir(queuePath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read %s queue: %w", status, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") {
			continue
		}
		nameWithoutExt := strings.TrimSuffix(name, ".yaml")
		if nameWithoutExt != taskID && nameWithoutExt != taskIDPrefix {
			continue
		}

		path := filepath.Join(queuePath, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return "", nil, fmt.Errorf("failed to read task file %s: %w", path, err)
		}

		var task TaskItem
		if err := yaml.Unmarshal(data, &task); err != nil {
			continue
		}
		if task.ID == taskID {
			return path, data, nil
		}
	}

	return "", nil, fmt.Errorf("failed to find task file for ID %s in status %s", taskID, status)
}

// CurrentStatus returns the queue directory that currently holds the task, if any.
func (s *Storage) CurrentStatus(taskID string) (string, error) {
	_, status, err := s.GetTaskByID(taskID)
	if err != nil {
		return "", err
	}
	return status, nil
}

// atomicWriteFile writes data to a file atomically by writing to a temp file first
func (s *Storage) atomicWriteFile(filePath string, data []byte, perm os.FileMode) error {
	// Create temp file in same directory to ensure same filesystem
	dir := filepath.Dir(filePath)
	tempFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	tempPath := tempFile.Name()

	// Ensure cleanup on any error
	defer func() {
		// Remove temp file if it still exists (in case of error)
		os.Remove(tempPath)
	}()

	// Write data to temp file
	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to write to temp file: %v", err)
	}

	// Close temp file before rename
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %v", err)
	}

	// Set proper permissions
	if err := os.Chmod(tempPath, perm); err != nil {
		return fmt.Errorf("failed to set permissions: %v", err)
	}

	// Atomic rename - this is atomic on same filesystem
	if err := os.Rename(tempPath, filePath); err != nil {
		return fmt.Errorf("failed to rename temp file to target: %v", err)
	}

	return nil
}

// MoveQueueItem moves a task from one status to another (atomic rename)
func (s *Storage) MoveQueueItem(itemID, fromStatus, toStatus string) error {
	fromPath := filepath.Join(s.QueueDir, fromStatus, fmt.Sprintf("%s.yaml", itemID))
	toPath := filepath.Join(s.QueueDir, toStatus, fmt.Sprintf("%s.yaml", itemID))

	return os.Rename(fromPath, toPath)
}

// MoveTaskTo relocates a task file to the provided status directory.
// It always inspects the filesystem to discover the current location first.
func (s *Storage) MoveTaskTo(taskID, toStatus string) (*TaskItem, string, error) {
	if toStatus == "" {
		return nil, "", fmt.Errorf("destination status required")
	}

	task, currentStatus, err := s.GetTaskByID(taskID)
	if err != nil {
		return nil, "", err
	}

	if currentStatus == toStatus {
		systemlog.Debugf("MoveTaskTo noop: task=%s already in %s", taskID, toStatus)
		return task, currentStatus, nil
	}

	fromPath := filepath.Join(s.QueueDir, currentStatus, fmt.Sprintf("%s.yaml", taskID))
	toPath := filepath.Join(s.QueueDir, toStatus, fmt.Sprintf("%s.yaml", taskID))

	systemlog.Debugf("MoveTaskTo start: task=%s from=%s to=%s", taskID, currentStatus, toStatus)

	if err := os.MkdirAll(filepath.Dir(toPath), 0755); err != nil {
		return nil, "", fmt.Errorf("failed to ensure destination directory: %w", err)
	}

	if err := os.Rename(fromPath, toPath); err != nil {
		systemlog.Errorf("MoveTaskTo rename failed: task=%s from=%s to=%s err=%v", taskID, currentStatus, toStatus, err)
		return nil, "", fmt.Errorf("failed to move task %s from %s to %s: %w", taskID, currentStatus, toStatus, err)
	}

	log.Printf("Successfully moved task %s from %s to %s", taskID, currentStatus, toStatus)
	systemlog.Debugf("MoveTaskTo completed: task=%s from=%s to=%s", taskID, currentStatus, toStatus)

	return task, currentStatus, nil
}

// MoveTask moves a task between queue states and updates timestamps using filesystem discovery.
func (s *Storage) MoveTask(taskID, fromStatus, toStatus string) error {
	_, actualFrom, err := s.MoveTaskTo(taskID, toStatus)
	if err != nil {
		return err
	}

	if fromStatus != "" && fromStatus != actualFrom {
		systemlog.Warnf("MoveTask requested from %s but task %s located in %s", fromStatus, taskID, actualFrom)
	}

	return nil
}

// GetTaskByID finds a task by ID across all queue statuses
func (s *Storage) GetTaskByID(taskID string) (*TaskItem, string, error) {
	statuses := []string{"pending", "in-progress", "review", "completed", "failed", "completed-finalized", "failed-blocked"}

	// Strategy 1: Try exact filename match
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

	// Strategy 2: Try to find a file where the filename (without .yaml) starts with the taskID
	// This handles cases where the ID has extra suffix but filename doesn't
	// Use regex to detect timestamp patterns more robustly
	taskIDPrefix := taskID

	// Pattern matches common timestamp formats at the end:
	// - 6 digits (HHMMSS format like 010900)
	// - 8 digits (YYYYMMDD format like 20250110)
	// - timestamp with dashes like 2025-01-10
	if matches := timestampSuffixPattern.FindStringSubmatch(taskID); len(matches) > 0 {
		// Remove the timestamp suffix to get the base ID
		taskIDPrefix = taskID[:len(taskID)-len(matches[0])]
	}

	// Try to find file with the prefix
	for _, status := range statuses {
		dirPath := filepath.Join(s.QueueDir, status)
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			filename := entry.Name()
			if !strings.HasSuffix(filename, ".yaml") {
				continue
			}

			// Remove .yaml extension
			nameWithoutExt := strings.TrimSuffix(filename, ".yaml")

			// Check if this file matches our taskID or taskIDPrefix
			if nameWithoutExt == taskIDPrefix || nameWithoutExt == taskID {
				filePath := filepath.Join(dirPath, filename)
				data, err := os.ReadFile(filePath)
				if err != nil {
					continue
				}

				var task TaskItem
				if err := yaml.Unmarshal(data, &task); err != nil {
					continue
				}

				// Verify the ID matches what we're looking for
				if task.ID == taskID {
					return &task, status, nil
				}
			}
		}
	}

	// Strategy 3: As a last resort, scan all files and match by internal ID
	for _, status := range statuses {
		dirPath := filepath.Join(s.QueueDir, status)
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
				continue
			}

			filePath := filepath.Join(dirPath, entry.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}

			var task TaskItem
			if err := yaml.Unmarshal(data, &task); err != nil {
				continue
			}

			if task.ID == taskID {
				return &task, status, nil
			}
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
