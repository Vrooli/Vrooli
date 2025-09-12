package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// ResourceSecretsConfig represents a resource's secrets.yaml file
type ResourceSecretsConfig struct {
	Version string `yaml:"version"`
	Secrets []struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		Required    bool   `yaml:"required"`
		Type        string `yaml:"type"`
		Default     string `yaml:"default"`
		Example     string `yaml:"example"`
	} `yaml:"secrets"`
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
				resourcesWithSecrets = append(resourcesWithSecrets, resourceName)
			}
		}
		return nil
	})

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
	// For now, check environment variables as a fallback
	// In production, this would actually query vault
	envName := fmt.Sprintf("%s_%s", strings.ToUpper(resourceName), strings.ToUpper(secretName))
	envName = strings.ReplaceAll(envName, "-", "_")
	
	return os.Getenv(envName) != ""
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
			continue // Skip resources we can't read
		}

		status := VaultResourceStatus{
			ResourceName: resourceName,
			SecretsTotal: len(config.Secrets),
			LastChecked:  time.Now(),
		}

		// Check each secret
		for _, secret := range config.Secrets {
			if checkVaultForSecret(resourceName, secret.Name) {
				status.SecretsFound++
			} else if secret.Required {
				status.SecretsMissing++
				allMissingSecrets = append(allMissingSecrets, VaultMissingSecret{
					ResourceName: resourceName,
					SecretName:   secret.Name,
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