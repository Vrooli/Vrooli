package build

import (
	"fmt"
	"time"
)

// Status represents the status of a desktop application build.
type Status struct {
	BuildID            string                     `json:"build_id"`
	ScenarioName       string                     `json:"scenario_name"`
	Status             string                     `json:"status"` // building, ready, partial, failed
	Framework          string                     `json:"framework"`
	TemplateType       string                     `json:"template_type"`
	Platforms          []string                   `json:"platforms"`           // Legacy: platforms that were built
	RequestedPlatforms []string                   `json:"requested_platforms"` // NEW: platforms that were requested to build
	PlatformResults    map[string]*PlatformResult `json:"platform_results,omitempty"`
	OutputPath         string                     `json:"output_path"`
	CreatedAt          time.Time                  `json:"created_at"`
	CompletedAt        *time.Time                 `json:"completed_at,omitempty"`
	ErrorLog           []string                   `json:"error_log,omitempty"`
	BuildLog           []string                   `json:"build_log,omitempty"`
	Artifacts          map[string]string          `json:"artifacts,omitempty"`
	Metadata           map[string]interface{}     `json:"metadata,omitempty"`
}

// PlatformResult represents the result of building for a specific platform.
type PlatformResult struct {
	Platform    string              `json:"platform"`
	Status      PlatformBuildStatus `json:"status"` // pending, building, ready, failed, skipped
	StartedAt   *time.Time          `json:"started_at,omitempty"`
	CompletedAt *time.Time          `json:"completed_at,omitempty"`
	ErrorLog    []string            `json:"error_log,omitempty"`
	Artifact    string              `json:"artifact,omitempty"`
	FileSize    int64               `json:"file_size,omitempty"`
	SkipReason  string              `json:"skip_reason,omitempty"` // e.g., "Wine not installed"
}

// PlatformBuildStatus is the lifecycle of one platform build. The empty value
// is accepted as the legacy initial state for records created before status was
// initialized.
type PlatformBuildStatus string

const (
	PlatformBuildPending  PlatformBuildStatus = "pending"
	PlatformBuildBuilding PlatformBuildStatus = "building"
	PlatformBuildReady    PlatformBuildStatus = "ready"
	PlatformBuildFailed   PlatformBuildStatus = "failed"
	PlatformBuildSkipped  PlatformBuildStatus = "skipped"
)

// TransitionStatus is the single owner of platform build status changes.
func (r *PlatformResult) TransitionStatus(next PlatformBuildStatus) error {
	if r == nil {
		return fmt.Errorf("platform result is nil")
	}
	if !r.Status.CanTransitionTo(next) {
		return fmt.Errorf("invalid platform build transition %q -> %q", r.Status, next)
	}
	r.Status = next
	return nil
}

func (s PlatformBuildStatus) CanTransitionTo(next PlatformBuildStatus) bool {
	if s == next {
		return true
	}
	switch s {
	case "", PlatformBuildPending:
		return next == PlatformBuildBuilding || next == PlatformBuildFailed || next == PlatformBuildSkipped
	case PlatformBuildBuilding:
		return next == PlatformBuildReady || next == PlatformBuildFailed || next == PlatformBuildSkipped
	case PlatformBuildFailed, PlatformBuildSkipped:
		// A caller may retry a failed or previously unavailable platform.
		return next == PlatformBuildBuilding
	default:
		return false
	}
}

// BuildRequest represents a request to build a desktop application.
type BuildRequest struct {
	DesktopPath string   `json:"desktop_path"`
	Platforms   []string `json:"platforms"`
	Sign        bool     `json:"sign"`
	Publish     bool     `json:"publish"`
}

// ScenarioBuildRequest represents a request to build a scenario's desktop app.
type ScenarioBuildRequest struct {
	ScenarioName string   `json:"scenario_name"`
	DesktopPath  string   `json:"desktop_path"`
	Platforms    []string `json:"platforms"`
	Clean        bool     `json:"clean"`
}

// BuildResponse represents the response from a build endpoint.
type BuildResponse struct {
	BuildID   string `json:"build_id"`
	Status    string `json:"status"`
	StatusURL string `json:"status_url,omitempty"`
}
