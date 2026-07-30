package env

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	runtimestorage "github.com/vrooli/vrooli/internal/resources/runtime/storage"
	"github.com/vrooli/vrooli/internal/resources/securestore"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	credentialauthority "github.com/vrooli/vrooli/internal/secrets"
)

const (
	portRegistryJSONPath = "scripts/resources/port_registry.json"
)

var templatePattern = regexp.MustCompile(`\$\{([A-Z0-9_]+)\}`)

type PortRegistry struct {
	ResourcePorts  map[string]int    `json:"resource_ports"`
	ReservedRanges map[string]string `json:"reserved_ranges"`
}

type ResolveOptions struct {
	ScenarioName string
	// Variant selects the instance whose namespace the resolved values address.
	// Empty ⇒ live, so the Postgres database name stays "vrooli_<scenario>" for
	// the canonical instance and becomes "vrooli_<scenario>_<variant>" for a
	// shadow — both derived from the InstanceKey SSOT. See Baseline Modes §1a.
	Variant    string
	Dependency scenario.Dependency
}

type ResourceReport struct {
	Name     string            `json:"name"`
	Manifest string            `json:"manifest_path,omitempty"`
	Values   map[string]string `json:"values,omitempty"`
	Warnings []string          `json:"warnings,omitempty"`
}

type ScenarioResolution struct {
	Values    map[string]string `json:"values"`
	Resources []ResourceReport  `json:"resources,omitempty"`
	Warnings  []string          `json:"warnings,omitempty"`
}

func ResolveScenario(root, home, scenarioName, variant string, manifest scenario.ServiceManifest) (ScenarioResolution, error) {
	report := ScenarioResolution{
		Values:    map[string]string{},
		Resources: []ResourceReport{},
		Warnings:  []string{},
	}
	owners := map[string]string{}

	resourceNames := make([]string, 0, len(manifest.Dependencies.Resources))
	for name, dep := range manifest.Dependencies.Resources {
		if dep.Enabled || dep.Required {
			resourceNames = append(resourceNames, name)
		}
	}
	slices.Sort(resourceNames)

	for _, resourceName := range resourceNames {
		dep := manifest.Dependencies.Resources[resourceName]
		result, err := ResolveResource(root, home, resourceName, ResolveOptions{
			ScenarioName: scenarioName,
			Variant:      variant,
			Dependency:   dep,
		})
		if err != nil {
			return ScenarioResolution{}, err
		}
		report.Resources = append(report.Resources, result)
		for key, value := range result.Values {
			if owner, exists := owners[key]; exists && report.Values[key] != value {
				return ScenarioResolution{}, fmt.Errorf("resource env collision for %s: %s and %s export different values", key, owner, resourceName)
			}
			report.Values[key] = value
			owners[key] = resourceName
		}
		report.Warnings = append(report.Warnings, result.Warnings...)
	}

	return report, nil
}

func ResolveResource(root, home, resourceName string, opts ResolveOptions) (ResourceReport, error) {
	manifestPath := manifestpkg.DefaultPath(root, resourceName)
	if _, err := os.Stat(manifestPath); err != nil {
		if os.IsNotExist(err) {
			return ResourceReport{Name: resourceName, Values: map[string]string{}}, nil
		}
		return ResourceReport{}, err
	}

	resourceManifest, err := manifestpkg.Load(manifestPath)
	if err != nil {
		return ResourceReport{}, err
	}

	values, warnings, err := resolveFromManifest(root, home, resourceManifest, opts)
	if err != nil {
		return ResourceReport{}, err
	}

	return ResourceReport{
		Name:     resourceName,
		Manifest: manifestPath,
		Values:   values,
		Warnings: warnings,
	}, nil
}

func ResolveCredentialValues(root, home string, resourceManifest manifestpkg.ResourceManifest) (map[string]string, error) {
	_ = root
	_ = home
	values := map[string]string{}
	descriptors := resourceManifest.Credentials.All()
	if len(descriptors) == 0 {
		return values, nil
	}
	authority, err := credentialauthority.NewAuthority(securestore.Default())
	if err != nil {
		return nil, fmt.Errorf("resolve %s credentials: %w", resourceManifest.Name, err)
	}
	for _, descriptor := range descriptors {
		identity, err := credentialauthority.ParseIdentity(descriptor.LogicalID)
		if err != nil {
			return nil, fmt.Errorf("resolve %s credential %s: %w", resourceManifest.Name, descriptor.Env, err)
		}
		field := strings.TrimSpace(descriptor.Field)
		if field == "" {
			field = "value"
		}
		if err := authority.Inject(identity, field, descriptor.Env, values); err != nil {
			if !descriptor.Required && errors.Is(err, credentialauthority.ErrUnconfigured) {
				continue
			}
			return nil, fmt.Errorf("resolve %s credential %s: %w", resourceManifest.Name, descriptor.Env, err)
		}
	}
	return values, nil
}

