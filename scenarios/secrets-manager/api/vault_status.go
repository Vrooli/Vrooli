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
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Vault integration - uses resource-vault CLI commands to get secrets status
func getVaultSecretsStatus(resourceFilter string) (*VaultSecretsStatus, error) {
	// First try using resource-vault CLI directly
	status, err := getVaultSecretsStatusFromCLI(resourceFilter)
	if err != nil {
		logger.Info("resource-vault CLI failed: %v, using fallback implementation", err)
		// Fallback to direct scanning
		return getVaultSecretsStatusFallback(resourceFilter)
	}
	return status, nil
}

// getVaultSecretsStatusFromCLI uses resource-vault CLI commands
func getVaultSecretsStatusFromCLI(resourceFilter string) (*VaultSecretsStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use resource-vault secrets validate to get status
	var cmd *exec.Cmd
	if resourceFilter != "" {
		cmd = exec.CommandContext(ctx, "resource-vault", "secrets", "check", resourceFilter)
	} else {
		cmd = exec.CommandContext(ctx, "resource-vault", "secrets", "validate")
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("resource-vault command failed: %w", err)
	}

	// Parse the output from resource-vault
	status := parseVaultCLIOutput(string(output), resourceFilter)
	status.LastUpdated = time.Now()

	return status, nil
}

// parseVaultCLIOutput parses resource-vault CLI output into structured data
func parseVaultCLIOutput(output, resourceFilter string) *VaultSecretsStatus {
	status := &VaultSecretsStatus{
		MissingSecrets:   []VaultMissingSecret{},
		ResourceStatuses: []VaultResourceStatus{},
	}

	lines := strings.Split(output, "\n")
	var currentResource string
	resourceCount := 0
	configuredCount := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse test format: "Resource: postgres" or "Status: Configured"
		if strings.HasPrefix(line, "Resource:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				currentResource = strings.TrimSpace(parts[1])
				resourceCount++
			}
		}

		if strings.HasPrefix(line, "Status:") && currentResource != "" {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				statusVal := strings.TrimSpace(parts[1])
				if strings.EqualFold(statusVal, "Configured") {
					configuredCount++
				}
			}
		}

		// Parse test format: "- DATABASE_URL (configured)" or "- OPENAI_API_KEY (required)"
		if strings.HasPrefix(line, "-") && currentResource != "" {
			// Check if it's a missing secret (contains "required" or "optional" but not "configured")
			if strings.Contains(line, "MISSING") || (strings.Contains(line, "(required)") || strings.Contains(line, "(optional)")) && !strings.Contains(line, "(configured)") {
				parts := strings.Split(line, "(")
				if len(parts) >= 1 {
					secretName := strings.TrimSpace(strings.TrimPrefix(parts[0], "-"))
					missing := VaultMissingSecret{
						ResourceName: currentResource,
						SecretName:   secretName,
						SecretPath:   fmt.Sprintf("secret/%s/%s", currentResource, secretName),
						Required:     strings.Contains(line, "(required)"),
						Description:  fmt.Sprintf("Missing required secret for %s", currentResource),
					}
					status.MissingSecrets = append(status.MissingSecrets, missing)
				}
			}
		}

		// Parse production format: "✓ postgres: 3 secrets defined"
		if strings.Contains(line, "✓") && strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) >= 2 {
				resourceName := strings.TrimSpace(strings.Replace(parts[0], "✓", "", 1))
				currentResource = resourceName
				resourceCount++

				// Check if all secrets are configured (no missing indicators)
				if !strings.Contains(parts[1], "MISSING") {
					configuredCount++
				}
			}
		}

		// Look for missing secrets like "✗ POSTGRES_PASSWORD: MISSING"
		if strings.Contains(line, "✗") && strings.Contains(line, "MISSING") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 && currentResource != "" {
				secretName := strings.TrimSpace(strings.Replace(parts[0], "✗", "", 1))
				missing := VaultMissingSecret{
					ResourceName: currentResource,
					SecretName:   secretName,
					SecretPath:   fmt.Sprintf("secret/%s/%s", currentResource, secretName),
					Required:     true,
					Description:  fmt.Sprintf("Missing required secret for %s", currentResource),
				}
				status.MissingSecrets = append(status.MissingSecrets, missing)
			}
		}

		// Parse test format: "Total Resources: 10" and "Configured: 7"
		if strings.HasPrefix(line, "Total Resources:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				if count, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					status.TotalResources = count
				}
			}
		}
		if strings.HasPrefix(line, "Configured:") && !strings.Contains(line, "Fully") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				if count, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					status.ConfiguredResources = count
				}
			}
		}

		// Look for "Fully configured: X" summary
		if strings.Contains(line, "Fully configured:") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "configured:" && i+1 < len(fields) {
					if count, err := strconv.Atoi(fields[i+1]); err == nil {
						configuredCount = count
					}
				}
			}
		}
	}

	// Only override totals if not already set from "Total Resources:" line
	if status.TotalResources == 0 {
		status.TotalResources = resourceCount
	}
	if status.ConfiguredResources == 0 {
		status.ConfiguredResources = configuredCount
	}

	return status
}

