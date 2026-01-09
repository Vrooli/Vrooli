package main

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

// DesktopAppRecord captures where a generated desktop wrapper currently lives and
// where it is intended to move. This enables later management (moving from staging
// to canonical destination, renaming, etc.).
type DesktopAppRecord struct {
	ID              string    `json:"id"`
	BuildID         string    `json:"build_id"`
	ScenarioName    string    `json:"scenario_name"`
	AppDisplayName  string    `json:"app_display_name,omitempty"`
	TemplateType    string    `json:"template_type,omitempty"`
	Framework       string    `json:"framework,omitempty"`
	LocationMode    string    `json:"location_mode,omitempty"`
	OutputPath      string    `json:"output_path"`
	DestinationPath string    `json:"destination_path,omitempty"`
	StagingPath     string    `json:"staging_path,omitempty"`
	CustomPath      string    `json:"custom_path,omitempty"`
	DeploymentMode  string    `json:"deployment_mode,omitempty"`
	Icon            string    `json:"icon,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type DesktopRecordStore struct {
	mu      sync.RWMutex
	records map[string]*DesktopAppRecord
	path    string
}

func NewDesktopRecordStore(path string) (*DesktopRecordStore, error) {
	store := &DesktopRecordStore{
		records: make(map[string]*DesktopAppRecord),
		path:    path,
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *DesktopRecordStore) load() error {
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
	cleaned := false
	for _, r := range records {
		if r == nil || r.ID == "" {
			continue
		}
		if isTestRecord(r) {
			cleaned = true
			continue
		}
		s.records[r.ID] = r
	}
	if cleaned {
		_ = s.persist()
	}
	return nil
}

func (s *DesktopRecordStore) persist() error {
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
func (s *DesktopRecordStore) Upsert(record *DesktopAppRecord) error {
	if record == nil || record.ID == "" {
		return errors.New("record must have an ID")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	record.UpdatedAt = now
	if existing, ok := s.records[record.ID]; ok && !existing.CreatedAt.IsZero() {
		record.CreatedAt = existing.CreatedAt
	} else {
		record.CreatedAt = now
	}
	s.records[record.ID] = record
	return s.persist()
}

// List returns a copy of all records for future management features.
func (s *DesktopRecordStore) List() []*DesktopAppRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*DesktopAppRecord, 0, len(s.records))
	for _, r := range s.records {
		out = append(out, r)
	}
	return out
}

// Get returns a record by ID.
func (s *DesktopRecordStore) Get(id string) (*DesktopAppRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.records[id]
	return rec, ok
}

// isTestRecord filters out historic test artifacts that were written to /tmp during CI runs,
// preventing the UI from showing thousands of irrelevant entries.
func isTestRecord(r *DesktopAppRecord) bool {
	if r == nil {
		return false
	}
	return strings.Contains(r.OutputPath, "/tmp/scenario-to-desktop-test")
}

// DeleteByScenario removes all records associated with a scenario name.
// Returns the number of records removed.
func (s *DesktopRecordStore) DeleteByScenario(scenario string) int {
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
