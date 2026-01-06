// Package domain defines the core domain types for the scenario-to-cloud scenario.
package domain

// PreflightCheckStatus represents the outcome of a preflight check.
type PreflightCheckStatus string

const (
	PreflightPass PreflightCheckStatus = "pass"
	PreflightWarn PreflightCheckStatus = "warn"
	PreflightFail PreflightCheckStatus = "fail"
)

const (
	PreflightDNSVPSHostID  = "dns_vps_host"
	PreflightDNSEdgeApexID = "dns_edge_apex"
	PreflightDNSEdgeWWWID  = "dns_edge_www"
	PreflightDNSDoOriginID = "dns_do_origin"
	PreflightDNSOGWorkerID = "dns_og_worker_ready"
	PreflightDNSEdgeIPv6ID = "dns_edge_ipv6"
	PreflightFirewallID    = "firewall_inbound"
)

// PreflightCheck represents a single preflight validation result.
type PreflightCheck struct {
	ID      string               `json:"id"`
	Title   string               `json:"title"`
	Status  PreflightCheckStatus `json:"status"`
	Details string               `json:"details,omitempty"`
	Hint    string               `json:"hint,omitempty"`
	Data    map[string]string    `json:"data,omitempty"`
}

// PreflightResponse contains the results of all preflight checks.
type PreflightResponse struct {
	OK        bool              `json:"ok"`
	Checks    []PreflightCheck  `json:"checks"`
	Issues    []ValidationIssue `json:"issues,omitempty"`
	Timestamp string            `json:"timestamp"`
}
