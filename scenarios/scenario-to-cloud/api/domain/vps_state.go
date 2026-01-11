// Package domain defines the core domain types for the scenario-to-cloud scenario.
package domain

import "encoding/json"

// LiveStateResult contains the comprehensive live state of a VPS.
type LiveStateResult struct {
	OK             bool              `json:"ok"`
	Timestamp      string            `json:"timestamp"`
	SyncDurationMs int64             `json:"sync_duration_ms"`
	Processes      *ProcessState     `json:"processes,omitempty"`
	Expected       []ExpectedProcess `json:"expected,omitempty"`
	Ports          []PortBinding     `json:"ports,omitempty"`
	Caddy          *CaddyState       `json:"caddy,omitempty"`
	System         *SystemState      `json:"system,omitempty"`
	Error          string            `json:"error,omitempty"`
}

// ExpectedProcess represents a process expected from the manifest.
type ExpectedProcess struct {
	ID              string `json:"id"`
	Type            string `json:"type"`  // "scenario" or "resource"
	State           string `json:"state"` // "running", "stopped", "needs_setup"
	DirectoryExists bool   `json:"directory_exists"`
}

// ProcessState contains all process information.
type ProcessState struct {
	Scenarios  []ScenarioProcess   `json:"scenarios"`
	Resources  []ResourceProcess   `json:"resources"`
	Unexpected []UnexpectedProcess `json:"unexpected"`
}

// ScenarioProcess represents a running scenario.
type ScenarioProcess struct {
	ID            string           `json:"id"`
	Status        string           `json:"status"` // running, stopped, failed
	PID           int              `json:"pid"`
	UptimeSeconds int64            `json:"uptime_seconds"`
	LastRestart   string           `json:"last_restart,omitempty"`
	Ports         []ProcessPort    `json:"ports,omitempty"`
	Resources     ProcessResources `json:"resources"`
	VrooliStatus  json.RawMessage  `json:"vrooli_status,omitempty"` // Raw output from vrooli scenario status
}

// ResourceProcess represents a running resource (postgres, redis, etc).
type ResourceProcess struct {
	ID            string          `json:"id"`
	Status        string          `json:"status"`
	PID           int             `json:"pid"`
	Port          int             `json:"port,omitempty"`
	UptimeSeconds int64           `json:"uptime_seconds"`
	Metrics       json.RawMessage `json:"metrics,omitempty"` // Resource-specific metrics
	VrooliStatus  json.RawMessage `json:"vrooli_status,omitempty"`
}

// UnexpectedProcess represents a process that isn't expected from the manifest.
type UnexpectedProcess struct {
	PID     int    `json:"pid"`
	Command string `json:"command"`
	Port    int    `json:"port,omitempty"`
	User    string `json:"user"`
}

// ProcessPort represents a port a process is listening on.
type ProcessPort struct {
	Name       string `json:"name"`
	Port       int    `json:"port"`
	Status     string `json:"status"`     // listening, responding, not_responding
	Responding bool   `json:"responding"` // true if health check passed
	Clients    int    `json:"clients,omitempty"`
}

// ProcessResources contains resource usage for a process.
type ProcessResources struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryMB      int     `json:"memory_mb"`
	MemoryPercent float64 `json:"memory_percent"`
}

// PortBinding represents a single port binding on the VPS.
type PortBinding struct {
	Port            int    `json:"port"`
	Process         string `json:"process"`
	Type            string `json:"type"` // system, edge, scenario, resource, unexpected
	MatchesManifest *bool  `json:"matches_manifest,omitempty"`
	PID             *int   `json:"pid,omitempty"`
	Command         string `json:"command,omitempty"`
}

// CaddyState contains Caddy/TLS information.
type CaddyState struct {
	Running bool         `json:"running"`
	Domain  string       `json:"domain"`
	TLS     TLSInfo      `json:"tls"`
	Routes  []CaddyRoute `json:"routes"`
}

