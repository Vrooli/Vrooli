package main

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	healthyMissingSecretsThreshold  = 0
	degradedMissingSecretsThreshold = 2
)

// determineResourceHealthStatus classifies authoritative Vault state. It never
// infers that a secret exists from an environment variable or local file.
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

func isResourceFullyConfigured(missingRequiredSecrets int) bool {
	return missingRequiredSecrets <= healthyMissingSecretsThreshold
}

// ResourceSecretsConfig is declarative requirement metadata, not an alternate
// secret source or a substitute for the managed Vault provider.
type ResourceSecretsConfig struct {
	Version        string                        `yaml:"version"`
	Resource       string                        `yaml:"resource"`
	Description    string                        `yaml:"description"`
	Secrets        map[string][]SecretDefinition `yaml:"secrets"`
	Initialization *InitializationConfig         `yaml:"initialization"`
}

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

type InitializationConfig struct {
	PromptUser []struct {
		Name       string `yaml:"name"`
		Prompt     string `yaml:"prompt"`
		Validation string `yaml:"validation"`
		Optional   bool   `yaml:"optional"`
	} `yaml:"prompt_user"`
}

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
	return strings.ReplaceAll(name, " ", "_")
}

func extractPromptInfo(init *InitializationConfig, secret SecretDefinition) (string, string, string) {
	if init == nil {
		return "", "", ""
	}
	secretNameNorm, defaultEnvNorm := normalizeName(secret.Name), normalizeName(secret.DefaultEnv)
	for _, prompt := range init.PromptUser {
		promptNorm := normalizeName(prompt.Name)
		if promptNorm != secretNameNorm && (defaultEnvNorm == "" || promptNorm != defaultEnvNorm) {
			continue
		}
		acquisitionURL := ""
		if urlStart := strings.Index(prompt.Prompt, "https://"); urlStart != -1 {
			rest := prompt.Prompt[urlStart:]
			if urlEnd := strings.IndexAny(rest, " \t\n)"); urlEnd == -1 {
				acquisitionURL = rest
			} else {
				acquisitionURL = rest[:urlEnd]
			}
		}
		return prompt.Prompt, prompt.Validation, acquisitionURL
	}
	return "", "", ""
}

// loadResourceSecrets reads declared secret requirements for a single resource.
func loadResourceSecrets(resourceName string) (*ResourceSecretsConfig, error) {
	path := filepath.Join(resolveTopLevelDir("resources"), resourceName, "config", "secrets.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config ResourceSecretsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
