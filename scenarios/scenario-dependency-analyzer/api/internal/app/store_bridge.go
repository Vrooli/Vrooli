package app

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"time"

	appconfig "scenario-dependency-analyzer/internal/config"
	"scenario-dependency-analyzer/internal/store"
	types "scenario-dependency-analyzer/internal/types"
)

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

func listScenarioNames() ([]string, error) {
	cfg := appconfig.Load()
	if st := currentStore(); st != nil {
		return st.ListScenarioNames(cfg.ScenariosDir)
	}

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

	metrics := map[string]interface{}{
		"scenarios_found":     0,
		"resources_available": 0,
		"database_status":     "unknown",
		"last_analysis":       nil,
	}

	if db == nil {
		return metrics, errors.New("database connection not initialized")
	}

	if err := db.Ping(); err != nil {
		metrics["database_status"] = "unreachable"
		return metrics, err
	}

	metrics["database_status"] = "connected"

	var scenarioCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM scenario_metadata").Scan(&scenarioCount); err != nil {
		metrics["database_status"] = "error"
		return metrics, err
	}
	metrics["scenarios_found"] = scenarioCount

	var resourceCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM scenario_dependencies WHERE dependency_type = 'resource'").Scan(&resourceCount); err != nil {
		metrics["database_status"] = "error"
		return metrics, err
	}
	metrics["resources_available"] = resourceCount

	var lastAnalysis sql.NullTime
	if err := db.QueryRow("SELECT MAX(last_scanned) FROM scenario_metadata").Scan(&lastAnalysis); err == nil && lastAnalysis.Valid {
		metrics["last_analysis"] = lastAnalysis.Time.UTC().Format(time.RFC3339)
	}

	return metrics, nil
}
