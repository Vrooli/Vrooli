package scenario

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/credentialspec"
	"github.com/vrooli/vrooli/internal/discovery"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/operatorcapability"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
)

var ErrNotFound = errors.New("scenario not found")

const defaultScenarioServiceRelPath = repocontractmeta.ServiceManifestPathname

func ProjectServicePath(root string) string {
	return filepath.Join(filepath.Clean(root), filepath.FromSlash(defaultScenarioServiceRelPath))
}

func ServicePath(root, name string) string {
	scenarioPath := contractPaths.ScenarioRootPath(root, name)
	return contractPaths.ScenarioServicePath(root, name, scenarioPath)
}

type SandboxEnv struct {
	ID     string
	Merged string
	Scope  string
}

func SandboxEnvFromEnv() SandboxEnv {
	return SandboxEnv{
		ID:     strings.TrimSpace(os.Getenv("VROOLI_SANDBOX_ID")),
		Merged: strings.TrimSpace(os.Getenv("VROOLI_SANDBOX_MERGED")),
		Scope:  strings.TrimSpace(os.Getenv("VROOLI_SANDBOX_SCOPE")),
	}
}

func (env SandboxEnv) Enabled() bool {
	return env.Merged != ""
}

type Scenario struct {
	Slug        string
	Path        string
	ServicePath string
	Redirected  bool
	Manifest    ServiceManifest
	// Variant names which instance of the scenario this descriptor addresses.
	// Empty means the canonical "live" instance, so every pre-variant caller is
	// a backward-compatible no-op. The lifecycle sets it from the parsed
	// InstanceKey at the Start/Stop/Restart entry points; downstream registry,
	// lock, port, and storage-namespace derivations all read it (normalized
	// through scenarioruntime.InstanceKey). See the Baseline Modes plan, §1a.
	Variant string
}

type ServiceManifest struct {
	Schema          string                                 `json:"$schema,omitempty"`
	Version         string                                 `json:"version,omitempty"`
	Service         ServiceMetadata                        `json:"service"`
	Generation      *GenerationMetadata                    `json:"generation,omitempty"`
	CLI             *CLIConfig                             `json:"cli,omitempty"`
	Ports           map[string]Port                        `json:"ports,omitempty"`
	Components      map[string]Component                   `json:"components,omitempty"`
	Lifecycle       Lifecycle                              `json:"lifecycle,omitempty,omitzero"`
	Health          *HealthConfig                          `json:"health,omitempty"`
	Dependencies    Dependencies                           `json:"dependencies,omitempty"`
	Credentials     credentialspec.Declaration             `json:"credentials,omitempty"`
	Capabilities    []operatorcapability.ManifestReference `json:"operator_capabilities,omitempty"`
	TrustSigning    *TrustSigningConfig                    `json:"trust_signing,omitempty"`
	Environment     map[string]string                      `json:"environment,omitempty"`
	HostTools       []hostreqspec.Declaration              `json:"hostTools,omitempty"`
	HostSafeguards  []hostreqspec.Declaration              `json:"hostSafeguards,omitempty"`
	TierFeasibility *TierFeasibility                       `json:"tier_feasibility,omitempty"`
}

// TrustSigningConfig is a declarative lifecycle contract for a scenario that
// signs protected evidence. It identifies a provider and workload credential
// file; it never embeds a secret or signing key.
type TrustSigningConfig struct {
	Provider                string   `json:"provider"`
	Resource                string   `json:"resource"`
	Address                 string   `json:"address,omitempty"`
	KeyName                 string   `json:"key_name,omitempty"`
	CredentialFile          string   `json:"credential_file,omitempty"`
	OperatorCredentialFile  string   `json:"operator_credential_file,omitempty"`
	OperatorSubjects        []string `json:"operator_subjects,omitempty"`
	OperatorTLSCertFile     string   `json:"operator_tls_cert_file,omitempty"`
	OperatorTLSKeyFile      string   `json:"operator_tls_key_file,omitempty"`
	OperatorTLSClientCAFile string   `json:"operator_tls_client_ca_file,omitempty"`
}

func (config TrustSigningConfig) Validate(dependencies Dependencies) error {
	switch strings.TrimSpace(config.Provider) {
	case "development":
		return nil
	case "vault-transit":
		if strings.TrimSpace(config.Resource) != "vault" {
			return fmt.Errorf("trust_signing vault-transit provider requires resource=vault")
		}
		if _, ok := dependencies.Resources["vault"]; !ok {
			return fmt.Errorf("trust_signing vault-transit provider requires a declared vault resource dependency")
		}
		if !strings.HasPrefix(strings.TrimSpace(config.Address), "https://") {
			return fmt.Errorf("trust_signing vault-transit address must use https")
		}
		if strings.TrimSpace(config.KeyName) == "" || strings.TrimSpace(config.CredentialFile) == "" {
			return fmt.Errorf("trust_signing vault-transit requires key_name and credential_file")
		}
		if config.OperatorCredentialFile != "" && len(config.OperatorSubjects) == 0 {
			return fmt.Errorf("trust_signing operator_credential_file requires operator_subjects")
		}
		if config.OperatorCredentialFile != "" && (strings.TrimSpace(config.OperatorTLSCertFile) == "" || strings.TrimSpace(config.OperatorTLSKeyFile) == "" || strings.TrimSpace(config.OperatorTLSClientCAFile) == "") {
			return fmt.Errorf("trust_signing operator rotation requires operator_tls_cert_file, operator_tls_key_file, and operator_tls_client_ca_file")
		}
		return nil
	default:
		return fmt.Errorf("trust_signing provider must be development or vault-transit")
	}
}

type GenerationMetadata struct {
	Template    GenerationTemplate `json:"template,omitempty"`
	GeneratedAt string             `json:"generated_at,omitempty"`
	Design      GenerationDesign   `json:"design,omitempty"`
	ManifestSha string             `json:"manifest_sha,omitempty"`
	ContentSha  string             `json:"content_sha,omitempty"`
}

type GenerationTemplate struct {
	ID      string `json:"id,omitempty"`
	Version string `json:"version,omitempty"`
}

type GenerationDesign struct {
	ID      string `json:"id,omitempty"`
	Version string `json:"version,omitempty"`
	Adapter string `json:"adapter,omitempty"`
}

