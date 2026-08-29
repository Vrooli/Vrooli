package scenario

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/repocontractmeta"
)

const (
	validationParameterA           = 2
	scenarioInvokeInstalledCommand = "installed_command"
	scenarioStatusUnhealthy        = "unhealthy"
	scenarioStatusRunning          = "running"
)

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
		cfg.Invoke.Kind = scenarioInvokeInstalledCommand
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
	if err := validateDependencyCollection(repocontractmeta.ScenarioDir, deps.Scenarios); err != nil {
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

//nolint:gocyclo // dependency decoding maintains legacy aliases, validation, and unknown-field behavior.
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
//
//nolint:gocyclo // dependency encoding preserves canonical field names and compatibility projections.
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
	scopedName = strings.SplitN(scopedName, "/", validationParameterA)[0]
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
	slices.Sort(names)

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
		return scenarioStatusRunning
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
		return scenarioStatusUnhealthy
	case nonCriticalFailure:
		return "degraded"
	default:
		return "healthy"
	}
}
