package build

import "time"

// AutoBuildPlatformStatus describes build progress for one platform.
type AutoBuildPlatformStatus struct {
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	OutputPath  string     `json:"output_path,omitempty"`
	Error       string     `json:"error,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// AutoBuildTargetStatus captures progress for a single build target.
type AutoBuildTargetStatus struct {
	ID        string                    `json:"id"`
	Folder    string                    `json:"folder"`
	Platforms []AutoBuildPlatformStatus `json:"platforms"`
}

// AutoBuildStatus tracks the async auto-build lifecycle.
type AutoBuildStatus struct {
	BuildID      string                  `json:"build_id"`
	Scenario     string                  `json:"scenario"`
	Status       string                  `json:"status"`
	Message      string                  `json:"message,omitempty"`
	CreatedAt    time.Time               `json:"created_at"`
	StartedAt    *time.Time              `json:"started_at,omitempty"`
	CompletedAt  *time.Time              `json:"completed_at,omitempty"`
	Targets      []AutoBuildTargetStatus `json:"targets"`
	BuildLog     []string                `json:"build_log,omitempty"`
	ErrorLog     []string                `json:"error_log,omitempty"`
	StatusURL    string                  `json:"status_url,omitempty"`
	CheckCommand string                  `json:"check_command,omitempty"`
	PollAfterMs  int                     `json:"poll_after_ms,omitempty"`
}
