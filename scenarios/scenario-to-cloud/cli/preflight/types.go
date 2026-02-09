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
	OK        bool     `json:"ok"`
	Stopped   []string `json:"stopped,omitempty"`
	Failed    []string `json:"failed,omitempty"`
	Message   string   `json:"message,omitempty"`
	Timestamp string   `json:"timestamp"`
}

// FixPortsRequest represents the request for fixing port conflicts.
type FixPortsRequest struct {
	Host              string   `json:"host"`
	Port              int      `json:"port,omitempty"`
	User              string   `json:"user,omitempty"`
	KeyPath           string   `json:"key_path"`
	Ports             []int    `json:"ports,omitempty"`               // Specific ports to fix, empty = all conflicting
	PIDs              []int    `json:"pids,omitempty"`                // Optional explicit PIDs to stop
	Services          []string `json:"services,omitempty"`            // Optional explicit services to stop
	PreferServiceStop *bool    `json:"prefer_service_stop,omitempty"` // Prefer stopping owning systemd service first
}

// FixFirewallRequest represents the request for fixing firewall rules.
type FixFirewallRequest struct {
	Host    string `json:"host"`
	Port    int    `json:"port,omitempty"`
	User    string `json:"user,omitempty"`
	KeyPath string `json:"key_path"`
	Ports   []int  `json:"ports,omitempty"` // Specific ports to open
}

// FixFirewallResponse represents firewall rule update results.
type FixFirewallResponse struct {
	OK        bool   `json:"ok"`
	Message   string `json:"message"`
	Ports     []int  `json:"ports"`
	Status    string `json:"status,omitempty"`
	Timestamp string `json:"timestamp"`
}

// FixProcessesRequest represents the request for stopping stale scenario processes.
type FixProcessesRequest struct {
	Host       string `json:"host"`
	Port       int    `json:"port,omitempty"`
	User       string `json:"user,omitempty"`
	KeyPath    string `json:"key_path"`
	Workdir    string `json:"workdir"`
	ScenarioID string `json:"scenario_id,omitempty"`
}

// FixProcessesResponse is returned by /preflight/fix/stop-processes.
type FixProcessesResponse struct {
	OK        bool   `json:"ok"`
	Action    string `json:"action"`
	Message   string `json:"message"`
	Output    string `json:"output,omitempty"`
	Timestamp string `json:"timestamp"`
}

// DiskUsageResponse represents the response from disk usage query.
type DiskUsageResponse struct {
	OK          bool             `json:"ok"`
	FreeSpace   string           `json:"free_space"`
	FreeBytes   int64            `json:"free_bytes"`
	TotalSpace  string           `json:"total_space"`
	TotalBytes  int64            `json:"total_bytes"`
	UsedPercent int              `json:"used_percent"`
	LargestDirs []DiskUsageEntry `json:"largest_dirs,omitempty"`
	Timestamp   string           `json:"timestamp"`
}

// DiskUsageEntry represents one directory usage row.
type DiskUsageEntry struct {
	Path  string `json:"path"`
	Size  string `json:"size"`
	Bytes int64  `json:"bytes"`
}

// DiskUsageRequest represents the request for disk usage.
type DiskUsageRequest struct {
	Host    string `json:"host"`
	Port    int    `json:"port,omitempty"`
	User    string `json:"user,omitempty"`
	KeyPath string `json:"key_path"`
}

// DiskCleanupRequest represents the request for disk cleanup.
type DiskCleanupRequest struct {
	Host    string   `json:"host"`
	Port    int      `json:"port,omitempty"`
	User    string   `json:"user,omitempty"`
	KeyPath string   `json:"key_path"`
	Actions []string `json:"actions,omitempty"` // apt_clean, journal_vacuum, docker_prune, tmp_clean
}

// DiskCleanupResponse represents the response from disk cleanup.
type DiskCleanupResponse struct {
	OK            bool                `json:"ok"`
	SpaceFreed    string              `json:"space_freed"`
	SpaceFreedKB  int64               `json:"space_freed_kb"`
	Message       string              `json:"message,omitempty"`
	ActionsRun    []string            `json:"actions_run,omitempty"`
	ActionsFailed []string            `json:"actions_failed,omitempty"`
	ActionResults []DiskCleanupAction `json:"action_results,omitempty"`
	Timestamp     string              `json:"timestamp"`
}

// DiskCleanupAction captures execution details for one cleanup action.
type DiskCleanupAction struct {
	Action   string `json:"action"`
	OK       bool   `json:"ok"`
	ExitCode int    `json:"exit_code"`
	Summary  string `json:"summary,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	Hint     string `json:"hint,omitempty"`
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
