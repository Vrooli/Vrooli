// Package vps provides VPS setup and deploy commands for the CLI.
package vps

// SetupPlanResponse represents the response from VPS setup planning.
type SetupPlanResponse struct {
	Plan      SetupPlan `json:"plan"`
	Timestamp string    `json:"timestamp"`
}

// SetupPlan contains the setup plan details.
type SetupPlan struct {
	RemoteTarPath string    `json:"remote_tar_path"`
	Commands      []Command `json:"commands"`
}

// SetupApplyResponse represents the response from VPS setup execution.
type SetupApplyResponse struct {
	Result    Result `json:"result"`
	Timestamp string `json:"timestamp"`
}

// DeployPlanResponse represents the response from VPS deploy planning.
type DeployPlanResponse struct {
	Plan      DeployPlan `json:"plan"`
	Timestamp string     `json:"timestamp"`
}

// DeployPlan contains the deploy plan details.
type DeployPlan struct {
	Commands []Command `json:"commands"`
}

// DeployApplyResponse represents the response from VPS deploy execution.
type DeployApplyResponse struct {
	Result    Result `json:"result"`
	Timestamp string `json:"timestamp"`
}

// Command represents a single command in the plan.
type Command struct {
	ID          string `json:"id"`
	Description string `json:"description,omitempty"`
}

// Result represents the execution result.
type Result struct {
	OK    bool   `json:"ok"`
	Steps []Step `json:"steps"`
}

// Step represents a single execution step result.
type Step struct {
	ID      string `json:"id"`
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
}
