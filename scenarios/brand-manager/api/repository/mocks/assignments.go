package mocks

import (
	"context"
	"database/sql"
	"sync"

	"brand-manager/domain"
)

// AssignmentRepository is an in-memory mock implementing repository.AssignmentRepository.
type AssignmentRepository struct {
	mu          sync.RWMutex
	assignments map[string]*domain.Assignment // keyed by ID
	byScenario  map[string]string             // scenario_name → assignment ID

	// Error overrides.
	CreateErr        error
	GetByScenarioErr error
	ListByBrandErr   error
	DeleteErr        error
}

// NewAssignmentRepository creates a ready-to-use mock AssignmentRepository.
func NewAssignmentRepository() *AssignmentRepository {
	return &AssignmentRepository{
		assignments: make(map[string]*domain.Assignment),
		byScenario:  make(map[string]string),
	}
}

func (m *AssignmentRepository) Create(_ context.Context, a *domain.Assignment) error {
	if m.CreateErr != nil {
		return m.CreateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	// INSERT OR REPLACE semantics: remove old assignment for same scenario
	if oldID, ok := m.byScenario[a.ScenarioName]; ok {
		delete(m.assignments, oldID)
	}
	cp := *a
	m.assignments[a.ID] = &cp
	m.byScenario[a.ScenarioName] = a.ID
	return nil
}

func (m *AssignmentRepository) GetByScenario(_ context.Context, scenarioName string) (*domain.Assignment, error) {
	if m.GetByScenarioErr != nil {
		return nil, m.GetByScenarioErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.byScenario[scenarioName]
	if !ok {
		return nil, sql.ErrNoRows
	}
	a := m.assignments[id]
	cp := *a
	return &cp, nil
}

func (m *AssignmentRepository) ListByBrandID(_ context.Context, brandID string) ([]*domain.Assignment, error) {
	if m.ListByBrandErr != nil {
		return nil, m.ListByBrandErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*domain.Assignment
	for _, a := range m.assignments {
		if a.BrandID == brandID {
			cp := *a
			result = append(result, &cp)
		}
	}
	return result, nil
}

func (m *AssignmentRepository) Delete(_ context.Context, id string) error {
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.assignments[id]
	if !ok {
		return sql.ErrNoRows
	}
	delete(m.byScenario, a.ScenarioName)
	delete(m.assignments, id)
	return nil
}
