package vps

import (
	"fmt"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/ssh"
)

// ActionRequest is the request body for VPS management actions.
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

// BuildStopAllCommand builds a command to stop all Vrooli scenarios and resources.
func BuildStopAllCommand(workdir string, manifest domain.CloudManifest) string {
	var commands []string

	// Stop target scenario
	commands = append(commands, ssh.VrooliCommand(workdir,
		fmt.Sprintf("vrooli scenario stop %s 2>/dev/null || true", ssh.QuoteSingle(manifest.Scenario.ID))))

	// Stop dependent scenarios
	for _, scenarioID := range manifest.Dependencies.Scenarios {
		if scenarioID == manifest.Scenario.ID {
			continue
		}
		commands = append(commands, ssh.VrooliCommand(workdir,
			fmt.Sprintf("vrooli scenario stop %s 2>/dev/null || true", ssh.QuoteSingle(scenarioID))))
	}

	// Stop resources
	for _, resourceID := range manifest.Dependencies.Resources {
		commands = append(commands, ssh.VrooliCommand(workdir,
			fmt.Sprintf("vrooli resource stop %s 2>/dev/null || true", ssh.QuoteSingle(resourceID))))
	}

	if len(commands) == 0 {
		return "echo 'No processes to stop'"
	}

	// Join commands with && for sequential execution
	result := commands[0]
	for i := 1; i < len(commands); i++ {
		result += " && " + commands[i]
	}
	return result
}

// BuildDockerPruneCommand builds commands to stop and prune all Docker resources.
// This is useful for resetting databases and other stateful containers.
func BuildDockerPruneCommand() string {
	// Stop all running containers, remove all containers, images, volumes, and networks
	// Using || true to continue even if some commands fail (e.g., no containers running)
	// NOTE: docker system prune --volumes only removes ANONYMOUS volumes, not named ones.
	// We explicitly remove all volumes with: docker volume rm $(docker volume ls -q)
	commands := []string{
		"docker stop $(docker ps -aq) 2>/dev/null || true",
		"docker rm $(docker ps -aq) 2>/dev/null || true",
		"docker volume rm $(docker volume ls -q) 2>/dev/null || true",
		"docker system prune -af 2>/dev/null || true",
		"echo 'Docker system pruned'",
	}
	result := commands[0]
	for i := 1; i < len(commands); i++ {
		result += " && " + commands[i]
	}
	return result
}

// BuildCleanupCommand builds a command for the specified cleanup level.
// Returns the command string and a description of the action.
func BuildCleanupCommand(workdir string, manifest domain.CloudManifest, level int) (string, string) {
	switch level {
	case 1:
		// Level 1: Remove builds only
		cmd := fmt.Sprintf("rm -rf %s/scenarios/*/builds 2>/dev/null && echo 'Builds removed'", ssh.QuoteSingle(workdir))
		return cmd, "Scenario builds removed"

	case 2:
		// Level 2: Stop all + remove builds
		stopCmd := BuildStopAllCommand(workdir, manifest)
		cleanCmd := fmt.Sprintf("rm -rf %s/scenarios/*/builds 2>/dev/null", ssh.QuoteSingle(workdir))
		return stopCmd + " && " + cleanCmd + " && echo 'Stopped processes and removed builds'", "All Vrooli processes stopped and builds removed"

	case 3:
		// Level 3: Docker prune - useful for resetting databases without removing Vrooli
		stopCmd := BuildStopAllCommand(workdir, manifest)
		dockerPrune := BuildDockerPruneCommand()
		return stopCmd + " && " + dockerPrune, "All Docker containers, images, and volumes removed"

	case 4:
		// Level 4: Remove entire Vrooli installation
		stopCmd := BuildStopAllCommand(workdir, manifest)
		removeCmd := fmt.Sprintf("rm -rf %s && echo 'Vrooli installation removed'", ssh.QuoteSingle(workdir))
		return stopCmd + " && " + removeCmd, "Vrooli installation removed"

	case 5:
		// Level 5: Full reset - remove Vrooli + Docker prune + cleanup system packages
		stopCmd := BuildStopAllCommand(workdir, manifest)
		removeCmd := fmt.Sprintf("rm -rf %s", ssh.QuoteSingle(workdir))
		dockerPrune := BuildDockerPruneCommand()
		aptClean := "apt-get autoremove -y 2>/dev/null || true"
		journalClean := "journalctl --vacuum-time=1d 2>/dev/null || true"
		return stopCmd + " && " + removeCmd + " && " + dockerPrune + " && " + aptClean + " && " + journalClean + " && echo 'Full reset completed'", "Full VPS reset completed (Vrooli removed, Docker pruned, system cleaned)"
	}

	return "echo 'Invalid cleanup level'", "Invalid cleanup level"
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
