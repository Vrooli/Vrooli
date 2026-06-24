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
var _ ProfileRepository = (*FileProfileRepository)(nil)

// ExecutionStateRepository abstracts persistence and lifecycle for the
// controller's ProfileExecutionState.
type ExecutionStateRepository interface {
	// Persistence operations
	Get(taskID string) (*ProfileExecutionState, error)
	Save(state *ProfileExecutionState) error
	Delete(taskID string) error

	// InitializeState creates a fresh controller state for a task.
	InitializeState(taskID, profileID string) *ProfileExecutionState

	// FinalizeExecution archives the completed run to history and removes the
	// active state.
	FinalizeExecution(state *ProfileExecutionState, scenarioName string) error
}

// Compile-time interface assertion
var _ ExecutionStateRepository = (*ExecutionStateManager)(nil)

// ExecutionHistoryRepository abstracts persistence operations for profile execution history.
type ExecutionHistoryRepository interface {
	GetHistory(filters HistoryFilters) ([]ProfilePerformance, error)
	GetExecution(executionID string) (*ProfilePerformance, error)
	GetProfileAnalytics(profileID string) (*ProfileAnalytics, error)
}

// Compile-time interface assertion
var _ ExecutionHistoryRepository = (*HistoryService)(nil)
