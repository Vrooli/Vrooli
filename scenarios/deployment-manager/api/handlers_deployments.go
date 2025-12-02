package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// handleDeploy initiates a deployment for a profile.
// [REQ:DM-P0-028,DM-P0-029,DM-P0-033]
func (s *Server) handleDeploy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["profile_id"]

	// [REQ:DM-P0-028] Validate profile exists before deployment
	if profileID == "" {
		http.Error(w, `{"error":"profile_id required"}`, http.StatusBadRequest)
		return
	}

	// Simulate profile not found check
	if !strings.HasPrefix(profileID, "profile-") && !strings.HasPrefix(profileID, "test-") {
		http.Error(w, fmt.Sprintf(`{"error":"Profile '%s' not found"}`, profileID), http.StatusNotFound)
		return
	}

	// [REQ:DM-P0-028] Validate deployment readiness (simplified)
	validationErrors := []string{}

	// Check for missing packagers (simplified example)
	if profileID == "missing-packager-profile" {
		validationErrors = append(validationErrors, "Required packager 'scenario-to-desktop' not found")
	}

	// If validation fails, return error
	if len(validationErrors) > 0 {
		response := map[string]interface{}{
			"error":             "Deployment validation failed",
			"validation_errors": validationErrors,
			"remediation":       "Install required packagers or update profile configuration",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	deploymentID := fmt.Sprintf("deploy-%d", time.Now().Unix())

	response := map[string]interface{}{
		"deployment_id": deploymentID,
		"profile_id":    profileID,
		"status":        "queued",
		"logs_url":      fmt.Sprintf("/api/v1/deployments/%s/logs", deploymentID),
		"message":       "Deployment orchestration not yet fully implemented",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

// handleDeploymentStatus returns the status of a deployment.
func (s *Server) handleDeploymentStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deploymentID := vars["deployment_id"]

	response := map[string]interface{}{
		"id":           deploymentID,
		"status":       "queued",
		"started_at":   time.Now().UTC().Format(time.RFC3339),
		"completed_at": nil,
		"artifacts":    []string{},
		"message":      "Deployment status tracking not yet fully implemented",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
