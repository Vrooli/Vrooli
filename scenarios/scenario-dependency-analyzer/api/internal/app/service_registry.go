
package app

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"scenario-dependency-analyzer/internal/app/services"
	appconfig "scenario-dependency-analyzer/internal/config"
	"scenario-dependency-analyzer/internal/store"
	types "scenario-dependency-analyzer/internal/types"
)

var errScenarioNotFound = errors.New("scenario not found")

func newServices(analyzer *Analyzer) services.Registry {
	analysis := &analysisService{}
	scan := &scanService{analysis: analysis}
	optimization := &optimizationService{analysis: analysis}
	scenarios := &scenarioService{cfg: analyzer.cfg, store: analyzer.store}
	deployment := &deploymentService{cfg: analyzer.cfg}
	proposal := &proposalService{}
	graph := &graphService{analyzer: analyzer}

	return services.Registry{
		Analysis:     analysis,
		Scan:         scan,
		Graph:        graph,
		Optimization: optimization,
		Scenarios:    scenarios,
		Deployment:   deployment,
		Proposal:     proposal,
	}
}

type analysisService struct{}

func (analysisService) AnalyzeScenario(name string) (*types.DependencyAnalysisResponse, error) {
	return analyzeScenario(name)
}

func (analysisService) AnalyzeAllScenarios() (map[string]*types.DependencyAnalysisResponse, error) {
	return analyzeAllScenarios()
}

type scanService struct {
	analysis services.AnalysisService
}

func (s *scanService) ScanScenario(name string, req types.ScanRequest) (*services.ScanResult, error) {
	analysis, err := s.analysis.AnalyzeScenario(name)
	if err != nil {
		return nil, err
	}

	applyResources := req.ApplyResources || req.Apply
	applyScenarios := req.ApplyScenarios || req.Apply
	var applySummary map[string]interface{}
	applied := false

	if applyResources || applyScenarios {
		applySummary, err = applyDetectedDiffs(name, analysis, applyResources, applyScenarios)
		if err != nil {
			return nil, err
		}
		if changed, ok := applySummary["changed"].(bool); ok && changed {
			applied = true
			analysis, err = s.analysis.AnalyzeScenario(name)
			if err != nil {
				return nil, err
			}
		}
	}

	return &services.ScanResult{
		Analysis:     analysis,
		Applied:      applied,
		ApplySummary: applySummary,
	}, nil
}

type graphService struct {
	analyzer *Analyzer
}

func (g *graphService) GenerateGraph(graphType string) (*types.DependencyGraph, error) {
	if g == nil || g.analyzer == nil {
		return nil, fmt.Errorf("analyzer not initialized")
	}
	return g.analyzer.generateDependencyGraph(graphType)
}

type optimizationService struct {
	analysis services.AnalysisService
}

func (s *optimizationService) RunOptimization(req types.OptimizationRequest) (map[string]*types.OptimizationResult, error) {
	scenario := strings.TrimSpace(req.Scenario)
	if scenario == "" {
		scenario = "all"
	}

	var targets []string
	if scenario == "all" {
		names, err := listScenarioNames()
		if err != nil {
			return nil, err
		}
		targets = names
	} else {
		if !isKnownScenario(scenario) {
			return nil, fmt.Errorf("%w: %s", errScenarioNotFound, scenario)
		}
		targets = []string{scenario}
	}

	results := make(map[string]*types.OptimizationResult, len(targets))
	for _, target := range targets {
		result, err := runScenarioOptimization(target, req)
		if err != nil {
			results[target] = &types.OptimizationResult{
				Scenario:          target,
				Recommendations:   nil,
				Summary:           types.OptimizationSummary{},
				Applied:           false,
				AnalysisTimestamp: time.Now().UTC(),
				Error:             err.Error(),
			}
			continue
		}
		results[target] = result
	}

	return results, nil
}

type scenarioService struct {
	cfg   appconfig.Config
	store *store.Store
}

