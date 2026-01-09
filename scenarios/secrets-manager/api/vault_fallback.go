package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// -----------------------------------------------------------------------------
// Health Status Decision Thresholds
// -----------------------------------------------------------------------------
//
// These constants define the thresholds for determining resource health status
// based on the number of missing required secrets.
//
// Decision logic:
//   - 0 missing secrets → healthy (fully operational)
//   - 1-2 missing secrets → degraded (partially operational, attention needed)
//   - 3+ missing secrets → critical (likely non-functional)

const (
	// healthyMissingSecretsThreshold is the maximum number of missing required
	// secrets for a resource to be considered "healthy" (fully configured).
	healthyMissingSecretsThreshold = 0

	// degradedMissingSecretsThreshold is the maximum number of missing required
	// secrets for a resource to be considered "degraded" rather than "critical".
	// Resources with more missing secrets than this threshold are considered critical.
	degradedMissingSecretsThreshold = 2
)

// determineResourceHealthStatus classifies a resource's operational readiness
// based on how many required secrets are missing.
//
// Decision outcomes:
//   - "healthy": All required secrets are configured, resource is fully operational
//   - "degraded": Some required secrets missing, resource may work partially
//   - "critical": Many required secrets missing, resource likely non-functional
func determineResourceHealthStatus(missingRequiredSecrets int) string {
	switch {
	case missingRequiredSecrets <= healthyMissingSecretsThreshold:
		return "healthy"
	case missingRequiredSecrets <= degradedMissingSecretsThreshold:
		return "degraded"
	default:
		return "critical"
	}
}

// isResourceFullyConfigured returns true if a resource has no missing required secrets.
// Used to count configured resources for summary statistics.
func isResourceFullyConfigured(missingRequiredSecrets int) bool {
	return missingRequiredSecrets <= healthyMissingSecretsThreshold
}

// ResourceSecretsConfig represents a resource's secrets.yaml file
type ResourceSecretsConfig struct {
	Version        string                        `yaml:"version"`
	Resource       string                        `yaml:"resource"`
	Description    string                        `yaml:"description"`
	Secrets        map[string][]SecretDefinition `yaml:"secrets"`
	Initialization *InitializationConfig         `yaml:"initialization"`
}

// SecretDefinition represents a single secret configuration
type SecretDefinition struct {
	Name          string              `yaml:"name"`
	Description   string              `yaml:"description"`
	Required      bool                `yaml:"required"`
	DefaultEnv    string              `yaml:"default_env"`
	Example       string              `yaml:"example"`
	Documentation string              `yaml:"documentation"`
	Path          string              `yaml:"path"`
	Format        string              `yaml:"format"`
	Validation    *ValidationConfig   `yaml:"validation"`
	Fallback      string              `yaml:"fallback"`
	Fields        []map[string]string `yaml:"fields"`
	Links         []SecretLink        `yaml:"links"`
	TTL           string              `yaml:"ttl"`
	Renewable     bool                `yaml:"renewable"`
	AutoGenerate  bool                `yaml:"auto_generate"`
	Regenerate    string              `yaml:"regenerate"`
}

type SecretLink struct {
	Label string `yaml:"label"`
	URL   string `yaml:"url"`
}

// InitializationConfig represents initialization guidance
type InitializationConfig struct {
	PromptUser []struct {
		Name       string `yaml:"name"`
		Prompt     string `yaml:"prompt"`
		Validation string `yaml:"validation"`
		Optional   bool   `yaml:"optional"`
	} `yaml:"prompt_user"`
}

// ValidationConfig represents validation rules
type ValidationConfig struct {
	Pattern string `yaml:"pattern"`
}

func extractFieldEnvVars(secret SecretDefinition) []string {
	var envs []string
	for _, field := range secret.Fields {
		for _, value := range field {
			if value != "" {
				envs = append(envs, value)
			}
		}
	}
	return envs
}

func normalizeName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, " ", "_")
	return name
}

func extractPromptInfo(init *InitializationConfig, secret SecretDefinition) (string, string, string) {
	if init == nil {
		return "", "", ""
	}

	secretNameNorm := normalizeName(secret.Name)
	defaultEnvNorm := normalizeName(secret.DefaultEnv)

	for _, prompt := range init.PromptUser {
		promptNorm := normalizeName(prompt.Name)
		if promptNorm == secretNameNorm || (defaultEnvNorm != "" && promptNorm == defaultEnvNorm) {
			acquisitionURL := ""
			if urlStart := strings.Index(prompt.Prompt, "https://"); urlStart != -1 {
				rest := prompt.Prompt[urlStart:]
				urlEnd := strings.IndexAny(rest, " \t\n)")
				if urlEnd == -1 {
					acquisitionURL = rest
				} else {
					acquisitionURL = rest[:urlEnd]
				}
			}
			return prompt.Prompt, prompt.Validation, acquisitionURL
		}
	}

	return "", "", ""
}

