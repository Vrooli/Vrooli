package manifest

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"github.com/vrooli/vrooli/internal/credentialspec"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/safeguards"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/tools"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
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
	"managed-service",
}

var (
	AllowedPlatformSupportStates = []string{"supported", "partial", "unsupported"}
	AllowedGPUProbes             = []string{"nvidia"}
	AllowedHealthCheckKinds      = []string{"readiness", "liveness"}
)

type ResourceManifest struct {
	Schema                string                       `json:"$schema,omitempty"`
	Name                  string                       `json:"name"`
	DisplayName           string                       `json:"display_name,omitempty"`
	Description           string                       `json:"description,omitempty"`
	CLI                   *scenario.CLIConfig          `json:"cli"`
	LegacyRepoDataAllowed bool                         `json:"legacy_repo_data_allowed,omitempty"`
	DurableData           *ResourceDurableData         `json:"durable_data,omitempty"`
	Template              string                       `json:"template,omitempty"`
	Driver                string                       `json:"driver"`
	ComposeFile           string                       `json:"compose_file,omitempty"`
	Binary                string                       `json:"binary,omitempty"`
	VersionArgs           []string                     `json:"version_args,omitempty"`
	Endpoint              string                       `json:"endpoint,omitempty"`
	Credentials           ResourceCredentials          `json:"credentials,omitempty"`
	Privilege             hostreqspec.Privilege        `json:"privilege"`
	Bundling              hostreqspec.Bundling         `json:"bundling"`
	Category              string                       `json:"category,omitempty"`
	Requirements          ResourceRequirements         `json:"requirements"`
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
	Deployment            ResourceDeployment           `json:"deployment,omitempty"`
	ManagedService        *ResourceManagedService      `json:"managed_service,omitempty"`
	// ProviderPolicy declares verified reuse policy for non-service resources
	// such as host tools. It has no lifecycle authority and is intentionally
	// separate from managed_service.provider_policy.
	ProviderPolicy *resourcedeployment.ProviderPolicy `json:"provider_policy,omitempty"`
	// Companions are long-lived HOST-side processes the compose-service driver
	// starts after the container(s) come up and stops when the resource stops.
	// They exist for resources whose container alone cannot do everything on the
	// host — e.g. whisper's activity edge (a reverse proxy that brackets each
	// /asr to report capacity activity). Absent (the default) => the driver is
	// byte-identical to no-companion behavior.
	Companions []ResourceCompanion `json:"companions,omitempty"`
	// RuntimeEnvCommand, when set, runs before every compose invocation
	// for this resource. Its stdout must be lines of `KEY=VALUE` pairs;
	// each pair is appended to the compose process environment. Use it
	// for dynamic env values the static fields cannot express — e.g.
	// hardware-aware model selection for whisper.
	RuntimeEnvCommand *ResourceRuntimeEnvCommand `json:"runtime_env_command,omitempty"`
}

// ResourceRequirements is the authored resource footprint consumed by
// deployability resolution. It intentionally contains no readiness verdict.
type ResourceRequirements struct {
	Class            string  `json:"class"`
	Weight           float64 `json:"weight"`
	RAMMB            float64 `json:"ram_mb,omitempty"`
	DiskMB           float64 `json:"disk_mb,omitempty"`
	CPUCores         float64 `json:"cpu_cores,omitempty"`
	GPU              bool    `json:"gpu,omitempty"`
	Network          string  `json:"network,omitempty"`
	StorageMBPerUser float64 `json:"storage_mb_per_user,omitempty"`
	StartupTimeMS    float64 `json:"startup_time_ms,omitempty"`
	Bucket           string  `json:"bucket,omitempty"`
	Source           string  `json:"source"`
	Confidence       string  `json:"confidence"`
}

// ResourceDeployment is the resource-owned, target-specific delivery claim.
// Scenarios may select a declared fallback, but cannot invent a portable route.
type (
	ResourceDeployment        = resourcedeployment.Deployment
	ResourceDeploymentProfile = resourcedeployment.Profile
	ResourceDeploymentTarget  = resourcedeployment.Target
	ResourceManagedService    = resourcedeployment.ManagedService
)

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

