package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

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

	orphaned := make([]string, 0)
	for rows.Next() {
		var dep string
		if scanErr := rows.Scan(&dep); scanErr != nil {
			continue
		}
		if _, ok := known[normalizeName(dep)]; ok {
			continue
		}
		orphaned = append(orphaned, dep)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("iterate scenario dependencies: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close scenario dependency query: %w", err)
	}

	// The production database intentionally has a single connection. Release the
	// query cursor before issuing writes so this maintenance task cannot exhaust
	// that pool and make the health endpoint time out.
	for _, dep := range orphaned {
		if _, err := s.db.Exec(`
            DELETE FROM scenario_dependencies
            WHERE dependency_type = 'scenario' AND dependency_name = ?`, dep); err != nil {
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
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec("DELETE FROM scenario_dependencies WHERE scenario_name = ?", analysis.Scenario); err != nil {
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
            VALUES (?,?,?,?,?,?,?,?,?)`,
			dep.ScenarioName, dep.DependencyType, dep.DependencyName, dep.Required,
			dep.Purpose, dep.AccessMethod, string(configJSON), dep.DiscoveredAt, dep.LastVerified); err != nil {
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
        WHERE scenario_name = ?
        ORDER BY dependency_type, dependency_name`, scenario)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		dep, err := scanDependency(rows)
		if err != nil {
			continue
		}
		result[dependencyBucket(dep.DependencyType)] = append(result[dependencyBucket(dep.DependencyType)], dep)
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
        WHERE scenario_name = ?
        ORDER BY created_at DESC`, scenario)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recs []types.OptimizationRecommendation
	for rows.Next() {
		var rec types.OptimizationRecommendation
		var currentJSON, recommendedJSON, estimatedJSON sql.NullString
		if err := rows.Scan(&rec.ID, &rec.ScenarioName, &rec.RecommendationType, &rec.Title, &rec.Description, &currentJSON, &recommendedJSON, &estimatedJSON, &rec.ConfidenceScore, &rec.Priority, &rec.Status, &rec.CreatedAt); err != nil {
			continue
		}
		if currentJSON.Valid {
			_ = json.Unmarshal([]byte(currentJSON.String), &rec.CurrentState)
		}
		if recommendedJSON.Valid {
			_ = json.Unmarshal([]byte(recommendedJSON.String), &rec.RecommendedState)
		}
		if estimatedJSON.Valid {
			_ = json.Unmarshal([]byte(estimatedJSON.String), &rec.EstimatedImpact)
		}
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
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec("DELETE FROM optimization_recommendations WHERE scenario_name = ?", scenario); err != nil {
		return err
	}

	stmt := `
        INSERT INTO optimization_recommendations
        (id, scenario_name, recommendation_type, title, description, current_state, recommended_state, estimated_impact, confidence_score, priority, status, created_at)
        VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`

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
			string(currentJSON),
			string(recommendedJSON),
			string(estimatedJSON),
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

	tagsJSON, err := json.Marshal(cfg.Service.Tags)
	if err != nil {
		return err
	}

	query := `
        INSERT INTO scenario_metadata (scenario_name, display_name, description, tags, service_config, file_path, last_scanned)
        VALUES (?,?,?,?,?,?,?)
        ON CONFLICT (scenario_name) DO UPDATE SET
            display_name = EXCLUDED.display_name,
            description = EXCLUDED.description,
            tags = EXCLUDED.tags,
            service_config = EXCLUDED.service_config,
            file_path = EXCLUDED.file_path,
            last_scanned = EXCLUDED.last_scanned,
            updated_at = datetime('now');
    `

	_, err = s.db.Exec(query,
		name,
		cfg.Service.DisplayName,
		cfg.Service.Description,
		string(tagsJSON),
		string(payload),
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
		var tagsJSON sql.NullString
		var lastScanned sql.NullTime
		if err := rows.Scan(&summary.Name, &summary.DisplayName, &summary.Description, &tagsJSON, &lastScanned); err != nil {
			continue
		}
		if tagsJSON.Valid && tagsJSON.String != "" {
			_ = json.Unmarshal([]byte(tagsJSON.String), &summary.Tags)
		}
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
		dep, err := scanDependency(rows)
		if err != nil {
			continue
		}
		deps = append(deps, dep)
	}

	return deps, nil
}

// ReplaceGraphEdges atomically replaces the entire unified graph_edges store
// with the supplied merged edge set. Used by a full fleet rebuild; the
// incremental sweeper uses the per-scenario upsert path instead.
func (s *Store) ReplaceGraphEdges(edges []types.UnifiedGraphEdge) error {
	if s.db == nil {
		return errors.New("store not initialized")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec("DELETE FROM graph_edges"); err != nil {
		return err
	}
	if err := insertGraphEdgesTx(tx, edges); err != nil {
		return err
	}
	return tx.Commit()
}

// UpsertGraphEdgesForScenario replaces the edges originating from a single
// source scenario with the supplied merged set. This is the incremental,
// freshness-gated path: it never touches other scenarios' rows, so a partial
// (per-scenario) re-ingest is safe and idempotent.
func (s *Store) UpsertGraphEdgesForScenario(scenario string, edges []types.UnifiedGraphEdge) error {
	if s.db == nil {
		return errors.New("store not initialized")
	}
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return errors.New("scenario is required")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec("DELETE FROM graph_edges WHERE from_scenario = ?", scenario); err != nil {
		return err
	}
	if err := insertGraphEdgesTx(tx, edges); err != nil {
		return err
	}
	return tx.Commit()
}

// MarkScenarioEdgesStale flags every edge from the named scenario as stale
// without dropping it. Used by the sweeper's graceful-degradation path so an
// upstream-source outage never zeroes out a scenario's contribution mid-cycle.
func (s *Store) MarkScenarioEdgesStale(scenario string) error {
	if s.db == nil {
		return errors.New("store not initialized")
	}
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil
	}
	_, err := s.db.Exec("UPDATE graph_edges SET stale = 1 WHERE from_scenario = ?", scenario)
	return err
}

func insertGraphEdgesTx(tx *sql.Tx, edges []types.UnifiedGraphEdge) error {
	stmt := `
        INSERT INTO graph_edges
        (from_scenario, to_node, kind, evidence_source, confidence, required, evidence_json, stale, last_verified, updated_at)
        VALUES (?,?,?,?,?,?,?,?,?,?)`
	now := time.Now().UTC()
	for _, edge := range edges {
		evidenceJSON, _ := json.Marshal(edge.Evidence)
		if len(edge.Evidence) == 0 {
			evidenceJSON = []byte("[]")
		}
		lastVerified := edge.LastVerified
		if lastVerified.IsZero() {
			lastVerified = now
		}
		if _, err := tx.Exec(stmt,
			edge.From, edge.To, edge.Kind, edge.Source, edge.Confidence, edge.Required,
			string(evidenceJSON), boolToInt(edge.Stale), lastVerified.UTC().Format(time.RFC3339Nano), now.Format(time.RFC3339Nano),
		); err != nil {
			return err
		}
	}
	return nil
}

// LoadGraphEdges returns every persisted unified edge, ordered for deterministic
// graph construction.
func (s *Store) LoadGraphEdges() ([]types.UnifiedGraphEdge, error) {
	if s.db == nil {
		return nil, errors.New("store not initialized")
	}
	rows, err := s.db.Query(`
        SELECT from_scenario, to_node, kind, evidence_source, confidence, required, evidence_json, stale, last_verified
        FROM graph_edges
        ORDER BY from_scenario, to_node`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []types.UnifiedGraphEdge
	for rows.Next() {
		var (
			edge         types.UnifiedGraphEdge
			required     int
			stale        int
			evidenceJSON sql.NullString
			lastVerified any
		)
		if err := rows.Scan(&edge.From, &edge.To, &edge.Kind, &edge.Source, &edge.Confidence, &required, &evidenceJSON, &stale, &lastVerified); err != nil {
			continue
		}
		edge.Required = required != 0
		edge.Stale = stale != 0
		if evidenceJSON.Valid && evidenceJSON.String != "" {
			_ = json.Unmarshal([]byte(evidenceJSON.String), &edge.Evidence)
		}
		edge.LastVerified = parseDBTime(lastVerified)
		edges = append(edges, edge)
	}
	return edges, nil
}

// GraphEdgeStats summarizes the persisted unified graph store for status surfaces.
type GraphEdgeStats struct {
	TotalEdges    int
	ScenarioEdges int
	ResourceEdges int
	StaleEdges    int
	BySource      map[string]int
	LastUpdated   time.Time
}

// GraphEdgeStats aggregates summary counts over the unified graph store.
func (s *Store) GraphEdgeStats() (GraphEdgeStats, error) {
	stats := GraphEdgeStats{BySource: map[string]int{}}
	if s.db == nil {
		return stats, errors.New("store not initialized")
	}
	rows, err := s.db.Query(`SELECT kind, evidence_source, stale, COUNT(*) FROM graph_edges GROUP BY kind, evidence_source, stale`)
	if err != nil {
		return stats, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			kind, source string
			stale, count int
		)
		if err := rows.Scan(&kind, &source, &stale, &count); err != nil {
			continue
		}
		stats.TotalEdges += count
		switch kind {
		case "resource":
			stats.ResourceEdges += count
		default:
			stats.ScenarioEdges += count
		}
		if stale != 0 {
			stats.StaleEdges += count
		}
		stats.BySource[source] += count
	}
	var lastUpdated sql.NullString
	if err := s.db.QueryRow(`SELECT MAX(updated_at) FROM graph_edges`).Scan(&lastUpdated); err == nil && lastUpdated.Valid {
		stats.LastUpdated = parseTimeString(lastUpdated.String)
	}
	return stats, nil
}

// GetIngestDigest returns the scenario tree digest of the last successful ingest.
func (s *Store) GetIngestDigest(scenario string) (string, bool, error) {
	if s.db == nil {
		return "", false, errors.New("store not initialized")
	}
	var digest string
	err := s.db.QueryRow("SELECT digest FROM graph_ingest_state WHERE scenario = ?", scenario).Scan(&digest)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return digest, true, nil
}

// SetIngestDigest records the scenario tree digest after a successful ingest.
func (s *Store) SetIngestDigest(scenario, digest string) error {
	if s.db == nil {
		return errors.New("store not initialized")
	}
	_, err := s.db.Exec(`
        INSERT INTO graph_ingest_state (scenario, digest, last_ingested_at)
        VALUES (?,?,?)
        ON CONFLICT (scenario) DO UPDATE SET
            digest = EXCLUDED.digest,
            last_ingested_at = EXCLUDED.last_ingested_at`,
		scenario, digest, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func normalizeName(name string) string {
	return strings.TrimSpace(strings.ToLower(name))
}

func dependencyBucket(depType string) string {
	switch depType {
	case "resource":
		return "resources"
	case "scenario":
		return "scenarios"
	case "shared_workflow":
		return "shared_workflows"
	default:
		return depType
	}
}

type dependencyScanner interface {
	Scan(dest ...any) error
}

func scanDependency(row dependencyScanner) (types.ScenarioDependency, error) {
	var dep types.ScenarioDependency
	var configJSON sql.NullString
	var discoveredAt, lastVerified any
	if err := row.Scan(
		&dep.ScenarioName,
		&dep.DependencyType,
		&dep.DependencyName,
		&dep.Required,
		&dep.Purpose,
		&dep.AccessMethod,
		&configJSON,
		&discoveredAt,
		&lastVerified,
	); err != nil {
		return dep, err
	}
	if configJSON.Valid && configJSON.String != "" {
		_ = json.Unmarshal([]byte(configJSON.String), &dep.Configuration)
	}
	dep.DiscoveredAt = parseDBTime(discoveredAt)
	dep.LastVerified = parseDBTime(lastVerified)
	return dep, nil
}

func parseDBTime(value any) time.Time {
	switch v := value.(type) {
	case time.Time:
		return v
	case string:
		return parseTimeString(v)
	case []byte:
		return parseTimeString(string(v))
	default:
		return time.Time{}
	}
}

func parseTimeString(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t
		}
	}
	return time.Time{}
}
