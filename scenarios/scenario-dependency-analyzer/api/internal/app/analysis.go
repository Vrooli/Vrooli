package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/deployment"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/seams"

	appconfig "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/config"

	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

func (a *Analyzer) AnalyzeScenario(scenarioName string) (*types.DependencyAnalysisResponse, error) {
	return a.AnalyzeScenarioContext(context.Background(), scenarioName)
}

func (a *Analyzer) AnalyzeScenarioContext(ctx context.Context, scenarioName string) (*types.DependencyAnalysisResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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

	if err := a.collectDetections(ctx, response, scenarioPath, scenarioName, serviceConfig); err != nil {
		return nil, err
	}

	declaredScenarioDeps := convertDeclaredScenariosToDependencies(scenarioName, response.DeclaredScenarioSpecs)
	response.ResourceDiff = buildResourceDiff(appconfig.ResolvedResourceMap(serviceConfig), response.DetectedResources)
	response.ScenarioDiff = buildScenarioDiff(response.DeclaredScenarioSpecs, response.Scenarios)

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := a.persistAnalysisResults(response, declaredScenarioDeps, scenarioName, scenarioPath, serviceConfig); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := a.attachDeploymentReport(ctx, response, scenarioName, scenarioPath, serviceConfig); err != nil {
		return nil, err
	}
	return response, nil
}

// ScenarioNames captures the fleet without retaining full analysis responses.
func (a *Analyzer) ScenarioNames(ctx context.Context) ([]string, error) {
	if a == nil {
		return nil, fmt.Errorf("analyzer not initialized")
	}
	entries, err := os.ReadDir(a.cfg.ScenariosDir)
	if err != nil {
		return nil, fmt.Errorf("read scenarios: %w", err)
	}
	var names []string
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !entry.IsDir() {
			continue
		}
		_, err := os.Stat(filepath.Join(a.cfg.ScenariosDir, entry.Name(), ".vrooli", "service.json"))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		names = append(names, entry.Name())
	}
	return names, nil
}

func newAnalysisResponse(scenarioName string, serviceConfig *types.Manifest) *types.DependencyAnalysisResponse {
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

func (a *Analyzer) collectDetections(ctx context.Context, response *types.DependencyAnalysisResponse, scenarioPath, scenarioName string, serviceConfig *types.Manifest) error {
	resources, err := a.detector.ScanResourcesContext(ctx, scenarioPath, scenarioName, serviceConfig)
	if err != nil {
		return fmt.Errorf("scan resource usage: %w", err)
	}
	scenarios, err := a.detector.ScanScenarioDependenciesContext(ctx, scenarioPath, scenarioName)
	if err != nil {
		return fmt.Errorf("scan scenario dependencies: %w", err)
	}
	response.DetectedResources, response.Scenarios = resources, scenarios
	return nil
}

func (a *Analyzer) persistAnalysisResults(
	response *types.DependencyAnalysisResponse,
	declaredScenarioDeps []types.ScenarioDependency,
	scenarioName string,
	scenarioPath string,
	serviceConfig *types.Manifest,
) error {
	if a == nil || a.store == nil {
		return fmt.Errorf("dependency store not initialized")
	}

	if err := a.store.StoreDependencies(response, declaredScenarioDeps); err != nil {
		return fmt.Errorf("store dependencies: %w", err)
	}

	if err := a.store.UpdateScenarioMetadata(scenarioName, serviceConfig, scenarioPath); err != nil {
		return fmt.Errorf("store metadata: %w", err)
	}
	return nil
}

func (a *Analyzer) attachDeploymentReport(
	ctx context.Context,
	response *types.DependencyAnalysisResponse,
	scenarioName string,
	scenarioPath string,
	serviceConfig *types.Manifest,
) error {
	deploymentReport, err := deployment.BuildReportContext(ctx, scenarioName, scenarioPath, a.cfg.ScenariosDir, serviceConfig)
	if err != nil {
		return err
	}
	if deploymentReport != nil {
		response.DeploymentReport = deploymentReport
		if err := deployment.PersistReport(scenarioPath, deploymentReport); err != nil {
			return fmt.Errorf("persist deployment report: %w", err)
		}
	}
	return nil
}

// extractDeclaredResources extracts declared resources using default seams.
// For testable code, use extractDeclaredResourcesWithSeams.
func extractDeclaredResources(scenarioName string, cfg *types.Manifest) []types.ScenarioDependency {
	return extractDeclaredResourcesWithSeams(scenarioName, cfg, seams.Default)
}

// extractDeclaredResourcesWithSeams extracts declared resources using injected seams.
// This enables deterministic testing by controlling time and ID generation.
func extractDeclaredResourcesWithSeams(scenarioName string, cfg *types.Manifest, deps *seams.Dependencies) []types.ScenarioDependency {
	resources := appconfig.ResolvedResourceMap(cfg)
	declared := make([]types.ScenarioDependency, 0, len(resources))
	now := deps.Clock.Now()
	for name, resource := range resources {
		configuration := map[string]interface{}{
			"type":     resource.Type,
			"enabled":  resource.Enabled,
			"purpose":  resource.Purpose,
			"declared": true,
		}
		var resourceConfig struct {
			Models []string `json:"models"`
		}
		if len(resource.Config) > 0 && json.Unmarshal(resource.Config, &resourceConfig) == nil && len(resourceConfig.Models) > 0 {
			configuration["models"] = resourceConfig.Models
		}
		declared = append(declared, types.ScenarioDependency{
			ID:             deps.IDs.NewID(),
			ScenarioName:   scenarioName,
			DependencyType: "resource",
			DependencyName: name,
			Required:       resource.Required,
			Purpose:        resource.Purpose,
			AccessMethod:   "declared",
			Configuration:  configuration,
			DiscoveredAt:   now,
			LastVerified:   now,
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
			"required":      spec.Required,
			"version_range": spec.VersionRange,
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
