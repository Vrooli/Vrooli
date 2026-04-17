package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"scenario-to-cloud/internal/shellutil"
	"scenario-to-cloud/ssh"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
)

// Metadata contains metadata about the secrets.json file.
type Metadata struct {
	Environment string    `json:"environment"`
	LastUpdated time.Time `json:"last_updated"`
	Notes       string    `json:"notes"`
	GeneratedBy string    `json:"generated_by"`
	ScenarioID  string    `json:"scenario_id"`
}

// JSONPayload represents the structure of ~/.vrooli/secrets.json.
// This matches the format expected by secrets::resolve() in Vrooli core.
type JSONPayload struct {
	Metadata Metadata `json:"_metadata"`
	// Secrets are added as top-level keys via custom marshaling
	secrets map[string]string
}

// MarshalJSON flattens secrets to top-level while keeping _metadata.
func (p JSONPayload) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	m["_metadata"] = p.Metadata
	for k, v := range p.secrets {
		m[k] = v
	}
	return json.MarshalIndent(m, "", "  ")
}

func parseVPSSecretsData(data []byte) (*VPSSecretsData, error) {
	if len(strings.TrimSpace(string(data))) == 0 {
		return &VPSSecretsData{Secrets: map[string]string{}}, nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("invalid secrets.json: %w", err)
	}

	parsed := &VPSSecretsData{
		Secrets: map[string]string{},
	}
	for key, value := range raw {
		if key == "_metadata" {
			var metadata Metadata
			if err := json.Unmarshal(value, &metadata); err != nil {
				return nil, fmt.Errorf("invalid secrets metadata: %w", err)
			}
			parsed.Metadata = metadata
			continue
		}

		var secret string
		if err := json.Unmarshal(value, &secret); err != nil {
			return nil, fmt.Errorf("secret %q must be a JSON string", key)
		}
		parsed.Secrets[key] = secret
	}

	return parsed, nil
}

// ReadFromVPS reads existing secrets.json from the VPS if it exists.
// Returns nil map if file doesn't exist (not an error - just means fresh install).
func ReadFromVPS(
	ctx context.Context,
	sshRunner ssh.Runner,
	cfg ssh.Config,
	_ string,
) (map[string]string, error) {
	secretsPath := remoteUserSecretsPath()

	// Try to read existing secrets file
	cmd := fmt.Sprintf("cat %s 2>/dev/null || echo '{}'", shellutil.QuoteSingle(secretsPath))
	result, err := sshRunner.Run(ctx, cfg, cmd, ssh.DefaultRunOptions())
	if err != nil {
		return nil, fmt.Errorf("read secrets.json: %w", err)
	}

	// Parse the JSON to extract existing secrets
	parsed, err := parseVPSSecretsData([]byte(result.Stdout))
	if err != nil {
		return nil, err
	}

	return parsed.Secrets, nil
}

