package app

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"

	types "scenario-dependency-analyzer/internal/types"
)

func (a *Analyzer) AnalyzeScenario(scenarioName string) (*types.DependencyAnalysisResponse, error) {
	if a == nil {
		return nil, fmt.Errorf("analyzer not initialized")
	}
	scenarioPath := filepath.Join(a.cfg.ScenariosDir, scenarioName)
	serviceConfig, err := loadServiceConfigFromFile(scenarioPath)
	if err != nil {
		return nil, err
	}

	response := &types.DependencyAnalysisResponse{
		Scenario:              scenarioName,
		Resources:             []types.ScenarioDependency{},
		DetectedResources:     []types.ScenarioDependency{},
		Scenarios:             []types.ScenarioDependency{},
		DeclaredScenarioSpecs: map[string]types.ScenarioDependencySpec{},
		SharedWorkflows:       []types.ScenarioDependency{},
		TransitiveDepth:       0,
		ResourceDiff:          types.DependencyDiff{},
		ScenarioDiff:          types.DependencyDiff{},
	}

	declaredResources := extractDeclaredResources(scenarioName, serviceConfig)
	response.Resources = declaredResources
	response.DeclaredScenarioSpecs = normalizeScenarioSpecs(serviceConfig.Dependencies.Scenarios)

	if a.detector == nil {
		return nil, fmt.Errorf("detector not initialized")
	}

	detectedResources, err := a.detector.ScanResources(scenarioPath, scenarioName, serviceConfig)
	if err != nil {
		log.Printf("Warning: failed to scan for resource usage: %v", err)
	} else {
		response.DetectedResources = detectedResources
	}

	scenarioDeps, err := a.detector.ScanScenarioDependencies(scenarioPath, scenarioName)
	if err != nil {
		log.Printf("Warning: failed to scan for scenario dependencies: %v", err)
	} else {
		response.Scenarios = append(response.Scenarios, scenarioDeps...)
	}

	workflowDeps, err := a.detector.ScanSharedWorkflows(scenarioPath, scenarioName)
	if err != nil {
		log.Printf("Warning: failed to scan for shared workflows: %v", err)
	} else {
		response.SharedWorkflows = append(response.SharedWorkflows, workflowDeps...)
	}

	declaredScenarioDeps := convertDeclaredScenariosToDependencies(scenarioName, response.DeclaredScenarioSpecs)
	response.ResourceDiff = buildResourceDiff(resolvedResourceMap(serviceConfig), response.DetectedResources)
	response.ScenarioDiff = buildScenarioDiff(response.DeclaredScenarioSpecs, response.Scenarios)

	if a.store != nil {
		if err := a.store.StoreDependencies(response, declaredScenarioDeps); err != nil {
			log.Printf("Warning: failed to store dependencies in database: %v", err)
		}
	} else {
		log.Printf("Warning: store not initialized; dependency metadata not persisted")
	}

	if a.store != nil {
		if err := a.store.UpdateScenarioMetadata(scenarioName, serviceConfig, scenarioPath); err != nil {
			log.Printf("Warning: failed to update scenario metadata: %v", err)
		}
	}

	if deploymentReport := buildDeploymentReport(scenarioName, scenarioPath, a.cfg.ScenariosDir, serviceConfig); deploymentReport != nil {
		response.DeploymentReport = deploymentReport
		if err := persistDeploymentReport(scenarioPath, deploymentReport); err != nil {
			log.Printf("Warning: failed to persist deployment report: %v", err)
		}
	}

	return response, nil
}

func (a *Analyzer) AnalyzeAllScenarios() (map[string]*types.DependencyAnalysisResponse, error) {
	if a == nil {
		return nil, fmt.Errorf("analyzer not initialized")
	}
	results := make(map[string]*types.DependencyAnalysisResponse)
	scenariosDir := a.cfg.ScenariosDir
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read scenarios directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		scenarioName := entry.Name()
		serviceConfigPath := filepath.Join(scenariosDir, scenarioName, ".vrooli", "service.json")
		if _, err := os.Stat(serviceConfigPath); os.IsNotExist(err) {
			continue
		}

		analysis, err := a.AnalyzeScenario(scenarioName)
		if err != nil {
			log.Printf("Warning: failed to analyze scenario %s: %v", scenarioName, err)
			continue
		}

		results[scenarioName] = analysis
	}

	return results, nil
}

func analyzeScenario(scenarioName string) (*types.DependencyAnalysisResponse, error) {
	analyzer := analyzerInstance()
	if analyzer == nil {
		return nil, fmt.Errorf("analyzer not initialized")
	}
	return analyzer.AnalyzeScenario(scenarioName)
}

func analyzeAllScenarios() (map[string]*types.DependencyAnalysisResponse, error) {
	analyzer := analyzerInstance()
	if analyzer == nil {
		return nil, fmt.Errorf("analyzer not initialized")
	}
	return analyzer.AnalyzeAllScenarios()
}

