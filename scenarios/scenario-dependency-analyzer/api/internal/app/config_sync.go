package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	orderedmap "github.com/iancoleman/orderedmap"

	appconfig "scenario-dependency-analyzer/internal/config"
	types "scenario-dependency-analyzer/internal/types"
)

func applyDetectedDiffs(scenarioName string, analysis *types.DependencyAnalysisResponse, applyResources, applyScenarios bool) (map[string]interface{}, error) {
	updates := map[string]interface{}{}
	envCfg := appconfig.Load()
	scenarioPath := filepath.Join(envCfg.ScenariosDir, scenarioName)
	cfg, err := appconfig.LoadServiceConfig(scenarioPath)
	if err != nil {
		return nil, err
	}
	if os.Getenv("SCENARIO_ANALYZER_TRACE") == "1" {
		logTrace("[dependency-apply] %s declared resources=%d scenarios=%d", scenarioName, len(cfg.Dependencies.Resources), len(cfg.Dependencies.Scenarios))
	}
	if applyDiffsHook != nil {
		applyDiffsHook(scenarioName, cfg)
	}

	rawConfig, err := loadRawServiceConfigMap(scenarioPath)
	if err != nil {
		return nil, err
	}
	rawDependencies := ensureOrderedMap(rawConfig, "dependencies")
	rawResources := ensureOrderedMap(rawDependencies, "resources")
	rawScenarios := ensureOrderedMap(rawDependencies, "scenarios")
	rawDependencies.Set("resources", rawResources)
	rawDependencies.Set("scenarios", rawScenarios)
	if os.Getenv("SCENARIO_ANALYZER_TRACE") == "1" {
		logTrace("[dependency-apply] raw resources keys=%d", len(rawResources.Keys()))
	}
	if len(rawResources.Keys()) == 0 && len(cfg.Dependencies.Resources) > 0 {
		rawResources = orderedMapFromStruct(cfg.Dependencies.Resources)
		rawDependencies.Set("resources", rawResources)
	}
	if len(rawScenarios.Keys()) == 0 && len(cfg.Dependencies.Scenarios) > 0 {
		rawScenarios = orderedMapFromStruct(cfg.Dependencies.Scenarios)
		rawDependencies.Set("scenarios", rawScenarios)
	}

	resourcesAdded := []string{}
	if applyResources {
		missing := map[string]struct{}{}
		for _, drift := range analysis.ResourceDiff.Missing {
			missing[drift.Name] = struct{}{}
		}
		if len(missing) > 0 {
			if cfg.Dependencies.Resources == nil {
				cfg.Dependencies.Resources = map[string]types.Resource{}
			}
			for _, dep := range analysis.DetectedResources {
				if _, ok := missing[dep.DependencyName]; !ok {
					continue
				}
				if _, exists := cfg.Dependencies.Resources[dep.DependencyName]; exists {
					continue
				}
				typeHint := "custom"
				if dep.Configuration != nil {
					if val, ok := dep.Configuration["resource_type"].(string); ok && val != "" {
						typeHint = val
					}
				}
				description := fmt.Sprintf("Auto-detected via analyzer (%s)", dep.AccessMethod)
				cfg.Dependencies.Resources[dep.DependencyName] = types.Resource{
					Type:     typeHint,
					Enabled:  true,
					Required: true,
					Purpose:  description,
				}
				resourceEntry := orderedmap.New()
				resourceEntry.Set("type", typeHint)
				resourceEntry.Set("enabled", true)
				resourceEntry.Set("required", true)
				resourceEntry.Set("description", description)
				rawResources.Set(dep.DependencyName, resourceEntry)
				resourcesAdded = append(resourcesAdded, dep.DependencyName)
			}
		}
	}

	scenariosAdded := []string{}
	if applyScenarios {
		missing := map[string]struct{}{}
		for _, drift := range analysis.ScenarioDiff.Missing {
			missing[drift.Name] = struct{}{}
		}
		if len(missing) > 0 {
			if cfg.Dependencies.Scenarios == nil {
				cfg.Dependencies.Scenarios = map[string]types.ScenarioDependencySpec{}
			}
			for _, dep := range analysis.Scenarios {
				if _, ok := missing[dep.DependencyName]; !ok {
					continue
				}
				if _, exists := cfg.Dependencies.Scenarios[dep.DependencyName]; exists {
					continue
				}
				description := fmt.Sprintf("Auto-detected dependency via %s", dep.AccessMethod)
				version, versionRange := resolveScenarioVersionSpec(dep.DependencyName)
				cfg.Dependencies.Scenarios[dep.DependencyName] = types.ScenarioDependencySpec{
					Required:     true,
					Version:      version,
					VersionRange: versionRange,
					Description:  description,
				}
				scenarioEntry := orderedmap.New()
				scenarioEntry.Set("required", true)
				scenarioEntry.Set("version", version)
				scenarioEntry.Set("versionRange", versionRange)
				scenarioEntry.Set("description", description)
				rawScenarios.Set(dep.DependencyName, scenarioEntry)
				scenariosAdded = append(scenariosAdded, dep.DependencyName)
			}
		}
	}

	changed := len(resourcesAdded) > 0 || len(scenariosAdded) > 0
	if changed {
		reordered := reorderTopLevelKeys(rawConfig)
		if err := writeRawServiceConfigMap(scenarioPath, reordered); err != nil {
			return nil, err
		}
		if err := updateScenarioMetadata(scenarioName, cfg, scenarioPath); err != nil {
			return nil, err
		}
		refreshDependencyCatalogs()
	}

	updates["changed"] = changed
	updates["resources_added"] = resourcesAdded
	updates["scenarios_added"] = scenariosAdded
	return updates, nil
}

