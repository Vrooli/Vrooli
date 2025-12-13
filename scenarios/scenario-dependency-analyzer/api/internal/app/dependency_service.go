package app

import (
	"fmt"

	"scenario-dependency-analyzer/internal/store"
	types "scenario-dependency-analyzer/internal/types"
)

// scenarioDetector captures the detector capabilities needed for dependency analysis.
type scenarioDetector interface {
	KnownScenario(string) bool
	ScenarioCatalog() map[string]struct{}
	RefreshCatalogs()
}

// dependencyService centralizes dependency catalog access and impact analysis.
type dependencyService struct {
	store    *store.Store
	detector scenarioDetector
}

func newDependencyService(st *store.Store, detector scenarioDetector) *dependencyService {
	return &dependencyService{
		store:    st,
		detector: detector,
	}
}

// defaultDependencyService provides a best-effort service using active runtime dependencies when available.
func defaultDependencyService() *dependencyService {
	if analyzer := analyzerInstance(); analyzer != nil {
		return newDependencyService(analyzer.Store(), analyzer.Detector())
	}
	return newDependencyService(currentStore(), detectorInstance())
}

func (s *dependencyService) StoredDependencies(name string) (map[string][]types.ScenarioDependency, error) {
	stored := map[string][]types.ScenarioDependency{
		"resources":        {},
		"scenarios":        {},
		"shared_workflows": {},
	}

	if s == nil || s.store == nil {
		return stored, nil
	}

	loaded, err := s.store.LoadStoredDependencies(name)
	if err != nil {
		return nil, err
	}
	return loaded, nil
}

func (s *dependencyService) DependencyImpact(name string) (*types.DependencyImpactReport, error) {
	var dependencies []types.ScenarioDependency
	if s != nil && s.store != nil {
		loaded, err := s.store.LoadAllDependencies()
		if err != nil {
			return nil, err
		}
		dependencies = loaded
	}
	return analyzeDependencyImpact(name, dependencies, s.catalogSnapshot())
}

func (s *dependencyService) AnalysisMetrics() (map[string]interface{}, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("dependency store unavailable")
	}
	return s.store.CollectAnalysisMetrics()
}

func (s *dependencyService) catalogSnapshot() map[string]struct{} {
	if s == nil || s.detector == nil {
		return nil
	}
	return s.detector.ScenarioCatalog()
}

func (s *dependencyService) UpdateScenarioMetadata(name string, cfg *types.ServiceConfig, scenarioPath string) error {
	if s == nil || s.store == nil {
		return nil
	}
	return s.store.UpdateScenarioMetadata(name, cfg, scenarioPath)
}

func (s *dependencyService) RefreshCatalogs() {
	if s == nil || s.detector == nil {
		return
	}
	s.detector.RefreshCatalogs()
}

func (s *dependencyService) CleanupInvalidDependencies() {
	if s == nil || s.store == nil {
		return
	}
	_ = s.store.CleanupInvalidScenarioDependencies(s.catalogSnapshot())
}
