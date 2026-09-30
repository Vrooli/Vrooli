// Package domain defines the core domain types for the scenario-to-cloud scenario.
package domain

// BundleArtifact represents a completed bundle build result.
type BundleArtifact struct {
	Path      string `json:"path"`
	Sha256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

// BundleInfo represents metadata about a stored bundle.
type BundleInfo struct {
	Path       string `json:"path"`
	Filename   string `json:"filename"`
	ScenarioID string `json:"scenario_id"`
	Sha256     string `json:"sha256"`
	SizeBytes  int64  `json:"size_bytes"`
	CreatedAt  string `json:"created_at"`
}

// ScenarioStats holds per-scenario bundle statistics.
type ScenarioStats struct {
	Count     int   `json:"count"`
	SizeBytes int64 `json:"size_bytes"`
}

// BundleStats holds aggregate bundle storage statistics.
type BundleStats struct {
	TotalCount      int                      `json:"total_count"`
	TotalSizeBytes  int64                    `json:"total_size_bytes"`
	OldestCreatedAt string                   `json:"oldest_created_at,omitempty"`
	NewestCreatedAt string                   `json:"newest_created_at,omitempty"`
	ByScenario      map[string]ScenarioStats `json:"by_scenario"`
}

// BundleCleanupRequest is the request body for bundle cleanup operations.
type BundleCleanupRequest struct {
	// Local cleanup options
	ScenarioID string `json:"scenario_id,omitempty"` // If set, only clean this scenario's bundles
	KeepLatest int    `json:"keep_latest"`           // Keep N most recent per scenario (default: 3)

	// CleanVPS is refused: target release retention belongs to the
	// deployment-scoped GC (POST /deployments/{id}/bundles/vps/gc).
	CleanVPS bool `json:"clean_vps,omitempty"`
}

// BundleCleanupResponse is the response from bundle cleanup operations.
type BundleCleanupResponse struct {
	OK              bool         `json:"ok"`
	LocalDeleted    []BundleInfo `json:"local_deleted,omitempty"`
	LocalFreedBytes int64        `json:"local_freed_bytes"`
	Message         string       `json:"message"`
	Timestamp       string       `json:"timestamp"`
}

// BundleStatsResponse is the response from bundle stats endpoint.
type BundleStatsResponse struct {
	Stats     BundleStats `json:"stats"`
	Timestamp string      `json:"timestamp"`
}

// BundleDeleteResponse is the response from deleting a single bundle.
type BundleDeleteResponse struct {
	OK         bool   `json:"ok"`
	FreedBytes int64  `json:"freed_bytes"`
	Message    string `json:"message"`
	Timestamp  string `json:"timestamp"`
}

// VPSBundleInfo is one release in the target owner's release store.
// Filename carries the release digest (the owner's identity); Sha256 the
// bundle digest artifact leases protect; Role and State are the owner's
// standing for the release (active, previous, staged; complete, staging).
type VPSBundleInfo struct {
	Filename   string `json:"filename"`
	ScenarioID string `json:"scenario_id"`
	Sha256     string `json:"sha256"`
	SizeBytes  int64  `json:"size_bytes"`
	ModTime    string `json:"mod_time"`
	Role       string `json:"role,omitempty"`
	State      string `json:"state,omitempty"`
}

// VPSBundleListResponse is the response from listing VPS bundles.
type VPSBundleListResponse struct {
	OK             bool            `json:"ok"`
	Bundles        []VPSBundleInfo `json:"bundles"`
	TotalSizeBytes int64           `json:"total_size_bytes"`
	Error          string          `json:"error,omitempty"`
	Timestamp      string          `json:"timestamp"`
}

// VPSBundleGCRequest is the request body for garbage-collecting bundle cache on a VPS.
//
// This is intentionally conservative: it is designed to keep deployments recoverable
// (keep current + previous known-good bundles) while bounding disk growth from repeated redeploys.
type VPSBundleGCRequest struct {
	// ScenarioID limits GC to one scenario's bundles. If empty, GC may operate on all scenarios.
	ScenarioID string `json:"scenario_id,omitempty"`

	// KeepLatest keeps N newest bundles per scenario (default applied server-side).
	KeepLatest int `json:"keep_latest,omitempty"`

	// ProtectSHA256 are additional bundle hashes that must not be deleted even if older than KeepLatest.
	ProtectSHA256 []string `json:"protect_sha256,omitempty"`

	// DryRun returns the plan without deleting.
	DryRun bool `json:"dry_run,omitempty"`
}

// VPSBundleGCResponse is the response from garbage-collecting VPS bundles.
type VPSBundleGCResponse struct {
	OK bool `json:"ok"`

	DryRun bool `json:"dry_run,omitempty"`

	// BundlesBefore/BundlesAfter are the observed bundle lists (after may equal before in dry-run).
	BundlesBefore []VPSBundleInfo `json:"bundles_before,omitempty"`
	BundlesAfter  []VPSBundleInfo `json:"bundles_after,omitempty"`

	// Deleted/Kept are the computed plan decisions (subset of BundlesBefore).
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
