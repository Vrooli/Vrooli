package tasks

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ecosystem-manager/api/pkg/internal/slices"
	"github.com/ecosystem-manager/api/pkg/systemlog"
	"gopkg.in/yaml.v3"
)

var activeTaskStatuses = []string{StatusPending, StatusInProgress}
var queueStatuses = QueueStatuses

// IsValidStatus checks if a status string is a valid queue status
func IsValidStatus(status string) bool {
	for _, valid := range QueueStatuses {
		if valid == status {
			return true
		}
	}
	return false
}

// GetValidStatuses returns a copy of all valid queue statuses
func GetValidStatuses() []string {
	result := make([]string, len(QueueStatuses))
	copy(result, QueueStatuses)
	return result
}

var statusOrder = func() map[string]int {
	order := make(map[string]int, len(QueueStatuses))
	for idx, status := range QueueStatuses {
		order[status] = idx
	}
	return order
}()

// Storage handles file-based task persistence
// DESIGN DECISION: File-based task storage is intentional and provides several benefits:
// 1. Manual task editing - developers can directly edit .yaml files for debugging/testing
// 2. Git version control - task history and changes are tracked in source control
// 3. Multi-scenario collaboration - other scenarios can create/modify tasks via file system
// 4. Transparency - task state is always visible and inspectable without special tools
// 5. Atomic operations - file renames provide atomic status transitions
type Storage struct {
	QueueDir string

	taskLocks sync.Map // taskID -> *sync.Mutex
}

// NewStorage creates a new storage instance
func NewStorage(queueDir string) *Storage {
	return &Storage{
		QueueDir: queueDir,
	}
}

// convertToStringMap converts map[any]any to map[string]any
// This is needed because YAML unmarshals to any keys but JSON needs string keys
func convertToStringMap(m any) any {
	switch v := m.(type) {
	case map[any]any:
		result := make(map[string]any)
		for key, value := range v {
			keyStr, ok := key.(string)
			if !ok {
				keyStr = fmt.Sprintf("%v", key)
			}
			result[keyStr] = convertToStringMap(value)
		}
		return result
	case []any:
		result := make([]any, len(v))
		for i, value := range v {
			result[i] = convertToStringMap(value)
		}
		return result
	default:
		return v
	}
}

func hasLegacyTaskFields(raw map[string]any) bool {
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

func (s *Storage) normalizeTaskItem(item *TaskItem, status string, raw map[string]any) bool {
	changed := false

	// Always align status with directory as the source of truth
	if status != "" && item.Status != status {
		item.Status = status
		changed = true
	}

	normalizedTargets := CollectTargets(item)
	canonicalTarget := ""
	if len(normalizedTargets) > 0 {
		canonicalTarget = normalizedTargets[0]
	}

	if !slices.EqualStringSlices(item.Targets, normalizedTargets) {
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

	return changed
}

// FindActiveTargetTask returns an active task (pending/in-progress) for the same type/operation/target if it exists.
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

			normalizedTargets := CollectTargets(&item)
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

	files, err := filepath.Glob(filepath.Join(queuePath, "*.yaml"))
	if err != nil {
		log.Printf("Error globbing files in %s: %v", status, err)
		return nil, err
	}

	var items []TaskItem
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			log.Printf("Error reading file %s: %v", file, err)
			continue
		}

		var raw map[string]any
		if err := yaml.Unmarshal(data, &raw); err != nil {
			// Silent - not critical for operation
		}

		var item TaskItem
		if err := yaml.Unmarshal(data, &item); err != nil {
			log.Printf("Error unmarshaling YAML from %s: %v", file, err)
			continue
		}

		// Convert any maps to string maps for JSON compatibility
		if item.Results != nil {
			converted := convertToStringMap(item.Results)
			if m, ok := converted.(map[string]any); ok {
				item.Results = m
			}
		}

		// PERFORMANCE OPTIMIZATION: Only rewrite task files if normalization detected changes
		// This avoids unnecessary disk I/O when reading tasks that are already properly formatted
		// Use SkipCleanup since we're just reading, not moving - cleanup only needed during moves
		if s.normalizeTaskItem(&item, status, raw) {
			systemlog.Debugf("Task %s required normalization, rewriting to disk", item.ID)
			if err := s.SaveQueueItemSkipCleanup(item, status); err != nil {
				log.Printf("Warning: failed to rewrite sanitized task %s: %v", item.ID, err)
			}
		}

		items = append(items, item)
	}

	return items, nil
}

