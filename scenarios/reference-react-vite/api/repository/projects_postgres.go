package repository

import (
	"context"
	"database/sql"
	"fmt"

	"reference-react-vite/api/domain/projects"
)

// PostgresProjectRepository implements ProjectRepository using PostgreSQL.
type PostgresProjectRepository struct {
	db *sql.DB
}

// NewPostgresProjectRepository creates a new PostgreSQL project repository.
func NewPostgresProjectRepository(db *sql.DB) *PostgresProjectRepository {
	return &PostgresProjectRepository{db: db}
}

// Create inserts a new project into the database.
func (r *PostgresProjectRepository) Create(ctx context.Context, project *projects.Project) error {
	query := `
		INSERT INTO projects (id, name, description, status, color, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	var color *string
	if project.Color != "" {
		color = &project.Color
	}
	_, err := r.db.ExecContext(ctx, query,
		project.ID, project.Name, project.Description,
		string(project.Status), color,
		project.CreatedAt, project.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	return nil
}

// FindByID retrieves a project by its ID, including task count.
func (r *PostgresProjectRepository) FindByID(ctx context.Context, id string) (*projects.Project, error) {
	query := `
		SELECT p.id, p.name, p.description, p.status, p.color, p.created_at, p.updated_at,
		       COALESCE((SELECT COUNT(*) FROM tasks t WHERE t.project_id = p.id), 0) as task_count
		FROM projects p
		WHERE p.id = $1
	`
	project := &projects.Project{}
	var color sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&project.ID, &project.Name, &project.Description,
		&project.Status, &color,
		&project.CreatedAt, &project.UpdatedAt,
		&project.TaskCount,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find project by id: %w", err)
	}
	if color.Valid {
		project.Color = color.String
	}
	return project, nil
}

// List retrieves projects matching the filter with pagination.
func (r *PostgresProjectRepository) List(ctx context.Context, filter projects.ListFilter) ([]*projects.Project, int, error) {
	qb := newQueryBuilder()

	// Build WHERE conditions using safe column names (not user input)
	if filter.Status != nil {
		qb.addCondition("status", string(*filter.Status))
	}

	whereClause := qb.whereClause()

	// Count total matching records
	countQuery := "SELECT COUNT(*) FROM projects " + whereClause
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, qb.getArgsForCount()...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count projects: %w", err)
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
		SELECT p.id, p.name, p.description, p.status, p.color, p.created_at, p.updated_at,
		       COALESCE((SELECT COUNT(*) FROM tasks t WHERE t.project_id = p.id), 0) as task_count
		FROM projects p
		` + whereClause + `
		ORDER BY p.created_at DESC
		LIMIT ` + limitParam + ` OFFSET ` + offsetParam

	rows, err := r.db.QueryContext(ctx, query, qb.getArgs()...)
	if err != nil {
		return nil, 0, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var result []*projects.Project
	for rows.Next() {
		project := &projects.Project{}
		var color sql.NullString
		if err := rows.Scan(
			&project.ID, &project.Name, &project.Description,
			&project.Status, &color,
			&project.CreatedAt, &project.UpdatedAt,
			&project.TaskCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan project: %w", err)
		}
		if color.Valid {
			project.Color = color.String
		}
		result = append(result, project)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate projects: %w", err)
	}

	return result, total, nil
}

// Update saves changes to an existing project.
func (r *PostgresProjectRepository) Update(ctx context.Context, project *projects.Project) error {
	query := `
		UPDATE projects
		SET name = $2, description = $3, status = $4, color = $5, updated_at = $6
		WHERE id = $1
	`
	var color *string
	if project.Color != "" {
		color = &project.Color
	}
	result, err := r.db.ExecContext(ctx, query,
		project.ID, project.Name, project.Description,
		string(project.Status), color,
		project.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update project: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check update result: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("update project: %w", ErrNotFound)
	}
	return nil
}

// Delete removes a project by ID.
func (r *PostgresProjectRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM projects WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check delete result: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("delete project: %w", ErrNotFound)
	}
	return nil
}
