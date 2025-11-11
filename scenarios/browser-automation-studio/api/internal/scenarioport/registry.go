package scenarioport

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
)

const registryEnvVar = "SCENARIO_REGISTRY"

var (
	registryOnce sync.Once
	registryData *scenarioRegistry
	registryErr  error
)

func lookupRegistryEntry(name string) *scenarioRegistryEntry {
	registry := getScenarioRegistry()
	if registry == nil {
		return nil
	}
	return registry.lookup(name)
}

func resetRegistryCacheForTests() {
	registryOnce = sync.Once{}
	registryData = nil
	registryErr = nil
}

func getScenarioRegistry() *scenarioRegistry {
	registryOnce.Do(func() {
		registryData, registryErr = loadScenarioRegistryFromEnv()
		if registryErr != nil {
			log.Printf("scenarioport: failed to load %s: %v", registryEnvVar, registryErr)
		}
	})
	return registryData
}

func loadScenarioRegistryFromEnv() (*scenarioRegistry, error) {
	raw := strings.TrimSpace(os.Getenv(registryEnvVar))
	if raw == "" {
		return nil, nil
	}

	source := raw
	if strings.HasPrefix(raw, "@") {
		source = strings.TrimSpace(strings.TrimPrefix(raw, "@"))
	}

	var payload []byte
	if fileInfo, err := os.Stat(source); err == nil && !fileInfo.IsDir() {
		data, err := os.ReadFile(source)
		if err != nil {
			return nil, err
		}
		payload = data
	} else if strings.HasPrefix(raw, "@") {
		return nil, fmt.Errorf("registry file %s not found", source)
	} else {
		payload = []byte(raw)
	}

	return parseScenarioRegistry(payload)
}

func parseScenarioRegistry(payload []byte) (*scenarioRegistry, error) {
	payload = []byte(strings.TrimSpace(string(payload)))
	if len(payload) == 0 {
		return nil, errors.New("registry payload is empty")
	}

	var raw interface{}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, err
	}

	registry := &scenarioRegistry{entries: make(map[string]*scenarioRegistryEntry)}

	switch typed := raw.(type) {
	case map[string]interface{}:
		for key, value := range typed {
			entry, err := normalizeRegistryEntry(value, key)
			if err != nil {
				return nil, err
			}
			if entry != nil {
				registry.entries[entry.Key()] = entry
			}
		}
	case []interface{}:
		for _, value := range typed {
			entry, err := normalizeRegistryEntry(value, "")
			if err != nil {
				return nil, err
			}
			if entry != nil {
				registry.entries[entry.Key()] = entry
			}
		}
	default:
		return nil, fmt.Errorf("unsupported registry format %T", raw)
	}

	if len(registry.entries) == 0 {
		return nil, errors.New("registry does not contain entries")
	}

	return registry, nil
}

func normalizeRegistryEntry(raw interface{}, defaultName string) (*scenarioRegistryEntry, error) {
	switch typed := raw.(type) {
	case string:
		name := normalizeScenarioName(defaultName)
		if name == "" {
			return nil, fmt.Errorf("registry entry %q missing scenario name", typed)
		}
		return &scenarioRegistryEntry{Name: name, URL: strings.TrimSpace(typed)}, nil
	case map[string]interface{}:
		entry := &scenarioRegistryEntry{}

		if nameVal, ok := typed["name"].(string); ok {
			entry.Name = nameVal
		}
		if entry.Name == "" {
			entry.Name = defaultName
		}
		entry.Name = normalizeScenarioName(entry.Name)
		if entry.Name == "" {
			return nil, errors.New("registry entry missing scenario name")
		}

		if urlVal, ok := typed["url"].(string); ok {
			entry.URL = strings.TrimSpace(urlVal)
		}

		if portsVal, ok := typed["ports"].(map[string]interface{}); ok {
			entry.Ports = make(map[string]int)
			for portName, portValue := range portsVal {
				if port, ok := parsePortValue(portValue); ok && port > 0 {
					entry.Ports[strings.ToUpper(strings.TrimSpace(portName))] = port
				}
			}
		}

		return entry, nil
	default:
		return nil, fmt.Errorf("unsupported registry entry type %T", raw)
	}
}

func parsePortValue(value interface{}) (int, bool) {
	switch v := value.(type) {
	case float64:
		return int(v), true
	case float32:
		return int(v), true
	case int:
		return v, true
	case int64:
		return int(v), true
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i), true
		}
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false
		}
		if port, err := strconv.Atoi(trimmed); err == nil {
			return port, true
		}
	}
	return 0, false
}

func normalizeScenarioName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

type scenarioRegistry struct {
	entries map[string]*scenarioRegistryEntry
}

func (r *scenarioRegistry) lookup(name string) *scenarioRegistryEntry {
	if r == nil {
		return nil
	}
	key := normalizeScenarioName(name)
	if key == "" {
		return nil
	}
	if entry, ok := r.entries[key]; ok {
		return entry
	}
	return nil
}

type scenarioRegistryEntry struct {
	Name  string
	URL   string
	Ports map[string]int
}

func (e *scenarioRegistryEntry) Key() string {
	return normalizeScenarioName(e.Name)
}

func (e *scenarioRegistryEntry) hasURL() bool {
	return strings.TrimSpace(e.URL) != ""
}

func (e *scenarioRegistryEntry) portFor(names []string) (string, int) {
	if len(e.Ports) == 0 {
		return "", 0
	}
	for _, name := range names {
		normalized := strings.ToUpper(strings.TrimSpace(name))
		if normalized == "" {
			continue
		}
		if port, ok := e.Ports[normalized]; ok && port > 0 {
			return normalized, port
		}
	}
	return "", 0
}

func (e *scenarioRegistryEntry) resolveURL(path string, portNames []string) (string, *PortInfo, bool) {
	trimmedPath := strings.TrimSpace(path)

	if e.hasURL() {
		resolved, err := combineURL(e.URL, trimmedPath)
		if err == nil {
			if portName, port := e.portFor(portNames); port > 0 {
				return resolved, &PortInfo{Name: portName, Port: port}, true
			}
			if port := portFromURL(resolved); port > 0 {
				return resolved, &PortInfo{Name: "URL", Port: port}, true
			}
			return resolved, nil, true
		}
	}

	if portName, port := e.portFor(portNames); port > 0 {
		if built, err := BuildURL(port, trimmedPath); err == nil {
			return built, &PortInfo{Name: portName, Port: port}, true
		}
	}

	return "", nil, false
}

func combineURL(baseURL, rawPath string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", err
	}
	if rawPath == "" {
		return parsed.String(), nil
	}
	rel, err := url.Parse(rawPath)
	if err != nil {
		return "", err
	}
	return parsed.ResolveReference(rel).String(), nil
}

func portFromURL(raw string) int {
	parsed, err := url.Parse(raw)
	if err != nil {
		return 0
	}
	if portStr := parsed.Port(); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			return port
		}
	}
	switch strings.ToLower(parsed.Scheme) {
	case "https":
		return 443
	case "http":
		return 80
	default:
		return 0
	}
}
