// Package repository implements UX metrics persistence using PostgreSQL.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/vrooli/browser-automation-studio/services/uxmetrics"
	"github.com/vrooli/browser-automation-studio/services/uxmetrics/contracts"
)

// PostgresRepository implements uxmetrics.Repository using PostgreSQL.
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL-backed repository.
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// SaveInteractionTrace persists a single interaction trace.
func (r *PostgresRepository) SaveInteractionTrace(ctx context.Context, trace *contracts.InteractionTrace) error {
	query := `
		INSERT INTO ux_interaction_traces
		(id, execution_id, step_index, action_type, element_id, selector,
		 position_x, position_y, timestamp, duration_ms, success, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	var posX, posY *float64
	if trace.Position != nil {
		posX, posY = &trace.Position.X, &trace.Position.Y
	}

	metadataJSON, err := json.Marshal(trace.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	_, err = r.db.ExecContext(ctx, query,
		trace.ID, trace.ExecutionID, trace.StepIndex, trace.ActionType,
		trace.ElementID, trace.Selector, posX, posY,
		trace.Timestamp, trace.DurationMs, trace.Success, metadataJSON,
	)
	return err
}

// SaveCursorPath persists a cursor path for a step.
func (r *PostgresRepository) SaveCursorPath(ctx context.Context, executionID uuid.UUID, path *contracts.CursorPath) error {
	query := `
		INSERT INTO ux_cursor_paths
		(execution_id, step_index, points, total_distance_px, direct_distance_px,
		 duration_ms, directness, zigzag_score, average_speed, max_speed, hesitation_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (execution_id, step_index)
		DO UPDATE SET points = EXCLUDED.points,
					  total_distance_px = EXCLUDED.total_distance_px,
					  direct_distance_px = EXCLUDED.direct_distance_px,
					  directness = EXCLUDED.directness,
					  zigzag_score = EXCLUDED.zigzag_score,
					  average_speed = EXCLUDED.average_speed,
					  max_speed = EXCLUDED.max_speed,
					  hesitation_count = EXCLUDED.hesitation_count
	`

	pointsJSON, err := json.Marshal(path.Points)
	if err != nil {
		pointsJSON = []byte("[]")
	}

	_, err = r.db.ExecContext(ctx, query,
		executionID, path.StepIndex, pointsJSON, path.TotalDistancePx,
		path.DirectDistance, path.DurationMs, path.Directness,
		path.ZigZagScore, path.AverageSpeed, path.MaxSpeed, path.Hesitations,
	)
	return err
}

// SaveExecutionMetrics persists computed execution metrics.
func (r *PostgresRepository) SaveExecutionMetrics(ctx context.Context, metrics *contracts.ExecutionMetrics) error {
	query := `
		INSERT INTO ux_execution_metrics
		(execution_id, workflow_id, computed_at, total_duration_ms, step_count,
		 successful_steps, failed_steps, total_retries, avg_step_duration_ms,
		 total_cursor_distance, overall_friction_score, friction_signals,
		 step_metrics, summary)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (execution_id) DO UPDATE SET
			computed_at = EXCLUDED.computed_at,
			total_duration_ms = EXCLUDED.total_duration_ms,
			step_count = EXCLUDED.step_count,
			successful_steps = EXCLUDED.successful_steps,
			failed_steps = EXCLUDED.failed_steps,
			total_retries = EXCLUDED.total_retries,
			avg_step_duration_ms = EXCLUDED.avg_step_duration_ms,
			total_cursor_distance = EXCLUDED.total_cursor_distance,
			overall_friction_score = EXCLUDED.overall_friction_score,
			friction_signals = EXCLUDED.friction_signals,
			step_metrics = EXCLUDED.step_metrics,
			summary = EXCLUDED.summary
	`

	signalsJSON, _ := json.Marshal(metrics.FrictionSignals)
	stepMetricsJSON, _ := json.Marshal(metrics.StepMetrics)
	summaryJSON, _ := json.Marshal(metrics.Summary)

	_, err := r.db.ExecContext(ctx, query,
		metrics.ExecutionID, metrics.WorkflowID, metrics.ComputedAt,
		metrics.TotalDurationMs, metrics.StepCount, metrics.SuccessfulSteps,
		metrics.FailedSteps, metrics.TotalRetries, metrics.AvgStepDurationMs,
		metrics.TotalCursorDist, metrics.OverallFriction,
		signalsJSON, stepMetricsJSON, summaryJSON,
	)
	return err
}

// GetExecutionMetrics retrieves computed metrics for an execution.
func (r *PostgresRepository) GetExecutionMetrics(ctx context.Context, executionID uuid.UUID) (*contracts.ExecutionMetrics, error) {
	query := `
		SELECT execution_id, workflow_id, computed_at, total_duration_ms,
			   step_count, successful_steps, failed_steps, total_retries,
			   avg_step_duration_ms, total_cursor_distance, overall_friction_score,
			   friction_signals, step_metrics, summary
		FROM ux_execution_metrics
		WHERE execution_id = $1
	`

	var m contracts.ExecutionMetrics
	var signalsJSON, stepMetricsJSON, summaryJSON []byte

	err := r.db.QueryRowContext(ctx, query, executionID).Scan(
		&m.ExecutionID, &m.WorkflowID, &m.ComputedAt, &m.TotalDurationMs,
		&m.StepCount, &m.SuccessfulSteps, &m.FailedSteps, &m.TotalRetries,
		&m.AvgStepDurationMs, &m.TotalCursorDist, &m.OverallFriction,
		&signalsJSON, &stepMetricsJSON, &summaryJSON,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal(signalsJSON, &m.FrictionSignals)
	_ = json.Unmarshal(stepMetricsJSON, &m.StepMetrics)
	_ = json.Unmarshal(summaryJSON, &m.Summary)

	return &m, nil
}

// GetStepMetrics retrieves metrics for a specific step.
func (r *PostgresRepository) GetStepMetrics(ctx context.Context, executionID uuid.UUID, stepIndex int) (*contracts.StepMetrics, error) {
	metrics, err := r.GetExecutionMetrics(ctx, executionID)
	if err != nil || metrics == nil {
		return nil, err
	}

	for _, sm := range metrics.StepMetrics {
		if sm.StepIndex == stepIndex {
			return &sm, nil
		}
	}
	return nil, nil
}

// ListInteractionTraces retrieves all interaction traces for an execution.
func (r *PostgresRepository) ListInteractionTraces(ctx context.Context, executionID uuid.UUID) ([]contracts.InteractionTrace, error) {
	query := `
		SELECT id, execution_id, step_index, action_type, element_id, selector,
			   position_x, position_y, timestamp, duration_ms, success, metadata
		FROM ux_interaction_traces
		WHERE execution_id = $1
		ORDER BY timestamp ASC
	`

	rows, err := r.db.QueryContext(ctx, query, executionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	traces := make([]contracts.InteractionTrace, 0)
	for rows.Next() {
		var t contracts.InteractionTrace
		var posX, posY sql.NullFloat64
		var metadataJSON []byte

		err := rows.Scan(
			&t.ID, &t.ExecutionID, &t.StepIndex, &t.ActionType,
			&t.ElementID, &t.Selector, &posX, &posY,
			&t.Timestamp, &t.DurationMs, &t.Success, &metadataJSON,
		)
		if err != nil {
			return nil, err
		}

		if posX.Valid && posY.Valid {
			t.Position = &contracts.Point{X: posX.Float64, Y: posY.Float64}
		}
		_ = json.Unmarshal(metadataJSON, &t.Metadata)

		traces = append(traces, t)
	}

	return traces, rows.Err()
}

// GetCursorPath retrieves the cursor path for a specific step.
func (r *PostgresRepository) GetCursorPath(ctx context.Context, executionID uuid.UUID, stepIndex int) (*contracts.CursorPath, error) {
	query := `
		SELECT step_index, points, total_distance_px, direct_distance_px,
			   duration_ms, directness, zigzag_score, average_speed, max_speed, hesitation_count
		FROM ux_cursor_paths
		WHERE execution_id = $1 AND step_index = $2
	`

	var path contracts.CursorPath
	var pointsJSON []byte

	err := r.db.QueryRowContext(ctx, query, executionID, stepIndex).Scan(
		&path.StepIndex, &pointsJSON, &path.TotalDistancePx, &path.DirectDistance,
		&path.DurationMs, &path.Directness, &path.ZigZagScore,
		&path.AverageSpeed, &path.MaxSpeed, &path.Hesitations,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal(pointsJSON, &path.Points)
	return &path, nil
}

// GetWorkflowMetricsAggregate computes aggregate metrics across executions.
func (r *PostgresRepository) GetWorkflowMetricsAggregate(ctx context.Context, workflowID uuid.UUID, limit int) (*contracts.WorkflowMetricsAggregate, error) {
	query := `
		SELECT
			COUNT(*) as execution_count,
			COALESCE(AVG(overall_friction_score), 0) as avg_friction,
			COALESCE(AVG(total_duration_ms), 0) as avg_duration
		FROM (
			SELECT overall_friction_score, total_duration_ms
			FROM ux_execution_metrics
			WHERE workflow_id = $1
			ORDER BY computed_at DESC
			LIMIT $2
		) recent
	`

	var agg contracts.WorkflowMetricsAggregate
	agg.WorkflowID = workflowID
	agg.HighFrictionStepFreq = make(map[int]int)

	err := r.db.QueryRowContext(ctx, query, workflowID, limit).Scan(
		&agg.ExecutionCount, &agg.AvgFrictionScore, &agg.AvgDurationMs,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get workflow metrics aggregate: %w", err)
	}

	// Compute trend direction based on recent executions
	agg.TrendDirection = r.computeTrendDirection(ctx, workflowID, limit)

	// Compute high friction step frequency
	r.computeHighFrictionStepFreq(ctx, workflowID, limit, &agg)

	return &agg, nil
}

func (r *PostgresRepository) computeTrendDirection(ctx context.Context, workflowID uuid.UUID, limit int) string {
	query := `
		SELECT overall_friction_score
		FROM ux_execution_metrics
		WHERE workflow_id = $1
		ORDER BY computed_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, workflowID, limit)
	if err != nil {
		return "stable"
	}
	defer rows.Close()

	var scores []float64
	for rows.Next() {
		var score float64
		if err := rows.Scan(&score); err == nil {
			scores = append(scores, score)
		}
	}

	if len(scores) < 3 {
		return "stable"
	}

	// Compare first half to second half
	mid := len(scores) / 2
	recentAvg := avg(scores[:mid])
	olderAvg := avg(scores[mid:])

	diff := recentAvg - olderAvg
	if diff < -5 {
		return "improving"
	} else if diff > 5 {
		return "degrading"
	}
	return "stable"
}

func (r *PostgresRepository) computeHighFrictionStepFreq(ctx context.Context, workflowID uuid.UUID, limit int, agg *contracts.WorkflowMetricsAggregate) {
	query := `
		SELECT step_metrics
		FROM ux_execution_metrics
		WHERE workflow_id = $1
		ORDER BY computed_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, workflowID, limit)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var stepMetricsJSON []byte
		if err := rows.Scan(&stepMetricsJSON); err != nil {
			continue
		}

		var stepMetrics []contracts.StepMetrics
		if err := json.Unmarshal(stepMetricsJSON, &stepMetrics); err != nil {
			continue
		}

		for _, sm := range stepMetrics {
			if sm.FrictionScore > 50 {
				agg.HighFrictionStepFreq[sm.StepIndex]++
			}
		}
	}
}

func avg(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// Compile-time interface check
var _ uxmetrics.Repository = (*PostgresRepository)(nil)
