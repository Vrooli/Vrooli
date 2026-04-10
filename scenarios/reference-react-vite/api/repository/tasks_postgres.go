package repository

import (
	"context"
	"database/sql"
	"fmt"

	"reference-react-vite/api/domain/tasks"
)

// PostgresTaskRepository implements TaskRepository using PostgreSQL.
type PostgresTaskRepository struct {
	db *sql.DB
}

// NewPostgresTaskRepository creates a new PostgreSQL task repository.
func NewPostgresTaskRepository(db *sql.DB) *PostgresTaskRepository {
	return &PostgresTaskRepository{db: db}
}

// Create inserts a new task into the database.
func (r *PostgresTaskRepository) Create(ctx context.Context, task *tasks.Task) error {
	query := `
		INSERT INTO tasks (id, project_id, title, description, status, priority, due_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	var projectID *string
	if task.ProjectID != "" {
		projectID = &task.ProjectID
	}
	_, err := r.db.ExecContext(ctx, query,
		task.ID, projectID, task.Title, task.Description,
		string(task.Status), int(task.Priority), task.DueDate,
		task.CreatedAt, task.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	return nil
}

// FindByID retrieves a task by its ID.
func (r *PostgresTaskRepository) FindByID(ctx context.Context, id string) (*tasks.Task, error) {
	query := `
		SELECT id, project_id, title, description, status, priority, due_date, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`
	task := &tasks.Task{}
	var projectID sql.NullString
	var dueDate sql.NullTime
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID, &projectID, &task.Title, &task.Description,
		&task.Status, &task.Priority, &dueDate,
		&task.CreatedAt, &task.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find task by id: %w", err)
	}
	if projectID.Valid {
		task.ProjectID = projectID.String
	}
	if dueDate.Valid {
		task.DueDate = &dueDate.Time
	}
	return task, nil
}

// List retrieves tasks matching the filter with pagination.
func (r *PostgresTaskRepository) List(ctx context.Context, filter tasks.ListFilter) ([]*tasks.Task, int, error) {
	qb := newQueryBuilder()

	// Build WHERE conditions using safe column names (not user input)
	if filter.ProjectID != nil {
		qb.addCondition("project_id", *filter.ProjectID)
	}
	if filter.Status != nil {
		qb.addCondition("status", string(*filter.Status))
	}
	if filter.Priority != nil {
		qb.addCondition("priority", int(*filter.Priority))
	}

	whereClause := qb.whereClause()

	// Count total matching records
	countQuery := "SELECT COUNT(*) FROM tasks " + whereClause
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, qb.getArgsForCount()...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tasks: %w", err)
	}

	// Apply pagination defaults
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	// Build paginated query with parameterized limit/offset
	limitParam, offsetParam := qb.addLimitOffset(limit, offset)
	query := `
		SELECT id, project_id, title, description, status, priority, due_date, created_at, updated_at
		FROM tasks
		` + whereClause + `
		ORDER BY created_at DESC
		LIMIT ` + limitParam + ` OFFSET ` + offsetParam

	rows, err := r.db.QueryContext(ctx, query, qb.getArgs()...)
	if err != nil {
		return nil, 0, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var result []*tasks.Task
	for rows.Next() {
		task := &tasks.Task{}
		var projectID sql.NullString
		var dueDate sql.NullTime
		if err := rows.Scan(
			&task.ID, &projectID, &task.Title, &task.Description,
			&task.Status, &task.Priority, &dueDate,
			&task.CreatedAt, &task.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan task: %w", err)
		}
		if projectID.Valid {
			task.ProjectID = projectID.String
		}
		if dueDate.Valid {
			task.DueDate = &dueDate.Time
		}
		result = append(result, task)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate tasks: %w", err)
	}

	return result, total, nil
}

// Update saves changes to an existing task.
func (r *PostgresTaskRepository) Update(ctx context.Context, task *tasks.Task) error {
	query := `
		UPDATE tasks
		SET project_id = $2, title = $3, description = $4, status = $5, priority = $6, due_date = $7, updated_at = $8
		WHERE id = $1
	`
	var projectID *string
	if task.ProjectID != "" {
		projectID = &task.ProjectID
	}
	result, err := r.db.ExecContext(ctx, query,
		task.ID, projectID, task.Title, task.Description,
		string(task.Status), int(task.Priority), task.DueDate,
		task.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check update result: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("update task: %w", ErrNotFound)
	}
	return nil
}

// Delete removes a task by ID.
func (r *PostgresTaskRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM tasks WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check delete result: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("delete task: %w", ErrNotFound)
	}
	return nil
}
