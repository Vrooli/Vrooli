package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	pq "github.com/lib/pq"

	types "scenario-dependency-analyzer/internal/types"
)

// Store centralizes all persistence logic for the dependency analyzer.
type Store struct {
	db *sql.DB
}

// New builds a store bound to the provided database handle.
func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// DB exposes the underlying sql.DB for low-level scenarios (e.g., ping).
func (s *Store) DB() *sql.DB { return s.db }

// CleanupInvalidScenarioDependencies removes scenario dependency rows for
// names that are no longer known to the catalog.
func (s *Store) CleanupInvalidScenarioDependencies(known map[string]struct{}) error {
	if s.db == nil || len(known) == 0 {
		return nil
	}

	rows, err := s.db.Query(`
        SELECT DISTINCT dependency_name
        FROM scenario_dependencies
        WHERE dependency_type = 'scenario'`)
	if err != nil {
		return fmt.Errorf("query scenario dependencies: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var dep string
		if scanErr := rows.Scan(&dep); scanErr != nil {
			continue
		}
		if _, ok := known[normalizeName(dep)]; ok {
			continue
		}
		if _, err := s.db.Exec(`
            DELETE FROM scenario_dependencies
            WHERE dependency_type = 'scenario' AND dependency_name = $1`, dep); err != nil {
			return fmt.Errorf("delete orphaned dependency %s: %w", dep, err)
		}
	}
	return nil
}

// StoreDependencies persists the declared and detected dependencies for a scenario.
func (s *Store) StoreDependencies(analysis *types.DependencyAnalysisResponse, extras []types.ScenarioDependency) error {
	if s.db == nil {
		return errors.New("store not initialized")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM scenario_dependencies WHERE scenario_name = $1", analysis.Scenario); err != nil {
		return err
	}

	allDeps := make([]types.ScenarioDependency, 0, len(analysis.Resources)+len(analysis.DetectedResources)+len(analysis.Scenarios)+len(analysis.SharedWorkflows)+len(extras))
	allDeps = append(allDeps, analysis.Resources...)
	allDeps = append(allDeps, analysis.DetectedResources...)
	allDeps = append(allDeps, analysis.Scenarios...)
	allDeps = append(allDeps, analysis.SharedWorkflows...)
	allDeps = append(allDeps, extras...)

	for _, dep := range allDeps {
		configJSON, _ := json.Marshal(dep.Configuration)
		if _, err := tx.Exec(`
            INSERT INTO scenario_dependencies
            (scenario_name, dependency_type, dependency_name, required, purpose, access_method, configuration, discovered_at, last_verified)
            VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			dep.ScenarioName, dep.DependencyType, dep.DependencyName, dep.Required,
			dep.Purpose, dep.AccessMethod, configJSON, dep.DiscoveredAt, dep.LastVerified); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// LoadStoredDependencies returns all persisted dependencies for a scenario, grouped by type.
func (s *Store) LoadStoredDependencies(scenario string) (map[string][]types.ScenarioDependency, error) {
	result := map[string][]types.ScenarioDependency{
		"resources":        {},
		"scenarios":        {},
		"shared_workflows": {},
	}

	if s.db == nil {
		return result, nil
	}

	rows, err := s.db.Query(`
        SELECT scenario_name, dependency_type, dependency_name, required, purpose, access_method, configuration, discovered_at, last_verified
        FROM scenario_dependencies
        WHERE scenario_name = $1
        ORDER BY dependency_type, dependency_name`, scenario)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var dep types.ScenarioDependency
		var configJSON []byte
		if err := rows.Scan(&dep.ScenarioName, &dep.DependencyType, &dep.DependencyName, &dep.Required, &dep.Purpose, &dep.AccessMethod, &configJSON, &dep.DiscoveredAt, &dep.LastVerified); err != nil {
			continue
		}
		if len(configJSON) > 0 {
			_ = json.Unmarshal(configJSON, &dep.Configuration)
		}
		result[dep.DependencyType] = append(result[dep.DependencyType], dep)
	}

	return result, nil
}

// LoadOptimizationRecommendations fetches stored optimization recommendations.
func (s *Store) LoadOptimizationRecommendations(scenario string) ([]types.OptimizationRecommendation, error) {
	if s.db == nil {
		return nil, nil
	}

	rows, err := s.db.Query(`
        SELECT id, scenario_name, recommendation_type, title, description, current_state, recommended_state, estimated_impact, confidence_score, priority, status, created_at
        FROM optimization_recommendations
        WHERE scenario_name = $1
        ORDER BY created_at DESC`, scenario)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recs []types.OptimizationRecommendation
	for rows.Next() {
		var rec types.OptimizationRecommendation
		var currentJSON, recommendedJSON, estimatedJSON []byte
		if err := rows.Scan(&rec.ID, &rec.ScenarioName, &rec.RecommendationType, &rec.Title, &rec.Description, &currentJSON, &recommendedJSON, &estimatedJSON, &rec.ConfidenceScore, &rec.Priority, &rec.Status, &rec.CreatedAt); err != nil {
			continue
		}
		_ = json.Unmarshal(currentJSON, &rec.CurrentState)
		_ = json.Unmarshal(recommendedJSON, &rec.RecommendedState)
		_ = json.Unmarshal(estimatedJSON, &rec.EstimatedImpact)
		recs = append(recs, rec)
	}
	return recs, nil
}

// PersistOptimizationRecommendations replaces prior recommendations for a scenario.
func (s *Store) PersistOptimizationRecommendations(scenario string, recs []types.OptimizationRecommendation) error {
	if s.db == nil {
		return errors.New("store not initialized")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM optimization_recommendations WHERE scenario_name = $1", scenario); err != nil {
		return err
	}

	stmt := `
        INSERT INTO optimization_recommendations
        (id, scenario_name, recommendation_type, title, description, current_state, recommended_state, estimated_impact, confidence_score, priority, status, created_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`

	for _, rec := range recs {
		currentJSON, _ := json.Marshal(rec.CurrentState)
		recommendedJSON, _ := json.Marshal(rec.RecommendedState)
		estimatedJSON, _ := json.Marshal(rec.EstimatedImpact)
		if _, err := tx.Exec(stmt,
			rec.ID,
			rec.ScenarioName,
			rec.RecommendationType,
			rec.Title,
			rec.Description,
			currentJSON,
			recommendedJSON,
			estimatedJSON,
			rec.ConfidenceScore,
			rec.Priority,
			rec.Status,
			rec.CreatedAt,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// UpdateScenarioMetadata upserts cached metadata about a scenario's service.json.
func (s *Store) UpdateScenarioMetadata(name string, cfg *types.ServiceConfig, scenarioPath string) error {
	if s.db == nil {
		return nil
	}

	payload, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	query := `
        INSERT INTO scenario_metadata (scenario_name, display_name, description, tags, service_config, file_path, last_scanned)
        VALUES ($1,$2,$3,$4,$5,$6,$7)
        ON CONFLICT (scenario_name) DO UPDATE SET
            display_name = EXCLUDED.display_name,
            description = EXCLUDED.description,
            tags = EXCLUDED.tags,
            service_config = EXCLUDED.service_config,
            file_path = EXCLUDED.file_path,
            last_scanned = EXCLUDED.last_scanned,
            updated_at = NOW();
    `

	_, err = s.db.Exec(query,
		name,
		cfg.Service.DisplayName,
		cfg.Service.Description,
		pqArray(cfg.Service.Tags),
		payload,
		scenarioPath,
		time.Now(),
	)
	return err
}

// LoadScenarioMetadataMap fetches cached summaries for each scenario.
func (s *Store) LoadScenarioMetadataMap() (map[string]types.ScenarioSummary, error) {
	summaries := map[string]types.ScenarioSummary{}
	if s.db == nil {
		return summaries, nil
	}

	rows, err := s.db.Query("SELECT scenario_name, display_name, description, tags, last_scanned FROM scenario_metadata")
	if err != nil {
		return summaries, err
	}
	defer rows.Close()

	for rows.Next() {
		var summary types.ScenarioSummary
		var tags pq.StringArray
		var lastScanned sql.NullTime
		if err := rows.Scan(&summary.Name, &summary.DisplayName, &summary.Description, &tags, &lastScanned); err != nil {
			continue
		}
		summary.Tags = []string(tags)
		if lastScanned.Valid {
			summary.LastScanned = &lastScanned.Time
		}
		summaries[summary.Name] = summary
	}

	return summaries, nil
}

// CollectAnalysisMetrics aggregates summary stats for health endpoints.
func (s *Store) CollectAnalysisMetrics() (map[string]interface{}, error) {
	metrics := map[string]interface{}{
		"scenarios_found":     0,
		"resources_available": 0,
		"database_status":     "unknown",
		"last_analysis":       nil,
	}

	if s.db == nil {
		return metrics, errors.New("database not initialized")
	}

	if err := s.db.Ping(); err != nil {
		metrics["database_status"] = "unreachable"
		return metrics, err
	}

	metrics["database_status"] = "connected"

	var scenarioCount int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM scenario_metadata").Scan(&scenarioCount); err != nil {
		metrics["database_status"] = "error"
		return metrics, err
	}
	metrics["scenarios_found"] = scenarioCount

	var resourceCount int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM scenario_dependencies WHERE dependency_type = 'resource'").Scan(&resourceCount); err != nil {
		metrics["database_status"] = "error"
		return metrics, err
	}
	metrics["resources_available"] = resourceCount

	var lastAnalysis sql.NullTime
	if err := s.db.QueryRow("SELECT MAX(last_scanned) FROM scenario_metadata").Scan(&lastAnalysis); err == nil && lastAnalysis.Valid {
		metrics["last_analysis"] = lastAnalysis.Time.UTC().Format(time.RFC3339)
	}

	return metrics, nil
}

// LoadAllDependencies returns all scenario dependencies ordered for graph generation.
func (s *Store) LoadAllDependencies() ([]types.ScenarioDependency, error) {
	if s.db == nil {
		return nil, errors.New("store not initialized")
	}

	rows, err := s.db.Query(`
        SELECT scenario_name, dependency_type, dependency_name, required, purpose, access_method, configuration, discovered_at, last_verified
        FROM scenario_dependencies
        ORDER BY scenario_name, dependency_type, dependency_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deps []types.ScenarioDependency
	for rows.Next() {
		var dep types.ScenarioDependency
		var configJSON []byte
		if err := rows.Scan(&dep.ScenarioName, &dep.DependencyType, &dep.DependencyName, &dep.Required, &dep.Purpose, &dep.AccessMethod, &configJSON, &dep.DiscoveredAt, &dep.LastVerified); err != nil {
			continue
		}
		if len(configJSON) > 0 {
			_ = json.Unmarshal(configJSON, &dep.Configuration)
		}
		deps = append(deps, dep)
	}

	return deps, nil
}

func pqArray(values []string) interface{} {
	if len(values) == 0 {
		return pq.StringArray([]string{})
	}
	return pq.StringArray(values)
}

func normalizeName(name string) string {
	return strings.TrimSpace(strings.ToLower(name))
}
