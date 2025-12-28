// Package domain defines the core domain types for the scenario-to-cloud scenario.
package domain

import (
	"encoding/json"
	"time"
)

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
	SetupResult       json.RawMessage `json:"setup_result,omitempty"`
	DeployResult      json.RawMessage `json:"deploy_result,omitempty"`
	LastInspectResult json.RawMessage `json:"last_inspect_result,omitempty"`

	// Error tracking
	ErrorMessage *string `json:"error_message,omitempty"`
	ErrorStep    *string `json:"error_step,omitempty"`

	// Timestamps
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	LastDeployedAt  *time.Time `json:"last_deployed_at,omitempty"`
	LastInspectedAt *time.Time `json:"last_inspected_at,omitempty"`
}

// DeploymentSummary is a lightweight view of a deployment for list views.
type DeploymentSummary struct {
	ID             string           `json:"id"`
	Name           string           `json:"name"`
	ScenarioID     string           `json:"scenario_id"`
	Status         DeploymentStatus `json:"status"`
	Domain         string           `json:"domain,omitempty"`
	Host           string           `json:"host,omitempty"`
	ErrorMessage   *string          `json:"error_message,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	LastDeployedAt *time.Time       `json:"last_deployed_at,omitempty"`
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
	Name     string          `json:"name,omitempty"` // Optional, auto-generated if empty
	Manifest json.RawMessage `json:"manifest"`
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
