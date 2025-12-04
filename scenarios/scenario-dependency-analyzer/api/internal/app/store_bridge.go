package app

// store_bridge.go provides a functional API for store operations.
//
// These functions retrieve the current store from the active runtime (or fallback)
// and delegate to the store's methods. This decouples callers from needing to
// manage the runtime/analyzer lifecycle directly.
//
// Use these functions when:
// - You need store access from code that doesn't hold a Runtime reference
// - You want simple function calls rather than chained method access
//
// For new code, prefer receiving the store as a dependency when possible.

import (
	"errors"
	"os"
	"path/filepath"

	appconfig "scenario-dependency-analyzer/internal/config"
	"scenario-dependency-analyzer/internal/store"
	types "scenario-dependency-analyzer/internal/types"
)

// currentStore returns the store from the active runtime, or nil if unavailable.
func currentStore() *store.Store {
	if analyzer := analyzerInstance(); analyzer != nil {
		return analyzer.Store()
	}
	return nil
}

func storeDependencies(analysis *types.DependencyAnalysisResponse, extras []types.ScenarioDependency) error {
	if st := currentStore(); st != nil {
		return st.StoreDependencies(analysis, extras)
	}
	return nil
}

func updateScenarioMetadata(name string, cfg *types.ServiceConfig, scenarioPath string) error {
	if st := currentStore(); st != nil {
		return st.UpdateScenarioMetadata(name, cfg, scenarioPath)
	}
	return nil
}

func loadScenarioMetadataMap() (map[string]types.ScenarioSummary, error) {
	if st := currentStore(); st != nil {
		return st.LoadScenarioMetadataMap()
	}
	return map[string]types.ScenarioSummary{}, nil
}

func loadStoredDependencies(scenario string) (map[string][]types.ScenarioDependency, error) {
	if st := currentStore(); st != nil {
		return st.LoadStoredDependencies(scenario)
	}
	return map[string][]types.ScenarioDependency{
		"resources":        {},
		"scenarios":        {},
		"shared_workflows": {},
	}, nil
}

func loadOptimizationRecommendations(scenario string) ([]types.OptimizationRecommendation, error) {
	if st := currentStore(); st != nil {
		return st.LoadOptimizationRecommendations(scenario)
	}
	return nil, nil
}

func loadAllDependencies() ([]types.ScenarioDependency, error) {
	if st := currentStore(); st != nil {
		return st.LoadAllDependencies()
	}
	return nil, errors.New("store not initialized")
}

func listScenarioNames() ([]string, error) {
	cfg := appconfig.Load()
	entries, err := os.ReadDir(cfg.ScenariosDir)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		serviceConfigPath := filepath.Join(cfg.ScenariosDir, entry.Name(), ".vrooli", "service.json")
		if _, err := os.Stat(serviceConfigPath); errors.Is(err, os.ErrNotExist) {
			continue
		}
		names = append(names, entry.Name())
	}
	return names, nil
}

func collectAnalysisMetrics() (map[string]interface{}, error) {
	if st := currentStore(); st != nil {
		return st.CollectAnalysisMetrics()
	}

	// Return default metrics when store is unavailable
	return map[string]interface{}{
		"scenarios_found":     0,
		"resources_available": 0,
		"database_status":     "unavailable",
		"last_analysis":       nil,
	}, errors.New("store not initialized")
}
