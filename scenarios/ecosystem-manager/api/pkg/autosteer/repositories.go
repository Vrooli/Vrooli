package autosteer

// ProfileRepository abstracts persistence operations for AutoSteerProfile.
type ProfileRepository interface {
	GetProfile(id string) (*AutoSteerProfile, error)
	CreateProfile(profile *AutoSteerProfile) error
	UpdateProfile(id string, profile *AutoSteerProfile) error
	DeleteProfile(id string) error
	ListProfiles(tags []string) ([]*AutoSteerProfile, error)
}

// Compile-time interface assertion
var _ ProfileRepository = (*ProfileService)(nil)

// ExecutionStateRepository abstracts persistence and state manipulation for ProfileExecutionState.
// This interface includes both persistence operations and state helper methods to enable
// proper dependency injection and testability of components that manage execution state.
type ExecutionStateRepository interface {
	// Persistence operations
	Get(taskID string) (*ProfileExecutionState, error)
	Save(state *ProfileExecutionState) error
	Delete(taskID string) error

	// State initialization and manipulation
	InitializeState(taskID, profileID string, initialMetrics MetricsSnapshot) *ProfileExecutionState
	IncrementIteration(state *ProfileExecutionState, newMetrics MetricsSnapshot)
	AdvanceToNextPhase(state *ProfileExecutionState)

	// Phase lifecycle
	RecordPhaseCompletion(state *ProfileExecutionState, phase SteerPhase, stopReason string) error
	FinalizeExecution(state *ProfileExecutionState, scenarioName string) error
}

// Compile-time interface assertion
var _ ExecutionStateRepository = (*ExecutionStateManager)(nil)

// ExecutionHistoryRepository abstracts persistence operations for profile execution history.
type ExecutionHistoryRepository interface {
	GetHistory(filters HistoryFilters) ([]ProfilePerformance, error)
	GetExecution(executionID string) (*ProfilePerformance, error)
	SubmitFeedback(executionID string, rating int, comments string) error
	SubmitFeedbackEntry(executionID string, req ExecutionFeedbackRequest) (*ExecutionFeedbackEntry, error)
	GetProfileAnalytics(profileID string) (*ProfileAnalytics, error)
}

// Compile-time interface assertion
var _ ExecutionHistoryRepository = (*HistoryService)(nil)

// MetricsProvider abstracts metrics collection for testability.
type MetricsProvider interface {
	CollectMetrics(scenarioName string, phaseLoops, totalLoops int) (*MetricsSnapshot, error)
}

// ProfileExecution represents a completed profile execution for history tracking.
type ProfileExecution struct {
	ProfileID       string
	TaskID          string
	ScenarioName    string
	StartMetrics    MetricsSnapshot
	EndMetrics      MetricsSnapshot
	PhaseBreakdown  []PhasePerformance
	TotalIterations int
	TotalDurationMs int64
}
