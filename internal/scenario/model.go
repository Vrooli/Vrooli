package scenario

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
	// Outputs declares additional artifacts produced by the same component
	// build. Entry and Output remain the primary target for compatibility;
	// additional targets are built and fingerprinted by the lifecycle as part
	// of the same typed component contract.
	Outputs []ComponentBuildOutput `json:"outputs,omitempty"`
}

type ComponentBuildOutput struct {
	Entry  string `json:"entry,omitempty"`
	Output string `json:"output"`
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

// LoadServiceManifest reads the shared typed service-manifest shape without
// applying runtime validation or defaults. Consumers that only need declared
// fields use this path so newer, unknown fields remain forward-compatible.
func LoadServiceManifest(path string) (ServiceManifest, error) {
	_, manifest, err := loadServiceManifest(path)
	return manifest, err
}

func loadServiceManifest(path string) ([]byte, ServiceManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, ServiceManifest{}, err
	}
	var manifest ServiceManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, ServiceManifest{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return data, manifest, nil
}

func ReadService(path string) (ServiceManifest, error) {
	data, manifest, err := loadServiceManifest(path)
	if err != nil {
		return ServiceManifest{}, err
	}
	if err := repocontract.ValidateCredentialDescriptorUniqueness(data, path); err != nil {
		return ServiceManifest{}, fmt.Errorf("validate credentials in %s: %w", path, err)
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
