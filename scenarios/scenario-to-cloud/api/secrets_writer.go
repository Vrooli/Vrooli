package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// SecretsMetadata contains metadata about the secrets.json file.
type SecretsMetadata struct {
	Environment string    `json:"environment"`
	LastUpdated time.Time `json:"last_updated"`
	Notes       string    `json:"notes"`
	GeneratedBy string    `json:"generated_by"`
	ScenarioID  string    `json:"scenario_id"`
}

// SecretsJSONPayload represents the structure of .vrooli/secrets.json.
// This matches the format expected by secrets::resolve() in Vrooli core.
type SecretsJSONPayload struct {
	Metadata SecretsMetadata `json:"_metadata"`
	// Secrets are added as top-level keys via custom marshaling
	secrets map[string]string
}

// MarshalJSON flattens secrets to top-level while keeping _metadata.
func (p SecretsJSONPayload) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	m["_metadata"] = p.Metadata
	for k, v := range p.secrets {
		m[k] = v
	}
	return json.MarshalIndent(m, "", "  ")
}

// ReadSecretsFromVPS reads existing secrets.json from the VPS if it exists.
// Returns nil map if file doesn't exist (not an error - just means fresh install).
func ReadSecretsFromVPS(
	ctx context.Context,
	sshRunner SSHRunner,
	cfg SSHConfig,
	workdir string,
) (map[string]string, error) {
	secretsPath := safeRemoteJoin(workdir, ".vrooli", "secrets.json")

	// Try to read existing secrets file
	cmd := fmt.Sprintf("cat %s 2>/dev/null || echo '{}'", shellQuoteSingle(secretsPath))
	result, err := sshRunner.Run(ctx, cfg, cmd)
	if err != nil {
		return nil, fmt.Errorf("read secrets.json: %w", err)
	}

	// Parse the JSON to extract existing secrets
	var rawJSON map[string]interface{}
	if err := json.Unmarshal([]byte(result.Stdout), &rawJSON); err != nil {
		// If we can't parse it, treat as empty (fresh install)
		return make(map[string]string), nil
	}

	// Extract string values (skip _metadata)
	existing := make(map[string]string)
	for k, v := range rawJSON {
		if k == "_metadata" {
			continue
		}
		if strVal, ok := v.(string); ok {
			existing[k] = strVal
		}
	}

	return existing, nil
}

// WriteSecretsToVPS writes secrets.json to the VPS via SSH.
// This creates .vrooli/secrets.json with generated credentials BEFORE resource startup.
// IMPORTANT: Preserves existing secrets to avoid breaking database connections on redeploy.
func WriteSecretsToVPS(
	ctx context.Context,
	sshRunner SSHRunner,
	cfg SSHConfig,
	workdir string,
	secrets []GeneratedSecret,
	scenarioID string,
) error {
	if len(secrets) == 0 {
		return nil // Nothing to write
	}

	// CRITICAL: Read existing secrets first to preserve them across redeployments
	// This prevents "password authentication failed" errors when redeploying
	existingSecrets, err := ReadSecretsFromVPS(ctx, sshRunner, cfg, workdir)
	if err != nil {
		// Log but don't fail - treat as fresh install
		existingSecrets = make(map[string]string)
	}

	// Build secrets map, preserving existing values for per_install_generated secrets
	secretsMap := make(map[string]string)
	for _, s := range secrets {
		if existing, ok := existingSecrets[s.Key]; ok && existing != "" {
			// PRESERVE existing secret - don't regenerate!
			// This is critical for database passwords, API keys, etc.
			secretsMap[s.Key] = existing
		} else {
			// New secret or empty - use generated value
			secretsMap[s.Key] = s.Value
		}
	}

	payload := SecretsJSONPayload{
		Metadata: SecretsMetadata{
			Environment: "production",
			LastUpdated: time.Now().UTC(),
			Notes:       "Generated during VPS deployment - managed by scenario-to-cloud",
			GeneratedBy: "scenario-to-cloud",
			ScenarioID:  scenarioID,
		},
		secrets: secretsMap,
	}

	jsonBytes, err := payload.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshal secrets.json: %w", err)
	}

	// Paths on VPS
	secretsDir := safeRemoteJoin(workdir, ".vrooli")
	secretsPath := safeRemoteJoin(secretsDir, "secrets.json")

	// Write secrets.json with proper permissions (600 = owner read/write only)
	// Use printf with %s to avoid shell interpretation of special characters
	// The JSON is passed through shellQuoteSingle to escape it safely
	cmd := fmt.Sprintf(
		"mkdir -p %s && printf '%%s' %s > %s && chmod 600 %s",
		shellQuoteSingle(secretsDir),
		shellQuoteSingle(string(jsonBytes)),
		shellQuoteSingle(secretsPath),
		shellQuoteSingle(secretsPath),
	)

	result, err := sshRunner.Run(ctx, cfg, cmd)
	if err != nil {
		return fmt.Errorf("write secrets.json: %w (exit: %d, stderr: %s)", err, result.ExitCode, result.Stderr)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("write secrets.json failed: exit %d, stderr: %s", result.ExitCode, result.Stderr)
	}

	return nil
}

// BuildSecretsJSON builds the secrets.json content without writing it.
// Useful for testing or including in bundles.
func BuildSecretsJSON(secrets []GeneratedSecret, scenarioID string) ([]byte, error) {
	secretsMap := make(map[string]string)
	for _, s := range secrets {
		secretsMap[s.Key] = s.Value
	}

	payload := SecretsJSONPayload{
		Metadata: SecretsMetadata{
			Environment: "production",
			LastUpdated: time.Now().UTC(),
			Notes:       "Generated during VPS deployment - managed by scenario-to-cloud",
			GeneratedBy: "scenario-to-cloud",
			ScenarioID:  scenarioID,
		},
		secrets: secretsMap,
	}

	return payload.MarshalJSON()
}