// TLSInfo contains TLS certificate information.
type TLSInfo struct {
	Valid         bool       `json:"valid"`
	Validation    string     `json:"validation,omitempty"`
	Issuer        string     `json:"issuer,omitempty"`
	Expires       string     `json:"expires,omitempty"`
	DaysRemaining int        `json:"days_remaining,omitempty"`
	Error         string     `json:"error,omitempty"`
	ALPN          *ALPNCheck `json:"alpn,omitempty"`
}

// ALPNCheck captures TLS-ALPN readiness details.
type ALPNCheck struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Hint     string `json:"hint,omitempty"`
	Protocol string `json:"protocol,omitempty"`
	Error    string `json:"error,omitempty"`
}

// CaddyRoute represents a route in the Caddyfile.
type CaddyRoute struct {
	Path     string `json:"path"`
	Upstream string `json:"upstream"`
}

// SystemState contains system resource information.
type SystemState struct {
	CPU           CPUInfo    `json:"cpu"`
	Memory        MemoryInfo `json:"memory"`
	Disk          DiskInfo   `json:"disk"`
	Swap          SwapInfo   `json:"swap"`
	SSH           SSHHealth  `json:"ssh"`
	UptimeSeconds int64      `json:"uptime_seconds"`
}

// SSHHealth contains SSH connectivity status.
type SSHHealth struct {
	Connected bool   `json:"connected"`
	LatencyMs int64  `json:"latency_ms"`
	KeyInAuth bool   `json:"key_in_auth"` // Is manifest key in authorized_keys?
	KeyPath   string `json:"key_path"`    // Path to the key file used
	Error     string `json:"error,omitempty"`
}

// CPUInfo contains CPU information.
type CPUInfo struct {
	Cores        int       `json:"cores"`
	Model        string    `json:"model,omitempty"`
	UsagePercent float64   `json:"usage_percent"`
	LoadAverage  []float64 `json:"load_average"`
}

// MemoryInfo contains memory information.
type MemoryInfo struct {
	TotalMB      int     `json:"total_mb"`
	UsedMB       int     `json:"used_mb"`
	FreeMB       int     `json:"free_mb"`
	UsagePercent float64 `json:"usage_percent"`
}

// DiskInfo contains disk information.
type DiskInfo struct {
	TotalGB      int     `json:"total_gb"`
	UsedGB       int     `json:"used_gb"`
	FreeGB       int     `json:"free_gb"`
	UsagePercent float64 `json:"usage_percent"`
}

// SwapInfo contains swap information.
type SwapInfo struct {
	TotalMB      int     `json:"total_mb"`
	UsedMB       int     `json:"used_mb"`
	UsagePercent float64 `json:"usage_percent"`
}

// DriftReport contains the drift detection results comparing manifest expectations vs actual state.
type DriftReport struct {
	OK        bool         `json:"ok"`
	Timestamp string       `json:"timestamp"`
	Summary   DriftSummary `json:"summary"`
	Checks    []DriftCheck `json:"checks"`
}

// DriftSummary contains aggregate drift statistics.
type DriftSummary struct {
	Passed   int `json:"passed"`
	Warnings int `json:"warnings"`
	Drifts   int `json:"drifts"`
}

// DriftCheck represents a single drift check result.
type DriftCheck struct {
	Category string   `json:"category"` // scenarios, resources, ports, edge
	Name     string   `json:"name"`
	Status   string   `json:"status"` // pass, warning, drift
	Expected string   `json:"expected"`
	Actual   string   `json:"actual"`
	Message  string   `json:"message,omitempty"`
	Actions  []string `json:"actions,omitempty"`
}

// FileEntry represents a file or directory in the file explorer.
type FileEntry struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // file, directory, symlink
	SizeBytes   int64  `json:"size_bytes"`
	Modified    string `json:"modified"`
	Permissions string `json:"permissions"`
}