type ServiceMetadata struct {
	Parent      string   `json:"parent,omitempty"`
	Name        string   `json:"name,omitempty"`
	DisplayName string   `json:"displayName,omitempty"`
	Description string   `json:"description,omitempty"`
	Version     string   `json:"version,omitempty"`
	Type        string   `json:"type,omitempty"`
	Category    string   `json:"category,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type CLIConfig struct {
	Enabled   bool               `json:"enabled"`
	Command   string             `json:"command,omitempty"`
	Adapter   CLIAdapterConfig   `json:"adapter,omitempty"`
	Artifacts CLIArtifactsConfig `json:"artifacts,omitempty"`
	// Distribution is the release artifact consumed by deployment packaging.
	// Install remains a source/developer workflow, not a target requirement.
	Distribution *CLIDistributionConfig `json:"distribution,omitempty"`
	// SourceBuild declares how a source checkout builds this CLI for developer
	// workflows. Deployment must use Distribution artifacts instead; it never
	// builds source on the target device.
	SourceBuild *CLISourceBuildConfig `json:"source_build,omitempty"`
	Invoke      CLIInvokeConfig       `json:"invoke,omitempty"`
	Freshness   *CLIFreshnessCheck    `json:"freshness,omitempty"`
}

type CLIDistributionConfig struct {
	Kind         string `json:"kind,omitempty"`
	ArtifactName string `json:"artifact_name,omitempty"`
}

type CLISourceBuildConfig struct {
	Kind string `json:"kind,omitempty"`
}

type CLIArtifactsConfig struct {
	Manifest      CLIArtifactConfig `json:"manifest,omitempty"`
	BuildMetadata CLIArtifactConfig `json:"build_metadata,omitempty"`
}

type CLIArtifactConfig struct {
	Location string `json:"location,omitempty"`
}

type CLIAdapterConfig struct {
	Kind      string `json:"kind,omitempty"`
	ModuleDir string `json:"module_dir,omitempty"`
}

type CLIInvokeConfig struct {
	Kind    string `json:"kind,omitempty"`
	Command string `json:"command,omitempty"`
}

type CLIFreshnessCheck struct {
	Inputs       []string `json:"inputs,omitempty"`
	MetadataFile string   `json:"metadata_file,omitempty"`
}

const CLIArtifactLocationSibling = "sibling"

type Dependencies struct {
	Resources map[string]Dependency `json:"resources,omitempty"`
	Scenarios map[string]Dependency `json:"scenarios,omitempty"`
}

type Dependency struct {
	Type                 string `json:"type,omitempty"`
	Enabled              bool   `json:"enabled,omitempty"`
	Required             bool   `json:"required,omitempty"`
	StartupPolicy        string `json:"startup_policy,omitempty"`
	FreshnessPolicy      string `json:"freshness_policy,omitempty"`
	DegradedBehavior     string `json:"degraded_behavior,omitempty"`
	Purpose              string `json:"purpose,omitempty"`
	Description          string `json:"description,omitempty"`
	Database             string `json:"database,omitempty"`
	VersionRange         string `json:"versionRange,omitempty"`
	RuntimeOnly          bool   `json:"runtime_only,omitempty"`
	RuntimeOnlyRationale string `json:"runtime_only_rationale,omitempty"`
	BundlePolicy         string `json:"bundle_policy,omitempty"`

	// Config holds dependency-specific keys that aren't modeled as typed fields.
	// The declaring scenario and the dependency own the config schema together.
	// Always a JSON object when non-empty.
	Config json.RawMessage `json:"config,omitempty"`
}

// Component is one long-running process owned by a scenario. Build declares
// the portable artifact and Run declares how that artifact is launched. It
// intentionally has no secret field: credentials and resource exports keep
// their existing authorities.
type Component struct {
	Role  string         `json:"role"`
	Build ComponentBuild `json:"build"`
	Run   ComponentRun   `json:"run"`
}

type ComponentBuild struct {
	Kind   string `json:"kind,omitempty"`
	Dir    string `json:"dir,omitempty"`
	Entry  string `json:"entry,omitempty"`
	Output string `json:"output,omitempty"`
	Reuse  string `json:"reuse,omitempty"`
}

type ComponentRun struct {
	Argv         []string              `json:"argv"`
	CWD          string                `json:"cwd,omitempty"`
	Env          map[string]string     `json:"env,omitempty"`
	Port         string                `json:"port,omitempty"`
	DataDirs     []string              `json:"data_dirs,omitempty"`
	LogDir       string                `json:"log_dir,omitempty"`
	Readiness    *ComponentReadiness   `json:"readiness,omitempty"`
	DependsOn    []ComponentDependency `json:"depends_on,omitempty"`
	SupervisedBy string                `json:"supervised_by,omitempty"`
	Condition    *Condition            `json:"condition,omitempty"`
}

type ComponentReadiness struct {
	Type      string `json:"type"`
	Path      string `json:"path,omitempty"`
	TimeoutMS int    `json:"timeout_ms,omitempty"`
}

type ComponentDependency struct {
	Component string `json:"component"`
	Wait      string `json:"wait"`
}

// TierFeasibility is authored evidence about where a scenario can run. It is
// an input to deployment analysis, never a persisted analyzer verdict.
type TierFeasibility struct {
	MetadataVersion  int                           `json:"metadata_version,omitempty"`
	Tiers            map[string]DeploymentTier     `json:"tiers,omitempty"`
	Dependencies     DeploymentDependencyCatalog   `json:"dependencies,omitempty"`
	Overrides        []DeploymentOverride          `json:"overrides,omitempty"`
	SupportedTiers   []int                         `json:"supported_tiers,omitempty"`
	Platforms        []string                      `json:"platforms,omitempty"`
	DesktopReady     bool                          `json:"desktop_ready,omitempty"`
	MinimalResources []string                      `json:"minimal_resources,omitempty"`
	BuildConfigs     map[string]ServiceBuildConfig `json:"build_configs,omitempty"`
}

type ServiceBuildConfig struct {
	Type          string `json:"type,omitempty"`
	SourceDir     string `json:"source_dir,omitempty"`
	EntryPoint    string `json:"entry_point,omitempty"`
	OutputPattern string `json:"output_pattern,omitempty"`
}

type DeploymentDependencyCatalog struct {
	Resources map[string]DeploymentDependency `json:"resources,omitempty"`
	Scenarios map[string]DeploymentDependency `json:"scenarios,omitempty"`
}

type DeploymentDependency struct {
	ResourceType    string                           `json:"resource_type,omitempty"`
	Footprint       *DeploymentRequirements          `json:"footprint,omitempty"`
	PlatformSupport map[string]DependencyTierSupport `json:"platform_support,omitempty"`
	SwappableWith   []DependencySwap                 `json:"swappable_with,omitempty"`
	PackagingHints  []string                         `json:"packaging_hints,omitempty"`
}

type DeploymentRequirements struct {
	Class            string   `json:"class,omitempty"`
	Weight           *float64 `json:"weight,omitempty"`
	RAMMB            *float64 `json:"ram_mb,omitempty"`
	DiskMB           *float64 `json:"disk_mb,omitempty"`
	CPUCores         *float64 `json:"cpu_cores,omitempty"`
	GPU              *bool    `json:"gpu,omitempty"`
	Network          string   `json:"network,omitempty"`
	StorageMBPerUser *float64 `json:"storage_mb_per_user,omitempty"`
	StartupTimeMS    *float64 `json:"startup_time_ms,omitempty"`
	Bucket           string   `json:"bucket,omitempty"`
	Source           string   `json:"source,omitempty"`
	Confidence       string   `json:"confidence,omitempty"`
}

type DependencyTierSupport struct {
	Supported    *bool                   `json:"supported,omitempty"`
	FitnessScore *float64                `json:"fitness_score,omitempty"`
	Reason       string                  `json:"reason,omitempty"`
	Requirements *DeploymentRequirements `json:"requirements,omitempty"`
	Alternatives []string                `json:"alternatives,omitempty"`
	Notes        string                  `json:"notes,omitempty"`
}

type DependencySwap struct {
	ID           string `json:"id,omitempty"`
	Relationship string `json:"relationship,omitempty"`
	Notes        string `json:"notes,omitempty"`
}

type DeploymentTier struct {
	Requirements *DeploymentRequirements `json:"requirements,omitempty"`
	Adaptations  []DeploymentAdaptation  `json:"adaptations,omitempty"`
	Artifacts    []DeploymentArtifact    `json:"artifacts,omitempty"`
	Notes        string                  `json:"notes,omitempty"`
}

type DeploymentAdaptation struct {
	Dependency string  `json:"dependency,omitempty"`
	Swap       string  `json:"swap,omitempty"`
	Impact     string  `json:"impact,omitempty"`
	EffortDays float64 `json:"effort_days,omitempty"`
	Notes      string  `json:"notes,omitempty"`
}

type DeploymentArtifact struct {
	Type     string `json:"type,omitempty"`
	Producer string `json:"producer,omitempty"`
	Status   string `json:"status,omitempty"`
	Notes    string `json:"notes,omitempty"`
}

type DeploymentOverride struct {
	Tier      string      `json:"tier,omitempty"`
	Field     string      `json:"field,omitempty"`
	Value     interface{} `json:"value,omitempty"`
	Reason    string      `json:"reason,omitempty"`
	ExpiresAt string      `json:"expires_at,omitempty"`
}

const (
	DependencyStartupPolicyMustStart = "must_start"
	DependencyStartupPolicyTryStart  = "try_start"
	DependencyStartupPolicyIgnore    = "ignore"
)

// Freshness policy governs whether a running, healthy dependency whose sources
// have changed (stale) is disrupted. It is orthogonal to startup_policy
// (availability): startup_policy decides whether to start a dependency at all,
// freshness_policy decides how aggressively to react when a started one is
// stale. The default preserves historical behavior (restart on stale).
const (
	// DependencyFreshnessPolicyRestartWhenStale rebuilds AND restarts a stale
	// dependency (gated by the artifact-digest check: a byte-identical rebuild
	// does not restart). This is the default when unset.
	DependencyFreshnessPolicyRestartWhenStale = "restart_when_stale"
	// DependencyFreshnessPolicyReuseRunning keeps the running process even when
	// stale (a warning is emitted); no rebuild, no restart.
	DependencyFreshnessPolicyReuseRunning = "reuse_running"
	// DependencyFreshnessPolicyRebuildOnly rebuilds the artifact when stale but
	// never restarts the running process.
	DependencyFreshnessPolicyRebuildOnly = "rebuild_only"
)

type rawDependencies struct {
	Resources json.RawMessage `json:"resources,omitempty"`
	Scenarios json.RawMessage `json:"scenarios,omitempty"`
}

type Port struct {
	EnvVar      string `json:"env_var,omitempty"`
	Description string `json:"description,omitempty"`
	Range       string `json:"range,omitempty"`
	Port        *int   `json:"port,omitempty"`
}

type Lifecycle struct {
	Version    string        `json:"version,omitempty"`
	Defaults   PhaseDefaults `json:"defaults,omitempty,omitzero"`
	Health     *HealthConfig `json:"health,omitempty"`
	Setup      Phase         `json:"setup,omitempty,omitzero"`
	Develop    Phase         `json:"develop,omitempty,omitzero"`
	Build      Phase         `json:"build,omitempty,omitzero"`
	Deploy     Phase         `json:"deploy,omitempty,omitzero"`
	Clean      Phase         `json:"clean,omitempty,omitzero"`
	Backup     Phase         `json:"backup,omitempty,omitzero"`
	Restore    Phase         `json:"restore,omitempty,omitzero"`
	Production Phase         `json:"production,omitempty,omitzero"`
	Stop       Phase         `json:"stop,omitempty,omitzero"`
}

type PhaseDefaults struct {
	Error string `json:"error,omitempty"`
}

type Phase struct {
	Description string      `json:"description,omitempty"`
	Steps       []PhaseStep `json:"steps,omitempty"`
}

type PhaseStep struct {
	Name        string            `json:"name,omitempty"`
	Exec        []string          `json:"exec,omitempty"`
	CWD         string            `json:"cwd,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	OnError     string            `json:"on_error,omitempty"`
	Retry       *RetryPolicy      `json:"retry,omitempty"`
	Timeout     int               `json:"timeout,omitempty"`
	Description string            `json:"description,omitempty"`
	Background  bool              `json:"background,omitempty"`
	Error       string            `json:"error,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Condition   *Condition        `json:"condition,omitempty"`
}

type RetryPolicy struct {
	MaxAttempts int    `json:"max_attempts,omitempty"`
	Delay       int    `json:"delay,omitempty"`
	Backoff     string `json:"backoff,omitempty"`
}

type Condition struct {
	FileExists      string           `json:"file_exists,omitempty"`
	FileNotExists   string           `json:"file_not_exists,omitempty"`
	DirectoryExists string           `json:"directory_exists,omitempty"`
	JSONPathExists  string           `json:"json_path_exists,omitempty"`
	ResourceEnabled string           `json:"resource_enabled,omitempty"`
	CommandExists   string           `json:"command_exists,omitempty"`
	BinaryExists    string           `json:"binary_exists,omitempty"`
	EnvVarSet       string           `json:"env_var_set,omitempty"`
	EnvSet          string           `json:"env_set,omitempty"`
	EnvNotSet       string           `json:"env_not_set,omitempty"`
	Always          string           `json:"always,omitempty"`
	Checks          []ConditionCheck `json:"checks,omitempty"`
}

type ConditionCheck struct {
	Type                  string   `json:"type,omitempty"`
	Name                  string   `json:"name,omitempty"`
	Command               string   `json:"command,omitempty"`
	BundlePath            string   `json:"bundle_path,omitempty"`
	SourceDir             string   `json:"source_dir,omitempty"`
	Targets               []string `json:"targets,omitempty"`
	Paths                 []string `json:"paths,omitempty"`
	Path                  string   `json:"path,omitempty"`
	Resources             []string `json:"resources,omitempty"`
	Populated             bool     `json:"populated,omitempty"`
	WatchFileDependencies *bool    `json:"watch_file_dependencies,omitempty"`
	DependencyExcludes    []string `json:"dependency_excludes,omitempty"`
}

type HealthConfig struct {
	Description        string          `json:"description,omitempty"`
	Endpoints          HealthEndpoints `json:"endpoints,omitempty"`
	Checks             []HealthCheck   `json:"checks,omitempty"`
	StartupGracePeriod int             `json:"startup_grace_period,omitempty"`
	Timeout            int             `json:"timeout,omitempty"`
	Interval           int             `json:"interval,omitempty"`
}

type HealthEndpoints struct {
	API string `json:"api,omitempty"`
	UI  string `json:"ui,omitempty"`
}

type HealthCheck struct {
	Name     string `json:"name,omitempty"`
	Type     string `json:"type,omitempty"`
	Target   string `json:"target,omitempty"`
	Critical bool   `json:"critical,omitempty"`
	Timeout  int    `json:"timeout,omitempty"`
	Interval int    `json:"interval,omitempty"`
}

type PhaseSummary struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Steps       int    `json:"steps"`
	Defined     bool   `json:"defined"`
}

type PortSummary struct {
	Name        string `json:"name"`
	EnvVar      string `json:"env_var"`
	Description string `json:"description,omitempty"`
	Range       string `json:"range,omitempty"`
	FixedPort   *int   `json:"fixed_port,omitempty"`
}

func Load(root, name string, env SandboxEnv) (Scenario, error) {
	if strings.TrimSpace(name) == "" {
		return Scenario{}, ErrNotFound
	}

	scenarioPath, redirected := ResolveScenarioPath(root, name, env)
	servicePath := scenarioServicePath(root, name, scenarioPath)
	manifest, err := ReadService(servicePath)
	if err != nil {
		if os.IsNotExist(err) {
			return Scenario{}, ErrNotFound
		}
		return Scenario{}, err
	}
	if manifest.Service.Name == "" {
		manifest.Service.Name = name
	}

	return Scenario{
		Slug:        name,
		Path:        scenarioPath,
		ServicePath: servicePath,
		Redirected:  redirected,
		Manifest:    manifest,
	}, nil
}

func Discover(root string, env SandboxEnv) ([]Scenario, error) {
	report, err := DiscoverReport(root, env)
	if err != nil {
		return nil, err
	}
	if len(report.Failures) > 0 {
		failure := report.Failures[0]
		return nil, fmt.Errorf("load scenario %s: %s", failure.Name, failure.Error)
	}
	return report.Items, nil
}

func DiscoverReport(root string, env SandboxEnv) (discovery.Report[Scenario], error) {
	names := make(map[string]struct{})

	canonicalNames, err := scanScenarioNames(scenarioBaseDir(root))
	if err != nil {
		return discovery.Report[Scenario]{}, err
	}
	for _, name := range canonicalNames {
		names[name] = struct{}{}
	}

	sandboxNames, err := scanSandboxScenarioNames(root, env)
	if err != nil {
		return discovery.Report[Scenario]{}, err
	}
	for _, name := range sandboxNames {
		names[name] = struct{}{}
	}

	ordered := make([]string, 0, len(names))
	for name := range names {
		ordered = append(ordered, name)
	}
	sort.Strings(ordered)

	report := discovery.Report[Scenario]{
		Items:    make([]Scenario, 0, len(ordered)),
		Failures: make([]discovery.Failure, 0),
	}
	for _, name := range ordered {
		scenario, err := Load(root, name, env)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			report.Failures = append(report.Failures, discovery.Failure{
				Kind:  "scenario",
				Name:  name,
				Path:  ServicePath(root, name),
				Stage: "load",
				Error: err.Error(),
			})
			continue
		}
		report.Items = append(report.Items, scenario)
	}

	return report, nil
}

func ReadService(path string) (ServiceManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ServiceManifest{}, err
	}
	if err := repocontract.ValidateCredentialDescriptorUniqueness(data, path); err != nil {
		return ServiceManifest{}, fmt.Errorf("validate credentials in %s: %w", path, err)
	}

	var manifest ServiceManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return ServiceManifest{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if manifest.Lifecycle.Health == nil && manifest.Health != nil {
		manifest.Lifecycle.Health = manifest.Health
	}
	if manifest.CLI != nil {
		manifest.CLI.applyDefaults()
		if err := manifest.CLI.Validate(); err != nil {
			return ServiceManifest{}, fmt.Errorf("validate cli in %s: %w", path, err)
		}
	}
	if err := hostreqspec.ValidateDeclarations(hostreqspec.KindTool, manifest.HostTools); err != nil {
		return ServiceManifest{}, fmt.Errorf("validate hostTools in %s: %w", path, err)
	}
	if err := hostreqspec.ValidateDeclarations(hostreqspec.KindSafeguard, manifest.HostSafeguards); err != nil {
		return ServiceManifest{}, fmt.Errorf("validate hostSafeguards in %s: %w", path, err)
	}
	if err := manifest.Dependencies.Validate(); err != nil {
		return ServiceManifest{}, fmt.Errorf("validate dependencies in %s: %w", path, err)
	}
	if manifest.TrustSigning != nil {
		if err := manifest.TrustSigning.Validate(manifest.Dependencies); err != nil {
			return ServiceManifest{}, fmt.Errorf("validate trust_signing in %s: %w", path, err)
		}
	}
	// Note: port policy validation (ephemeral-range overlap, canonical-band
	// membership) is intentionally NOT run here. It is enforced by the
	// lifecycle on Start (see ValidateManifestPorts). Stop, Status, List,
	// and inventory paths must remain tolerant of manifests that pre-date
	// the canonical bands so operators can always tear down and inspect
	// scenarios regardless of their port configuration.
	return manifest, nil
}

// ValidateManifestPorts checks whether the manifest's declared ports sit
// inside the canonical bands and outside the host OS's live ephemeral range.
// Intended to be called from code paths that are about to allocate a port
// (Start), not from observation paths (Stop, List, Status).
//
// The check honors the VROOLI_PORT_VALIDATION=off env var escape hatch.
func ValidateManifestPorts(servicePath string, ports map[string]Port) error {
	return validateManifestPorts(servicePath, ports)
}

func (manifest ServiceManifest) CLIEnabled() bool {
	return manifest.CLI != nil && manifest.CLI.Enabled
}

func (manifest ServiceManifest) CLICommand() string {
	if manifest.CLI == nil {
		return ""
	}
	return strings.TrimSpace(manifest.CLI.Command)
}

func (cfg *CLIConfig) applyDefaults() {
	if cfg == nil {
		return
	}
	cfg.Command = strings.TrimSpace(cfg.Command)
	cfg.Adapter.Kind = strings.TrimSpace(cfg.Adapter.Kind)
	cfg.Adapter.ModuleDir = strings.TrimSpace(cfg.Adapter.ModuleDir)
	cfg.Artifacts.Manifest.Location = strings.TrimSpace(cfg.Artifacts.Manifest.Location)
	cfg.Artifacts.BuildMetadata.Location = strings.TrimSpace(cfg.Artifacts.BuildMetadata.Location)
	if cfg.Distribution != nil {
		cfg.Distribution.Kind = strings.TrimSpace(cfg.Distribution.Kind)
		cfg.Distribution.ArtifactName = strings.TrimSpace(cfg.Distribution.ArtifactName)
	}
	if cfg.SourceBuild != nil {
		cfg.SourceBuild.Kind = strings.TrimSpace(cfg.SourceBuild.Kind)
	}
	cfg.Invoke.Kind = strings.TrimSpace(cfg.Invoke.Kind)
	cfg.Invoke.Command = strings.TrimSpace(cfg.Invoke.Command)
	if cfg.Enabled && cfg.Artifacts.Manifest.Location == "" {
		cfg.Artifacts.Manifest.Location = CLIArtifactLocationSibling
	}
	if cfg.Enabled && cfg.Artifacts.BuildMetadata.Location == "" {
		cfg.Artifacts.BuildMetadata.Location = CLIArtifactLocationSibling
	}
	if cfg.Enabled && cfg.Invoke.Kind == "" {
		cfg.Invoke.Kind = "installed_command"
	}
	if cfg.Enabled && cfg.Invoke.Command == "" {
		cfg.Invoke.Command = cfg.Command
	}
}

func (cfg *CLIConfig) ApplyDefaultsForManifest() {
	cfg.applyDefaults()
}

func (cfg CLIConfig) Validate() error {
	if !cfg.Enabled {
		return nil
	}
	if strings.TrimSpace(cfg.Command) == "" {
		return errors.New("command is required when cli.enabled=true")
	}
	switch cfg.Adapter.Kind {
	case "go_module":
		if cfg.Adapter.ModuleDir == "" {
			return errors.New("adapter.module_dir is required for cli.adapter.kind=go_module")
		}
	default:
		return fmt.Errorf("unsupported cli.adapter.kind %q", cfg.Adapter.Kind)
	}
	switch cfg.Invoke.Kind {
	case "installed_command":
	default:
		return fmt.Errorf("unsupported cli.invoke.kind %q", cfg.Invoke.Kind)
	}
	if strings.TrimSpace(cfg.Invoke.Command) == "" {
		return errors.New("cli.invoke.command is required when cli.enabled=true")
	}
	if cfg.Invoke.Command != cfg.Command {
		return errors.New("cli.invoke.command must match cli.command")
	}
	switch cfg.Artifacts.Manifest.Location {
	case "", CLIArtifactLocationSibling:
	default:
		return fmt.Errorf("unsupported cli.artifacts.manifest.location %q", cfg.Artifacts.Manifest.Location)
	}
	switch cfg.Artifacts.BuildMetadata.Location {
	case "", CLIArtifactLocationSibling:
	default:
		return fmt.Errorf("unsupported cli.artifacts.build_metadata.location %q", cfg.Artifacts.BuildMetadata.Location)
	}
	if cfg.Distribution != nil {
		if cfg.Distribution.Kind != "prebuilt_artifact" {
			return fmt.Errorf("unsupported cli.distribution.kind %q", cfg.Distribution.Kind)
		}
		if cfg.Distribution.ArtifactName == "" {
			return errors.New("distribution.artifact_name is required for prebuilt_artifact")
		}
		if !strings.Contains(cfg.Distribution.ArtifactName, "${os}") || !strings.Contains(cfg.Distribution.ArtifactName, "${arch}") {
			return errors.New("distribution.artifact_name must contain ${os} and ${arch}")
		}
	}
	if cfg.SourceBuild == nil || cfg.SourceBuild.Kind != "go_module" {
		return errors.New("cli.source_build.kind=go_module is required when cli.enabled=true")
	}
	if cfg.Freshness == nil || len(cfg.Freshness.Inputs) == 0 {
		return errors.New("cli.freshness.inputs is required when cli.enabled=true")
	}
	return nil
}

func (deps *Dependencies) UnmarshalJSON(data []byte) error {
	var raw rawDependencies
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	resources, err := decodeDependencyCollection(raw.Resources, "resource")
	if err != nil {
		return fmt.Errorf("resources: %w", err)
	}
	scenarios, err := decodeDependencyCollection(raw.Scenarios, "scenario")
	if err != nil {
		return fmt.Errorf("scenarios: %w", err)
	}

	deps.Resources = resources
	deps.Scenarios = scenarios
	return nil
}

func (deps Dependencies) Validate() error {
	if err := validateDependencyCollection("resources", deps.Resources); err != nil {
		return err
	}
	if err := validateDependencyCollection("scenarios", deps.Scenarios); err != nil {
		return err
	}
	return nil
}

func ResolveScenarioPath(root, name string, env SandboxEnv) (string, bool) {
	defaultPath := scenarioRootPath(root, name)
	if !env.Enabled() || !ScenarioInScope(root, name, env.Scope) {
		return defaultPath, false
	}

	mergedPath := filepath.Clean(ResolveMergedPath(root, name, env.Scope, env.Merged))
	mergedServicePath := filepath.Join(mergedPath, filepath.FromSlash(defaultScenarioServiceRelPath))
	if info, err := os.Stat(mergedPath); err == nil && info.IsDir() {
		if _, err := os.Stat(mergedServicePath); err == nil {
			return mergedPath, true
		}
	}
	if _, err := os.Stat(mergedServicePath); err == nil {
		return mergedPath, true
	}

	return defaultPath, false
}

func decodeDependencyCollection(data json.RawMessage, defaultType string) (map[string]Dependency, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}

	var direct map[string]Dependency
	if err := json.Unmarshal(data, &direct); err != nil {
		return nil, err
	}
	for name, dependency := range direct {
		if strings.TrimSpace(dependency.Type) == "" {
			dependency.Type = defaultType
			direct[name] = dependency
		}
	}
	return direct, nil
}

func (dependency *Dependency) UnmarshalJSON(data []byte) error {
	raw := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*dependency = Dependency{}

	takeString := func(key string, dest *string) error {
		v, ok := raw[key]
		if !ok {
			return nil
		}
		if err := json.Unmarshal(v, dest); err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		delete(raw, key)
		return nil
	}
	takeBool := func(key string, dest *bool) (present bool, err error) {
		v, ok := raw[key]
		if !ok {
			return false, nil
		}
		if err := json.Unmarshal(v, dest); err != nil {
			return true, fmt.Errorf("%s: %w", key, err)
		}
		delete(raw, key)
		return true, nil
	}

	if err := takeString("type", &dependency.Type); err != nil {
		return err
	}
	enabledPresent, err := takeBool("enabled", &dependency.Enabled)
	if err != nil {
		return err
	}
	if !enabledPresent {
		dependency.Enabled = true
	}
	if _, err := takeBool("required", &dependency.Required); err != nil {
		return err
	}
	if err := takeString("startup_policy", &dependency.StartupPolicy); err != nil {
		return err
	}
	if err := takeString("freshness_policy", &dependency.FreshnessPolicy); err != nil {
		return err
	}
	if err := takeString("degraded_behavior", &dependency.DegradedBehavior); err != nil {
		return err
	}
	if err := takeString("purpose", &dependency.Purpose); err != nil {
		return err
	}
	if err := takeString("description", &dependency.Description); err != nil {
		return err
	}
	if err := takeString("database", &dependency.Database); err != nil {
		return err
	}
	if err := takeString("versionRange", &dependency.VersionRange); err != nil {
		return err
	}
	if _, err := takeBool("runtime_only", &dependency.RuntimeOnly); err != nil {
		return err
	}
	if err := takeString("runtime_only_rationale", &dependency.RuntimeOnlyRationale); err != nil {
		return err
	}
	if err := takeString("bundle_policy", &dependency.BundlePolicy); err != nil {
		return err
	}
	if v, ok := raw["bindings"]; ok {
		return fmt.Errorf("bindings are no longer supported; resolve scenario addresses through discovery: %s", string(v))
	}
	if v, ok := raw["config"]; ok {
		var cfg map[string]json.RawMessage
		if err := json.Unmarshal(v, &cfg); err != nil {
			return fmt.Errorf("config: %w", err)
		}
		if cfg == nil {
			return fmt.Errorf("config: must be a JSON object")
		}
		dependency.Config = append(dependency.Config[:0], v...)
		delete(raw, "config")
	}

	dependency.Type = strings.TrimSpace(dependency.Type)
	dependency.StartupPolicy = strings.TrimSpace(dependency.StartupPolicy)
	dependency.FreshnessPolicy = strings.TrimSpace(dependency.FreshnessPolicy)
	dependency.DegradedBehavior = strings.TrimSpace(dependency.DegradedBehavior)
	dependency.Purpose = strings.TrimSpace(dependency.Purpose)
	dependency.Description = strings.TrimSpace(dependency.Description)
	dependency.Database = strings.TrimSpace(dependency.Database)
	dependency.VersionRange = strings.TrimSpace(dependency.VersionRange)
	dependency.RuntimeOnlyRationale = strings.TrimSpace(dependency.RuntimeOnlyRationale)
	dependency.BundlePolicy = strings.TrimSpace(dependency.BundlePolicy)

	if len(raw) > 0 {
		cfg := map[string]json.RawMessage{}
		if len(dependency.Config) > 0 {
			if err := json.Unmarshal(dependency.Config, &cfg); err != nil {
				return fmt.Errorf("config: %w", err)
			}
		}
		for key, value := range raw {
			cfg[key] = value
		}
		encoded, err := json.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("config: %w", err)
		}
		dependency.Config = encoded
	}
	return nil
}

// MarshalJSON emits typed fields and the dependency-specific Config object.
// The `enabled` key is always emitted (default is true when absent on input).
func (dependency Dependency) MarshalJSON() ([]byte, error) {
	out := map[string]json.RawMessage{}
	if len(dependency.Config) > 0 {
		var cfg map[string]json.RawMessage
		if err := json.Unmarshal(dependency.Config, &cfg); err != nil {
			return nil, fmt.Errorf("dependency config: %w", err)
		}
		if cfg == nil {
			return nil, fmt.Errorf("dependency config: must be a JSON object")
		}
	}

	emit := func(key string, v any) error {
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("%s: %w", key, err)
		}
		out[key] = b
		return nil
	}
	emitIfNonEmpty := func(key string, s string) error {
		if s == "" {
			return nil
		}
		return emit(key, s)
	}
	emitIfTrue := func(key string, b bool) error {
		if !b {
			return nil
		}
		return emit(key, b)
	}

	if err := emitIfNonEmpty("type", dependency.Type); err != nil {
		return nil, err
	}
	// Matches legacy `omitempty` behavior: false drops; the reader defaults
	// a missing key back to true.
	if err := emitIfTrue("enabled", dependency.Enabled); err != nil {
		return nil, err
	}
	if err := emitIfTrue("required", dependency.Required); err != nil {
		return nil, err
	}
	if err := emitIfNonEmpty("startup_policy", dependency.StartupPolicy); err != nil {
		return nil, err
	}
	if err := emitIfNonEmpty("freshness_policy", dependency.FreshnessPolicy); err != nil {
		return nil, err
	}
	if err := emitIfNonEmpty("degraded_behavior", dependency.DegradedBehavior); err != nil {
		return nil, err
	}
	if err := emitIfNonEmpty("purpose", dependency.Purpose); err != nil {
		return nil, err
	}
	if err := emitIfNonEmpty("description", dependency.Description); err != nil {
		return nil, err
	}
	if err := emitIfNonEmpty("database", dependency.Database); err != nil {
		return nil, err
	}
	if err := emitIfNonEmpty("versionRange", dependency.VersionRange); err != nil {
		return nil, err
	}
	if err := emitIfTrue("runtime_only", dependency.RuntimeOnly); err != nil {
		return nil, err
	}
	if err := emitIfNonEmpty("runtime_only_rationale", dependency.RuntimeOnlyRationale); err != nil {
		return nil, err
	}
	if err := emitIfNonEmpty("bundle_policy", dependency.BundlePolicy); err != nil {
		return nil, err
	}
	if len(dependency.Config) > 0 {
		out["config"] = append([]byte(nil), dependency.Config...)
	}

	return json.Marshal(out)
}

func (dependency Dependency) NormalizedStartupPolicy() string {
	if !dependency.Enabled {
		return DependencyStartupPolicyIgnore
	}
	switch strings.TrimSpace(dependency.StartupPolicy) {
	case "":
		if dependency.Required {
			return DependencyStartupPolicyMustStart
		}
		return DependencyStartupPolicyIgnore
	case DependencyStartupPolicyMustStart, DependencyStartupPolicyTryStart, DependencyStartupPolicyIgnore:
		return strings.TrimSpace(dependency.StartupPolicy)
	default:
		return strings.TrimSpace(dependency.StartupPolicy)
	}
}

// NormalizedFreshnessPolicy returns the effective freshness policy, defaulting
// to restart_when_stale when unset. Unlike startup_policy it is never derived
// from another field — the two axes (availability vs disruption-tolerance) are
// kept orthogonal so editing one never silently changes the other.
func (dependency Dependency) NormalizedFreshnessPolicy() string {
	switch strings.TrimSpace(dependency.FreshnessPolicy) {
	case "":
		return DependencyFreshnessPolicyRestartWhenStale
	default:
		return strings.TrimSpace(dependency.FreshnessPolicy)
	}
}

func validateDependencyCollection(kind string, dependencies map[string]Dependency) error {
	for name, dependency := range dependencies {
		if err := dependency.Validate(kind, name); err != nil {
			return err
		}
	}
	return nil
}

func (dependency Dependency) Validate(kind, name string) error {
	if dependency.RuntimeOnly && strings.TrimSpace(dependency.RuntimeOnlyRationale) == "" {
		return fmt.Errorf("%s.%s.runtime_only requires runtime_only_rationale", kind, name)
	}
	policy := dependency.NormalizedStartupPolicy()
	if !dependency.Enabled {
		return nil
	}
	switch policy {
	case DependencyStartupPolicyMustStart, DependencyStartupPolicyTryStart, DependencyStartupPolicyIgnore:
	default:
		return fmt.Errorf("%s.%s.startup_policy must be one of %q, %q, or %q; got %q",
			kind, name,
			DependencyStartupPolicyMustStart,
			DependencyStartupPolicyTryStart,
			DependencyStartupPolicyIgnore,
			dependency.StartupPolicy,
		)
	}
	if dependency.Required && policy == DependencyStartupPolicyIgnore {
		return fmt.Errorf("%s.%s is required but resolves to startup_policy=%q", kind, name, policy)
	}
	if dependency.Required && policy == DependencyStartupPolicyTryStart && dependency.DegradedBehavior == "" {
		return fmt.Errorf("%s.%s is required with startup_policy=%q and must declare degraded_behavior", kind, name, policy)
	}
	if raw := strings.TrimSpace(dependency.FreshnessPolicy); raw != "" {
		switch raw {
		case DependencyFreshnessPolicyRestartWhenStale, DependencyFreshnessPolicyReuseRunning, DependencyFreshnessPolicyRebuildOnly:
		default:
			return fmt.Errorf("%s.%s.freshness_policy must be one of %q, %q, or %q; got %q",
				kind, name,
				DependencyFreshnessPolicyRestartWhenStale,
				DependencyFreshnessPolicyReuseRunning,
				DependencyFreshnessPolicyRebuildOnly,
				dependency.FreshnessPolicy,
			)
		}
		// An ignore edge is never touched, so a freshness policy on it is a
		// contradiction (and almost certainly a mistake).
		if policy == DependencyStartupPolicyIgnore {
			return fmt.Errorf("%s.%s sets freshness_policy=%q but resolves to startup_policy=%q (an ignored dependency is never started or restarted)", kind, name, raw, policy)
		}
	}
	return nil
}

func ScenarioInScope(root, name, scope string) bool {
	scope = normalizeSandboxScope(scope)
	if contractPaths.IsFullRepoScope(root, scope) {
		return true
	}

	scenarioDir := strings.Trim(strings.TrimSpace(filepath.ToSlash(contractPaths.ScenarioDirName(root))), "/")
	if scope == scenarioDir {
		return true
	}

	prefix := strings.Trim(strings.TrimSpace(filepath.ToSlash(contractPaths.ScenarioScopePrefix(root))), "/")
	if prefix == "" {
		prefix = scenarioDir
	}
	if !strings.HasPrefix(scope, prefix+"/") {
		return false
	}

	scopedName := strings.TrimPrefix(scope, prefix+"/")
	scopedName = strings.SplitN(scopedName, "/", 2)[0]
	return name == scopedName
}

func ResolveMergedPath(root, name, scope, merged string) string {
	scope = normalizeSandboxScope(scope)
	merged = filepath.Clean(merged)
	scenarioRel := filepath.ToSlash(filepath.Join(contractPaths.ScenarioDirName(root), name))

	if contractPaths.IsFullRepoScope(root, scope) {
		return filepath.Join(merged, filepath.FromSlash(scenarioRel))
	}

	if scenarioRel == scope {
		return merged
	}

	if strings.HasPrefix(scenarioRel, scope+"/") {
		relative := strings.TrimPrefix(scenarioRel, scope+"/")
		return filepath.Join(merged, filepath.FromSlash(relative))
	}

	return filepath.Join(merged, filepath.FromSlash(scenarioRel))
}

func (manifest ServiceManifest) HealthConfig() *HealthConfig {
	if manifest.Lifecycle.Health != nil {
		return manifest.Lifecycle.Health
	}
	return manifest.Health
}

func (manifest ServiceManifest) PortEnvVars() []string {
	ports := manifest.SortedPorts()
	keys := make([]string, 0, len(ports))
	for _, port := range ports {
		if port.EnvVar != "" {
			keys = append(keys, port.EnvVar)
		}
	}
	return keys
}

func (manifest ServiceManifest) PortEnvVar(portName string) string {
	definition, ok := manifest.Ports[portName]
	if !ok {
		return ""
	}
	if definition.EnvVar != "" {
		return definition.EnvVar
	}
	return strings.ToUpper(strings.ReplaceAll(portName, "-", "_")) + "_PORT"
}

func (manifest ServiceManifest) SortedPorts() []PortSummary {
	names := make([]string, 0, len(manifest.Ports))
	for name := range manifest.Ports {
		names = append(names, name)
	}
	sort.Strings(names)

	ports := make([]PortSummary, 0, len(names))
	for _, name := range names {
		definition := manifest.Ports[name]
		envVar := definition.EnvVar
		if envVar == "" {
			envVar = strings.ToUpper(strings.ReplaceAll(name, "-", "_")) + "_PORT"
		}
		ports = append(ports, PortSummary{
			Name:        name,
			EnvVar:      envVar,
			Description: definition.Description,
			Range:       definition.Range,
			FixedPort:   definition.Port,
		})
	}
	return ports
}

func (manifest ServiceManifest) PhaseSummaries() []PhaseSummary {
	phases := []struct {
		name  string
		phase Phase
	}{
		{name: "setup", phase: manifest.Lifecycle.Setup},
		{name: "develop", phase: manifest.Lifecycle.Develop},
		{name: "build", phase: manifest.Lifecycle.Build},
		{name: "deploy", phase: manifest.Lifecycle.Deploy},
		{name: "clean", phase: manifest.Lifecycle.Clean},
		{name: "backup", phase: manifest.Lifecycle.Backup},
		{name: "restore", phase: manifest.Lifecycle.Restore},
		{name: "production", phase: manifest.Lifecycle.Production},
		{name: "stop", phase: manifest.Lifecycle.Stop},
	}

	summaries := make([]PhaseSummary, 0, len(phases))
	for _, phase := range phases {
		defined := len(phase.phase.Steps) > 0 || phase.phase.Description != ""
		summaries = append(summaries, PhaseSummary{
			Name:        phase.name,
			Description: phase.phase.Description,
			Steps:       len(phase.phase.Steps),
			Defined:     defined,
		})
	}
	return summaries
}

// ExpandTemplate resolves the manifest's closed ${NAME}/$NAME placeholder
// language. Every value must be present before expansion; shell defaults and
// dotted expression languages are deliberately rejected.
func ExpandTemplate(value string, environment map[string]string) (string, error) {
	var output strings.Builder
	for cursor := 0; cursor < len(value); {
		dollar := strings.IndexByte(value[cursor:], '$')
		if dollar < 0 {
			output.WriteString(value[cursor:])
			break
		}
		dollar += cursor
		output.WriteString(value[cursor:dollar])
		nameStart := dollar + 1
		if nameStart >= len(value) {
			return "", fmt.Errorf("invalid placeholder at byte %d", dollar)
		}
		nameEnd := nameStart
		if value[nameStart] == '{' {
			nameStart++
			closing := strings.IndexByte(value[nameStart:], '}')
			if closing < 0 {
				return "", fmt.Errorf("unterminated placeholder at byte %d", dollar)
			}
			nameEnd = nameStart + closing
			cursor = nameEnd + 1
		} else {
			for nameEnd < len(value) && isTemplateIdentifierByte(value[nameEnd], nameEnd == nameStart) {
				nameEnd++
			}
			cursor = nameEnd
		}
		name := value[nameStart:nameEnd]
		if !validTemplateIdentifier(name) {
			return "", fmt.Errorf("invalid placeholder %q", name)
		}
		resolved, ok := environment[name]
		if !ok {
			return "", fmt.Errorf("unresolved placeholder %s", name)
		}
		output.WriteString(resolved)
	}
	return output.String(), nil
}

func validTemplateIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for index := 0; index < len(value); index++ {
		if !isTemplateIdentifierByte(value[index], index == 0) {
			return false
		}
	}
	return true
}

func isTemplateIdentifierByte(value byte, first bool) bool {
	if value == '_' || value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z' {
		return true
	}
	return !first && value >= '0' && value <= '9'
}

func ExpandHealthTarget(target string, ports map[string]int) (string, error) {
	environment := make(map[string]string, len(ports))
	for key, port := range ports {
		environment[key] = strconv.Itoa(port)
	}
	return ExpandTemplate(target, environment)
}

func EvaluateHealth(health *HealthConfig, ports map[string]int) string {
	if health == nil || len(health.Checks) == 0 {
		return "running"
	}

	criticalFailure := false
	nonCriticalFailure := false
	for _, check := range health.Checks {
		if err := PerformHealthCheck(check, ports); err != nil {
			if check.Critical {
				criticalFailure = true
			} else {
				nonCriticalFailure = true
			}
		}
	}

	switch {
	case criticalFailure:
		return "unhealthy"
	case nonCriticalFailure:
		return "degraded"
	default:
		return "healthy"
	}
}

func PerformHealthCheck(check HealthCheck, ports map[string]int) error {
	switch strings.TrimSpace(check.Type) {
	case "", "http":
		target, err := ExpandHealthTarget(check.Target, ports)
		if err != nil {
			return err
		}
		if _, err := url.Parse(target); err != nil {
			return fmt.Errorf("invalid URL %q: %w", target, err)
		}

		timeout := time.Duration(check.Timeout) * time.Millisecond
		if timeout == 0 {
			timeout = 5 * time.Second
		}

		client := &http.Client{Timeout: timeout}
		resp, err := client.Get(target)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		return nil
	case "connect_rpc":
		// Connect-RPC probe: POST {} with application/json. Matches the wire
		// shape generated *_v1connect handlers expect for unary unary calls
		// and lets scenarios that have migrated their health domain off
		// REST keep using the standard lifecycle.checks config.
		target, err := ExpandHealthTarget(check.Target, ports)
		if err != nil {
			return err
		}
		if _, err := url.Parse(target); err != nil {
			return fmt.Errorf("invalid URL %q: %w", target, err)
		}

		timeout := time.Duration(check.Timeout) * time.Millisecond
		if timeout == 0 {
			timeout = 5 * time.Second
		}

		client := &http.Client{Timeout: timeout}
		req, err := http.NewRequest(http.MethodPost, target, bytes.NewReader([]byte("{}")))
		if err != nil {
			return fmt.Errorf("build connect_rpc request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		return nil
	case "postgres":
		timeout := time.Duration(check.Timeout) * time.Millisecond
		if timeout == 0 {
			timeout = 3 * time.Second
		}

		address := "127.0.0.1:5432"
		if parsed, err := parsePostgresAddress(check.Target); err == nil && parsed != "" {
			address = parsed
		}

		conn, err := net.DialTimeout("tcp", address, timeout)
		if err != nil {
			return err
		}
		_ = conn.Close()
		return nil
	default:
		return fmt.Errorf("unsupported health check type %q", check.Type)
	}
}

func parsePostgresAddress(target string) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", nil
	}

	if strings.HasPrefix(target, "postgres://") || strings.HasPrefix(target, "postgresql://") {
		parsed, err := url.Parse(target)
		if err != nil {
			return "", err
		}
		host := parsed.Hostname()
		if host == "" {
			return "", nil
		}
		port := parsed.Port()
		if port == "" {
			port = "5432"
		}
		return net.JoinHostPort(host, port), nil
	}

	if strings.Contains(target, ":") {
		host, port, err := net.SplitHostPort(target)
		if err == nil && host != "" && port != "" {
			return net.JoinHostPort(host, port), nil
		}
		if err != nil {
			return "", err
		}
	}

	return "", nil
}

func scanScenarioNames(baseDir string) ([]string, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		servicePath := filepath.Join(baseDir, entry.Name(), filepath.FromSlash(defaultScenarioServiceRelPath))
		if _, err := os.Stat(servicePath); err == nil {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// scanSandboxScenarioNames mirrors the bash sandbox discovery contract: the
// merged dir can represent the repo root, the scenarios directory, or one
// specific scenario depending on the active sandbox scope.
func scanSandboxScenarioNames(root string, env SandboxEnv) ([]string, error) {
	if !env.Enabled() {
		return nil, nil
	}
	if info, err := os.Stat(env.Merged); err != nil || !info.IsDir() {
		return nil, nil
	}
	scope := normalizeSandboxScope(env.Scope)
	scenarioDir := strings.Trim(strings.TrimSpace(filepath.ToSlash(contractPaths.ScenarioDirName(root))), "/")
	prefix := strings.Trim(strings.TrimSpace(filepath.ToSlash(contractPaths.ScenarioScopePrefix(root))), "/")
	if prefix == "" {
		prefix = scenarioDir
	}
	switch {
	case contractPaths.IsFullRepoScope(root, scope):
		return scanScenarioNames(filepath.Join(env.Merged, filepath.FromSlash(scenarioDir)))
	case scope == scenarioDir:
		return scanScenarioNames(env.Merged)
	case strings.HasPrefix(scope, prefix+"/"):
		name := strings.TrimPrefix(scope, prefix+"/")
		name = strings.SplitN(name, "/", 2)[0]
		resolved := ResolveMergedPath(root, name, env.Scope, env.Merged)
		if _, err := os.Stat(filepath.Join(resolved, filepath.FromSlash(defaultScenarioServiceRelPath))); err == nil {
			return []string{name}, nil
		}
	}

	return nil, nil
}

func normalizeSandboxScope(scope string) string {
	scope = strings.TrimSpace(filepath.ToSlash(scope))
	scope = strings.TrimSuffix(scope, "/")
	return scope
}

func scenarioBaseDir(root string) string {
	return contractPaths.ScenarioBaseDir(root)
}

func scenarioRootPath(root, name string) string {
	return contractPaths.ScenarioRootPath(root, name)
}

func scenarioServicePath(root, name, scenarioPath string) string {
	return contractPaths.ScenarioServicePath(root, name, scenarioPath)
}
