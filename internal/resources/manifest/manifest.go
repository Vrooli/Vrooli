package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/scenario"
)

const SchemaPath = ".vrooli/schemas/resource.schema.json"

var AllowedDrivers = []string{
	"docker-service",
	"compose-service",
	"external-cli",
	"cloud-api",
	"desktop-app",
	"manual",
	"native-cli",
}

var (
	AllowedPortabilityTiers      = []string{"full", "partial", "platform-specific"}
	AllowedPlatformSupportStates = []string{"supported", "partial", "unsupported"}
	AllowedGPUProbes             = []string{"nvidia"}
)

type ResourceManifest struct {
	Schema                string                       `json:"$schema,omitempty"`
	Name                  string                       `json:"name"`
	DisplayName           string                       `json:"display_name,omitempty"`
	Description           string                       `json:"description,omitempty"`
	CLI                   *scenario.CLIConfig          `json:"cli"`
	LegacyRepoDataAllowed bool                         `json:"legacy_repo_data_allowed,omitempty"`
	Template              string                       `json:"template,omitempty"`
	Driver                string                       `json:"driver"`
	ComposeFile           string                       `json:"compose_file,omitempty"`
	Binary                string                       `json:"binary,omitempty"`
	VersionArgs           []string                     `json:"version_args,omitempty"`
	Endpoint              string                       `json:"endpoint,omitempty"`
	Credentials           ResourceCredentials          `json:"credentials,omitempty"`
	PortabilityTier       string                       `json:"portability_tier,omitempty"`
	Category              string                       `json:"category,omitempty"`
	Platforms             ResourcePlatforms            `json:"platforms,omitempty"`
	Dependencies          []string                     `json:"dependencies,omitempty"`
	Ports                 []ResourcePort               `json:"ports,omitempty"`
	HealthChecks          []ResourceHealthCheck        `json:"health_checks,omitempty"`
	Install               ResourceInstall              `json:"install,omitempty"`
	Runtime               ResourceRuntime              `json:"runtime,omitempty"`
	DependencySchema      json.RawMessage              `json:"dependency_schema,omitempty"`
	EnvironmentExports    ResourceEnvironmentExports   `json:"environment_exports,omitempty"`
	Orchestration         ResourceOrchestration        `json:"orchestration,omitempty"`
	Lifecycle             ResourceLifecycle            `json:"lifecycle,omitempty"`
	Capabilities          ResourceManifestCapabilities `json:"capabilities,omitempty"`
	TemplateVersion       string                       `json:"template_version,omitempty"`
	HostTools             []hostreqspec.Declaration    `json:"hostTools,omitempty"`
	HostSafeguards        []hostreqspec.Declaration    `json:"hostSafeguards,omitempty"`
	GPU                   *ResourceGPU                 `json:"gpu,omitempty"`
	// RuntimeEnvCommand, when set, runs before every compose invocation
	// for this resource. Its stdout must be lines of `KEY=VALUE` pairs;
	// each pair is appended to the compose process environment. Use it
	// for dynamic env values the static fields cannot express — e.g.
	// hardware-aware model selection for whisper.
	RuntimeEnvCommand *ResourceRuntimeEnvCommand `json:"runtime_env_command,omitempty"`
}

// ResourceRuntimeEnvCommand declares a CLI invocation the compose
// driver runs to harvest dynamic env variables. Stdout is parsed as
// KEY=VALUE lines (blank lines and `#` comments ignored). Stderr is
// surfaced on failure but does not block startup — the driver falls
// through with whatever static env was already built.
type ResourceRuntimeEnvCommand struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
	// TimeoutSeconds caps the harvest. Default 5s.
	TimeoutSeconds int `json:"timeout_seconds,omitempty"`
}