// ResourceCompanion is one host-side companion process supervised alongside a
// compose-service resource. It is launched detached (its own session) so it
// survives the short-lived control CLI, tracked by a pidfile under the
// runtime-home processes dir, and signaled on stop. Start/stop is idempotent and
// best-effort: a companion that fails to start is a logged warning, not a fatal
// error (the resource's health check surfaces a dead edge).
type ResourceCompanion struct {
	// Name is a stable identifier (the pidfile/logfile basename).
	Name string `json:"name"`
	// Command is the executable to run (resolved on PATH), e.g. "resource-whisper".
	Command string `json:"command"`
	// Args are the subcommand + flags, e.g. ["activity-proxy"].
	Args []string `json:"args,omitempty"`
	// Port is the host port the companion owns (informational; surfaced in the
	// process record so operators can see what binds it).
	Port int `json:"port,omitempty"`
}

type ResourceGPU struct {
	Probe          string            `json:"probe"`
	ComposeOverlay string            `json:"compose_overlay,omitempty"`
	EnvOverrides   map[string]string `json:"env_overrides,omitempty"`
}

// ResourceDurableData declares durable host-filesystem state a resource
// accumulates that is worth backing up (e.g. an external-cli coding agent's
// conversation history). Host-filesystem only: container/compose resources
// store their durable data in Docker volumes that data-backup-manager captures
// via live source-kind connectors and MUST NOT declare it here. The block is
// consumed read-only by data-backup-manager discovery to surface one-click
// backup target suggestions.
type ResourceDurableData struct {
	// Base is the host directory entries are relative to. Supports a leading
	// $HOME, ~, or %USERPROFILE% token (resolved to the operator's home);
	// defaults to $HOME when empty. Slash-normalized.
	Base string `json:"base,omitempty"`
	// HostOnly is reserved; always true today. A pointer so an explicit false is
	// distinguishable and rejected.
	HostOnly *bool `json:"host_only,omitempty"`
	// Entries are durable locations under Base, keyed by stable logical name
	// (the key becomes the suggested backup-target name).
	Entries map[string]DurableDataEntry `json:"entries"`
}

// DurableDataEntry is one durable on-disk location. It mirrors the shared
// `durableDataEntry` schema definition in .vrooli/schemas/common.schema.json;
// data-backup-manager keeps its own local mirror for the same shape.
type DurableDataEntry struct {
	Path        string `json:"path"`
	Kind        string `json:"kind"`
	Regenerable bool   `json:"regenerable"`
	Format      string `json:"format,omitempty"`
	Sensitive   bool   `json:"sensitive,omitempty"`
	Rationale   string `json:"rationale,omitempty"`
}

// durableDataBaseTokens are the host-home tokens a durable_data base may start
// with; the resolver substitutes the operator's home for them.
var durableDataBaseTokens = []string{"$HOME", "~", "%USERPROFILE%"}

type ResourcePlatforms struct {
	Linux   string `json:"linux,omitempty"`
	MacOS   string `json:"macos,omitempty"`
	Windows string `json:"windows,omitempty"`
}

