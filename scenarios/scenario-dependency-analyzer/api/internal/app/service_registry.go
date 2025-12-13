package app

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"scenario-dependency-analyzer/internal/app/services"
	appconfig "scenario-dependency-analyzer/internal/config"
	"scenario-dependency-analyzer/internal/deployment"
	"scenario-dependency-analyzer/internal/store"
	types "scenario-dependency-analyzer/internal/types"
)

var errScenarioNotFound = errors.New("scenario not found")

func newServices(analyzer *Analyzer, workspace *scenarioWorkspace) services.Registry {
	if workspace == nil {
		workspace = newScenarioWorkspace(analyzer.cfg)
	}
	analysis := &analysisService{analyzer: analyzer}
	dependencies := newDependencyService(analyzer.store, analyzer.detector)
	scan := &scanService{analysis: analysis, dependencies: dependencies}
	optimization := &optimizationService{
		analysis:     analysis,
		workspace:    workspace,
		detector:     analyzer.detector,
		dependencies: dependencies,
		store:        analyzer.store,
	}
	scenarios := &scenarioService{workspace: workspace, store: analyzer.store}
	deployment := &deploymentService{workspace: workspace}
	proposal := &proposalService{}
	graph := &graphService{analyzer: analyzer}

	return services.Registry{
		Analysis:     analysis,
		Scan:         scan,
		Graph:        graph,
		Optimization: optimization,
		Scenarios:    scenarios,
		Dependencies: dependencies,
		Deployment:   deployment,
		Proposal:     proposal,
	}
}

type analysisService struct {
	analyzer *Analyzer
}

func (s *analysisService) AnalyzeScenario(name string) (*types.DependencyAnalysisResponse, error) {
	if s == nil || s.analyzer == nil {
		return nil, fmt.Errorf("analysis service not initialized")
	}
	return s.analyzer.AnalyzeScenario(name)
}

func (s *analysisService) AnalyzeAllScenarios() (map[string]*types.DependencyAnalysisResponse, error) {
	if s == nil || s.analyzer == nil {
		return nil, fmt.Errorf("analysis service not initialized")
	}
	return s.analyzer.AnalyzeAllScenarios()
}

type scanService struct {
	analysis     services.AnalysisService
	dependencies *dependencyService
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
		applySummary, err = applyDetectedDiffs(name, analysis, applyResources, applyScenarios, s.dependencies)
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
	return g.analyzer.GenerateGraph(graphType)
}

type optimizationService struct {
	analysis     services.AnalysisService
	workspace    *scenarioWorkspace
	detector     scenarioDetector
	dependencies *dependencyService
	store        *store.Store
}

func (s *optimizationService) RunOptimization(req types.OptimizationRequest) (map[string]*types.OptimizationResult, error) {
	if s.workspace == nil {
		return nil, fmt.Errorf("optimization service not initialized")
	}
	scenario := strings.TrimSpace(req.Scenario)
	if scenario == "" {
		scenario = "all"
	}

	var targets []string
	if scenario == "all" {
		names, err := s.workspace.listScenarioNames()
		if err != nil {
			return nil, err
		}
		targets = names
	} else {
		if !s.workspace.hasServiceConfig(scenario) {
			return nil, fmt.Errorf("%w: %s", errScenarioNotFound, scenario)
		}
		if s.detector != nil && !s.detector.KnownScenario(scenario) {
			return nil, fmt.Errorf("%w: %s", errScenarioNotFound, scenario)
		}
		targets = []string{scenario}
	}

	results := make(map[string]*types.OptimizationResult, len(targets))
	for _, target := range targets {
		result, err := s.runScenarioOptimization(target, req)
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
	workspace *scenarioWorkspace
	store     *store.Store
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

	summaries := []types.ScenarioSummary{}
	names, err := s.workspace.listScenarioNames()
	if err != nil {
		return nil, err
	}

	for _, name := range names {
		summary, ok := metadata[name]
		if !ok {
			cfg, cfgErr := s.workspace.loadConfig(name)
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
	scenarioPath := s.workspace.pathFor(name)
	cfg, err := s.workspace.loadConfig(name)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errScenarioNotFound, err)
	}

	stored := map[string][]types.ScenarioDependency{
		"resources":        {},
		"scenarios":        {},
		"shared_workflows": {},
	}
	var optRecs []types.OptimizationRecommendation
	if s.store != nil {
		stored, err = s.store.LoadStoredDependencies(name)
		if err != nil {
			return nil, err
		}
		optRecs, err = s.store.LoadOptimizationRecommendations(name)
		if err != nil {
			return nil, err
		}
	}

	declaredResources := appconfig.ResolvedResourceMap(cfg)
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

	if report := deployment.BuildReport(name, scenarioPath, s.workspace.root, cfg); report != nil {
		detail.DeploymentReport = report
	}

	return &detail, nil
}

type deploymentService struct {
	workspace *scenarioWorkspace
}

func (d *deploymentService) GetDeploymentReport(name string) (*types.DeploymentAnalysisReport, error) {
	scenarioPath := d.workspace.pathFor(name)
	cfg, err := d.workspace.loadConfig(name)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errScenarioNotFound, err)
	}

	report, err := deployment.LoadReport(scenarioPath)
	if err != nil {
		report = deployment.BuildReport(name, scenarioPath, d.workspace.root, cfg)
		if report != nil {
			if persistErr := deployment.PersistReport(scenarioPath, report); persistErr != nil {
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
