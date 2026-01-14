// Package secrets provides secrets management commands for the CLI.
package secrets

// GetResponse is the response from getting secrets for a scenario.
type GetResponse struct {
	ScenarioID string            `json:"scenario_id"`
	Secrets    map[string]Secret `json:"secrets"`
	Timestamp  string            `json:"timestamp"`
}

// Secret represents a secret value.
type Secret struct {
	Key       string `json:"key"`
	Value     string `json:"value,omitempty"` // Only included if reveal=true
	Source    string `json:"source"`          // env, vault, provided, generated
	Required  bool   `json:"required"`
	HasValue  bool   `json:"has_value"`
	Masked    string `json:"masked,omitempty"` // Masked value like "****"
}
