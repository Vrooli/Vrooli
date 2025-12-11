package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/services/uxmetrics"
	"github.com/vrooli/browser-automation-studio/services/uxmetrics/contracts"
)

// MockRepository is a test double for uxmetrics.Repository.
type MockRepository struct {
	mu sync.Mutex

	// Saved data (write path)
	SavedTraces      []contracts.InteractionTrace
	SavedCursorPaths map[string]*contracts.CursorPath
	SavedMetrics     map[uuid.UUID]*contracts.ExecutionMetrics

	// Pre-loaded data for queries (read path)
	Traces      []contracts.InteractionTrace
	CursorPaths map[string]*contracts.CursorPath
	Metrics     map[uuid.UUID]*contracts.ExecutionMetrics

	// Control behavior for testing error paths
	SaveError error
	GetError  error
}

// NewMockRepository creates an empty mock repository.
func NewMockRepository() *MockRepository {
	return &MockRepository{
		SavedCursorPaths: make(map[string]*contracts.CursorPath),
		SavedMetrics:     make(map[uuid.UUID]*contracts.ExecutionMetrics),
		CursorPaths:      make(map[string]*contracts.CursorPath),
		Metrics:          make(map[uuid.UUID]*contracts.ExecutionMetrics),
	}
}

// SaveInteractionTrace persists a single interaction trace.
func (m *MockRepository) SaveInteractionTrace(ctx context.Context, trace *contracts.InteractionTrace) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.SaveError != nil {
		return m.SaveError
	}
	m.SavedTraces = append(m.SavedTraces, *trace)
	// Also add to queryable data
	m.Traces = append(m.Traces, *trace)
	return nil
}

// SaveCursorPath persists a cursor path for a step.
func (m *MockRepository) SaveCursorPath(ctx context.Context, executionID uuid.UUID, path *contracts.CursorPath) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.SaveError != nil {
		return m.SaveError
	}
	key := cursorPathKey(executionID, path.StepIndex)
	m.SavedCursorPaths[key] = path
	// Also add to queryable data
	m.CursorPaths[key] = path
	return nil
}

// SaveExecutionMetrics persists computed execution metrics.
func (m *MockRepository) SaveExecutionMetrics(ctx context.Context, metrics *contracts.ExecutionMetrics) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.SaveError != nil {
		return m.SaveError
	}
	m.SavedMetrics[metrics.ExecutionID] = metrics
	// Also add to queryable data
	m.Metrics[metrics.ExecutionID] = metrics
	return nil
}

// GetExecutionMetrics retrieves computed metrics for an execution.
func (m *MockRepository) GetExecutionMetrics(ctx context.Context, executionID uuid.UUID) (*contracts.ExecutionMetrics, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.GetError != nil {
		return nil, m.GetError
	}
	return m.Metrics[executionID], nil
}

// GetStepMetrics retrieves metrics for a specific step.
func (m *MockRepository) GetStepMetrics(ctx context.Context, executionID uuid.UUID, stepIndex int) (*contracts.StepMetrics, error) {
	metrics, err := m.GetExecutionMetrics(ctx, executionID)
	if err != nil || metrics == nil {
		return nil, err
	}
	for _, sm := range metrics.StepMetrics {
		if sm.StepIndex == stepIndex {
			return &sm, nil
		}
	}
	return nil, nil
}

// ListInteractionTraces retrieves all interaction traces for an execution.
func (m *MockRepository) ListInteractionTraces(ctx context.Context, executionID uuid.UUID) ([]contracts.InteractionTrace, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.GetError != nil {
		return nil, m.GetError
	}

	result := make([]contracts.InteractionTrace, 0)
	for _, t := range m.Traces {
		if t.ExecutionID == executionID {
			result = append(result, t)
		}
	}
	return result, nil
}

// GetCursorPath retrieves the cursor path for a specific step.
func (m *MockRepository) GetCursorPath(ctx context.Context, executionID uuid.UUID, stepIndex int) (*contracts.CursorPath, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.GetError != nil {
		return nil, m.GetError
	}

	key := cursorPathKey(executionID, stepIndex)
	return m.CursorPaths[key], nil
}

// GetWorkflowMetricsAggregate computes aggregate metrics across executions.
func (m *MockRepository) GetWorkflowMetricsAggregate(ctx context.Context, workflowID uuid.UUID, limit int) (*contracts.WorkflowMetricsAggregate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.GetError != nil {
		return nil, m.GetError
	}

	return &contracts.WorkflowMetricsAggregate{
		WorkflowID:           workflowID,
		ExecutionCount:       0,
		TrendDirection:       "stable",
		HighFrictionStepFreq: make(map[int]int),
	}, nil
}

// Reset clears all stored data (useful between tests).
func (m *MockRepository) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SavedTraces = nil
	m.SavedCursorPaths = make(map[string]*contracts.CursorPath)
	m.SavedMetrics = make(map[uuid.UUID]*contracts.ExecutionMetrics)
	m.Traces = nil
	m.CursorPaths = make(map[string]*contracts.CursorPath)
	m.Metrics = make(map[uuid.UUID]*contracts.ExecutionMetrics)
	m.SaveError = nil
	m.GetError = nil
}

func cursorPathKey(executionID uuid.UUID, stepIndex int) string {
	return fmt.Sprintf("%s:%d", executionID.String(), stepIndex)
}

// Compile-time interface check
var _ uxmetrics.Repository = (*MockRepository)(nil)
