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
	"lint",
	"docs",
	"smoke",
	"unit",
	"integration",
	"playbooks",
	"e2e", // alias for playbooks
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

// NamesFromDescriptors returns the ordered, normalized list of phase names
// provided by the server catalog.
func NamesFromDescriptors(descriptors []Descriptor) []string {
	var names []string
	seen := make(map[string]struct{}, len(descriptors))
	for _, desc := range descriptors {
		name := NormalizeAlias(NormalizeName(desc.Name))
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	return names
}

// ApplySkip removes any phases present in the skip list, honoring aliases.
func ApplySkip(phases, skip []string) []string {
	if len(phases) == 0 || len(skip) == 0 {
		// Still de-duplicate aliases to avoid repeated entries.
		return dedupeNormalized(phases)
	}
	skipSet := make(map[string]struct{}, len(skip))
	for _, s := range skip {
		name := NormalizeAlias(NormalizeName(s))
		if name == "" {
			continue
		}
		skipSet[name] = struct{}{}
	}

	seen := make(map[string]struct{}, len(phases))
	var filtered []string
	for _, phase := range phases {
		name := NormalizeAlias(NormalizeName(phase))
		if name == "" {
			continue
		}
		if _, blocked := skipSet[name]; blocked {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		filtered = append(filtered, name)
	}
	return filtered
}

func dedupeNormalized(phases []string) []string {
	if len(phases) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(phases))
	var result []string
	for _, phase := range phases {
		name := NormalizeAlias(NormalizeName(phase))
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, name)
	}
	return result
}

// NormalizeName lowercases and trims a phase name.
func NormalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// NormalizeAlias maps phase aliases to canonical names.
func NormalizeAlias(name string) string {
	switch name {
	case "e2e":
		return "playbooks"
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
		"lint":         30 * time.Second,
		"docs":         60 * time.Second,
		"smoke":        90 * time.Second,
		"unit":         120 * time.Second,
		"integration":  600 * time.Second,
		"playbooks":    900 * time.Second, // BAS workflow execution can be slow
		"business":     120 * time.Second,
		"performance":  60 * time.Second,
	}
}

// TargetDurations extracts target durations from descriptors for external callers.
func TargetDurations(descriptors []Descriptor) map[string]time.Duration {
	_, targets := MakeDescriptorMaps(descriptors)
	return targets
}