// WriteToVPS writes secrets.json to the VPS via SSH.
// This creates ~/.vrooli/secrets.json with generated credentials BEFORE resource startup.
// IMPORTANT: Preserves existing secrets to avoid breaking database connections on redeploy.
func WriteToVPS(
	ctx context.Context,
	sshRunner ssh.Runner,
	cfg ssh.Config,
	workdir string,
	secrets []GeneratedSecret,
	userSecrets map[string]string,
	scenarioID string,
) error {
	if len(secrets) == 0 && len(userSecrets) == 0 {
		return nil // Nothing to write
	}

	// CRITICAL: Read existing secrets first to preserve them across redeployments
	// This prevents "password authentication failed" errors when redeploying
	existingSecrets, err := ReadFromVPS(ctx, sshRunner, cfg, workdir)
	if err != nil {
		return fmt.Errorf("read existing secrets.json: %w", err)
	}

	// Build secrets map, preserving existing values for per_install_generated secrets
	secretsMap := make(map[string]string)
	for k, v := range existingSecrets {
		if v != "" {
			secretsMap[k] = v
		}
	}
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
	for key, value := range userSecrets {
		if strings.TrimSpace(value) == "" {
			continue
		}
		secretsMap[key] = value
	}

	payload := JSONPayload{
		Metadata: Metadata{
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
	secretsDir := remoteVrooliDir()
	secretsPath := remoteUserSecretsPath()

	// Write secrets.json with proper permissions (600 = owner read/write only)
	// Use printf with %s to avoid shell interpretation of special characters
	// The JSON is passed through shellutil.QuoteSingle to escape it safely
	cmd := fmt.Sprintf(
		"mkdir -p %s && printf '%%s' %s > %s && chmod 600 %s",
		shellutil.QuoteSingle(secretsDir),
		shellutil.QuoteSingle(string(jsonBytes)),
		shellutil.QuoteSingle(secretsPath),
		shellutil.QuoteSingle(secretsPath),
	)

	result, err := sshRunner.Run(ctx, cfg, cmd, ssh.DefaultRunOptions())
	if err != nil {
		return fmt.Errorf("write secrets.json: %w (exit: %d, stderr: %s)", err, result.ExitCode, result.Stderr)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("write secrets.json failed: exit %d, stderr: %s", result.ExitCode, result.Stderr)
	}

	return nil
}

func remoteVrooliDir() string {
	dir, _ := repocontract.VrooliUserRoot("$HOME")
	return shellutil.SafeRemoteJoin(dir)
}

func remoteUserSecretsPath() string {
	path, _ := repocontract.UserPlaintextSecretsPath("$HOME")
	return shellutil.SafeRemoteJoin(path)
}

// BuildJSON builds the secrets.json content without writing it.
// Useful for testing or including in bundles.
func BuildJSON(secrets []GeneratedSecret, scenarioID string) ([]byte, error) {
	secretsMap := make(map[string]string)
	for _, s := range secrets {
		secretsMap[s.Key] = s.Value
	}

	payload := JSONPayload{
		Metadata: Metadata{
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

// VPSSecretsData represents the complete parsed secrets.json file from VPS.
type VPSSecretsData struct {
	Metadata Metadata
	Secrets  map[string]string
}

// ReadAllFromVPS reads secrets.json from VPS and returns structured data with metadata.
// Unlike ReadFromVPS, this preserves the metadata for display purposes.
func ReadAllFromVPS(
	ctx context.Context,
	sshRunner ssh.Runner,
	cfg ssh.Config,
	workdir string,
) (*VPSSecretsData, error) {
	secretsPath := shellutil.SafeRemoteJoin(workdir, ".vrooli", "secrets.json")

	// Try to read existing secrets file
	cmd := fmt.Sprintf("cat %s 2>/dev/null || echo '{}'", shellutil.QuoteSingle(secretsPath))
	result, err := sshRunner.Run(ctx, cfg, cmd, ssh.DefaultRunOptions())
	if err != nil {
		return nil, fmt.Errorf("read secrets.json: %w", err)
	}

	return parseVPSSecretsData([]byte(result.Stdout))
}

// AddSecretToVPS adds a new secret to the VPS secrets.json file.
// Returns an error if the key already exists.
func AddSecretToVPS(
	ctx context.Context,
	sshRunner ssh.Runner,
	cfg ssh.Config,
	workdir string,
	key string,
	value string,
	scenarioID string,
) error {
	// Read existing secrets
	data, err := ReadAllFromVPS(ctx, sshRunner, cfg, workdir)
	if err != nil {
		return fmt.Errorf("read existing secrets: %w", err)
	}

	// Check if key already exists
	if _, exists := data.Secrets[key]; exists {
		return fmt.Errorf("secret key %q already exists", key)
	}

	// Add new secret
	data.Secrets[key] = value

	// Write back
	return writeSecretsData(ctx, sshRunner, cfg, workdir, data, scenarioID)
}

// UpdateSecretOnVPS updates an existing secret value on the VPS.
// Returns an error if the key doesn't exist.
func UpdateSecretOnVPS(
	ctx context.Context,
	sshRunner ssh.Runner,
	cfg ssh.Config,
	workdir string,
	key string,
	value string,
	scenarioID string,
) error {
	// Read existing secrets
	data, err := ReadAllFromVPS(ctx, sshRunner, cfg, workdir)
	if err != nil {
		return fmt.Errorf("read existing secrets: %w", err)
	}

	// Check if key exists
	if _, exists := data.Secrets[key]; !exists {
		return fmt.Errorf("secret key %q not found", key)
	}

	// Update secret
	data.Secrets[key] = value

	// Write back
	return writeSecretsData(ctx, sshRunner, cfg, workdir, data, scenarioID)
}

// DeleteSecretFromVPS removes a secret from the VPS secrets.json file.
// Returns an error if the key doesn't exist.
func DeleteSecretFromVPS(
	ctx context.Context,
	sshRunner ssh.Runner,
	cfg ssh.Config,
	workdir string,
	key string,
	scenarioID string,
) error {
	// Read existing secrets
	data, err := ReadAllFromVPS(ctx, sshRunner, cfg, workdir)
	if err != nil {
		return fmt.Errorf("read existing secrets: %w", err)
	}

	// Check if key exists
	if _, exists := data.Secrets[key]; !exists {
		return fmt.Errorf("secret key %q not found", key)
	}

	// Delete secret
	delete(data.Secrets, key)

	// Write back
	return writeSecretsData(ctx, sshRunner, cfg, workdir, data, scenarioID)
}

// writeSecretsData writes the secrets data back to VPS.
// This is a helper function used by Add, Update, and Delete operations.
func writeSecretsData(
	ctx context.Context,
	sshRunner ssh.Runner,
	cfg ssh.Config,
	workdir string,
	data *VPSSecretsData,
	scenarioID string,
) error {
	// Use the scenario ID from metadata if available, otherwise use provided
	effectiveScenarioID := scenarioID
	if data.Metadata.ScenarioID != "" {
		effectiveScenarioID = data.Metadata.ScenarioID
	}

	payload := JSONPayload{
		Metadata: Metadata{
			Environment: "production",
			LastUpdated: time.Now().UTC(),
			Notes:       "Updated via scenario-to-cloud secrets management",
			GeneratedBy: "scenario-to-cloud",
			ScenarioID:  effectiveScenarioID,
		},
		secrets: data.Secrets,
	}

	jsonBytes, err := payload.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshal secrets.json: %w", err)
	}

	// Paths on VPS
	secretsDir := shellutil.SafeRemoteJoin(workdir, ".vrooli")
	secretsPath := shellutil.SafeRemoteJoin(secretsDir, "secrets.json")

	// Write secrets.json with proper permissions (600 = owner read/write only)
	cmd := fmt.Sprintf(
		"mkdir -p %s && printf '%%s' %s > %s && chmod 600 %s",
		shellutil.QuoteSingle(secretsDir),
		shellutil.QuoteSingle(string(jsonBytes)),
		shellutil.QuoteSingle(secretsPath),
		shellutil.QuoteSingle(secretsPath),
	)

	result, err := sshRunner.Run(ctx, cfg, cmd, ssh.DefaultRunOptions())
	if err != nil {
		return fmt.Errorf("write secrets.json: %w (exit: %d, stderr: %s)", err, result.ExitCode, result.Stderr)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("write secrets.json failed: exit %d, stderr: %s", result.ExitCode, result.Stderr)
	}

	return nil
}
