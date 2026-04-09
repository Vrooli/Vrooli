package records

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileStore is a file-based implementation of the record Store.
type FileStore struct {
	mu      sync.RWMutex
	records map[string]*DesktopAppRecord
	path    string
}

// NewFileStore creates a new file-based record store.
func NewFileStore(path string) (*FileStore, error) {
	store := &FileStore{
		records: make(map[string]*DesktopAppRecord),
		path:    path,
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *FileStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.path == "" {
		return errors.New("record store path is empty")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create record directory: %w", err)
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read record store: %w", err)
	}

	var records []*DesktopAppRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return fmt.Errorf("parse record store: %w", err)
	}

	cleaned := s.deduplicateRecords(records)
	if cleaned {
		_ = s.persist()
	}
	return nil
}

// deduplicateRecords filters test records and deduplicates by scenario+output,
// keeping the newest record per key. Returns true if any records were removed.
func (s *FileStore) deduplicateRecords(records []*DesktopAppRecord) bool {
	cleaned := false
	// Track latest record per scenario+output to deduplicate on load.
	// Key: "scenarioName\x00outputPath" → newest record ID.
	latest := map[string]string{}
	for _, r := range records {
		if r == nil || r.ID == "" {
			continue
		}
		if isTestRecord(r) {
			cleaned = true
			continue
		}
		s.records[r.ID] = r

		if r.ScenarioName == "" || r.OutputPath == "" {
			continue
		}
		key := r.ScenarioName + "\x00" + r.OutputPath
		prevID, exists := latest[key]
		if !exists {
			latest[key] = r.ID
			continue
		}
		// Keep the one with the later UpdatedAt (or the newer insertion).
		prev := s.records[prevID]
		if prev != nil && !r.UpdatedAt.After(prev.UpdatedAt) {
			delete(s.records, r.ID)
		} else {
			delete(s.records, prevID)
			latest[key] = r.ID
		}
		cleaned = true
	}
	return cleaned
}

func (s *FileStore) persist() error {
	if s.path == "" {
		return errors.New("record store path is empty")
	}
	var records []*DesktopAppRecord
	for _, r := range s.records {
		records = append(records, r)
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal record store: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("write record store: %w", err)
	}
	return nil
}

// Upsert saves or updates a record keyed by ID.
func (s *FileStore) Upsert(record *DesktopAppRecord) error {
	if record == nil || record.ID == "" {
		return errors.New("record must have an ID")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	record.UpdatedAt = now

	// Deduplicate: if another record already exists for the same scenario and
	// output path, remove it so we don't accumulate stale entries across
	// repeated pipeline runs that write to the same location.
	if record.ScenarioName != "" && record.OutputPath != "" {
		for id, existing := range s.records {
			if id == record.ID {
				continue
			}
			if existing != nil &&
				existing.ScenarioName == record.ScenarioName &&
				existing.OutputPath == record.OutputPath {
				delete(s.records, id)
			}
		}
	}

	if existing, ok := s.records[record.ID]; ok && !existing.CreatedAt.IsZero() {
		record.CreatedAt = existing.CreatedAt
	} else {
		record.CreatedAt = now
	}
	s.records[record.ID] = record
	return s.persist()
}

// List returns a copy of all records.
func (s *FileStore) List() []*DesktopAppRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*DesktopAppRecord, 0, len(s.records))
	for _, r := range s.records {
		out = append(out, r)
	}
	return out
}

// Get returns a record by ID.
func (s *FileStore) Get(id string) (*DesktopAppRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.records[id]
	return rec, ok
}

// DeleteByScenario removes all records associated with a scenario name.
// Returns the number of records removed.
func (s *FileStore) DeleteByScenario(scenario string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	removed := 0
	for id, rec := range s.records {
		if rec != nil && rec.ScenarioName == scenario {
			delete(s.records, id)
			removed++
		}
	}
	if removed > 0 {
		_ = s.persist()
	}
	return removed
}

// isTestRecord filters out historic test artifacts that were written to /tmp during CI runs.
func isTestRecord(r *DesktopAppRecord) bool {
	if r == nil {
		return false
	}
	return strings.Contains(r.OutputPath, "/tmp/scenario-to-desktop-test")
}