// Parse vault scan output to extract resource names
func parseVaultScanOutput(output string) []string {
	var resources []string
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse test format: "Found: postgres"
		if strings.HasPrefix(line, "Found:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				resourceName := strings.TrimSpace(parts[1])
				if resourceName != "" {
					resources = append(resources, resourceName)
				}
			}
		}

		// Parse production format: "✓ openrouter: 3 secrets defined"
		if strings.Contains(line, "✓") && strings.Contains(line, ":") && strings.Contains(line, "secrets defined") {
			parts := strings.Split(line, ":")
			if len(parts) > 0 {
				resourceName := strings.TrimSpace(strings.Replace(parts[0], "✓", "", -1))
				if resourceName != "" {
					resources = append(resources, resourceName)
				}
			}
		}
	}

	return resources
}

// Parse vault validation output
func parseVaultValidationOutput(output string) VaultValidationSummary {
	var summary VaultValidationSummary
	lines := strings.Split(output, "\n")

	configuredCount := 0
	var missingSecrets []VaultMissingSecret

	for _, line := range lines {
		// Look for "Fully configured: X"
		if strings.Contains(line, "Fully configured:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				if count := parts[2]; count != "" {
					fmt.Sscanf(count, "%d", &configuredCount)
				}
			}
		}

		// Look for missing secret indicators (this would need more sophisticated parsing)
		if strings.Contains(line, "✗") && strings.Contains(line, "MISSING") {
			// Parse missing secret details - simplified for now
			missingSecrets = append(missingSecrets, VaultMissingSecret{
				ResourceName: "unknown", // Would need better parsing
				SecretName:   "unknown",
				Required:     true,
				Description:  strings.TrimSpace(line),
			})
		}
	}

	summary.ConfiguredCount = configuredCount
	summary.MissingSecrets = missingSecrets

	return summary
}

// Parse vault resource check output
func parseVaultResourceCheck(resourceName, output string) VaultResourceStatus {
	status := VaultResourceStatus{
		ResourceName: resourceName,
		LastChecked:  time.Now(),
	}

	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.Contains(line, "Found:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				fmt.Sscanf(parts[1], "%d", &status.SecretsFound)
			}
		}
		if strings.Contains(line, "Missing (required):") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				fmt.Sscanf(parts[2], "%d", &status.SecretsMissing)
			}
		}
		if strings.Contains(line, "Not set (optional):") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				fmt.Sscanf(parts[3], "%d", &status.SecretsOptional)
			}
		}
	}

	status.SecretsTotal = status.SecretsFound + status.SecretsMissing + status.SecretsOptional

	// Determine health status
	if status.SecretsMissing == 0 {
		status.HealthStatus = "healthy"
	} else if status.SecretsMissing <= 2 {
		status.HealthStatus = "degraded"
	} else {
		status.HealthStatus = "critical"
	}

	return status
}