func MissingCredentialKeys(root, home string, resourceManifest manifestpkg.ResourceManifest) ([]string, error) {
	if len(resourceManifest.Credentials.All()) == 0 {
		return nil, nil
	}
	resolved, err := ResolveCredentialValues(root, home, resourceManifest)
	if err != nil {
		return nil, err
	}

	missing := make([]string, 0)
	for _, descriptor := range resourceManifest.Credentials.All() {
		name := strings.TrimSpace(descriptor.Env)
		if name == "" {
			continue
		}
		if strings.TrimSpace(resolved[name]) == "" {
			missing = append(missing, name)
		}
	}
	return missing, nil
}

func resolveFromManifest(root, home string, resourceManifest manifestpkg.ResourceManifest, opts ResolveOptions) (map[string]string, []string, error) {
	values := map[string]string{}
	warnings := []string{}
	templateContext := buildTemplateContext(root, home, resourceManifest.Name)

	for key, value := range resourceManifest.EnvironmentExports.Static {
		values[key] = expandTemplateWithContext(value, values, templateContext)
	}

	hostPorts, err := loadHostPorts(root, resourceManifest)
	if err != nil {
		return nil, nil, err
	}
	for key, portName := range resourceManifest.EnvironmentExports.FromPorts {
		resolved, ok := hostPorts[strings.TrimSpace(portName)]
		if !ok {
			return nil, nil, fmt.Errorf("resource %s environment_exports.from_ports[%s] references unknown port %q", resourceManifest.Name, key, portName)
		}
		values[key] = strconv.Itoa(resolved)
	}

	runtimeValues, runtimeWarnings, err := resolveRequestedEnvValues(root, home, resourceManifest, resourceManifest.EnvironmentExports.FromRuntimeEnv, values)
	if err != nil {
		return nil, nil, err
	}
	for key, value := range runtimeValues {
		values[key] = value
	}
	warnings = append(warnings, runtimeWarnings...)

	// Credentials are the authoritative source for their declared process
	// variables. Apply them after manifest defaults so a persisted resource can
	// safely outlive a changed bootstrap value in its image configuration.
	credentialValues, err := ResolveCredentialValues(root, home, resourceManifest)
	if err != nil {
		return nil, nil, err
	}
	for key, value := range credentialValues {
		values[key] = value
	}

	applyDependencyOverrides(resourceManifest.Name, opts, values)

	if len(resourceManifest.EnvironmentExports.Static) == 0 &&
		len(resourceManifest.EnvironmentExports.FromPorts) == 0 &&
		len(resourceManifest.EnvironmentExports.FromRuntimeEnv) == 0 &&
		len(resourceManifest.EnvironmentExports.Derived) == 0 {
		return values, warnings, nil
	} else {
		for key, derived := range resourceManifest.EnvironmentExports.Derived {
			values[key] = expandTemplateWithContext(derived.Template, values, templateContext)
		}
	}

	applyFallbackDefaults(resourceManifest.Name, values, hostPorts)
	applyDependencyOverrides(resourceManifest.Name, opts, values)
	for key, derived := range resourceManifest.EnvironmentExports.Derived {
		values[key] = expandTemplateWithContext(derived.Template, values, templateContext)
	}

	return values, warnings, nil
}

func resolveRequestedEnvValues(
	root, home string,
	resourceManifest manifestpkg.ResourceManifest,
	requested []string,
	baseValues map[string]string,
) (map[string]string, []string, error) {
	values := map[string]string{}
	warnings := []string{}

	templateContext := buildTemplateContext(root, home, resourceManifest.Name)

	for _, key := range requested {
		name := strings.TrimSpace(key)
		if name == "" {
			continue
		}
		if value, ok := resourceManifest.Runtime.Env[name]; ok {
			values[name] = expandTemplateWithContext(value, baseValues, templateContext)
			continue
		}
		warnings = append(warnings, fmt.Sprintf("%s environment export %s was requested but no runtime or secret value was found", resourceManifest.Name, name))
	}

	return values, warnings, nil
}

