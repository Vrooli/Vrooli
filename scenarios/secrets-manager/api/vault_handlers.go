package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
)

type VaultHandlers struct {
	db        *database.RoutedDB
	logger    *Logger
	validator *SecretValidator
}

func NewVaultHandlers(db *database.RoutedDB, logger *Logger, validator *SecretValidator) *VaultHandlers {
	return &VaultHandlers{
		db:        db,
		logger:    logger,
		validator: validator,
	}
}

// RegisterRoutes mounts credential-authority endpoints.
func (h *VaultHandlers) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/secrets/status", h.Status).Methods("GET", "POST")
	router.HandleFunc("/secrets/provision", h.Provision).Methods("POST")
	router.HandleFunc("/secrets/validate", h.Validate).Methods("GET", "POST")
	router.HandleFunc("/doctor", h.Doctor).Methods("GET")
	router.HandleFunc("/keyring/inspect", h.KeyringInspect).Methods("GET")
	router.HandleFunc("/keyring/repair", h.KeyringRepair).Methods("POST")
}

// Status returns credential-authority metadata only.
func (h *VaultHandlers) Status(w http.ResponseWriter, r *http.Request) {
	resourceFilter := r.URL.Query().Get("resource")

	status, err := getVaultSecretsStatus(resourceFilter)
	if err != nil {
		if h.logger != nil {
			h.logger.Info("credential authority status unavailable: %v", err)
		}
		http.Error(w, "credential authority status is unavailable", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}

// Provision writes supplied values to the canonical credential authority.
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
	_ = json.NewEncoder(w).Encode(result)
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
	resource = strings.TrimSpace(resource)
	if resource == "" {
		return nil, fmt.Errorf("resource is required for credential provisioning")
	}

	response := &ProvisionResponse{
		Resource: resource,
	}

	results, provisionErr := h.provisionSecretsToVault(ctx, resource, secrets)
	response.Details = results
	response.VaultStored = countSuccessfullyStoredSecrets(results)
	response.StoredSecrets = response.VaultStored

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

func (h *VaultHandlers) provisionSecretsToVault(ctx context.Context, resourceName string, secrets map[string]string) ([]secretProvisionResult, error) {
	results := []secretProvisionResult{}
	if len(secrets) == 0 {
		return results, fmt.Errorf("no secrets provided")
	}
	errs := []string{}
	for rawKey, rawValue := range secrets {
		envKey := strings.ToUpper(strings.TrimSpace(rawKey))
		value := strings.TrimSpace(rawValue)
		if envKey == "" || value == "" {
			continue
		}
		descriptor, err := credentialDescriptorForEnv(resourceName, envKey)
		if err != nil {
			result := secretProvisionResult{EnvKey: envKey, Status: "failed", Error: err.Error()}
			results = append(results, result)
			errs = append(errs, fmt.Sprintf("%s: %v", envKey, err))
			continue
		}
		result := secretProvisionResult{EnvKey: envKey, VaultPath: descriptor.LogicalID, VaultKey: descriptor.Field}
		if err := provisionCredential(ctx, descriptor, value); err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			errs = append(errs, fmt.Sprintf("%s: %v", envKey, err))
		} else {
			result.Status = "stored"
			h.recordSecretProvision(ctx, resourceName, envKey, descriptor.LogicalID)
		}
		results = append(results, result)
	}
	if len(errs) > 0 {
		return results, errors.New(strings.Join(errs, "; "))
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
	provisionedBy := "unknown"
	if currentUser, userErr := user.Current(); userErr == nil && currentUser.Username != "" {
		provisionedBy = currentUser.Username
	}
	_, err = h.db.ExecContext(ctx, `
			INSERT INTO secret_provisions (resource_secret_id, storage_method, storage_location, provisioned_at, provisioned_by, provision_status)
			VALUES ($1, 'native-secure-store', $2, CURRENT_TIMESTAMP, $3, 'active')
		`, secretID, vaultPath, provisionedBy)
	if err != nil {
		if h.logger != nil {
			h.logger.Info("failed to record secret provision for %s: %v", envKey, err)
		}
	}
}

type credentialDescriptor struct {
	LogicalID   string `json:"logical_id"`
	Field       string `json:"field"`
	Env         string `json:"env"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

func credentialDescriptorForEnv(resourceName, env string) (credentialDescriptor, error) {
	root := getVrooliRoot()
	if root == "" {
		return credentialDescriptor{}, fmt.Errorf("resolve repository root for credential descriptor")
	}
	path := filepath.Join(root, "resources", resourceName, "resource.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return credentialDescriptor{}, fmt.Errorf("read credential descriptor: %w", err)
	}
	var resource struct {
		Credentials struct {
			Descriptors []credentialDescriptor `json:"descriptors"`
		} `json:"credentials"`
	}
	if err := json.Unmarshal(data, &resource); err != nil {
		return credentialDescriptor{}, fmt.Errorf("parse credential descriptor: %w", err)
	}
	for _, descriptor := range resource.Credentials.Descriptors {
		if strings.EqualFold(strings.TrimSpace(descriptor.Env), env) && strings.TrimSpace(descriptor.LogicalID) != "" {
			if strings.TrimSpace(descriptor.Field) == "" {
				descriptor.Field = "value"
			}
			return descriptor, nil
		}
	}
	return credentialDescriptor{}, fmt.Errorf("%s is not a declared credential for resource %s", env, resourceName)
}

func credentialDescriptorsForResource(resourceName string) ([]credentialDescriptor, error) {
	root := getVrooliRoot()
	if root == "" {
		return nil, fmt.Errorf("resolve repository root for credential descriptors")
	}
	data, err := os.ReadFile(filepath.Join(root, "resources", resourceName, "resource.json"))
	if err != nil {
		return nil, fmt.Errorf("read credential descriptors: %w", err)
	}
	var resource struct {
		Credentials struct {
			Descriptors []credentialDescriptor `json:"descriptors"`
		} `json:"credentials"`
	}
	if err := json.Unmarshal(data, &resource); err != nil {
		return nil, fmt.Errorf("parse credential descriptors: %w", err)
	}
	return resource.Credentials.Descriptors, nil
}

var provisionCredential = provisionNativeCredential

func provisionNativeCredential(ctx context.Context, descriptor credentialDescriptor, value string) error {
	if err := secretsProvision(ctx, descriptor.LogicalID, descriptor.Field, value); err != nil {
		return fmt.Errorf("provision native credential: %w", err)
	}
	return nil
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

	response, err := h.validator.ValidateSecretsContext(r.Context(), req.Resource)
	if err != nil {
		http.Error(w, fmt.Sprintf("Validation failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
