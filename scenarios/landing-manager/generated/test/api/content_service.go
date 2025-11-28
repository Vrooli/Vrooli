package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ContentSection represents a customizable section of the landing page
type ContentSection struct {
	ID          int64                  `json:"id"`
	VariantID   int64                  `json:"variant_id"`
	SectionType string                 `json:"section_type"` // hero, features, pricing, cta, etc.
	Content     map[string]interface{} `json:"content"`
	Order       int                    `json:"order"`
	Enabled     bool                   `json:"enabled"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// ContentService handles CRUD operations for landing page content
type ContentService struct {
	db *sql.DB
}

// NewContentService creates a new content service
func NewContentService(db *sql.DB) *ContentService {
	return &ContentService{db: db}
}

// GetSections retrieves all sections for a variant
func (s *ContentService) GetSections(variantID int64) ([]ContentSection, error) {
	query := `
		SELECT id, variant_id, section_type, content, "order", enabled, created_at, updated_at
		FROM content_sections
		WHERE variant_id = $1
		ORDER BY "order" ASC
	`

	rows, err := s.db.Query(query, variantID)
	if err != nil {
		return nil, fmt.Errorf("query sections: %w", err)
	}
	defer rows.Close()

	var sections []ContentSection
	for rows.Next() {
		var section ContentSection
		var contentJSON []byte

		err := rows.Scan(
			&section.ID,
			&section.VariantID,
			&section.SectionType,
			&contentJSON,
			&section.Order,
			&section.Enabled,
			&section.CreatedAt,
			&section.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan section: %w", err)
		}

		if err := json.Unmarshal(contentJSON, &section.Content); err != nil {
			return nil, fmt.Errorf("unmarshal content: %w", err)
		}

		sections = append(sections, section)
	}

	return sections, nil
}

// GetSection retrieves a single section by ID
func (s *ContentService) GetSection(sectionID int64) (*ContentSection, error) {
	query := `
		SELECT id, variant_id, section_type, content, "order", enabled, created_at, updated_at
		FROM content_sections
		WHERE id = $1
	`

	var section ContentSection
	var contentJSON []byte

	err := s.db.QueryRow(query, sectionID).Scan(
		&section.ID,
		&section.VariantID,
		&section.SectionType,
		&contentJSON,
		&section.Order,
		&section.Enabled,
		&section.CreatedAt,
		&section.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("section not found")
	}
	if err != nil {
		return nil, fmt.Errorf("query section: %w", err)
	}

	if err := json.Unmarshal(contentJSON, &section.Content); err != nil {
		return nil, fmt.Errorf("unmarshal content: %w", err)
	}

	return &section, nil
}

// UpdateSection updates a section's content
func (s *ContentService) UpdateSection(sectionID int64, content map[string]interface{}) error {
	contentJSON, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("marshal content: %w", err)
	}

	query := `
		UPDATE content_sections
		SET content = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := s.db.Exec(query, contentJSON, sectionID)
	if err != nil {
		return fmt.Errorf("update section: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("section not found")
	}

	return nil
}

// CreateSection creates a new content section
func (s *ContentService) CreateSection(section ContentSection) (*ContentSection, error) {
	contentJSON, err := json.Marshal(section.Content)
	if err != nil {
		return nil, fmt.Errorf("marshal content: %w", err)
	}

	query := `
		INSERT INTO content_sections (variant_id, section_type, content, "order", enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	err = s.db.QueryRow(
		query,
		section.VariantID,
		section.SectionType,
		contentJSON,
		section.Order,
		section.Enabled,
	).Scan(&section.ID, &section.CreatedAt, &section.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("insert section: %w", err)
	}

	return &section, nil
}

// DeleteSection soft-deletes a section
func (s *ContentService) DeleteSection(sectionID int64) error {
	query := `DELETE FROM content_sections WHERE id = $1`

	result, err := s.db.Exec(query, sectionID)
	if err != nil {
		return fmt.Errorf("delete section: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("section not found")
	}

	return nil
}