func ValidateResourceManifest(root string, resourceManifest manifestpkg.ResourceManifest) []string {
	issues := []string{}
	portNames := map[string]struct{}{}
	hostPorts := map[string]string{}
	for _, port := range resourceManifest.Ports {
		if name := strings.TrimSpace(port.Name); name != "" {
			if _, exists := portNames[name]; exists {
				issues = append(issues, fmt.Sprintf("duplicate port name %q", name))
			}
			portNames[name] = struct{}{}
		}
		if port.Host > 0 {
			key := hostPortProtocolKey(port)
			if prior, exists := hostPorts[key]; exists {
				issues = append(issues, fmt.Sprintf("duplicate host port %s for %s and %s", key, prior, defaultPortLabel(port)))
			}
			hostPorts[key] = defaultPortLabel(port)
		}
	}

	exports := map[string]string{}
	for key := range resourceManifest.EnvironmentExports.Static {
		exports[key] = "static"
	}
	for key, portName := range resourceManifest.EnvironmentExports.FromPorts {
		if previous, exists := exports[key]; exists {
			issues = append(issues, fmt.Sprintf("environment export %s declared in both %s and from_ports", key, previous))
		}
		exports[key] = "from_ports"
		if _, exists := portNames[strings.TrimSpace(portName)]; !exists {
			issues = append(issues, fmt.Sprintf("environment_exports.from_ports[%s] references missing port %q", key, portName))
		}
	}
	for _, key := range resourceManifest.EnvironmentExports.FromRuntimeEnv {
		if previous, exists := exports[key]; exists {
			issues = append(issues, fmt.Sprintf("environment export %s declared in both %s and from_runtime_env", key, previous))
		}
		exports[key] = "from_runtime_env"
	}
	baseValues := map[string]string{}
	for key, value := range resourceManifest.EnvironmentExports.Static {
		baseValues[key] = value
	}
	hostPortValues, err := loadHostPorts(root, resourceManifest)
	if err == nil {
		for key, portName := range resourceManifest.EnvironmentExports.FromPorts {
			if port, ok := hostPortValues[portName]; ok {
				baseValues[key] = strconv.Itoa(port)
			}
		}
	}
	for _, key := range resourceManifest.EnvironmentExports.FromRuntimeEnv {
		if value, ok := resourceManifest.Runtime.Env[key]; ok {
			baseValues[key] = value
		}
	}
	for key, derived := range resourceManifest.EnvironmentExports.Derived {
		if previous, exists := exports[key]; exists {
			issues = append(issues, fmt.Sprintf("environment export %s declared in both %s and derived", key, previous))
		}
		matches := templatePattern.FindAllStringSubmatch(derived.Template, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			if _, exists := baseValues[match[1]]; !exists &&
				!mapsContainsKey(resourceManifest.EnvironmentExports.Derived, match[1]) &&
				!isKnownTemplateContextVariable(root, resourceManifest.Name, match[1]) {
				issues = append(issues, fmt.Sprintf("environment_exports.derived[%s] references unknown variable %s", key, match[1]))
			}
		}
		exports[key] = "derived"
	}
	issues = append(issues, validateResourceStorageSources(root, resourceManifest)...)
	return issues
}

func hostPortProtocolKey(port manifestpkg.ResourcePort) string {
	hostIP := strings.TrimSpace(port.HostIP)
	if hostIP == "" {
		hostIP = "*"
	}
	return fmt.Sprintf("%s:%d/%s", hostIP, port.Host, portProtocol(port))
}

func portProtocol(port manifestpkg.ResourcePort) string {
	protocol := strings.ToLower(strings.TrimSpace(port.Protocol))
	if protocol == "" {
		return "tcp"
	}
	return protocol
}

func validateResourceStorageSources(root string, resourceManifest manifestpkg.ResourceManifest) []string {
	issues := []string{}
	for _, volume := range resourceManifest.Runtime.Volumes {
		source := strings.TrimSpace(volume.Source)
		if source == "" {
			continue
		}
		if !isLegacyRepoDataPath(root, resourceManifest.Name, source) {
			continue
		}
		if resourceManifest.LegacyRepoDataAllowed {
			continue
		}
		issues = append(issues, fmt.Sprintf("runtime volume source %q uses repo-local data; migrate to ${RESOURCE_*_DIR} or set legacy_repo_data_allowed=true while retained shell-era paths are being removed", source))
	}
	return issues
}

