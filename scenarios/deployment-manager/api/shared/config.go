package shared

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/storage"
)

// ValidateServiceURL accepts only an absolute HTTP(S) service URL without
// embedded credentials. Scenario-to-scenario calls use discovery or an
// operator-provided endpoint, so every override must pass this boundary before
// it can become an outbound request target.
func ValidateServiceURL(raw string) (string, error) {
	value := strings.TrimRight(strings.TrimSpace(raw), "/")
	u, err := url.Parse(value)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("service URL must be an absolute HTTP(S) URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("service URL scheme %q is not allowed", u.Scheme)
	}
	if u.User != nil {
		return "", fmt.Errorf("service URL must not contain embedded credentials")
	}
	return value, nil
}

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
	ResolveTelemetryDir() (string, error)
}

// EnvConfigResolver resolves scenario services through discovery and local
// operator settings through their owning mechanisms.
type EnvConfigResolver struct{}

// NewEnvConfigResolver creates a new environment-based configuration resolver.
func NewEnvConfigResolver() *EnvConfigResolver {
	return &EnvConfigResolver{}
}

// ResolveAnalyzerURL returns the URL for the scenario-dependency-analyzer service.
func (r *EnvConfigResolver) ResolveAnalyzerURL() (string, error) {
	if url := strings.TrimSpace(os.Getenv("SCENARIO_DEPENDENCY_ANALYZER_URL")); url != "" {
		return ValidateServiceURL(url)
	}
	return discovery.ResolveScenarioURLDefault(context.Background(), "scenario-dependency-analyzer")
}

// ResolveSecretsManagerURL returns the URL for the secrets-manager service.
// It checks SECRETS_MANAGER_URL first for testing, then uses discovery.
func (r *EnvConfigResolver) ResolveSecretsManagerURL() (string, error) {
	return discovery.ResolveScenarioURLDefault(context.Background(), "secrets-manager")
}

// ResolveDesktopPackagerURL returns the URL for the scenario-to-desktop service.
// It checks SCENARIO_TO_DESKTOP_URL first for testing, then uses discovery.
func (r *EnvConfigResolver) ResolveDesktopPackagerURL() (string, error) {
	return discovery.ResolveScenarioURLDefault(context.Background(), "scenario-to-desktop")
}

// ResolveTelemetryDir returns the directory for storing telemetry files.
func (r *EnvConfigResolver) ResolveTelemetryDir() (string, error) {
	if override := strings.TrimSpace(os.Getenv("DEPLOYMENT_MANAGER_TELEMETRY_DIR")); override != "" {
		return override, nil
	}
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", err
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: "deployment-manager"},
		storage.ClassLogs,
		"telemetry",
	)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", err
	}
	return path, nil
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
