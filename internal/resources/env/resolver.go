package env

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/secrets"
)

const (
	portRegistryJSONPath        = "scripts/resources/port_registry.json"
	resourceDefinitionsJSONPath = ".vrooli/schemas/resource-definitions.json"
)

var templatePattern = regexp.MustCompile(`\$\{([A-Z0-9_]+)\}`)

type PortRegistry struct {
	ResourcePorts  map[string]int    `json:"resource_ports"`
	ReservedRanges map[string]string `json:"reserved_ranges"`
}

type ResolveOptions struct {
	ScenarioName string
	Dependency   scenario.Dependency
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

type resourceDefinitionsFile struct {
	Definitions struct {
		ResourceSchemas map[string]resourceSchema `json:"resourceSchemas"`
	} `json:"definitions"`
}

type resourceSchema struct {
	Properties map[string]schemaProperty `json:"properties"`
}

type schemaProperty struct {
	Default    any                       `json:"default"`
	Properties map[string]schemaProperty `json:"properties"`
}

func ResolveScenario(root, home, scenarioName string, manifest scenario.ServiceManifest) (ScenarioResolution, error) {
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

func resolveFromManifest(root, home string, resourceManifest manifestpkg.ResourceManifest, opts ResolveOptions) (map[string]string, []string, error) {
	values := map[string]string{}
	warnings := []string{}
	prefix := resourceEnvPrefix(resourceManifest.Name)
	templateContext := buildTemplateContext(root, home, resourceManifest.Name)

	secretsMap, err := loadSecrets(root)
	if err != nil {
		return nil, nil, err
	}

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

	for _, key := range resourceManifest.EnvironmentExports.FromRuntimeEnv {
		name := strings.TrimSpace(key)
		if name == "" {
			continue
		}
		if value, ok := resourceManifest.Runtime.Env[name]; ok {
			values[name] = expandTemplateWithContext(value, values, templateContext)
			continue
		}
		if value, ok := secretsMap[name]; ok {
			values[name] = value
			continue
		}
		warnings = append(warnings, fmt.Sprintf("%s environment export %s was requested but no runtime or secret value was found", resourceManifest.Name, name))
	}

	applyDependencyOverrides(resourceManifest.Name, opts, values)

	// Compatibility bridge during migration: keep prefix-matched secrets available
	// while canonical manifests adopt explicit exports.
	for key, value := range secretsMap {
		if strings.HasPrefix(key, prefix+"_") {
			values[key] = value
		}
	}

	if len(resourceManifest.EnvironmentExports.Static) == 0 &&
		len(resourceManifest.EnvironmentExports.FromPorts) == 0 &&
		len(resourceManifest.EnvironmentExports.FromRuntimeEnv) == 0 &&
		len(resourceManifest.EnvironmentExports.Derived) == 0 {
		if strings.TrimSpace(resourceManifest.Driver) == "docker-service" {
			return values, warnings, nil
		}
		legacyValues, err := loadLegacyDefaults(root, home, resourceManifest.Name)
		if err != nil {
			return nil, nil, err
		}
		for key, value := range legacyValues {
			if _, exists := values[key]; !exists {
				values[key] = value
			}
		}
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

func ValidateResourceManifest(root string, resourceManifest manifestpkg.ResourceManifest) []string {
	issues := []string{}
	portNames := map[string]struct{}{}
	hostPorts := map[int]string{}
	for _, port := range resourceManifest.Ports {
		if name := strings.TrimSpace(port.Name); name != "" {
			if _, exists := portNames[name]; exists {
				issues = append(issues, fmt.Sprintf("duplicate port name %q", name))
			}
			portNames[name] = struct{}{}
		}
		if port.Host > 0 {
			if prior, exists := hostPorts[port.Host]; exists {
				issues = append(issues, fmt.Sprintf("duplicate host port %d for %s and %s", port.Host, prior, defaultPortLabel(port)))
			}
			hostPorts[port.Host] = defaultPortLabel(port)
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
			if _, exists := baseValues[match[1]]; !exists && !mapsContainsKey(resourceManifest.EnvironmentExports.Derived, match[1]) {
				issues = append(issues, fmt.Sprintf("environment_exports.derived[%s] references unknown variable %s", key, match[1]))
			}
		}
		exports[key] = "derived"
	}
	return issues
}

func ValidateScenario(root, home, scenarioName string, manifest scenario.ServiceManifest) (ScenarioResolution, []string, error) {
	resolution, err := ResolveScenario(root, home, scenarioName, manifest)
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

func loadSecrets(root string) (map[string]string, error) {
	store := secrets.NewProjectStore(root)
	values, err := store.LoadMigrationCompatible()
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	return values, nil
}

func loadLegacyDefaults(root, home, resourceName string) (map[string]string, error) {
	definitions, err := loadResourceDefinitions(root)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}

	env := map[string]string{}
	prefix := resourceEnvPrefix(resourceName)
	if schema, ok := definitions.Definitions.ResourceSchemas[resourceName]; ok {
		collectSchemaDefaults(prefix, nil, schema.Properties, home, env)
	}
	applyLegacySpecialCases(resourceName, env)
	return env, nil
}

func loadResourceDefinitions(root string) (resourceDefinitionsFile, error) {
	path := filepath.Join(root, filepath.FromSlash(resourceDefinitionsJSONPath))
	data, err := os.ReadFile(path)
	if err != nil {
		return resourceDefinitionsFile{}, err
	}

	var definitions resourceDefinitionsFile
	if err := json.Unmarshal(data, &definitions); err != nil {
		return resourceDefinitionsFile{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if definitions.Definitions.ResourceSchemas == nil {
		definitions.Definitions.ResourceSchemas = map[string]resourceSchema{}
	}
	return definitions, nil
}

func collectSchemaDefaults(prefix string, path []string, properties map[string]schemaProperty, home string, out map[string]string) {
	for name, property := range properties {
		currentPath := append(append([]string(nil), path...), name)
		if len(property.Properties) > 0 {
			collectSchemaDefaults(prefix, currentPath, property.Properties, home, out)
		}
		if property.Default == nil {
			continue
		}
		rendered, ok := stringifySchemaDefault(property.Default, home)
		if !ok {
			continue
		}
		for _, key := range candidateEnvKeys(prefix, currentPath) {
			if strings.TrimSpace(key) == "" {
				continue
			}
			if _, exists := out[key]; !exists {
				out[key] = rendered
			}
		}
	}
}

func candidateEnvKeys(prefix string, path []string) []string {
	if len(path) == 0 || prefix == "" {
		return nil
	}
	if len(path) == 1 {
		name := path[0]
		if isExplicitEnvVar(name) {
			return []string{name}
		}
		if strings.EqualFold(name, "port") {
			return []string{prefix + "_PORT"}
		}
		return []string{prefix + "_" + normalizeEnvSegment(name)}
	}
	if len(path) == 2 && strings.EqualFold(path[0], "ports") {
		leaf := normalizeEnvSegment(path[1])
		keys := []string{prefix + "_" + leaf + "_PORT"}
		if leaf == "API" {
			keys = append([]string{prefix + "_PORT"}, keys...)
		}
		return keys
	}
	name := path[len(path)-1]
	if isExplicitEnvVar(name) {
		return []string{name}
	}
	segments := make([]string, 0, len(path))
	for _, segment := range path {
		if strings.EqualFold(segment, "ports") {
			continue
		}
		segments = append(segments, normalizeEnvSegment(segment))
	}
	if len(segments) == 0 {
		return nil
	}
	return []string{prefix + "_" + strings.Join(segments, "_")}
}

func stringifySchemaDefault(value any, home string) (string, bool) {
	switch typed := value.(type) {
	case string:
		if home != "" && strings.HasPrefix(typed, "~/") {
			return filepath.Join(home, strings.TrimPrefix(typed, "~/")), true
		}
		return typed, true
	case bool:
		if typed {
			return "true", true
		}
		return "false", true
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10), true
		}
		return strconv.FormatFloat(typed, 'f', -1, 64), true
	case int:
		return strconv.Itoa(typed), true
	case int64:
		return strconv.FormatInt(typed, 10), true
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			rendered, ok := stringifySchemaDefault(item, home)
			if ok {
				values = append(values, rendered)
			}
		}
		return strings.Join(values, ","), true
	default:
		return "", false
	}
}

func resourceEnvPrefix(resourceName string) string {
	return normalizeEnvSegment(strings.ReplaceAll(resourceName, "-", "_"))
}

func normalizeEnvSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	replacer := strings.NewReplacer("-", "_", ".", "_", " ", "_")
	return strings.ToUpper(replacer.Replace(value))
}

func isExplicitEnvVar(value string) bool {
	if value == "" {
		return false
	}
	return value == strings.ToUpper(value) && strings.ContainsAny(value, "_ABCDEFGHIJKLMNOPQRSTUVWXYZ")
}

func applyLegacySpecialCases(resourceName string, env map[string]string) {
	switch resourceName {
	case "postgres":
		if strings.TrimSpace(env["POSTGRES_HOST"]) == "" {
			env["POSTGRES_HOST"] = "localhost"
		}
		if strings.TrimSpace(env["POSTGRES_USER"]) == "" {
			env["POSTGRES_USER"] = "vrooli"
		}
		if strings.TrimSpace(env["POSTGRES_SSLMODE"]) == "" {
			env["POSTGRES_SSLMODE"] = "disable"
		}
	case "minio":
		if port := strings.TrimSpace(env["MINIO_PORT"]); port != "" && strings.TrimSpace(env["MINIO_BASE_URL"]) == "" {
			env["MINIO_BASE_URL"] = "http://localhost:" + port
		}
		if consolePort := strings.TrimSpace(env["MINIO_CONSOLE_PORT"]); consolePort != "" && strings.TrimSpace(env["MINIO_CONSOLE_URL"]) == "" {
			env["MINIO_CONSOLE_URL"] = "http://localhost:" + consolePort
		}
	}
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
	case "browserless":
		if strings.TrimSpace(values["BROWSERLESS_HOST"]) == "" {
			values["BROWSERLESS_HOST"] = "localhost"
		}
	}
}

func applyDependencyOverrides(resourceName string, opts ResolveOptions, values map[string]string) {
	if resourceName != "postgres" {
		return
	}
	dbName := strings.TrimSpace(opts.Dependency.Database)
	if dbName == "" && strings.TrimSpace(opts.ScenarioName) != "" {
		dbName = "vrooli_" + strings.ReplaceAll(opts.ScenarioName, "-", "_")
	}
	if dbName != "" {
		values["POSTGRES_DB"] = dbName
	}
}

func expandTemplate(value string, env map[string]string) string {
	return expandTemplateWithContext(value, env, nil)
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
		dataRoot = filepath.Join(home, ".vrooli", "data")
	}

	context := map[string]string{}
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
	return context
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
