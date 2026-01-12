package preflight

import (
	"encoding/json"
	"time"

	bundleruntime "scenario-to-desktop-runtime"
	runtimeapi "scenario-to-desktop-runtime/api"
	"scenario-to-desktop-runtime/health"
	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

// Request asks the API to dry-run the bundle runtime.
type Request struct {
	BundleManifestPath string            `json:"bundle_manifest_path"`
	BundleRoot         string            `json:"bundle_root,omitempty"`
	Secrets            map[string]string `json:"secrets,omitempty"`
	TimeoutSeconds     int               `json:"timeout_seconds,omitempty"`
	StartServices      bool              `json:"start_services,omitempty"`
	LogTailLines       int               `json:"log_tail_lines,omitempty"`
	LogTailServices    []string          `json:"log_tail_services,omitempty"`
	StatusOnly         bool              `json:"status_only,omitempty"`
	SessionID          string            `json:"session_id,omitempty"`
	SessionTTLSeconds  int               `json:"session_ttl_seconds,omitempty"`
	SessionStop        bool              `json:"session_stop,omitempty"`
}

// Response reports the dry-run validation results.
type Response struct {
	Status       string                       `json:"status"`
	Validation   *runtimeapi.BundleValidationResult `json:"validation,omitempty"`
	Ready        *Ready                       `json:"ready,omitempty"`
	Secrets      []Secret                     `json:"secrets,omitempty"`
	Ports        map[string]map[string]int    `json:"ports,omitempty"`
	Telemetry    *Telemetry                   `json:"telemetry,omitempty"`
	LogTails     []LogTail                    `json:"log_tails,omitempty"`
	Checks       []Check                      `json:"checks,omitempty"`
	Runtime      *Runtime                     `json:"runtime,omitempty"`
	Fingerprints []ServiceFingerprint         `json:"service_fingerprints,omitempty"`
	Errors       []string                     `json:"errors,omitempty"`
	SessionID    string                       `json:"session_id,omitempty"`
	ExpiresAt    string                       `json:"expires_at,omitempty"`
}

// JobStartResponse returns the job identifier for async preflight.
type JobStartResponse struct {
	JobID string `json:"job_id"`
}

// JobStatusResponse reports async preflight job progress.
type JobStatusResponse struct {
	JobID     string    `json:"job_id"`
	Status    string    `json:"status"`
	Steps     []Step    `json:"steps,omitempty"`
	Result    *Response `json:"result,omitempty"`
	Error     string    `json:"error,omitempty"`
	StartedAt string    `json:"started_at,omitempty"`
	UpdatedAt string    `json:"updated_at,omitempty"`
}

// Step tracks the state of a single preflight phase.
type Step struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	State  string `json:"state"`
	Detail string `json:"detail,omitempty"`
}

// Check enumerates a single preflight test case and outcome.
type Check struct {
	ID     string `json:"id"`
	Step   string `json:"step"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// Ready captures readiness results from the runtime control API.
type Ready struct {
	Ready         bool                     `json:"ready"`
	Details       map[string]health.Status `json:"details"`
	GPU           GPU                      `json:"gpu,omitempty"`
	SnapshotAt    string                   `json:"snapshot_at,omitempty"`
	WaitedSeconds int                      `json:"waited_seconds,omitempty"`
}

// GPU surfaces GPU detection info from the runtime.
type GPU struct {
	Available    bool              `json:"available"`
	Method       string            `json:"method,omitempty"`
	Reason       string            `json:"reason,omitempty"`
	Requirements map[string]string `json:"requirements,omitempty"`
}

// Secret mirrors the runtime /secrets view.
type Secret struct {
	ID          string            `json:"id"`
	Class       string            `json:"class"`
	Required    bool              `json:"required"`
	HasValue    bool              `json:"has_value"`
	Description string            `json:"description,omitempty"`
	Format      string            `json:"format,omitempty"`
	Prompt      map[string]string `json:"prompt,omitempty"`
}

// Telemetry provides telemetry file details for diagnostics.
type Telemetry struct {
	Path      string `json:"path"`
	UploadURL string `json:"upload_url,omitempty"`
}

// Runtime reports runtime instance metadata.
type Runtime struct {
	InstanceID     string `json:"instance_id,omitempty"`
	StartedAt      string `json:"started_at,omitempty"`
	AppDataDir     string `json:"app_data_dir,omitempty"`
	BundleRoot     string `json:"bundle_root,omitempty"`
	DryRun         bool   `json:"dry_run,omitempty"`
	ManifestHash   string `json:"manifest_hash,omitempty"`
	ManifestSchema string `json:"manifest_schema,omitempty"`
	Target         string `json:"target,omitempty"`
	AppName        string `json:"app_name,omitempty"`
	AppVersion     string `json:"app_version,omitempty"`
	IPCHost        string `json:"ipc_host,omitempty"`
	IPCPort        int    `json:"ipc_port,omitempty"`
	RuntimeVersion string `json:"runtime_version,omitempty"`
	BuildVersion   string `json:"build_version,omitempty"`
}

// ServiceFingerprint captures service binary metadata for debugging.
type ServiceFingerprint struct {
	ServiceID          string `json:"service_id"`
	Platform           string `json:"platform,omitempty"`
	BinaryPath         string `json:"binary_path,omitempty"`
	BinaryResolvedPath string `json:"binary_resolved_path,omitempty"`
	BinarySHA256       string `json:"binary_sha256,omitempty"`
	BinarySizeBytes    int64  `json:"binary_size_bytes,omitempty"`
	BinaryMtime        string `json:"binary_mtime,omitempty"`
	Error              string `json:"error,omitempty"`
}

// LogTail captures optional log tail diagnostics.
type LogTail struct {
	ServiceID string `json:"service_id"`
	Lines     int    `json:"lines"`
	Content   string `json:"content,omitempty"`
	Error     string `json:"error,omitempty"`
}

// ManifestRequest requests a manifest file to be loaded for display.
type ManifestRequest struct {
	BundleManifestPath string `json:"bundle_manifest_path"`
}

// ManifestResponse provides a parsed manifest payload for UI display.
type ManifestResponse struct {
	Path     string          `json:"path"`
	Manifest json.RawMessage `json:"manifest"`
}

// StatusError wraps an error with an HTTP status code.
type StatusError struct {
	Status int
	Err    error
}

func (e *StatusError) Error() string {
	return e.Err.Error()
}

// issue is an internal type for tracking preflight issues.
type issue struct {
	status string
	detail string
}

// Session represents a long-lived preflight runtime environment.
type Session struct {
	ID         string
	Manifest   *bundlemanifest.Manifest
	BundleDir  string
	AppData    string
	Supervisor *bundleruntime.Supervisor
	BaseURL    string
	Token      string
	CreatedAt  time.Time
	ExpiresAt  time.Time
}

// Job represents an async preflight operation.
type Job struct {
	ID        string
	Status    string
	Steps     map[string]Step
	Result    *Response
	Err       string
	StartedAt time.Time
	UpdatedAt time.Time
}

// RuntimeHandle wraps a runtime client with lifecycle management.
type RuntimeHandle struct {
	Client    RuntimeClient
	Cleanup   func()
	SessionID string
	ExpiresAt time.Time
}
