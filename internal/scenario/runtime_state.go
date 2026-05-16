package scenario

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/process"
)

type RuntimePortBinding struct {
	Key            string `json:"key"`
	Step           string `json:"step,omitempty"`
	Port           int    `json:"port"`
	ListenerStatus string `json:"listener_status,omitempty"`
}

type RuntimeEndpoint struct {
	Name        string `json:"name"`
	Key         string `json:"key"`
	Description string `json:"description,omitempty"`
	Port        int    `json:"port"`
	URL         string `json:"url"`
}

type RuntimePortResolution struct {
	Key  string
	Step string
	Port int
}

type RuntimeDetails struct {
	Status       string               `json:"status"`
	Processes    int                  `json:"processes"`
	Runtime      string               `json:"runtime"`
	StartedAt    *time.Time           `json:"started_at,omitempty"`
	Ports        map[string]int       `json:"ports,omitempty"`
	PortBindings []RuntimePortBinding `json:"port_bindings,omitempty"`
	ProcessInfo  []process.Record     `json:"process_info,omitempty"`
	Health       string               `json:"health_status,omitempty"`
	HealthError  string               `json:"health_error,omitempty"`
}

func DescribeRuntime(manifest ServiceManifest, runtime process.ScenarioRuntime) RuntimeDetails {
	bindings, ports := RuntimePortBindings(manifest, runtime.Records)
	status := "stopped"
	health := ""
	if runtime.ProcessCount > 0 {
		status = "running"
		health = EvaluateHealth(manifest.HealthConfig(), ports)
	}

	return RuntimeDetails{
		Status:       status,
		Processes:    runtime.ProcessCount,
		Runtime:      runtime.Runtime,
		StartedAt:    runtime.StartedAt,
		Ports:        ports,
		PortBindings: bindings,
		ProcessInfo:  append([]process.Record(nil), runtime.Records...),
		Health:       health,
	}
}

func RuntimePorts(manifest ServiceManifest, records []process.Record) map[string]int {
	_, ports := RuntimePortBindings(manifest, records)
	return ports
}

func RuntimeEndpoints(manifest ServiceManifest, ports map[string]int) []RuntimeEndpoint {
	if len(ports) == 0 {
		return nil
	}

	endpoints := make([]RuntimeEndpoint, 0, len(ports))
	seen := make(map[string]struct{}, len(ports))
	for _, definition := range manifest.SortedPorts() {
		port, ok := ports[definition.EnvVar]
		if !ok || port <= 0 {
			continue
		}
		endpoints = append(endpoints, RuntimeEndpoint{
			Name:        definition.Name,
			Key:         definition.EnvVar,
			Description: definition.Description,
			Port:        port,
			URL:         "http://localhost:" + strconv.Itoa(port),
		})
		seen[definition.EnvVar] = struct{}{}
	}

	extraKeys := make([]string, 0, len(ports))
	for key, port := range ports {
		if port <= 0 {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		extraKeys = append(extraKeys, key)
	}
	sort.Strings(extraKeys)
	for _, key := range extraKeys {
		port := ports[key]
		endpoints = append(endpoints, RuntimeEndpoint{
			Name: key,
			Key:  key,
			Port: port,
			URL:  "http://localhost:" + strconv.Itoa(port),
		})
	}

	return endpoints
}

func ResolveRuntimePort(manifest ServiceManifest, bindings []RuntimePortBinding, ports map[string]int, requested string) (RuntimePortResolution, bool) {
	for _, key := range runtimePortCandidates(manifest, requested) {
		port, ok := ports[key]
		if !ok {
			continue
		}
		step := ""
		for _, binding := range bindings {
			if binding.Key == key && binding.Port == port {
				step = binding.Step
				break
			}
		}
		return RuntimePortResolution{Key: key, Step: step, Port: port}, true
	}
	return RuntimePortResolution{}, false
}

func RuntimePortBindings(manifest ServiceManifest, records []process.Record) ([]RuntimePortBinding, map[string]int) {
	ports := make(map[string]int)
	bindings := make([]RuntimePortBinding, 0, len(records))
	seen := make(map[string]struct{})

	for _, record := range records {
		if record.Port <= 0 {
			continue
		}

		key := InferPortEnvVar(manifest, record.Step)
		if key == "" {
			continue
		}

		if _, exists := ports[key]; !exists {
			ports[key] = record.Port
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		bindings = append(bindings, RuntimePortBinding{
			Key:  key,
			Step: record.Step,
			Port: record.Port,
		})
	}

	envPorts := process.ReadEnvironmentPorts(records, manifest.PortEnvVars())
	for key, port := range envPorts {
		if _, exists := ports[key]; !exists {
			ports[key] = port
		}
	}

	sort.Slice(bindings, func(i, j int) bool {
		if bindings[i].Key == bindings[j].Key {
			return bindings[i].Step < bindings[j].Step
		}
		return bindings[i].Key < bindings[j].Key
	})

	return bindings, ports
}

func runtimePortCandidates(manifest ServiceManifest, requested string) []string {
	candidates := []string{requested}
	if envVar := manifest.PortEnvVar(strings.ToLower(strings.TrimSuffix(requested, "_PORT"))); envVar != "" {
		candidates = append(candidates, envVar)
	}
	normalized := strings.ToUpper(strings.TrimSpace(requested))
	if normalized != "" && normalized != requested {
		candidates = append(candidates, normalized)
		if !strings.HasSuffix(normalized, "_PORT") {
			candidates = append(candidates, normalized+"_PORT")
		}
	}

	out := make([]string, 0, len(candidates))
	seen := map[string]struct{}{}
	for _, key := range candidates {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

// InferPortEnvVar maps lifecycle step names like "start-api" or "launch-ui"
// back to manifest port keys. The process metadata is historical and not fully
// normalized, so the mapping intentionally uses lightweight heuristics.
func InferPortEnvVar(manifest ServiceManifest, step string) string {
	step = strings.ToLower(strings.TrimSpace(step))
	step = strings.TrimPrefix(step, "start-")
	step = strings.TrimPrefix(step, "run-")
	step = strings.TrimPrefix(step, "serve-")
	step = strings.TrimPrefix(step, "launch-")

	if step != "" {
		if envVar := manifest.PortEnvVar(step); envVar != "" {
			return envVar
		}
	}

	switch {
	case strings.Contains(step, "ui"), strings.Contains(step, "frontend"), strings.Contains(step, "vite"):
		if envVar := manifest.PortEnvVar("ui"); envVar != "" {
			return envVar
		}
	case strings.Contains(step, "ws"), strings.Contains(step, "socket"):
		for _, candidate := range []string{"websocket", "ws"} {
			if envVar := manifest.PortEnvVar(candidate); envVar != "" {
				return envVar
			}
		}
	}

	for _, definition := range manifest.SortedPorts() {
		name := strings.ToLower(definition.Name)
		if step == name || strings.Contains(step, name) || strings.Contains(name, step) {
			return definition.EnvVar
		}
		normalizedEnv := strings.TrimSuffix(strings.ToLower(definition.EnvVar), "_port")
		if normalizedEnv != "" && (step == normalizedEnv || strings.Contains(step, normalizedEnv)) {
			return definition.EnvVar
		}
	}

	return ""
}
