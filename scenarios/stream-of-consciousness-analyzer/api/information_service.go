// DOC: docs/internal/SEAMS.md#api--database
// DOC: docs/reference/api-endpoints.md#information-canvas-items
package main

import "database/sql"

// InformationService manages information item CRUD
type InformationService struct {
	db *sql.DB
}

// NewInformationService creates a new InformationService
func NewInformationService(db *sql.DB) *InformationService {
	return &InformationService{db: db}
}

// Create creates a new information item in a scheme.
// If no type is specified, defaults to "text" — the most common capture mode.
// CHANGE AXIS: New content types (voice, url, image) only need to pass
// a different Type string; no structural changes required.
func (s *InformationService) Create(schemeID string, input *CreateInformationInput) (*Information, error) {
	itemType := input.Type
	if itemType == "" {
		itemType = "text"
	}
	info, err := scanInformation(s.db.QueryRow(
		`INSERT INTO information (scheme_id, type, content, canvas_x, canvas_y)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, scheme_id, type, content, canvas_x, canvas_y, created_at, updated_at`,
		schemeID, itemType, input.Content, input.CanvasX, input.CanvasY,
	))
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// ListByScheme returns all information items for a scheme
func (s *InformationService) ListByScheme(schemeID string) ([]Information, error) {
	rows, err := s.db.Query(
		`SELECT id, scheme_id, type, content, canvas_x, canvas_y, created_at, updated_at
		 FROM information WHERE scheme_id = $1 ORDER BY created_at ASC`,
		schemeID,
	)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanInformation)
}

// Update updates an information item
func (s *InformationService) Update(id string, input *UpdateInformationInput) (*Information, error) {
	info, err := scanInformation(s.db.QueryRow(
		`UPDATE information SET
			type = COALESCE($1, type),
			content = COALESCE($2, content),
			canvas_x = COALESCE($3, canvas_x),
			canvas_y = COALESCE($4, canvas_y),
			updated_at = NOW()
		 WHERE id = $5
		 RETURNING id, scheme_id, type, content, canvas_x, canvas_y, created_at, updated_at`,
		input.Type, input.Content, input.CanvasX, input.CanvasY, id,
	))
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// Delete deletes an information item
func (s *InformationService) Delete(id string) error {
	return deleteByID(s.db, `DELETE FROM information WHERE id = $1`, id)
}