type ResourcePort struct {
	Name      string `json:"name,omitempty"`
	HostIP    string `json:"host_ip,omitempty"`
	Container int    `json:"container,omitempty"`
	Host      int    `json:"host,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
}

type ResourceHealthCheck struct {
	Type    string   `json:"type"`
	Target  string   `json:"target,omitempty"`
	Command []string `json:"command,omitempty"`
	// Kind declares the check's semantics: "readiness" means the check must
	// fail until the resource can actually serve its primary capability
	// (including model/data load), "liveness" means process-alive only.
	// Manifest-level health checks are treated as readiness probes by the
	// control plane; declare "liveness" only for supplementary checks.
	Kind            string `json:"kind,omitempty"`
	ExpectedStatus  []int  `json:"expected_status,omitempty"`
	IntervalSeconds int    `json:"interval_seconds,omitempty"`
	TimeoutSeconds  int    `json:"timeout_seconds,omitempty"`
}

type ResourceInstall struct {
	Command   []string            `json:"command,omitempty"`
	Platforms map[string][]string `json:"platforms,omitempty"`
}

// The credential declaration shape is shared with scenario service manifests
// and therefore lives in credentialspec, which sits below both. These are
// aliases rather than distinct types on purpose: a scenario-declared
// credential and a resource-declared one must be the same thing to the store,
// the diagnosis, and the recovery bundle.
type (
	ResourceCredentials  = credentialspec.Declaration
	CredentialDescriptor = credentialspec.Descriptor
)

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
	StartupOrder            int                    `json:"startup_order,omitempty"`
	StartupTimeoutSeconds   int                    `json:"startup_timeout_seconds,omitempty"`
	StartupTimeEstimate     string                 `json:"startup_time_estimate,omitempty"`
	StartupBudget           *ResourceStartupBudget `json:"startup_budget,omitempty"`
	Dependencies            []string               `json:"dependencies,omitempty"`
	OptionalDependencies    []string               `json:"optional_dependencies,omitempty"`
	RecoveryAttempts        int                    `json:"recovery_attempts,omitempty"`
	RecoveryStrategy        string                 `json:"recovery_strategy,omitempty"`
	RecoveryWaitSeconds     int                    `json:"recovery_wait_seconds,omitempty"`
	RecoveryDelaySeconds    int                    `json:"recovery_delay_seconds,omitempty"`
	HealthCheckRetries      int                    `json:"health_check_retries,omitempty"`
	HealthCheckDelaySeconds int                    `json:"health_check_delay_seconds,omitempty"`
	Priority                string                 `json:"priority,omitempty"`
}

// ResourceStartupBudget describes an optional driver-owned readiness budget
// that grows with a measured unit count. The declared lifecycle timeout remains
// the floor; a driver may raise it when it can observe the unit count without
// guessing. Keeping this contract in the manifest lets resource authors state
// why a timeout grows while leaving the actual observation to the driver.
type ResourceStartupBudget struct {
	Kind           string `json:"kind,omitempty"`
	BaseSeconds    int    `json:"base_seconds,omitempty"`
	PerUnitSeconds int    `json:"per_unit_seconds,omitempty"`
	MaxSeconds     int    `json:"max_seconds,omitempty"`
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
	if err := validateDurableData(manifest.Driver, manifest.DurableData); err != nil {
		return err
	}
	if err := validateDeployment(manifest); err != nil {
		return err
	}
	if manifest.ProviderPolicy != nil {
		if manifest.Driver != "external-cli" && manifest.Driver != "native-cli" {
			return fmt.Errorf("provider_policy is only supported by external-cli and native-cli resources")
		}
		if _, err := manifest.ProviderPolicy.ResolveProvider(resourcedeployment.ProviderRequest{}); err != nil {
			return fmt.Errorf("provider_policy: %w", err)
		}
		if manifest.ProviderPolicy.DefaultMode != resourcedeployment.ProviderManagedDiscovered {
			return fmt.Errorf("non-service provider_policy default_mode must be managed-discovered")
		}
	}
	switch manifest.Driver {
	case "docker-service":
		if strings.TrimSpace(manifest.Runtime.Image) == "" {
			return fmt.Errorf("runtime.image is required for docker-service resources")
		}
		if err := ValidatePinnedImageRef(manifest.Runtime.Image); err != nil {
			return err
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
	case "managed-service":
		if manifest.ManagedService == nil {
			return fmt.Errorf("managed_service is required for managed-service resources")
		}
		if err := manifest.ManagedService.ProviderPolicy.ValidateManagedServiceTargets(); err != nil {
			return fmt.Errorf("managed_service.provider_policy: %w", err)
		}
		if slices.Contains(manifest.ManagedService.ProviderPolicy.AllowedModes, resourcedeployment.ProviderAttachOnly) {
			if err := manifest.ManagedService.ValidateAttachHealthPath(); err != nil {
				return err
			}
		}
		if err := manifest.ManagedService.Artifact.Validate(); err != nil {
			return fmt.Errorf("managed_service.artifact: %w", err)
		}
		if manifest.ManagedService.Config != nil {
			if err := manifest.ManagedService.Config.Validate(); err != nil {
				return fmt.Errorf("managed_service.config: %w", err)
			}
		}
		for key, value := range manifest.ManagedService.Environment {
			if !envNamePattern.MatchString(key) {
				return fmt.Errorf("managed_service.environment key %q is invalid", key)
			}
			if strings.ContainsRune(value, '\x00') {
				return fmt.Errorf("managed_service.environment value for %q contains NUL", key)
			}
		}
	}
	return nil
}

var (
	AllowedDeploymentSupports = []string{"supported", "conditional", "degraded", "unsupported"}
	AllowedDeploymentModes    = []string{"bundled-client", "bundled-service", "native-host-tool", "docker-desktop", "remote-service", "manual"}
	AllowedDeploymentArchs    = []string{"amd64", "arm64"}
)

func validateDeployment(manifest ResourceManifest) error {
	requirementNames, err := registeredRequirementNames()
	if err != nil {
		return err
	}
	platformSupport := map[string]string{
		"linux":   strings.TrimSpace(manifest.Platforms.Linux),
		"macos":   strings.TrimSpace(manifest.Platforms.MacOS),
		"windows": strings.TrimSpace(manifest.Platforms.Windows),
	}
	for profileName, profile := range manifest.Deployment.Profiles {
		if strings.TrimSpace(profileName) == "" {
			return fmt.Errorf("deployment profile name must not be empty")
		}
		for platform, target := range map[string]*ResourceDeploymentTarget{"linux": profile.Linux, "macos": profile.MacOS, "windows": profile.Windows} {
			if target == nil {
				return fmt.Errorf("deployment.profiles.%s.%s is required", profileName, platform)
			}
			if platformSupport[platform] == "unsupported" && target.Support != "unsupported" {
				return fmt.Errorf("deployment.profiles.%s.%s.support %q contradicts platforms.%s \"unsupported\"", profileName, platform, target.Support, platform)
			}
			if !slices.Contains(AllowedDeploymentSupports, target.Support) {
				return fmt.Errorf("deployment.profiles.%s.%s.support %q is invalid", profileName, platform, target.Support)
			}
			if !slices.Contains(AllowedDeploymentModes, target.Mode) {
				return fmt.Errorf("deployment.profiles.%s.%s.mode %q is invalid", profileName, platform, target.Mode)
			}
			if !deploymentModeAllowedForDriver(manifest.Driver, target.Mode) {
				return fmt.Errorf("deployment.profiles.%s.%s.mode %q is not permitted for %s", profileName, platform, target.Mode, manifest.Driver)
			}
			if target.Support == "unsupported" {
				if strings.TrimSpace(target.Reason) == "" {
					return fmt.Errorf("deployment.profiles.%s.%s.reason is required for unsupported targets", profileName, platform)
				}
				continue
			}
			if err := validateTargetRequirements(profileName, platform, target.Requires, requirementNames); err != nil {
				return err
			}
			if len(target.Architectures) == 0 {
				return fmt.Errorf("deployment.profiles.%s.%s.architectures is required", profileName, platform)
			}
			for _, arch := range target.Architectures {
				if !slices.Contains(AllowedDeploymentArchs, arch) {
					return fmt.Errorf("deployment.profiles.%s.%s.architectures contains invalid arch %q", profileName, platform, arch)
				}
			}
			if len(target.Evidence) == 0 {
				return fmt.Errorf("deployment.profiles.%s.%s.evidence is required for %s targets", profileName, platform, target.Support)
			}
			if len(target.Limitations) == 0 {
				return fmt.Errorf("deployment.profiles.%s.%s.limitations is required for %s targets", profileName, platform, target.Support)
			}
			if target.Support == "supported" {
				for _, requiredEvidence := range []string{"manifest-validation", "artifact-checksum", "target-smoke"} {
					if !slices.Contains(target.Evidence, requiredEvidence) {
						return fmt.Errorf("deployment.profiles.%s.%s.supported requires %q evidence", profileName, platform, requiredEvidence)
					}
				}
			}
			if strings.HasPrefix(target.Mode, "bundled-") && (manifest.CLI == nil || manifest.CLI.Adapter.Kind != "go_module" || manifest.CLI.Distribution == nil || manifest.CLI.Distribution.Kind != "prebuilt_artifact") {
				return fmt.Errorf("deployment.profiles.%s.%s bundled mode requires cli.adapter.kind=go_module and cli.distribution.kind=prebuilt_artifact", profileName, platform)
			}
			if target.Mode == "bundled-service" {
				if manifest.ManagedService == nil {
					return fmt.Errorf("deployment.profiles.%s.%s bundled-service requires managed_service", profileName, platform)
				}
				for _, arch := range target.Architectures {
					if _, err := manifest.ManagedService.Artifact.ForPlatform(platform, arch); err != nil {
						return fmt.Errorf("deployment.profiles.%s.%s bundled-service artifact: %w", profileName, platform, err)
					}
					if _, err := manifest.ManagedService.Artifact.BundleArtifactForPlatform(platform, arch); err != nil {
						return fmt.Errorf("deployment.profiles.%s.%s bundled-service artifact: %w", profileName, platform, err)
					}
				}
			}
		}
	}
	return nil
}

// registeredRequirementNames provides the authoritative vocabulary for a
// resource deployment target's requires list. These names are intentionally
// sourced from the embedded catalogs rather than duplicated in a schema enum:
// the catalog is the registry that owns install and bundle semantics.
func registeredRequirementNames() (map[string]struct{}, error) {
	names := make(map[string]struct{})
	for _, catalog := range []struct {
		fs       fs.FS
		filename string
	}{
		{fs: tools.Manifests, filename: "tool.json"},
		{fs: safeguards.Manifests, filename: "safeguard.json"},
	} {
		err := fs.WalkDir(catalog.fs, ".", func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || entry.Name() != catalog.filename {
				return nil
			}
			data, readErr := fs.ReadFile(catalog.fs, path)
			if readErr != nil {
				return readErr
			}
			var item struct {
				Name string `json:"name"`
			}
			if decodeErr := json.Unmarshal(data, &item); decodeErr != nil {
				return fmt.Errorf("decode %s: %w", path, decodeErr)
			}
			name := strings.TrimSpace(item.Name)
			if name == "" {
				return fmt.Errorf("catalog manifest %s has no name", path)
			}
			if _, exists := names[name]; exists {
				return fmt.Errorf("host requirement catalog name %q is ambiguous", name)
			}
			names[name] = struct{}{}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("load host requirement catalog: %w", err)
		}
	}
	return names, nil
}

func validateTargetRequirements(profile, platform string, requirements []string, registered map[string]struct{}) error {
	seen := make(map[string]struct{}, len(requirements))
	for _, raw := range requirements {
		name := strings.TrimSpace(raw)
		if name == "" {
			return fmt.Errorf("deployment.profiles.%s.%s.requires contains an empty name", profile, platform)
		}
		if _, exists := registered[name]; !exists {
			return fmt.Errorf("deployment.profiles.%s.%s.requires names unknown registered tool or safeguard %q", profile, platform, name)
		}
		if _, duplicate := seen[name]; duplicate {
			return fmt.Errorf("deployment.profiles.%s.%s.requires contains duplicate %q", profile, platform, name)
		}
		seen[name] = struct{}{}
	}
	return nil
}

// deploymentModeAllowedForDriver encodes the archetype baseline as a feasible
// envelope, not a promise. A resource still has to supply target evidence and
// artifact/host requirements before its individual profile is accepted.
func deploymentModeAllowedForDriver(driver, mode string) bool {
	allowed := map[string][]string{
		"cloud-api":       {"bundled-client", "remote-service"},
		"native-cli":      {"native-host-tool", "bundled-client", "manual"},
		"docker-service":  {"docker-desktop", "manual"},
		"compose-service": {"docker-desktop", "manual"},
		"external-cli":    {"native-host-tool", "bundled-client", "manual"},
		"managed-service": {"bundled-service", "remote-service", "manual"},
		"desktop-app":     {"native-host-tool", "manual"},
		"manual":          {"manual"},
	}
	return slices.Contains(allowed[driver], mode)
}

// validateDurableData enforces the durable_data block: only host-filesystem
// drivers may declare it, the base (if any) uses a permitted home token, and
// every entry is a relative slash path with a valid kind/format. It never reads
// the host filesystem — it validates the declaration shape only.
func validateDurableData(driver string, dd *ResourceDurableData) error {
	if dd == nil {
		return nil
	}
	switch strings.TrimSpace(driver) {
	case "external-cli", "native-cli", "desktop-app", "manual", "managed-service":
		// Host-filesystem-bearing drivers may declare durable host data.
	default:
		return fmt.Errorf("durable_data is only valid for host-filesystem drivers (external-cli, native-cli, desktop-app, manual), not %q", driver)
	}
	if dd.HostOnly != nil && !*dd.HostOnly {
		return fmt.Errorf("durable_data.host_only must be true (host-filesystem state only)")
	}
	if base := strings.TrimSpace(dd.Base); base != "" {
		if err := validateDurableDataBase(base); err != nil {
			return err
		}
	}
	if len(dd.Entries) == 0 {
		return fmt.Errorf("durable_data.entries must not be empty")
	}
	seenPaths := map[string]struct{}{}
	for key, entry := range dd.Entries {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("durable_data.entries has an empty key")
		}
		field := "durable_data.entries." + key
		if err := validateDurableDataEntry(field, entry); err != nil {
			return err
		}
		clean := strings.TrimSpace(entry.Path)
		if _, dup := seenPaths[clean]; dup {
			return fmt.Errorf("%s duplicates path %q", field, clean)
		}
		seenPaths[clean] = struct{}{}
	}
	return nil
}

func validateDurableDataBase(base string) error {
	if strings.Contains(base, "\\") {
		return fmt.Errorf("durable_data.base must be slash-normalized (no backslashes): %q", base)
	}
	rest, matched := "", false
	for _, tok := range durableDataBaseTokens {
		if base == tok {
			matched = true
			break
		}
		if strings.HasPrefix(base, tok+"/") {
			rest, matched = strings.TrimPrefix(base, tok+"/"), true
			break
		}
	}
	if !matched {
		return fmt.Errorf("durable_data.base must start with $HOME, ~, or %%USERPROFILE%%: %q", base)
	}
	for _, part := range strings.Split(rest, "/") {
		if part == ".." {
			return fmt.Errorf("durable_data.base must not contain parent traversal: %q", base)
		}
	}
	return nil
}

func validateDurableDataEntry(field string, e DurableDataEntry) error {
	path := strings.TrimSpace(e.Path)
	if path == "" {
		return fmt.Errorf("%s.path must not be empty", field)
	}
	if strings.Contains(path, "\\") {
		return fmt.Errorf("%s.path must be slash-normalized (no backslashes): %q", field, path)
	}
	if strings.HasPrefix(path, "/") {
		return fmt.Errorf("%s.path must be relative to base (no leading slash): %q", field, path)
	}
	for _, part := range strings.Split(path, "/") {
		if part == ".." {
			return fmt.Errorf("%s.path must not contain parent traversal: %q", field, path)
		}
	}
	if e.Kind != "dir" && e.Kind != "file" {
		return fmt.Errorf("%s.kind must be \"dir\" or \"file\": %q", field, e.Kind)
	}
	if e.Format != "" && e.Format != "sqlite" && e.Format != "json" {
		return fmt.Errorf("%s.format must be \"sqlite\" or \"json\": %q", field, e.Format)
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

var (
	runtimeMemoryLimitPattern = regexp.MustCompile(`^[1-9][0-9]*[bBkKmMgG]?$`)
	envNamePattern            = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

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
	if kind := strings.TrimSpace(check.Kind); kind != "" && !slices.Contains(AllowedHealthCheckKinds, kind) {
		return fmt.Errorf("health check kind %q is invalid", check.Kind)
	}
	return nil
}

// ValidatePinnedImageRef enforces the Pinned Runtime Principle
// (docs/resources/deployment-contract.md): a container image must be an
// immutable reference — a version tag or digest — so install/pull can never
// silently change the running engine. Used for docker-service runtime.image
// at manifest validation and by the fleet lint over compose-service files.
func ValidatePinnedImageRef(image string) error {
	ref := strings.TrimSpace(image)
	if at := strings.Index(ref, "@"); at >= 0 {
		if strings.TrimSpace(ref[at+1:]) == "" {
			return fmt.Errorf("runtime.image %q has an empty digest", image)
		}
		return nil
	}
	slash := strings.LastIndex(ref, "/")
	colon := strings.LastIndex(ref, ":")
	if colon <= slash {
		return fmt.Errorf("runtime.image %q must pin a version tag or digest", image)
	}
	tag := ref[colon+1:]
	if tag == "" || tag == "latest" || tag == "stable" || strings.HasPrefix(tag, "latest-") {
		return fmt.Errorf("runtime.image %q must pin a version tag or digest, not a floating tag", image)
	}
	return nil
}
