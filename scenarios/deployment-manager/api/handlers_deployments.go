package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// handleDeploy initiates a deployment for a profile.
// [REQ:DM-P0-028,DM-P0-029,DM-P0-033]
func (s *Server) handleDeploy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["profile_id"]

	// [REQ:DM-P0-028] Validate profile using domain logic (extracted to domain_deployment.go)
	validation := ValidateDeploymentProfile(profileID)
	if !validation.Valid {
		status, response := formatDeploymentValidationError(validation)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(response)
		return
	}

	deploymentID := GenerateDeploymentID()

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
		"started_at":   GetTimeProvider().Now().UTC().Format(time.RFC3339),
		"completed_at": nil,
		"artifacts":    []string{},
		"message":      "Deployment status tracking not yet fully implemented",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