func extractDeclaredResources(scenarioName string, cfg *types.ServiceConfig) []types.ScenarioDependency {
	resources := resolvedResourceMap(cfg)
	declared := make([]types.ScenarioDependency, 0, len(resources))
	for name, resource := range resources {
		declared = append(declared, types.ScenarioDependency{
			ID:             uuid.New().String(),
			ScenarioName:   scenarioName,
			DependencyType: "resource",
			DependencyName: name,
			Required:       resource.Required,
			Purpose:        resource.Purpose,
			AccessMethod:   "declared",
			Configuration: map[string]interface{}{
				"type":       resource.Type,
				"enabled":    resource.Enabled,
				"purpose":    resource.Purpose,
				"declared":   true,
				"models":     resource.Models,
				"init_files": resource.Initialization,
			},
			DiscoveredAt: time.Now(),
			LastVerified: time.Now(),
		})
	}
	sort.Slice(declared, func(i, j int) bool { return declared[i].DependencyName < declared[j].DependencyName })
	return declared
}

func buildResourceDiff(declared map[string]types.Resource, detected []types.ScenarioDependency) types.DependencyDiff {
	declaredSet := map[string]types.Resource{}
	for name, cfg := range declared {
		declaredSet[name] = cfg
	}
	detectedSet := map[string]types.ScenarioDependency{}
	for _, dep := range detected {
		detectedSet[dep.DependencyName] = dep
	}

	missing := []types.DependencyDrift{}
	for name, dep := range detectedSet {
		if _, ok := declaredSet[name]; !ok {
			missing = append(missing, types.DependencyDrift{
				Name:    name,
				Details: dep.Configuration,
			})
		}
	}

	extra := []types.DependencyDrift{}
	for name, cfg := range declaredSet {
		if _, ok := detectedSet[name]; !ok {
			extra = append(extra, types.DependencyDrift{
				Name: name,
				Details: map[string]interface{}{
					"type":     cfg.Type,
					"required": cfg.Required,
				},
			})
		}
	}

	sort.Slice(missing, func(i, j int) bool { return missing[i].Name < missing[j].Name })
	sort.Slice(extra, func(i, j int) bool { return extra[i].Name < extra[j].Name })

	return types.DependencyDiff{Missing: missing, Extra: extra}
}

func buildScenarioDiff(declared map[string]types.ScenarioDependencySpec, detected []types.ScenarioDependency) types.DependencyDiff {
	declaredSet := map[string]types.ScenarioDependencySpec{}
	for name, spec := range declared {
		declaredSet[name] = spec
	}
	detectedSet := map[string]types.ScenarioDependency{}
	for _, dep := range detected {
		detectedSet[dep.DependencyName] = dep
	}

	missing := []types.DependencyDrift{}
	for name, dep := range detectedSet {
		if _, ok := declaredSet[name]; !ok {
			missing = append(missing, types.DependencyDrift{
				Name:    name,
				Details: dep.Configuration,
			})
		}
	}

	extra := []types.DependencyDrift{}
	for name, spec := range declaredSet {
		if _, ok := detectedSet[name]; !ok {
			extra = append(extra, types.DependencyDrift{
				Name: name,
				Details: map[string]interface{}{
					"required": spec.Required,
					"version":  spec.Version,
				},
			})
		}
	}

	sort.Slice(missing, func(i, j int) bool { return missing[i].Name < missing[j].Name })
	sort.Slice(extra, func(i, j int) bool { return extra[i].Name < extra[j].Name })

	return types.DependencyDiff{Missing: missing, Extra: extra}
}

func filterDetectedDependencies(deps []types.ScenarioDependency) []types.ScenarioDependency {
	filtered := []types.ScenarioDependency{}
	for _, dep := range deps {
		source := ""
		if dep.Configuration != nil {
			if val, ok := dep.Configuration["source"].(string); ok {
				source = val
			}
		}
		if source == "declared" {
			continue
		}
		filtered = append(filtered, dep)
	}
	return filtered
}

func normalizeScenarioSpecs(specs map[string]types.ScenarioDependencySpec) map[string]types.ScenarioDependencySpec {
	if specs == nil {
		return map[string]types.ScenarioDependencySpec{}
	}
	return specs
}

func convertDeclaredScenariosToDependencies(scenarioName string, specs map[string]types.ScenarioDependencySpec) []types.ScenarioDependency {
	if len(specs) == 0 {
		return nil
	}
	deps := make([]types.ScenarioDependency, 0, len(specs))
	for name, spec := range specs {
		deps = append(deps, types.ScenarioDependency{
			ID:             uuid.New().String(),
			ScenarioName:   scenarioName,
			DependencyType: "scenario",
			DependencyName: name,
			Required:       spec.Required,
			Purpose:        spec.Description,
			AccessMethod:   "declared",
			Configuration: map[string]interface{}{
				"version":       spec.Version,
				"version_range": spec.VersionRange,
				"declared":      true,
			},
			DiscoveredAt: time.Now(),
			LastVerified: time.Now(),
		})
	}
	sort.Slice(deps, func(i, j int) bool { return deps[i].DependencyName < deps[j].DependencyName })
	return deps
}

func resolvedResourceMap(cfg *types.ServiceConfig) map[string]types.Resource {
	if cfg.Dependencies.Resources != nil && len(cfg.Dependencies.Resources) > 0 {
		return cfg.Dependencies.Resources
	}
	if cfg.Resources == nil {
		return map[string]types.Resource{}
	}
	return cfg.Resources
}
