package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"metareasoning-api/internal/domain/workflow"
)

// PostgresWorkflowRepository implements the workflow.Repository interface using PostgreSQL
type PostgresWorkflowRepository struct {
	db *sql.DB
	tx *sql.Tx
}

// NewPostgresWorkflowRepository creates a new PostgreSQL workflow repository
func NewPostgresWorkflowRepository(db *sql.DB) workflow.Repository {
	return &PostgresWorkflowRepository{
		db: db,
	}
}

// Create inserts a new workflow into the database
func (r *PostgresWorkflowRepository) Create(entity *workflow.Entity) error {
	if entity.ID == uuid.Nil {
		entity.ID = uuid.New()
	}
	
	now := time.Now()
	entity.CreatedAt = now
	entity.UpdatedAt = now
	
	if entity.Version == 0 {
		entity.Version = 1
	}
	
	if entity.IsActive == nil {
		active := true
		entity.IsActive = &active
	}
	
	if entity.IsBuiltin == nil {
		builtin := false
		entity.IsBuiltin = &builtin
	}
	
	conn := r.getConnection()
	
	query := `INSERT INTO workflows (
		id, name, description, type, platform, config, webhook_path, job_path,
		schema, estimated_duration_ms, version, parent_id, is_active, is_builtin,
		tags, created_by, created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`
	
	tagsArray := formatPostgresTags(entity.Tags)
	
	_, err := conn.Exec(
		query, entity.ID, entity.Name, entity.Description, entity.Type,
		entity.Platform, entity.Config, entity.WebhookPath, entity.JobPath,
		entity.Schema, entity.EstimatedDuration, entity.Version, entity.ParentID,
		entity.IsActive, entity.IsBuiltin, tagsArray, entity.CreatedBy,
		entity.CreatedAt, entity.UpdatedAt,
	)
	
	return err
}

// GetByID retrieves a workflow by its ID
func (r *PostgresWorkflowRepository) GetByID(id uuid.UUID) (*workflow.Entity, error) {
	conn := r.getConnection()
	
	query := `SELECT id, name, description, type, platform, config, webhook_path, job_path,
		schema, estimated_duration_ms, version, parent_id, is_active, is_builtin,
		tags, usage_count, success_count, failure_count, avg_execution_time_ms,
		created_by, created_at, updated_at
		FROM workflows WHERE id = $1`
	
	entity := &workflow.Entity{}
	var tags string
	
	err := conn.QueryRow(query, id).Scan(
		&entity.ID, &entity.Name, &entity.Description, &entity.Type,
		&entity.Platform, &entity.Config, &entity.WebhookPath, &entity.JobPath,
		&entity.Schema, &entity.EstimatedDuration, &entity.Version, &entity.ParentID,
		&entity.IsActive, &entity.IsBuiltin, &tags, &entity.UsageCount,
		&entity.SuccessCount, &entity.FailureCount, &entity.AvgExecutionTime,
		&entity.CreatedBy, &entity.CreatedAt, &entity.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}
	
	entity.Tags = parsePostgresTags(tags)
	return entity, nil
}

// Update modifies an existing workflow by creating a new version
func (r *PostgresWorkflowRepository) Update(id uuid.UUID, entity *workflow.Entity) error {
	// Get current workflow
	current, err := r.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get current workflow: %w", err)
	}
	if current == nil {
		return fmt.Errorf("workflow not found")
	}
	
	// Set up new version
	entity.ID = uuid.New()
	entity.Version = current.Version + 1
	entity.ParentID = &id
	entity.UpdatedAt = time.Now()
	
	return r.WithTransaction(func(repo workflow.Repository) error {
		txRepo := repo.(*PostgresWorkflowRepository)
		
		// Deactivate current version
		_, err := txRepo.getConnection().Exec("UPDATE workflows SET is_active = false WHERE id = $1", id)
		if err != nil {
			return fmt.Errorf("failed to deactivate current workflow: %w", err)
		}
		
		// Create new version
		return txRepo.Create(entity)
	})
}

// Delete soft-deletes a workflow by setting is_active to false
func (r *PostgresWorkflowRepository) Delete(id uuid.UUID) error {
	conn := r.getConnection()
	
	result, err := conn.Exec("UPDATE workflows SET is_active = false WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete workflow: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("workflow not found")
	}
	
	return nil
}

