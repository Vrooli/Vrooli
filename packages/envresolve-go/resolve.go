// Package envresolve derives environment producers from existing repository
// manifests. It intentionally has no consumer-specific variable table.
package envresolve

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type ProducerKind string

const (
	ResourceProducer       ProducerKind = "resource"
	ScenarioPortProducer   ProducerKind = "scenario_port"
	ScenarioAbsoluteSource ProducerKind = "scenario_absolute"
	LifecycleProducer      ProducerKind = "lifecycle_builtin"
)

type Producer struct {
	Kind     ProducerKind
	Name     string
	Resource string
	Scenario string
	Port     string
}

type Manifest struct {
	Name         string            `json:"-"`
	Dependencies Dependencies      `json:"dependencies"`
	Ports        map[string]Port   `json:"ports"`
	Environment  map[string]string `json:"environment"`
}

type Dependencies struct {
	Resources map[string]json.RawMessage `json:"resources"`
	Scenarios map[string]json.RawMessage `json:"scenarios"`
}

type Port struct {
	EnvVar string `json:"env_var"`
}

type Index struct {
	producers           map[string][]Producer
	patterns            []Producer
	resources           map[string]struct{}
	scenarios           map[string]struct{}
	referencedResources map[string]struct{}
}

var resourcePortReferencePattern = regexp.MustCompile(`getResourcePort\(\s*["']([^"']+)["']\s*\)`)

// Load scans the repository's resource and scenario manifests.
func Load(root string) (*Index, error) {
	idx := &Index{producers: map[string][]Producer{}, resources: map[string]struct{}{}, scenarios: map[string]struct{}{}, referencedResources: map[string]struct{}{}}
	resourceDirs, err := filepath.Glob(filepath.Join(root, "resources", "*", "resource.json"))
	if err != nil {
		return nil, err
	}
	for _, path := range resourceDirs {
		var doc struct {
			EnvironmentExports struct {
				Static         map[string]json.RawMessage `json:"static"`
				FromPorts      map[string]json.RawMessage `json:"from_ports"`
				FromRuntimeEnv []string                   `json:"from_runtime_env"`
				Derived        map[string]json.RawMessage `json:"derived"`
			} `json:"environment_exports"`
		}
		if err := readJSON(path, &doc); err != nil {
			return nil, err
		}
		resource := filepath.Base(filepath.Dir(path))
		idx.resources[resource] = struct{}{}
		for key := range doc.EnvironmentExports.Static {
			idx.add(key, Producer{Kind: ResourceProducer, Name: resource, Resource: resource})
		}
		for key := range doc.EnvironmentExports.FromPorts {
			idx.add(key, Producer{Kind: ResourceProducer, Name: resource, Resource: resource})
		}
		for _, key := range doc.EnvironmentExports.FromRuntimeEnv {
			idx.add(key, Producer{Kind: ResourceProducer, Name: resource, Resource: resource})
		}
		for key := range doc.EnvironmentExports.Derived {
			idx.add(key, Producer{Kind: ResourceProducer, Name: resource, Resource: resource})
		}
	}
	scenarioDirs, err := filepath.Glob(filepath.Join(root, "scenarios", "*", ".vrooli", "service.json"))
	if err != nil {
		return nil, err
	}
	for _, path := range scenarioDirs {
		var doc Manifest
		if err := readJSON(path, &doc); err != nil {
			return nil, err
		}
		scenario := filepath.Base(filepath.Dir(filepath.Dir(path)))
		doc.Name = scenario
		idx.scenarios[scenario] = struct{}{}
		for port, declaration := range doc.Ports {
			if declaration.EnvVar != "" {
				idx.add(declaration.EnvVar, Producer{Kind: ScenarioPortProducer, Name: scenario, Scenario: scenario, Port: port})
			}
		}
		for key := range doc.Environment {
			idx.add(key, Producer{Kind: ScenarioAbsoluteSource, Name: scenario, Scenario: scenario})
		}
		idx.patterns = append(idx.patterns, Producer{Kind: ScenarioAbsoluteSource, Name: scenario, Scenario: scenario})
	}
	for _, scenarioDir := range scenarioDirsFromRoot(root) {
		_ = filepath.Walk(scenarioDir, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil || info == nil || info.IsDir() || filepath.Ext(path) != ".go" {
				return nil
			}
			payload, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			for _, name := range referencedResourceNames(payload) {
				idx.referencedResources[name] = struct{}{}
			}
			return nil
		})
	}
	for _, key := range BuiltinVariables() {
		idx.add(key, Producer{Kind: LifecycleProducer, Name: "lifecycle"})
	}
	for key := range idx.producers {
		sort.Slice(idx.producers[key], func(i, j int) bool { return idx.producers[key][i].Kind < idx.producers[key][j].Kind })
	}
	return idx, nil
}

