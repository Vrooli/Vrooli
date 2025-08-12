package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"metareasoning-api/internal/domain/common"
	"metareasoning-api/internal/domain/workflow"
)

// PostgresExecutionRepository implements the workflow.ExecutionRepository interface using PostgreSQL
type PostgresExecutionRepository struct {
	db *sql.DB
}

// NewPostgresExecutionRepository creates a new PostgreSQL execution repository
func NewPostgresExecutionRepository(db *sql.DB) workflow.ExecutionRepository {
	return &PostgresExecutionRepository{
		db: db,
	}
}

// RecordExecution records a workflow execution in the database
func (r *PostgresExecutionRepository) RecordExecution(workflowID uuid.UUID, req *common.ExecutionRequest, resp *common.ExecutionResponse) error {
	inputJSON, err := json.Marshal(req.Input)
	if err != nil {
		inputJSON = []byte("{}")
	}
	
	outputJSON, err := json.Marshal(resp.Data)
	if err != nil {
		outputJSON = []byte("{}")
	}
	
	metadataJSON, err := json.Marshal(req.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}
	
	query := `INSERT INTO execution_history (
		id, workflow_id, workflow_type, resource_type, resource_id,
		input_data, output_data, execution_time_ms, status,
		model_used, error_message, metadata, created_at
	) VALUES (
		$1, $2, 
		(SELECT type FROM workflows WHERE id = $2), 
		(SELECT platform FROM workflows WHERE id = $2),
		$3, $4, $5, $6, $7, $8, $9, $10, $11
	)`
	
	_, err = r.db.Exec(
		query, resp.ID, workflowID, resp.ID, inputJSON, outputJSON,
		resp.ExecutionMS, resp.Status, req.Model, resp.Error, 
		metadataJSON, resp.Timestamp,
	)
	
	if err != nil {
		return fmt.Errorf("failed to record execution: %w", err)
	}
	
	return nil
}

// GetExecutionHistory retrieves execution history for a workflow
func (r *PostgresExecutionRepository) GetExecutionHistory(workflowID uuid.UUID, limit int) ([]*workflow.ExecutionHistory, error) {
	query := `SELECT id, workflow_id, status, execution_time_ms, model_used, created_at,
		CASE 
			WHEN LENGTH(input_data::text) > 50 
			THEN SUBSTRING(input_data::text, 1, 50) || '...'
			ELSE input_data::text 
		END as input_summary,
		output_data IS NOT NULL AND output_data != '{}' as has_output
		FROM execution_history
		WHERE workflow_id = $1
		ORDER BY created_at DESC
		LIMIT $2`
	
	rows, err := r.db.Query(query, workflowID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution history: %w", err)
	}
	defer rows.Close()
	
	history := []*workflow.ExecutionHistory{}
	for rows.Next() {
		h := &workflow.ExecutionHistory{}
		err := rows.Scan(&h.ID, &h.WorkflowID, &h.Status, &h.ExecutionTime, 
			&h.ModelUsed, &h.CreatedAt, &h.InputSummary, &h.HasOutput)
		if err != nil {
			return nil, fmt.Errorf("failed to scan execution history: %w", err)
		}
		history = append(history, h)
	}
	
	return history, nil
}

// GetMetrics retrieves execution metrics for a workflow
func (r *PostgresExecutionRepository) GetMetrics(workflowID uuid.UUID) (*workflow.MetricsResponse, error) {
	query := `SELECT 
		COUNT(*) as total,
		COUNT(CASE WHEN status = 'success' THEN 1 END) as success,
		COUNT(CASE WHEN status = 'failed' THEN 1 END) as failure,
		COALESCE(AVG(execution_time_ms)::int, 0) as avg_time,
		COALESCE(MIN(execution_time_ms), 0) as min_time,
		COALESCE(MAX(execution_time_ms), 0) as max_time,
		MAX(created_at) as last_exec,
		array_agg(DISTINCT model_used) FILTER (WHERE model_used IS NOT NULL AND model_used != '') as models
		FROM execution_history
		WHERE workflow_id = $1`
	
	metrics := &workflow.MetricsResponse{}
	var lastExec sql.NullTime
	var modelsArray sql.NullString
	
	err := r.db.QueryRow(query, workflowID).Scan(
		&metrics.TotalExecutions,
		&metrics.SuccessCount,
		&metrics.FailureCount,
		&metrics.AvgExecutionTime,
		&metrics.MinExecutionTime,
		&metrics.MaxExecutionTime,
		&lastExec,
		&modelsArray,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}
	
	if lastExec.Valid {
		metrics.LastExecution = &lastExec.Time
	}
	
	if modelsArray.Valid {
		// Parse PostgreSQL array format
		models := parsePostgresTags(modelsArray.String)
		metrics.ModelsUsed = models
	}
	
	if metrics.TotalExecutions > 0 {
		metrics.SuccessRate = float64(metrics.SuccessCount) / float64(metrics.TotalExecutions) * 100
	}
	
	return metrics, nil
}

// GetSystemStats retrieves system-wide execution statistics
func (r *PostgresExecutionRepository) GetSystemStats() (*workflow.StatsResponse, error) {
	stats := &workflow.StatsResponse{}
	
	// Get workflow counts
	err := r.db.QueryRow(`
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN is_active THEN 1 END) as active
		FROM workflows
	`).Scan(&stats.TotalWorkflows, &stats.ActiveWorkflows)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow stats: %w", err)
	}
	
	// Get execution stats
	var lastExec sql.NullTime
	var mostUsedWorkflow sql.NullString
	var mostUsedModel sql.NullString
	
	err = r.db.QueryRow(`
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'success' THEN 1 END) as success,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failure,
			COALESCE(AVG(execution_time_ms)::int, 0) as avg_time,
			MAX(created_at) as last_exec
		FROM execution_history
	`).Scan(
		&stats.TotalExecutions,
		&stats.SuccessfulExecs,
		&stats.FailedExecs,
		&stats.AvgExecutionTime,
		&lastExec,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution stats: %w", err)
	}
	
	if lastExec.Valid {
		stats.LastExecution = &lastExec.Time
	}
	
	// Get most used workflow
	err = r.db.QueryRow(`
		SELECT w.name
		FROM workflows w
		JOIN execution_history e ON w.id = e.workflow_id
		GROUP BY w.name
		ORDER BY COUNT(*) DESC
		LIMIT 1
	`).Scan(&mostUsedWorkflow)
	if err == nil && mostUsedWorkflow.Valid {
		stats.MostUsedWorkflow = mostUsedWorkflow.String
	}
	
	// Get most used model
	err = r.db.QueryRow(`
		SELECT model_used
		FROM execution_history
		WHERE model_used IS NOT NULL AND model_used != ''
		GROUP BY model_used
		ORDER BY COUNT(*) DESC
		LIMIT 1
	`).Scan(&mostUsedModel)
	if err == nil && mostUsedModel.Valid {
		stats.MostUsedModel = mostUsedModel.String
	}
	
	return stats, nil
}

// CleanupOldExecutions removes execution records older than the specified time
func (r *PostgresExecutionRepository) CleanupOldExecutions(olderThan time.Time) error {
	query := `DELETE FROM execution_history WHERE created_at < $1`
	
	result, err := r.db.Exec(query, olderThan)
	if err != nil {
		return fmt.Errorf("failed to cleanup old executions: %w", err)
	}
	
	rowsDeleted, _ := result.RowsAffected()
	if rowsDeleted > 0 {
		// Log cleanup operation (could be done via injected logger)
		fmt.Printf("Cleaned up %d old execution records\n", rowsDeleted)
	}
	
	return nil
}