// scanResourcesDirectly scans resources directory for secrets.yaml files
func scanResourcesDirectly() ([]string, error) {
	resourcesPath := filepath.Join(getVrooliRoot(), "resources")
	logger.Info("🔍 Scanning resources directory: %s", resourcesPath)
	var resourcesWithSecrets []string

	// Walk through resources directory
	err := filepath.Walk(resourcesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, continue scanning
		}

		// Look for config/secrets.yaml files
		if strings.HasSuffix(path, "config/secrets.yaml") {
			// Extract resource name from path
			rel, _ := filepath.Rel(resourcesPath, path)
			parts := strings.Split(rel, string(filepath.Separator))
			if len(parts) > 0 {
				resourceName := parts[0]
				logger.Info("  📦 Found resource with secrets: %s", resourceName)
				resourcesWithSecrets = append(resourcesWithSecrets, resourceName)
			}
		}
		return nil
	})

	logger.Info("✅ Found %d resources with secrets: %v", len(resourcesWithSecrets), resourcesWithSecrets)
	return resourcesWithSecrets, err
}

// loadResourceSecrets loads secrets configuration for a specific resource
func loadResourceSecrets(resourceName string) (*ResourceSecretsConfig, error) {
	secretsPath := filepath.Join(getVrooliRoot(), "resources", resourceName, "config", "secrets.yaml")

	data, err := os.ReadFile(secretsPath)
	if err != nil {
		return nil, err
	}

	var config ResourceSecretsConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// checkVaultForSecret checks if a secret exists in vault (simplified check)
func checkVaultForSecret(resourceName, secretName string) bool {
	var envName string

	// If resourceName is empty, use secretName directly (for default_env fields)
	if resourceName == "" {
		envName = secretName
	} else {
		// Construct prefixed name for backwards compatibility
		envName = fmt.Sprintf("%s_%s", strings.ToUpper(resourceName), strings.ToUpper(secretName))
		envName = strings.ReplaceAll(envName, "-", "_")
	}

	logger.Info("🔍 checkVaultForSecret: resource=%s, secret=%s, envName=%s", resourceName, secretName, envName)

	// Use getVaultSecret which already implements resource-vault CLI access
	if value, err := getVaultSecret(envName); err == nil && value != "" {
		logger.Info("✅ Found secret %s in vault", envName)
		return true
	}

	// Fall back to environment variables
	if envValue := os.Getenv(envName); envValue != "" {
		logger.Info("✅ Found secret %s in environment", envName)
		return true
	}

	// Fall back to local secrets file (~/.vrooli/secrets.json)
	if value, err := loadLocalSecret(envName); err == nil && value != "" {
		logger.Info("✅ Found secret %s in local secrets store", envName)
		return true
	}

	logger.Info("❌ Secret %s not found in vault or environment", envName)
	return false
}

func loadLocalSecret(key string) (string, error) {
	secretsPath := filepath.Join(getVrooliRoot(), ".vrooli", "secrets.json")
	data, err := os.ReadFile(secretsPath)
	if err != nil {
		return "", err
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", err
	}

	if value, ok := payload[key]; ok {
		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return v, nil
			}
		}
	}

	return "", fmt.Errorf("secret %s not found in local store", key)
}

