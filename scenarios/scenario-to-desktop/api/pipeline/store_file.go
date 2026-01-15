package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// FileStore implements Store with file-backed persistence.
// It uses an in-memory store as a hot cache and persists changes to disk.
type FileStore struct {
	mu      sync.RWMutex
	mem     *InMemoryStore
	dataDir string
	logger  Logger
}

// FileStoreOption configures a FileStore.
type FileStoreOption func(*FileStore)

// WithFileStoreLogger sets the logger for the file store.
func WithFileStoreLogger(l Logger) FileStoreOption {
	return func(s *FileStore) {
		s.logger = l
	}
}

// NewFileStore creates a new file-backed pipeline store.
// It creates the data directory if it doesn't exist and loads existing pipelines from disk.
func NewFileStore(dataDir string, opts ...FileStoreOption) (*FileStore, error) {
	s := &FileStore{
		mem:     NewInMemoryStore(),
		dataDir: dataDir,
	}

	for _, opt := range opts {
		opt(s)
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create pipeline data directory: %w", err)
	}

	// Load existing pipelines from disk
	if err := s.loadFromDisk(); err != nil {
		return nil, fmt.Errorf("failed to load pipelines from disk: %w", err)
	}

	return s, nil
}

// Save creates or updates a pipeline status.
func (s *FileStore) Save(status *Status) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Save to memory
	s.mem.Save(status)

	// Persist to disk
	if err := s.writeToDisk(status); err != nil {
		s.logError("failed to persist pipeline to disk", "pipeline_id", status.PipelineID, "error", err)
	}
}

// Get retrieves a pipeline status by ID.
func (s *FileStore) Get(pipelineID string) (*Status, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mem.Get(pipelineID)
}

// Update updates a pipeline status using a modifier function.
func (s *FileStore) Update(pipelineID string, fn func(status *Status)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update in memory
	if !s.mem.Update(pipelineID, fn) {
		return false
	}

	// Get updated status and persist
	status, ok := s.mem.Get(pipelineID)
	if ok {
		if err := s.writeToDisk(status); err != nil {
			s.logError("failed to persist pipeline update to disk", "pipeline_id", pipelineID, "error", err)
		}
	}

	return true
}

// UpdateStage updates a specific stage's result within a pipeline.
func (s *FileStore) UpdateStage(pipelineID, stageName string, result *StageResult) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update in memory
	if !s.mem.UpdateStage(pipelineID, stageName, result) {
		return false
	}

	// Get updated status and persist
	status, ok := s.mem.Get(pipelineID)
	if ok {
		if err := s.writeToDisk(status); err != nil {
			s.logError("failed to persist stage update to disk", "pipeline_id", pipelineID, "stage", stageName, "error", err)
		}
	}

	return true
}

// Delete removes a pipeline status.
func (s *FileStore) Delete(pipelineID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Delete from memory
	if !s.mem.Delete(pipelineID) {
		return false
	}

	// Delete from disk
	filePath := s.filePath(pipelineID)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		s.logError("failed to delete pipeline file from disk", "pipeline_id", pipelineID, "error", err)
	}

	return true
}

// List returns all pipeline statuses.
func (s *FileStore) List() []*Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mem.List()
}

// Cleanup removes completed pipelines older than the given duration.
func (s *FileStore) Cleanup(olderThanUnix int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get list of pipelines to clean up before calling cleanup
	toDelete := make([]string, 0)
	for _, status := range s.mem.List() {
		if status.IsComplete() && status.CompletedAt > 0 && status.CompletedAt < olderThanUnix {
			toDelete = append(toDelete, status.PipelineID)
		}
	}

	// Cleanup in memory
	s.mem.Cleanup(olderThanUnix)

	// Delete files from disk
	for _, pipelineID := range toDelete {
		filePath := s.filePath(pipelineID)
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			s.logError("failed to delete pipeline file during cleanup", "pipeline_id", pipelineID, "error", err)
		}
	}
}

// loadFromDisk loads all pipeline statuses from disk into memory.
func (s *FileStore) loadFromDisk() error {
	entries, err := os.ReadDir(s.dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No data directory yet, nothing to load
		}
		return fmt.Errorf("failed to read pipeline data directory: %w", err)
	}

	loadedCount := 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(s.dataDir, entry.Name())
		status, err := s.readFromFile(filePath)
		if err != nil {
			s.logError("failed to load pipeline from disk, skipping", "file", entry.Name(), "error", err)
			continue
		}

		s.mem.Save(status)
		loadedCount++
	}

	if loadedCount > 0 {
		s.logInfo("loaded pipelines from disk", "count", loadedCount)
	}

	return nil
}

// writeToDisk persists a pipeline status to disk.
func (s *FileStore) writeToDisk(status *Status) error {
	filePath := s.filePath(status.PipelineID)

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal pipeline status: %w", err)
	}

	// Write atomically using temp file + rename
	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath) // Clean up temp file on rename failure
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// readFromFile reads a pipeline status from a file.
func (s *FileStore) readFromFile(filePath string) (*Status, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var status Status
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pipeline status: %w", err)
	}

	return &status, nil
}

// filePath returns the file path for a pipeline status.
func (s *FileStore) filePath(pipelineID string) string {
	return filepath.Join(s.dataDir, pipelineID+".json")
}

// logInfo logs an info message if logger is configured.
func (s *FileStore) logInfo(msg string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Info(msg, args...)
	}
}

// logError logs an error message if logger is configured.
func (s *FileStore) logError(msg string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Error(msg, args...)
	}
}
