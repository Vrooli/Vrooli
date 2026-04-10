// Package tags provides tag management for prompt categorization.
package tags

import (
	"database/sql"

	"github.com/google/uuid"
)

// Repository handles database operations for tags.
// This is a testing seam: inject a mock Repository in tests to avoid database access.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new tags repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// GetAll retrieves all tags ordered by name.
func (r *Repository) GetAll() ([]Tag, error) {
	query := `SELECT id, name, color, description FROM tags ORDER BY name`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Color, &tag.Description); err != nil {
			continue
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

	query := `INSERT INTO tags (id, name, color, description) VALUES ($1, $2, $3, $4)`
	_, err := r.db.Exec(query, tag.ID, tag.Name, tag.Color, tag.Description)
	return err
}
