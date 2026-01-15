package domain

import "time"

// VPSSecretEntry represents a single secret stored on the VPS.
type VPSSecretEntry struct {
	Key         string `json:"key"`
	Value       string `json:"value,omitempty"` // Omitted when masked
	Masked      bool   `json:"masked"`
	Source      string `json:"source"` // "auto_generated", "user_provided", "manual"
	LastUpdated string `json:"last_updated,omitempty"`
}

// VPSSecretsMetadata contains metadata about the secrets.json file on VPS.
type VPSSecretsMetadata struct {
	Environment string `json:"environment"`
	LastUpdated string `json:"last_updated"`
	ScenarioID  string `json:"scenario_id"`
	GeneratedBy string `json:"generated_by,omitempty"`
	Notes       string `json:"notes,omitempty"`
}

// ListVPSSecretsResponse is the response for GET /deployments/{id}/secrets.
type ListVPSSecretsResponse struct {
	Secrets   []VPSSecretEntry   `json:"secrets"`
	Metadata  VPSSecretsMetadata `json:"metadata"`
	Timestamp string             `json:"timestamp"`
}

// GetVPSSecretResponse is the response for GET /deployments/{id}/secrets/{key}.
type GetVPSSecretResponse struct {
	Secret    VPSSecretEntry `json:"secret"`
	Timestamp string         `json:"timestamp"`
}

// CreateSecretRequest is the request body for POST /deployments/{id}/secrets.
type CreateSecretRequest struct {
	Key             string `json:"key"`
	Value           string `json:"value"`
	RestartScenario bool   `json:"restart_scenario"`
}

// UpdateSecretRequest is the request body for PUT /deployments/{id}/secrets/{key}.
type UpdateSecretRequest struct {
	Value           string `json:"value"`
	RestartScenario bool   `json:"restart_scenario"`
}

// DeleteSecretRequest is the request body for DELETE /deployments/{id}/secrets/{key}.
type DeleteSecretRequest struct {
	Confirmation    string `json:"confirmation"` // Must be "DELETE"
	RestartScenario bool   `json:"restart_scenario"`
}

// SecretOperationResponse is the response for create/update/delete operations.
type SecretOperationResponse struct {
	OK              bool   `json:"ok"`
	Key             string `json:"key"`
	Action          string `json:"action"` // "created", "updated", "deleted"
	Message         string `json:"message"`
	ScenarioRestart bool   `json:"scenario_restart,omitempty"` // True if scenario was restarted
	Timestamp       string `json:"timestamp"`
}

// SecretKeyValidationRegex is the regex pattern for valid secret keys.
// Keys must start with uppercase letter and contain only uppercase letters, numbers, and underscores.
const SecretKeyValidationRegex = `^[A-Z][A-Z0-9_]*$`

// ReservedKeyPrefixes are prefixes that cannot be used for manual secrets.
var ReservedKeyPrefixes = []string{"VROOLI_INTERNAL_", "_"}

// MaxSecretKeyLength is the maximum allowed length for a secret key.
const MaxSecretKeyLength = 256

// MaxSecretValueLength is the maximum allowed length for a secret value (64KB).
const MaxSecretValueLength = 64 * 1024

// VPSSecretsData represents the complete parsed secrets.json file from VPS.
type VPSSecretsData struct {
	Metadata VPSSecretsMetadata `json:"_metadata"`
	Secrets  map[string]string  `json:"-"` // Populated from top-level keys
}

// NewSecretOperationResponse creates a new SecretOperationResponse with current timestamp.
func NewSecretOperationResponse(ok bool, key, action, message string) SecretOperationResponse {
	return SecretOperationResponse{
		OK:        ok,
		Key:       key,
		Action:    action,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}