// SaveQueueItem saves a task to the specified status directory using atomic write
func (s *Storage) SaveQueueItem(item TaskItem, status string) error {
	return s.saveQueueItemWithOptions(item, status, true)
}

// SaveQueueItemSkipCleanup saves a task without duplicate cleanup (optimization for post-move saves)
func (s *Storage) SaveQueueItemSkipCleanup(item TaskItem, status string) error {
	return s.saveQueueItemWithOptions(item, status, false)
}

// saveQueueItemWithOptions is the internal implementation with cleanup control
func (s *Storage) saveQueueItemWithOptions(item TaskItem, status string, cleanupDuplicates bool) error {
	unlock := s.lockTask(item.ID)
	defer unlock()

	// Always enforce status to match the destination directory
	if status != "" && item.Status != status {
		item.Status = status
	}

	queuePath := filepath.Join(s.QueueDir, status)
	filename := fmt.Sprintf("%s.yaml", item.ID)
	filePath := filepath.Join(queuePath, filename)

	data, err := yaml.Marshal(item)
	if err != nil {
		return err
	}

	// Use atomic write to prevent partial writes or corruption
	if err := s.atomicWriteFile(filePath, data, 0644); err != nil {
		return err
	}

	// PERFORMANCE OPTIMIZATION: Skip duplicate cleanup if caller knows it's already done
	// This avoids redundant filesystem scans after MoveTaskTo() which already cleaned up
	if cleanupDuplicates {
		if err := s.cleanupDuplicateTaskFiles(item.ID, status); err != nil {
			return err
		}
	}

	return nil
}

