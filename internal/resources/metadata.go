package resources

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/secrets"
)

const (
	portRegistryJSONPath        = "scripts/resources/port_registry.json"
	resourceDefinitionsJSONPath = ".vrooli/schemas/resource-definitions.json"
)

type PortRegistry struct {
	ResourcePorts  map[string]int    `json:"resource_ports"`
	ReservedRanges map[string]string `json:"reserved_ranges"`
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

func LoadResourceEnvironment(root, home, resourceName string) (map[string]string, error) {
	env := map[string]string{}
	prefix := resourceEnvPrefix(resourceName)

	registry, err := LoadPortRegistry(root)
	if err != nil {
		return nil, err
	}
	if port, ok := registry.ResourcePorts[resourceName]; ok {
		env[prefix+"_PORT"] = strconv.Itoa(port)
		if prefix != "" {
			env[prefix+"_BASE_URL"] = "http://localhost:" + strconv.Itoa(port)
		}
	}

	definitions, err := loadResourceDefinitions(root)
	if err != nil {
		return nil, err
	}
	if schema, ok := definitions.Definitions.ResourceSchemas[resourceName]; ok {
		collectSchemaDefaults(prefix, nil, schema.Properties, home, env)
	}

	secrets, err := loadSecrets(root)
	if err != nil {
		return nil, err
	}
	for key, value := range secrets {
		if strings.HasPrefix(key, prefix+"_") {
			env[key] = value
		}
	}

	applyResourceSpecialCases(resourceName, env)
	return env, nil
}

func loadResourceDefinitions(root string) (resourceDefinitionsFile, error) {
	path := filepath.Join(root, filepath.FromSlash(resourceDefinitionsJSONPath))
	data, err := os.ReadFile(path)
	if err != nil {
		return resourceDefinitionsFile{}, fmt.Errorf("read %s: %w", path, err)
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
			if !ok {
				continue
			}
			values = append(values, rendered)
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

func applyResourceSpecialCases(resourceName string, env map[string]string) {
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
