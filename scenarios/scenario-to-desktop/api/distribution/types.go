package distribution

import (
	"io"
	"time"
)

// SchemaVersion is the current version of the distribution config schema.
const SchemaVersion = "1.0"

// Provider constants for distribution targets.
const (
	ProviderS3           = "s3"
	ProviderR2           = "r2"
	ProviderS3Compatible = "s3-compatible"
)

// Status constants for distribution operations.
const (
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusUploading = "uploading"
	StatusCompleted = "completed"
	StatusPartial   = "partial"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
)

// DistributionConfig is the top-level global distribution configuration.
type DistributionConfig struct {
	// SchemaVersion for migration handling.
	SchemaVersion string `json:"schema_version,omitempty"`

	// Targets is a map of target names to their configurations.
	Targets map[string]*DistributionTarget `json:"targets"`
}

// DistributionTarget represents a single upload destination.
type DistributionTarget struct {
	// Name is the human-readable target name.
	Name string `json:"name"`

	// Enabled controls whether this target is active.
	Enabled bool `json:"enabled"`

	// Provider is the storage provider type: "s3", "r2", "s3-compatible".
	Provider string `json:"provider"`

	// Endpoint is the S3-compatible endpoint URL.
	// Required for R2 and s3-compatible providers.
	// Optional for AWS S3 (uses default endpoint).
	Endpoint string `json:"endpoint,omitempty"`

	// Region is the AWS region or equivalent.
	Region string `json:"region,omitempty"`

	// Bucket is the target bucket name.
	Bucket string `json:"bucket"`

	// PathPrefix is an optional prefix for all uploaded files.
	// Example: "releases/desktop" results in "releases/desktop/v1.0.0/app.exe"
	PathPrefix string `json:"path_prefix,omitempty"`

	// AccessKeyIDEnv is the environment variable name containing the access key ID.
	AccessKeyIDEnv string `json:"access_key_id_env"`

	// SecretAccessKeyEnv is the environment variable name containing the secret key.
	SecretAccessKeyEnv string `json:"secret_access_key_env"`

	// ACL is the access control list for uploaded objects.
	// Values: "private", "public-read", "authenticated-read".
	ACL string `json:"acl,omitempty"`

	// CDNUrl is an optional CDN URL for public download links.
	CDNUrl string `json:"cdn_url,omitempty"`

	// Retry contains retry behavior configuration.
	Retry *RetryConfig `json:"retry,omitempty"`

	// CreatedAt is when the target was created.
	CreatedAt string `json:"created_at,omitempty"`

	// UpdatedAt is when the target was last updated.
	UpdatedAt string `json:"updated_at,omitempty"`
}

// RetryConfig controls upload retry behavior.
type RetryConfig struct {
	// MaxAttempts is the maximum number of upload attempts.
	MaxAttempts int `json:"max_attempts,omitempty"`

	// InitialBackoffMs is the initial backoff duration in milliseconds.
	InitialBackoffMs int `json:"initial_backoff_ms,omitempty"`

	// MaxBackoffMs is the maximum backoff duration in milliseconds.
	MaxBackoffMs int `json:"max_backoff_ms,omitempty"`

	// BackoffMultiplier for exponential backoff.
	BackoffMultiplier float64 `json:"backoff_multiplier,omitempty"`
}

// DefaultRetryConfig provides sensible defaults for retry behavior.
var DefaultRetryConfig = &RetryConfig{
	MaxAttempts:       3,
	InitialBackoffMs:  1000,  // 1 second
	MaxBackoffMs:      30000, // 30 seconds
	BackoffMultiplier: 2.0,
}

// UploadRequest contains parameters for an upload operation.
type UploadRequest struct {
	// LocalPath is the path to the file to upload.
	LocalPath string

	// Key is the destination object key (path in bucket).
	Key string

	// ContentType overrides auto-detected content type.
	ContentType string

	// Metadata to attach to the object.
	Metadata map[string]string

	// ProgressCallback receives upload progress updates.
	ProgressCallback func(bytesUploaded, totalBytes int64)
}

// UploadResult contains the result of an upload operation.
type UploadResult struct {
	// Key is the final object key.
	Key string `json:"key"`

	// URL is the public URL (if applicable).
	URL string `json:"url,omitempty"`

	// ETag is the object's ETag for verification.
	ETag string `json:"etag,omitempty"`

	// Size is the uploaded file size in bytes.
	Size int64 `json:"size"`

	// Duration is how long the upload took.
	Duration time.Duration `json:"duration"`
}

// ObjectInfo contains metadata about a stored object.
type ObjectInfo struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	ETag         string    `json:"etag"`
}

