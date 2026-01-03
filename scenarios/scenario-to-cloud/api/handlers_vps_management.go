package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// VPSActionRequest is the request body for VPS management actions.
type VPSActionRequest struct {
	Action       string `json:"action"`        // "reboot", "stop_vrooli", "cleanup"
	CleanupLevel int    `json:"cleanup_level"` // 1-4 for cleanup action
	Confirmation string `json:"confirmation"`  // typed confirmation text
}

// VPSActionResponse is the response for VPS management actions.
type VPSActionResponse struct {
	OK        bool   `json:"ok"`
	Action    string `json:"action"`
	Message   string `json:"message"`
	Output    string `json:"output,omitempty"`
	Timestamp string `json:"timestamp"`
}

// handleVPSAction handles destructive VPS management operations.
// POST /api/v1/deployments/{id}/actions/vps
func (s *Server) handleVPSAction(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	req, err := decodeJSON[VPSActionRequest](r.Body, 1<<20)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	// Validate action
	validActions := map[string]bool{"reboot": true, "stop_vrooli": true, "cleanup": true}
	if !validActions[req.Action] {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_action",
			Message: "Action must be 'reboot', 'stop_vrooli', or 'cleanup'",
		})
		return
	}

	// Validate cleanup level if action is cleanup
	if req.Action == "cleanup" && (req.CleanupLevel < 1 || req.CleanupLevel > 5) {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_cleanup_level",
			Message: "Cleanup level must be between 1 and 5",
		})
		return
	}

	// Get deployment
	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "get_failed",
			Message: "Failed to get deployment",
			Hint:    err.Error(),
		})
		return
	}

	if deployment == nil {
		writeAPIError(w, http.StatusNotFound, APIError{
			Code:    "not_found",
			Message: "Deployment not found",
		})
		return
	}

	// Parse manifest
	var manifest CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &manifest); err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "manifest_parse_failed",
			Message: "Failed to parse deployment manifest",
			Hint:    err.Error(),
		})
		return
	}

	normalized, _ := ValidateAndNormalizeManifest(manifest)
	if normalized.Target.VPS == nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "no_vps_target",
			Message: "Deployment does not have a VPS target",
		})
		return
	}

	// Validate confirmation
	if err := validateVPSConfirmation(req.Action, req.CleanupLevel, req.Confirmation, deployment.Name); err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_confirmation",
			Message: err.Error(),
		})
		return
	}

	cfg := sshConfigFromManifest(normalized)
	workdir := normalized.Target.VPS.Workdir

	// Set appropriate timeout based on action
	timeout := 2 * time.Minute
	if req.Action == "cleanup" && req.CleanupLevel >= 3 {
		timeout = 5 * time.Minute
	}

	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	// Build command based on action
	var cmd string
	var actionDesc string

	switch req.Action {
	case "reboot":
		cmd = "sudo reboot"
		actionDesc = "VPS reboot initiated"
	case "stop_vrooli":
		cmd = buildStopAllCommand(workdir, normalized)
		actionDesc = "All Vrooli processes stopped"
	case "cleanup":
		cmd, actionDesc = buildCleanupCommand(workdir, normalized, req.CleanupLevel)
	}

	sshRunner := ExecSSHRunner{}
	result, err := sshRunner.Run(ctx, cfg, cmd)

	response := VPSActionResponse{
		Action:    req.Action,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// For reboot, we expect the connection to drop, so an error is expected
	if req.Action == "reboot" && err != nil {
		// Connection drop during reboot is expected
		response.OK = true
		response.Message = actionDesc
		response.Output = "Server is rebooting. Connection was closed as expected."
		writeJSON(w, http.StatusOK, response)
		return
	}

	if err != nil {
		response.OK = false
		response.Message = "Failed to execute " + req.Action
		response.Output = result.Stderr
		writeJSON(w, http.StatusInternalServerError, response)
		return
	}

	response.OK = true
	response.Message = actionDesc
	response.Output = result.Stdout
	writeJSON(w, http.StatusOK, response)
}

// validateVPSConfirmation validates the typed confirmation for VPS actions.
func validateVPSConfirmation(action string, level int, confirmation, deploymentName string) error {
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

// buildStopAllCommand builds a command to stop all Vrooli scenarios and resources.
func buildStopAllCommand(workdir string, manifest CloudManifest) string {
	var commands []string

	// Stop target scenario
	commands = append(commands, vrooliCommand(workdir,
		fmt.Sprintf("vrooli scenario stop %s 2>/dev/null || true", shellQuoteSingle(manifest.Scenario.ID))))

	// Stop dependent scenarios
	for _, scenarioID := range manifest.Dependencies.Scenarios {
		if scenarioID == manifest.Scenario.ID {
			continue
		}
		commands = append(commands, vrooliCommand(workdir,
			fmt.Sprintf("vrooli scenario stop %s 2>/dev/null || true", shellQuoteSingle(scenarioID))))
	}

	// Stop resources
	for _, resourceID := range manifest.Dependencies.Resources {
		commands = append(commands, vrooliCommand(workdir,
			fmt.Sprintf("vrooli resource stop %s 2>/dev/null || true", shellQuoteSingle(resourceID))))
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

// buildDockerPruneCommand builds commands to stop and prune all Docker resources.
// This is useful for resetting databases and other stateful containers.
func buildDockerPruneCommand() string {
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

// buildCleanupCommand builds a command for the specified cleanup level.
func buildCleanupCommand(workdir string, manifest CloudManifest, level int) (string, string) {
	switch level {
	case 1:
		// Level 1: Remove builds only
		cmd := fmt.Sprintf("rm -rf %s/scenarios/*/builds 2>/dev/null && echo 'Builds removed'", shellQuoteSingle(workdir))
		return cmd, "Scenario builds removed"

	case 2:
		// Level 2: Stop all + remove builds
		stopCmd := buildStopAllCommand(workdir, manifest)
		cleanCmd := fmt.Sprintf("rm -rf %s/scenarios/*/builds 2>/dev/null", shellQuoteSingle(workdir))
		return stopCmd + " && " + cleanCmd + " && echo 'Stopped processes and removed builds'", "All Vrooli processes stopped and builds removed"

	case 3:
		// Level 3: Docker prune - useful for resetting databases without removing Vrooli
		stopCmd := buildStopAllCommand(workdir, manifest)
		dockerPrune := buildDockerPruneCommand()
		return stopCmd + " && " + dockerPrune, "All Docker containers, images, and volumes removed"

	case 4:
		// Level 4: Remove entire Vrooli installation
		stopCmd := buildStopAllCommand(workdir, manifest)
		removeCmd := fmt.Sprintf("rm -rf %s && echo 'Vrooli installation removed'", shellQuoteSingle(workdir))
		return stopCmd + " && " + removeCmd, "Vrooli installation removed"

	case 5:
		// Level 5: Full reset - remove Vrooli + Docker prune + cleanup system packages
		stopCmd := buildStopAllCommand(workdir, manifest)
		removeCmd := fmt.Sprintf("rm -rf %s", shellQuoteSingle(workdir))
		dockerPrune := buildDockerPruneCommand()
		aptClean := "apt-get autoremove -y 2>/dev/null || true"
		journalClean := "journalctl --vacuum-time=1d 2>/dev/null || true"
		return stopCmd + " && " + removeCmd + " && " + dockerPrune + " && " + aptClean + " && " + journalClean + " && echo 'Full reset completed'", "Full VPS reset completed (Vrooli removed, Docker pruned, system cleaned)"
	}

	return "echo 'Invalid cleanup level'", "Invalid cleanup level"
}
