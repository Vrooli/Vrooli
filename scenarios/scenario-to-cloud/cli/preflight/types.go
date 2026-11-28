// Package preflight provides VPS preflight check commands for the CLI.
package preflight

// Response represents the response from preflight checks. The field names
// mirror the API's domain.PreflightResponse exactly; an earlier shape here
// used `name`/`passed`, which no producer ever sent, so every check decoded
// as an unnamed failure and the CLI could not tell pass from warn from fail.
type Response struct {
	OK        bool              `json:"ok"`
	Checks    []Check           `json:"checks"`
	Issues    []ValidationIssue `json:"issues,omitempty"`
	Timestamp string            `json:"timestamp"`
}

// CheckStatus is a preflight check's outcome.
type CheckStatus string

const (
	// CheckPass means the condition holds.
	CheckPass CheckStatus = "pass"
	// CheckWarn means the condition is imperfect but does not block a
	// deployment. Warnings are printed and never change the exit code.
	CheckWarn CheckStatus = "warn"
	// CheckFail means the condition blocks a deployment.
	CheckFail CheckStatus = "fail"
)

// Check represents a single preflight check result.
type Check struct {
	ID      string            `json:"id"`
	Title   string            `json:"title"`
	Status  CheckStatus       `json:"status"`
	Details string            `json:"details,omitempty"`
	Hint    string            `json:"hint,omitempty"`
	Data    map[string]string `json:"data,omitempty"`
}

// ValidationIssue is one manifest problem reported alongside the checks.
type ValidationIssue struct {
	Path     string `json:"path"`
	Message  string `json:"message"`
	Hint     string `json:"hint,omitempty"`
	Severity string `json:"severity"`
}

// FixResponse represents a generic response from fix operations.
type FixResponse struct {
	OK        bool     `json:"ok"`
	Stopped   []string `json:"stopped,omitempty"`
	Failed    []string `json:"failed,omitempty"`
	Message   string   `json:"message,omitempty"`
	Timestamp string   `json:"timestamp"`
}

// FixFirewallRequest represents the request for fixing firewall rules.
type FixFirewallRequest struct {
	Host  string `json:"host"`
	Port  int    `json:"port,omitempty"`
	User  string `json:"user,omitempty"`
	Ports []int  `json:"ports,omitempty"` // Specific ports to open
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
