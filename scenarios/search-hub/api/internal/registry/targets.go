package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MaturityTarget is the Search Hub-owned discovery result used by the
// maturity scan and its downstream readers. It deliberately includes
// capability-only scenarios, which are expected to fail validation with an
// explicit missing-descriptor finding rather than disappear from the fleet.
type MaturityTarget struct {
	Scenario            string
	Path                string
	ApplicabilityReason string
}

var searchCapabilityTokens = []string{"search", "ai-search"}

// DiscoverMaturityTargets returns the same descriptor/capability target set
// used by the maturity scan. Keeping discovery in the Search Hub API makes
// this contract available to consumers without asking them to inspect the
// repository or infer membership from routing telemetry.
func DiscoverMaturityTargets(repoRoot string) ([]MaturityTarget, error) {
	entries, err := os.ReadDir(filepath.Join(repoRoot, "scenarios"))
	if err != nil {
		return nil, fmt.Errorf("read scenarios directory: %w", err)
	}
	targets := make([]MaturityTarget, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		scenarioPath := filepath.Join(repoRoot, "scenarios", entry.Name())
		reason, applicable, err := searchApplicability(scenarioPath)
		if err != nil {
			return nil, fmt.Errorf("inspect %s search applicability: %w", entry.Name(), err)
		}
		if applicable {
			targets = append(targets, MaturityTarget{Scenario: entry.Name(), Path: scenarioPath, ApplicabilityReason: reason})
		}
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].Scenario < targets[j].Scenario })
	return targets, nil
}

// MergeRegisteredMaturityTargets adds registered provider groups that do not
// yet have a local descriptor/capability match. Search Hub's CLI scan uses the
// same union so a registered provider with a missing source descriptor remains
// visible as an explicit validation target.
func MergeRegisteredMaturityTargets(targets []MaturityTarget, providerGroups []string, repoRoot string) []MaturityTarget {
	byScenario := make(map[string]MaturityTarget, len(targets)+len(providerGroups))
	for _, target := range targets {
		byScenario[target.Scenario] = target
	}
	for _, group := range providerGroups {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		path := filepath.Join(repoRoot, "scenarios", group)
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			continue
		}
		if target, ok := byScenario[group]; ok {
			if !strings.Contains(target.ApplicabilityReason, "registered-provider") {
				target.ApplicabilityReason += "+registered-provider"
				byScenario[group] = target
			}
			continue
		}
		byScenario[group] = MaturityTarget{Scenario: group, Path: path, ApplicabilityReason: "registered-provider"}
	}
	out := make([]MaturityTarget, 0, len(byScenario))
	for _, target := range byScenario {
		out = append(out, target)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Scenario < out[j].Scenario })
	return out
}

func searchApplicability(scenarioPath string) (string, bool, error) {
	descriptor := false
	if _, err := os.Stat(filepath.Join(scenarioPath, ".vrooli", "search.json")); err == nil {
		descriptor = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, err
	}
	caps := serviceCapabilities(scenarioPath)
	matched := make([]string, 0, len(searchCapabilityTokens))
	for _, token := range searchCapabilityTokens {
		if caps[token] {
			matched = append(matched, token)
		}
	}
	switch {
	case descriptor && len(matched) > 0:
		return "descriptor+capability:" + strings.Join(matched, ","), true, nil
	case descriptor:
		return "descriptor", true, nil
	case len(matched) > 0:
		return "capability:" + strings.Join(matched, ","), true, nil
	default:
		return "", false, nil
	}
}

func serviceCapabilities(scenarioPath string) map[string]bool {
	caps := map[string]bool{}
	raw, err := os.ReadFile(filepath.Join(scenarioPath, ".vrooli", "service.json"))
	if err != nil {
		return caps
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return caps
	}
	addCapabilities(caps, nestedStringSlice(doc, "service", "tags"))
	addCapabilities(caps, nestedStringSlice(doc, "service", "capabilities"))
	addCapabilities(caps, nestedStringSlice(doc, "capabilities"))
	return caps
}

func addCapabilities(caps map[string]bool, values []string) {
	for _, value := range values {
		if value = strings.ToLower(strings.TrimSpace(value)); value != "" {
			caps[value] = true
		}
	}
}

func nestedStringSlice(doc map[string]any, path ...string) []string {
	var current any = doc
	for _, segment := range path {
		obj, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = obj[segment]
	}
	values, ok := current.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if text, ok := value.(string); ok {
			out = append(out, text)
		}
	}
	return out
}
