package autosteer

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MockProfileRepository is an in-memory implementation of ProfileRepository for testing.
type MockProfileRepository struct {
	mu       sync.RWMutex
	profiles map[string]*AutoSteerProfile

	// Error injection for testing error paths
	GetProfileError    error
	CreateProfileError error
	UpdateProfileError error
	DeleteProfileError error
	ListProfilesError  error
}

// Compile-time interface assertion
var _ ProfileRepository = (*MockProfileRepository)(nil)

// NewMockProfileRepository creates a new mock profile repository.
func NewMockProfileRepository() *MockProfileRepository {
	return &MockProfileRepository{
		profiles: make(map[string]*AutoSteerProfile),
	}
}

// GetProfile retrieves a profile by ID.
func (r *MockProfileRepository) GetProfile(id string) (*AutoSteerProfile, error) {
	if r.GetProfileError != nil {
		return nil, r.GetProfileError
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	profile, ok := r.profiles[id]
	if !ok {
		return nil, fmt.Errorf("profile not found: %s", id)
	}

	// Return a copy to prevent mutation
	copy := *profile
	return &copy, nil
}

// CreateProfile inserts a new profile.
func (r *MockProfileRepository) CreateProfile(profile *AutoSteerProfile) error {
	if r.CreateProfileError != nil {
		return r.CreateProfileError
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if profile.ID == "" {
		profile.ID = uuid.New().String()
	}

	now := time.Now()
	profile.CreatedAt = now
	profile.UpdatedAt = now

	// Store a copy
	copy := *profile
	r.profiles[profile.ID] = &copy

	return nil
}

// UpdateProfile updates an existing profile.
func (r *MockProfileRepository) UpdateProfile(id string, profile *AutoSteerProfile) error {
	if r.UpdateProfileError != nil {
		return r.UpdateProfileError
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.profiles[id]; !ok {
		return fmt.Errorf("profile not found: %s", id)
	}

	profile.ID = id
	profile.UpdatedAt = time.Now()

	copy := *profile
	r.profiles[id] = &copy

	return nil
}

// DeleteProfile removes a profile by ID.
func (r *MockProfileRepository) DeleteProfile(id string) error {
	if r.DeleteProfileError != nil {
		return r.DeleteProfileError
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.profiles[id]; !ok {
		return fmt.Errorf("profile not found: %s", id)
	}

	delete(r.profiles, id)
	return nil
}

// ListProfiles retrieves all profiles with optional tag filtering.
func (r *MockProfileRepository) ListProfiles(tags []string) ([]*AutoSteerProfile, error) {
	if r.ListProfilesError != nil {
		return nil, r.ListProfilesError
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*AutoSteerProfile, 0, len(r.profiles))
	for _, profile := range r.profiles {
		if len(tags) > 0 && !hasAnyTag(profile.Tags, tags) {
			continue
		}
		copy := *profile
		result = append(result, &copy)
	}

	return result, nil
}

// Reset clears all profiles (useful between tests).
func (r *MockProfileRepository) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.profiles = make(map[string]*AutoSteerProfile)
	r.GetProfileError = nil
	r.CreateProfileError = nil
	r.UpdateProfileError = nil
	r.DeleteProfileError = nil
	r.ListProfilesError = nil
}

// MockExecutionStateRepository is an in-memory implementation of ExecutionStateRepository for testing.
type MockExecutionStateRepository struct {
	mu     sync.RWMutex
	states map[string]*ProfileExecutionState

	// Error injection
	GetError                   error
	SaveError                  error
	DeleteError                error
	RecordPhaseCompletionError error
	FinalizeExecutionError     error
}

// Compile-time interface assertion
var _ ExecutionStateRepository = (*MockExecutionStateRepository)(nil)

// NewMockExecutionStateRepository creates a new mock execution state repository.
func NewMockExecutionStateRepository() *MockExecutionStateRepository {
	return &MockExecutionStateRepository{
		states: make(map[string]*ProfileExecutionState),
	}
}

// Get retrieves the execution state for a task.
func (r *MockExecutionStateRepository) Get(taskID string) (*ProfileExecutionState, error) {
	if r.GetError != nil {
		return nil, r.GetError
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	state, ok := r.states[taskID]
	if !ok {
		return nil, nil
	}

	copy := *state
	return &copy, nil
}

// Save persists the execution state.
func (r *MockExecutionStateRepository) Save(state *ProfileExecutionState) error {
	if r.SaveError != nil {
		return r.SaveError
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	copy := *state
	r.states[state.TaskID] = &copy

	return nil
}

// Delete removes the execution state for a task.
func (r *MockExecutionStateRepository) Delete(taskID string) error {
	if r.DeleteError != nil {
		return r.DeleteError
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.states, taskID)
	return nil
}

// InitializeState creates a new execution state for a task.
func (r *MockExecutionStateRepository) InitializeState(taskID, profileID string, initialMetrics MetricsSnapshot) *ProfileExecutionState {
	now := time.Now()
	return &ProfileExecutionState{
		TaskID:                taskID,
		ProfileID:             profileID,
		CurrentPhaseIndex:     0,
		CurrentPhaseIteration: 0,
		AutoSteerIteration:    0,
		PhaseStartedAt:        now,
		PhaseHistory:          []PhaseExecution{},
		Metrics:               initialMetrics,
		PhaseStartMetrics:     initialMetrics,
		StartedAt:             now,
		LastUpdated:           now,
	}
}

// IncrementIteration increments the iteration counters and updates metrics.
func (r *MockExecutionStateRepository) IncrementIteration(state *ProfileExecutionState, newMetrics MetricsSnapshot) {
	state.AutoSteerIteration++
	state.CurrentPhaseIteration++
	state.Metrics = newMetrics
	state.LastUpdated = time.Now()
}

// AdvanceToNextPhase updates the state to move to the next phase.
func (r *MockExecutionStateRepository) AdvanceToNextPhase(state *ProfileExecutionState) {
	state.CurrentPhaseIndex++
	state.CurrentPhaseIteration = 0
	state.PhaseStartMetrics = state.Metrics
	state.PhaseStartedAt = time.Now()
	state.LastUpdated = time.Now()
}

// RecordPhaseCompletion appends a completed phase to the execution history.
func (r *MockExecutionStateRepository) RecordPhaseCompletion(state *ProfileExecutionState, phase SteerPhase, stopReason string) error {
	if r.RecordPhaseCompletionError != nil {
		return r.RecordPhaseCompletionError
	}

	now := time.Now()
	phaseExecution := PhaseExecution{
		PhaseID:      phase.ID,
		Mode:         phase.Mode,
		Iterations:   state.CurrentPhaseIteration,
		StartMetrics: state.PhaseStartMetrics,
		EndMetrics:   state.Metrics,
		Commits:      []string{},
		StartedAt:    state.PhaseStartedAt,
		CompletedAt:  &now,
		StopReason:   stopReason,
	}

	state.PhaseHistory = append(state.PhaseHistory, phaseExecution)
	return r.Save(state)
}

// FinalizeExecution archives the completed execution and removes active state.
func (r *MockExecutionStateRepository) FinalizeExecution(state *ProfileExecutionState, scenarioName string) error {
	if r.FinalizeExecutionError != nil {
		return r.FinalizeExecutionError
	}

	// In mock, just delete the active state (real impl would archive to history)
	return r.Delete(state.TaskID)
}

// Reset clears all states.
func (r *MockExecutionStateRepository) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.states = make(map[string]*ProfileExecutionState)
	r.GetError = nil
	r.SaveError = nil
	r.DeleteError = nil
	r.RecordPhaseCompletionError = nil
	r.FinalizeExecutionError = nil
}

// MockMetricsProvider is a mock implementation of MetricsProvider for testing.
type MockMetricsProvider struct {
	// Metrics to return
	Metrics *MetricsSnapshot

	// Error to return
	Error error

	// Call tracking
	mu             sync.Mutex
	CallCount      int
	LastScenario   string
	LastPhaseLoops int
	LastTotalLoops int
}

// Compile-time interface assertion
var _ MetricsProvider = (*MockMetricsProvider)(nil)

// NewMockMetricsProvider creates a new mock metrics provider.
func NewMockMetricsProvider() *MockMetricsProvider {
	return &MockMetricsProvider{
		Metrics: &MetricsSnapshot{
			OperationalTargetsPercentage: 50.0,
			TotalLoops:                   0,
			PhaseLoops:                   0,
			BuildStatus:                  1, // 1 = passing
		},
	}
}

// CollectMetrics returns the configured mock metrics.
func (m *MockMetricsProvider) CollectMetrics(scenarioName string, phaseLoops, totalLoops int) (*MetricsSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CallCount++
	m.LastScenario = scenarioName
	m.LastPhaseLoops = phaseLoops
	m.LastTotalLoops = totalLoops

	if m.Error != nil {
		return nil, m.Error
	}

	// Return a copy with updated loop counts
	metrics := *m.Metrics
	metrics.PhaseLoops = phaseLoops
	metrics.TotalLoops = totalLoops

	return &metrics, nil
}

// Reset clears call tracking and error state.
func (m *MockMetricsProvider) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CallCount = 0
	m.Error = nil
	m.LastScenario = ""
	m.LastPhaseLoops = 0
	m.LastTotalLoops = 0
}

// MockIterationEvaluatorAPI is a mock implementation of IterationEvaluatorAPI for testing.
type MockIterationEvaluatorAPI struct {
	mu sync.Mutex

	// Configurable return values
	EvaluateResult                          *IterationEvaluation
	EvaluateError                           error
	EvaluateWithoutMetricsCollectionResult  *IterationEvaluation
	EvaluateWithoutMetricsCollectionError   error

	// Call tracking
	EvaluateCallCount                         int
	EvaluateWithoutMetricsCollectionCallCount int
	LastTaskID                                string
	LastScenarioName                          string
}

// Compile-time interface assertion
var _ IterationEvaluatorAPI = (*MockIterationEvaluatorAPI)(nil)

// NewMockIterationEvaluatorAPI creates a new mock iteration evaluator.
func NewMockIterationEvaluatorAPI() *MockIterationEvaluatorAPI {
	return &MockIterationEvaluatorAPI{
		EvaluateResult: &IterationEvaluation{
			ShouldStop: false,
			Reason:     "",
		},
		EvaluateWithoutMetricsCollectionResult: &IterationEvaluation{
			ShouldStop: false,
			Reason:     "",
		},
	}
}

// Evaluate evaluates the current iteration for a task.
func (m *MockIterationEvaluatorAPI) Evaluate(taskID string, scenarioName string) (*IterationEvaluation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.EvaluateCallCount++
	m.LastTaskID = taskID
	m.LastScenarioName = scenarioName

	if m.EvaluateError != nil {
		return nil, m.EvaluateError
	}

	return m.EvaluateResult, nil
}

// EvaluateWithoutMetricsCollection evaluates using existing metrics in the state.
func (m *MockIterationEvaluatorAPI) EvaluateWithoutMetricsCollection(taskID string) (*IterationEvaluation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.EvaluateWithoutMetricsCollectionCallCount++
	m.LastTaskID = taskID

	if m.EvaluateWithoutMetricsCollectionError != nil {
		return nil, m.EvaluateWithoutMetricsCollectionError
	}

	return m.EvaluateWithoutMetricsCollectionResult, nil
}

// Reset clears call tracking and error state.
func (m *MockIterationEvaluatorAPI) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.EvaluateCallCount = 0
	m.EvaluateWithoutMetricsCollectionCallCount = 0
	m.EvaluateError = nil
	m.EvaluateWithoutMetricsCollectionError = nil
	m.LastTaskID = ""
	m.LastScenarioName = ""
}

// MockPromptEnhancerAPI is a mock implementation of PromptEnhancerAPI for testing.
type MockPromptEnhancerAPI struct {
	mu sync.Mutex

	// Configurable return values
	ModeSectionResult           string
	AutoSteerSectionResult      string
	PhaseTransitionResult       string
	CompletionMessageResult     string

	// Call tracking
	GenerateModeSectionCallCount        int
	GenerateAutoSteerSectionCallCount   int
	GeneratePhaseTransitionCallCount    int
	GenerateCompletionMessageCallCount  int
	LastMode                            SteerMode
}

// Compile-time interface assertion
var _ PromptEnhancerAPI = (*MockPromptEnhancerAPI)(nil)

// NewMockPromptEnhancerAPI creates a new mock prompt enhancer.
func NewMockPromptEnhancerAPI() *MockPromptEnhancerAPI {
	return &MockPromptEnhancerAPI{
		ModeSectionResult:      "## Mock Mode Section\nFocus on testing.",
		AutoSteerSectionResult: "## Auto Steer\nMock steering instructions.",
		PhaseTransitionResult:  "## Phase Transition\nMock transition message.",
		CompletionMessageResult: "## Complete\nMock completion message.",
	}
}

// GenerateModeSection renders a standalone section for a specific mode.
func (m *MockPromptEnhancerAPI) GenerateModeSection(mode SteerMode) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GenerateModeSectionCallCount++
	m.LastMode = mode

	return m.ModeSectionResult
}

// GenerateAutoSteerSection generates the full Auto Steer section for agent prompts.
func (m *MockPromptEnhancerAPI) GenerateAutoSteerSection(state *ProfileExecutionState, profile *AutoSteerProfile, evaluator ConditionEvaluatorAPI) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GenerateAutoSteerSectionCallCount++

	return m.AutoSteerSectionResult
}

