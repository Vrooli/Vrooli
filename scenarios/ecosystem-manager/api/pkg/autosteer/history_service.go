package autosteer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
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
		       phase_breakdown, total_iterations, total_duration_ms,
		       user_rating, user_comments, user_feedback_at, executed_at
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
		var userRating sql.NullInt64
		var userComments sql.NullString
		var userFeedbackAt sql.NullTime

		err := rows.Scan(
			&perf.ID,
			&perf.ProfileID,
			&perf.ExecutionID,
			&perf.ScenarioName,
			&phaseBreakdownJSON,
			&perf.TotalIterations,
			&perf.TotalDuration,
			&userRating,
			&userComments,
			&userFeedbackAt,
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

		// Handle structured feedback entries
		if userRating.Valid {
			perf.UserFeedback = &UserFeedback{
				Rating:      int(userRating.Int64),
				Comments:    userComments.String,
				SubmittedAt: userFeedbackAt.Time,
			}
		}

		history = append(history, perf)
	}

	errRows := rows.Err()
	// Release the connection BEFORE the per-row feedback queries below. The
	// production pool is single-connection (SQLite), so issuing a nested query
	// while these rows are still open would deadlock waiting on the only conn.
	rows.Close()
	if errRows != nil {
		return nil, fmt.Errorf("error iterating history: %w", errRows)
	}

	// Enrich with structured feedback entries now that the outer cursor is closed.
	for i := range history {
		if history[i].ExecutionID == "" {
			continue
		}
		entries, err := s.loadFeedbackEntries(history[i].ExecutionID)
		if err != nil {
			return nil, err
		}
		history[i].FeedbackEntries = entries
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
		WHERE execution_task_id = ?
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
		SELECT id, profile_id, task_id as execution_id, scenario_name,
		       phase_breakdown, total_iterations, total_duration_ms,
		       user_rating, user_comments, user_feedback_at, executed_at
		FROM profile_executions
		WHERE task_id = ?
	`

	var perf ProfilePerformance
	var phaseBreakdownJSON []byte
	var userRating sql.NullInt64
	var userComments sql.NullString
	var userFeedbackAt sql.NullTime

	err := s.db.QueryRow(query, executionID).Scan(
		&perf.ID,
		&perf.ProfileID,
		&perf.ExecutionID,
		&perf.ScenarioName,
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
		SET user_rating = ?, user_comments = ?, user_feedback_at = ?
		WHERE task_id = ?
	`

	result, err := s.db.Exec(query, rating, comments, time.Now().UTC(), executionID)
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

	// SQLite has no gen_random_uuid()/RETURNING contract here; the id and
	// created_at are app-generated so the insert is a plain statement.
	entry := ExecutionFeedbackEntry{
		ID:        uuid.NewString(),
		CreatedAt: time.Now().UTC(),
	}
	if _, err := s.db.Exec(`
		INSERT INTO execution_feedback_entries (
			id, execution_task_id, category, severity, suggested_action, comments, metadata, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, entry.ID, execID, category, severity, req.SuggestedAction, req.Comments, metadataValue, entry.CreatedAt); err != nil {
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
	ProfileID       string                `json:"profile_id"`
	TotalExecutions int                   `json:"total_executions"`
	AvgRating       float64               `json:"avg_rating"`
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
	ScenarioName   string  `json:"scenario_name"`
	ExecutionCount int     `json:"execution_count"`
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
	totalRating := 0.0
	ratingCount := 0
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
		count       int
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
		stats := ScenarioStats{
			ScenarioName:   name,
			ExecutionCount: data.count,
		}
		if data.ratingCount > 0 {
			stats.AvgRating = data.rating / float64(data.ratingCount)
		}
		analytics.ScenarioStats = append(analytics.ScenarioStats, stats)
	}

	return analytics, nil
}
