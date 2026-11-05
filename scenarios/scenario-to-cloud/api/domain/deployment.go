// Package domain defines the core domain types for the scenario-to-cloud scenario.
package domain

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"scenario-to-cloud/identity"
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

// DesiredState is the operator's intent for a deployment's workload.
type DesiredState string

// Desired states.
const (
	DesiredRunning DesiredState = "running"
	DesiredStopped DesiredState = "stopped"
	DesiredRetired DesiredState = "retired"
)

// Normalized returns the state with the running default applied.
func (d DesiredState) Normalized() DesiredState {
	if d == "" {
		return DesiredRunning
	}
	return d
}

// PersistentDataBindings is the recorded legacy conversion of a deployment:
// each mapping binds one mutable directory (scenario-relative) to a declared
// binding id. Unmapped directories are carried, never deleted.
type PersistentDataBindings struct {
	SchemaVersion int                       `json:"schema_version"`
	Mappings      []PersistentDataMapping   `json:"mappings"`
	RecordedAt    time.Time                 `json:"recorded_at"`
	Inventory     []PersistentDataInventory `json:"inventory,omitempty"`
}

// PersistentDataMapping maps <scenario>/<path> onto a binding id.
type PersistentDataMapping struct {
	BindingID string `json:"binding_id"`
	Scenario  string `json:"scenario"`
	Path      string `json:"path"`
}

// PersistentDataInventory is one observed mutable directory at adoption time.
type PersistentDataInventory struct {
	Scenario string `json:"scenario"`
	Path     string `json:"path"`
	Bytes    int64  `json:"bytes"`
	Files    int64  `json:"files"`
	Mapped   bool   `json:"mapped"`
}

// Specs renders the mappings as <binding id>=<scenario>/<path> (the plan
// compiler's input form).
func (b PersistentDataBindings) Specs() []string {
	out := make([]string, 0, len(b.Mappings))
	for _, m := range b.Mappings {
		out = append(out, m.BindingID+"="+m.Scenario+"/"+m.Path)
	}
	return out
}

// Deployment represents a deployment record in the database.
// It tracks the full lifecycle of deploying a scenario to a VPS target.
type Deployment struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ScenarioID string `json:"scenario_id"`
	// Environment separates installations of one scenario (production,
	// staging, ...). Empty on input means identity.DefaultEnvironment.
	Environment string `json:"environment"`
	// Target is the bound target identity. The manifest's target.vps block
	// remains the SSH transport configuration until the reach layer owns it.
	Target identity.TargetRef `json:"target"`
	// Fence is incremented whenever an operation acquires the deployment;
	// every target effect carries it and refuses a lower value.
	Fence uint64 `json:"fence"`
	// DesiredState is the operator's intent for the workload: running or
	// stopped. It is distinct from Status (the observed/projected lifecycle)
	// so an intentional stop is never read as drift and never restarted by
	// observation. Empty means running.
	DesiredState DesiredState `json:"desired_state"`
	// PersistentData records the legacy data-binding conversion: which
	// mutable directories map onto which declared bindings. It is written by
	// the adopt endpoint and read by release.activate; nothing else moves data.
	PersistentData  NullRawMessage   `json:"persistent_data,omitempty"`
	Status          DeploymentStatus `json:"status"`
	Manifest        json.RawMessage  `json:"manifest"`
	BundlePath      *string          `json:"bundle_path,omitempty"`
	BundleSHA256    *string          `json:"bundle_sha256,omitempty"`
	BundleSizeBytes *int64           `json:"bundle_size_bytes,omitempty"`

	// Results from each deployment phase (stored as JSON for flexibility)
	SetupResult       NullRawMessage `json:"setup_result,omitempty"`
	DeployResult      NullRawMessage `json:"deploy_result,omitempty"`
	PreflightResult   NullRawMessage `json:"preflight_result,omitempty"`
	LastInspectResult NullRawMessage `json:"last_inspect_result,omitempty"`
	SSHIdentity       NullRawMessage `json:"ssh_identity,omitempty"`

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

// Ref returns the stable identity of the deployment.
func (d *Deployment) Ref() identity.DeploymentRef {
	return identity.DeploymentRef{
		ID:          d.ID,
		ScenarioID:  d.ScenarioID,
		Environment: identity.NormalizeEnvironment(d.Environment),
		Target:      d.Target,
	}
}

// DeploymentSummary is a lightweight view of a deployment for list views.
type DeploymentSummary struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	ScenarioID      string           `json:"scenario_id"`
	Environment     string           `json:"environment"`
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
	Status      *DeploymentStatus `json:"status,omitempty"`
	ScenarioID  *string           `json:"scenario_id,omitempty"`
	Environment *string           `json:"environment,omitempty"`
	Limit       int               `json:"limit,omitempty"`
	Offset      int               `json:"offset,omitempty"`
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
	ProvidedSecrets  map[string]string `json:"provided_secrets,omitempty"`   // User-provided secrets (user_prompt class)
	RunPreflight     bool              `json:"run_preflight,omitempty"`      // Run VPS preflight checks before deployment
	ForceBundleBuild bool              `json:"force_bundle_build,omitempty"` // Build a new bundle even if one exists
	// RequestKey is the caller's idempotency key. Equal replays return the
	// same operation; a missing key admits a fresh operation.
	RequestKey string `json:"request_key,omitempty"`
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
	EventManifestRefreshed  HistoryEventType = "manifest_refreshed"
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
	EventStarted            HistoryEventType = "started"
	EventRestarted          HistoryEventType = "restarted"
	EventAutohealTriggered  HistoryEventType = "autoheal_triggered"
	// EventVPSBundleGC records automatic or manual garbage collection of VPS bundle cache.
	EventVPSBundleGC HistoryEventType = "vps_bundle_gc"
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