// GeneratePhaseTransitionMessage generates a message for phase transitions.
func (m *MockPromptEnhancerAPI) GeneratePhaseTransitionMessage(oldPhase, newPhase SteerPhase, phaseNumber, totalPhases int) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GeneratePhaseTransitionCallCount++

	return m.PhaseTransitionResult
}

// GenerateCompletionMessage generates a message when all phases are complete.
func (m *MockPromptEnhancerAPI) GenerateCompletionMessage(profile *AutoSteerProfile, state *ProfileExecutionState) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GenerateCompletionMessageCallCount++

	return m.CompletionMessageResult
}

// Reset clears call tracking.
func (m *MockPromptEnhancerAPI) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.GenerateModeSectionCallCount = 0
	m.GenerateAutoSteerSectionCallCount = 0
	m.GeneratePhaseTransitionCallCount = 0
	m.GenerateCompletionMessageCallCount = 0
	m.LastMode = ""
}

// MockPhaseCoordinatorAPI is a mock implementation of PhaseCoordinatorAPI for testing.
type MockPhaseCoordinatorAPI struct {
	mu sync.Mutex

	// Configurable return values
	ShouldAdvancePhaseResult        PhaseAdvanceDecision
	EvaluateQualityGatesResult      []QualityGateEvaluation
	ShouldHaltOnQualityGatesResult  struct {
		Halt       bool
		FailedGate string
		Message    string
	}
	DetermineStopReasonResult string
	EvaluatorResult           ConditionEvaluatorAPI

	// Call tracking
	ShouldAdvancePhaseCallCount       int
	EvaluateQualityGatesCallCount     int
	ShouldHaltOnQualityGatesCallCount int
	DetermineStopReasonCallCount      int
	EvaluatorCallCount                int
}

