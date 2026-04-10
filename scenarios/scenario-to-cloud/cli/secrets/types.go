// Package secrets provides secrets management commands for the CLI.
package secrets

// LegacyGetResponse is the response from GET /api/v1/secrets/{scenario}.
// The payload is intentionally loose because the API returns a nested "secrets" object.
type LegacyGetResponse struct {
	Secrets map[string]interface{} `json:"secrets"`
}

// LocalSecretSetRequest writes/updates a local secret.
type LocalSecretSetRequest struct {
	Value    string `json:"value,omitempty"`
	Generate string `json:"generate,omitempty"`
}

// LocalSecretGetResponse is the response for GET local secret endpoints.
type LocalSecretGetResponse struct {
	Key       string `json:"key"`
	Value     string `json:"value,omitempty"`
	Masked    bool   `json:"masked"`
	Scope     string `json:"scope"`
	Scenario  string `json:"scenario_id,omitempty"`
	Path      string `json:"path"`
	Timestamp string `json:"timestamp"`
}

// LocalSecretSetResponse is the response for PUT local secret endpoints.
type LocalSecretSetResponse struct {
	OK        bool   `json:"ok"`
	Key       string `json:"key"`
	Scope     string `json:"scope"`
	Scenario  string `json:"scenario_id,omitempty"`
	Path      string `json:"path"`
	Generated bool   `json:"generated"`
	Timestamp string `json:"timestamp"`
}

// DeploymentSecretEntry is one deployment secret row.
type DeploymentSecretEntry struct {
	Key    string `json:"key"`
	Value  string `json:"value,omitempty"`
	Masked bool   `json:"masked"`
	Source string `json:"source"`
}

// DeploymentSecretGetResponse returns one deployment secret.
type DeploymentSecretGetResponse struct {
	Secret DeploymentSecretEntry `json:"secret"`
}

// DeploymentSecretListResponse returns deployment secret inventory.
type DeploymentSecretListResponse struct {
	Secrets []DeploymentSecretEntry `json:"secrets"`
}

// DeploymentSecretCreateRequest creates a deployment secret.
type DeploymentSecretCreateRequest struct {
	Key             string `json:"key"`
	Value           string `json:"value"`
	RestartScenario bool   `json:"restart_scenario"`
}

// DeploymentSecretUpdateRequest updates a deployment secret.
type DeploymentSecretUpdateRequest struct {
	Value           string `json:"value"`
	RestartScenario bool   `json:"restart_scenario"`
}

// DeploymentSecretDeleteRequest deletes a deployment secret.
type DeploymentSecretDeleteRequest struct {
	Confirmation    string `json:"confirmation"`
	RestartScenario bool   `json:"restart_scenario"`
}

// SecretOperationResponse is returned by create/update/delete APIs.
type SecretOperationResponse struct {
	OK              bool   `json:"ok"`
	Key             string `json:"key"`
	Action          string `json:"action"`
	Message         string `json:"message"`
	ScenarioRestart bool   `json:"scenario_restart,omitempty"`
	Timestamp       string `json:"timestamp"`
}
