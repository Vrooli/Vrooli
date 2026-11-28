package app

import (
	"fmt"
	"sort"
	"time"

	appconfig "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/config"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/deployment"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/store"
	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

type scenarioService struct {
	workspace *scenarioWorkspace
	store     *store.Store
}

func (s *scenarioService) ListScenarios() ([]types.ScenarioSummary, error) {
	var (
		metadata map[string]types.ScenarioSummary
		err      error
	)
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