func scanResourceDirectory(resourceName, resourceDir string) ([]ResourceSecret, error) {
	var secrets []ResourceSecret

	// Patterns to look for environment variables and credentials
	envVarPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\$\{([A-Z_]+[A-Z0-9_]*)\}`),           // ${VAR_NAME}
		regexp.MustCompile(`\$([A-Z_]+[A-Z0-9_]*)`),               // $VAR_NAME
		regexp.MustCompile(`([A-Z_]+[A-Z0-9_]*)=`),                // VAR_NAME=
		regexp.MustCompile(`env\.([A-Z_]+[A-Z0-9_]*)`),            // env.VAR_NAME
		regexp.MustCompile(`getenv\("([A-Z_]+[A-Z0-9_]*)"\)`),     // getenv("VAR_NAME")
		regexp.MustCompile(`os\.Getenv\("([A-Z_]+[A-Z0-9_]*)"\)`), // os.Getenv("VAR_NAME")
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
					if !foundVars[varName] && isLikelySecret(varName) {
						foundVars[varName] = true

						secret := ResourceSecret{
							ID:                uuid.New().String(),
							ResourceName:      resourceName,
							SecretKey:         varName,
							SecretType:        classifySecretType(varName),
							Required:          isLikelyRequired(varName),
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
	textExtensions := []string{".sh", ".bash", ".yml", ".yaml", ".json", ".env", ".conf", ".config", ".md", ".txt", ".go", ".js", ".py", ".dockerfile", ".sql"}

	for _, textExt := range textExtensions {
		if ext == textExt {
			return true
		}
	}

	return false
}

func isLikelySecret(varName string) bool {
	secretKeywords := []string{
		"PASSWORD", "PASS", "PWD", "SECRET", "KEY", "TOKEN", "AUTH", "API", "CREDENTIAL", "CERT", "TLS", "SSL", "PRIVATE",
	}

	upperVar := strings.ToUpper(varName)
	for _, keyword := range secretKeywords {
		if strings.Contains(upperVar, keyword) {
			return true
		}
	}

	// Also include common configuration variables that might be sensitive
	configKeywords := []string{"HOST", "PORT", "URL", "ADDR", "DATABASE", "DB", "USER", "NAMESPACE"}
	for _, keyword := range configKeywords {
		if strings.Contains(upperVar, keyword) {
			return true
		}
	}

	return false
}

func classifySecretType(varName string) string {
	upperVar := strings.ToUpper(varName)

	if strings.Contains(upperVar, "PASSWORD") || strings.Contains(upperVar, "PASS") || strings.Contains(upperVar, "PWD") {
		return "password"
	}
	if strings.Contains(upperVar, "TOKEN") {
		return "token"
	}
	if strings.Contains(upperVar, "KEY") && (strings.Contains(upperVar, "API") || strings.Contains(upperVar, "ACCESS")) {
		return "api_key"
	}
	if strings.Contains(upperVar, "SECRET") || strings.Contains(upperVar, "CREDENTIAL") {
		return "credential"
	}
	if strings.Contains(upperVar, "CERT") || strings.Contains(upperVar, "TLS") || strings.Contains(upperVar, "SSL") {
		return "certificate"
	}

	return "env_var"
}

func isLikelyRequired(varName string) bool {
	requiredKeywords := []string{"PASSWORD", "SECRET", "TOKEN", "KEY", "DATABASE", "DB", "HOST"}
	upperVar := strings.ToUpper(varName)

	for _, keyword := range requiredKeywords {
		if strings.Contains(upperVar, keyword) {
			return true
		}
	}

	return false
}

func getVaultSecret(key string) (string, error) {
	// Use resource-vault CLI to get secret value
	cmd := exec.Command("resource-vault", "secrets", "get", key)

	// Set timeout for vault command
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd = exec.CommandContext(ctx, "resource-vault", "secrets", "get", key)

	output, err := cmd.Output()
	if err != nil {
		// Check if it's a timeout or command not found
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("vault command timeout")
		}

		// If exit error, likely secret not found or vault unavailable
		if exitErr, ok := err.(*exec.ExitError); ok {
			// resource-vault typically returns exit code 1 for not found
			if exitErr.ExitCode() == 1 {
				return "", fmt.Errorf("secret not found in vault")
			}
			return "", fmt.Errorf("vault command failed: %v", err)
		}

		return "", fmt.Errorf("failed to execute resource-vault command: %v", err)
	}

	// Clean up the output (remove trailing whitespace/newlines)
	secretValue := strings.TrimSpace(string(output))

	if secretValue == "" {
		return "", fmt.Errorf("empty secret value returned from vault")
	}

	return secretValue, nil
}

func getLocalSecretsPath() (string, error) {
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		vrooliRoot = filepath.Join(home, "Vrooli")
	}

	secretsDir := filepath.Join(vrooliRoot, ".vrooli")
	if err := os.MkdirAll(secretsDir, 0o755); err != nil {
		return "", err
	}

	return filepath.Join(secretsDir, "secrets.json"), nil
}

func loadLocalSecretsFile() (map[string]interface{}, error) {
	path, err := getLocalSecretsPath()
	if err != nil {
		return nil, err
	}

	store := map[string]interface{}{}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			store["_metadata"] = map[string]interface{}{
				"environment":  "development",
				"last_updated": time.Now().Format(time.RFC3339),
			}
			return store, nil
		}
		return nil, err
	}

	if len(data) == 0 {
		store["_metadata"] = map[string]interface{}{
			"environment":  "development",
			"last_updated": time.Now().Format(time.RFC3339),
		}
		return store, nil
	}

	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}

	return store, nil
}

func saveSecretsToLocalStore(secrets map[string]string) (int, error) {
	if len(secrets) == 0 {
		return 0, nil
	}

	store, err := loadLocalSecretsFile()
	if err != nil {
		return 0, err
	}

	for key, value := range secrets {
		store[key] = value
	}

	meta, ok := store["_metadata"].(map[string]interface{})
	if !ok || meta == nil {
		meta = map[string]interface{}{}
	}
	if _, exists := meta["environment"]; !exists {
		meta["environment"] = "development"
	}
	meta["last_updated"] = time.Now().Format(time.RFC3339)
	store["_metadata"] = meta

	path, err := getLocalSecretsPath()
	if err != nil {
		return 0, err
	}

	encoded, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return 0, err
	}

	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		return 0, err
	}

	return len(secrets), nil
}
