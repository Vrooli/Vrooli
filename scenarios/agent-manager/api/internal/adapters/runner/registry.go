// Package runner provides runner registry and adapter implementations.
//
// This file contains the runner registry implementation and a mock runner
// for testing purposes.
package runner

import (
	"context"
	"errors"
	"sync"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

// =============================================================================
// Runner Registry Implementation
// =============================================================================

// DefaultRegistry is the standard runner registry implementation.
type DefaultRegistry struct {
	mu      sync.RWMutex
	runners map[domain.RunnerType]Runner
}

// NewRegistry creates a new runner registry.
func NewRegistry() *DefaultRegistry {
	return &DefaultRegistry{
		runners: make(map[domain.RunnerType]Runner),
	}
}

// Register adds a runner to the registry.
func (r *DefaultRegistry) Register(runner Runner) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rt := runner.Type()
	if _, exists := r.runners[rt]; exists {
		return domain.NewValidationErrorWithCode("runnerType", "runner already registered", domain.ErrCodeValidationConflict)
	}

	r.runners[rt] = runner
	return nil
}

// Get retrieves a runner by type.
func (r *DefaultRegistry) Get(runnerType domain.RunnerType) (Runner, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	runner, exists := r.runners[runnerType]
	if !exists {
		return nil, domain.NewNotFoundErrorWithID("Runner", string(runnerType))
	}

	return runner, nil
}

// List returns all registered runners.
func (r *DefaultRegistry) List() []Runner {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Runner, 0, len(r.runners))
	for _, runner := range r.runners {
		result = append(result, runner)
	}
	return result
}

// Available returns runners that are currently available.
func (r *DefaultRegistry) Available(ctx context.Context) []Runner {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Runner
	for _, runner := range r.runners {
		if available, _ := runner.IsAvailable(ctx); available {
			result = append(result, runner)
		}
	}
	return result
}

// Verify interface compliance
var _ Registry = (*DefaultRegistry)(nil)

// =============================================================================
// Mock Runner (for testing)
// =============================================================================

// MockRunner is a test runner that simulates agent execution.
type MockRunner struct {
	runnerType   domain.RunnerType
	capabilities Capabilities
	available    bool
	message      string

	// Execution behavior
	ExecuteFunc func(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error)
	StopFunc    func(ctx context.Context, runID uuid.UUID) error
}

// NewMockRunner creates a new mock runner.
func NewMockRunner(rt domain.RunnerType) *MockRunner {
	return &MockRunner{
		runnerType: rt,
		available:  true,
		message:    "mock runner available",
		capabilities: Capabilities{
			SupportsMessages:     true,
			SupportsToolEvents:   true,
			SupportsCostTracking: false,
			SupportsStreaming:    true,
			SupportsCancellation: true,
			MaxTurns:             100,
			SupportedModels:      []string{"mock-model"},
		},
	}
}

// Type returns the runner type identifier.
func (m *MockRunner) Type() domain.RunnerType {
	return m.runnerType
}

// Capabilities returns what this runner supports.
func (m *MockRunner) Capabilities() Capabilities {
	return m.capabilities
}

// Execute runs the agent with the given configuration.
func (m *MockRunner) Execute(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error) {
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, req)
	}

	// Default mock behavior: simulate successful execution
	return &ExecuteResult{
		Success:  true,
		ExitCode: 0,
		Summary: &domain.RunSummary{
			Description:   "Mock execution completed successfully",
			FilesModified: []string{},
			FilesCreated:  []string{},
			FilesDeleted:  []string{},
		},
	}, nil
}

// Stop attempts to gracefully stop a running agent.
func (m *MockRunner) Stop(ctx context.Context, runID uuid.UUID) error {
	if m.StopFunc != nil {
		return m.StopFunc(ctx, runID)
	}
	return nil
}

// IsAvailable checks if this runner is currently available.
func (m *MockRunner) IsAvailable(ctx context.Context) (bool, string) {
	return m.available, m.message
}

// SetAvailable sets the availability state for testing.
func (m *MockRunner) SetAvailable(available bool, message string) {
	m.available = available
	m.message = message
}

// SetCapabilities sets the capabilities for testing.
func (m *MockRunner) SetCapabilities(caps Capabilities) {
	m.capabilities = caps
}

// Continue continues a previous conversation in the same session.
func (m *MockRunner) Continue(ctx context.Context, req ContinueRequest) (*ExecuteResult, error) {
	// Default mock behavior: simulate successful continuation
	return &ExecuteResult{
		Success:   true,
		ExitCode:  0,
		SessionID: req.SessionID,
		Summary: &domain.RunSummary{
			Description: "Mock continuation completed successfully",
		},
	}, nil
}

// Verify interface compliance
var _ Runner = (*MockRunner)(nil)

// =============================================================================
// Stub Runner (for unavailable runners)
// =============================================================================

// StubRunner represents a runner that exists but is not available.
// Used for runners that are configured but not yet installed.
type StubRunner struct {
	runnerType domain.RunnerType
	message    string
}

// NewStubRunner creates a stub runner for an unavailable runner type.
func NewStubRunner(rt domain.RunnerType, message string) *StubRunner {
	return &StubRunner{
		runnerType: rt,
		message:    message,
	}
}

func (s *StubRunner) Type() domain.RunnerType {
	return s.runnerType
}

func (s *StubRunner) Capabilities() Capabilities {
	return Capabilities{}
}

func (s *StubRunner) Execute(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error) {
	return nil, &domain.RunnerError{
		RunnerType:  s.runnerType,
		Operation:   "availability",
		Cause:       errors.New(s.message),
		IsTransient: false,
	}
}

func (s *StubRunner) Stop(ctx context.Context, runID uuid.UUID) error {
	return &domain.RunnerError{
		RunnerType:  s.runnerType,
		Operation:   "availability",
		Cause:       errors.New(s.message),
		IsTransient: false,
	}
}

func (s *StubRunner) IsAvailable(ctx context.Context) (bool, string) {
	return false, s.message
}

// Continue is not supported for stub runners.
func (s *StubRunner) Continue(ctx context.Context, req ContinueRequest) (*ExecuteResult, error) {
	return nil, ErrContinuationNotSupported
}

// Verify interface compliance
var _ Runner = (*StubRunner)(nil)