func isLegacyRepoDataPath(root, resourceName, source string) bool {
	normalized := filepath.ToSlash(strings.TrimSpace(source))
	if normalized == "" {
		return false
	}
	resourceName = strings.Trim(strings.TrimSpace(resourceName), `/\`)
	for _, prefix := range []string{
		"./data",
		"data/",
		"../data",
		"${ROOT}/data",
		"${VROOLI_ROOT}/data",
		"./instances",
		"instances/",
		"${RESOURCE_ROOT}/instances",
	} {
		if normalized == prefix || strings.HasPrefix(normalized, prefix+"/") {
			return true
		}
	}
	if resourceName != "" {
		for _, prefix := range []string{
			"resources/" + resourceName + "/instances",
			"${ROOT}/resources/" + resourceName + "/instances",
			"${VROOLI_ROOT}/resources/" + resourceName + "/instances",
		} {
			if normalized == prefix || strings.HasPrefix(normalized, prefix+"/") {
				return true
			}
		}
	}
	if strings.TrimSpace(root) == "" {
		return false
	}
	cleanSource := filepath.Clean(source)
	for _, legacyRoot := range []string{
		filepath.Join(root, "data"),
		filepath.Join(root, "resources", resourceName, "instances"),
	} {
		if strings.TrimSpace(legacyRoot) == "" {
			continue
		}
		rel, err := filepath.Rel(filepath.Clean(legacyRoot), cleanSource)
		if err != nil {
			continue
		}
		if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
			return true
		}
	}
	return false
}

func ValidateScenario(root, home, scenarioName string, manifest scenario.ServiceManifest) (ScenarioResolution, []string, error) {
	// Validation resolves against the canonical (live) namespace.
	resolution, err := ResolveScenario(root, home, scenarioName, "", manifest)
	if err != nil {
		return ScenarioResolution{}, nil, err
	}
	issues := []string{}
	for key := range manifest.Environment {
		if _, exists := resolution.Values[key]; exists {
			issues = append(issues, fmt.Sprintf("scenario environment key %s overrides a resource-provided value", key))
		}
	}
	return resolution, issues, nil
}

func loadHostPorts(root string, resourceManifest manifestpkg.ResourceManifest) (map[string]int, error) {
	hostPorts := map[string]int{}
	for _, port := range resourceManifest.Ports {
		if strings.TrimSpace(port.Name) == "" {
			continue
		}
		hostPort := port.Host
		if hostPort <= 0 {
			hostPort = port.Container
		}
		if hostPort > 0 {
			hostPorts[strings.TrimSpace(port.Name)] = hostPort
		}
	}

	registry, err := LoadPortRegistry(root)
	if err == nil {
		if port, exists := registry.ResourcePorts[resourceManifest.Name]; exists && len(hostPorts) == 0 {
			hostPorts["default"] = port
		}
	}
	return hostPorts, nil
}

func LoadPortRegistry(root string) (PortRegistry, error) {
	path := filepath.Join(root, filepath.FromSlash(portRegistryJSONPath))
	data, err := os.ReadFile(path)
	if err != nil {
		return PortRegistry{}, fmt.Errorf("read %s: %w", path, err)
	}

	var registry PortRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		return PortRegistry{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if registry.ResourcePorts == nil {
		registry.ResourcePorts = map[string]int{}
	}
	if registry.ReservedRanges == nil {
		registry.ReservedRanges = map[string]string{}
	}
	return registry, nil
}

func applyFallbackDefaults(resourceName string, values map[string]string, hostPorts map[string]int) {
	switch resourceName {
	case "postgres":
		if strings.TrimSpace(values["POSTGRES_HOST"]) == "" {
			values["POSTGRES_HOST"] = "localhost"
		}
		if strings.TrimSpace(values["POSTGRES_SSLMODE"]) == "" {
			values["POSTGRES_SSLMODE"] = "disable"
		}
		if strings.TrimSpace(values["POSTGRES_PORT"]) == "" {
			if port, ok := hostPorts["postgresql"]; ok {
				values["POSTGRES_PORT"] = strconv.Itoa(port)
			}
		}
	case "redis":
		if strings.TrimSpace(values["REDIS_HOST"]) == "" {
			values["REDIS_HOST"] = "localhost"
		}
	case "qdrant":
		if strings.TrimSpace(values["QDRANT_HOST"]) == "" {
			values["QDRANT_HOST"] = "localhost"
		}
	case "ollama":
		if strings.TrimSpace(values["OLLAMA_HOST"]) == "" {
			values["OLLAMA_HOST"] = "localhost"
		}
	}
}

func applyDependencyOverrides(resourceName string, opts ResolveOptions, values map[string]string) {
	if resourceName != "postgres" {
		return
	}
	dbName := strings.TrimSpace(opts.Dependency.Database)
	if dbName == "" && strings.TrimSpace(opts.ScenarioName) != "" {
		// SSOT: live ⇒ "vrooli_<scenario>" (unchanged); shadow ⇒
		// "vrooli_<scenario>_<variant>" so a shadow never writes live's database.
		dbName = scenarioruntime.InstanceKey{
			Scenario: opts.ScenarioName,
			Variant:  opts.Variant,
		}.Namespace().PostgresDB
	}
	if dbName != "" {
		values["POSTGRES_DB"] = dbName
	}
}

func expandTemplateWithContext(value string, env map[string]string, extra map[string]string) string {
	expanded := value
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		expanded = strings.ReplaceAll(expanded, "${"+key+"}", env[key])
		expanded = strings.ReplaceAll(expanded, "$"+key, env[key])
	}
	extraKeys := make([]string, 0, len(extra))
	for key := range extra {
		if _, exists := env[key]; exists {
			continue
		}
		extraKeys = append(extraKeys, key)
	}
	slices.Sort(extraKeys)
	for _, key := range extraKeys {
		expanded = strings.ReplaceAll(expanded, "${"+key+"}", extra[key])
		expanded = strings.ReplaceAll(expanded, "$"+key, extra[key])
	}
	return expanded
}

func buildTemplateContext(root, home, resourceName string) map[string]string {
	dataRoot := strings.TrimSpace(os.Getenv("VROOLI_DATA"))
	if dataRoot == "" && home != "" {
		// Resource-env output var: the data root name comes from the runtime_home
		// authority, not a hard-coded literal.
		if resolved, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyData); err == nil {
			dataRoot = resolved
		}
	}

	context := map[string]string{
		"ROOT": filepath.Clean(root),
	}
	if home != "" {
		context["HOME"] = home
	}
	if dataRoot != "" {
		context["VROOLI_DATA"] = dataRoot
	}
	if root != "" {
		context["VROOLI_ROOT"] = filepath.Clean(root)
		context["RESOURCE_ROOT"] = filepath.Join(filepath.Clean(root), "resources", resourceName)
	}
	if paths, err := resolveResourceStoragePaths(home, resourceName); err == nil {
		context["RESOURCE_CONFIG_DIR"] = paths.ConfigDir
		context["RESOURCE_DATA_DIR"] = paths.DataDir
		context["RESOURCE_CACHE_DIR"] = paths.CacheDir
		context["RESOURCE_LOGS_DIR"] = paths.LogsDir
		context["RESOURCE_STATE_DIR"] = paths.StateDir
	}
	return context
}

func resolveResourceStoragePaths(home, resourceName string) (runtimestorage.Paths, error) {
	cfg := runtimestorage.ResolverConfig{AppID: "vrooli"}
	if strings.TrimSpace(home) != "" {
		cfg.UserHomeDir = func() (string, error) { return home, nil }
		cfg.UserConfigDir = func() (string, error) { return filepath.Join(home, ".config"), nil }
		cfg.UserCacheDir = func() (string, error) { return filepath.Join(home, ".cache"), nil }
	}
	resolver, err := runtimestorage.NewResolver(cfg)
	if err != nil {
		return runtimestorage.Paths{}, err
	}
	return resolver.Resolve(runtimestorage.Options{ResourceID: resourceName})
}

func defaultPortLabel(port manifestpkg.ResourcePort) string {
	if strings.TrimSpace(port.Name) != "" {
		return port.Name
	}
	if port.Container > 0 {
		return strconv.Itoa(port.Container)
	}
	return "unnamed"
}

func mapsContainsKey(m map[string]manifestpkg.ResourceDerivedTemplate, key string) bool {
	_, ok := m[key]
	return ok
}

func isKnownTemplateContextVariable(root, resourceName, key string) bool {
	switch key {
	case "HOME", "ROOT", "VROOLI_ROOT", "VROOLI_DATA", "RESOURCE_ROOT",
		"RESOURCE_CONFIG_DIR", "RESOURCE_DATA_DIR", "RESOURCE_CACHE_DIR", "RESOURCE_LOGS_DIR", "RESOURCE_STATE_DIR":
		return true
	default:
		_, ok := buildTemplateContext(root, "", resourceName)[key]
		return ok
	}
}
