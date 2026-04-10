// DOC: docs/internal/SEAMS.md#api--database
// DOC: docs/reference/api-endpoints.md#thoughts
package main

import "database/sql"

// ThoughtService manages thought CRUD and edge operations
type ThoughtService struct {
	db *sql.DB
}

// NewThoughtService creates a new ThoughtService
func NewThoughtService(db *sql.DB) *ThoughtService {
	return &ThoughtService{db: db}
}

// Create creates a new thought
func (s *ThoughtService) Create(input *CreateThoughtInput) (*Thought, error) {
	t, err := scanThought(s.db.QueryRow(
		`INSERT INTO thoughts (scheme_id, title, body, canvas_x, canvas_y)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, scheme_id, title, body, canvas_x, canvas_y, created_at, updated_at`,
		input.SchemeID, input.Title, input.Body, input.CanvasX, input.CanvasY,
	))
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// List returns all thoughts, optionally filtered by scheme.
// When schemeID is empty, returns ALL thoughts across schemes — this supports
// the cross-scheme navigation feature (P2) where users explore connections
// that span multiple workspaces.
func (s *ThoughtService) List(schemeID string) ([]Thought, error) {
	var rows *sql.Rows
	var err error
	if schemeID != "" {
		rows, err = s.db.Query(
			`SELECT id, scheme_id, title, body, canvas_x, canvas_y, created_at, updated_at
			 FROM thoughts WHERE scheme_id = $1 ORDER BY created_at ASC`, schemeID,
		)
	} else {
		rows, err = s.db.Query(
			`SELECT id, scheme_id, title, body, canvas_x, canvas_y, created_at, updated_at
			 FROM thoughts ORDER BY created_at ASC`,
		)
	}
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanThought)
}

// GetByID returns a thought by ID
func (s *ThoughtService) GetByID(id string) (*Thought, error) {
	t, err := scanThought(s.db.QueryRow(
		`SELECT id, scheme_id, title, body, canvas_x, canvas_y, created_at, updated_at
		 FROM thoughts WHERE id = $1`, id,
	))
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Update updates a thought
func (s *ThoughtService) Update(id string, input *UpdateThoughtInput) (*Thought, error) {
	t, err := scanThought(s.db.QueryRow(
		`UPDATE thoughts SET
			title = COALESCE($1, title),
			body = COALESCE($2, body),
			canvas_x = COALESCE($3, canvas_x),
			canvas_y = COALESCE($4, canvas_y),
			updated_at = NOW()
		 WHERE id = $5
		 RETURNING id, scheme_id, title, body, canvas_x, canvas_y, created_at, updated_at`,
		input.Title, input.Body, input.CanvasX, input.CanvasY, id,
	))
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Delete deletes a thought
func (s *ThoughtService) Delete(id string) error {
	return deleteByID(s.db, `DELETE FROM thoughts WHERE id = $1`, id)
}

// CreateEdge creates a directional edge between two thoughts
func (s *ThoughtService) CreateEdge(sourceID string, input *CreateEdgeInput) (*ThoughtEdge, error) {
	e, err := scanEdge(s.db.QueryRow(
		`INSERT INTO thought_edges (source_id, target_id, label)
		 VALUES ($1, $2, $3)
		 RETURNING id, source_id, target_id, label, created_at`,
		sourceID, input.TargetID, input.Label,
	))
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// ListEdges returns all edges for a thought (both as source and target)
func (s *ThoughtService) ListEdges(thoughtID string) ([]ThoughtEdge, error) {
	rows, err := s.db.Query(
		`SELECT id, source_id, target_id, label, created_at
		 FROM thought_edges WHERE source_id = $1 OR target_id = $1
		 ORDER BY created_at ASC`, thoughtID,
	)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanEdge)
}

// DeleteEdge deletes an edge
func (s *ThoughtService) DeleteEdge(id string) error {
	return deleteByID(s.db, `DELETE FROM thought_edges WHERE id = $1`, id)
}
