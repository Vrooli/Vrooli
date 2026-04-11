package scenario

import (
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/process"
)

type RuntimePortBinding struct {
	Key  string `json:"key"`
	Step string `json:"step,omitempty"`
	Port int    `json:"port"`
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
