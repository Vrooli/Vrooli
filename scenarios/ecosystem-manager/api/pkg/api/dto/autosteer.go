package dto

// AutoSteerProfileCreateRequest represents a request to create a new Auto Steer profile.
// The actual profile data should be validated by the autosteer package.
type AutoSteerProfileCreateRequest struct {
	ID          string   `json:"id,omitempty"`
	Name        string   `json:"name" validate:"required,min=1,max=100"`
	Description string   `json:"description,omitempty"`
	TaskTypes   []string `json:"task_types,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Phases      []any    `json:"phases,omitempty"` // Phase configuration
	Enabled     bool     `json:"enabled"`
}

// AutoSteerProfileResponse wraps a profile with API metadata.
// Note: For now, handlers return the domain type directly.
// This DTO is provided for future use when we want to add API-specific fields.
type AutoSteerProfileResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Profile any    `json:"profile,omitempty"` // Use any for now to avoid coupling
}

// AutoSteerProfileListResponse represents a list of profiles.
type AutoSteerProfileListResponse struct {
	Profiles []any `json:"profiles"`
	Count    int   `json:"count"`
}

// AutoSteerExecutionStateResponse represents the current execution state.
type AutoSteerExecutionStateResponse struct {
	TaskID     string `json:"task_id"`
	ProfileID  string `json:"profile_id"`
	PhaseIndex int    `json:"phase_index"`
	Iteration  int    `json:"iteration"`
	Status     string `json:"status"`
}

// AutoSteerIterationRequest represents a request for iteration evaluation.
type AutoSteerIterationRequest struct {
	ExecutionID string         `json:"execution_id,omitempty"`
	Output      string         `json:"output,omitempty"`
	Metrics     map[string]any `json:"metrics,omitempty"`
}