// Compile-time interface assertion
var _ PhaseCoordinatorAPI = (*MockPhaseCoordinatorAPI)(nil)

// NewMockPhaseCoordinatorAPI creates a new mock phase coordinator.
func NewMockPhaseCoordinatorAPI() *MockPhaseCoordinatorAPI {
	return &MockPhaseCoordinatorAPI{
		ShouldAdvancePhaseResult: PhaseAdvanceDecision{
			ShouldStop: false,
			Reason:     "continue",
		},
		EvaluateQualityGatesResult: []QualityGateEvaluation{},
		DetermineStopReasonResult:  "max_iterations",
		EvaluatorResult:            NewConditionEvaluator(),
	}
}

// ShouldAdvancePhase evaluates whether the current phase should stop.
func (m *MockPhaseCoordinatorAPI) ShouldAdvancePhase(phase SteerPhase, metrics MetricsSnapshot, currentIteration int) PhaseAdvanceDecision {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ShouldAdvancePhaseCallCount++
	return m.ShouldAdvancePhaseResult
}

// EvaluateQualityGates evaluates all quality gates against current metrics.
func (m *MockPhaseCoordinatorAPI) EvaluateQualityGates(gates []QualityGate, metrics MetricsSnapshot) []QualityGateEvaluation {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.EvaluateQualityGatesCallCount++
	return m.EvaluateQualityGatesResult
}