func (s *Storage) cleanupDuplicateTaskFiles(taskID, keepStatus string) error {
	var errs []error
	duplicatesFound := false

	for _, status := range queueStatuses {
		if status == keepStatus {
			continue
		}

		path := filepath.Join(s.QueueDir, status, fmt.Sprintf("%s.yaml", taskID))
		// Fast path: check if file exists before attempting removal
		// This avoids unnecessary error handling for the common case of no duplicates
		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			errs = append(errs, fmt.Errorf("stat duplicate task %s in %s: %w", taskID, status, err))
			continue
		}

		// File exists - we have a duplicate
		duplicatesFound = true
		if err := os.Remove(path); err != nil {
			errs = append(errs, fmt.Errorf("remove duplicate task %s from %s: %w", taskID, status, err))
			continue
		}

		log.Printf("Removed duplicate task %s from %s queue", taskID, status)
		systemlog.Warnf("Removed duplicate task %s from %s queue", taskID, status)
	}

	// Only log if we actually found and cleaned up duplicates
	if duplicatesFound && len(errs) == 0 {
		systemlog.Debugf("Successfully cleaned up duplicate task files for %s", taskID)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// CleanupDuplicates scans all queue directories and removes stale copies of tasks,
// keeping the instance with the most recent modification time (breaking ties by status order).
func (s *Storage) CleanupDuplicates() error {
	type fileEntry struct {
		status  string
		path    string
		modTime time.Time
	}

	entriesByID := make(map[string][]fileEntry)
	var errs []error

	for _, status := range queueStatuses {
		dirPath := filepath.Join(s.QueueDir, status)
		files, err := os.ReadDir(dirPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("read %s queue: %w", status, err)
		}

		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".yaml") {
				continue
			}

			id := strings.TrimSuffix(file.Name(), ".yaml")
			info, err := file.Info()
			if err != nil {
				errs = append(errs, fmt.Errorf("stat task %s in %s: %w", id, status, err))
				continue
			}

			entriesByID[id] = append(entriesByID[id], fileEntry{
				status:  status,
				path:    filepath.Join(s.QueueDir, status, file.Name()),
				modTime: info.ModTime(),
			})
		}
	}

	for id, entries := range entriesByID {
		if len(entries) <= 1 {
			continue
		}

		keep := entries[0]
		for _, entry := range entries[1:] {
			if entry.modTime.After(keep.modTime) {
				keep = entry
				continue
			}
			if entry.modTime.Equal(keep.modTime) && statusOrder[entry.status] < statusOrder[keep.status] {
				keep = entry
			}
		}

		for _, entry := range entries {
			if entry.path == keep.path {
				continue
			}

			if err := os.Remove(entry.path); err != nil {
				errs = append(errs, fmt.Errorf("remove duplicate task %s from %s: %w", id, entry.status, err))
				continue
			}

			log.Printf("Startup cleanup removed duplicate task %s from %s queue (keeping %s)", id, entry.status, keep.status)
			systemlog.Warnf("Startup cleanup removed duplicate task %s from %s queue (keeping %s)", id, entry.status, keep.status)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// ResyncStatuses enforces that each task file's status matches its directory.
// This prevents stale in-file statuses from drifting after moves or manual edits.
func (s *Storage) ResyncStatuses() error {
	var errs []error
	rewritten := 0

	for _, status := range queueStatuses {
		dirPath := filepath.Join(s.QueueDir, status)
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			errs = append(errs, fmt.Errorf("read %s queue: %w", status, err))
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
				continue
			}

			filePath := filepath.Join(dirPath, entry.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				errs = append(errs, fmt.Errorf("read %s: %w", filePath, err))
				continue
			}

			var raw map[string]any
			if err := yaml.Unmarshal(data, &raw); err != nil {
				// Not fatal; continue with best-effort
			}

			var task TaskItem
			if err := yaml.Unmarshal(data, &task); err != nil {
				errs = append(errs, fmt.Errorf("unmarshal %s: %w", filePath, err))
				continue
			}

			if s.normalizeTaskItem(&task, status, raw) {
				if err := s.SaveQueueItemSkipCleanup(task, status); err != nil {
					errs = append(errs, fmt.Errorf("rewrite %s: %w", filePath, err))
					continue
				}
				rewritten++
			}
		}
	}

	if rewritten > 0 {
		systemlog.Infof("Resynced %d task file status fields to match directory", rewritten)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
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
		return fmt.Errorf("failed to create temp file: %w", err)
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
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	// Close temp file before rename
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Set proper permissions
	if err := os.Chmod(tempPath, perm); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	// Atomic rename - this is atomic on same filesystem
	if err := os.Rename(tempPath, filePath); err != nil {
		return fmt.Errorf("failed to rename temp file to target: %w", err)
	}

	return nil
}

// MoveQueueItem moves a task from one status to another (atomic rename)
func (s *Storage) MoveQueueItem(itemID, fromStatus, toStatus string) error {
	unlock := s.lockTask(itemID)
	defer unlock()

	fromPath := filepath.Join(s.QueueDir, fromStatus, fmt.Sprintf("%s.yaml", itemID))
	toPath := filepath.Join(s.QueueDir, toStatus, fmt.Sprintf("%s.yaml", itemID))

	return os.Rename(fromPath, toPath)
}

// MoveTaskTo relocates a task file to the provided status directory.
// It always inspects the filesystem to discover the current location first.
func (s *Storage) MoveTaskTo(taskID, toStatus string) (*TaskItem, string, error) {
	unlock := s.lockTask(taskID)
	defer unlock()

	if toStatus == "" {
		return nil, "", fmt.Errorf("move task %s: destination status required", taskID)
	}

	task, currentStatus, err := s.GetTaskByID(taskID)
	if err != nil {
		return nil, "", fmt.Errorf("move task %s to %s: %w", taskID, toStatus, err)
	}

	if currentStatus == toStatus {
		systemlog.Debugf("MoveTaskTo noop: task=%s already in %s", taskID, toStatus)
		return task, currentStatus, nil
	}

	fromPath := filepath.Join(s.QueueDir, currentStatus, fmt.Sprintf("%s.yaml", taskID))
	toPath := filepath.Join(s.QueueDir, toStatus, fmt.Sprintf("%s.yaml", taskID))

	systemlog.Debugf("MoveTaskTo start: task=%s from=%s to=%s", taskID, currentStatus, toStatus)

	if err := os.MkdirAll(filepath.Dir(toPath), 0755); err != nil {
		return nil, "", fmt.Errorf("move task %s: ensure destination dir %s (path=%s): %w", taskID, toStatus, filepath.Dir(toPath), err)
	}

	if err := os.Rename(fromPath, toPath); err != nil {
		wrappedErr := fmt.Errorf("move task %s: rename %s -> %s failed (from_path=%s, to_path=%s): %w", taskID, currentStatus, toStatus, fromPath, toPath, err)
		systemlog.Errorf("MoveTaskTo rename failed: task=%s from=%s to=%s from_path=%s to_path=%s err=%v", taskID, currentStatus, toStatus, fromPath, toPath, wrappedErr)
		return nil, "", wrappedErr
	}

	log.Printf("Successfully moved task %s from %s to %s", taskID, currentStatus, toStatus)
	systemlog.Debugf("MoveTaskTo completed: task=%s from=%s to=%s", taskID, currentStatus, toStatus)

	if err := s.cleanupDuplicateTaskFiles(taskID, toStatus); err != nil {
		return task, currentStatus, fmt.Errorf("move task %s: cleanup duplicates after move to %s: %w", taskID, toStatus, err)
	}

	// Keep returned task aligned with destination status
	if task != nil {
		task.Status = toStatus
	}

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

// readAndValidateTaskFile reads a task file and validates its ID matches the expected taskID
func (s *Storage) readAndValidateTaskFile(filePath, taskID string) (*TaskItem, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var task TaskItem
	if err := yaml.Unmarshal(data, &task); err != nil {
		return nil, err
	}

	// Verify the internal ID matches what we're looking for
	if task.ID != taskID {
		return nil, fmt.Errorf("task ID mismatch: expected %s, got %s", taskID, task.ID)
	}

	return &task, nil
}

// GetTaskByID finds a task by ID across all queue statuses using a progressive search strategy
func (s *Storage) GetTaskByID(taskID string) (*TaskItem, string, error) {
	// Strategy 1: Try exact filename match (fastest path)
	for _, status := range queueStatuses {
		filePath := filepath.Join(s.QueueDir, status, fmt.Sprintf("%s.yaml", taskID))
		if task, err := s.readAndValidateTaskFile(filePath, taskID); err == nil {
			// PERFORMANCE: Don't cleanup on read - cleanup is handled during saves/moves
			// This avoids expensive filesystem scans on every task lookup
			return task, status, nil
		}
	}

	// Strategy 2 & 3: Scan directories for filename or internal ID match
	// Optimized to read each file only once by caching the first read attempt
	for _, status := range queueStatuses {
		dirPath := filepath.Join(s.QueueDir, status)
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
				continue
			}

			filename := entry.Name()
			filePath := filepath.Join(dirPath, filename)

			// Read the file once and check both filename and internal ID
			// This avoids double-read when filename matches but ID validation fails
			data, readErr := os.ReadFile(filePath)
			if readErr != nil {
				continue
			}

			var task TaskItem
			if err := yaml.Unmarshal(data, &task); err != nil {
				continue
			}

			// Check if internal ID matches (works for both strategies)
			if task.ID == taskID {
				// PERFORMANCE: Don't cleanup on read - cleanup is handled during saves/moves
				return &task, status, nil
			}
		}
	}

	return nil, "", fmt.Errorf("task not found: %s", taskID)
}

// DeleteTask removes a task file from the appropriate status directory
func (s *Storage) DeleteTask(taskID string) (string, error) {
	unlock := s.lockTask(taskID)
	defer unlock()

	for _, status := range queueStatuses {
		filePath := filepath.Join(s.QueueDir, status, fmt.Sprintf("%s.yaml", taskID))
		if _, err := os.Stat(filePath); err == nil {
			// Delete the file
			if err := os.Remove(filePath); err != nil {
				return status, fmt.Errorf("failed to delete task: %w", err)
			}

			log.Printf("Deleted task %s from %s", taskID, status)
			return status, nil
		}
	}

	return "", fmt.Errorf("task not found: %s", taskID)
}

// lockTask provides a per-task critical section to prevent concurrent moves/writes that can create duplicates.
func (s *Storage) lockTask(taskID string) func() {
	val, _ := s.taskLocks.LoadOrStore(taskID, &sync.Mutex{})
	mtx := val.(*sync.Mutex)
	mtx.Lock()
	return func() { mtx.Unlock() }
}
