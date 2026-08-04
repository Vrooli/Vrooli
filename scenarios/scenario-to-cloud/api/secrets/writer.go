package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"scenario-to-cloud/internal/shellutil"
	"scenario-to-cloud/ssh"

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
	// Validated before any remote call. The identity is built from this, so an
	// empty value would otherwise send a run of `vrooli//<field>` queries to the
	// target before anything noticed.
	if strings.TrimSpace(scenarioID) == "" {
		return fmt.Errorf("provision remote credentials: scenario id is required to name the credential identity")
	}

	// A generated secret is written only when the remote store does not already
	// hold one. Regenerating a database password on redeploy is what produced
	// "password authentication failed" against a database that still had the
	// old one, so preservation is load-bearing rather than an optimization.
	//
	// Preservation now asks the remote authority whether a value exists instead
	// of reading a remote plaintext file — and it never reads the value back.
	// Knowing that something is stored is the whole question here.
	secretsMap := make(map[string]string)
	for _, s := range secrets {
		configured, err := remoteCredentialConfigured(ctx, sshRunner, cfg, scenarioID, s.Key)
		if err != nil {
			return err
		}
		if configured {
			continue
		}
		secretsMap[s.Key] = s.Value
	}
	// An operator-supplied value is an explicit instruction for this deploy and
	// overrides what is stored.
	for key, value := range userSecrets {
		if strings.TrimSpace(value) == "" {
			continue
		}
		secretsMap[key] = value
	}

	// Each value is provisioned into the remote host's own credential
	// authority, one SSH call per secret, with the value on standard input.
	//
	// It used to be one call that embedded the whole secrets.json in the
	// command string. A command string becomes argv for the local ssh process
	// and the argument to the remote shell, so every value was visible in both
	// process listings for the duration of the deploy — the precise exposure
	// every credential-store adapter in this platform is built to avoid. It
	// also left a plaintext secrets.json on the VPS afterwards.
	//
	// The remote store is the same seam as the local one: on a headless VPS
	// that is the encrypted file store, so the values land encrypted at rest
	// under a key the file alone does not contain.
	identity := "vrooli/" + strings.TrimSpace(scenarioID)
	if err := ensureRemoteCredentialStore(ctx, sshRunner, cfg); err != nil {
		return err
	}

	keys := make([]string, 0, len(secretsMap))
	for key := range secretsMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := secretsMap[key]
		if strings.TrimSpace(value) == "" {
			continue
		}
		field := CredentialField(key)
		if field == "" {
			continue
		}
		opts := ssh.DefaultRunOptions()
		opts.Stdin = []byte(value)
		command := fmt.Sprintf("vrooli credentials provision --identity %s --field %s",
			shellutil.QuoteSingle(identity), shellutil.QuoteSingle(field))
		result, err := sshRunner.Run(ctx, cfg, command, opts)
		if err != nil {
			// The value is never echoed back, so neither is it in this error.
			return fmt.Errorf("provision remote credential %s/%s: %w (exit: %d, stderr: %s)",
				identity, field, err, result.ExitCode, result.Stderr)
		}
		if result.ExitCode != 0 {
			return fmt.Errorf("provision remote credential %s/%s failed: exit %d, stderr: %s",
				identity, field, result.ExitCode, result.Stderr)
		}
	}

	return nil
}

// remoteCredentialConfigured reports whether the target host already holds a
// value for this secret. It reads the status, never the value: a deploy has no
// business materializing a credential it is only trying not to overwrite.
func remoteCredentialConfigured(ctx context.Context, sshRunner ssh.Runner, cfg ssh.Config, scenarioID, key string) (bool, error) {
	field := CredentialField(key)
	if field == "" {
		return false, nil
	}
	identity := "vrooli/" + strings.TrimSpace(scenarioID)
	command := fmt.Sprintf("vrooli credentials status --identity %s --field %s --format json",
		shellutil.QuoteSingle(identity), shellutil.QuoteSingle(field))
	result, err := sshRunner.Run(ctx, cfg, command, ssh.DefaultRunOptions())
	if err != nil || result.ExitCode != 0 {
		return false, fmt.Errorf("read remote credential status for %s/%s: exit %d, stderr: %s",
			identity, field, result.ExitCode, result.Stderr)
	}
	var status struct {
		Configured    bool   `json:"configured"`
		ProviderState string `json:"provider_state"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &status); err != nil {
		return false, fmt.Errorf("remote credential status for %s/%s was unreadable: %w", identity, field, err)
	}
	// "not configured" is only meaningful when the provider answered. Treating
	// an unreachable store as "absent" would regenerate a password that is in
	// fact still in use.
	if status.ProviderState != "available" {
		return false, fmt.Errorf("remote credential store is %s; refusing to decide whether %s/%s already exists",
			status.ProviderState, identity, field)
	}
	return status.Configured, nil
}

// CredentialField converts a bundle secret key into the durable field name the
// credential authority addresses. It matches the normalization the deploy path
// uses, so a value written here is a value the scenario can read back.
func CredentialField(key string) string {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return ""
	}
	return strings.ToLower(strings.NewReplacer("_", "-", ".", "-").Replace(trimmed))
}

// ensureRemoteCredentialStore fails early, and by name, when the target host
// cannot accept a credential.
//
// Without this the deploy would report a per-secret provisioning failure whose
// real cause — no vrooli on the host, or a store nobody has initialized — is
// two layers down. A deploy that half-provisions is worse than one that
// refuses, because the operator cannot tell which values landed.
func ensureRemoteCredentialStore(ctx context.Context, sshRunner ssh.Runner, cfg ssh.Config) error {
	result, err := sshRunner.Run(ctx, cfg, "vrooli credentials doctor --format json", ssh.DefaultRunOptions())
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf(
			"remote host cannot accept credentials: `vrooli credentials doctor` did not run there (exit %d, stderr: %s). "+
				"The target must have the vrooli CLI installed; on a headless host also run `vrooli credentials store init`",
			result.ExitCode, result.Stderr)
	}
	var diagnosis struct {
		Provider struct {
			Condition string `json:"condition"`
			Fix       string `json:"fix"`
		} `json:"provider"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &diagnosis); err != nil {
		return fmt.Errorf("remote credential diagnosis was unreadable: %w", err)
	}
	if diagnosis.Provider.Condition != "available" {
		return fmt.Errorf("remote credential store is %s: %s",
			diagnosis.Provider.Condition, diagnosis.Provider.Fix)
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
