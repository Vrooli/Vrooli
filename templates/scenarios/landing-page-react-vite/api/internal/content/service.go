// Package content is the application layer for a variant's ordered landing-page
// sections: typed blocks (hero, features, pricing, …) each carrying a free-form
// JSON content payload. The Connect handler in handlers/content is a thin
// proto<->domain adapter over this Service; all SQL and validation live here.
package content

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Section is a single ordered, typed block of a variant's landing page.
type Section struct {
	ID          int64
	VariantID   int64
	SectionType string
	Content     map[string]interface{}
	Order       int
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SectionInput is the shape used by snapshot import: a section without server
// identity, with presence-tracked enabled (nil => default true).
type SectionInput struct {
	SectionType string
	Content     map[string]interface{}
	Order       int
	Enabled     *bool
}

// AllowedSectionTypes is the whitelist of section types accepted on write.
var AllowedSectionTypes = map[string]bool{
	"hero":         true,
	"features":     true,
	"pricing":      true,
	"cta":          true,
	"testimonials": true,
	"faq":          true,
	"footer":       true,
	"video":        true,
	"downloads":    true,
}

// Service owns content-section reads and writes.
type Service struct {
	db *sql.DB
}

// NewService constructs the content Service.
func NewService(db *sql.DB) *Service { return &Service{db: db} }

const sectionColumns = `id, variant_id, section_type, content, "order", enabled, created_at, updated_at`

// GetSections returns every section for a variant, ordered.
func (s *Service) GetSections(variantID int64) ([]Section, error) {
	return s.querySections(`SELECT `+sectionColumns+`
		FROM content_sections WHERE variant_id = $1 ORDER BY "order" ASC`, variantID)
}

// GetPublicSections returns only enabled sections for public display.
func (s *Service) GetPublicSections(variantID int64) ([]Section, error) {
	return s.querySections(`SELECT `+sectionColumns+`
		FROM content_sections WHERE variant_id = $1 AND enabled = TRUE ORDER BY "order" ASC`, variantID)
}

func (s *Service) querySections(query string, variantID int64) ([]Section, error) {
	rows, err := s.db.Query(query, variantID)
	if err != nil {
		return nil, fmt.Errorf("query sections: %w", err)
	}
	defer rows.Close()

	sections := []Section{}
	for rows.Next() {
		section, err := scanSection(rows)
		if err != nil {
			return nil, err
		}
		sections = append(sections, *section)
	}
	return sections, rows.Err()
}

// GetSection retrieves a single section by id.
func (s *Service) GetSection(sectionID int64) (*Section, error) {
	section, err := scanSection(s.db.QueryRow(`SELECT `+sectionColumns+` FROM content_sections WHERE id = $1`, sectionID))
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("section not found")
	}
	if err != nil {
		return nil, fmt.Errorf("query section: %w", err)
	}
	return section, nil
}

// CreateSection inserts a new content section and returns it with identity.
func (s *Service) CreateSection(section Section) (*Section, error) {
	contentJSON, err := json.Marshal(section.Content)
	if err != nil {
		return nil, fmt.Errorf("marshal content: %w", err)
	}
	err = s.db.QueryRow(`
		INSERT INTO content_sections (variant_id, section_type, content, "order", enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, created_at, updated_at`,
		section.VariantID, section.SectionType, contentJSON, section.Order, section.Enabled,
	).Scan(&section.ID, &section.CreatedAt, &section.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert section: %w", err)
	}
	return &section, nil
}

// UpdateSection replaces a section's content payload.
func (s *Service) UpdateSection(sectionID int64, content map[string]interface{}) error {
	contentJSON, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("marshal content: %w", err)
	}
	result, err := s.db.Exec(`UPDATE content_sections SET content = $1, updated_at = NOW() WHERE id = $2`, contentJSON, sectionID)
	if err != nil {
		return fmt.Errorf("update section: %w", err)
	}
	return requireAffected(result)
}

// DeleteSection removes a section by id.
func (s *Service) DeleteSection(sectionID int64) error {
	result, err := s.db.Exec(`DELETE FROM content_sections WHERE id = $1`, sectionID)
	if err != nil {
		return fmt.Errorf("delete section: %w", err)
	}
	return requireAffected(result)
}

// CopySectionsFromVariant copies every section from source to target. Used when
// creating a variant so it inherits the control's content.
func (s *Service) CopySectionsFromVariant(sourceVariantID, targetVariantID int64) error {
	sections, err := s.GetSections(sourceVariantID)
	if err != nil {
		return fmt.Errorf("get source sections: %w", err)
	}
	for _, section := range sections {
		if _, err := s.CreateSection(Section{
			VariantID:   targetVariantID,
			SectionType: section.SectionType,
			Content:     section.Content,
			Order:       section.Order,
			Enabled:     section.Enabled,
		}); err != nil {
			return fmt.Errorf("copy section %s: %w", section.SectionType, err)
		}
	}
	return nil
}

// ReplaceSectionsTx replaces all sections for a variant inside a transaction.
// It validates section types before writing and re-numbers zero/negative order
// values by stable position.
func (s *Service) ReplaceSectionsTx(tx *sql.Tx, variantID int64, sections []SectionInput) error {
	if tx == nil {
		return fmt.Errorf("transaction required")
	}

	for _, section := range sections {
		sectionType := strings.TrimSpace(section.SectionType)
		if sectionType == "" {
			return fmt.Errorf("section_type is required")
		}
		if !AllowedSectionTypes[sectionType] {
			return fmt.Errorf("section_type %q is not supported", sectionType)
		}
		if section.Content == nil {
			return fmt.Errorf("content is required for section %q", sectionType)
		}
	}

	if _, err := tx.Exec(`DELETE FROM content_sections WHERE variant_id = $1`, variantID); err != nil {
		return fmt.Errorf("clear existing sections: %w", err)
	}

	sort.SliceStable(sections, func(i, j int) bool { return sections[i].Order < sections[j].Order })

	for idx, section := range sections {
		order := section.Order
		if order <= 0 {
			order = idx + 1
		}
		enabled := true
		if section.Enabled != nil {
			enabled = *section.Enabled
		}
		contentJSON, err := json.Marshal(section.Content)
		if err != nil {
			return fmt.Errorf("marshal section %s content: %w", section.SectionType, err)
		}
		if _, err := tx.Exec(`
			INSERT INTO content_sections (variant_id, section_type, content, "order", enabled, created_at, updated_at)
			VALUES ($1, $2, $3::jsonb, $4, $5, NOW(), NOW())`,
			variantID, section.SectionType, contentJSON, order, enabled); err != nil {
			return fmt.Errorf("insert section %s: %w", section.SectionType, err)
		}
	}
	return nil
}

func scanSection(row interface{ Scan(...any) error }) (*Section, error) {
	var section Section
	var contentJSON []byte
	if err := row.Scan(
		&section.ID, &section.VariantID, &section.SectionType, &contentJSON,
		&section.Order, &section.Enabled, &section.CreatedAt, &section.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(contentJSON, &section.Content); err != nil {
		return nil, fmt.Errorf("unmarshal content: %w", err)
	}
	return &section, nil
}

func requireAffected(result sql.Result) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("section not found")
	}
	return nil
}
