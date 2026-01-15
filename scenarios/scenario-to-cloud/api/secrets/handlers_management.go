package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/ssh"
)

// DeploymentRepository provides access to deployment data.
type DeploymentRepository interface {
	GetDeployment(ctx context.Context, id string) (*domain.Deployment, error)
}

// ManagementDeps holds dependencies for secrets management handlers.
type ManagementDeps struct {
	Repo      DeploymentRepository
	SSHRunner ssh.Runner
}

// secretKeyRegex validates secret keys (uppercase + underscore format).
var secretKeyRegex = regexp.MustCompile(domain.SecretKeyValidationRegex)

// validateSecretKey validates that a key is in the correct format.
func validateSecretKey(key string) error {
	if key == "" {
		return fmt.Errorf("key is required")
	}
	if len(key) > domain.MaxSecretKeyLength {
		return fmt.Errorf("key exceeds maximum length of %d characters", domain.MaxSecretKeyLength)
	}
	if !secretKeyRegex.MatchString(key) {
		return fmt.Errorf("key must be uppercase letters, numbers, and underscores (e.g., MY_API_KEY)")
	}
	for _, prefix := range domain.ReservedKeyPrefixes {
		if strings.HasPrefix(key, prefix) {
			return fmt.Errorf("key prefix %q is reserved", prefix)
		}
	}
	return nil
}

// validateSecretValue validates that a value is acceptable.
func validateSecretValue(value string) error {
	if value == "" {
		return fmt.Errorf("value is required")
	}
	if len(value) > domain.MaxSecretValueLength {
		return fmt.Errorf("value exceeds maximum length of %d bytes", domain.MaxSecretValueLength)
	}
	if strings.ContainsRune(value, 0) {
		return fmt.Errorf("value contains null bytes")
	}
	return nil
}

// getDeploymentContext extracts deployment and SSH config from request.
func getDeploymentContext(
	ctx context.Context,
	repo DeploymentRepository,
	deploymentID string,
) (*domain.Deployment, domain.CloudManifest, ssh.Config, string, error) {
	dep, err := repo.GetDeployment(ctx, deploymentID)
	if err != nil {
		return nil, domain.CloudManifest{}, ssh.Config{}, "", fmt.Errorf("get deployment: %w", err)
	}
	if dep == nil {
		return nil, domain.CloudManifest{}, ssh.Config{}, "", fmt.Errorf("deployment not found")
	}

	var m domain.CloudManifest
	if err := json.Unmarshal(dep.Manifest, &m); err != nil {
		return nil, domain.CloudManifest{}, ssh.Config{}, "", fmt.Errorf("parse manifest: %w", err)
	}

	if m.Target.VPS == nil {
		return nil, domain.CloudManifest{}, ssh.Config{}, "", fmt.Errorf("deployment has no VPS target")
	}

	cfg := ssh.ConfigFromManifest(m)
	workdir := m.Target.VPS.Workdir

	return dep, m, cfg, workdir, nil
}

// restartScenarioOnVPS restarts the scenario on the VPS after a secret change.
func restartScenarioOnVPS(
	ctx context.Context,
	sshRunner ssh.Runner,
	cfg ssh.Config,
	workdir string,
	scenarioID string,
) error {
	// Stop then start the scenario
	stopCmd := ssh.VrooliCommand(workdir, fmt.Sprintf("vrooli scenario stop %s", ssh.QuoteSingle(scenarioID)))
	if _, err := sshRunner.Run(ctx, cfg, stopCmd); err != nil {
		return fmt.Errorf("stop scenario: %w", err)
	}

	// Small delay to allow resources to release
	time.Sleep(2 * time.Second)

	startCmd := ssh.VrooliCommand(workdir, fmt.Sprintf("vrooli scenario start %s", ssh.QuoteSingle(scenarioID)))
	if _, err := sshRunner.Run(ctx, cfg, startCmd); err != nil {
		return fmt.Errorf("start scenario: %w", err)
	}

	return nil
}

