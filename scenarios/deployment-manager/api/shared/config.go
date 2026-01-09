package shared

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/discovery"
)

// ConfigResolver defines the interface for resolving configuration values.
// This seam allows tests to substitute configuration resolution with mocks.
type ConfigResolver interface {
	// ResolveAnalyzerURL returns the URL for the scenario-dependency-analyzer service.
	ResolveAnalyzerURL() (string, error)
	// ResolveSecretsManagerURL returns the URL for the secrets-manager service.
	ResolveSecretsManagerURL() (string, error)
	// ResolveDesktopPackagerURL returns the URL for the scenario-to-desktop service.
	ResolveDesktopPackagerURL() (string, error)
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
// It checks SCENARIO_DEPENDENCY_ANALYZER_URL first for testing, then uses discovery.
func (r *EnvConfigResolver) ResolveAnalyzerURL() (string, error) {
	if url := strings.TrimSpace(os.Getenv("SCENARIO_DEPENDENCY_ANALYZER_URL")); url != "" {
		return strings.TrimRight(url, "/"), nil
	}
	return discovery.ResolveScenarioURLDefault(context.Background(), "scenario-dependency-analyzer")
}

// ResolveSecretsManagerURL returns the URL for the secrets-manager service.
// It checks SECRETS_MANAGER_URL first for testing, then uses discovery.
func (r *EnvConfigResolver) ResolveSecretsManagerURL() (string, error) {
	if url := strings.TrimSpace(os.Getenv("SECRETS_MANAGER_URL")); url != "" {
		return strings.TrimRight(url, "/"), nil
	}
	return discovery.ResolveScenarioURLDefault(context.Background(), "secrets-manager")
}

// ResolveDesktopPackagerURL returns the URL for the scenario-to-desktop service.
// It checks SCENARIO_TO_DESKTOP_URL first for testing, then uses discovery.
func (r *EnvConfigResolver) ResolveDesktopPackagerURL() (string, error) {
	if url := strings.TrimSpace(os.Getenv("SCENARIO_TO_DESKTOP_URL")); url != "" {
		return strings.TrimRight(url, "/"), nil
	}
	return discovery.ResolveScenarioURLDefault(context.Background(), "scenario-to-desktop")
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
