package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// -----------------------------------------------------------------------------
// VaultCLI Interface
// -----------------------------------------------------------------------------

// VaultCLI abstracts vault CLI operations for testability.
// This interface enables mocking native credential status responses in tests.
//
// seam: VaultCLI isolates credential authority checks.
type VaultCLI interface {
	// GetSecretsStatus retrieves vault secrets status, optionally filtered by resource.
	GetSecretsStatus(ctx context.Context, resourceFilter string) (*VaultSecretsStatus, error)

	// GetSecret retrieves a single secret value from vault.
	GetSecret(ctx context.Context, resourceName, key string) (string, error)

	// PutSecret stores a secret in vault at the specified path.
	PutSecret(ctx context.Context, path, vaultKey, value string) error
}

// DefaultVaultCLI is retained as an API compatibility seam. Production checks
// the native credential authority through `vrooli credentials status`; it never
// asks a vault for a plaintext value.
type DefaultVaultCLI struct{}

func determineResourceHealthStatus(missingRequiredSecrets int) string {
	switch {
	case missingRequiredSecrets == 0:
		return "healthy"
	case missingRequiredSecrets <= 2:
		return "degraded"
	default:
		return "critical"
	}
}

// NewDefaultVaultCLI creates the production VaultCLI implementation.
func NewDefaultVaultCLI() *DefaultVaultCLI {
	return &DefaultVaultCLI{}
}

// GetSecretsStatus implements VaultCLI using canonical resource descriptors.
func (v *DefaultVaultCLI) GetSecretsStatus(ctx context.Context, resourceFilter string) (*VaultSecretsStatus, error) {
	return getNativeCredentialStatus(ctx, resourceFilter)
}

// GetSecret intentionally refuses value reads. Consumers receive credentials by
// explicit process injection, never through the Secrets Manager API.
func (v *DefaultVaultCLI) GetSecret(ctx context.Context, resourceName, key string) (string, error) {
	return "", fmt.Errorf("plaintext credential reads are not supported")
}

// PutSecret intentionally refuses writes. Provisioning is stdin-only through
// `vrooli credentials provision` in vault_handlers.go.
func (v *DefaultVaultCLI) PutSecret(ctx context.Context, path, vaultKey, value string) error {
	return fmt.Errorf("use canonical credential provisioning")
}

// defaultVaultCLI is the package-level vault CLI instance.
// It can be replaced in tests via SetVaultCLI.
var defaultVaultCLI VaultCLI = NewDefaultVaultCLI()

var credentialStatusCommand = func(ctx context.Context, logicalID, field string) ([]byte, error) {
	return exec.CommandContext(ctx, "vrooli", "credentials", "status", "--format", "json", "--identity", logicalID, "--field", field).Output()
}

// SetVaultCLI replaces the default vault CLI implementation.
// This is primarily used for testing with mock implementations.
func SetVaultCLI(cli VaultCLI) {
	defaultVaultCLI = cli
}

// -----------------------------------------------------------------------------
// Public API (uses VaultCLI interface)
// -----------------------------------------------------------------------------

// Credential authority integration for status metadata.
func getVaultSecretsStatus(resourceFilter string) (*VaultSecretsStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	status, err := defaultVaultCLI.GetSecretsStatus(ctx, resourceFilter)
	if err != nil {
		return nil, fmt.Errorf("Vault status is unavailable: %w", err)
	}
	return status, nil
}

func getNativeCredentialStatus(ctx context.Context, resourceFilter string) (*VaultSecretsStatus, error) {
	status := &VaultSecretsStatus{MissingSecrets: []VaultMissingSecret{}, ResourceStatuses: []VaultResourceStatus{}, LastUpdated: time.Now()}
	resources := listKnownResources()
	for _, resourceName := range resources {
		if resourceFilter != "" && resourceFilter != resourceName {
			continue
		}
		descriptors, err := credentialDescriptorsForResource(resourceName)
		if err != nil {
			continue
		}
		row := VaultResourceStatus{ResourceName: resourceName, SecretsTotal: len(descriptors), LastChecked: time.Now()}
		for _, descriptor := range descriptors {
			configured, _ := credentialConfiguredDescriptor(ctx, descriptor)
			if configured {
				row.SecretsFound++
				continue
			}
			if descriptor.Required {
				row.SecretsMissing++
				status.MissingSecrets = append(status.MissingSecrets, VaultMissingSecret{ResourceName: resourceName, SecretName: descriptor.Env, SecretPath: descriptor.LogicalID, Required: true, Description: descriptor.Description})
			} else {
				row.SecretsOptional++
			}
		}
		row.HealthStatus = determineResourceHealthStatus(row.SecretsMissing)
		status.ResourceStatuses = append(status.ResourceStatuses, row)
	}
	status.TotalResources = len(status.ResourceStatuses)
	for _, row := range status.ResourceStatuses {
		if row.SecretsMissing == 0 {
			status.ConfiguredResources++
		}
	}
	return status, nil
}

func credentialConfigured(ctx context.Context, resourceName, env string) (bool, error) {
	descriptor, err := credentialDescriptorForEnv(resourceName, env)
	if err != nil {
		return false, err
	}
	return credentialConfiguredDescriptor(ctx, descriptor)
}