// HandleListVPSSecrets returns an HTTP handler that lists all secrets on a deployed VPS.
//
// GET /api/v1/deployments/{id}/secrets
//
// Response:
//
//	{
//	  "secrets": [{"key": "...", "masked": true, "source": "..."}],
//	  "metadata": {...},
//	  "timestamp": "..."
//	}
func HandleListVPSSecrets(deps ManagementDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deploymentID := mux.Vars(r)["id"]
		if deploymentID == "" {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
				Code:    "missing_deployment_id",
				Message: "Deployment ID is required",
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		_, _, cfg, workdir, err := getDeploymentContext(ctx, deps.Repo, deploymentID)
		if err != nil {
			status := http.StatusInternalServerError
			if strings.Contains(err.Error(), "not found") {
				status = http.StatusNotFound
			}
			httputil.WriteAPIError(w, status, httputil.APIError{
				Code:    "deployment_context_error",
				Message: err.Error(),
			})
			return
		}

		data, err := ReadAllFromVPS(ctx, deps.SSHRunner, cfg, workdir)
		if err != nil {
			httputil.WriteAPIError(w, http.StatusBadGateway, httputil.APIError{
				Code:    "read_secrets_failed",
				Message: "Failed to read secrets from VPS",
				Hint:    err.Error(),
			})
			return
		}

		// Convert to response format (masked by default)
		secrets := make([]domain.VPSSecretEntry, 0, len(data.Secrets))
		for key := range data.Secrets {
			secrets = append(secrets, domain.VPSSecretEntry{
				Key:    key,
				Masked: true,
				Source: "unknown", // We don't track source in secrets.json yet
			})
		}

		// Sort by key for consistent ordering
		sort.Slice(secrets, func(i, j int) bool {
			return secrets[i].Key < secrets[j].Key
		})

		response := domain.ListVPSSecretsResponse{
			Secrets: secrets,
			Metadata: domain.VPSSecretsMetadata{
				Environment: data.Metadata.Environment,
				LastUpdated: data.Metadata.LastUpdated.Format(time.RFC3339),
				ScenarioID:  data.Metadata.ScenarioID,
				GeneratedBy: data.Metadata.GeneratedBy,
				Notes:       data.Metadata.Notes,
			},
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}

		httputil.WriteJSON(w, http.StatusOK, response)
	}
}

