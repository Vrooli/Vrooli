package autosteer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// HistoryService handles historical performance tracking
type HistoryService struct {
	db *sql.DB
}

// NewHistoryService creates a new history service
func NewHistoryService(db *sql.DB) *HistoryService {
	return &HistoryService{
		db: db,
	}
}

// HistoryFilters represents filters for querying execution history
type HistoryFilters struct {
	ProfileID    string
	ScenarioName string
	StartDate    *time.Time
	EndDate      *time.Time
}

// GetHistory retrieves execution history with optional filtering
func (s *HistoryService) GetHistory(filters HistoryFilters) ([]ProfilePerformance, error) {
	query := `
		SELECT id, profile_id, task_id as execution_id, scenario_name,
		       phase_breakdown, total_iterations, total_duration_ms, executed_at
		FROM profile_executions
		WHERE 1=1
	`

	var args []interface{}

	if filters.ProfileID != "" {
		query += " AND profile_id = ?"
		args = append(args, filters.ProfileID)
	}

	if filters.ScenarioName != "" {
		query += " AND scenario_name = ?"
		args = append(args, filters.ScenarioName)
	}

	if filters.StartDate != nil {
		query += " AND executed_at >= ?"
		args = append(args, filters.StartDate.UTC())
	}

	if filters.EndDate != nil {
		query += " AND executed_at <= ?"
		args = append(args, filters.EndDate.UTC())
	}

	query += " ORDER BY executed_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query history: %w", err)
	}

	// Initialize to empty slice (not nil) so it serializes as [] instead of null
	history := make([]ProfilePerformance, 0)

	for rows.Next() {
		var perf ProfilePerformance
		var phaseBreakdownJSON []byte

		err := rows.Scan(
			&perf.ID,
			&perf.ProfileID,
			&perf.ExecutionID,
			&perf.ScenarioName,
			&phaseBreakdownJSON,
			&perf.TotalIterations,
			&perf.TotalDuration,
			&perf.ExecutedAt,
		)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("failed to scan history row: %w", err)
		}

		if err := json.Unmarshal(phaseBreakdownJSON, &perf.PhaseBreakdown); err != nil {
			rows.Close()
			return nil, fmt.Errorf("failed to unmarshal phase breakdown: %w", err)
		}

		history = append(history, perf)
	}

	errRows := rows.Err()
	rows.Close()
	if errRows != nil {
		return nil, fmt.Errorf("error iterating history: %w", errRows)
	}

	return history, nil
}

// GetExecution retrieves a specific execution by ID
func (s *HistoryService) GetExecution(executionID string) (*ProfilePerformance, error) {
	query := `
		SELECT id, profile_id, task_id as execution_id, scenario_name,
		       phase_breakdown, total_iterations, total_duration_ms, executed_at
		FROM profile_executions
		WHERE task_id = ?
	`

	var perf ProfilePerformance
	var phaseBreakdownJSON []byte

	err := s.db.QueryRow(query, executionID).Scan(
		&perf.ID,
		&perf.ProfileID,
		&perf.ExecutionID,
		&perf.ScenarioName,
		&phaseBreakdownJSON,
		&perf.TotalIterations,
		&perf.TotalDuration,
		&perf.ExecutedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("execution not found: %s", executionID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query execution: %w", err)
	}

	if err := json.Unmarshal(phaseBreakdownJSON, &perf.PhaseBreakdown); err != nil {
		return nil, fmt.Errorf("failed to unmarshal phase breakdown: %w", err)
	}

	return &perf, nil
}

// ProfileAnalytics represents aggregated analytics for a profile
type ProfileAnalytics struct {
	ProfileID       string                `json:"profile_id"`
	TotalExecutions int                   `json:"total_executions"`
	AvgIterations   float64               `json:"avg_iterations"`
	AvgDuration     int64                 `json:"avg_duration"`
	PhaseStats      map[string]PhaseStats `json:"phase_stats"`
	ScenarioStats   []ScenarioStats       `json:"scenario_stats"`
}

// PhaseStats represents aggregated statistics for a specific steer skill.
type PhaseStats struct {
	SkillName        string  `json:"skill_name"`
	TotalExecutions  int     `json:"total_executions"`
	AvgIterations    float64 `json:"avg_iterations"`
	AvgWeightedDelta float64 `json:"avg_weighted_delta"`
}

// ScenarioStats represents statistics for a specific scenario
type ScenarioStats struct {
	ScenarioName   string `json:"scenario_name"`
	ExecutionCount int    `json:"execution_count"`
}

// GetProfileAnalytics retrieves aggregated analytics for a profile
func (s *HistoryService) GetProfileAnalytics(profileID string) (*ProfileAnalytics, error) {
	// Get all executions for this profile
	history, err := s.GetHistory(HistoryFilters{ProfileID: profileID})
	if err != nil {
		return nil, err
	}

	if len(history) == 0 {
		return &ProfileAnalytics{
			ProfileID:       profileID,
			TotalExecutions: 0,
			PhaseStats:      make(map[string]PhaseStats),
			ScenarioStats:   []ScenarioStats{},
		}, nil
	}

	analytics := &ProfileAnalytics{
		ProfileID:       profileID,
		TotalExecutions: len(history),
		PhaseStats:      make(map[string]PhaseStats),
	}

	// Calculate aggregate statistics
	totalIterations := 0
	totalDuration := int64(0)

	// Skill statistics accumulator (keyed by skill name)
	phaseData := make(map[string]struct {
		count         int
		iterations    int
		weightedDelta float64
	})

	// Scenario statistics accumulator
	scenarioData := make(map[string]struct {
		count int
	})

	for _, exec := range history {
		// Overall statistics
		totalIterations += exec.TotalIterations
		totalDuration += exec.TotalDuration

		// Skill statistics
		for _, phase := range exec.PhaseBreakdown {
			key := phase.SkillName
			data := phaseData[key]
			data.count++
			data.iterations += phase.Iterations
			data.weightedDelta += phase.WeightedDelta
			phaseData[key] = data
		}

		// Scenario statistics
		data := scenarioData[exec.ScenarioName]
		data.count++
		scenarioData[exec.ScenarioName] = data
	}

	// Calculate averages
	analytics.AvgIterations = float64(totalIterations) / float64(len(history))
	analytics.AvgDuration = totalDuration / int64(len(history))

	// Build skill stats
	for skillName, data := range phaseData {
		analytics.PhaseStats[skillName] = PhaseStats{
			SkillName:        skillName,
			TotalExecutions:  data.count,
			AvgIterations:    float64(data.iterations) / float64(data.count),
			AvgWeightedDelta: data.weightedDelta / float64(data.count),
		}
	}

	// Build scenario stats
	for name, data := range scenarioData {
		analytics.ScenarioStats = append(analytics.ScenarioStats, ScenarioStats{
			ScenarioName:   name,
			ExecutionCount: data.count,
		})
	}

	return analytics, nil
}
