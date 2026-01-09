// Package domain defines the core domain types for the scenario-to-cloud scenario.
package domain

// VPSPlanStep represents a single step in a VPS setup or deploy plan.
type VPSPlanStep struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Command     string `json:"command,omitempty"`
}

// VPSSetupResult represents the outcome of VPS setup operations.
type VPSSetupResult struct {
	OK         bool          `json:"ok"`
	Steps      []VPSPlanStep `json:"steps"`
	Error      string        `json:"error,omitempty"`
	FailedStep string        `json:"failed_step,omitempty"`
	Timestamp  string        `json:"timestamp"`
}

// VPSDeployResult represents the outcome of VPS deployment operations.
type VPSDeployResult struct {
	OK         bool          `json:"ok"`
	Steps      []VPSPlanStep `json:"steps"`
	Error      string        `json:"error,omitempty"`
	FailedStep string        `json:"failed_step,omitempty"`
	Timestamp  string        `json:"timestamp"`
}

// VPSInspectResult represents the outcome of inspecting a deployed VPS.
type VPSInspectResult struct {
	OK             bool   `json:"ok"`
	ScenarioStatus string `json:"scenario_status,omitempty"`
	Logs           string `json:"logs,omitempty"`
	Error          string `json:"error,omitempty"`
	Timestamp      string `json:"timestamp"`
}

// MissingSecretInfo describes a missing user_prompt secret.
type MissingSecretInfo struct {
	ID          string `json:"id"`
	Key         string `json:"key"`         // env var name
	Label       string `json:"label"`       // human-readable label
	Description string `json:"description"` // help text
}

// VPSStopResult represents the outcome of stopping a scenario on VPS.
type VPSStopResult struct {
	OK                bool   `json:"ok"`
	Error             string `json:"error,omitempty"`
	ScenarioStopped   bool   `json:"scenario_stopped"`
	ProcessesKilled   int    `json:"processes_killed"`
	PortsCleared      []int  `json:"ports_cleared,omitempty"`
	ProcessesRemained int    `json:"processes_remained"`
	Timestamp         string `json:"timestamp"`
}
