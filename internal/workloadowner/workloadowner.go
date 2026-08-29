// Package workloadowner reconciles observed host workloads with declarations
// owned by the control plane. It intentionally has no relationship to the
// legacy process snapshot check.
package workloadowner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/scenario"
)

const (
	workloadownerParameterA = 2
)

// DeclarationsFromRoot derives the live container declarations from the
// repository's resource manifests and enabled-resource state. Historical
// declarations are intentionally supplied by the caller because they require
// durable evidence rather than a guess from a current directory scan.
func DeclarationsFromRoot(root string) ([]Declaration, error) {
	service, err := scenario.LoadServiceManifest(filepath.Join(root, repocontractmeta.ProjectConfigDir, "service.json"))
	if err != nil {
		return nil, fmt.Errorf("read enabled resource state: %w", err)
	}
	var declarations []Declaration
	for name, state := range service.Dependencies.Resources {
		if !state.Enabled {
			continue
		}
		manifestPath := filepath.Join(root, "resources", name, "resource.json")
		manifestBytes, readErr := os.ReadFile(manifestPath)
		if readErr != nil {
			continue
		}
		var manifest struct {
			Name    string `json:"name"`
			Driver  string `json:"driver"`
			Runtime struct {
				ContainerName string `json:"container_name"`
			} `json:"runtime"`
		}
		if json.Unmarshal(manifestBytes, &manifest) != nil || manifest.Name == "" {
			continue
		}
		if manifest.Driver != "managed-service" {
			continue
		}
		container := strings.TrimSpace(manifest.Runtime.ContainerName)
		if container == "" {
			container = "vrooli-" + manifest.Name
		}
		declarations = append(declarations, Declaration{Kind: "container", Name: container, Live: true, Evidence: []string{"enabled resource manifest: " + manifestPath}})
	}
	for _, unit := range []string{
		"vrooli-runtime-supervisor.service",
		"vrooli-autoheal.service",
		"vrooli-emergency-watchdog.service",
		"vrooli-emergency-watchdog.timer",
	} {
		declarations = append(declarations, Declaration{
			Kind: "service-unit", Name: unit, Live: true,
			Evidence: []string{"Vrooli control-plane unit declaration: " + unit},
		})
	}
	return declarations, nil
}

type Class string

const (
	Declared  Class = "declared"
	Unmanaged Class = "unmanaged"
	Abandoned Class = "abandoned"
)

type Posture string

const (
	WholeHost  Posture = "whole_host"
	VrooliOnly Posture = "vrooli_only"
)

type Workload struct {
	Kind         string   `json:"kind"`
	Name         string   `json:"name"`
	Image        string   `json:"image,omitempty"`
	CommandLine  string   `json:"command_line,omitempty"`
	Running      bool     `json:"running"`
	RestartCount float64  `json:"restart_count"`
	WindowHours  float64  `json:"window_hours,omitempty"`
	Evidence     []string `json:"evidence,omitempty"`
}

type Declaration struct {
	Kind     string
	Name     string
	Live     bool
	Owner    string
	Evidence []string
}

type Finding struct {
	Workload
	Class          Class  `json:"class"`
	Finding        bool   `json:"finding"`
	Informative    bool   `json:"informative"`
	CrashLoop      bool   `json:"crash_loop,omitempty"`
	ProposedAction string `json:"proposed_action,omitempty"`
	Reason         string `json:"reason"`
}

type Report struct {
	Posture       Posture   `json:"posture"`
	Declared      []Finding `json:"declared,omitempty"`
	Findings      []Finding `json:"findings"`
	Informational []Finding `json:"informational,omitempty"`
}

