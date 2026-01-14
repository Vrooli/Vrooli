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
	SHA256      string `json:"sha256"`
	SizeBytes   int64  `json:"size_bytes"`
	ScenarioID  string `json:"scenario_id,omitempty"`
	CreatedAt   string `json:"created_at"`
	LastUsedAt  string `json:"last_used_at,omitempty"`
	UseCount    int    `json:"use_count"`
	Path        string `json:"path,omitempty"`
}

// StatsResponse represents bundle storage statistics.
type StatsResponse struct {
	TotalCount      int    `json:"total_count"`
	TotalSizeBytes  int64  `json:"total_size_bytes"`
	OldestBundle    string `json:"oldest_bundle,omitempty"`
	NewestBundle    string `json:"newest_bundle,omitempty"`
	OrphanedCount   int    `json:"orphaned_count"`
	OrphanedBytes   int64  `json:"orphaned_bytes"`
	Timestamp       string `json:"timestamp"`
}

// DeleteResponse represents the response from deleting a bundle.
type DeleteResponse struct {
	Success   bool   `json:"success"`
	SHA256    string `json:"sha256"`
	Message   string `json:"message,omitempty"`
	Timestamp string `json:"timestamp"`
}

// CleanupRequest represents the request for bundle cleanup.
type CleanupRequest struct {
	MaxAgeDays   int  `json:"max_age_days,omitempty"`   // Remove bundles older than this
	MaxCount     int  `json:"max_count,omitempty"`      // Keep only this many bundles
	OrphanedOnly bool `json:"orphaned_only,omitempty"`  // Only remove orphaned bundles
	DryRun       bool `json:"dry_run,omitempty"`        // Just report what would be cleaned
}

// CleanupResponse represents the response from bundle cleanup.
type CleanupResponse struct {
	Success        bool     `json:"success"`
	RemovedCount   int      `json:"removed_count"`
	RemovedBytes   int64    `json:"removed_bytes"`
	RemovedBundles []string `json:"removed_bundles,omitempty"` // SHA256s
	Message        string   `json:"message,omitempty"`
	DryRun         bool     `json:"dry_run"`
	Timestamp      string   `json:"timestamp"`
}

// VPSListResponse represents the response from listing bundles on VPS.
type VPSListResponse struct {
	Host       string       `json:"host"`
	Bundles    []BundleInfo `json:"bundles"`
	TotalBytes int64        `json:"total_bytes"`
	Timestamp  string       `json:"timestamp"`
}

// VPSDeleteRequest represents the request for deleting bundles from VPS.
type VPSDeleteRequest struct {
	SHA256s    []string `json:"sha256s,omitempty"`     // Specific bundles to delete
	All        bool     `json:"all,omitempty"`         // Delete all bundles
	OrphanedOnly bool   `json:"orphaned_only,omitempty"` // Only delete orphaned
}

// VPSDeleteResponse represents the response from deleting VPS bundles.
type VPSDeleteResponse struct {
	Success      bool     `json:"success"`
	RemovedCount int      `json:"removed_count"`
	RemovedBytes int64    `json:"removed_bytes"`
	Removed      []string `json:"removed,omitempty"` // SHA256s
	Message      string   `json:"message,omitempty"`
	Timestamp    string   `json:"timestamp"`
}