func applyOptimizationRecommendations(scenarioName string, recs []types.OptimizationRecommendation) (map[string]interface{}, error) {
	if len(recs) == 0 {
		return map[string]interface{}{"changed": false}, nil
	}
	envCfg := appconfig.Load()
	scenarioPath := filepath.Join(envCfg.ScenariosDir, scenarioName)
	removedResources := []string{}
	scenariosRemoved := []string{}
	for _, rec := range recs {
		action, _ := rec.RecommendedState["action"].(string)
		switch action {
		case "remove_resource":
			resourceName, _ := rec.RecommendedState["resource_name"].(string)
			if resourceName == "" {
				continue
			}
			removed, err := removeResourceFromServiceConfig(scenarioPath, resourceName)
			if err != nil {
				return nil, err
			}
			if removed {
				removedResources = append(removedResources, resourceName)
			}
		case "remove_scenario_dependency":
			depName, _ := rec.RecommendedState["scenario_name"].(string)
			if depName == "" {
				continue
			}
			removed, err := removeScenarioDependencyFromServiceConfig(scenarioPath, depName)
			if err != nil {
				return nil, err
			}
			if removed {
				scenariosRemoved = append(scenariosRemoved, depName)
			}
		}
	}
	changed := len(removedResources) > 0 || len(scenariosRemoved) > 0
	if changed {
		cfg, err := appconfig.LoadServiceConfig(scenarioPath)
		if err == nil {
			_ = updateScenarioMetadata(scenarioName, cfg, scenarioPath)
		}
		refreshDependencyCatalogs()
	}
	return map[string]interface{}{
		"changed":           changed,
		"resources_removed": removedResources,
		"scenarios_removed": scenariosRemoved,
	}, nil
}

// removeDependencyFromServiceConfig removes a dependency entry from the specified
// section ("resources" or "scenarios") in the service.json dependencies block.
func removeDependencyFromServiceConfig(scenarioPath, sectionKey, dependencyName string) (bool, error) {
	raw, err := loadRawServiceConfigMap(scenarioPath)
	if err != nil {
		return false, err
	}
	deps := ensureOrderedMap(raw, "dependencies")
	section := ensureOrderedMap(deps, sectionKey)
	if _, ok := section.Get(dependencyName); !ok {
		return false, nil
	}
	section.Delete(dependencyName)
	reordered := reorderTopLevelKeys(raw)
	if err := writeRawServiceConfigMap(scenarioPath, reordered); err != nil {
		return false, err
	}
	return true, nil
}

func removeResourceFromServiceConfig(scenarioPath, resourceName string) (bool, error) {
	return removeDependencyFromServiceConfig(scenarioPath, "resources", resourceName)
}

