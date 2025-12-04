// Package phases provides phase normalization and descriptor types.
package phases

import (
	"fmt"
	"strings"
	"time"
)

// Descriptor describes a test phase from the server catalog.
type Descriptor struct {
	Name                  string `json:"name"`
	Optional              bool   `json:"optional"`
	Description           string `json:"description"`
	Source                string `json:"source"`
	DefaultTimeoutSeconds int    `json:"defaultTimeoutSeconds"`
}

// AllowedPhases enumerates the standard phase set the planner understands.
var AllowedPhases = []string{
	"structure",
	"dependencies",
	"smoke",
	"unit",
	"integration",
	"e2e",
	"business",
	"performance",
}

// NormalizeSelection validates and deduplicates a phase selection.
// Returns nil if "all" or "default" is requested, signaling server defaults.
func NormalizeSelection(phases []string) ([]string, error) {
	if len(phases) == 0 {
		return nil, nil
	}

	// Allow "all" (and synonyms) to request the default planner behavior.
	for _, phase := range phases {
		if p := NormalizeName(phase); p == "all" || p == "default" {
			return nil, nil
		}
	}

	allowed := make(map[string]struct{}, len(AllowedPhases))
	for _, phase := range AllowedPhases {
		allowed[phase] = struct{}{}
	}

	var normalized []string
	seen := make(map[string]struct{}, len(phases))
	for _, phase := range phases {
		normalizedName := NormalizeName(phase)
		if normalizedName == "" {
			continue
		}
		normalizedName = NormalizeAlias(normalizedName)
		if _, exists := allowed[normalizedName]; !exists {
			return nil, fmt.Errorf("unknown phase '%s' (allowed: %s)", phase, strings.Join(AllowedPhases, ","))
		}
		if _, dup := seen[normalizedName]; dup {
			continue
		}
		seen[normalizedName] = struct{}{}
		normalized = append(normalized, normalizedName)
	}
	return normalized, nil
}

// NormalizeName lowercases and trims a phase name.
func NormalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// NormalizeAlias maps phase aliases to canonical names.
func NormalizeAlias(name string) string {
	switch name {
	case "e2e":
		return "integration"
	default:
		return name
	}
}

// MakeDescriptorMaps builds lookup maps from phase descriptors.
func MakeDescriptorMaps(descriptors []Descriptor) (map[string]Descriptor, map[string]time.Duration) {
	descMap := make(map[string]Descriptor, len(descriptors))
	targets := make(map[string]time.Duration, len(descriptors))
	for _, desc := range descriptors {
		key := strings.ToLower(strings.TrimSpace(desc.Name))
		if key == "" {
			continue
		}
		if _, exists := descMap[key]; !exists {
			descMap[key] = desc
		}
		if desc.DefaultTimeoutSeconds > 0 {
			targets[key] = time.Duration(desc.DefaultTimeoutSeconds) * time.Second
		}
	}
	if len(targets) == 0 {
		targets = DefaultTargetDurations()
	}
	return descMap, targets
}

// DefaultTargetDurations returns fallback target durations when descriptors are unavailable.
func DefaultTargetDurations() map[string]time.Duration {
	return map[string]time.Duration{
		"structure":    120 * time.Second,
		"dependencies": 60 * time.Second,
		"smoke":        90 * time.Second,
		"unit":         120 * time.Second,
		"integration":  600 * time.Second,
		"business":     120 * time.Second,
		"performance":  60 * time.Second,
	}
}

// TargetDurations extracts target durations from descriptors for external callers.
func TargetDurations(descriptors []Descriptor) map[string]time.Duration {
	_, targets := MakeDescriptorMaps(descriptors)
	return targets
}
