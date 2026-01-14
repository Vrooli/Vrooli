// Package deployment provides deployment lifecycle commands for the CLI.
package deployment

import (
	"encoding/json"
	"time"
)

// DeploymentStatus represents the status of a deployment.
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

// Deployment represents a deployment record.
type Deployment struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	ScenarioID      string           `json:"scenario_id"`
	Status          DeploymentStatus `json:"status"`
	Manifest        json.RawMessage  `json:"manifest"`
	BundlePath      *string          `json:"bundle_path,omitempty"`
	BundleSHA256    *string          `json:"bundle_sha256,omitempty"`
	BundleSizeBytes *int64           `json:"bundle_size_bytes,omitempty"`

	ErrorMessage    *string `json:"error_message,omitempty"`
	ErrorStep       *string `json:"error_step,omitempty"`
	ProgressStep    *string `json:"progress_step,omitempty"`
	ProgressPercent float64 `json:"progress_percent"`
	RunID           *string `json:"run_id,omitempty"`

	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	LastDeployedAt *time.Time `json:"last_deployed_at,omitempty"`
}

// DeploymentSummary is a lightweight deployment for list views.
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

// PlanResponse represents the response from deployment plan generation.
type PlanResponse struct {
	Plan      []PlanStep `json:"plan"`
	Timestamp string     `json:"timestamp"`
}

// PlanStep represents a single step in the deployment plan.
type PlanStep struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// CreateRequest is the request for creating a deployment.
type CreateRequest struct {
	Name            string            `json:"name,omitempty"`
	Manifest        json.RawMessage   `json:"manifest"`
	BundlePath      string            `json:"bundle_path,omitempty"`
	BundleSHA256    string            `json:"bundle_sha256,omitempty"`
	BundleSizeBytes int64             `json:"bundle_size_bytes,omitempty"`
	ProvidedSecrets map[string]string `json:"provided_secrets,omitempty"`
}

// CreateResponse is the response from creating a deployment.
type CreateResponse struct {
	Deployment *Deployment `json:"deployment"`
	Created    bool        `json:"created"`
	Updated    bool        `json:"updated"`
	Timestamp  string      `json:"timestamp"`
}

// ListResponse is the response from listing deployments.
type ListResponse struct {
	Deployments []DeploymentSummary `json:"deployments"`
	Timestamp   string              `json:"timestamp"`
}

// GetResponse is the response from getting a deployment.
type GetResponse struct {
	Deployment *Deployment `json:"deployment"`
	Timestamp  string      `json:"timestamp"`
}

// DeleteResponse is the response from deleting a deployment.
type DeleteResponse struct {
	Deleted   bool   `json:"deleted"`
	Timestamp string `json:"timestamp"`
}

// ExecuteRequest is the request for executing a deployment.
type ExecuteRequest struct {
	ProvidedSecrets  map[string]string `json:"provided_secrets,omitempty"`
	RunPreflight     bool              `json:"run_preflight,omitempty"`
	ForceBundleBuild bool              `json:"force_bundle_build,omitempty"`
}

// ExecuteResponse is the response from starting deployment execution.
type ExecuteResponse struct {
	Deployment *Deployment `json:"deployment"`
	RunID      string      `json:"run_id"`
	Message    string      `json:"message"`
	Timestamp  string      `json:"timestamp"`
}

// StartResponse is the response from starting a stopped deployment.
type StartResponse struct {
	DeploymentID string `json:"deployment_id"`
	RunID        string `json:"run_id"`
	Message      string `json:"message"`
	Timestamp    string `json:"timestamp"`
}

// StopResponse is the response from stopping a deployment.
type StopResponse struct {
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
	Timestamp string `json:"timestamp"`
}

// HistoryEvent represents a deployment history event.
type HistoryEvent struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
	Success   *bool     `json:"success,omitempty"`
	Duration  *int64    `json:"duration_ms,omitempty"`
}

// HistoryResponse is the response from getting deployment history.
type HistoryResponse struct {
	DeploymentID string         `json:"deployment_id"`
	Events       []HistoryEvent `json:"events"`
	Timestamp    string         `json:"timestamp"`
}

// DeleteOptions contains options for deployment deletion.
type DeleteOptions struct {
	Stop    bool // Stop the deployment on VPS before deleting
	Cleanup bool // Clean up bundle files
}

// ListOptions contains options for listing deployments.
type ListOptions struct {
	Status     string // Filter by status
	ScenarioID string // Filter by scenario ID
}