func removeScenarioDependencyFromServiceConfig(scenarioPath, scenarioName string) (bool, error) {
	return removeDependencyFromServiceConfig(scenarioPath, "scenarios", scenarioName)
}

func ensureOrderedMap(parent *orderedmap.OrderedMap, key string) *orderedmap.OrderedMap {
	if parent == nil {
		return orderedmap.New()
	}
	if val, ok := parent.Get(key); ok {
		switch typed := val.(type) {
		case *orderedmap.OrderedMap:
			return typed
		case orderedmap.OrderedMap:
			converted := orderedmap.New()
			for _, childKey := range typed.Keys() {
				if childVal, ok := typed.Get(childKey); ok {
					converted.Set(childKey, childVal)
				}
			}
			parent.Set(key, converted)
			return converted
		case map[string]interface{}:
			converted := orderedmap.New()
			for k, v := range typed {
				converted.Set(k, v)
			}
			parent.Set(key, converted)
			return converted
		}
	}
	child := orderedmap.New()
	parent.Set(key, child)
	return child
}

func reorderTopLevelKeys(cfg *orderedmap.OrderedMap) *orderedmap.OrderedMap {
	if cfg == nil {
		return orderedmap.New()
	}
	preferred := []string{"$schema", "version", "service", "ports", "lifecycle", "dependencies"}
	reordered := orderedmap.New()
	seen := map[string]struct{}{}
	for _, key := range preferred {
		if val, ok := cfg.Get(key); ok {
			reordered.Set(key, val)
			seen[key] = struct{}{}
		}
	}
	for _, key := range cfg.Keys() {
		if _, ok := seen[key]; ok {
			continue
		}
		if val, ok := cfg.Get(key); ok {
			reordered.Set(key, val)
		}
	}
	return reordered
}

func cloneOrderedMap(src *orderedmap.OrderedMap) *orderedmap.OrderedMap {
	if src == nil {
		return orderedmap.New()
	}
	clone := orderedmap.New()
	for _, key := range src.Keys() {
		if val, ok := src.Get(key); ok {
			clone.Set(key, val)
		}
	}
	return clone
}

func resolveScenarioVersionSpec(dependencyName string) (string, string) {
	scenarioPath := filepath.Join(appconfig.Load().ScenariosDir, dependencyName)
	cfg, err := appconfig.LoadServiceConfig(scenarioPath)
	if err != nil {
		return "", ">=0.0.0"
	}
	version := strings.TrimSpace(cfg.Service.Version)
	if version == "" {
		return "", ">=0.0.0"
	}
	return version, fmt.Sprintf(">=%s", version)
}

func orderedMapFromStruct(value interface{}) *orderedmap.OrderedMap {
	ordered := orderedmap.New()
	if value == nil {
		return ordered
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return ordered
	}
	if err := json.Unmarshal(payload, ordered); err != nil {
		return orderedmap.New()
	}
	return ordered
}

func loadRawServiceConfigMap(scenarioPath string) (*orderedmap.OrderedMap, error) {
	serviceConfigPath := filepath.Join(scenarioPath, ".vrooli", "service.json")
	data, err := os.ReadFile(serviceConfigPath)
	if err != nil {
		return nil, err
	}
	raw := orderedmap.New()
	if err := json.Unmarshal(data, raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func writeRawServiceConfigMap(scenarioPath string, cfg *orderedmap.OrderedMap) error {
	serviceConfigPath := filepath.Join(scenarioPath, ".vrooli", "service.json")
	payload, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	payload = bytes.TrimRight(payload, "\n")
	payload = bytes.ReplaceAll(payload, []byte(`\u003c`), []byte("<"))
	payload = bytes.ReplaceAll(payload, []byte(`\u003e`), []byte(">"))
	payload = bytes.ReplaceAll(payload, []byte(`\u0026`), []byte("&"))
	return os.WriteFile(serviceConfigPath, payload, 0644)
}

func toStringSlice(value interface{}) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []interface{}:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	default:
		return nil
	}
}

func logTrace(format string, args ...interface{}) {
	if os.Getenv("SCENARIO_ANALYZER_TRACE") != "1" {
		return
	}
	log.Printf(format, args...)
}