// HandleGetVPSSecret returns an HTTP handler that gets a single secret from the VPS.
//
// GET /api/v1/deployments/{id}/secrets/{key}?reveal=true
//
// Query params:
//   - reveal: if "true", include the actual secret value (default: masked)
func HandleGetVPSSecret(deps ManagementDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		deploymentID := vars["id"]
		secretKey := vars["key"]

		if deploymentID == "" || secretKey == "" {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
				Code:    "missing_parameters",
				Message: "Deployment ID and secret key are required",
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		_, _, cfg, workdir, err := getDeploymentContext(ctx, deps.Repo, deploymentID)
		if err != nil {
			status := http.StatusInternalServerError
			if strings.Contains(err.Error(), "not found") {
				status = http.StatusNotFound
			}
			httputil.WriteAPIError(w, status, httputil.APIError{
				Code:    "deployment_context_error",
				Message: err.Error(),
			})
			return
		}

		data, err := ReadAllFromVPS(ctx, deps.SSHRunner, cfg, workdir)
		if err != nil {
			httputil.WriteAPIError(w, http.StatusBadGateway, httputil.APIError{
				Code:    "read_secrets_failed",
				Message: "Failed to read secrets from VPS",
				Hint:    err.Error(),
			})
			return
		}

		value, exists := data.Secrets[secretKey]
		if !exists {
			httputil.WriteAPIError(w, http.StatusNotFound, httputil.APIError{
				Code:    "secret_not_found",
				Message: fmt.Sprintf("Secret %q not found", secretKey),
			})
			return
		}

		reveal := r.URL.Query().Get("reveal") == "true"
		entry := domain.VPSSecretEntry{
			Key:    secretKey,
			Masked: !reveal,
			Source: "unknown",
		}
		if reveal {
			entry.Value = value
		}

		httputil.WriteJSON(w, http.StatusOK, domain.GetVPSSecretResponse{
			Secret:    entry,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// HandleCreateVPSSecret returns an HTTP handler that creates a new secret on the VPS.
//
// POST /api/v1/deployments/{id}/secrets
//
// Request body:
//
//	{
//	  "key": "MY_API_KEY",
//	  "value": "secret-value",
//	  "restart_scenario": false
//	}
func HandleCreateVPSSecret(deps ManagementDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deploymentID := mux.Vars(r)["id"]
		if deploymentID == "" {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
				Code:    "missing_deployment_id",
				Message: "Deployment ID is required",
			})
			return
		}

		var req domain.CreateSecretRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
				Code:    "invalid_json",
				Message: "Invalid request body",
				Hint:    err.Error(),
			})
			return
		}

		// Validate key
		if err := validateSecretKey(req.Key); err != nil {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
				Code:    "invalid_key",
				Message: err.Error(),
			})
			return
		}

		// Validate value
		if err := validateSecretValue(req.Value); err != nil {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
				Code:    "invalid_value",
				Message: err.Error(),
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancel()

		_, m, cfg, workdir, err := getDeploymentContext(ctx, deps.Repo, deploymentID)
		if err != nil {
			status := http.StatusInternalServerError
			if strings.Contains(err.Error(), "not found") {
				status = http.StatusNotFound
			}
			httputil.WriteAPIError(w, status, httputil.APIError{
				Code:    "deployment_context_error",
				Message: err.Error(),
			})
			return
		}

		// Add the secret
		if err := AddSecretToVPS(ctx, deps.SSHRunner, cfg, workdir, req.Key, req.Value, m.Scenario.ID); err != nil {
			status := http.StatusInternalServerError
			if strings.Contains(err.Error(), "already exists") {
				status = http.StatusConflict
			}
			httputil.WriteAPIError(w, status, httputil.APIError{
				Code:    "create_secret_failed",
				Message: "Failed to create secret",
				Hint:    err.Error(),
			})
			return
		}

		response := domain.NewSecretOperationResponse(true, req.Key, "created", "Secret created successfully")

		// Optionally restart the scenario
		if req.RestartScenario {
			if err := restartScenarioOnVPS(ctx, deps.SSHRunner, cfg, workdir, m.Scenario.ID); err != nil {
				// Secret was created but restart failed - report partial success
				response.Message = fmt.Sprintf("Secret created but scenario restart failed: %v", err)
				response.ScenarioRestart = false
			} else {
				response.ScenarioRestart = true
				response.Message = "Secret created and scenario restarted"
			}
		}

		httputil.WriteJSON(w, http.StatusCreated, response)
	}
}

