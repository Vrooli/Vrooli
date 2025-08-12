package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Database connection
var db *sql.DB

// InitDatabase initializes the database connection
func InitDatabase(databaseURL string) error {
	var err error
	db, err = sql.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	
	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	
	// Note: Connection pool settings are configured in ApplyPerformanceOptimizations()
	
	return nil
}

// CloseDatabase closes the database connection
func CloseDatabase() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// Database operations for workflows

func listWorkflows(platform, typeFilter string, activeOnly bool, page, pageSize int) ([]Workflow, int, error) {
	offset := (page - 1) * pageSize
	
	// Build query
	query := `SELECT id, name, description, type, platform, config, webhook_path, job_path,
	          schema, estimated_duration_ms, version, parent_id, is_active, is_builtin,
	          tags, usage_count, success_count, failure_count, avg_execution_time_ms,
	          created_by, created_at, updated_at
	          FROM workflows WHERE 1=1`
	
	args := []interface{}{}
	argCount := 0
	
	if activeOnly {
		argCount++
		query += fmt.Sprintf(" AND is_active = $%d", argCount)
		args = append(args, true)
	}
	
	if platform != "" {
		argCount++
		query += fmt.Sprintf(" AND platform = $%d", argCount)
		args = append(args, platform)
	}
	
	if typeFilter != "" {
		argCount++
		query += fmt.Sprintf(" AND type = $%d", argCount)
		args = append(args, typeFilter)
	}
	
	// Get total count
	countQuery := strings.Replace(query, "SELECT id, name, description, type, platform, config, webhook_path, job_path, schema, estimated_duration_ms, version, parent_id, is_active, is_builtin, tags, usage_count, success_count, failure_count, avg_execution_time_ms, created_by, created_at, updated_at", "SELECT COUNT(*)", 1)
	
	var total int
	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	
	// Add pagination
	argCount++
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", argCount)
	args = append(args, pageSize)
	
	argCount++
	query += fmt.Sprintf(" OFFSET $%d", argCount)
	args = append(args, offset)
	
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	
	workflows := []Workflow{}
	for rows.Next() {
		var w Workflow
		var tags string
		
		err := rows.Scan(
			&w.ID, &w.Name, &w.Description, &w.Type, &w.Platform,
			&w.Config, &w.WebhookPath, &w.JobPath, &w.Schema,
			&w.EstimatedDuration, &w.Version, &w.ParentID,
			&w.IsActive, &w.IsBuiltin, &tags, &w.UsageCount,
			&w.SuccessCount, &w.FailureCount, &w.AvgExecutionTime,
			&w.CreatedBy, &w.CreatedAt, &w.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		
		// Parse tags
		w.Tags = parsePostgresTags(tags)
		workflows = append(workflows, w)
	}
	
	return workflows, total, nil
}

func getWorkflow(id uuid.UUID) (*Workflow, error) {
	var w Workflow
	var tags string
	
	query := `SELECT id, name, description, type, platform, config, webhook_path, job_path,
	          schema, estimated_duration_ms, version, parent_id, is_active, is_builtin,
	          tags, usage_count, success_count, failure_count, avg_execution_time_ms,
	          created_by, created_at, updated_at
	          FROM workflows WHERE id = $1`
	
	err := db.QueryRow(query, id).Scan(
		&w.ID, &w.Name, &w.Description, &w.Type, &w.Platform,
		&w.Config, &w.WebhookPath, &w.JobPath, &w.Schema,
		&w.EstimatedDuration, &w.Version, &w.ParentID,
		&w.IsActive, &w.IsBuiltin, &tags, &w.UsageCount,
		&w.SuccessCount, &w.FailureCount, &w.AvgExecutionTime,
		&w.CreatedBy, &w.CreatedAt, &w.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	
	w.Tags = parsePostgresTags(tags)
	return &w, nil
}

func createWorkflow(wc WorkflowCreate, createdBy string) (*Workflow, error) {
	id := uuid.New()
	now := time.Now()
	
	tagsArray := formatPostgresTags(wc.Tags)
	
	query := `INSERT INTO workflows (id, name, description, type, platform, config, 
	          webhook_path, job_path, schema, estimated_duration_ms, tags, 
	          created_by, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	          RETURNING id`
	
	err := db.QueryRow(
		query, id, wc.Name, wc.Description, wc.Type, wc.Platform,
		wc.Config, wc.WebhookPath, wc.JobPath, wc.Schema,
		wc.EstimatedDuration, tagsArray, createdBy, now, now,
	).Scan(&id)
	
	if err != nil {
		return nil, err
	}
	
	return getWorkflow(id)
}

func updateWorkflow(id uuid.UUID, wc WorkflowCreate, updatedBy string) (*Workflow, error) {
	// Get current workflow to create new version
	current, err := getWorkflow(id)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, fmt.Errorf("workflow not found")
	}
	
	// Start database transaction to prevent race conditions
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Will be ignored if transaction is committed
	
	// Deactivate current version within transaction
	_, err = tx.Exec("UPDATE workflows SET is_active = false WHERE id = $1", id)
	if err != nil {
		return nil, fmt.Errorf("failed to deactivate current workflow: %w", err)
	}
	
	// Create new version within same transaction
	newID := uuid.New()
	now := time.Now()
	newVersion := current.Version + 1
	tagsArray := formatPostgresTags(wc.Tags)
	
	query := `INSERT INTO workflows (id, name, description, type, platform, config,
	          webhook_path, job_path, schema, estimated_duration_ms, version,
	          parent_id, tags, created_by, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	          RETURNING id`
	
	err = tx.QueryRow(
		query, newID, wc.Name, wc.Description, wc.Type, wc.Platform,
		wc.Config, wc.WebhookPath, wc.JobPath, wc.Schema,
		wc.EstimatedDuration, newVersion, id, tagsArray, updatedBy, now, now,
	).Scan(&newID)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create new workflow version: %w", err)
	}
	
	// Commit transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return getWorkflow(newID)
}