// List retrieves workflows with filtering and pagination
func (r *PostgresWorkflowRepository) List(query *workflow.RepositoryQuery) (*workflow.ListResponse, error) {
	conn := r.getConnection()
	
	// Build base query
	baseSQL := `FROM workflows WHERE 1=1`
	args := []interface{}{}
	argCount := 0
	
	// Apply filters
	if query.ActiveOnly {
		argCount++
		baseSQL += fmt.Sprintf(" AND is_active = $%d", argCount)
		args = append(args, true)
	}
	
	if query.Platform != "" {
		argCount++
		baseSQL += fmt.Sprintf(" AND platform = $%d", argCount)
		args = append(args, query.Platform)
	}
	
	if query.Type != "" {
		argCount++
		baseSQL += fmt.Sprintf(" AND type = $%d", argCount)
		args = append(args, query.Type)
	}
	
	if query.CreatedBy != "" {
		argCount++
		baseSQL += fmt.Sprintf(" AND created_by = $%d", argCount)
		args = append(args, query.CreatedBy)
	}
	
	if len(query.Tags) > 0 {
		argCount++
		baseSQL += fmt.Sprintf(" AND tags && $%d", argCount)
		args = append(args, formatPostgresTags(query.Tags))
	}
	
	// Get total count
	countQuery := "SELECT COUNT(*) " + baseSQL
	var total int
	err := conn.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}
	
	// Set pagination defaults
	page := 1
	pageSize := 20
	if query.Pagination != nil {
		if query.Pagination.Page > 0 {
			page = query.Pagination.Page
		}
		if query.Pagination.PageSize > 0 {
			pageSize = query.Pagination.PageSize
		}
	}
	
	// Build select query with pagination
	selectSQL := `SELECT id, name, description, type, platform, config, webhook_path, job_path,
		schema, estimated_duration_ms, version, parent_id, is_active, is_builtin,
		tags, usage_count, success_count, failure_count, avg_execution_time_ms,
		created_by, created_at, updated_at ` + baseSQL
	
	// Add sorting
	sortBy := "created_at"
	sortDir := "DESC"
	if query.Pagination != nil {
		if query.Pagination.SortBy != "" {
			sortBy = query.Pagination.SortBy
		}
		if query.Pagination.SortDir != "" {
			sortDir = query.Pagination.SortDir
		}
	}
	
	selectSQL += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortDir)
	
	// Add pagination
	offset := (page - 1) * pageSize
	argCount++
	selectSQL += fmt.Sprintf(" LIMIT $%d", argCount)
	args = append(args, pageSize)
	
	argCount++
	selectSQL += fmt.Sprintf(" OFFSET $%d", argCount)
	args = append(args, offset)
	
	// Execute query
	rows, err := conn.Query(selectSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}
	defer rows.Close()
	
	workflows := []*workflow.Entity{}
	for rows.Next() {
		entity := &workflow.Entity{}
		var tags string
		
		err := rows.Scan(
			&entity.ID, &entity.Name, &entity.Description, &entity.Type,
			&entity.Platform, &entity.Config, &entity.WebhookPath, &entity.JobPath,
			&entity.Schema, &entity.EstimatedDuration, &entity.Version, &entity.ParentID,
			&entity.IsActive, &entity.IsBuiltin, &tags, &entity.UsageCount,
			&entity.SuccessCount, &entity.FailureCount, &entity.AvgExecutionTime,
			&entity.CreatedBy, &entity.CreatedAt, &entity.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workflow: %w", err)
		}
		
		entity.Tags = parsePostgresTags(tags)
		workflows = append(workflows, entity)
	}
	
	return &workflow.ListResponse{
		Workflows: workflows,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
		HasNext:   offset+pageSize < total,
	}, nil
}

