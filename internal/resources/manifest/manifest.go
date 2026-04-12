package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const SchemaPath = ".vrooli/schemas/resource.schema.json"

var AllowedDrivers = []string{
	"docker-service",
	"compose-service",
	"external-cli",
	"cloud-api",
	"desktop-app",
	"manual",
	"legacy-adapter",
}

var AllowedPortabilityTiers = []string{"full", "partial", "platform-specific"}
var AllowedPlatformSupportStates = []string{"supported", "partial", "unsupported"}

type ResourceManifest struct {
	Schema          string                       `json:"$schema,omitempty"`
	Name            string                       `json:"name"`
	DisplayName     string                       `json:"display_name,omitempty"`
	Description     string                       `json:"description,omitempty"`
	Template        string                       `json:"template,omitempty"`
	Driver          string                       `json:"driver"`
	ComposeFile     string                       `json:"compose_file,omitempty"`
	LegacyAdapter   ResourceLegacyAdapter        `json:"legacy_adapter,omitempty"`
	Binary          string                       `json:"binary,omitempty"`
	VersionArgs     []string                     `json:"version_args,omitempty"`
	Endpoint        string                       `json:"endpoint,omitempty"`
	Credentials     ResourceCredentials          `json:"credentials,omitempty"`
	PortabilityTier string                       `json:"portability_tier,omitempty"`
	Category        string                       `json:"category,omitempty"`
	Platforms       ResourcePlatforms            `json:"platforms,omitempty"`
	Dependencies    []string                     `json:"dependencies,omitempty"`
	Ports           []ResourcePort               `json:"ports,omitempty"`
	HealthChecks    []ResourceHealthCheck        `json:"health_checks,omitempty"`
	Install         ResourceInstall              `json:"install,omitempty"`
	Runtime         ResourceRuntime              `json:"runtime,omitempty"`
	Lifecycle       ResourceLifecycle            `json:"lifecycle,omitempty"`
	Capabilities    ResourceManifestCapabilities `json:"capabilities,omitempty"`
	TemplateVersion string                       `json:"template_version,omitempty"`
	HostTools       []hostreqspec.Declaration    `json:"hostTools,omitempty"`
	HostSafeguards  []hostreqspec.Declaration    `json:"hostSafeguards,omitempty"`
}

type ResourceLegacyAdapter struct {
	Owner            string `json:"owner,omitempty"`
	DecisionDeadline string `json:"decision_deadline,omitempty"`
	FinalDisposition string `json:"final_disposition,omitempty"`
	LegacyCLIPath    string `json:"legacy_cli_path,omitempty"`
	Notes            string `json:"notes,omitempty"`
}

type ResourcePlatforms struct {
	Linux   string `json:"linux,omitempty"`
	MacOS   string `json:"macos,omitempty"`
	Windows string `json:"windows,omitempty"`
}

type ResourcePort struct {
	Name      string `json:"name,omitempty"`
	Container int    `json:"container,omitempty"`
	Host      int    `json:"host,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
}

type ResourceHealthCheck struct {
	Type            string   `json:"type"`
	Target          string   `json:"target,omitempty"`
	Command         []string `json:"command,omitempty"`
	ExpectedStatus  []int    `json:"expected_status,omitempty"`
	IntervalSeconds int      `json:"interval_seconds,omitempty"`
	TimeoutSeconds  int      `json:"timeout_seconds,omitempty"`
}

type ResourceInstall struct {
	Command   []string            `json:"command,omitempty"`
	Platforms map[string][]string `json:"platforms,omitempty"`
}

type ResourceCredentials struct {
	Env       []string `json:"env,omitempty"`
	SecretRef string   `json:"secret_ref,omitempty"`
}

type ResourceRuntime struct {
	Image         string            `json:"image,omitempty"`
	ContainerName string            `json:"container_name,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	Volumes       []ResourceVolume  `json:"volumes,omitempty"`
	Command       []string          `json:"command,omitempty"`
	WorkingDir    string            `json:"working_dir,omitempty"`
}

type ResourceVolume struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type ResourceLifecycle struct {
	StartTimeoutSeconds int `json:"start_timeout_seconds,omitempty"`
	StopTimeoutSeconds  int `json:"stop_timeout_seconds,omitempty"`
	Retries             int `json:"retries,omitempty"`
}

type ResourceManifestCapabilities struct {
	SupportsLogs       bool `json:"supports_logs,omitempty"`
	SupportsContentOps bool `json:"supports_content_ops,omitempty"`
}

func DefaultPath(root, name string) string {
	return filepath.Join(root, "resources", name, "resource.json")
}

