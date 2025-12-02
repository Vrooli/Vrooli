package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

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

	if strings.TrimSpace(resource) == "" {
		response.Success = true
		response.Message = "Secrets stored locally. Provide a resource to sync with Vault."
		return response, nil
	}

	results, provisionErr := h.provisionSecretsToVault(ctx, resource, secrets)
	response.Details = results
	for _, result := range results {
		if strings.EqualFold(result.Status, "stored") {
			response.VaultStored++
		}
	}

	if provisionErr != nil && response.VaultStored == 0 {
		return response, fmt.Errorf("failed to store secrets in vault: %w", provisionErr)
	}

	response.Success = provisionErr == nil || response.VaultStored > 0
	if provisionErr != nil {
		response.Message = provisionErr.Error()
	}
	return response, nil
}

type secretMapping struct {
	Path     string
	VaultKey string
}

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

func buildSecretMappings(resourceName string) map[string]secretMapping {
	mappings := map[string]secretMapping{}
	config, err := loadResourceSecrets(resourceName)
	if err != nil || config == nil {
		return mappings
	}
	replacer := strings.NewReplacer("{resource}", resourceName)
	for _, definitions := range config.Secrets {
		for _, def := range definitions {
			path := strings.TrimSpace(def.Path)
			if path == "" {
				continue
			}
			path = replacer.Replace(path)
			if len(def.Fields) > 0 {
				for _, field := range def.Fields {
					for keyName, env := range field {
						envVar := strings.ToUpper(strings.TrimSpace(env))
						if envVar == "" {
							continue
						}
						mappings[envVar] = secretMapping{Path: path, VaultKey: keyName}
					}
				}
			}
			defaultEnv := strings.ToUpper(strings.TrimSpace(def.DefaultEnv))
			if defaultEnv != "" {
				mappings[defaultEnv] = secretMapping{Path: path, VaultKey: "value"}
			}
			nameAlias := strings.ToUpper(strings.TrimSpace(def.Name))
			if nameAlias != "" {
				alias := fmt.Sprintf("%s_%s", strings.ToUpper(resourceName), strings.ReplaceAll(nameAlias, " ", "_"))
				mappings[alias] = secretMapping{Path: path, VaultKey: "value"}
			}
		}
	}
	return mappings
}

func putSecretInVault(path, vaultKey, value string) error {
	args := []string{"content", "put-secret", "--path", path, "--value", value}
	if vaultKey != "" && vaultKey != "value" {
		args = append(args, "--key", vaultKey)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "resource-vault", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}
	return nil
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
