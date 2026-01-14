// Package inspect provides VPS inspection commands for the CLI.
package inspect

// PlanResponse represents the response from inspection planning.
type PlanResponse struct {
	Plan      Plan   `json:"plan"`
	Timestamp string `json:"timestamp"`
}

// Plan contains the inspection plan details.
type Plan struct {
	Commands []Command `json:"commands"`
}

// ApplyResponse represents the response from inspection execution.
type ApplyResponse struct {
	Result    Result `json:"result"`
	Timestamp string `json:"timestamp"`
}

// Result represents the inspection result.
type Result struct {
	OK    bool   `json:"ok"`
	Steps []Step `json:"steps"`
}

// Command represents a single command in the plan.
type Command struct {
	ID          string `json:"id"`
	Description string `json:"description,omitempty"`
}

// Step represents a single execution step result.
type Step struct {
	ID      string `json:"id"`
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Output  string `json:"output,omitempty"`
}

// Options contains inspection options.
type Options struct {
	TailLines int `json:"tail_lines"`
}

// LiveStateResponse represents the response from live state inspection.
type LiveStateResponse struct {
	DeploymentID string           `json:"deployment_id"`
	State        LiveState        `json:"state"`
	Processes    []ProcessInfo    `json:"processes,omitempty"`
	Resources    []ResourceStatus `json:"resources,omitempty"`
	Timestamp    string           `json:"timestamp"`
}

// LiveState contains the current live state of the deployment.
type LiveState struct {
	Running        bool   `json:"running"`
	Healthy        bool   `json:"healthy"`
	Uptime         string `json:"uptime,omitempty"`
	CPUPercent     string `json:"cpu_percent,omitempty"`
	MemoryPercent  string `json:"memory_percent,omitempty"`
	DiskUsedGB     string `json:"disk_used_gb,omitempty"`
	LastHealthAt   string `json:"last_health_at,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
	PublicIP       string `json:"public_ip,omitempty"`
	InternalIP     string `json:"internal_ip,omitempty"`
	SSHFingerprint string `json:"ssh_fingerprint,omitempty"`
}

// ProcessInfo represents a running process.
type ProcessInfo struct {
	PID        int    `json:"pid"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	CPUPercent string `json:"cpu_percent,omitempty"`
	MemoryMB   string `json:"memory_mb,omitempty"`
	Command    string `json:"command,omitempty"`
	User       string `json:"user,omitempty"`
	StartTime  string `json:"start_time,omitempty"`
}

// ResourceStatus represents a deployed resource status.
type ResourceStatus struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Port    int    `json:"port,omitempty"`
	Healthy bool   `json:"healthy"`
	Message string `json:"message,omitempty"`
}

// DriftResponse represents the response from drift detection.
type DriftResponse struct {
	DeploymentID string      `json:"deployment_id"`
	HasDrift     bool        `json:"has_drift"`
	DriftItems   []DriftItem `json:"drift_items,omitempty"`
	CheckedAt    string      `json:"checked_at"`
	Timestamp    string      `json:"timestamp"`
}

// DriftItem represents a single configuration drift.
type DriftItem struct {
	Path     string `json:"path"`
	Type     string `json:"type"` // added, removed, modified
	Expected string `json:"expected,omitempty"`
	Actual   string `json:"actual,omitempty"`
	Severity string `json:"severity,omitempty"` // info, warning, error
	Message  string `json:"message,omitempty"`
}

// LogsResponse represents the response from logs retrieval.
type LogsResponse struct {
	DeploymentID string     `json:"deployment_id"`
	Logs         []LogEntry `json:"logs"`
	TotalCount   int        `json:"total_count"`
	HasMore      bool       `json:"has_more"`
	Timestamp    string     `json:"timestamp"`
}

// LogEntry represents a single log entry.
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Source    string `json:"source"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	PID       int    `json:"pid,omitempty"`
}

// LogsOptions contains options for fetching logs.
type LogsOptions struct {
	Source string // Filter by source (scenario name, resource name, etc.)
	Level  string // Filter by level (debug, info, warn, error)
	Search string // Search string filter
	Tail   int    // Number of lines to fetch (default 100)
	Since  string // Fetch logs since (timestamp or duration like "1h")
}

// FilesResponse represents the response from files listing.
type FilesResponse struct {
	DeploymentID string     `json:"deployment_id"`
	Path         string     `json:"path"`
	Files        []FileInfo `json:"files,omitempty"`
	Content      string     `json:"content,omitempty"`
	IsDirectory  bool       `json:"is_directory"`
	Timestamp    string     `json:"timestamp"`
}

// FileInfo represents a file or directory entry.
type FileInfo struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	IsDirectory bool   `json:"is_directory"`
	Size        int64  `json:"size"`
	ModTime     string `json:"mod_time"`
	Mode        string `json:"mode"`
}

// FilesOptions contains options for file operations.
type FilesOptions struct {
	Path    string // Path within deployment (default: /)
	Content bool   // Read file content instead of listing
}
