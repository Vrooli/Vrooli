package interfacegraph

import (
	"path"
	"sort"
	"strings"
)

type Attributor struct {
	scenarios map[string]struct{}
}

func NewAttributor(scenarios []string) Attributor {
	set := make(map[string]struct{}, len(scenarios))
	for _, scenario := range scenarios {
		scenario = strings.TrimSpace(scenario)
		if scenario != "" && !isSharedProtoPackage(scenario) {
			set[scenario] = struct{}{}
		}
	}
	return Attributor{scenarios: set}
}

func (a Attributor) AddScenario(scenario string) {
	if a.scenarios == nil {
		a.scenarios = map[string]struct{}{}
	}
	scenario = strings.TrimSpace(scenario)
	if scenario != "" && !isSharedProtoPackage(scenario) {
		a.scenarios[scenario] = struct{}{}
	}
}

func (a Attributor) Scenarios() []string {
	out := make([]string, 0, len(a.scenarios))
	for scenario := range a.scenarios {
		out = append(out, scenario)
	}
	sort.Strings(out)
	return out
}

// IsKnown reports whether name is a known scenario (not a shared/owned proto
// package). Used to filter proto-import edges whose target is a package name
// rather than a scenario (e.g. architecture-cartographer's own "architecture"
// proto package), mirroring the filtering the go-import path gets via Attribute.
func (a Attributor) IsKnown(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	_, ok := a.scenarios[name]
	return ok
}

func (a Attributor) Attribute(importPath string) (string, bool) {
	cleaned := cleanImportPath(importPath)
	if cleaned == "" {
		return "", false
	}
	candidates := []string{
		scenarioAfter(cleaned, "packages/proto/gen/go"),
		scenarioAfter(cleaned, "packages/proto/schemas"),
		scenarioAfter(cleaned, "schemas"),
		scenarioAfter(cleaned, "scenarios"),
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, ok := a.scenarios[candidate]; ok {
			return candidate, true
		}
	}
	return "", false
}

func cleanImportPath(importPath string) string {
	importPath = strings.TrimSpace(importPath)
	importPath = strings.Trim(importPath, "`\"'")
	importPath = strings.ReplaceAll(importPath, "\\", "/")
	importPath = strings.TrimPrefix(importPath, "./")
	return path.Clean(importPath)
}

func scenarioAfter(importPath, marker string) string {
	parts := strings.Split(importPath, "/")
	markerParts := strings.Split(marker, "/")
	for i := 0; i+len(markerParts) < len(parts); i++ {
		if sameParts(parts[i:i+len(markerParts)], markerParts) {
			return parts[i+len(markerParts)]
		}
	}
	return ""
}

func isSharedProtoPackage(name string) bool {
	switch strings.TrimSpace(name) {
	case "common":
		return true
	default:
		return false
	}
}

func sameParts(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
