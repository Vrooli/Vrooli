// Package preflight provides VPS preflight check commands for the CLI.
package preflight

// Response represents the response from preflight checks.
type Response struct {
	OK        bool     `json:"ok"`
	Checks    []Check  `json:"checks"`
	Issues    []string `json:"issues,omitempty"`
	Timestamp string   `json:"timestamp"`
}

// Check represents a single preflight check result.
type Check struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message,omitempty"`
}

// FixResponse represents a generic response from fix operations.
type FixResponse struct {
	Success   bool     `json:"success"`
	Fixed     []string `json:"fixed,omitempty"`
	Failed    []string `json:"failed,omitempty"`
	Message   string   `json:"message,omitempty"`
	Timestamp string   `json:"timestamp"`
}

// FixPortsRequest represents the request for fixing port conflicts.
type FixPortsRequest struct {
	Ports []int `json:"ports,omitempty"` // Specific ports to fix, empty = all conflicting
}

// FixFirewallRequest represents the request for fixing firewall rules.
type FixFirewallRequest struct {
	Ports    []int  `json:"ports,omitempty"`    // Specific ports to open
	Protocol string `json:"protocol,omitempty"` // tcp, udp, both
}

// FixProcessesRequest represents the request for stopping conflicting processes.
type FixProcessesRequest struct {
	PIDs  []int `json:"pids,omitempty"`  // Specific PIDs to stop
	Ports []int `json:"ports,omitempty"` // Stop processes using these ports
	Force bool  `json:"force,omitempty"` // Use SIGKILL instead of SIGTERM
}

// DiskUsageResponse represents the response from disk usage query.
type DiskUsageResponse struct {
	TotalBytes     int64           `json:"total_bytes"`
	UsedBytes      int64           `json:"used_bytes"`
	AvailableBytes int64           `json:"available_bytes"`
	UsedPercent    float64         `json:"used_percent"`
	Breakdown      []DiskBreakdown `json:"breakdown,omitempty"`
	Timestamp      string          `json:"timestamp"`
}

// DiskBreakdown represents disk usage for a specific path/category.
type DiskBreakdown struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	Category  string `json:"category,omitempty"` // bundles, logs, tmp, etc.
}

// DiskCleanupRequest represents the request for disk cleanup.
type DiskCleanupRequest struct {
	CleanBundles  bool `json:"clean_bundles,omitempty"`   // Remove old bundles
	CleanLogs     bool `json:"clean_logs,omitempty"`      // Remove old logs
	CleanTmp      bool `json:"clean_tmp,omitempty"`       // Remove temp files
	OlderThanDays int  `json:"older_than_days,omitempty"` // Only clean files older than this
	DryRun        bool `json:"dry_run,omitempty"`         // Just report, don't delete
}

// DiskCleanupResponse represents the response from disk cleanup.
type DiskCleanupResponse struct {
	Success      bool     `json:"success"`
	FreedBytes   int64    `json:"freed_bytes"`
	RemovedFiles int      `json:"removed_files"`
	Removed      []string `json:"removed,omitempty"` // List of removed items
	DryRun       bool     `json:"dry_run"`
	Message      string   `json:"message,omitempty"`
	Timestamp    string   `json:"timestamp"`
}

// RequirementsResponse represents canonical VPS requirements from the API.
type RequirementsResponse struct {
	VPS struct {
		OS struct {
			RequiredID            string   `json:"required_id"`
			RequiredFamily        string   `json:"required_family"`
			RecommendedVersion    string   `json:"recommended_version"`
			CompatibleVersions    []string `json:"compatible_versions"`
			UnsupportedBehavior   string   `json:"unsupported_behavior"`
			CompatibilityBehavior string   `json:"compatibility_behavior"`
		} `json:"os"`
		Resources struct {
			MinDiskFreeKB       int64 `json:"min_disk_free_kb"`
			MinDiskFreeBytes    int64 `json:"min_disk_free_bytes"`
			MinRAMKB            int64 `json:"min_ram_kb"`
			MinRAMBytes         int64 `json:"min_ram_bytes"`
			RecommendedRAMKB    int64 `json:"recommended_ram_kb"`
			RecommendedRAMBytes int64 `json:"recommended_ram_bytes"`
		} `json:"resources"`
		Network struct {
			RequiredInboundPorts []int `json:"required_inbound_ports"`
			SSHPort              int   `json:"ssh_port"`
		} `json:"network"`
		Authentication struct {
			RequiredMethod string `json:"required_method"`
			BootstrapFlow  string `json:"bootstrap_flow"`
		} `json:"authentication"`
	} `json:"vps"`
}
