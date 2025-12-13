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

func applyDetectedDiffs(scenarioName string, analysis *types.DependencyAnalysisResponse, applyResources, applyScenarios bool, depSvc *dependencyService) (map[string]interface{}, error) {
	if analysis == nil {
		return nil, fmt.Errorf("analysis is required")
	}

	ctx, err := newDependencyApplyContext(scenarioName)
	if err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}

	if applyDiffsHook != nil {
		applyDiffsHook(scenarioName, ctx.cfg)
	}

	resourcesAdded := []string{}
	if applyResources {
		resourcesAdded = ctx.applyMissingResources(analysis)
	}

	scenariosAdded := []string{}
	if applyScenarios {
		scenariosAdded = ctx.applyMissingScenarios(analysis)
	}

	changed := len(resourcesAdded) > 0 || len(scenariosAdded) > 0
	if changed {
		if err := ctx.persist(depSvc); err != nil {
			return nil, err
		}
	}

	updates["changed"] = changed
	updates["resources_added"] = resourcesAdded
	updates["scenarios_added"] = scenariosAdded
	return updates, nil
}

type dependencyApplyContext struct {
	scenarioName string
	scenarioPath string
	cfg          *types.ServiceConfig
	rawConfig    *orderedmap.OrderedMap
	rawResources *orderedmap.OrderedMap
	rawScenarios *orderedmap.OrderedMap
}

func newDependencyApplyContext(scenarioName string) (*dependencyApplyContext, error) {
	envCfg := appconfig.Load()
	scenarioPath := filepath.Join(envCfg.ScenariosDir, scenarioName)

	cfg, err := appconfig.LoadServiceConfig(scenarioPath)
	if err != nil {
		return nil, err
	}

	if os.Getenv("SCENARIO_ANALYZER_TRACE") == "1" {
		logTrace("[dependency-apply] %s declared resources=%d scenarios=%d", scenarioName, len(cfg.Dependencies.Resources), len(cfg.Dependencies.Scenarios))
	}

	rawConfig, err := loadRawServiceConfigMap(scenarioPath)
	if err != nil {
		return nil, err
	}

	ctx := &dependencyApplyContext{
		scenarioName: scenarioName,
		scenarioPath: scenarioPath,
		cfg:          cfg,
		rawConfig:    rawConfig,
	}
	ctx.prepareDependencySections()
	return ctx, nil
}

func (c *dependencyApplyContext) prepareDependencySections() {
	rawDependencies := ensureOrderedMap(c.rawConfig, "dependencies")
	c.rawResources = ensureOrderedMap(rawDependencies, "resources")
	c.rawScenarios = ensureOrderedMap(rawDependencies, "scenarios")
	rawDependencies.Set("resources", c.rawResources)
	rawDependencies.Set("scenarios", c.rawScenarios)

	if os.Getenv("SCENARIO_ANALYZER_TRACE") == "1" {
		logTrace("[dependency-apply] raw resources keys=%d", len(c.rawResources.Keys()))
	}

	if len(c.rawResources.Keys()) == 0 && len(c.cfg.Dependencies.Resources) > 0 {
		c.rawResources = orderedMapFromStruct(c.cfg.Dependencies.Resources)
		rawDependencies.Set("resources", c.rawResources)
	}
	if len(c.rawScenarios.Keys()) == 0 && len(c.cfg.Dependencies.Scenarios) > 0 {
		c.rawScenarios = orderedMapFromStruct(c.cfg.Dependencies.Scenarios)
		rawDependencies.Set("scenarios", c.rawScenarios)
	}
}

func (c *dependencyApplyContext) applyMissingResources(analysis *types.DependencyAnalysisResponse) []string {
	missing := driftSet(analysis.ResourceDiff.Missing)
	if len(missing) == 0 {
		return nil
	}

	added := []string{}
	for _, dep := range analysis.DetectedResources {
		if _, ok := missing[dep.DependencyName]; !ok {
			continue
		}

		if name, ok := c.addDetectedResource(dep); ok {
			added = append(added, name)
		}
	}

	return added
}

func (c *dependencyApplyContext) applyMissingScenarios(analysis *types.DependencyAnalysisResponse) []string {
	missing := driftSet(analysis.ScenarioDiff.Missing)
	if len(missing) == 0 {
		return nil
	}

	if c.cfg.Dependencies.Scenarios == nil {
		c.cfg.Dependencies.Scenarios = map[string]types.ScenarioDependencySpec{}
	}

	added := []string{}
	for _, dep := range analysis.Scenarios {
		if _, ok := missing[dep.DependencyName]; !ok {
			continue
		}

		if name, ok := c.addDetectedScenario(dep); ok {
			added = append(added, name)
		}
	}

	return added
}

