package fix

import (
	"sync"
	"time"
)

// Status represents the state of a fix operation.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

// PhaseInfo contains information about a phase to fix.
type PhaseInfo struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Error    string `json:"error,omitempty"`
	Duration int    `json:"durationSeconds,omitempty"`
	LogPath  string `json:"logPath,omitempty"`
}

// Record represents a fix operation for a scenario.
type Record struct {
	ID           string      `json:"id"`
	ScenarioName string      `json:"scenarioName"`
	Phases       []PhaseInfo `json:"phases"`
	Status       Status      `json:"status"`
	RunID        string      `json:"runId,omitempty"`
	Tag          string      `json:"tag,omitempty"`
	StartedAt    time.Time   `json:"startedAt"`
	CompletedAt  *time.Time  `json:"completedAt,omitempty"`
	Output       string      `json:"output,omitempty"`
	Error        string      `json:"error,omitempty"`
}

// IsTerminal returns true if the fix is in a terminal state.
func (r *Record) IsTerminal() bool {
	return r.Status == StatusCompleted || r.Status == StatusFailed || r.Status == StatusCancelled
}

// Store provides in-memory storage for fix records.
type Store struct {
	mu      sync.RWMutex
	records map[string]*Record
	// byScenario maps scenario name to list of fix IDs (most recent first)
	byScenario map[string][]string
}

// NewStore creates a new fix record store.
func NewStore() *Store {
	return &Store{
		records:    make(map[string]*Record),
		byScenario: make(map[string][]string),
	}
}

// Create stores a new fix record.
func (s *Store) Create(record *Record) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records[record.ID] = record
	s.byScenario[record.ScenarioName] = append(
		[]string{record.ID},
		s.byScenario[record.ScenarioName]...,
	)
}

// Get retrieves a fix record by ID.
func (s *Store) Get(id string) (*Record, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, ok := s.records[id]
	if !ok {
		return nil, false
	}
	// Return a copy to prevent concurrent modification
	copy := *record
	return &copy, true
}

// Update modifies an existing fix record.
func (s *Store) Update(record *Record) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.records[record.ID]; ok {
		s.records[record.ID] = record
	}
}

// ListByScenario returns all fix records for a scenario (most recent first).
func (s *Store) ListByScenario(scenarioName string, limit int) []*Record {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := s.byScenario[scenarioName]
	if len(ids) == 0 {
		return nil
	}

	if limit > 0 && len(ids) > limit {
		ids = ids[:limit]
	}

	records := make([]*Record, 0, len(ids))
	for _, id := range ids {
		if r, ok := s.records[id]; ok {
			copy := *r
			records = append(records, &copy)
		}
	}
	return records
}

// GetActiveForScenario returns the active (non-terminal) fix for a scenario, if any.
func (s *Store) GetActiveForScenario(scenarioName string) *Record {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := s.byScenario[scenarioName]
	for _, id := range ids {
		if r, ok := s.records[id]; ok && !r.IsTerminal() {
			copy := *r
			return &copy
		}
	}
	return nil
}

// Delete removes a fix record by ID.
func (s *Store) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.records[id]
	if !ok {
		return
	}

	delete(s.records, id)

	// Remove from byScenario index
	ids := s.byScenario[record.ScenarioName]
	for i, rid := range ids {
		if rid == id {
			s.byScenario[record.ScenarioName] = append(ids[:i], ids[i+1:]...)
			break
		}
	}
}
