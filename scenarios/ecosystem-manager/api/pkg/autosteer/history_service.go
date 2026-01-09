package autosteer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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
		SELECT id, profile_id, task_id as execution_id, scenario_name, start_metrics,
		       end_metrics, phase_breakdown, total_iterations, total_duration_ms,
		       user_rating, user_comments, user_feedback_at, executed_at
		FROM profile_executions
		WHERE 1=1
	`

	var args []interface{}
	argIndex := 1

	if filters.ProfileID != "" {
		query += fmt.Sprintf(" AND profile_id = $%d", argIndex)
		args = append(args, filters.ProfileID)
		argIndex++
	}

	if filters.ScenarioName != "" {
		query += fmt.Sprintf(" AND scenario_name = $%d", argIndex)
		args = append(args, filters.ScenarioName)
		argIndex++
	}

	if filters.StartDate != nil {
		query += fmt.Sprintf(" AND executed_at >= $%d", argIndex)
		args = append(args, filters.StartDate)
		argIndex++
	}

	if filters.EndDate != nil {
		query += fmt.Sprintf(" AND executed_at <= $%d", argIndex)
		args = append(args, filters.EndDate)
		argIndex++
	}

	query += " ORDER BY executed_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query history: %w", err)
	}
	defer rows.Close()

	// Initialize to empty slice (not nil) so it serializes as [] instead of null
	history := make([]ProfilePerformance, 0)

	for rows.Next() {
		var perf ProfilePerformance
		var startMetricsJSON, endMetricsJSON, phaseBreakdownJSON []byte
		var userRating sql.NullInt64
		var userComments sql.NullString
		var userFeedbackAt sql.NullTime

		err := rows.Scan(
			&perf.ID,
			&perf.ProfileID,
			&perf.ExecutionID,
			&perf.ScenarioName,
			&startMetricsJSON,
			&endMetricsJSON,
			&phaseBreakdownJSON,
			&perf.TotalIterations,
			&perf.TotalDuration,
			&userRating,
			&userComments,
			&userFeedbackAt,
			&perf.ExecutedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan history row: %w", err)
		}

		// Unmarshal JSON fields
		if err := json.Unmarshal(startMetricsJSON, &perf.StartMetrics); err != nil {
			return nil, fmt.Errorf("failed to unmarshal start metrics: %w", err)
		}

		if err := json.Unmarshal(endMetricsJSON, &perf.EndMetrics); err != nil {
			return nil, fmt.Errorf("failed to unmarshal end metrics: %w", err)
		}

		if err := json.Unmarshal(phaseBreakdownJSON, &perf.PhaseBreakdown); err != nil {
			return nil, fmt.Errorf("failed to unmarshal phase breakdown: %w", err)
		}

		// Handle structured feedback entries
		if userRating.Valid {
			perf.UserFeedback = &UserFeedback{
				Rating:      int(userRating.Int64),
				Comments:    userComments.String,
				SubmittedAt: userFeedbackAt.Time,
			}
		}

		if perf.ExecutionID != "" {
			entries, err := s.loadFeedbackEntries(perf.ExecutionID)
			if err != nil {
				return nil, err
			}
			perf.FeedbackEntries = entries
		}

		history = append(history, perf)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating history: %w", err)
	}

	return history, nil
}

func (s *HistoryService) loadFeedbackEntries(executionID string) ([]ExecutionFeedbackEntry, error) {
	if strings.TrimSpace(executionID) == "" {
		return nil, nil
	}

	query := `
		SELECT id, category, severity, suggested_action, comments, metadata, created_at
		FROM execution_feedback_entries
		WHERE execution_task_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query feedback entries: %w", err)
	}
	defer rows.Close()

	var entries []ExecutionFeedbackEntry
	for rows.Next() {
		var entry ExecutionFeedbackEntry
		var metadataJSON []byte
		var suggestedAction sql.NullString
		var comments sql.NullString

		if err := rows.Scan(
			&entry.ID,
			&entry.Category,
			&entry.Severity,
			&suggestedAction,
			&comments,
			&metadataJSON,
			&entry.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan feedback row: %w", err)
		}

		if suggestedAction.Valid {
			entry.SuggestedAction = suggestedAction.String
		}
		if comments.Valid {
			entry.Comments = comments.String
		}

		if len(metadataJSON) > 0 {
			var metadata map[string]any
			if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal feedback metadata: %w", err)
			}
			entry.Metadata = metadata
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating feedback rows: %w", err)
	}

	return entries, nil
}

