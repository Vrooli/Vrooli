// Package process provides process control commands for the CLI.
package process

// KillRequest is the request for killing a process.
type KillRequest struct {
	PID    int    `json:"pid"`
	Signal string `json:"signal,omitempty"` // SIGTERM, SIGKILL, etc.
}

// KillResponse is the response from killing a process.
type KillResponse struct {
	Success   bool   `json:"success"`
	PID       int    `json:"pid"`
	Signal    string `json:"signal"`
	Message   string `json:"message,omitempty"`
	Timestamp string `json:"timestamp"`
}

// RestartRequest is the request for restarting a process/service.
type RestartRequest struct {
	Type string `json:"type"` // scenario, resource
	Name string `json:"name"` // Name of the scenario or resource
}

// RestartResponse is the response from restarting a process.
type RestartResponse struct {
	Success   bool   `json:"success"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	NewPID    int    `json:"new_pid,omitempty"`
	Message   string `json:"message,omitempty"`
	Timestamp string `json:"timestamp"`
}

// ControlRequest is the request for process control actions.
type ControlRequest struct {
	Action string `json:"action"` // start, stop, restart
	Type   string `json:"type"`   // scenario, resource, all
	Name   string `json:"name,omitempty"`
}

// ControlResponse is the response from process control.
type ControlResponse struct {
	Success   bool            `json:"success"`
	Action    string          `json:"action"`
	Results   []ProcessResult `json:"results,omitempty"`
	Message   string          `json:"message,omitempty"`
	Timestamp string          `json:"timestamp"`
}

// ProcessResult represents the result of an action on a single process.
type ProcessResult struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Success bool   `json:"success"`
	PID     int    `json:"pid,omitempty"`
	Error   string `json:"error,omitempty"`
}

// VPSActionRequest is the request for VPS-level actions.
type VPSActionRequest struct {
	Action string `json:"action"` // reboot, shutdown, start
}

// VPSActionResponse is the response from VPS actions.
type VPSActionResponse struct {
	Success   bool   `json:"success"`
	Action    string `json:"action"`
	Message   string `json:"message,omitempty"`
	Timestamp string `json:"timestamp"`
}
