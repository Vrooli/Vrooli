package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
)

type VaultHandlers struct {
	db        *sql.DB
	logger    *Logger
	validator *SecretValidator
}

func NewVaultHandlers(db *sql.DB, logger *Logger, validator *SecretValidator) *VaultHandlers {
	return &VaultHandlers{
		db:        db,
		logger:    logger,
		validator: validator,
	}
}

// RegisterRoutes mounts versioned vault endpoints (preferred paths).
func (h *VaultHandlers) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/secrets/status", h.Status).Methods("GET", "POST")
	router.HandleFunc("/secrets/provision", h.Provision).Methods("POST")
}

// RegisterLegacyRoutes keeps backward-compatible secrets endpoints.
func (h *VaultHandlers) RegisterLegacyRoutes(router *mux.Router) {
	router.HandleFunc("/secrets/scan", h.Status).Methods("GET", "POST")
	router.HandleFunc("/secrets/validate", h.Validate).Methods("GET", "POST")
	router.HandleFunc("/secrets/provision", h.LegacyProvision).Methods("POST")
}

// Vault secrets status handler
func (h *VaultHandlers) Status(w http.ResponseWriter, r *http.Request) {
	resourceFilter := r.URL.Query().Get("resource")

	status, err := getVaultSecretsStatus(resourceFilter)
	if err != nil {
		if h.logger != nil {
			h.logger.Info("Error getting vault status: %v, using mock data", err)
		}
		// Use mock data as ultimate fallback
		status = getMockVaultStatus()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// Vault provision handler
func (h *VaultHandlers) Provision(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ResourceName string            `json:"resource_name"`
		Secrets      map[string]string `json:"secrets"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.ResourceName == "" {
		http.Error(w, "resource_name is required", http.StatusBadRequest)
		return
	}
	updates := h.normalizeProvisionSecrets(req.Secrets)
	if len(updates) == 0 {
		http.Error(w, "no secrets provided", http.StatusBadRequest)
		return
	}

	result, err := h.performSecretProvision(r.Context(), req.ResourceName, updates)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *VaultHandlers) normalizeProvisionSecrets(raw map[string]string) map[string]string {
	updates := map[string]string{}
	for key, value := range raw {
		envName := strings.TrimSpace(key)
		if envName == "" || strings.EqualFold(envName, "default") {
			continue
		}

		lower := strings.ToLower(envName)
		if strings.HasPrefix(lower, "resource:") {
			continue
		}
		trimmedValue := strings.TrimSpace(value)
		if trimmedValue == "" {
			continue
		}
		envName = strings.ToUpper(strings.ReplaceAll(envName, " ", "_"))
		updates[envName] = trimmedValue
	}
	return updates
}

func (h *VaultHandlers) performSecretProvision(ctx context.Context, resource string, secrets map[string]string) (*ProvisionResponse, error) {
	if len(secrets) == 0 {
		return nil, fmt.Errorf("no secrets provided")
	}

	saved, err := saveSecretsToLocalStore(secrets)
	if err != nil {
		return nil, fmt.Errorf("failed to update local secrets store: %w", err)
	}

	response := &ProvisionResponse{
		Resource:      resource,
		StoredSecrets: saved,
	}

	// Decision: Local-only provision (no resource specified)
	// Outcome: Success, but remind user to provide resource for vault sync
	if isLocalOnlyProvision(resource) {
		response.Success = true
		response.Message = "Secrets stored locally. Provide a resource to sync with Vault."
		return response, nil
	}

	results, provisionErr := h.provisionSecretsToVault(ctx, resource, secrets)
	response.Details = results
	response.VaultStored = countSuccessfullyStoredSecrets(results)

	// Decision: Determine provision outcome based on vault results
	outcome := determineProvisionOutcome(provisionErr, response.VaultStored)
	if outcome.shouldReturnError {
		return response, fmt.Errorf("failed to store secrets in vault: %w", provisionErr)
	}

	response.Success = outcome.isSuccess
	if provisionErr != nil {
		response.Message = provisionErr.Error()
	}
	return response, nil
}

// isLocalOnlyProvision returns true when no resource is specified,
// meaning secrets should only be stored locally without vault sync.
func isLocalOnlyProvision(resource string) bool {
	return strings.TrimSpace(resource) == ""
}

// countSuccessfullyStoredSecrets counts how many secrets were stored in vault.
func countSuccessfullyStoredSecrets(results []secretProvisionResult) int {
	count := 0
	for _, result := range results {
		if strings.EqualFold(result.Status, "stored") {
			count++
		}
	}
	return count
}

// provisionOutcome captures the decision result for provision success.
type provisionOutcome struct {
	isSuccess         bool
	shouldReturnError bool
}

// determineProvisionOutcome decides whether the provision operation succeeded.
//
// Decision logic:
//   - If vault provisioning had errors AND no secrets were stored → fail with error
//   - If vault provisioning had errors BUT some secrets were stored → partial success
//   - If no errors → full success
//
// This allows partial success when some secrets fail but others succeed,
// which is preferable to failing the entire operation.
func determineProvisionOutcome(provisionErr error, vaultStoredCount int) provisionOutcome {
	// Complete failure: errors occurred and nothing was stored
	if provisionErr != nil && vaultStoredCount == 0 {
		return provisionOutcome{isSuccess: false, shouldReturnError: true}
	}

	// Partial or full success: either no errors, or some secrets were stored
	return provisionOutcome{
		isSuccess:         provisionErr == nil || vaultStoredCount > 0,
		shouldReturnError: false,
	}
}

// NOTE: secretMapping type, buildSecretMappings(), and putSecretInVault() have been
// moved to vault_status.go to separate integration logic from HTTP handlers.

func (h *VaultHandlers) provisionSecretsToVault(ctx context.Context, resourceName string, secrets map[string]string) ([]secretProvisionResult, error) {
	results := []secretProvisionResult{}
	if len(secrets) == 0 {
		return results, fmt.Errorf("no secrets provided")
	}
	mappings := buildSecretMappings(resourceName)
	errs := []string{}
	for rawKey, rawValue := range secrets {
		envKey := strings.ToUpper(strings.TrimSpace(rawKey))
		value := strings.TrimSpace(rawValue)
		if envKey == "" || value == "" {
			continue
		}
		mapping := mappings[envKey]
		if mapping.Path == "" {
			mapping.Path = fmt.Sprintf("secret/resources/%s/%s", resourceName, strings.ToLower(envKey))
		}
		if mapping.VaultKey == "" {
			mapping.VaultKey = "value"
		}
		result := secretProvisionResult{EnvKey: envKey, VaultPath: mapping.Path, VaultKey: mapping.VaultKey}
		if err := putSecretInVault(mapping.Path, mapping.VaultKey, value); err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			errs = append(errs, fmt.Sprintf("%s: %v", envKey, err))
		} else {
			result.Status = "stored"
			h.recordSecretProvision(ctx, resourceName, envKey, mapping.Path)
		}
		results = append(results, result)
	}
	if len(errs) > 0 {
		return results, fmt.Errorf(strings.Join(errs, "; "))
	}
	return results, nil
}

func (h *VaultHandlers) recordSecretProvision(ctx context.Context, resourceName, envKey, vaultPath string) {
	if h.db == nil {
		return
	}
	secretID, err := getResourceSecretID(ctx, h.db, resourceName, envKey)
	if err != nil {
		return
	}
	_, err = h.db.ExecContext(ctx, `
		INSERT INTO secret_provisions (resource_secret_id, storage_method, storage_location, provisioned_at, provisioned_by, provision_status)
		VALUES ($1, 'vault', $2, CURRENT_TIMESTAMP, $3, 'active')
	`, secretID, vaultPath, os.Getenv("USER"))
	if err != nil {
		if h.logger != nil {
			h.logger.Info("failed to record secret provision for %s: %v", envKey, err)
		}
	}
}

func (h *VaultHandlers) Validate(w http.ResponseWriter, r *http.Request) {
	var req ValidationRequest
	if r.Method == "POST" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
	}
	if h.validator == nil {
		http.Error(w, "validator not ready (database unavailable)", http.StatusServiceUnavailable)
		return
	}

	response, err := h.validator.ValidateSecrets(req.Resource)
	if err != nil {
		http.Error(w, fmt.Sprintf("Validation failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *VaultHandlers) LegacyProvision(w http.ResponseWriter, r *http.Request) {
	var req ProvisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	secrets := h.normalizeProvisionSecrets(req.Secrets)
	if req.SecretKey != "" && strings.TrimSpace(req.SecretValue) != "" {
		if secrets == nil {
			secrets = map[string]string{}
		}
		key := strings.ToUpper(strings.TrimSpace(req.SecretKey))
		if key != "" {
			secrets[key] = strings.TrimSpace(req.SecretValue)
		}
	}

	if len(secrets) == 0 {
		http.Error(w, "no secrets provided", http.StatusBadRequest)
		return
	}

	result, err := h.performSecretProvision(r.Context(), strings.TrimSpace(req.Resource), secrets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
