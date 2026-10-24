package vps

import "fmt"

// ActionRequest is the request body for VPS management actions. Every
// action runs through the target owner's typed verbs (privilegebroker
// actions via `vrooli cloud-target host repair`, scoped scenario stops
// through the lifecycle owner); there is no shell composition on the cloud
// side.
type ActionRequest struct {
	Action       string `json:"action"`        // "reboot", "stop_vrooli", "cleanup"
	CleanupLevel int    `json:"cleanup_level"` // 1-5 for cleanup action
	Confirmation string `json:"confirmation"`  // typed confirmation text
}

// ActionResponse is the response for VPS management actions.
type ActionResponse struct {
	OK        bool   `json:"ok"`
	Action    string `json:"action"`
	Message   string `json:"message"`
	Output    string `json:"output,omitempty"`
	Timestamp string `json:"timestamp"`
}

// ValidateActionConfirmation validates the typed confirmation for VPS actions.
func ValidateActionConfirmation(action string, level int, confirmation, deploymentName string) error {
	switch action {
	case "reboot":
		if confirmation != "REBOOT" {
			return fmt.Errorf("type REBOOT to confirm VPS reboot")
		}
	case "stop_vrooli":
		if confirmation != deploymentName {
			return fmt.Errorf("type the deployment name '%s' to confirm stopping all Vrooli processes", deploymentName)
		}
	case "cleanup":
		if level == 3 {
			if confirmation != "DOCKER-RESET" {
				return fmt.Errorf("type DOCKER-RESET to confirm Docker system prune")
			}
		} else if level >= 4 {
			if confirmation != "RESET" && confirmation != "DELETE-VROOLI" {
				return fmt.Errorf("type RESET or DELETE-VROOLI to confirm level %d cleanup", level)
			}
		} else {
			if confirmation != deploymentName {
				return fmt.Errorf("type the deployment name '%s' to confirm cleanup", deploymentName)
			}
		}
	}
	return nil
}

// KillProcessRequest is the request body for killing a process.
type KillProcessRequest struct {
	PID    int    `json:"pid"`
	Signal string `json:"signal,omitempty"` // TERM, KILL, etc.
}

// RestartRequest is the request body for restarting a scenario or resource.
type RestartRequest struct {
	Type string `json:"type"` // "scenario" or "resource"
	ID   string `json:"id"`
}

// ProcessControlRequest is the request body for controlling a scenario or resource.
type ProcessControlRequest struct {
	Action string `json:"action"` // "start", "stop", "restart", "setup"
	Type   string `json:"type"`   // "scenario" or "resource"
	ID     string `json:"id"`
}

// ProcessControlResponse is the response for process control actions.
type ProcessControlResponse struct {
	OK        bool   `json:"ok"`
	Action    string `json:"action"`
	Type      string `json:"type"`
	ID        string `json:"id"`
	Message   string `json:"message"`
	Output    string `json:"output,omitempty"`
	Timestamp string `json:"timestamp"`
}
