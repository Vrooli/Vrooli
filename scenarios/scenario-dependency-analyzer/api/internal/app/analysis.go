package app

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	appconfig "scenario-dependency-analyzer/internal/config"
	"scenario-dependency-analyzer/internal/deployment"
	"scenario-dependency-analyzer/internal/seams"
	types "scenario-dependency-analyzer/internal/types"
)

func (a *Analyzer) AnalyzeScenario(scenarioName string) (*types.DependencyAnalysisResponse, error) {
	if a == nil {
		return nil, fmt.Errorf("analyzer not initialized")
	}
	scenarioPath := filepath.Join(a.cfg.ScenariosDir, scenarioName)
	serviceConfig, err := appconfig.LoadServiceConfig(scenarioPath)
	if err != nil {
		return nil, err
	}

	response := newAnalysisResponse(scenarioName, serviceConfig)
	if a.detector == nil {
		return nil, fmt.Errorf("detector not initialized")
	}

	a.collectDetections(response, scenarioPath, scenarioName, serviceConfig)

	declaredScenarioDeps := convertDeclaredScenariosToDependencies(scenarioName, response.DeclaredScenarioSpecs)
	response.ResourceDiff = buildResourceDiff(appconfig.ResolvedResourceMap(serviceConfig), response.DetectedResources)
	response.ScenarioDiff = buildScenarioDiff(response.DeclaredScenarioSpecs, response.Scenarios)

	a.persistAnalysisResults(response, declaredScenarioDeps, scenarioName, scenarioPath, serviceConfig)
	a.attachDeploymentReport(response, scenarioName, scenarioPath, serviceConfig)
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

func newAnalysisResponse(scenarioName string, serviceConfig *types.ServiceConfig) *types.DependencyAnalysisResponse {
	declaredSpecs := map[string]types.ScenarioDependencySpec{}
	if serviceConfig != nil {
		declaredSpecs = normalizeScenarioSpecs(serviceConfig.Dependencies.Scenarios)
	}

	response := &types.DependencyAnalysisResponse{
		Scenario:              scenarioName,
		Resources:             []types.ScenarioDependency{},
		DetectedResources:     []types.ScenarioDependency{},
		Scenarios:             []types.ScenarioDependency{},
		DeclaredScenarioSpecs: declaredSpecs,
		SharedWorkflows:       []types.ScenarioDependency{},
		TransitiveDepth:       0,
		ResourceDiff:          types.DependencyDiff{},
		ScenarioDiff:          types.DependencyDiff{},
	}

	if serviceConfig != nil {
		response.Resources = extractDeclaredResources(scenarioName, serviceConfig)
	}

	return response
}

func (a *Analyzer) collectDetections(response *types.DependencyAnalysisResponse, scenarioPath, scenarioName string, serviceConfig *types.ServiceConfig) {
	if a == nil || a.detector == nil {
		return
	}

	if detectedResources, err := a.detector.ScanResources(scenarioPath, scenarioName, serviceConfig); err != nil {
		log.Printf("Warning: failed to scan for resource usage: %v", err)
	} else {
		response.DetectedResources = detectedResources
	}

	if scenarioDeps, err := a.detector.ScanScenarioDependencies(scenarioPath, scenarioName); err != nil {
		log.Printf("Warning: failed to scan for scenario dependencies: %v", err)
	} else {
		response.Scenarios = append(response.Scenarios, scenarioDeps...)
	}

	if workflowDeps, err := a.detector.ScanSharedWorkflows(scenarioPath, scenarioName); err != nil {
		log.Printf("Warning: failed to scan for shared workflows: %v", err)
	} else {
		response.SharedWorkflows = append(response.SharedWorkflows, workflowDeps...)
	}
}

func (a *Analyzer) persistAnalysisResults(
	response *types.DependencyAnalysisResponse,
	declaredScenarioDeps []types.ScenarioDependency,
	scenarioName string,
	scenarioPath string,
	serviceConfig *types.ServiceConfig,
) {
	if a == nil || a.store == nil {
		log.Printf("Warning: store not initialized; dependency metadata not persisted")
		return
	}

	if err := a.store.StoreDependencies(response, declaredScenarioDeps); err != nil {
		log.Printf("Warning: failed to store dependencies in database: %v", err)
	}

	if err := a.store.UpdateScenarioMetadata(scenarioName, serviceConfig, scenarioPath); err != nil {
		log.Printf("Warning: failed to update scenario metadata: %v", err)
	}
}

func (a *Analyzer) attachDeploymentReport(
	response *types.DependencyAnalysisResponse,
	scenarioName string,
	scenarioPath string,
	serviceConfig *types.ServiceConfig,
) {
	if deploymentReport := deployment.BuildReport(scenarioName, scenarioPath, a.cfg.ScenariosDir, serviceConfig); deploymentReport != nil {
		response.DeploymentReport = deploymentReport
		if err := deployment.PersistReport(scenarioPath, deploymentReport); err != nil {
			log.Printf("Warning: failed to persist deployment report: %v", err)
		}
	}
}

// extractDeclaredResources extracts declared resources using default seams.
// For testable code, use extractDeclaredResourcesWithSeams.
func extractDeclaredResources(scenarioName string, cfg *types.ServiceConfig) []types.ScenarioDependency {
	return extractDeclaredResourcesWithSeams(scenarioName, cfg, seams.Default)
}

// extractDeclaredResourcesWithSeams extracts declared resources using injected seams.
// This enables deterministic testing by controlling time and ID generation.
func extractDeclaredResourcesWithSeams(scenarioName string, cfg *types.ServiceConfig, deps *seams.Dependencies) []types.ScenarioDependency {
	resources := appconfig.ResolvedResourceMap(cfg)
	declared := make([]types.ScenarioDependency, 0, len(resources))
	now := deps.Clock.Now()
	for name, resource := range resources {
		declared = append(declared, types.ScenarioDependency{
			ID:             deps.IDs.NewID(),
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
			DiscoveredAt: now,
			LastVerified: now,
		})
	}
	sort.Slice(declared, func(i, j int) bool { return declared[i].DependencyName < declared[j].DependencyName })
	return declared
}

// buildDependencyDiff computes missing/extra drifts between declared and detected dependencies.
// declaredNames provides the set of names considered declared.
// extractExtraDetails is called for each declared name not found in detected to build its Details.
func buildDependencyDiff(
	declaredNames map[string]struct{},
	detected []types.ScenarioDependency,
	extractExtraDetails func(name string) map[string]interface{},
) types.DependencyDiff {
	detectedSet := make(map[string]types.ScenarioDependency, len(detected))
	for _, dep := range detected {
		detectedSet[dep.DependencyName] = dep
	}

	var missing []types.DependencyDrift
	for name, dep := range detectedSet {
		if _, ok := declaredNames[name]; !ok {
			missing = append(missing, types.DependencyDrift{
				Name:    name,
				Details: dep.Configuration,
			})
		}
	}

	var extra []types.DependencyDrift
	for name := range declaredNames {
		if _, ok := detectedSet[name]; !ok {
			extra = append(extra, types.DependencyDrift{
				Name:    name,
				Details: extractExtraDetails(name),
			})
		}
	}

	sort.Slice(missing, func(i, j int) bool { return missing[i].Name < missing[j].Name })
	sort.Slice(extra, func(i, j int) bool { return extra[i].Name < extra[j].Name })

	return types.DependencyDiff{Missing: missing, Extra: extra}
}

func buildResourceDiff(declared map[string]types.Resource, detected []types.ScenarioDependency) types.DependencyDiff {
	names := make(map[string]struct{}, len(declared))
	for name := range declared {
		names[name] = struct{}{}
	}
	return buildDependencyDiff(names, detected, func(name string) map[string]interface{} {
		cfg := declared[name]
		return map[string]interface{}{
			"type":     cfg.Type,
			"required": cfg.Required,
		}
	})
}

func buildScenarioDiff(declared map[string]types.ScenarioDependencySpec, detected []types.ScenarioDependency) types.DependencyDiff {
	names := make(map[string]struct{}, len(declared))
	for name := range declared {
		names[name] = struct{}{}
	}
	return buildDependencyDiff(names, detected, func(name string) map[string]interface{} {
		spec := declared[name]
		return map[string]interface{}{
			"required": spec.Required,
			"version":  spec.Version,
		}
	})
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

// convertDeclaredScenariosToDependencies converts declared scenario specs using default seams.
// For testable code, use convertDeclaredScenariosToDependenciesWithSeams.
func convertDeclaredScenariosToDependencies(scenarioName string, specs map[string]types.ScenarioDependencySpec) []types.ScenarioDependency {
	return convertDeclaredScenariosToDependenciesWithSeams(scenarioName, specs, seams.Default)
}

// convertDeclaredScenariosToDependenciesWithSeams converts declared scenario specs using injected seams.
// This enables deterministic testing by controlling time and ID generation.
func convertDeclaredScenariosToDependenciesWithSeams(scenarioName string, specs map[string]types.ScenarioDependencySpec, deps *seams.Dependencies) []types.ScenarioDependency {
	if len(specs) == 0 {
		return nil
	}
	result := make([]types.ScenarioDependency, 0, len(specs))
	now := deps.Clock.Now()
	for name, spec := range specs {
		result = append(result, types.ScenarioDependency{
			ID:             deps.IDs.NewID(),
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
			DiscoveredAt: now,
			LastVerified: now,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].DependencyName < result[j].DependencyName })
	return result
}