// getVaultSecretsStatusFallback provides fallback implementation when vault CLI hangs
func getVaultSecretsStatusFallback(resourceFilter string) (*VaultSecretsStatus, error) {
	// Scan resources directly instead of using potentially hanging CLI
	resourcesWithSecrets, err := scanResourcesDirectly()
	if err != nil {
		return nil, fmt.Errorf("failed to scan resources: %w", err)
	}

	var resourceStatuses []VaultResourceStatus
	var allMissingSecrets []VaultMissingSecret
	configuredCount := 0

	for _, resourceName := range resourcesWithSecrets {
		if resourceFilter != "" && resourceFilter != resourceName {
			continue
		}

		// Load secrets configuration
		config, err := loadResourceSecrets(resourceName)
		if err != nil {
			logger.Error("  ❌ Error loading %s secrets: %v", resourceName, err)
			continue // Skip resources we can't read
		}
		var categoryNames []string
		for category := range config.Secrets {
			categoryNames = append(categoryNames, category)
		}
		sort.Strings(categoryNames)
		logger.Info("  📋 Processing %s: categories=%v", resourceName, categoryNames)

		totalSecrets := 0

		status := VaultResourceStatus{
			ResourceName: resourceName,
			LastChecked:  time.Now(),
			AllSecrets:   []VaultSecret{},
		}

		for _, category := range categoryNames {
			secrets := config.Secrets[category]
			totalSecrets += len(secrets)

			for _, secret := range secrets {
				fieldEnvs := extractFieldEnvVars(secret)
				envName := secret.DefaultEnv
				if envName == "" && len(fieldEnvs) > 0 {
					envName = fieldEnvs[0]
				}
				if envName == "" {
					envName = strings.ToUpper(strings.ReplaceAll(secret.Name, "-", "_"))
				}

				configured := false
				if envName != "" {
					configured = checkVaultForSecret("", envName)
				}
				if !configured && len(fieldEnvs) > 0 {
					configured = true
					for _, fieldEnv := range fieldEnvs {
						if !checkVaultForSecret("", fieldEnv) {
							configured = false
							break
						}
					}
				}

				setupInstructions, validationHint, acquisitionURL := extractPromptInfo(config.Initialization, secret)
				if acquisitionURL == "" && len(secret.Links) > 0 {
					acquisitionURL = secret.Links[0].URL
				}

				vaultSecret := VaultSecret{
					Name:              envName,
					Description:       secret.Description,
					Required:          secret.Required,
					Configured:        configured,
					SecretType:        category,
					DocumentationURL:  secret.Documentation,
					AcquisitionURL:    acquisitionURL,
					SetupInstructions: setupInstructions,
					Example:           secret.Example,
					ValidationHint:    validationHint,
				}

				status.AllSecrets = append(status.AllSecrets, vaultSecret)

				if configured {
					status.SecretsFound++
				} else if secret.Required {
					status.SecretsMissing++
					allMissingSecrets = append(allMissingSecrets, VaultMissingSecret{
						ResourceName: resourceName,
						SecretName:   envName,
						SecretPath:   secret.Path,
						Required:     secret.Required,
						Description:  secret.Description,
					})
				} else {
					status.SecretsOptional++
				}
			}
		}

		status.SecretsTotal = totalSecrets

		// Determine health status using named decision function
		status.HealthStatus = determineResourceHealthStatus(status.SecretsMissing)
		if isResourceFullyConfigured(status.SecretsMissing) {
			configuredCount++
		}

		resourceStatuses = append(resourceStatuses, status)
	}

	status := &VaultSecretsStatus{
		TotalResources:      len(resourceStatuses),
		ConfiguredResources: configuredCount,
		MissingSecrets:      allMissingSecrets,
		ResourceStatuses:    resourceStatuses,
		LastUpdated:         time.Now(),
	}
	mergeKnownResources(status, resourceFilter)
	return status, nil
}

// getMockVaultStatus returns mock data for testing when vault is unavailable.
//
// REFACTORING NOTE: This function is currently used as a production fallback
// in vault_handlers.go when both the vault CLI and filesystem-based fallback fail.
// Using hardcoded mock data in production is a responsibility violation - this
// function belongs in test code, not production code. Recommended future refactoring:
//   - Return an error to the client when vault is truly unavailable
//   - Expose a clear "degraded" status in the API response
//   - Move this function to test_helpers.go once production code no longer relies on it
//
// The coupling exists because vault_handlers.Status() falls back to this mock
// data as an "ultimate fallback" rather than propagating the error. Fixing this
// requires updating the handler behavior and potentially the UI to handle the
// unavailable state gracefully.
func getMockVaultStatus() *VaultSecretsStatus {
	return &VaultSecretsStatus{
		TotalResources:      5,
		ConfiguredResources: 3,
		MissingSecrets: []VaultMissingSecret{
			{
				ResourceName: "openrouter",
				SecretName:   "api_key",
				Required:     true,
				Description:  "OpenRouter API key for AI model access",
			},
			{
				ResourceName: "postgres",
				SecretName:   "password",
				Required:     true,
				Description:  "PostgreSQL database password",
			},
		},
		ResourceStatuses: []VaultResourceStatus{
			{
				ResourceName:    "vault",
				SecretsTotal:    3,
				SecretsFound:    3,
				SecretsMissing:  0,
				SecretsOptional: 0,
				HealthStatus:    "healthy",
				LastChecked:     time.Now(),
			},
			{
				ResourceName:    "postgres",
				SecretsTotal:    5,
				SecretsFound:    4,
				SecretsMissing:  1,
				SecretsOptional: 0,
				HealthStatus:    "degraded",
				LastChecked:     time.Now(),
			},
			{
				ResourceName:    "openrouter",
				SecretsTotal:    2,
				SecretsFound:    1,
				SecretsMissing:  1,
				SecretsOptional: 0,
				HealthStatus:    "degraded",
				LastChecked:     time.Now(),
			},
			{
				ResourceName:    "redis",
				SecretsTotal:    2,
				SecretsFound:    2,
				SecretsMissing:  0,
				SecretsOptional: 0,
				HealthStatus:    "healthy",
				LastChecked:     time.Now(),
			},
			{
				ResourceName:    "n8n",
				SecretsTotal:    4,
				SecretsFound:    4,
				SecretsMissing:  0,
				SecretsOptional: 0,
				HealthStatus:    "healthy",
				LastChecked:     time.Now(),
			},
		},
		LastUpdated: time.Now(),
	}
}