type ResourceGPU struct {
	Probe          string            `json:"probe"`
	ComposeOverlay string            `json:"compose_overlay,omitempty"`
	EnvOverrides   map[string]string `json:"env_overrides,omitempty"`
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

func (c *ResourceCredentials) UnmarshalJSON(data []byte) error {
	type rawCredentials struct {
		Env       []json.RawMessage `json:"env,omitempty"`
		SecretRef string            `json:"secret_ref,omitempty"`
	}

	var raw rawCredentials
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	env := make([]string, 0, len(raw.Env))
	for _, item := range raw.Env {
		var legacy string
		if err := json.Unmarshal(item, &legacy); err == nil {
			env = append(env, legacy)
			continue
		}

		var descriptor struct {
			Env string `json:"env"`
		}
		if err := json.Unmarshal(item, &descriptor); err != nil {
			return fmt.Errorf("credentials.env entry must be a string or object with env: %w", err)
		}
		if strings.TrimSpace(descriptor.Env) == "" {
			return fmt.Errorf("credentials.env descriptor env is required")
		}
		env = append(env, descriptor.Env)
	}

	c.Env = env
	c.SecretRef = raw.SecretRef
	return nil
}

type ResourceRuntime struct {
	Image         string            `json:"image,omitempty"`
	ContainerName string            `json:"container_name,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	Volumes       []ResourceVolume  `json:"volumes,omitempty"`
	Command       []string          `json:"command,omitempty"`
	WorkingDir    string            `json:"working_dir,omitempty"`
	// MemoryLimit, when non-empty, is passed verbatim as `docker run --memory <value>`.
	// Format: a positive integer optionally followed by a unit suffix b/k/m/g (case-insensitive),
	// e.g. "12g", "8192m", "536870912". Only honored by the docker-service driver.
	MemoryLimit string `json:"memory_limit,omitempty"`
}

type ResourceEnvironmentExports struct {
	Static         map[string]string                  `json:"static,omitempty"`
	FromPorts      map[string]string                  `json:"from_ports,omitempty"`
	FromRuntimeEnv []string                           `json:"from_runtime_env,omitempty"`
	Derived        map[string]ResourceDerivedTemplate `json:"derived,omitempty"`
}

type ResourceOrchestration struct {
	StartupOrder            int      `json:"startup_order,omitempty"`
	StartupTimeoutSeconds   int      `json:"startup_timeout_seconds,omitempty"`
	StartupTimeEstimate     string   `json:"startup_time_estimate,omitempty"`
	Dependencies            []string `json:"dependencies,omitempty"`
	OptionalDependencies    []string `json:"optional_dependencies,omitempty"`
	RecoveryAttempts        int      `json:"recovery_attempts,omitempty"`
	RecoveryStrategy        string   `json:"recovery_strategy,omitempty"`
	RecoveryWaitSeconds     int      `json:"recovery_wait_seconds,omitempty"`
	RecoveryDelaySeconds    int      `json:"recovery_delay_seconds,omitempty"`
	HealthCheckRetries      int      `json:"health_check_retries,omitempty"`
	HealthCheckDelaySeconds int      `json:"health_check_delay_seconds,omitempty"`
	Priority                string   `json:"priority,omitempty"`
}

type ResourceDerivedTemplate struct {
	Template string `json:"template,omitempty"`
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

	// SupportsEnsure advertises that the resource CLI exposes an `ensure`
	// subcommand which accepts a scenario's resource-specific config (via
	// `--config-base64 <b64>`) and brings the resource into compliance
	// (e.g. pulling required models). The orchestrator calls this after
	// health check on any healthy resource whose declared dependency
	// includes extra config keys.
	SupportsEnsure bool `json:"supports_ensure,omitempty"`
}

func DefaultPath(root, name string) string {
	return repocontractmeta.ResourceManifestPath(root, name)
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
	if manifest.CLI == nil {
		return fmt.Errorf("cli is required")
	}
	normalizeCLIConfig(manifest.CLI)
	if err := manifest.CLI.Validate(); err != nil {
		return fmt.Errorf("cli: %w", err)
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
	if err := validateDependencySchema(manifest.DependencySchema); err != nil {
		return err
	}
	if err := validateOrchestration(manifest.Orchestration); err != nil {
		return err
	}
	if err := validateEnvironmentExports(manifest); err != nil {
		return err
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
	if err := validateGPU(manifest.GPU); err != nil {
		return err
	}
	if err := validateRuntimeMemoryLimit(manifest.Runtime.MemoryLimit); err != nil {
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
	case "external-cli", "native-cli":
		if strings.TrimSpace(manifest.Binary) == "" {
			return fmt.Errorf("binary is required for %s resources", manifest.Driver)
		}
	case "cloud-api":
		if strings.TrimSpace(manifest.Endpoint) == "" {
			return fmt.Errorf("endpoint is required for cloud-api resources")
		}
	}
	return nil
}

func normalizeCLIConfig(cfg *scenario.CLIConfig) {
	if cfg == nil {
		return
	}
	cfg.ApplyDefaultsForManifest()
}

func validateDependencySchema(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("dependency_schema must be valid JSON: %w", err)
	}
	if len(payload) == 0 {
		return nil
	}
	if got, _ := payload["type"].(string); strings.TrimSpace(got) != "" && got != "object" {
		return fmt.Errorf("dependency_schema.type must be \"object\" when specified")
	}
	properties, ok := payload["properties"]
	if !ok {
		return nil
	}
	propertyMap, ok := properties.(map[string]any)
	if !ok {
		return fmt.Errorf("dependency_schema.properties must be an object")
	}
	baseKeys := map[string]struct{}{
		"$schema":           {},
		"$profile":          {},
		"type":              {},
		"enabled":           {},
		"required":          {},
		"startup_policy":    {},
		"version":           {},
		"purpose":           {},
		"description":       {},
		"degraded_behavior": {},
		"startup_order":     {},
		"initialization":    {},
		"baseUrl":           {},
		"apiKey":            {},
		"healthCheck":       {},
		"connection":        {},
		"security":          {},
		"labels":            {},
		"annotations":       {},
	}
	for key := range propertyMap {
		if _, exists := baseKeys[key]; exists {
			return fmt.Errorf("dependency_schema.properties[%q] duplicates a base resource dependency key", key)
		}
	}
	return nil
}

func validateOrchestration(orchestration ResourceOrchestration) error {
	if orchestration.StartupOrder < 0 ||
		orchestration.StartupTimeoutSeconds < 0 ||
		orchestration.RecoveryAttempts < 0 ||
		orchestration.RecoveryWaitSeconds < 0 ||
		orchestration.RecoveryDelaySeconds < 0 ||
		orchestration.HealthCheckRetries < 0 ||
		orchestration.HealthCheckDelaySeconds < 0 {
		return fmt.Errorf("orchestration values must be non-negative")
	}
	return nil
}

func validateEnvironmentExports(manifest ResourceManifest) error {
	exported := map[string]string{}
	for key := range manifest.EnvironmentExports.Static {
		name := strings.TrimSpace(key)
		if name == "" {
			return fmt.Errorf("environment_exports.static contains an empty key")
		}
		exported[name] = "static"
	}
	for key, portName := range manifest.EnvironmentExports.FromPorts {
		name := strings.TrimSpace(key)
		if name == "" {
			return fmt.Errorf("environment_exports.from_ports contains an empty key")
		}
		if previous, exists := exported[name]; exists {
			return fmt.Errorf("environment_exports key %q is declared multiple times (%s and from_ports)", name, previous)
		}
		if strings.TrimSpace(portName) == "" {
			return fmt.Errorf("environment_exports.from_ports[%q] must reference a port name", name)
		}
		exported[name] = "from_ports"
	}
	for _, key := range manifest.EnvironmentExports.FromRuntimeEnv {
		name := strings.TrimSpace(key)
		if name == "" {
			return fmt.Errorf("environment_exports.from_runtime_env contains an empty key")
		}
		if previous, exists := exported[name]; exists {
			return fmt.Errorf("environment_exports key %q is declared multiple times (%s and from_runtime_env)", name, previous)
		}
		exported[name] = "from_runtime_env"
	}
	for key, derived := range manifest.EnvironmentExports.Derived {
		name := strings.TrimSpace(key)
		if name == "" {
			return fmt.Errorf("environment_exports.derived contains an empty key")
		}
		if previous, exists := exported[name]; exists {
			return fmt.Errorf("environment_exports key %q is declared multiple times (%s and derived)", name, previous)
		}
		if strings.TrimSpace(derived.Template) == "" {
			return fmt.Errorf("environment_exports.derived[%q].template is required", name)
		}
		exported[name] = "derived"
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

func validateGPU(gpu *ResourceGPU) error {
	if gpu == nil {
		return nil
	}
	probe := strings.TrimSpace(gpu.Probe)
	if probe == "" {
		return fmt.Errorf("gpu.probe is required when gpu block is present")
	}
	if !slices.Contains(AllowedGPUProbes, probe) {
		return fmt.Errorf("gpu.probe %q is invalid (allowed: %v)", probe, AllowedGPUProbes)
	}
	if strings.TrimSpace(gpu.ComposeOverlay) == "" && len(gpu.EnvOverrides) == 0 {
		return fmt.Errorf("gpu block must set compose_overlay or env_overrides")
	}
	return nil
}

var runtimeMemoryLimitPattern = regexp.MustCompile(`^[1-9][0-9]*[bBkKmMgG]?$`)

func validateRuntimeMemoryLimit(value string) error {
	v := strings.TrimSpace(value)
	if v == "" {
		return nil
	}
	if !runtimeMemoryLimitPattern.MatchString(v) {
		return fmt.Errorf("runtime.memory_limit %q is invalid (expected positive integer optionally suffixed with b/k/m/g)", value)
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
