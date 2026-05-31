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

	return cloneProfile(profile), nil
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

	r.profiles[profile.ID] = cloneProfile(profile)

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

	r.profiles[id] = cloneProfile(profile)

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
		result = append(result, cloneProfile(profile))
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

// MockExecutionStateRepository is an in-memory ExecutionStateRepository for testing.
type MockExecutionStateRepository struct {
	mu     sync.RWMutex
	states map[string]*ProfileExecutionState

	// Error injection
	GetError               error
	SaveError              error
	DeleteError            error
	FinalizeExecutionError error

	// Call tracking
	FinalizedTasks []string
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

	clone := *state
	return &clone, nil
}

// Save persists the execution state.
func (r *MockExecutionStateRepository) Save(state *ProfileExecutionState) error {
	if r.SaveError != nil {
		return r.SaveError
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	clone := *state
	r.states[state.TaskID] = &clone

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

// InitializeState creates a new controller state for a task.
func (r *MockExecutionStateRepository) InitializeState(taskID, profileID string) *ProfileExecutionState {
	now := time.Now()
	return &ProfileExecutionState{
		TaskID:       taskID,
		ProfileID:    profileID,
		ScoreHistory: []float64{},
		Trace:        []DecisionTraceEntry{},
		StartedAt:    now,
		LastUpdated:  now,
	}
}

// FinalizeExecution removes the active state (real impl archives to history).
func (r *MockExecutionStateRepository) FinalizeExecution(state *ProfileExecutionState, scenarioName string) error {
	if r.FinalizeExecutionError != nil {
		return r.FinalizeExecutionError
	}
	r.mu.Lock()
	r.FinalizedTasks = append(r.FinalizedTasks, state.TaskID)
	r.mu.Unlock()
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
	r.FinalizeExecutionError = nil
	r.FinalizedTasks = nil
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

// MockPromptEnhancerAPI is a mock implementation of PromptEnhancerAPI for testing.
type MockPromptEnhancerAPI struct {
	mu sync.Mutex

	// Configurable return values
	SkillSetSectionResult   string
	ControllerSectionResult string

	// Call tracking
	GenerateSkillSetSectionCallCount   int
	GenerateControllerSectionCallCount int
	LastSkillSet                       []string
}

// Compile-time interface assertion
var _ PromptEnhancerAPI = (*MockPromptEnhancerAPI)(nil)

// NewMockPromptEnhancerAPI creates a new mock prompt enhancer.
func NewMockPromptEnhancerAPI() *MockPromptEnhancerAPI {
	return &MockPromptEnhancerAPI{
		SkillSetSectionResult:   "## Mock Skill Set Section\nFocus on testing.",
		ControllerSectionResult: "## Auto Steer\nMock controller instructions.",
	}
}

// GenerateSkillSetSection renders a standalone section for a specific skill set.
func (m *MockPromptEnhancerAPI) GenerateSkillSetSection(skillIDs []string, withScope bool, scope string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GenerateSkillSetSectionCallCount++
	m.LastSkillSet = append([]string(nil), skillIDs...)

	return m.SkillSetSectionResult
}

// GenerateControllerSection generates the controller section for agent prompts.
func (m *MockPromptEnhancerAPI) GenerateControllerSection(state *ProfileExecutionState, profile *AutoSteerProfile) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GenerateControllerSectionCallCount++

	return m.ControllerSectionResult
}

// Reset clears call tracking.
func (m *MockPromptEnhancerAPI) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.GenerateSkillSetSectionCallCount = 0
	m.GenerateControllerSectionCallCount = 0
	m.LastSkillSet = nil
}
