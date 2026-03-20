// DOC: docs/internal/SEAMS.md#api--database
// DOC: docs/reference/api-endpoints.md#schemes
package main

import "database/sql"

// SchemeService manages scheme CRUD operations
type SchemeService struct {
	db *sql.DB
}

// NewSchemeService creates a new SchemeService
func NewSchemeService(db *sql.DB) *SchemeService {
	return &SchemeService{db: db}
}

// Create creates a new scheme
func (s *SchemeService) Create(input *CreateSchemeInput) (*Scheme, error) {
	name := input.Name
	if name == "" {
		name = "Untitled"
	}
	scheme, err := scanScheme(s.db.QueryRow(
		`INSERT INTO schemes (name) VALUES ($1)
		 RETURNING id, name, created_at, updated_at`,
		name,
	))
	if err != nil {
		return nil, err
	}
	return &scheme, nil
}

// List returns all schemes ordered by updated_at desc
func (s *SchemeService) List() ([]Scheme, error) {
	rows, err := s.db.Query(
		`SELECT id, name, created_at, updated_at FROM schemes ORDER BY updated_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanScheme)
}

// GetByID returns a scheme by ID
func (s *SchemeService) GetByID(id string) (*Scheme, error) {
	scheme, err := scanScheme(s.db.QueryRow(
		`SELECT id, name, created_at, updated_at FROM schemes WHERE id = $1`, id,
	))
	if err != nil {
		return nil, err
	}
	return &scheme, nil
}

// Update updates a scheme
func (s *SchemeService) Update(id string, input *UpdateSchemeInput) (*Scheme, error) {
	scheme, err := scanScheme(s.db.QueryRow(
		`UPDATE schemes SET name = $1, updated_at = NOW() WHERE id = $2
		 RETURNING id, name, created_at, updated_at`,
		input.Name, id,
	))
	if err != nil {
		return nil, err
	}
	return &scheme, nil
}

// Delete deletes a scheme
func (s *SchemeService) Delete(id string) error {
	return deleteByID(s.db, `DELETE FROM schemes WHERE id = $1`, id)
}
