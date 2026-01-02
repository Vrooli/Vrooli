// Package domain defines the core domain types for the scenario-to-cloud scenario.
package domain

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// NullRawMessage is a json.RawMessage that handles NULL values from the database.
type NullRawMessage struct {
	Data  json.RawMessage
	Valid bool
}

// Scan implements the sql.Scanner interface.
func (n *NullRawMessage) Scan(value interface{}) error {
	if value == nil {
		n.Data = nil
		n.Valid = false
		return nil
	}
	n.Valid = true
	switch v := value.(type) {
	case []byte:
		n.Data = v
	case string:
		n.Data = []byte(v)
	default:
		n.Data = nil
		n.Valid = false
	}
	return nil
}

// Value implements the driver.Valuer interface.
func (n NullRawMessage) Value() (driver.Value, error) {
	if !n.Valid || n.Data == nil {
		return nil, nil
	}
	return []byte(n.Data), nil
}

// MarshalJSON implements json.Marshaler.
func (n NullRawMessage) MarshalJSON() ([]byte, error) {
	if !n.Valid || n.Data == nil {
		return []byte("null"), nil
	}
	return n.Data, nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (n *NullRawMessage) UnmarshalJSON(data []byte) error {
	if data == nil || string(data) == "null" {
		n.Data = nil
		n.Valid = false
		return nil
	}
	n.Data = data
	n.Valid = true
	return nil
}

// DeploymentStatus represents the current state of a deployment.
type DeploymentStatus string

const (
	StatusPending       DeploymentStatus = "pending"
	StatusSetupRunning  DeploymentStatus = "setup_running"
	StatusSetupComplete DeploymentStatus = "setup_complete"
	StatusDeploying     DeploymentStatus = "deploying"
	StatusDeployed      DeploymentStatus = "deployed"
	StatusFailed        DeploymentStatus = "failed"
	StatusStopped       DeploymentStatus = "stopped"
)

// Deployment represents a deployment record in the database.
// It tracks the full lifecycle of deploying a scenario to a VPS target.
type Deployment struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	ScenarioID      string           `json:"scenario_id"`
	Status          DeploymentStatus `json:"status"`
	Manifest        json.RawMessage  `json:"manifest"`
	BundlePath      *string          `json:"bundle_path,omitempty"`
	BundleSHA256    *string          `json:"bundle_sha256,omitempty"`
	BundleSizeBytes *int64           `json:"bundle_size_bytes,omitempty"`

	// Results from each deployment phase (stored as JSON for flexibility)
	SetupResult       NullRawMessage `json:"setup_result,omitempty"`
	DeployResult      NullRawMessage `json:"deploy_result,omitempty"`
	LastInspectResult NullRawMessage `json:"last_inspect_result,omitempty"`

	// Deployment history (timeline of events)
	DeploymentHistory NullRawMessage `json:"deployment_history,omitempty"`

	// Error tracking
	ErrorMessage *string `json:"error_message,omitempty"`
	ErrorStep    *string `json:"error_step,omitempty"`

	// Progress tracking (for SSE streaming)
	ProgressStep    *string `json:"progress_step,omitempty"`
	ProgressPercent float64 `json:"progress_percent"`

	// Timestamps
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	LastDeployedAt  *time.Time `json:"last_deployed_at,omitempty"`
	LastInspectedAt *time.Time `json:"last_inspected_at,omitempty"`
}

// DeploymentSummary is a lightweight view of a deployment for list views.
type DeploymentSummary struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	ScenarioID      string           `json:"scenario_id"`
	Status          DeploymentStatus `json:"status"`
	Domain          string           `json:"domain,omitempty"`
	Host            string           `json:"host,omitempty"`
	ErrorMessage    *string          `json:"error_message,omitempty"`
	ProgressStep    *string          `json:"progress_step,omitempty"`
	ProgressPercent float64          `json:"progress_percent"`
	CreatedAt       time.Time        `json:"created_at"`
	LastDeployedAt  *time.Time       `json:"last_deployed_at,omitempty"`
}

// ListFilter contains options for filtering deployment lists.
type ListFilter struct {
	Status     *DeploymentStatus `json:"status,omitempty"`
	ScenarioID *string           `json:"scenario_id,omitempty"`
	Limit      int               `json:"limit,omitempty"`
	Offset     int               `json:"offset,omitempty"`
}

// CreateDeploymentRequest is the request body for creating a new deployment.
type CreateDeploymentRequest struct {
	Name            string            `json:"name,omitempty"`              // Optional, auto-generated if empty
	Manifest        json.RawMessage   `json:"manifest"`                    // Required: deployment manifest
	BundlePath      string            `json:"bundle_path,omitempty"`       // Optional: path to pre-built bundle
	BundleSHA256    string            `json:"bundle_sha256,omitempty"`     // Optional: bundle checksum
	BundleSizeBytes int64             `json:"bundle_size_bytes,omitempty"` // Optional: bundle size
	ProvidedSecrets map[string]string `json:"provided_secrets,omitempty"`  // Optional: user-provided secrets (user_prompt class)
}

// ExecuteDeploymentRequest is the request body for executing a deployment.
type ExecuteDeploymentRequest struct {
	ProvidedSecrets map[string]string `json:"provided_secrets,omitempty"` // User-provided secrets (user_prompt class)
}

// UpdateDeploymentStatusRequest is used to update deployment status.
type UpdateDeploymentStatusRequest struct {
	Status       DeploymentStatus `json:"status"`
	ErrorMessage *string          `json:"error_message,omitempty"`
	ErrorStep    *string          `json:"error_step,omitempty"`
}

// DeleteDeploymentRequest contains options for deployment deletion.
type DeleteDeploymentRequest struct {
	StopOnVPS bool `json:"stop_on_vps"` // Whether to stop the deployment on VPS before deleting
}

// HistoryEventType represents the type of deployment event.
type HistoryEventType string

const (
	EventDeploymentCreated  HistoryEventType = "deployment_created"
	EventBundleBuilt        HistoryEventType = "bundle_built"
	EventPreflightStarted   HistoryEventType = "preflight_started"
	EventPreflightCompleted HistoryEventType = "preflight_completed"
	EventSetupStarted       HistoryEventType = "setup_started"
	EventSetupCompleted     HistoryEventType = "setup_completed"
	EventDeployStarted      HistoryEventType = "deploy_started"
	EventDeployCompleted    HistoryEventType = "deploy_completed"
	EventDeployFailed       HistoryEventType = "deploy_failed"
	EventInspection         HistoryEventType = "inspection"
	EventStopped            HistoryEventType = "stopped"
	EventRestarted          HistoryEventType = "restarted"
	EventAutohealTriggered  HistoryEventType = "autoheal_triggered"
)

// HistoryEvent represents a single event in the deployment timeline.
type HistoryEvent struct {
	Type       HistoryEventType `json:"type"`
	Timestamp  time.Time        `json:"timestamp"`
	Message    string           `json:"message,omitempty"`
	Details    string           `json:"details,omitempty"`
	DurationMs int64            `json:"duration_ms,omitempty"`
	Success    *bool            `json:"success,omitempty"`
	BundleHash string           `json:"bundle_hash,omitempty"`
	StepName   string           `json:"step_name,omitempty"`
	Data       json.RawMessage  `json:"data,omitempty"`
}
