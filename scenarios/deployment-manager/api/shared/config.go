package shared

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ConfigResolver defines the interface for resolving configuration values.
// This seam allows tests to substitute configuration resolution with mocks.
type ConfigResolver interface {
	// ResolveAnalyzerURL returns the URL for the scenario-dependency-analyzer service.
	ResolveAnalyzerURL() (string, error)
	// ResolveSecretsManagerURL returns the URL for the secrets-manager service.
	ResolveSecretsManagerURL() (string, error)
	// ResolveTelemetryDir returns the directory for storing telemetry files.
	ResolveTelemetryDir() string
}

// EnvConfigResolver resolves configuration from environment variables.
type EnvConfigResolver struct{}

// NewEnvConfigResolver creates a new environment-based configuration resolver.
func NewEnvConfigResolver() *EnvConfigResolver {
	return &EnvConfigResolver{}
}

// ResolveAnalyzerURL returns the URL for the scenario-dependency-analyzer service.
// It checks SCENARIO_DEPENDENCY_ANALYZER_API_PORT first, then falls back to
// dynamic lookup via the vrooli CLI.
func (r *EnvConfigResolver) ResolveAnalyzerURL() (string, error) {
	if port := strings.TrimSpace(os.Getenv("SCENARIO_DEPENDENCY_ANALYZER_API_PORT")); port != "" {
		return fmt.Sprintf("http://127.0.0.1:%s", port), nil
	}

	cmd := exec.Command("vrooli", "scenario", "port", "scenario-dependency-analyzer", "API_PORT")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("SCENARIO_DEPENDENCY_ANALYZER_API_PORT not set and dynamic lookup failed: %w", err)
	}
	port := strings.TrimSpace(string(output))
	if port == "" {
		return "", fmt.Errorf("SCENARIO_DEPENDENCY_ANALYZER_API_PORT not set and dynamic lookup returned empty output")
	}
	return fmt.Sprintf("http://127.0.0.1:%s", port), nil
}

// ResolveSecretsManagerURL returns the URL for the secrets-manager service.
// It checks multiple environment variables in order of preference.
func (r *EnvConfigResolver) ResolveSecretsManagerURL() (string, error) {
	if base := os.Getenv("SECRETS_MANAGER_URL"); base != "" {
		return strings.TrimSuffix(base, "/"), nil
	}
	if base := os.Getenv("SECRETS_MANAGER_API_URL"); base != "" {
		return strings.TrimSuffix(base, "/"), nil
	}
	if port := os.Getenv("SECRETS_MANAGER_API_PORT"); port != "" {
		return fmt.Sprintf("http://127.0.0.1:%s", port), nil
	}
	return "", fmt.Errorf("SECRETS_MANAGER_URL, SECRETS_MANAGER_API_URL, or SECRETS_MANAGER_API_PORT must be set")
}

// ResolveTelemetryDir returns the directory for storing telemetry files.
func (r *EnvConfigResolver) ResolveTelemetryDir() string {
	if override := strings.TrimSpace(os.Getenv("DEPLOYMENT_MANAGER_TELEMETRY_DIR")); override != "" {
		return override
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".", ".vrooli", "deployment", "telemetry")
	}
	return filepath.Join(home, ".vrooli", "deployment", "telemetry")
}

// DefaultConfigResolver is the default environment-based config resolver.
var DefaultConfigResolver ConfigResolver = NewEnvConfigResolver()

// GetConfigResolver returns the current config resolver.
// In production this returns the environment-based resolver.
// Tests can override this by setting a custom resolver.
func GetConfigResolver() ConfigResolver {
	return DefaultConfigResolver
}

// SetConfigResolver allows overriding the config resolver (for testing).
func SetConfigResolver(cr ConfigResolver) {
	DefaultConfigResolver = cr
}