func deleteWorkflow(id uuid.UUID) error {
	// Soft delete - just deactivate
	_, err := db.Exec("UPDATE workflows SET is_active = false WHERE id = $1", id)
	return err
}

func searchWorkflows(searchQuery string) ([]Workflow, error) {
	query := `SELECT id, name, description, type, platform, config, webhook_path, job_path,
	          schema, estimated_duration_ms, version, parent_id, is_active, is_builtin,
	          tags, usage_count, success_count, failure_count, avg_execution_time_ms,
	          created_by, created_at, updated_at
	          FROM workflows 
	          WHERE is_active = true 
	          AND (name ILIKE $1 OR description ILIKE $1 OR type ILIKE $1 OR $1 = ANY(tags))
	          ORDER BY usage_count DESC
	          LIMIT 20`
	
	searchPattern := "%" + searchQuery + "%"
	rows, err := db.Query(query, searchPattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	workflows := []Workflow{}
	for rows.Next() {
		var w Workflow
		var tags string
		
		err := rows.Scan(
			&w.ID, &w.Name, &w.Description, &w.Type, &w.Platform,
			&w.Config, &w.WebhookPath, &w.JobPath, &w.Schema,
			&w.EstimatedDuration, &w.Version, &w.ParentID,
			&w.IsActive, &w.IsBuiltin, &tags, &w.UsageCount,
			&w.SuccessCount, &w.FailureCount, &w.AvgExecutionTime,
			&w.CreatedBy, &w.CreatedAt, &w.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		
		w.Tags = parsePostgresTags(tags)
		workflows = append(workflows, w)
	}
	
	return workflows, nil
}

func cloneWorkflow(id uuid.UUID, newName string) (*Workflow, error) {
	// Get source workflow
	source, err := getWorkflow(id)
	if err != nil {
		return nil, err
	}
	if source == nil {
		return nil, fmt.Errorf("workflow not found")
	}
	
	// Create clone
	wc := WorkflowCreate{
		Name:              newName,
		Description:       source.Description + " (cloned)",
		Type:              source.Type,
		Platform:          source.Platform,
		Config:            source.Config,
		WebhookPath:       source.WebhookPath,
		JobPath:           source.JobPath,
		Schema:            source.Schema,
		EstimatedDuration: source.EstimatedDuration,
		Tags:              append(source.Tags, "cloned"),
	}
	
	return createWorkflow(wc, "system")
}

// Execution history operations

func recordExecution(workflowID uuid.UUID, req ExecutionRequest, resp *ExecutionResponse) error {
	inputJSON, _ := json.Marshal(req)
	outputJSON, _ := json.Marshal(resp.Data)
	metadataJSON, _ := json.Marshal(req.Metadata)
	
	query := `INSERT INTO execution_history (workflow_id, workflow_type, resource_type, 
	          resource_id, input_data, output_data, execution_time_ms, status, 
	          model_used, error_message, metadata)
	          VALUES ($1, (SELECT type FROM workflows WHERE id = $1), 
	                  (SELECT platform FROM workflows WHERE id = $1),
	                  $2, $3, $4, $5, $6, $7, $8, $9)`
	
	_, err := db.Exec(
		query, workflowID, resp.ID, inputJSON, outputJSON,
		resp.ExecutionMS, resp.Status, req.Model, resp.Error, metadataJSON,
	)
	
	return err
}

func getExecutionHistory(workflowID uuid.UUID, limit int) ([]ExecutionHistory, error) {
	query := `SELECT id, status, execution_time_ms, model_used, created_at,
	          CASE 
	            WHEN LENGTH(input_data::text) > 50 
	            THEN SUBSTRING(input_data::text, 1, 50) || '...'
	            ELSE input_data::text 
	          END as input_summary,
	          output_data IS NOT NULL as has_output
	          FROM execution_history
	          WHERE workflow_id = $1
	          ORDER BY created_at DESC
	          LIMIT $2`
	
	rows, err := db.Query(query, workflowID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	history := []ExecutionHistory{}
	for rows.Next() {
		var h ExecutionHistory
		err := rows.Scan(&h.ID, &h.Status, &h.ExecutionTime, &h.ModelUsed,
			&h.CreatedAt, &h.InputSummary, &h.HasOutput)
		if err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	
	return history, nil
}

func getWorkflowMetrics(workflowID uuid.UUID) (*MetricsResponse, error) {
	query := `SELECT 
	          COUNT(*) as total,
	          COUNT(CASE WHEN status = 'success' THEN 1 END) as success,
	          COUNT(CASE WHEN status = 'error' THEN 1 END) as failure,
	          AVG(execution_time_ms)::int as avg_time,
	          MIN(execution_time_ms) as min_time,
	          MAX(execution_time_ms) as max_time,
	          MAX(created_at) as last_exec,
	          array_agg(DISTINCT model_used) FILTER (WHERE model_used IS NOT NULL) as models
	          FROM execution_history
	          WHERE workflow_id = $1`
	
	var metrics MetricsResponse
	var lastExec sql.NullTime
	var modelsArray sql.NullString
	
	err := db.QueryRow(query, workflowID).Scan(
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
		return nil, err
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
	
	return &metrics, nil
}

func getSystemStats() (*StatsResponse, error) {
	var stats StatsResponse
	
	// Get workflow counts
	err := db.QueryRow(`
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN is_active THEN 1 END) as active
		FROM workflows
	`).Scan(&stats.TotalWorkflows, &stats.ActiveWorkflows)
	if err != nil {
		return nil, err
	}
	
	// Get execution stats
	var lastExec sql.NullTime
	var mostUsedWorkflow sql.NullString
	var mostUsedModel sql.NullString
	
	err = db.QueryRow(`
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'success' THEN 1 END) as success,
			COUNT(CASE WHEN status = 'error' THEN 1 END) as failure,
			AVG(execution_time_ms)::int as avg_time,
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
		return nil, err
	}
	
	if lastExec.Valid {
		stats.LastExecution = &lastExec.Time
	}
	
	// Get most used workflow
	err = db.QueryRow(`
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
	err = db.QueryRow(`
		SELECT model_used
		FROM execution_history
		WHERE model_used IS NOT NULL
		GROUP BY model_used
		ORDER BY COUNT(*) DESC
		LIMIT 1
	`).Scan(&mostUsedModel)
	if err == nil && mostUsedModel.Valid {
		stats.MostUsedModel = mostUsedModel.String
	}
	
	return &stats, nil
}

// Helper functions for PostgreSQL arrays

func parsePostgresTags(tags string) []string {
	// Handle PostgreSQL array format: {tag1,tag2,tag3}
	tags = strings.TrimPrefix(tags, "{")
	tags = strings.TrimSuffix(tags, "}")
	if tags == "" {
		return []string{}
	}
	return strings.Split(tags, ",")
}

func formatPostgresTags(tags []string) string {
	if len(tags) == 0 {
		return "{}"
	}
	return "{" + strings.Join(tags, ",") + "}"
}