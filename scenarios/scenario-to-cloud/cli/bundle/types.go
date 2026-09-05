// Package bundle provides bundle management commands for the CLI.
package bundle

// BuildResponse represents the response from bundle build.
type BuildResponse struct {
	Artifact  Artifact `json:"artifact"`
	Issues    []string `json:"issues,omitempty"`
	Timestamp string   `json:"timestamp"`
}

// Artifact represents a built bundle.
type Artifact struct {
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

// ListResponse represents the response from listing bundles.
type ListResponse struct {
	Bundles   []BundleInfo `json:"bundles"`
	Timestamp string       `json:"timestamp"`
}

// BundleInfo represents information about a stored bundle.
type BundleInfo struct {
	Path       string `json:"path,omitempty"`
	Filename   string `json:"filename"`
	ScenarioID string `json:"scenario_id"`
	Sha256     string `json:"sha256"`
	SizeBytes  int64  `json:"size_bytes"`
	CreatedAt  string `json:"created_at"`
}

// StatsResponse represents bundle storage statistics.
type StatsResponse struct {
	Stats     BundleStats `json:"stats"`
	Timestamp string      `json:"timestamp"`
}

type BundleStats struct {
	TotalCount      int                     `json:"total_count"`
	TotalSizeBytes  int64                   `json:"total_size_bytes"`
	OldestCreatedAt string                  `json:"oldest_created_at,omitempty"`
	NewestCreatedAt string                  `json:"newest_created_at,omitempty"`
	ByScenario      map[string]ScenarioStat `json:"by_scenario"`
}

type ScenarioStat struct {
	Count     int   `json:"count"`
	SizeBytes int64 `json:"size_bytes"`
}

// DeleteResponse represents the response from deleting a bundle.
type DeleteResponse struct {
	OK         bool   `json:"ok"`
	FreedBytes int64  `json:"freed_bytes"`
	Message    string `json:"message,omitempty"`
	Timestamp  string `json:"timestamp"`
}

// CleanupRequest represents the request for bundle cleanup.
type CleanupRequest struct {
	ScenarioID string `json:"scenario_id,omitempty"`
	KeepLatest int    `json:"keep_latest,omitempty"`
	CleanVPS   bool   `json:"clean_vps,omitempty"`
	Host       string `json:"host,omitempty"`
	Port       int    `json:"port,omitempty"`
	User       string `json:"user,omitempty"`
	KeyPath    string `json:"key_path,omitempty"`
	Workdir    string `json:"workdir,omitempty"`
}

// CleanupResponse represents the response from bundle cleanup.
type CleanupResponse struct {
	OK              bool         `json:"ok"`
	LocalDeleted    []BundleInfo `json:"local_deleted,omitempty"`
	LocalFreedBytes int64        `json:"local_freed_bytes"`
	VPSDeleted      int          `json:"vps_deleted,omitempty"`
	VPSFreedBytes   int64        `json:"vps_freed_bytes,omitempty"`
	VPSError        string       `json:"vps_error,omitempty"`
	Message         string       `json:"message"`
	Timestamp       string       `json:"timestamp"`
}

// VPSBundleInfo represents a bundle stored on the target VPS.
type VPSBundleInfo struct {
	Filename   string `json:"filename"`
	ScenarioID string `json:"scenario_id"`
	Sha256     string `json:"sha256"`
	SizeBytes  int64  `json:"size_bytes"`
	ModTime    string `json:"mod_time"`
}

// VPSBundleListRequest is the request body for listing VPS bundles with explicit SSH config.
type VPSBundleListRequest struct {
	Host    string `json:"host"`
	Port    int    `json:"port,omitempty"`
	User    string `json:"user,omitempty"`
	KeyPath string `json:"key_path"`
	Workdir string `json:"workdir"`
}

// DeploymentVPSListResponse matches the API response for listing VPS bundles.
type DeploymentVPSListResponse struct {
	OK             bool            `json:"ok"`
	Bundles        []VPSBundleInfo `json:"bundles"`
	TotalSizeBytes int64           `json:"total_size_bytes"`
	Error          string          `json:"error,omitempty"`
	Timestamp      string          `json:"timestamp"`
}

// VPSBundleDeleteRequest deletes one bundle file from the VPS.
type VPSBundleDeleteRequest struct {
	Host     string `json:"host"`
	Port     int    `json:"port,omitempty"`
	User     string `json:"user,omitempty"`
	KeyPath  string `json:"key_path"`
	Workdir  string `json:"workdir"`
	Filename string `json:"filename"`
}

// VPSBundleDeleteResponse matches the API response for deleting a VPS bundle.
type VPSBundleDeleteResponse struct {
	OK         bool   `json:"ok"`
	FreedBytes int64  `json:"freed_bytes"`
	Message    string `json:"message"`
	Error      string `json:"error,omitempty"`
	Timestamp  string `json:"timestamp"`
}

// VPSBundleGCRequest requests garbage collection of the VPS bundle cache.
type VPSBundleGCRequest struct {
	ScenarioID    string   `json:"scenario_id,omitempty"`
	KeepLatest    int      `json:"keep_latest,omitempty"`
	ProtectSHA256 []string `json:"protect_sha256,omitempty"`
	DryRun        bool     `json:"dry_run,omitempty"`
}

// VPSBundleGCResponse matches the API response for VPS bundle GC.
type VPSBundleGCResponse struct {
	OK bool `json:"ok"`

	DryRun bool `json:"dry_run,omitempty"`

	BundlesBefore []VPSBundleInfo `json:"bundles_before,omitempty"`
	BundlesAfter  []VPSBundleInfo `json:"bundles_after,omitempty"`

	Deleted []VPSBundleInfo `json:"deleted,omitempty"`
	Kept    []VPSBundleInfo `json:"kept,omitempty"`

	DeletedCount int   `json:"deleted_count,omitempty"`
	DeletedBytes int64 `json:"deleted_bytes,omitempty"`

	TotalBeforeBytes int64 `json:"total_before_bytes,omitempty"`
	TotalAfterBytes  int64 `json:"total_after_bytes,omitempty"`

	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
	Timestamp string `json:"timestamp"`
}
