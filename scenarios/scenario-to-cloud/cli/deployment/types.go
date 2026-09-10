// Package deployment provides deployment lifecycle commands for the CLI.
// Typed reads and the plan/apply path use the generated Connect clients;
// the structs below mirror the REST bodies of the surfaces that have no
// generated service yet (snake_case, the proto JSON shape where one exists).
package deployment

import (
	"encoding/json"
	"time"
)

// Record mirrors the REST deployment record as returned by create/execute.
type Record struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	ScenarioID      string          `json:"scenario_id"`
	Environment     string          `json:"environment,omitempty"`
	Status          string          `json:"status"`
	Manifest        json.RawMessage `json:"manifest"`
	BundlePath      *string         `json:"bundle_path,omitempty"`
	BundleSHA256    *string         `json:"bundle_sha256,omitempty"`
	ErrorMessage    *string         `json:"error_message,omitempty"`
	ErrorStep       *string         `json:"error_step,omitempty"`
	ProgressStep    *string         `json:"progress_step,omitempty"`
	ProgressPercent float64         `json:"progress_percent"`
	Fence           uint64          `json:"fence"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	LastDeployedAt  *time.Time      `json:"last_deployed_at,omitempty"`
}

// CreateRequest is the request for creating or updating a deployment.
type CreateRequest struct {
	Name     string          `json:"name,omitempty"`
	Manifest json.RawMessage `json:"manifest"`
}

// CreateResponse is the response from creating a deployment.
type CreateResponse struct {
	Deployment *Record `json:"deployment"`
	Created    bool    `json:"created"`
	Updated    bool    `json:"updated"`
	Timestamp  string  `json:"timestamp"`
}

// DeleteResponse is the response from deleting a deployment.
type DeleteResponse struct {
	Deleted   bool   `json:"deleted"`
	Timestamp string `json:"timestamp"`
}

// DeleteOptions contains options for deployment deletion.
type DeleteOptions struct {
	Stop    bool
	Cleanup bool
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

// RecoveryPoint mirrors data.v1.RecoveryPoint (REST JSON of domain.RecoveryPoint).
type RecoveryPoint struct {
	ID                  string            `json:"id"`
	DeploymentID        string            `json:"deployment_id"`
	OperationID         string            `json:"operation_id,omitempty"`
	BindingIDs          []string          `json:"binding_ids"`
	SchemaVersion       string            `json:"schema_version"`
	ConfigurationDigest string            `json:"configuration_digest"`
	ReleaseDigest       string            `json:"release_digest"`
	CapturedAt          time.Time         `json:"captured_at"`
	Provider            string            `json:"provider"`
	Encrypted           bool              `json:"encrypted"`
	RecoveryKeyRef      string            `json:"recovery_key_ref"`
	RetentionPolicy     string            `json:"retention_policy"`
	Protected           bool              `json:"protected"`
	ProtectedBy         []string          `json:"protected_by,omitempty"`
	MigrationPosture    string            `json:"migration_posture"`
	Location            string            `json:"location"`
	Checksums           map[string]any    `json:"checksums"`
	Consistency         []json.RawMessage `json:"consistency"`
}

// RecoveryPointsResponse is GET /deployments/{id}/recovery-points.
type RecoveryPointsResponse struct {
	SchemaVersion  string          `json:"schema_version"`
	RecoveryPoints []RecoveryPoint `json:"recovery_points"`
}

// RecoveryPointCaptureRequest is the capture body (data.v1.CaptureRecoveryPointRequest).
type RecoveryPointCaptureRequest struct {
	OperationID     string   `json:"operation_id,omitempty"`
	Step            string   `json:"step,omitempty"`
	RecoveryPointID string   `json:"recovery_point_id,omitempty"`
	ReleaseDigest   string   `json:"release_digest,omitempty"`
	SchemaVersion   string   `json:"schema_version,omitempty"`
	RetentionPolicy string   `json:"retention_policy,omitempty"`
	ProtectedBy     []string `json:"protected_by,omitempty"`
}

// RecoveryPointResponse is the capture response.
type RecoveryPointResponse struct {
	SchemaVersion string         `json:"schema_version"`
	RecoveryPoint *RecoveryPoint `json:"recovery_point"`
}

// InvariantResult is one restore/verify invariant.
type InvariantResult struct {
	Binding  string `json:"binding"`
	Check    string `json:"check"`
	Expected string `json:"expected"`
	Observed string `json:"observed"`
	Passed   bool   `json:"passed"`
}

// RecoveryPointVerifyReport is the verify report.
type RecoveryPointVerifyReport struct {
	RecoveryPoint   *RecoveryPoint    `json:"recovery_point"`
	VerifiedAt      time.Time         `json:"verified_at"`
	ArtifactsIntact bool              `json:"artifacts_intact"`
	KeyResolved     bool              `json:"key_resolved"`
	ArtifactsOpened bool              `json:"artifacts_opened"`
	Invariants      []InvariantResult `json:"invariants"`
	Outcome         string            `json:"outcome"`
}

// RecoveryPointVerifyResponse is GET …/recovery-points/{rp}/verify.
type RecoveryPointVerifyResponse struct {
	SchemaVersion string                     `json:"schema_version"`
	Report        *RecoveryPointVerifyReport `json:"report"`
}

// RecoveryPointRestoreRequest is the restore body (data.v1.RestoreRecoveryPointRequest).
type RecoveryPointRestoreRequest struct {
	TargetRef string            `json:"target_ref"`
	Into      map[string]string `json:"into,omitempty"`
	Bindings  []string          `json:"bindings,omitempty"`
}

// RestoreReceipt is the restore receipt.
type RestoreReceipt struct {
	ID               string            `json:"id"`
	RecoveryPointID  string            `json:"recovery_point_id"`
	DeploymentID     string            `json:"deployment_id"`
	TargetRef        string            `json:"target_ref"`
	StartedAt        time.Time         `json:"started_at"`
	CompletedAt      time.Time         `json:"completed_at"`
	InvariantResults []InvariantResult `json:"invariant_results"`
	Outcome          string            `json:"outcome"`
	ErrorCode        string            `json:"error_code,omitempty"`
	ErrorMessage     string            `json:"error_message,omitempty"`
	WithinBudgets    bool              `json:"within_budgets"`
}

// RecoveryPointRestoreResponse is POST …/recovery-points/{rp}/restore.
type RecoveryPointRestoreResponse struct {
	SchemaVersion string          `json:"schema_version"`
	Receipt       *RestoreReceipt `json:"receipt"`
}

// RecoveryRequest is the governed cloud recovery body
// (POST /deployments/{id}/recovery).
type RecoveryRequest struct {
	Action            string `json:"action"`
	ExpectedBundleSHA string `json:"expected_bundle_sha256,omitempty"`
	RepairBundleSHA   string `json:"repair_bundle_sha256,omitempty"`
	DataCompatibility string `json:"data_compatibility,omitempty"`
	IdempotencyKey    string `json:"idempotency_key,omitempty"`
	Confirmation      string `json:"confirmation,omitempty"`
	DryRun            bool   `json:"dry_run"`
	ReviewRef         string `json:"review_ref,omitempty"`
	ReleaseDigest     string `json:"release_digest,omitempty"`
	PreviewRef        string `json:"preview_ref,omitempty"`
}

// RecoveryReceipt mirrors domain.CloudRecoveryReceipt.
type RecoveryReceipt struct {
	SchemaVersion int    `json:"schema_version"`
	DeploymentID  string `json:"deployment_id"`
	OperationID   string `json:"operation_id,omitempty"`
	Action        string `json:"action"`
	Outcome       string `json:"outcome"`
	Health        string `json:"health"`
	BundleSHA256  string `json:"bundle_sha256,omitempty"`
	DryRun        bool   `json:"dry_run"`
	Error         string `json:"error,omitempty"`
	PreviewRef    string `json:"preview_ref,omitempty"`
	ReviewKey     string `json:"review_key,omitempty"`
	ReleaseDigest string `json:"release_digest,omitempty"`
	RouteKind     string `json:"route_kind,omitempty"`
}

// RecoveryResponse is the recovery response.
type RecoveryResponse struct {
	Receipt   *RecoveryReceipt `json:"receipt"`
	Timestamp string           `json:"timestamp"`
}
