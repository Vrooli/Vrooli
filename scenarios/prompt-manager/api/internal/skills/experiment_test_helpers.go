package skills

import (
	"context"
	"errors"

	"prompt-manager/internal/store"
)

type mockExperimentStore struct {
	experiments  map[string]*store.Experiment
	outcomes     map[string][]store.ExperimentOutcome
	serves       []store.ExperimentServe
	assignments  []store.ExperimentAssignment
	exposures    []store.ExperimentExposure
	auditReceipt *store.ExperimentAuditReceipt
}

func (m *mockExperimentStore) RecordExposure(_ context.Context, exposure store.ExperimentExposure) error {
	m.exposures = append(m.exposures, exposure)
	return nil
}

func (m *mockExperimentStore) ListExposures(_ context.Context, _ string) ([]store.ExperimentExposure, error) {
	return append([]store.ExperimentExposure(nil), m.exposures...), nil
}

func (m *mockExperimentStore) ListAssignments(_ context.Context, _ string) ([]store.ExperimentAssignment, error) {
	return append([]store.ExperimentAssignment(nil), m.assignments...), nil
}

func (m *mockExperimentStore) GetAssignment(_ context.Context, experimentID, key string) (*store.ExperimentAssignment, error) {
	for i := range m.assignments {
		if m.assignments[i].ExperimentID == experimentID && m.assignments[i].IdempotencyKey == key {
			return &m.assignments[i], nil
		}
	}
	return nil, errors.New("assignment not found")
}

func (m *mockExperimentStore) CreateAssignment(_ context.Context, assignment store.ExperimentAssignment) error {
	m.assignments = append(m.assignments, assignment)
	return nil
}

func (m *mockExperimentStore) RecordAuditReceipt(_ context.Context, receipt store.ExperimentAuditReceipt) error {
	m.auditReceipt = &receipt
	return nil
}

func (m *mockExperimentStore) GetAuditReceipt(_ context.Context, _ string) (*store.ExperimentAuditReceipt, error) {
	if m.auditReceipt == nil {
		return nil, errors.New("audit receipt not found")
	}
	return m.auditReceipt, nil
}

func newMockExperimentStore() *mockExperimentStore {
	return &mockExperimentStore{
		experiments: make(map[string]*store.Experiment),
		outcomes:    make(map[string][]store.ExperimentOutcome),
	}
}

func (m *mockExperimentStore) List(_ context.Context) ([]store.Experiment, error) {
	var result []store.Experiment
	for _, e := range m.experiments {
		result = append(result, *e)
	}
	return result, nil
}

func (m *mockExperimentStore) ListBySkill(_ context.Context, skillID string) ([]store.Experiment, error) {
	var result []store.Experiment
	for _, e := range m.experiments {
		if e.SkillID == skillID {
			result = append(result, *e)
		}
	}
	return result, nil
}

func (m *mockExperimentStore) Get(_ context.Context, id string) (*store.Experiment, error) {
	if e, ok := m.experiments[id]; ok {
		return e, nil
	}
	return nil, errors.New("experiment not found")
}

func (m *mockExperimentStore) Create(_ context.Context, exp *store.Experiment) error {
	if _, ok := m.experiments[exp.ID]; ok {
		return errors.New("experiment already exists")
	}
	exp.Kind = store.KindExperiment
	exp.SchemaVersion = store.CurrentSchemaVersion
	if exp.Status == "" {
		exp.Status = store.ExperimentStatusDraft
	}
	exp.Timestamps = store.NewTimestamps()
	m.experiments[exp.ID] = exp
	return nil
}

func (m *mockExperimentStore) Update(_ context.Context, id string, exp *store.Experiment) error {
	if _, ok := m.experiments[id]; !ok {
		return errors.New("experiment not found")
	}
	exp.UpdateTimestamp()
	m.experiments[id] = exp
	return nil
}

func (m *mockExperimentStore) Delete(_ context.Context, id string) error {
	if _, ok := m.experiments[id]; !ok {
		return errors.New("experiment not found")
	}
	delete(m.experiments, id)
	delete(m.outcomes, id)
	return nil
}

func (m *mockExperimentStore) RecordOutcome(_ context.Context, id string, outcome store.ExperimentOutcome) error {
	if _, ok := m.experiments[id]; !ok {
		return errors.New("experiment not found")
	}
	m.outcomes[id] = append(m.outcomes[id], outcome)
	return nil
}

func (m *mockExperimentStore) RecordServe(_ context.Context, serve store.ExperimentServe) error {
	if _, ok := m.experiments[serve.ExperimentID]; !ok {
		return errors.New("experiment not found")
	}
	m.serves = append(m.serves, serve)
	return nil
}

func (m *mockExperimentStore) ListServes(_ context.Context, id string) ([]store.ExperimentServe, error) {
	if _, ok := m.experiments[id]; !ok {
		return nil, errors.New("experiment not found")
	}
	var result []store.ExperimentServe
	for _, s := range m.serves {
		if s.ExperimentID == id {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockExperimentStore) CountServesByVariant(_ context.Context, id string) (map[string]int, error) {
	if _, ok := m.experiments[id]; !ok {
		return nil, errors.New("experiment not found")
	}
	counts := make(map[string]int)
	for _, s := range m.serves {
		if s.ExperimentID == id {
			counts[s.VariantID]++
		}
	}
	return counts, nil
}

func (m *mockExperimentStore) ListOutcomes(_ context.Context, id string) ([]store.ExperimentOutcome, error) {
	if _, ok := m.experiments[id]; !ok {
		return nil, errors.New("experiment not found")
	}
	return m.outcomes[id], nil
}

func (m *mockExperimentStore) CountOutcomesByVariant(_ context.Context, id string) (map[string]int, error) {
	if _, ok := m.experiments[id]; !ok {
		return nil, errors.New("experiment not found")
	}
	counts := make(map[string]int)
	for _, o := range m.outcomes[id] {
		counts[o.VariantID]++
	}
	return counts, nil
}
