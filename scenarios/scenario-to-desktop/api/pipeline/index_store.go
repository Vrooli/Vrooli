package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ScenarioIndexStore manages scenario-to-pipeline mappings with file persistence.
// It provides thread-safe access to scenario indexes with memory caching and
// automatic file persistence.
type ScenarioIndexStore struct {
	mu       sync.RWMutex
	indexes  map[string]*ScenarioIndex
	dataDir  string
	logger   Logger
	timeFunc func() int64
}

// ScenarioIndexStoreOption configures a ScenarioIndexStore.
type ScenarioIndexStoreOption func(*ScenarioIndexStore)

// WithIndexStoreLogger sets the logger for the index store.
func WithIndexStoreLogger(l Logger) ScenarioIndexStoreOption {
	return func(s *ScenarioIndexStore) {
		s.logger = l
	}
}

// WithIndexStoreTimeFunc sets a custom time function for testing.
func WithIndexStoreTimeFunc(fn func() int64) ScenarioIndexStoreOption {
	return func(s *ScenarioIndexStore) {
		s.timeFunc = fn
	}
}

// NewScenarioIndexStore creates a new scenario index store.
// It loads existing indexes from disk on creation.
func NewScenarioIndexStore(dataDir string, opts ...ScenarioIndexStoreOption) (*ScenarioIndexStore, error) {
	s := &ScenarioIndexStore{
		indexes: make(map[string]*ScenarioIndex),
		dataDir: dataDir,
		timeFunc: func() int64 {
			return time.Now().Unix()
		},
	}

	for _, opt := range opts {
		opt(s)
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create index data directory: %w", err)
	}

	// Load existing indexes from disk
	if err := s.loadFromDisk(); err != nil {
		return nil, fmt.Errorf("failed to load indexes from disk: %w", err)
	}

	return s, nil
}

// Get retrieves the index for a scenario. Returns nil if no index exists.
func (s *ScenarioIndexStore) Get(scenarioName string) *ScenarioIndex {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.indexes[scenarioName]
}

// GetOrCreate retrieves or creates an index for a scenario.
func (s *ScenarioIndexStore) GetOrCreate(scenarioName string) *ScenarioIndex {
	s.mu.Lock()
	defer s.mu.Unlock()

	if idx, ok := s.indexes[scenarioName]; ok {
		return idx
	}

	idx := &ScenarioIndex{
		ScenarioName:   scenarioName,
		History:        make([]string, 0),
		MaxHistorySize: DefaultMaxHistorySize,
		UpdatedAt:      s.timeFunc(),
	}
	s.indexes[scenarioName] = idx
	_ = s.persistUnlocked(scenarioName)
	return idx
}

// SetActivePipeline sets the active pipeline for a scenario.
// If there was a previous active pipeline, it is NOT automatically archived.
// Use ArchiveActive first if you want to preserve the current active pipeline.
func (s *ScenarioIndexStore) SetActivePipeline(scenarioName, pipelineID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.getOrCreateUnlocked(scenarioName)
	idx.ActivePipelineID = pipelineID
	idx.UpdatedAt = s.timeFunc()

	return s.persistUnlocked(scenarioName)
}

// ArchiveActive moves the current active pipeline to history and clears the active slot.
// Returns the archived pipeline ID, or empty string if no active pipeline existed.
func (s *ScenarioIndexStore) ArchiveActive(scenarioName string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indexes[scenarioName]
	if idx == nil || idx.ActivePipelineID == "" {
		return "", nil
	}

	archivedID := idx.ActivePipelineID
	idx.AddToHistory(archivedID)
	idx.ActivePipelineID = ""
	idx.UpdatedAt = s.timeFunc()

	if err := s.persistUnlocked(scenarioName); err != nil {
		return "", err
	}

	return archivedID, nil
}

// ClearActive clears the active pipeline without archiving it.
// Use this when the pipeline should be discarded (e.g., cancelled or failed).
func (s *ScenarioIndexStore) ClearActive(scenarioName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.indexes[scenarioName]
	if idx == nil {
		return nil
	}

	idx.ActivePipelineID = ""
	idx.UpdatedAt = s.timeFunc()

	return s.persistUnlocked(scenarioName)
}

// GetHistory returns the pipeline history for a scenario.
// Returns an empty slice if no history exists.
func (s *ScenarioIndexStore) GetHistory(scenarioName string, limit int) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	idx := s.indexes[scenarioName]
	if idx == nil || len(idx.History) == 0 {
		return []string{}
	}

	if limit <= 0 || limit > len(idx.History) {
		limit = len(idx.History)
	}

	result := make([]string, limit)
	copy(result, idx.History[:limit])
	return result
}

// List returns all scenario names with indexes.
func (s *ScenarioIndexStore) List() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]string, 0, len(s.indexes))
	for name := range s.indexes {
		result = append(result, name)
	}
	return result
}

// getOrCreateUnlocked gets or creates an index without locking.
// Caller must hold the lock.
func (s *ScenarioIndexStore) getOrCreateUnlocked(scenarioName string) *ScenarioIndex {
	if idx, ok := s.indexes[scenarioName]; ok {
		return idx
	}

	idx := &ScenarioIndex{
		ScenarioName:   scenarioName,
		History:        make([]string, 0),
		MaxHistorySize: DefaultMaxHistorySize,
		UpdatedAt:      s.timeFunc(),
	}
	s.indexes[scenarioName] = idx
	return idx
}

// persistUnlocked writes an index to disk. Caller must hold the lock.
func (s *ScenarioIndexStore) persistUnlocked(scenarioName string) error {
	idx := s.indexes[scenarioName]
	if idx == nil {
		return nil
	}

	filePath := s.filePath(scenarioName)

	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	// Write atomically using temp file + rename
	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// loadFromDisk loads all indexes from disk into memory.
func (s *ScenarioIndexStore) loadFromDisk() error {
	entries, err := os.ReadDir(s.dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read index data directory: %w", err)
	}

	loadedCount := 0
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(s.dataDir, entry.Name())
		idx, err := s.readFromFile(filePath)
		if err != nil {
			s.logError("failed to load index from disk, skipping", "file", entry.Name(), "error", err)
			continue
		}

		s.indexes[idx.ScenarioName] = idx
		loadedCount++
	}

	if loadedCount > 0 {
		s.logInfo("loaded scenario indexes from disk", "count", loadedCount)
	}

	return nil
}

// readFromFile reads an index from a file.
func (s *ScenarioIndexStore) readFromFile(filePath string) (*ScenarioIndex, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var idx ScenarioIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal index: %w", err)
	}

	return &idx, nil
}

// filePath returns the file path for a scenario index.
func (s *ScenarioIndexStore) filePath(scenarioName string) string {
	return filepath.Join(s.dataDir, scenarioName+".json")
}

// logInfo logs an info message if logger is configured.
func (s *ScenarioIndexStore) logInfo(msg string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Info(msg, args...)
	}
}

// logError logs an error message if logger is configured.
func (s *ScenarioIndexStore) logError(msg string, args ...interface{}) {
	if s.logger != nil {
		s.logger.Error(msg, args...)
	}
}
