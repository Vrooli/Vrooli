package scenario

import (
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

	"github.com/vrooli/vrooli/internal/hostreqspec"
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
}

type ServiceManifest struct {
	Schema         string                    `json:"$schema,omitempty"`
	Version        string                    `json:"version,omitempty"`
	Service        ServiceMetadata           `json:"service"`
	Ports          map[string]Port           `json:"ports,omitempty"`
	Lifecycle      Lifecycle                 `json:"lifecycle,omitempty"`
	Health         *HealthConfig             `json:"health,omitempty"`
	Dependencies   Dependencies              `json:"dependencies,omitempty"`
	Environment    map[string]string         `json:"environment,omitempty"`
	HostTools      []hostreqspec.Declaration `json:"hostTools,omitempty"`
	HostSafeguards []hostreqspec.Declaration `json:"hostSafeguards,omitempty"`
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

type Dependencies struct {
	Resources map[string]Dependency `json:"resources,omitempty"`
	Scenarios map[string]Dependency `json:"scenarios,omitempty"`
}

type Dependency struct {
	Type             string `json:"type,omitempty"`
	Enabled          bool   `json:"enabled,omitempty"`
	Required         bool   `json:"required,omitempty"`
	StartupPolicy    string `json:"startup_policy,omitempty"`
	DegradedBehavior string `json:"degraded_behavior,omitempty"`
	Purpose          string `json:"purpose,omitempty"`
	Description      string `json:"description,omitempty"`
	Database         string `json:"database,omitempty"`
}

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
	Health     *HealthConfig `json:"health,omitempty"`
	Setup      Phase         `json:"setup,omitempty"`
	Develop    Phase         `json:"develop,omitempty"`
	Build      Phase         `json:"build,omitempty"`
	Deploy     Phase         `json:"deploy,omitempty"`
	Clean      Phase         `json:"clean,omitempty"`
	Test       Phase         `json:"test,omitempty"`
	Backup     Phase         `json:"backup,omitempty"`
	Restore    Phase         `json:"restore,omitempty"`
	VersionCmd Phase         `json:"version,omitempty"`
	Production Phase         `json:"production,omitempty"`
	Stop       Phase         `json:"stop,omitempty"`
}

type Phase struct {
	Description string      `json:"description,omitempty"`
	Condition   *Condition  `json:"condition,omitempty"`
	Steps       []PhaseStep `json:"steps,omitempty"`
}

type PhaseStep struct {
	Name        string     `json:"name,omitempty"`
	Run         string     `json:"run,omitempty"`
	Description string     `json:"description,omitempty"`
	Background  bool       `json:"background,omitempty"`
	Condition   *Condition `json:"condition,omitempty"`
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
	names := make(map[string]struct{})

	canonicalNames, err := scanScenarioNames(scenarioBaseDir(root))
	if err != nil {
		return nil, err
	}
	for _, name := range canonicalNames {
		names[name] = struct{}{}
	}

	sandboxNames, err := scanSandboxScenarioNames(root, env)
	if err != nil {
		return nil, err
	}
	for _, name := range sandboxNames {
		names[name] = struct{}{}
	}

	ordered := make([]string, 0, len(names))
	for name := range names {
		ordered = append(ordered, name)
	}
	sort.Strings(ordered)

	scenarios := make([]Scenario, 0, len(ordered))
	for _, name := range ordered {
		scenario, err := Load(root, name, env)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			return nil, fmt.Errorf("load scenario %s: %w", name, err)
		}
		scenarios = append(scenarios, scenario)
	}

	return scenarios, nil
}

func ReadService(path string) (ServiceManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ServiceManifest{}, err
	}

	var manifest ServiceManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return ServiceManifest{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if manifest.Lifecycle.Health == nil && manifest.Health != nil {
		manifest.Lifecycle.Health = manifest.Health
	}
	if err := hostreqspec.ValidateDeclarations(hostreqspec.KindTool, manifest.HostTools); err != nil {
		return ServiceManifest{}, fmt.Errorf("validate hostTools in %s: %w", path, err)
	}
	if err := hostreqspec.ValidateDeclarations(hostreqspec.KindSafeguard, manifest.HostSafeguards); err != nil {
		return ServiceManifest{}, fmt.Errorf("validate hostSafeguards in %s: %w", path, err)
	}
	return manifest, nil
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
		{name: "test", phase: manifest.Lifecycle.Test},
		{name: "backup", phase: manifest.Lifecycle.Backup},
		{name: "restore", phase: manifest.Lifecycle.Restore},
		{name: "version", phase: manifest.Lifecycle.VersionCmd},
		{name: "production", phase: manifest.Lifecycle.Production},
		{name: "stop", phase: manifest.Lifecycle.Stop},
	}

	summaries := make([]PhaseSummary, 0, len(phases))
	for _, phase := range phases {
		defined := len(phase.phase.Steps) > 0 || phase.phase.Description != "" || phase.phase.Condition != nil
		summaries = append(summaries, PhaseSummary{
			Name:        phase.name,
			Description: phase.phase.Description,
			Steps:       len(phase.phase.Steps),
			Defined:     defined,
		})
	}
	return summaries
}

func ExpandTarget(target string, ports map[string]int) string {
	expanded := target
	for key, port := range ports {
		value := strconv.Itoa(port)
		expanded = strings.ReplaceAll(expanded, "${"+key+"}", value)
		expanded = strings.ReplaceAll(expanded, "$"+key, value)
	}
	return expanded
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
		target := ExpandTarget(check.Target, ports)
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