// Search performs a text search across workflow names, descriptions, and tags
func (r *PostgresWorkflowRepository) Search(searchQuery string, limit int) ([]*workflow.Entity, error) {
	conn := r.getConnection()
	
	query := `SELECT id, name, description, type, platform, config, webhook_path, job_path,
		schema, estimated_duration_ms, version, parent_id, is_active, is_builtin,
		tags, usage_count, success_count, failure_count, avg_execution_time_ms,
		created_by, created_at, updated_at
		FROM workflows 
		WHERE is_active = true 
		AND (name ILIKE $1 OR description ILIKE $1 OR type ILIKE $1 OR $1 = ANY(tags))
		ORDER BY usage_count DESC
		LIMIT $2`
	
	searchPattern := "%" + searchQuery + "%"
	rows, err := conn.Query(query, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search workflows: %w", err)
	}
	defer rows.Close()
	
	workflows := []*workflow.Entity{}
	for rows.Next() {
		entity := &workflow.Entity{}
		var tags string
		
		err := rows.Scan(
			&entity.ID, &entity.Name, &entity.Description, &entity.Type,
			&entity.Platform, &entity.Config, &entity.WebhookPath, &entity.JobPath,
			&entity.Schema, &entity.EstimatedDuration, &entity.Version, &entity.ParentID,
			&entity.IsActive, &entity.IsBuiltin, &tags, &entity.UsageCount,
			&entity.SuccessCount, &entity.FailureCount, &entity.AvgExecutionTime,
			&entity.CreatedBy, &entity.CreatedAt, &entity.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workflow: %w", err)
		}
		
		entity.Tags = parsePostgresTags(tags)
		workflows = append(workflows, entity)
	}
	
	return workflows, nil
}

// Clone creates a copy of an existing workflow
func (r *PostgresWorkflowRepository) Clone(id uuid.UUID, newName string, clonedBy string) (*workflow.Entity, error) {
	// Get source workflow
	source, err := r.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get source workflow: %w", err)
	}
	if source == nil {
		return nil, fmt.Errorf("workflow not found")
	}
	
	// Create clone
	clone := &workflow.Entity{
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
		CreatedBy:         clonedBy,
	}
	
	err = r.Create(clone)
	if err != nil {
		return nil, fmt.Errorf("failed to create clone: %w", err)
	}
	
	return clone, nil
}

// GetVersions retrieves all versions of a workflow
func (r *PostgresWorkflowRepository) GetVersions(parentID uuid.UUID) ([]*workflow.Entity, error) {
	conn := r.getConnection()
	
	query := `SELECT id, name, description, type, platform, config, webhook_path, job_path,
		schema, estimated_duration_ms, version, parent_id, is_active, is_builtin,
		tags, usage_count, success_count, failure_count, avg_execution_time_ms,
		created_by, created_at, updated_at
		FROM workflows 
		WHERE id = $1 OR parent_id = $1
		ORDER BY version DESC`
	
	rows, err := conn.Query(query, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow versions: %w", err)
	}
	defer rows.Close()
	
	workflows := []*workflow.Entity{}
	for rows.Next() {
		entity := &workflow.Entity{}
		var tags string
		
		err := rows.Scan(
			&entity.ID, &entity.Name, &entity.Description, &entity.Type,
			&entity.Platform, &entity.Config, &entity.WebhookPath, &entity.JobPath,
			&entity.Schema, &entity.EstimatedDuration, &entity.Version, &entity.ParentID,
			&entity.IsActive, &entity.IsBuiltin, &tags, &entity.UsageCount,
			&entity.SuccessCount, &entity.FailureCount, &entity.AvgExecutionTime,
			&entity.CreatedBy, &entity.CreatedAt, &entity.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workflow: %w", err)
		}
		
		entity.Tags = parsePostgresTags(tags)
		workflows = append(workflows, entity)
	}
	
	return workflows, nil
}

// WithTransaction executes a function within a database transaction
func (r *PostgresWorkflowRepository) WithTransaction(fn func(workflow.Repository) error) error {
	if r.tx != nil {
		// Already in a transaction, just call the function
		return fn(r)
	}
	
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	txRepo := &PostgresWorkflowRepository{
		db: r.db,
		tx: tx,
	}
	
	if err := fn(txRepo); err != nil {
		return err
	}
	
	return tx.Commit()
}

// getConnection returns the appropriate connection (transaction or database)
func (r *PostgresWorkflowRepository) getConnection() dbExecutor {
	if r.tx != nil {
		return r.tx
	}
	return r.db
}

// dbExecutor interface for database operations that can be performed on both *sql.DB and *sql.Tx
type dbExecutor interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
}

// Helper functions for PostgreSQL arrays
func parsePostgresTags(tags string) []string {
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