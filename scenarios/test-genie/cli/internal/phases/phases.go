// Package phases provides phase normalization and descriptor types.
package phases

import (
	"fmt"
	"strings"
	"time"

	catalog "test-genie/internal/orchestrator/phases"
)

// Descriptor describes a test phase from the server catalog.
type Descriptor struct {
	Name                  string `json:"name"`
	Optional              bool   `json:"optional"`
	Description           string `json:"description"`
	Source                string `json:"source"`
	DefaultTimeoutSeconds int    `json:"defaultTimeoutSeconds"`
}

// allowedPhaseNames returns the canonical phase set the planner understands,
// derived from the test-genie catalog (the single source of truth).
func allowedPhaseNames() []string {
	return catalog.ValidPhaseNames()
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

	allowedNames := allowedPhaseNames()
	allowed := make(map[string]struct{}, len(allowedNames))
	for _, phase := range allowedNames {
		allowed[phase] = struct{}{}
	}

	var normalized []string
	seen := make(map[string]struct{}, len(phases))
	for _, phase := range phases {
		normalizedName := NormalizeName(phase)
		if normalizedName == "" {
			continue
		}
		if _, exists := allowed[normalizedName]; !exists {
			return nil, fmt.Errorf("unknown phase '%s' (allowed: %s)", phase, strings.Join(allowedNames, ","))
		}
		if _, dup := seen[normalizedName]; dup {
			continue
		}
		seen[normalizedName] = struct{}{}
		normalized = append(normalized, normalizedName)
	}
	return normalized, nil
}

// DeprecatedAliasWarnings reports user-facing migration notes for phase names
// that are still accepted but no longer canonical.
func DeprecatedAliasWarnings(phases []string) []string {
	if len(phases) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, 1)
	var warnings []string
	for _, phase := range phases {
		key := strings.ToLower(strings.TrimSpace(phase))
		switch key {
		case "playbooks", "playbook", "e2e":
			if _, ok := seen["workflow"]; ok {
				continue
			}
			seen["workflow"] = struct{}{}
			warnings = append(warnings, "phase \"playbooks\" is deprecated; use \"workflow\"")
		}
	}
	return warnings
}

// NamesFromDescriptors returns the ordered, normalized list of phase names
// provided by the server catalog.
func NamesFromDescriptors(descriptors []Descriptor) []string {
	var names []string
	seen := make(map[string]struct{}, len(descriptors))
	for _, desc := range descriptors {
		name := NormalizeName(desc.Name)
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
		name := NormalizeName(s)
		if name == "" {
			continue
		}
		skipSet[name] = struct{}{}
	}

	seen := make(map[string]struct{}, len(phases))
	var filtered []string
	for _, phase := range phases {
		name := NormalizeName(phase)
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
		name := NormalizeName(phase)
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
	return catalog.NormalizeKey(name)
}

// MakeDescriptorMaps builds lookup maps from phase descriptors.
func MakeDescriptorMaps(descriptors []Descriptor) (map[string]Descriptor, map[string]time.Duration) {
	descMap := make(map[string]Descriptor, len(descriptors))
	targets := make(map[string]time.Duration, len(descriptors))
	for _, desc := range descriptors {
		key := NormalizeName(desc.Name)
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
	return descMap, targets
}
