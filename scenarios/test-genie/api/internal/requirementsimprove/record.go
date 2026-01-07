package requirementsimprove

import (
	"sync"
	"time"
)

// Status represents the state of a requirements improve operation.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

// ActionType defines what the AI should do.
type ActionType string

const (
	ActionWriteTests ActionType = "write_tests"
	ActionUpdateReqs ActionType = "update_requirements"
	ActionBoth       ActionType = "both"
)

// ValidationInfo contains information about a validation.
type ValidationInfo struct {
	Type       string `json:"type"`
	Ref        string `json:"ref"`
	Phase      string `json:"phase,omitempty"`
	Status     string `json:"status"`
	LiveStatus string `json:"liveStatus"`
}

// RequirementInfo contains information about a requirement to improve.
type RequirementInfo struct {
	ID          string           `json:"id"`
	Title       string           `json:"title"`
	Description string           `json:"description,omitempty"`
	Status      string           `json:"status"`
	LiveStatus  string           `json:"liveStatus"`
	Criticality string           `json:"criticality,omitempty"`
	ModulePath  string           `json:"modulePath"`
	Validations []ValidationInfo `json:"validations,omitempty"`
}

// Record represents a requirements improve operation for a scenario.
type Record struct {
	ID           string            `json:"id"`
	ScenarioName string            `json:"scenarioName"`
	Requirements []RequirementInfo `json:"requirements"`
	ActionType   ActionType        `json:"actionType"`
	Status       Status            `json:"status"`
	RunID        string            `json:"runId,omitempty"`
	Tag          string            `json:"tag,omitempty"`
	StartedAt    time.Time         `json:"startedAt"`
	CompletedAt  *time.Time        `json:"completedAt,omitempty"`
	Output       string            `json:"output,omitempty"`
	Error        string            `json:"error,omitempty"`
}

// IsTerminal returns true if the operation is in a terminal state.
func (r *Record) IsTerminal() bool {
	return r.Status == StatusCompleted || r.Status == StatusFailed || r.Status == StatusCancelled
}

// Store provides in-memory storage for requirements improve records.
type Store struct {
	mu      sync.RWMutex
	records map[string]*Record
	// byScenario maps scenario name to list of improve IDs (most recent first)
	byScenario map[string][]string
}

// NewStore creates a new requirements improve record store.
func NewStore() *Store {
	return &Store{
		records:    make(map[string]*Record),
		byScenario: make(map[string][]string),
	}
}

// Create stores a new improve record.
func (s *Store) Create(record *Record) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records[record.ID] = record
	s.byScenario[record.ScenarioName] = append(
		[]string{record.ID},
		s.byScenario[record.ScenarioName]...,
	)
}

// Get retrieves an improve record by ID.
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

// Update modifies an existing improve record.
func (s *Store) Update(record *Record) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.records[record.ID]; ok {
		s.records[record.ID] = record
	}
}

// ListByScenario returns all improve records for a scenario (most recent first).
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

// GetActiveForScenario returns the active (non-terminal) improve for a scenario, if any.
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

// Delete removes an improve record by ID.
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