// DistributeRequest contains parameters for a distribution operation.
type DistributeRequest struct {
	// ScenarioName is the scenario being distributed.
	ScenarioName string `json:"scenario_name"`

	// Version is the release version (used in path).
	Version string `json:"version,omitempty"`

	// Artifacts maps platform names to artifact paths.
	// Example: {"win": "/path/to/app.msi", "mac": "/path/to/app.dmg"}
	Artifacts map[string]string `json:"artifacts"`

	// TargetNames specifies which targets to upload to.
	// Empty means all enabled targets.
	TargetNames []string `json:"target_names,omitempty"`

	// Parallel enables concurrent uploads to multiple targets.
	Parallel bool `json:"parallel,omitempty"`

	// InlineCredentials provides credentials directly instead of from env vars.
	// Keys are env var names (e.g., "R2_ACCESS_KEY_ID"), values are the actual secrets.
	// Used when env vars aren't set and user provides credentials via UI.
	// These are never persisted - only used for this single operation.
	InlineCredentials map[string]string `json:"inline_credentials,omitempty"`
}

// CheckCredentialsRequest contains parameters for checking credentials availability.
type CheckCredentialsRequest struct {
	// TargetNames specifies which targets to check.
	// Empty means all enabled targets.
	TargetNames []string `json:"target_names,omitempty"`
}

// CheckCredentialsResponse contains the credential check results.
type CheckCredentialsResponse struct {
	// AllPresent is true if all required credentials are available.
	AllPresent bool `json:"all_present"`

	// Targets contains per-target credential status.
	Targets map[string]*TargetCredentialStatus `json:"targets"`
}

// TargetCredentialStatus contains credential status for a single target.
type TargetCredentialStatus struct {
	// TargetName is the target being checked.
	TargetName string `json:"target_name"`

	// AllPresent is true if all credentials for this target are available.
	AllPresent bool `json:"all_present"`

	// MissingCredentials lists env var names that are not set.
	MissingCredentials []string `json:"missing_credentials,omitempty"`

	// RequiredCredentials lists all env var names needed for this target.
	RequiredCredentials []string `json:"required_credentials"`
}

// DistributeResponse contains the result of starting a distribution operation.
type DistributeResponse struct {
	DistributionID string `json:"distribution_id"`
	Status         string `json:"status"`
	StatusURL      string `json:"status_url,omitempty"`
}

// DistributionStatus tracks the status of a distribution operation.
type DistributionStatus struct {
	DistributionID string                         `json:"distribution_id"`
	ScenarioName   string                         `json:"scenario_name"`
	Version        string                         `json:"version,omitempty"`
	Status         string                         `json:"status"` // pending, running, completed, partial, failed
	StartedAt      int64                          `json:"started_at"`
	CompletedAt    int64                          `json:"completed_at,omitempty"`
	Targets        map[string]*TargetDistribution `json:"targets"`
	Error          string                         `json:"error,omitempty"`
}

// TargetDistribution tracks distribution to a single target.
type TargetDistribution struct {
	TargetName  string                     `json:"target_name"`
	Status      string                     `json:"status"` // pending, uploading, completed, failed
	StartedAt   int64                      `json:"started_at,omitempty"`
	CompletedAt int64                      `json:"completed_at,omitempty"`
	Uploads     map[string]*PlatformUpload `json:"uploads"` // keyed by platform
	Error       string                     `json:"error,omitempty"`
}

// PlatformUpload tracks upload of a single platform artifact.
type PlatformUpload struct {
	Platform      string `json:"platform"`
	Status        string `json:"status"` // pending, uploading, completed, failed
	LocalPath     string `json:"local_path"`
	RemoteKey     string `json:"remote_key,omitempty"`
	URL           string `json:"url,omitempty"`
	Size          int64  `json:"size,omitempty"`
	BytesUploaded int64  `json:"bytes_uploaded,omitempty"`
	Error         string `json:"error,omitempty"`
}

// ValidationResult contains target validation results.
type ValidationResult struct {
	Valid   bool                         `json:"valid"`
	Targets map[string]*TargetValidation `json:"targets"`
}

// TargetValidation contains validation for a single target.
type TargetValidation struct {
	TargetName  string   `json:"target_name"`
	Valid       bool     `json:"valid"`
	Connected   bool     `json:"connected"`
	Permissions bool     `json:"permissions"`
	Errors      []string `json:"errors,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
}

// ProgressReader wraps an io.Reader to track read progress.
type ProgressReader struct {
	reader      io.Reader
	total       int64
	read        int64
	callback    func(bytesRead, total int64)
	lastReport  int64
	reportEvery int64 // Report every N bytes
}

// NewProgressReader creates a progress-tracking reader.
func NewProgressReader(r io.Reader, total int64, callback func(bytesRead, total int64)) *ProgressReader {
	return &ProgressReader{
		reader:      r,
		total:       total,
		callback:    callback,
		reportEvery: 1024 * 1024, // Report every 1MB
	}
}

// Read implements io.Reader.
func (pr *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)
	pr.read += int64(n)

	// Report progress at intervals
	if pr.callback != nil && (pr.read-pr.lastReport) >= pr.reportEvery {
		pr.callback(pr.read, pr.total)
		pr.lastReport = pr.read
	}

	return n, err
}