// ShouldHaltOnQualityGates checks if any quality gate failures should halt progression.
func (m *MockPhaseCoordinatorAPI) ShouldHaltOnQualityGates(evaluations []QualityGateEvaluation) (halt bool, failedGate string, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ShouldHaltOnQualityGatesCallCount++
	return m.ShouldHaltOnQualityGatesResult.Halt,
		m.ShouldHaltOnQualityGatesResult.FailedGate,
		m.ShouldHaltOnQualityGatesResult.Message
}

// DetermineStopReason determines the stop reason for phase completion.
func (m *MockPhaseCoordinatorAPI) DetermineStopReason(currentIteration, maxIterations int) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.DetermineStopReasonCallCount++
	return m.DetermineStopReasonResult
}

// Evaluator returns the condition evaluator for prompt generation.
func (m *MockPhaseCoordinatorAPI) Evaluator() ConditionEvaluatorAPI {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.EvaluatorCallCount++
	return m.EvaluatorResult
}

// Reset clears call tracking.
func (m *MockPhaseCoordinatorAPI) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ShouldAdvancePhaseCallCount = 0
	m.EvaluateQualityGatesCallCount = 0
	m.ShouldHaltOnQualityGatesCallCount = 0
	m.DetermineStopReasonCallCount = 0
	m.EvaluatorCallCount = 0
}

// Helper functions

func hasAnyTag(profileTags, filterTags []string) bool {
	tagSet := make(map[string]struct{}, len(filterTags))
	for _, tag := range filterTags {
		tagSet[tag] = struct{}{}
	}

	for _, tag := range profileTags {
		if _, ok := tagSet[tag]; ok {
			return true
		}
	}

	return false
}