func credentialConfiguredDescriptor(ctx context.Context, descriptor credentialDescriptor) (bool, error) {
	output, err := credentialStatusCommand(ctx, descriptor.LogicalID, descriptor.Field)
	if err != nil {
		return false, fmt.Errorf("credential authority unavailable: %w", err)
	}
	var result struct {
		Configured bool `json:"configured"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return false, fmt.Errorf("credential authority returned invalid status metadata: %w", err)
	}
	return result.Configured, nil
}

// mergeKnownResources ensures resources without secrets still appear in API responses
// so the UI can render a complete table. It pulls names from .vrooli/service.json
// and the resources directory, then appends zero-secret rows where missing.
func mergeKnownResources(status *VaultSecretsStatus, resourceFilter string) {
	if status == nil {
		return
	}

	existing := make(map[string]struct{}, len(status.ResourceStatuses))
	for _, rs := range status.ResourceStatuses {
		existing[strings.ToLower(rs.ResourceName)] = struct{}{}
	}

	known := listKnownResources()
	now := time.Now()
	for _, name := range known {
		if resourceFilter != "" && resourceFilter != name {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := existing[key]; ok {
			continue
		}
		status.ResourceStatuses = append(status.ResourceStatuses, VaultResourceStatus{
			ResourceName:    name,
			SecretsTotal:    0,
			SecretsFound:    0,
			SecretsMissing:  0,
			SecretsOptional: 0,
			HealthStatus:    "healthy",
			LastChecked:     now,
		})
		existing[key] = struct{}{}
	}

	status.TotalResources = len(status.ResourceStatuses)
	configured := 0
	for _, rs := range status.ResourceStatuses {
		if rs.SecretsMissing == 0 {
			configured++
		}
	}
	status.ConfiguredResources = configured
}

func listKnownResources() []string {
	names := map[string]struct{}{}

	// From .vrooli/service.json dependencies.resources
	servicePath := filepath.Join(getVrooliRoot(), ".vrooli", "service.json")
	if data, err := os.ReadFile(servicePath); err == nil {
		var cfg struct {
			Dependencies struct {
				Resources map[string]struct {
					Enabled *bool `json:"enabled"`
				} `json:"resources"`
			} `json:"dependencies"`
		}
		if err := json.Unmarshal(data, &cfg); err == nil {
			for name, res := range cfg.Dependencies.Resources {
				if res.Enabled != nil && !*res.Enabled {
					continue
				}
				if name != "" {
					names[name] = struct{}{}
				}
			}
		}
	}

	// From resources directory (covers local resources without secrets)
	resourcesDir := filepath.Join(getVrooliRoot(), "resources")
	if entries, err := os.ReadDir(resourcesDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				names[entry.Name()] = struct{}{}
			}
		}
	}

	slice := make([]string, 0, len(names))
	for name := range names {
		slice = append(slice, name)
	}
	sort.Strings(slice)
	return slice
}

func scanResourceDirectory(resourceName, resourceDir string) ([]ResourceSecret, error) {
	var secrets []ResourceSecret

	// Patterns to look for environment variables and credentials
	envVarPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\$\{([A-Z_]+[A-Z0-9_]*)\}`),            // ${VAR_NAME}
		regexp.MustCompile(`\$([A-Z_]+[A-Z0-9_]*)`),                // $VAR_NAME
		regexp.MustCompile(`([A-Z_]+[A-Z0-9_]*)=`),                 // VAR_NAME=
		regexp.MustCompile(`env\.([A-Z_]+[A-Z0-9_]*)`),             // env.VAR_NAME
		regexp.MustCompile(`getenv\("([A-Z_]+[A-Z0-9_]*)"\)`),      // getenv("VAR_NAME")
		regexp.MustCompile(`os[.]Getenv\("([A-Z_]+[A-Z0-9_]*)"\)`), // Go environment getter syntax
	}

	// Walk through resource directory
	err := filepath.WalkDir(resourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip files we can't read
		}

		// Skip directories and non-text files
		if d.IsDir() || !isTextFile(path) {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		// Search for environment variables
		foundVars := make(map[string]bool)
		for _, pattern := range envVarPatterns {
			matches := pattern.FindAllStringSubmatch(string(content), -1)
			for _, match := range matches {
				if len(match) > 1 {
					varName := match[1]
					if !foundVars[varName] && IsLikelySecret(varName) {
						foundVars[varName] = true

						secret := ResourceSecret{
							ID:                uuid.New().String(),
							ResourceName:      resourceName,
							SecretKey:         varName,
							SecretType:        ClassifySecretType(varName),
							Required:          IsLikelyRequired(varName),
							Description:       stringPtr(fmt.Sprintf("Environment variable found in %s", filepath.Base(path))),
							ValidationPattern: nil,
							DocumentationURL:  nil,
							DefaultValue:      nil,
							CreatedAt:         time.Now(),
							UpdatedAt:         time.Now(),
						}
						secrets = append(secrets, secret)
					}
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return secrets, nil
}

func isTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return IsTextFileExtension(ext)
}

func resolveVaultSecretMapping(resourceName, key string) (secretMapping, bool) {
	key = strings.ToUpper(strings.TrimSpace(key))
	if key == "" {
		return secretMapping{}, false
	}
	resourceName = strings.TrimSpace(resourceName)
	if resourceName != "" {
		if descriptor, err := credentialDescriptorForEnv(resourceName, key); err == nil {
			return secretMapping{Path: descriptor.LogicalID, VaultKey: descriptor.Field}, true
		}
		return secretMapping{}, false
	}

	for _, resourceName := range listKnownResources() {
		if descriptor, err := credentialDescriptorForEnv(resourceName, key); err == nil {
			return secretMapping{Path: descriptor.LogicalID, VaultKey: descriptor.Field}, true
		}
	}

	return secretMapping{}, false
}

// secretMapping represents a vault path and key for a secret.
// Moved from vault_handlers.go to separate integration logic from HTTP handlers.
type secretMapping struct {
	Path     string
	VaultKey string
}

// buildSecretMappings builds a mapping of environment variable names to vault paths
// based on the resource's secrets.yaml configuration.
// Moved from vault_handlers.go to separate integration logic from HTTP handlers.
