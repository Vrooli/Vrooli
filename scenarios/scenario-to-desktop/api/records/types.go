package records

import "time"

// DesktopAppRecord captures where a generated desktop wrapper currently lives
// and where it is intended to move.
type DesktopAppRecord struct {
	ID              string    `json:"id"`
	BuildID         string    `json:"build_id"`
	ScenarioName    string    `json:"scenario_name"`
	AppDisplayName  string    `json:"app_display_name,omitempty"`
	TemplateType    string    `json:"template_type,omitempty"`
	Framework       string    `json:"framework,omitempty"`
	LocationMode    string    `json:"location_mode,omitempty"`
	OutputPath      string    `json:"output_path"`
	DestinationPath string    `json:"destination_path,omitempty"`
	StagingPath     string    `json:"staging_path,omitempty"`
	CustomPath      string    `json:"custom_path,omitempty"`
	DeploymentMode  string    `json:"deployment_mode,omitempty"`
	Icon            string    `json:"icon,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// RecordWithBuild combines a record with its associated build status.
type RecordWithBuild struct {
	Record     *DesktopAppRecord `json:"record"`
	Build      *BuildStatusView  `json:"build_status,omitempty"`
	HasBuild   bool              `json:"has_build"`
	BuildState string            `json:"build_state,omitempty"`
}

// BuildStatusView is a simplified view of build status for record responses.
type BuildStatusView struct {
	Status     string                 `json:"status"`
	OutputPath string                 `json:"output_path,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// MoveResult contains the result of a move operation.
type MoveResult struct {
	RecordID string `json:"record_id"`
	From     string `json:"from"`
	To       string `json:"to"`
	Status   string `json:"status"`
}

// MoveRequest is the request payload for moving a record.
type MoveRequest struct {
	Target          string `json:"target"`           // "destination" (default) or "custom"
	DestinationPath string `json:"destination_path"` // required when target == "custom"
}

// ListResponse is the response for listing records.
type ListResponse struct {
	Records []RecordWithBuild `json:"records"`
}