func (s *scenarioService) ListScenarios() ([]types.ScenarioSummary, error) {
	metadata := map[string]types.ScenarioSummary{}
	var err error
	if s.store != nil {
		metadata, err = s.store.LoadScenarioMetadataMap()
	} else {
		metadata, err = loadScenarioMetadataMap()
	}
	if err != nil {
		return nil, err
	}

	scenariosDir := s.cfg.ScenariosDir
	if scenariosDir == "" {
		scenariosDir = appconfig.Load().ScenariosDir
	}

	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		return nil, err
	}

	summaries := []types.ScenarioSummary{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		summary, ok := metadata[name]
		if !ok {
			scenarioPath := filepath.Join(scenariosDir, name)
			cfg, cfgErr := loadServiceConfigFromFile(scenarioPath)
			if cfgErr != nil {
				continue
			}
			summary = types.ScenarioSummary{
				Name:        name,
				DisplayName: cfg.Service.DisplayName,
				Description: cfg.Service.Description,
				Tags:        cfg.Service.Tags,
			}
		}
		summaries = append(summaries, summary)
	}

	sort.Slice(summaries, func(i, j int) bool { return summaries[i].Name < summaries[j].Name })
	return summaries, nil
}

func (s *scenarioService) GetScenarioDetail(name string) (*types.ScenarioDetailResponse, error) {
	envCfg := appconfig.Load()
	scenarioPath := filepath.Join(envCfg.ScenariosDir, name)
	cfg, err := loadServiceConfigFromFile(scenarioPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errScenarioNotFound, err)
	}

	stored, err := loadStoredDependencies(name)
	if err != nil {
		return nil, err
	}
	optRecs, err := loadOptimizationRecommendations(name)
	if err != nil {
		return nil, err
	}

	declaredResources := resolvedResourceMap(cfg)
	declaredScenarios := cfg.Dependencies.Scenarios
	if declaredScenarios == nil {
		declaredScenarios = map[string]types.ScenarioDependencySpec{}
	}

	resourceDiff := buildResourceDiff(declaredResources, filterDetectedDependencies(stored["resources"]))
	scenarioDiff := buildScenarioDiff(declaredScenarios, filterDetectedDependencies(stored["scenarios"]))

	var lastScanned *time.Time
	if s.store != nil {
		if metadata, err := s.store.LoadScenarioMetadataMap(); err == nil {
			if summary, ok := metadata[name]; ok {
				lastScanned = summary.LastScanned
			}
		}
	} else if metadata, err := loadScenarioMetadataMap(); err == nil {
		if summary, ok := metadata[name]; ok {
			lastScanned = summary.LastScanned
		}
	}

	detail := types.ScenarioDetailResponse{
		Scenario:                    name,
		DisplayName:                 cfg.Service.DisplayName,
		Description:                 cfg.Service.Description,
		LastScanned:                 lastScanned,
		DeclaredResources:           declaredResources,
		DeclaredScenarios:           declaredScenarios,
		StoredDependencies:          stored,
		ResourceDiff:                resourceDiff,
		ScenarioDiff:                scenarioDiff,
		OptimizationRecommendations: optRecs,
	}

	if report := buildDeploymentReport(name, scenarioPath, envCfg.ScenariosDir, cfg); report != nil {
		detail.DeploymentReport = report
	}

	return &detail, nil
}

type deploymentService struct {
	cfg appconfig.Config
}

func (d *deploymentService) GetDeploymentReport(name string) (*types.DeploymentAnalysisReport, error) {
	envCfg := appconfig.Load()
	scenarioPath := filepath.Join(envCfg.ScenariosDir, name)
	cfg, err := loadServiceConfigFromFile(scenarioPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errScenarioNotFound, err)
	}

	report, err := loadPersistedDeploymentReport(scenarioPath)
	if err != nil {
		report = buildDeploymentReport(name, scenarioPath, envCfg.ScenariosDir, cfg)
		if report != nil {
			if persistErr := persistDeploymentReport(scenarioPath, report); persistErr != nil {
				log.Printf("Warning: failed to persist deployment report for %s: %v", name, persistErr)
			}
		}
	}

	if report == nil {
		return nil, fmt.Errorf("failed to build deployment report for %s", name)
	}

	return report, nil
}

type proposalService struct{}

func (proposalService) AnalyzeProposedScenario(req types.ProposedScenarioRequest) (map[string]interface{}, error) {
	return analyzeProposedScenario(req)
}