func (c *dependencyApplyContext) addDetectedResource(dep types.ScenarioDependency) (string, bool) {
	if dep.DependencyName == "" {
		return "", false
	}
	if c.cfg.Dependencies.Resources == nil {
		c.cfg.Dependencies.Resources = map[string]types.Resource{}
	}
	if _, exists := c.cfg.Dependencies.Resources[dep.DependencyName]; exists {
		return "", false
	}

	typeHint := resourceTypeFromDetection(dep)
	description := fmt.Sprintf("Auto-detected via analyzer (%s)", dep.AccessMethod)

	c.cfg.Dependencies.Resources[dep.DependencyName] = types.Resource{
		Type:     typeHint,
		Enabled:  true,
		Required: true,
		Purpose:  description,
	}

	if c.rawResources == nil {
		c.rawResources = orderedmap.New()
	}
	c.rawResources.Set(dep.DependencyName, orderedResourceEntry(typeHint, description))

	return dep.DependencyName, true
}

func (c *dependencyApplyContext) addDetectedScenario(dep types.ScenarioDependency) (string, bool) {
	if dep.DependencyName == "" {
		return "", false
	}
	if c.cfg.Dependencies.Scenarios == nil {
		c.cfg.Dependencies.Scenarios = map[string]types.ScenarioDependencySpec{}
	}
	if _, exists := c.cfg.Dependencies.Scenarios[dep.DependencyName]; exists {
		return "", false
	}

	description := fmt.Sprintf("Auto-detected dependency via %s", dep.AccessMethod)
	version, versionRange := resolveScenarioVersionSpec(dep.DependencyName)
	c.cfg.Dependencies.Scenarios[dep.DependencyName] = types.ScenarioDependencySpec{
		Required:     true,
		Version:      version,
		VersionRange: versionRange,
		Description:  description,
	}

	if c.rawScenarios == nil {
		c.rawScenarios = orderedmap.New()
	}
	c.rawScenarios.Set(dep.DependencyName, orderedScenarioEntry(version, versionRange, description))

	return dep.DependencyName, true
}

func (c *dependencyApplyContext) persist(depSvc *dependencyService) error {
	reordered := reorderTopLevelKeys(c.rawConfig)
	if err := writeRawServiceConfigMap(c.scenarioPath, reordered); err != nil {
		return err
	}
	if depSvc == nil {
		depSvc = defaultDependencyService()
	}
	if err := depSvc.UpdateScenarioMetadata(c.scenarioName, c.cfg, c.scenarioPath); err != nil {
		return err
	}
	depSvc.RefreshCatalogs()
	return nil
}

func driftSet(drifts []types.DependencyDrift) map[string]struct{} {
	missing := map[string]struct{}{}
	for _, drift := range drifts {
		missing[drift.Name] = struct{}{}
	}
	return missing
}

func resourceTypeFromDetection(dep types.ScenarioDependency) string {
	if dep.Configuration == nil {
		return "custom"
	}
	if val, ok := dep.Configuration["resource_type"].(string); ok && val != "" {
		return val
	}
	return "custom"
}

func applyOptimizationRecommendations(scenarioName string, recs []types.OptimizationRecommendation, depSvc *dependencyService) (map[string]interface{}, error) {
	if len(recs) == 0 {
		return map[string]interface{}{"changed": false}, nil
	}
	envCfg := appconfig.Load()
	scenarioPath := filepath.Join(envCfg.ScenariosDir, scenarioName)
	if depSvc == nil {
		depSvc = defaultDependencyService()
	}
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
			_ = depSvc.UpdateScenarioMetadata(scenarioName, cfg, scenarioPath)
		}
		depSvc.RefreshCatalogs()
		depSvc.CleanupInvalidDependencies()
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

func logTrace(format string, args ...interface{}) {
	if os.Getenv("SCENARIO_ANALYZER_TRACE") != "1" {
		return
	}
	log.Printf(format, args...)
}

func orderedResourceEntry(typeHint, description string) *orderedmap.OrderedMap {
	entry := orderedmap.New()
	entry.Set("type", typeHint)
	entry.Set("enabled", true)
	entry.Set("required", true)
	entry.Set("description", description)
	return entry
}

func orderedScenarioEntry(version, versionRange, description string) *orderedmap.OrderedMap {
	entry := orderedmap.New()
	entry.Set("required", true)
	entry.Set("version", version)
	entry.Set("versionRange", versionRange)
	entry.Set("description", description)
	return entry
}