func Classify(observed []Workload, declarations []Declaration, posture Posture, crashRestartsPerHour float64) Report {
	if posture != WholeHost {
		posture = VrooliOnly
	}
	known := make(map[string]Declaration, len(declarations))
	for _, d := range declarations {
		known[key(d.Kind, d.Name)] = d
	}
	r := Report{Posture: posture}
	for _, w := range observed {
		d, exists := known[key(w.Kind, w.Name)]
		f := Finding{Workload: w, Finding: false, Informative: false}
		switch {
		case exists && d.Live:
			f.Class = Declared
			f.Reason = "matches a live Vrooli declaration"
			f.Evidence = append(f.Evidence, d.Evidence...)
		case exists:
			f.Class = Abandoned
			f.Finding = true
			f.Reason = "matches a historical Vrooli declaration that is no longer live"
			f.Evidence = append(f.Evidence, d.Evidence...)
			f.ProposedAction = "preview undeclared-workload disposal"
		case historicalNamingRule(w):
			f.Class = Abandoned
			f.Finding = true
			f.Reason = "matches a Vrooli historical workload naming rule without a live manifest"
			f.Evidence = append(f.Evidence, historicalNamingEvidence(w)...)
			f.ProposedAction = "preview undeclared-workload disposal"
		default:
			f.Class = Unmanaged
			f.Reason = "matches no Vrooli declaration"
			f.Evidence = append(f.Evidence, "no Vrooli declaration or historical naming rule matched")
			if posture == WholeHost {
				f.Finding = true
			} else {
				f.Informative = true
				f.CommandLine = ""
			}
		}
		if f.Class != Unmanaged && w.WindowHours > 0 && w.RestartCount/w.WindowHours >= crashRestartsPerHour {
			f.CrashLoop = true
			f.Finding = true
			f.Reason += "; restart rate exceeds the crash-loop bar"
		}
		if f.Finding {
			r.Findings = append(r.Findings, f)
		} else if f.Class == Declared {
			r.Declared = append(r.Declared, f)
		} else if f.Informative {
			r.Informational = append(r.Informational, f)
		}
	}
	return r
}

func historicalNamingRule(w Workload) bool {
	if w.Kind != "container" {
		return false
	}
	name := strings.ToLower(strings.TrimSpace(w.Name))
	image := strings.ToLower(strings.TrimSpace(w.Image))
	return strings.HasPrefix(name, "vrooli-") || strings.HasPrefix(image, "vrooli/") || strings.HasPrefix(image, "vrooli-") || strings.Contains(name, "airbyte-abctl")
}

func historicalNamingEvidence(w Workload) []string {
	if strings.Contains(strings.ToLower(w.Name), "airbyte-abctl") {
		return []string{"historical agent experiment: Airbyte KinD workload", "container image: " + w.Image, "no live Vrooli resource manifest"}
	}
	return []string{"Vrooli container naming rule matched: " + w.Name, "no live Vrooli resource manifest"}
}

func key(kind, name string) string {
	return strings.ToLower(strings.TrimSpace(kind)) + "\x00" + strings.ToLower(strings.TrimSpace(name))
}

// ParseDockerPS accepts Docker's --format json output, one object per line.
func ParseDockerPS(data []byte) ([]Workload, error) {
	var result []Workload
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var v struct {
			Name, Names, Image, Command, State, Status string
		}
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			return nil, fmt.Errorf("parse docker workload: %w", err)
		}
		name := v.Name
		if name == "" {
			name = v.Names
		}
		result = append(result, Workload{Kind: "container", Name: name, Image: v.Image, CommandLine: v.Command, Running: strings.EqualFold(v.State, "running"), Evidence: []string{"docker ps --format json: " + name}})
	}
	return result, nil
}

// ParseDockerInspectRestartCounts parses the compact, one-record-per-line
// output requested by the workload census. Docker is queried once for all
// observed names, so restart evidence does not turn enumeration into an
// N+1 process/command pattern.
func ParseDockerInspectRestartCounts(data []byte) map[string]float64 {
	counts := make(map[string]float64)
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		fields := strings.SplitN(strings.TrimSpace(line), "\t", workloadownerParameterA)
		if len(fields) != workloadownerParameterA {
			continue
		}
		name := strings.TrimPrefix(strings.TrimSpace(fields[0]), "/")
		var count float64
		if _, err := fmt.Sscanf(strings.TrimSpace(fields[1]), "%f", &count); err == nil && name != "" {
			counts[name] = count
		}
	}
	return counts
}

// ParseDockerInspectJSON reads the native docker inspect JSON fixture/API
// shape. It is intentionally separate from ParseDockerInspectRestartCounts,
// whose compact tabular form is the bounded live-census command output.
func ParseDockerInspectJSON(data []byte) map[string]float64 {
	var records []struct {
		Name  string `json:"Name"`
		State struct {
			RestartCount float64 `json:"RestartCount"`
		} `json:"State"`
		RestartCount float64 `json:"RestartCount"` // compact variants
	}
	if json.Unmarshal(data, &records) != nil {
		return map[string]float64{}
	}
	counts := make(map[string]float64, len(records))
	for _, record := range records {
		name := strings.TrimPrefix(strings.TrimSpace(record.Name), "/")
		if name != "" {
			count := record.RestartCount
			if count == 0 {
				count = record.State.RestartCount
			}
			counts[name] = count
		}
	}
	return counts
}