func Load(path string) (ResourceManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ResourceManifest{}, fmt.Errorf("read resource manifest %s: %w", path, err)
	}
	var manifest ResourceManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return ResourceManifest{}, fmt.Errorf("parse resource manifest %s: %w", path, err)
	}
	if err := Validate(manifest); err != nil {
		return ResourceManifest{}, fmt.Errorf("validate resource manifest %s: %w", path, err)
	}
	return manifest, nil
}

func Validate(manifest ResourceManifest) error {
	if strings.TrimSpace(manifest.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(manifest.Driver) == "" {
		return fmt.Errorf("driver is required")
	}
	if !slices.Contains(AllowedDrivers, strings.TrimSpace(manifest.Driver)) {
		return fmt.Errorf("driver %q is invalid", manifest.Driver)
	}
	if strings.TrimSpace(manifest.PortabilityTier) == "" {
		return fmt.Errorf("portability_tier is required")
	}
	if !slices.Contains(AllowedPortabilityTiers, strings.TrimSpace(manifest.PortabilityTier)) {
		return fmt.Errorf("portability_tier %q is invalid", manifest.PortabilityTier)
	}
	if err := validatePlatforms(manifest.Platforms); err != nil {
		return err
	}
	for _, port := range manifest.Ports {
		if port.Container < 0 || port.Host < 0 {
			return fmt.Errorf("ports must be non-negative")
		}
	}
	for _, check := range manifest.HealthChecks {
		if err := validateHealthCheck(check); err != nil {
			return err
		}
	}
	if err := hostreqspec.ValidateDeclarations(hostreqspec.KindTool, manifest.HostTools); err != nil {
		return err
	}
	if err := hostreqspec.ValidateDeclarations(hostreqspec.KindSafeguard, manifest.HostSafeguards); err != nil {
		return err
	}
	switch manifest.Driver {
	case "docker-service":
		if strings.TrimSpace(manifest.Runtime.Image) == "" {
			return fmt.Errorf("runtime.image is required for docker-service resources")
		}
	case "compose-service":
		if strings.TrimSpace(manifest.ComposeFile) == "" {
			return fmt.Errorf("compose_file is required for compose-service resources")
		}
	case "legacy-adapter":
		if err := validateLegacyAdapter(manifest.LegacyAdapter); err != nil {
			return err
		}
	case "external-cli":
		if strings.TrimSpace(manifest.Binary) == "" {
			return fmt.Errorf("binary is required for external-cli resources")
		}
	case "cloud-api":
		if strings.TrimSpace(manifest.Endpoint) == "" {
			return fmt.Errorf("endpoint is required for cloud-api resources")
		}
	}
	return nil
}

func CurrentPlatform() string {
	switch runtime.GOOS {
	case "linux":
		return "linux"
	case "darwin":
		return "macos"
	case "windows":
		return "windows"
	default:
		return runtime.GOOS
	}
}

func (platforms ResourcePlatforms) SupportForCurrentPlatform() string {
	switch CurrentPlatform() {
	case "linux":
		return strings.TrimSpace(platforms.Linux)
	case "macos":
		return strings.TrimSpace(platforms.MacOS)
	case "windows":
		return strings.TrimSpace(platforms.Windows)
	default:
		return ""
	}
}

func validateLegacyAdapter(adapter ResourceLegacyAdapter) error {
	if strings.TrimSpace(adapter.Owner) == "" {
		return fmt.Errorf("legacy_adapter.owner is required for legacy-adapter resources")
	}
	if strings.TrimSpace(adapter.DecisionDeadline) == "" {
		return fmt.Errorf("legacy_adapter.decision_deadline is required for legacy-adapter resources")
	}
	switch strings.TrimSpace(adapter.FinalDisposition) {
	case "migrate", "blueprint", "deprecate":
	default:
		return fmt.Errorf("legacy_adapter.final_disposition %q is invalid", adapter.FinalDisposition)
	}
	if strings.TrimSpace(adapter.LegacyCLIPath) == "" {
		return fmt.Errorf("legacy_adapter.legacy_cli_path is required for legacy-adapter resources")
	}
	return nil
}

func validatePlatforms(platforms ResourcePlatforms) error {
	values := []string{platforms.Linux, platforms.MacOS, platforms.Windows}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if !slices.Contains(AllowedPlatformSupportStates, value) {
			return fmt.Errorf("platform support value %q is invalid", value)
		}
	}
	return nil
}

func validateHealthCheck(check ResourceHealthCheck) error {
	switch strings.TrimSpace(check.Type) {
	case "tcp", "http":
		if strings.TrimSpace(check.Target) == "" {
			return fmt.Errorf("health check target is required for %s checks", check.Type)
		}
	case "command":
		if len(check.Command) == 0 {
			return fmt.Errorf("health check command is required for command checks")
		}
	default:
		return fmt.Errorf("health check type %q is invalid", check.Type)
	}
	return nil
}