// HandleUpdateVPSSecret returns an HTTP handler that updates an existing secret on the VPS.
//
// PUT /api/v1/deployments/{id}/secrets/{key}
//
// Request body:
//
//	{
//	  "value": "new-secret-value",
//	  "restart_scenario": false
//	}
func HandleUpdateVPSSecret(deps ManagementDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		deploymentID := vars["id"]
		secretKey := vars["key"]

		if deploymentID == "" || secretKey == "" {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
				Code:    "missing_parameters",
				Message: "Deployment ID and secret key are required",
			})
			return
		}

		var req domain.UpdateSecretRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
				Code:    "invalid_json",
				Message: "Invalid request body",
				Hint:    err.Error(),
			})
			return
		}

		// Validate value
		if err := validateSecretValue(req.Value); err != nil {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
				Code:    "invalid_value",
				Message: err.Error(),
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancel()

		_, m, cfg, workdir, err := getDeploymentContext(ctx, deps.Repo, deploymentID)
		if err != nil {
			status := http.StatusInternalServerError
			if strings.Contains(err.Error(), "not found") {
				status = http.StatusNotFound
			}
			httputil.WriteAPIError(w, status, httputil.APIError{
				Code:    "deployment_context_error",
				Message: err.Error(),
			})
			return
		}

		// Update the secret
		if err := UpdateSecretOnVPS(ctx, deps.SSHRunner, cfg, workdir, secretKey, req.Value, m.Scenario.ID); err != nil {
			status := http.StatusInternalServerError
			if strings.Contains(err.Error(), "not found") {
				status = http.StatusNotFound
			}
			httputil.WriteAPIError(w, status, httputil.APIError{
				Code:    "update_secret_failed",
				Message: "Failed to update secret",
				Hint:    err.Error(),
			})
			return
		}

		response := domain.NewSecretOperationResponse(true, secretKey, "updated", "Secret updated successfully")

		// Optionally restart the scenario
		if req.RestartScenario {
			if err := restartScenarioOnVPS(ctx, deps.SSHRunner, cfg, workdir, m.Scenario.ID); err != nil {
				response.Message = fmt.Sprintf("Secret updated but scenario restart failed: %v", err)
				response.ScenarioRestart = false
			} else {
				response.ScenarioRestart = true
				response.Message = "Secret updated and scenario restarted"
			}
		}

		httputil.WriteJSON(w, http.StatusOK, response)
	}
}

// HandleDeleteVPSSecret returns an HTTP handler that deletes a secret from the VPS.
//
// DELETE /api/v1/deployments/{id}/secrets/{key}
//
// Request body:
//
//	{
//	  "confirmation": "DELETE",
//	  "restart_scenario": false
//	}
func HandleDeleteVPSSecret(deps ManagementDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		deploymentID := vars["id"]
		secretKey := vars["key"]

		if deploymentID == "" || secretKey == "" {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
				Code:    "missing_parameters",
				Message: "Deployment ID and secret key are required",
			})
			return
		}

		var req domain.DeleteSecretRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
				Code:    "invalid_json",
				Message: "Invalid request body",
				Hint:    err.Error(),
			})
			return
		}

		// Validate confirmation
		if req.Confirmation != "DELETE" {
			httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
				Code:    "invalid_confirmation",
				Message: "Confirmation must be exactly 'DELETE'",
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancel()

		_, m, cfg, workdir, err := getDeploymentContext(ctx, deps.Repo, deploymentID)
		if err != nil {
			status := http.StatusInternalServerError
			if strings.Contains(err.Error(), "not found") {
				status = http.StatusNotFound
			}
			httputil.WriteAPIError(w, status, httputil.APIError{
				Code:    "deployment_context_error",
				Message: err.Error(),
			})
			return
		}

		// Delete the secret
		if err := DeleteSecretFromVPS(ctx, deps.SSHRunner, cfg, workdir, secretKey, m.Scenario.ID); err != nil {
			status := http.StatusInternalServerError
			if strings.Contains(err.Error(), "not found") {
				status = http.StatusNotFound
			}
			httputil.WriteAPIError(w, status, httputil.APIError{
				Code:    "delete_secret_failed",
				Message: "Failed to delete secret",
				Hint:    err.Error(),
			})
			return
		}

		response := domain.NewSecretOperationResponse(true, secretKey, "deleted", "Secret deleted successfully")

		// Optionally restart the scenario
		if req.RestartScenario {
			if err := restartScenarioOnVPS(ctx, deps.SSHRunner, cfg, workdir, m.Scenario.ID); err != nil {
				response.Message = fmt.Sprintf("Secret deleted but scenario restart failed: %v", err)
				response.ScenarioRestart = false
			} else {
				response.ScenarioRestart = true
				response.Message = "Secret deleted and scenario restarted"
			}
		}

		httputil.WriteJSON(w, http.StatusOK, response)
	}
}
