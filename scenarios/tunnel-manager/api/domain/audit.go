package domain

// PortAuditResult describes the compliance status of a single route.
type PortAuditResult struct {
	Subdomain    string `json:"subdomain"`
	ScenarioName string `json:"scenario_name"`
	ExpectedPort int    `json:"expected_port"`
	ActualPort   int    `json:"actual_port,omitempty"`
	Status       string `json:"status"` // "compliant", "mismatch", "missing_scenario", "missing_port"
	Detail       string `json:"detail,omitempty"`
}