// GetExecution retrieves a specific execution by ID
func (s *HistoryService) GetExecution(executionID string) (*ProfilePerformance, error) {
	query := `
		SELECT id, profile_id, task_id as execution_id, scenario_name, start_metrics,
		       end_metrics, phase_breakdown, total_iterations, total_duration_ms,
		       user_rating, user_comments, user_feedback_at, executed_at
		FROM profile_executions
		WHERE task_id = $1
	`

	var perf ProfilePerformance
	var startMetricsJSON, endMetricsJSON, phaseBreakdownJSON []byte
	var userRating sql.NullInt64
	var userComments sql.NullString
	var userFeedbackAt sql.NullTime

	err := s.db.QueryRow(query, executionID).Scan(
		&perf.ID,
		&perf.ProfileID,
		&perf.ExecutionID,
		&perf.ScenarioName,
		&startMetricsJSON,
		&endMetricsJSON,
		&phaseBreakdownJSON,
		&perf.TotalIterations,
		&perf.TotalDuration,
		&userRating,
		&userComments,
		&userFeedbackAt,
		&perf.ExecutedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("execution not found: %s", executionID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query execution: %w", err)
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal(startMetricsJSON, &perf.StartMetrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal start metrics: %w", err)
	}

	if err := json.Unmarshal(endMetricsJSON, &perf.EndMetrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal end metrics: %w", err)
	}

	if err := json.Unmarshal(phaseBreakdownJSON, &perf.PhaseBreakdown); err != nil {
		return nil, fmt.Errorf("failed to unmarshal phase breakdown: %w", err)
	}

	// Handle structured feedback entries
	if userRating.Valid {
		perf.UserFeedback = &UserFeedback{
			Rating:      int(userRating.Int64),
			Comments:    userComments.String,
			SubmittedAt: userFeedbackAt.Time,
		}
	}

	if perf.ExecutionID != "" {
		entries, err := s.loadFeedbackEntries(perf.ExecutionID)
		if err != nil {
			return nil, err
		}
		perf.FeedbackEntries = entries
	}

	return &perf, nil
}

// SubmitFeedback submits user feedback for an execution
func (s *HistoryService) SubmitFeedback(executionID string, rating int, comments string) error {
	query := `
		UPDATE profile_executions
		SET user_rating = $1, user_comments = $2, user_feedback_at = $3
		WHERE task_id = $4
	`

	result, err := s.db.Exec(query, rating, comments, time.Now(), executionID)
	if err != nil {
		return fmt.Errorf("failed to update feedback: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	return nil
}

// SubmitFeedbackEntry records structured feedback for an execution.
func (s *HistoryService) SubmitFeedbackEntry(executionID string, req ExecutionFeedbackRequest) (*ExecutionFeedbackEntry, error) {
	execID := strings.TrimSpace(executionID)
	if execID == "" {
		return nil, fmt.Errorf("execution ID is required")
	}

	category := strings.TrimSpace(req.Category)
	if category == "" {
		return nil, fmt.Errorf("category is required")
	}

	severity := strings.TrimSpace(req.Severity)
	if severity == "" {
		return nil, fmt.Errorf("severity is required")
	}

	var metadataValue interface{}
	if len(req.Metadata) > 0 {
		payload, err := json.Marshal(req.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataValue = string(payload)
	}

	var entry ExecutionFeedbackEntry
	err := s.db.QueryRow(`
		INSERT INTO execution_feedback_entries (
			execution_task_id, category, severity, suggested_action, comments, metadata
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`, execID, category, severity, req.SuggestedAction, req.Comments, metadataValue).Scan(&entry.ID, &entry.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert feedback entry: %w", err)
	}

	entry.Category = category
	entry.Severity = severity
	entry.SuggestedAction = req.SuggestedAction
	entry.Comments = req.Comments
	if len(req.Metadata) > 0 {
		entry.Metadata = req.Metadata
	}

	return &entry, nil
}

// ProfileAnalytics represents aggregated analytics for a profile
type ProfileAnalytics struct {
	ProfileID       string                   `json:"profile_id"`
	TotalExecutions int                      `json:"total_executions"`
	AvgRating       float64                  `json:"avg_rating"`
	AvgIterations   float64                  `json:"avg_iterations"`
	AvgDuration     int64                    `json:"avg_duration"`
	PhaseStats      map[SteerMode]PhaseStats `json:"phase_stats"`
	ScenarioStats   []ScenarioStats          `json:"scenario_stats"`
}

// PhaseStats represents statistics for a specific mode
type PhaseStats struct {
	TotalExecutions  int     `json:"total_executions"`
	AvgIterations    float64 `json:"avg_iterations"`
	AvgDuration      int64   `json:"avg_duration"`
	AvgEffectiveness float64 `json:"avg_effectiveness"`
}

// ScenarioStats represents statistics for a specific scenario
type ScenarioStats struct {
	ScenarioName   string  `json:"scenario_name"`
	ExecutionCount int     `json:"execution_count"`
	AvgImprovement float64 `json:"avg_improvement"`
	AvgRating      float64 `json:"avg_rating"`
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
			PhaseStats:      make(map[SteerMode]PhaseStats),
			ScenarioStats:   []ScenarioStats{},
		}, nil
	}

	analytics := &ProfileAnalytics{
		ProfileID:       profileID,
		TotalExecutions: len(history),
		PhaseStats:      make(map[SteerMode]PhaseStats),
	}

	// Calculate aggregate statistics
	totalRating := 0.0
	ratingCount := 0
	totalIterations := 0
	totalDuration := int64(0)

	// Phase statistics accumulator
	phaseData := make(map[SteerMode]struct {
		count         int
		iterations    int
		duration      int64
		effectiveness float64
	})

	// Scenario statistics accumulator
	scenarioData := make(map[string]struct {
		count       int
		improvement float64
		rating      float64
		ratingCount int
	})

	for _, exec := range history {
		// Overall statistics
		totalIterations += exec.TotalIterations
		totalDuration += exec.TotalDuration

		if exec.UserFeedback != nil {
			totalRating += float64(exec.UserFeedback.Rating)
			ratingCount++
		}

		// Phase statistics
		for _, phase := range exec.PhaseBreakdown {
			data := phaseData[phase.Mode]
			data.count++
			data.iterations += phase.Iterations
			data.duration += phase.Duration
			data.effectiveness += phase.Effectiveness
			phaseData[phase.Mode] = data
		}

		// Scenario statistics
		improvement := exec.EndMetrics.OperationalTargetsPercentage - exec.StartMetrics.OperationalTargetsPercentage
		data := scenarioData[exec.ScenarioName]
		data.count++
		data.improvement += improvement
		if exec.UserFeedback != nil {
			data.rating += float64(exec.UserFeedback.Rating)
			data.ratingCount++
		}
		scenarioData[exec.ScenarioName] = data
	}

	// Calculate averages
	analytics.AvgIterations = float64(totalIterations) / float64(len(history))
	analytics.AvgDuration = totalDuration / int64(len(history))

	if ratingCount > 0 {
		analytics.AvgRating = totalRating / float64(ratingCount)
	}

	// Build phase stats
	for mode, data := range phaseData {
		analytics.PhaseStats[mode] = PhaseStats{
			TotalExecutions:  data.count,
			AvgIterations:    float64(data.iterations) / float64(data.count),
			AvgDuration:      data.duration / int64(data.count),
			AvgEffectiveness: data.effectiveness / float64(data.count),
		}
	}

	// Build scenario stats
	for name, data := range scenarioData {
		stats := ScenarioStats{
			ScenarioName:   name,
			ExecutionCount: data.count,
			AvgImprovement: data.improvement / float64(data.count),
		}
		if data.ratingCount > 0 {
			stats.AvgRating = data.rating / float64(data.ratingCount)
		}
		analytics.ScenarioStats = append(analytics.ScenarioStats, stats)
	}

	return analytics, nil
}
