package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

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
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		vrooliRoot = filepath.Join(home, "Vrooli")
	}

	resourcesPath := filepath.Join(vrooliRoot, "resources")
	log.Printf("🔍 Scanning resources directory: %s", resourcesPath)
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
				log.Printf("  📦 Found resource with secrets: %s", resourceName)
				resourcesWithSecrets = append(resourcesWithSecrets, resourceName)
			}
		}
		return nil
	})

	log.Printf("✅ Found %d resources with secrets: %v", len(resourcesWithSecrets), resourcesWithSecrets)
	return resourcesWithSecrets, err
}

// loadResourceSecrets loads secrets configuration for a specific resource
func loadResourceSecrets(resourceName string) (*ResourceSecretsConfig, error) {
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		vrooliRoot = filepath.Join(home, "Vrooli")
	}

	secretsPath := filepath.Join(vrooliRoot, "resources", resourceName, "config", "secrets.yaml")

	data, err := ioutil.ReadFile(secretsPath)
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

	log.Printf("🔍 checkVaultForSecret: resource=%s, secret=%s, envName=%s", resourceName, secretName, envName)

	// Use getVaultSecret which already implements resource-vault CLI access
	if value, err := getVaultSecret(envName); err == nil && value != "" {
		log.Printf("✅ Found secret %s in vault", envName)
		return true
	}

	// Fall back to environment variables
	if envValue := os.Getenv(envName); envValue != "" {
		log.Printf("✅ Found secret %s in environment", envName)
		return true
	}

	// Fall back to local secrets file (~/.vrooli/secrets.json)
	if value, err := loadLocalSecret(envName); err == nil && value != "" {
		log.Printf("✅ Found secret %s in local secrets store", envName)
		return true
	}

	log.Printf("❌ Secret %s not found in vault or environment", envName)
	return false
}

func loadLocalSecret(key string) (string, error) {
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		vrooliRoot = filepath.Join(home, "Vrooli")
	}

	secretsPath := filepath.Join(vrooliRoot, ".vrooli", "secrets.json")
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
			log.Printf("  ❌ Error loading %s secrets: %v", resourceName, err)
			continue // Skip resources we can't read
		}
		var categoryNames []string
		for category := range config.Secrets {
			categoryNames = append(categoryNames, category)
		}
		sort.Strings(categoryNames)
		log.Printf("  📋 Processing %s: categories=%v", resourceName, categoryNames)

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

		// Determine health status
		if status.SecretsMissing == 0 {
			status.HealthStatus = "healthy"
			configuredCount++
		} else if status.SecretsMissing <= 2 {
			status.HealthStatus = "degraded"
		} else {
			status.HealthStatus = "critical"
		}

		resourceStatuses = append(resourceStatuses, status)
	}

	return &VaultSecretsStatus{
		TotalResources:      len(resourcesWithSecrets),
		ConfiguredResources: configuredCount,
		MissingSecrets:      allMissingSecrets,
		ResourceStatuses:    resourceStatuses,
		LastUpdated:         time.Now(),
	}, nil
}

// getMockVaultStatus returns mock data for testing when vault is unavailable
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
