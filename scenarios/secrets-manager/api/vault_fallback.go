package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// ResourceSecretsConfig represents a resource's secrets.yaml file
type ResourceSecretsConfig struct {
	Version        string `yaml:"version"`
	Resource       string `yaml:"resource"`
	Description    string `yaml:"description"`
	Secrets        struct {
		APIKeys []SecretDefinition `yaml:"api_keys"`
		Endpoints []SecretDefinition `yaml:"endpoints"`
		Quotas []SecretDefinition `yaml:"quotas"`
	} `yaml:"secrets"`
	Initialization *InitializationConfig `yaml:"initialization"`
}

// SecretDefinition represents a single secret configuration
type SecretDefinition struct {
	Name           string `yaml:"name"`
	Description    string `yaml:"description"`
	Required       bool   `yaml:"required"`
	DefaultEnv     string `yaml:"default_env"`
	Example        string `yaml:"example"`
	Documentation  string `yaml:"documentation"`
	Path           string `yaml:"path"`
	Format         string `yaml:"format"`
	Validation     *ValidationConfig `yaml:"validation"`
	Fallback       string `yaml:"fallback"`
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
	
	log.Printf("❌ Secret %s not found in vault or environment", envName)
	return false
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
		log.Printf("  📋 Processing %s: %d api_keys, %d endpoints, %d quotas", 
			resourceName, len(config.Secrets.APIKeys), len(config.Secrets.Endpoints), len(config.Secrets.Quotas))

		// Count all secrets across all categories
		totalSecrets := len(config.Secrets.APIKeys) + len(config.Secrets.Endpoints) + len(config.Secrets.Quotas)
		
		status := VaultResourceStatus{
			ResourceName: resourceName,
			SecretsTotal: totalSecrets,
			LastChecked:  time.Now(),
			AllSecrets:   []VaultSecret{},
		}

		// Check API keys
		for _, secret := range config.Secrets.APIKeys {
			isConfigured := checkVaultForSecret("", secret.DefaultEnv) // Use direct env name, not prefixed
			
			// Extract guidance information
			vaultSecret := VaultSecret{
				Name:               secret.DefaultEnv,
				Description:        secret.Description,
				Required:           secret.Required,
				Configured:         isConfigured,
				SecretType:         "api_key",
				DocumentationURL:   secret.Documentation,
				Example:            secret.Example,
			}
			
			// Extract setup instructions and acquisition URL from initialization config
			if config.Initialization != nil {
				for _, prompt := range config.Initialization.PromptUser {
					// Try to match by converting secret name to match initialization format
					// e.g., "openrouter_api_key" (yaml) vs "OPENROUTER_API_KEY" (env var)
					promptNameUpper := strings.ToUpper(strings.ReplaceAll(prompt.Name, "-", "_"))
					secretNameNormalized := strings.ToUpper(strings.ReplaceAll(secret.Name, "-", "_"))
					
					if prompt.Name == secret.Name || promptNameUpper == secretNameNormalized {
						vaultSecret.SetupInstructions = prompt.Prompt
						vaultSecret.ValidationHint = prompt.Validation
						
						// Extract URL from prompt text (look for https:// patterns)
						if urlStart := strings.Index(prompt.Prompt, "https://"); urlStart != -1 {
							urlEnd := strings.IndexAny(prompt.Prompt[urlStart:], " \t\n)")
							if urlEnd == -1 {
								vaultSecret.AcquisitionURL = prompt.Prompt[urlStart:]
							} else {
								vaultSecret.AcquisitionURL = prompt.Prompt[urlStart:urlStart+urlEnd]
							}
						}
						break
					}
				}
			}
			
			// Add to AllSecrets list
			status.AllSecrets = append(status.AllSecrets, vaultSecret)
			
			if isConfigured {
				status.SecretsFound++
			} else if secret.Required {
				status.SecretsMissing++
				allMissingSecrets = append(allMissingSecrets, VaultMissingSecret{
					ResourceName: resourceName,
					SecretName:   secret.DefaultEnv, // Use env var name
					Required:     secret.Required,
					Description:  secret.Description,
				})
			} else {
				status.SecretsOptional++
			}
		}
		
		// Check endpoints
		for _, secret := range config.Secrets.Endpoints {
			isConfigured := checkVaultForSecret("", secret.DefaultEnv) // Use direct env name, not prefixed
			
			// Extract guidance information
			vaultSecret := VaultSecret{
				Name:               secret.DefaultEnv,
				Description:        secret.Description,
				Required:           secret.Required,
				Configured:         isConfigured,
				SecretType:         "endpoint",
				DocumentationURL:   secret.Documentation,
				Example:            secret.Example,
			}
			
			// Extract setup instructions and acquisition URL from initialization config
			if config.Initialization != nil {
				for _, prompt := range config.Initialization.PromptUser {
					// Try to match by converting secret name to match initialization format
					// e.g., "openrouter_api_key" (yaml) vs "OPENROUTER_API_KEY" (env var)
					promptNameUpper := strings.ToUpper(strings.ReplaceAll(prompt.Name, "-", "_"))
					secretNameNormalized := strings.ToUpper(strings.ReplaceAll(secret.Name, "-", "_"))
					
					if prompt.Name == secret.Name || promptNameUpper == secretNameNormalized {
						vaultSecret.SetupInstructions = prompt.Prompt
						vaultSecret.ValidationHint = prompt.Validation
						
						// Extract URL from prompt text (look for https:// patterns)
						if urlStart := strings.Index(prompt.Prompt, "https://"); urlStart != -1 {
							urlEnd := strings.IndexAny(prompt.Prompt[urlStart:], " \t\n)")
							if urlEnd == -1 {
								vaultSecret.AcquisitionURL = prompt.Prompt[urlStart:]
							} else {
								vaultSecret.AcquisitionURL = prompt.Prompt[urlStart:urlStart+urlEnd]
							}
						}
						break
					}
				}
			}
			
			// Add to AllSecrets list
			status.AllSecrets = append(status.AllSecrets, vaultSecret)
			
			if isConfigured {
				status.SecretsFound++
			} else if secret.Required {
				status.SecretsMissing++
				allMissingSecrets = append(allMissingSecrets, VaultMissingSecret{
					ResourceName: resourceName,
					SecretName:   secret.DefaultEnv,
					Required:     secret.Required,
					Description:  secret.Description,
				})
			} else {
				status.SecretsOptional++
			}
		}
		
		// Check quotas
		for _, secret := range config.Secrets.Quotas {
			isConfigured := checkVaultForSecret("", secret.DefaultEnv) // Use direct env name, not prefixed
			
			// Extract guidance information
			vaultSecret := VaultSecret{
				Name:               secret.DefaultEnv,
				Description:        secret.Description,
				Required:           secret.Required,
				Configured:         isConfigured,
				SecretType:         "quota",
				DocumentationURL:   secret.Documentation,
				Example:            secret.Example,
			}
			
			// Extract setup instructions and acquisition URL from initialization config
			if config.Initialization != nil {
				for _, prompt := range config.Initialization.PromptUser {
					// Try to match by converting secret name to match initialization format
					// e.g., "openrouter_api_key" (yaml) vs "OPENROUTER_API_KEY" (env var)
					promptNameUpper := strings.ToUpper(strings.ReplaceAll(prompt.Name, "-", "_"))
					secretNameNormalized := strings.ToUpper(strings.ReplaceAll(secret.Name, "-", "_"))
					
					if prompt.Name == secret.Name || promptNameUpper == secretNameNormalized {
						vaultSecret.SetupInstructions = prompt.Prompt
						vaultSecret.ValidationHint = prompt.Validation
						
						// Extract URL from prompt text (look for https:// patterns)
						if urlStart := strings.Index(prompt.Prompt, "https://"); urlStart != -1 {
							urlEnd := strings.IndexAny(prompt.Prompt[urlStart:], " \t\n)")
							if urlEnd == -1 {
								vaultSecret.AcquisitionURL = prompt.Prompt[urlStart:]
							} else {
								vaultSecret.AcquisitionURL = prompt.Prompt[urlStart:urlStart+urlEnd]
							}
						}
						break
					}
				}
			}
			
			// Add to AllSecrets list
			status.AllSecrets = append(status.AllSecrets, vaultSecret)
			
			if isConfigured {
				status.SecretsFound++
			} else if secret.Required {
				status.SecretsMissing++
				allMissingSecrets = append(allMissingSecrets, VaultMissingSecret{
					ResourceName: resourceName,
					SecretName:   secret.DefaultEnv,
					Required:     secret.Required,
					Description:  secret.Description,
				})
			} else {
				status.SecretsOptional++
			}
		}

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