// DeadResource reports whether a variable uses the conventional resource
// prefix but the corresponding resource directory is absent. The variable is
// still supplied by the caller; this method only derives the resource name.
func (i *Index) DeadResource(variable string) string {
	prefix, _, ok := strings.Cut(strings.ToUpper(strings.TrimSpace(variable)), "_")
	if !ok || prefix == "" {
		return ""
	}
	for resource := range i.referencedResources {
		if strings.ToUpper(strings.ReplaceAll(resource, "-", "_")) != prefix {
			continue
		}
		if _, exists := i.resources[resource]; !exists {
			return resource
		}
	}
	return ""
}

func (i *Index) add(variable string, producer Producer) {
	variable = strings.TrimSpace(variable)
	if variable == "" {
		return
	}
	for _, existing := range i.producers[variable] {
		if existing == producer {
			return
		}
	}
	i.producers[variable] = append(i.producers[variable], producer)
}

// Producers returns all manifest-derived producers for variable. A scenario
// prefix is a producer pattern because the exact peer port is intentionally
// resolved by discovery rather than injected into the environment.
func (i *Index) Producers(variable string) []Producer {
	variable = strings.TrimSpace(variable)
	result := append([]Producer(nil), i.producers[variable]...)
	for _, pattern := range i.patterns {
		prefix := strings.ToUpper(strings.ReplaceAll(pattern.Scenario, "-", "_")) + "_"
		if strings.HasPrefix(variable, prefix) {
			result = append(result, pattern)
		}
	}
	return result
}

// IsScenarioAddressVariable narrows slug-shaped scenario patterns to names
// that conventionally carry a peer coordinate. A scenario-prefixed setting,
// credential, or path remains residual configuration rather than becoming a
// false address violation merely because its name shares a slug prefix.
func IsScenarioAddressVariable(variable string) bool {
	upper := strings.ToUpper(strings.TrimSpace(variable))
	for _, suffix := range []string{"_API_URL", "_BASE_URL", "_URL", "_API_PORT", "_PORT"} {
		if strings.HasSuffix(upper, suffix) {
			return true
		}
	}
	return false
}

func (i *Index) Satisfiable(manifest Manifest, variable string) (bool, []Producer) {
	producers := i.Producers(variable)
	if len(producers) == 0 {
		return false, nil
	}
	for _, producer := range producers {
		switch producer.Kind {
		case ResourceProducer:
			if _, ok := manifest.Dependencies.Resources[producer.Resource]; ok {
				return true, producers
			}
		case ScenarioPortProducer, ScenarioAbsoluteSource:
			if manifest.Name != "" && producer.Scenario == manifest.Name {
				return true, producers
			}
			if _, ok := manifest.Dependencies.Scenarios[producer.Scenario]; ok {
				return true, producers
			}
		case LifecycleProducer:
			return true, producers
		}
	}
	return false, producers
}

// BuiltinVariables is the single lifecycle-owned list. Prefix-shaped
// scenario identity variables are also recognized by callers through the
// scenario producer patterns and are not duplicated as per-variable entries.
func BuiltinVariables() []string {
	return []string{"SCENARIO_NAME", "SCENARIO_MODE", "SCENARIO_PATH", "SCENARIO_DATA_DIR", "VROOLI_SCENARIO", "VROOLI_SCENARIO_DIR", "VROOLI_VARIANT", "VROOLI_STORAGE_NAMESPACE"}
}

// OSStandardVariables is the shared allowlist for variables supplied by the
// host/toolchain rather than by Vrooli manifests.
func OSStandardVariables() map[string]struct{} {
	keys := []string{"PATH", "Path", "HOME", "USER", "USERNAME", "TEMP", "TMP", "TMPDIR", "SYSTEMROOT", "COMSPEC", "PATHEXT", "SHELL", "PWD", "OLDPWD", "LANG", "LC_ALL", "TERM", "CI", "CGO_ENABLED", "GOFLAGS", "PYTHONPATH", "ANDROID_SDK_ROOT"}
	result := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		result[key] = struct{}{}
	}
	return result
}

func readJSON(path string, target any) error {
	payload, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func scenarioDirsFromRoot(root string) []string {
	entries, _ := os.ReadDir(filepath.Join(root, "scenarios"))
	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			result = append(result, filepath.Join(root, "scenarios", entry.Name()))
		}
	}
	return result
}

func referencedResourceNames(payload []byte) []string {
	matches := resourcePortReferencePattern.FindAllSubmatch(payload, -1)
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) == 2 {
			result = append(result, strings.TrimSpace(string(match[1])))
		}
	}
	return result
}
