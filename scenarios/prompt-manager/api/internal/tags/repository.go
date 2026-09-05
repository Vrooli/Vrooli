// Package tags provides tag management for prompt categorization.
package tags

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

// Repository handles database operations for tags.
// This is a testing seam: inject a mock Repository in tests to avoid database access.
type Repository struct {
	db  databaseExecutor
	ctx context.Context
}

type databaseExecutor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// NewRepository creates a new tags repository.
func NewRepository(db databaseExecutor) *Repository {
	return &Repository{db: db, ctx: context.Background()}
}

func (r *Repository) WithContext(ctx context.Context) *Repository {
	copy := *r
	copy.ctx = ctx
	return &copy
}

func (r *Repository) WithRequestContext(ctx context.Context) TagRepository {
	return r.WithContext(ctx)
}

// GetAll retrieves all tags ordered by name.
func (r *Repository) GetAll() ([]Tag, error) {
	query := `SELECT id, name, color, description FROM tags ORDER BY name`

	rows, err := r.db.QueryContext(r.ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var (
			tag         Tag
			color       sql.NullString
			description sql.NullString
		)
		if err := rows.Scan(&tag.ID, &tag.Name, &color, &description); err != nil {
			continue
		}
		if color.Valid {
			tag.Color = &color.String
		}
		if description.Valid {
			tag.Description = &description.String
		}
		tags = append(tags, tag)
	}

	return tags, nil
}

// Create adds a new tag.
func (r *Repository) Create(tag *Tag) error {
	if tag.ID == "" {
		tag.ID = uuid.New().String()
	}

	query := `INSERT INTO tags (id, name, color, description) VALUES (?, ?, ?, ?)`
	_, err := r.db.ExecContext(r.ctx, query, tag.ID, tag.Name, tag.Color, tag.Description)
	return err
}
