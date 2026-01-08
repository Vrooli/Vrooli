package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/manifest"
	"scenario-to-cloud/ssh"
	"scenario-to-cloud/vps"
)

// Type aliases for backward compatibility.
type (
	VPSActionRequest  = vps.ActionRequest
	VPSActionResponse = vps.ActionResponse
)

// handleVPSAction handles destructive VPS management operations.
// POST /api/v1/deployments/{id}/actions/vps
func (s *Server) handleVPSAction(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	req, err := httputil.DecodeJSON[VPSActionRequest](r.Body, 1<<20)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	// Validate action
	validActions := map[string]bool{"reboot": true, "stop_vrooli": true, "cleanup": true}
	if !validActions[req.Action] {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_action",
			Message: "Action must be 'reboot', 'stop_vrooli', or 'cleanup'",
		})
		return
	}

	// Validate cleanup level if action is cleanup
	if req.Action == "cleanup" && (req.CleanupLevel < 1 || req.CleanupLevel > 5) {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_cleanup_level",
			Message: "Cleanup level must be between 1 and 5",
		})
		return
	}

	// Get deployment
	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "get_failed",
			Message: "Failed to get deployment",
			Hint:    err.Error(),
		})
		return
	}

	if deployment == nil {
		httputil.WriteAPIError(w, http.StatusNotFound, httputil.APIError{
			Code:    "not_found",
			Message: "Deployment not found",
		})
		return
	}

	// Parse manifest
	var m domain.CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &m); err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "manifest_parse_failed",
			Message: "Failed to parse deployment manifest",
			Hint:    err.Error(),
		})
		return
	}

	normalized, _ := manifest.ValidateAndNormalize(m)
	if normalized.Target.VPS == nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "no_vps_target",
			Message: "Deployment does not have a VPS target",
		})
		return
	}

	// Validate confirmation
	if err := vps.ValidateActionConfirmation(req.Action, req.CleanupLevel, req.Confirmation, deployment.Name); err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_confirmation",
			Message: err.Error(),
		})
		return
	}

	cfg := ssh.ConfigFromManifest(normalized)
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
		cmd = vps.BuildStopAllCommand(workdir, normalized)
		actionDesc = "All Vrooli processes stopped"
	case "cleanup":
		cmd, actionDesc = vps.BuildCleanupCommand(workdir, normalized, req.CleanupLevel)
	}

	result, err := s.sshRunner.Run(ctx, cfg, cmd)

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
		httputil.WriteJSON(w, http.StatusOK, response)
		return
	}

	if err != nil {
		response.OK = false
		response.Message = "Failed to execute " + req.Action
		response.Output = result.Stderr
		httputil.WriteJSON(w, http.StatusInternalServerError, response)
		return
	}

	response.OK = true
	response.Message = actionDesc
	response.Output = result.Stdout
	httputil.WriteJSON(w, http.StatusOK, response)
}
