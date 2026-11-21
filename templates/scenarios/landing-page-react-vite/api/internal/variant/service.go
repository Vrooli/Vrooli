// Package variant is the application layer for A/B landing variants: weighted
// random selection, admin CRUD, axis validation against the variant space,
// header-config normalization, and portable export/import snapshots (which fold
// in the variant's content sections). The Connect handler in handlers/variant
// is a thin proto<->domain adapter over this Service.
package variant

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"landing-page-react-vite-api/internal/content"
	"landing-page-react-vite-api/internal/variantspace"
	"log"
	"math/rand"
	"strings"
	"time"
)

// Variant is an authored A/B landing variation.
type Variant struct {
	ID           int
	Slug         string
	Name         string
	Description  string
	Weight       int
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ArchivedAt   *time.Time
	Axes         map[string]string
	HeaderConfig LandingHeaderConfig
	// SEOConfig is the raw per-variant SEO override JSON, present only on
	// single-variant admin reads (Get/Export). Nil when unset.
	SEOConfig *json.RawMessage
}

// Snapshot is the portable export payload: authoring fields plus ordered
// sections and axis selections.
type Snapshot struct {
	Slug         string
	Name         string
	Description  string
	Weight       int
	Status       string
	Axes         map[string]string
	HeaderConfig LandingHeaderConfig
	SEOConfig    json.RawMessage
	Sections     []content.Section
}

// SnapshotInput is the import payload; a nil HeaderConfig means "use defaults".
type SnapshotInput struct {
	Slug         string
	Name         string
	Description  string
	Weight       int
	Status       string
	Axes         map[string]string
	HeaderConfig *LandingHeaderConfig
	SEOConfig    json.RawMessage
	Sections     []content.SectionInput
}

var allowedVariantStatuses = map[string]bool{
	"active":   true,
	"archived": true,
}

// Service owns variant reads and writes.
type Service struct {
	db      *sql.DB
	space   *variantspace.Space
	content *content.Service
}

// NewService constructs the variant Service. When space is nil the baked-in
// default variant space is used.
func NewService(db *sql.DB, space *variantspace.Space, contentSvc *content.Service) *Service {
	if space == nil {
		space = variantspace.Default()
	}
	return &Service{db: db, space: space, content: contentSvc}
}

// SelectVariant returns a weighted-random active variant.
func (s *Service) SelectVariant() (*Variant, error) {
	variants, err := s.ListVariants("active")
	if err != nil {
		return nil, err
	}
	if len(variants) == 0 {
		return nil, errors.New("no active variants available")
	}

	totalWeight := 0
	for _, v := range variants {
		totalWeight += v.Weight
	}
	if totalWeight == 0 {
		rand.Seed(time.Now().UnixNano())
		return &variants[rand.Intn(len(variants))], nil
	}

	rand.Seed(time.Now().UnixNano())
	randomWeight := rand.Intn(totalWeight)
	cumulativeWeight := 0
	for i := range variants {
		cumulativeWeight += variants[i].Weight
		if randomWeight < cumulativeWeight {
			return &variants[i], nil
		}
	}
	return &variants[0], nil
}

// GetVariantBySlug retrieves a variant by slug including its SEO override.
func (s *Service) GetVariantBySlug(slug string) (*Variant, error) {
	var v Variant
	var archivedAt sql.NullTime
	var headerJSON, seoJSON []byte
	err := s.db.QueryRow(`
		SELECT id, slug, name, COALESCE(description, ''), weight, status, created_at, updated_at, archived_at,
			header_config, COALESCE(seo_config, '{}'::jsonb)
		FROM variants WHERE slug = $1`, slug).Scan(
		&v.ID, &v.Slug, &v.Name, &v.Description, &v.Weight,
		&v.Status, &v.CreatedAt, &v.UpdatedAt, &archivedAt, &headerJSON, &seoJSON,
	)
	if err == sql.ErrNoRows {
		return nil, errors.New("variant not found")
	}
	if err != nil {
		return nil, err
	}
	if archivedAt.Valid {
		v.ArchivedAt = &archivedAt.Time
	}
	v.HeaderConfig = decodeHeaderConfig(headerJSON, v.Name, v.Slug)
	if len(seoJSON) > 0 && string(seoJSON) != "{}" {
		raw := json.RawMessage(seoJSON)
		v.SEOConfig = &raw
	}
	axes, err := s.getVariantAxes(v.ID)
	if err != nil {
		return nil, err
	}
	v.Axes = axes
	return &v, nil
}

// ListVariants returns all non-deleted variants, optionally filtered by status.
func (s *Service) ListVariants(statusFilter string) ([]Variant, error) {
	query := `
		SELECT id, slug, name, COALESCE(description, ''), weight, status, created_at, updated_at, archived_at, header_config
		FROM variants WHERE status != 'deleted'`
	args := []interface{}{}
	if statusFilter != "" {
		query += " AND status = $1"
		args = append(args, statusFilter)
	}
	query += " ORDER BY created_at ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	variants := []Variant{}
	for rows.Next() {
		var v Variant
		var archivedAt sql.NullTime
		var headerJSON []byte
		if err := rows.Scan(&v.ID, &v.Slug, &v.Name, &v.Description, &v.Weight,
			&v.Status, &v.CreatedAt, &v.UpdatedAt, &archivedAt, &headerJSON); err != nil {
			return nil, err
		}
		if archivedAt.Valid {
			v.ArchivedAt = &archivedAt.Time
		}
		v.HeaderConfig = decodeHeaderConfig(headerJSON, v.Name, v.Slug)
		variants = append(variants, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range variants {
		axes, err := s.getVariantAxes(variants[i].ID)
		if err != nil {
			return nil, err
		}
		variants[i].Axes = axes
	}
	return variants, nil
}

// CreateVariant creates a new active variant with validated axes and a default
// header config.
func (s *Service) CreateVariant(slug, name, description string, weight int, axes map[string]string) (*Variant, error) {
	if weight < 0 || weight > 100 {
		return nil, errors.New("weight must be between 0 and 100")
	}
	if len(axes) == 0 {
		return nil, errors.New("axes selection is required")
	}
	if err := s.space.ValidateSelection(axes); err != nil {
		return nil, err
	}

	headerCfg := defaultLandingHeaderConfig(name)
	headerJSON, err := marshalHeaderConfig(headerCfg)
	if err != nil {
		return nil, fmt.Errorf("marshal header config: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	var v Variant
	err = tx.QueryRow(`
		INSERT INTO variants (slug, name, description, weight, status, header_config)
		VALUES ($1, $2, $3, $4, 'active', $5)
		RETURNING id, slug, name, description, weight, status, created_at, updated_at`,
		slug, name, description, weight, headerJSON,
	).Scan(&v.ID, &v.Slug, &v.Name, &v.Description, &v.Weight, &v.Status, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if err := s.saveVariantAxesTx(tx, v.ID, axes); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	v.Axes = axes
	v.HeaderConfig = headerCfg
	return &v, nil
}

// UpdateVariant applies a partial update. A nil scalar pointer leaves the field
// unchanged; a nil axes map leaves axes unchanged; a nil headerConfig leaves the
// header unchanged.
func (s *Service) UpdateVariant(slug string, name, description *string, weight *int, axes map[string]string, headerConfig *LandingHeaderConfig) (*Variant, error) {
	if weight != nil && (*weight < 0 || *weight > 100) {
		return nil, errors.New("weight must be between 0 and 100")
	}

	headerJSON, err := s.resolveHeaderJSON(slug, name, headerConfig)
	if err != nil {
		return nil, err
	}

	query, args := buildVariantUpdateQuery(slug, name, description, weight, headerJSON)

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	var v Variant
	var archivedAt sql.NullTime
	var headerFromDB []byte
	err = tx.QueryRow(query, args...).Scan(
		&v.ID, &v.Slug, &v.Name, &v.Description, &v.Weight,
		&v.Status, &v.CreatedAt, &v.UpdatedAt, &archivedAt, &headerFromDB,
	)
	if err == sql.ErrNoRows {
		return nil, errors.New("variant not found")
	}
	if err != nil {
		return nil, err
	}
	if archivedAt.Valid {
		v.ArchivedAt = &archivedAt.Time
	}
	v.HeaderConfig = decodeHeaderConfig(headerFromDB, v.Name, v.Slug)

	if axes != nil {
		if err := s.saveVariantAxesTx(tx, v.ID, axes); err != nil {
			return nil, err
		}
		v.Axes = axes
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	if axes == nil {
		axesMap, err := s.getVariantAxes(v.ID)
		if err != nil {
			return nil, err
		}
		v.Axes = axesMap
	}
	return &v, nil
}

// resolveHeaderJSON normalizes and marshals the header config for an update,
// resolving the target variant name (from the incoming update or the current
// row) so the normalized config carries a stable title. Returns nil when no
// header config is supplied.
func (s *Service) resolveHeaderJSON(slug string, name *string, headerConfig *LandingHeaderConfig) ([]byte, error) {
	if headerConfig == nil {
		return nil, nil
	}
	targetName := ""
	if name != nil && strings.TrimSpace(*name) != "" {
		targetName = strings.TrimSpace(*name)
	} else if err := s.db.QueryRow(`SELECT name FROM variants WHERE slug = $1`, slug).Scan(&targetName); err != nil {
		return nil, fmt.Errorf("fetch variant for header config: %w", err)
	}
	headerJSON, err := marshalHeaderConfig(normalizeLandingHeaderConfig(headerConfig, targetName))
	if err != nil {
		return nil, fmt.Errorf("marshal header config: %w", err)
	}
	return headerJSON, nil
}

// buildVariantUpdateQuery assembles the partial UPDATE statement and its
// positional args, appending only the fields present in the request.
func buildVariantUpdateQuery(slug string, name, description *string, weight *int, headerJSON []byte) (string, []interface{}) {
	query := "UPDATE variants SET updated_at = NOW()"
	args := []interface{}{}
	appendField := func(column string, value interface{}) {
		query += fmt.Sprintf(", %s = $%d", column, len(args)+1)
		args = append(args, value)
	}
	if name != nil {
		appendField("name", *name)
	}
	if description != nil {
		appendField("description", *description)
	}
	if weight != nil {
		appendField("weight", *weight)
	}
	if headerJSON != nil {
		appendField("header_config", headerJSON)
	}
	query += fmt.Sprintf(" WHERE slug = $%d", len(args)+1)
	args = append(args, slug)
	query += " RETURNING id, slug, name, description, weight, status, created_at, updated_at, archived_at, header_config"
	return query, args
}

// ArchiveVariant marks a variant archived so it is no longer selected.
func (s *Service) ArchiveVariant(slug string) error {
	result, err := s.db.Exec(`
		UPDATE variants SET status = 'archived', archived_at = NOW(), updated_at = NOW()
		WHERE slug = $1 AND status != 'deleted'`, slug)
	if err != nil {
		return err
	}
	return requireRows(result, "variant not found or already deleted")
}

// DeleteVariant soft-deletes a variant.
func (s *Service) DeleteVariant(slug string) error {
	result, err := s.db.Exec(`UPDATE variants SET status = 'deleted', updated_at = NOW() WHERE slug = $1`, slug)
	if err != nil {
		return err
	}
	return requireRows(result, "variant not found")
}

func (s *Service) getVariantAxes(variantID int) (map[string]string, error) {
	rows, err := s.db.Query(`SELECT axis_id, variant_value FROM variant_axes WHERE variant_id = $1`, variantID)
	if err != nil {
		return nil, fmt.Errorf("query variant axes: %w", err)
	}
	defer rows.Close()

	axes := map[string]string{}
	for rows.Next() {
		var axisID, value string
		if err := rows.Scan(&axisID, &value); err != nil {
			return nil, fmt.Errorf("scan variant axis: %w", err)
		}
		axes[axisID] = value
	}
	return axes, rows.Err()
}

func (s *Service) saveVariantAxesTx(tx *sql.Tx, variantID int, axes map[string]string) error {
	if len(axes) == 0 {
		return errors.New("axes selection is required")
	}
	if err := s.space.ValidateSelection(axes); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM variant_axes WHERE variant_id = $1`, variantID); err != nil {
		return fmt.Errorf("clear variant axes: %w", err)
	}
	for axisID, value := range axes {
		if _, err := tx.Exec(`
			INSERT INTO variant_axes (variant_id, axis_id, variant_value, created_at, updated_at)
			VALUES ($1, $2, $3, NOW(), NOW())
			ON CONFLICT (variant_id, axis_id)
			DO UPDATE SET variant_value = EXCLUDED.variant_value, updated_at = NOW()`,
			variantID, axisID, value); err != nil {
			return fmt.Errorf("insert variant axis %s: %w", axisID, err)
		}
	}
	return nil
}

// CopyControlSections copies the control variant's (id=1) sections onto the
// target, giving a newly-created variant default content. Failures are logged
// but not fatal, matching the create-with-sections flow.
func (s *Service) CopyControlSections(targetVariantID int64) {
	if s.content == nil {
		return
	}
	if err := s.content.CopySectionsFromVariant(1, targetVariantID); err != nil {
		log.Printf("variant: failed to copy control sections to variant %d: %v", targetVariantID, err)
	}
}

// ExportSnapshot returns a full variant payload (metadata + sections).
func (s *Service) ExportSnapshot(slug string) (*Snapshot, error) {
	v, err := s.GetVariantBySlug(slug)
	if err != nil {
		return nil, err
	}
	sections, err := s.content.GetSections(int64(v.ID))
	if err != nil {
		return nil, fmt.Errorf("load sections: %w", err)
	}
	var seoRaw json.RawMessage
	if v.SEOConfig != nil {
		seoRaw = *v.SEOConfig
	}
	return &Snapshot{
		Slug:         v.Slug,
		Name:         v.Name,
		Description:  v.Description,
		Weight:       v.Weight,
		Status:       v.Status,
		Axes:         v.Axes,
		HeaderConfig: v.HeaderConfig,
		SEOConfig:    seoRaw,
		Sections:     sections,
	}, nil
}

// ImportSnapshot fully replaces a variant (metadata, axes, sections) from a
// snapshot in a single transaction. The snapshot slug must equal the path slug.
func (s *Service) ImportSnapshot(slug string, snapshot SnapshotInput) (*Snapshot, error) {
	if strings.TrimSpace(slug) == "" {
		return nil, errors.New("variant slug required")
	}
	if strings.TrimSpace(snapshot.Slug) != slug {
		return nil, fmt.Errorf("payload slug %q does not match route slug %q", snapshot.Slug, slug)
	}

	status := strings.TrimSpace(snapshot.Status)
	if status == "" {
		status = "active"
	}
	if !allowedVariantStatuses[status] {
		return nil, fmt.Errorf("status %q is not allowed", status)
	}
	if snapshot.Weight < 0 || snapshot.Weight > 100 {
		return nil, errors.New("weight must be between 0 and 100")
	}
	if len(snapshot.Axes) == 0 {
		return nil, errors.New("axes selection is required")
	}
	if err := s.space.ValidateSelection(snapshot.Axes); err != nil {
		return nil, err
	}

	headerCfg := normalizeLandingHeaderConfig(snapshot.HeaderConfig, snapshot.Name)
	headerJSON, err := marshalHeaderConfig(headerCfg)
	if err != nil {
		return nil, fmt.Errorf("marshal header config: %w", err)
	}

	seoJSON := []byte(`{}`)
	if len(snapshot.SEOConfig) > 0 {
		seoJSON = snapshot.SEOConfig
	}

	v, err := s.GetVariantBySlug(slug)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		UPDATE variants
		SET name = $1, description = $2, weight = $3, status = $4,
			header_config = $5, seo_config = $6, updated_at = NOW()
		WHERE slug = $7`,
		snapshot.Name, snapshot.Description, snapshot.Weight, status, headerJSON, seoJSON, slug); err != nil {
		return nil, fmt.Errorf("update variant: %w", err)
	}
	if err := s.saveVariantAxesTx(tx, v.ID, snapshot.Axes); err != nil {
		return nil, err
	}
	if err := s.content.ReplaceSectionsTx(tx, int64(v.ID), snapshot.Sections); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit variant import: %w", err)
	}

	return s.ExportSnapshot(slug)
}

// UpdateSEOConfig replaces the seo_config JSON for a variant by id.
func (s *Service) UpdateSEOConfig(variantID int, seoJSON []byte) error {
	_, err := s.db.Exec(`UPDATE variants SET seo_config = $1::jsonb, updated_at = NOW() WHERE id = $2`, seoJSON, variantID)
	return err
}

// ActiveWithSEO returns active variants with their raw SEO override populated,
// for sitemap generation. Only id, slug, status, and SEOConfig are hydrated.
func (s *Service) ActiveWithSEO() ([]Variant, error) {
	rows, err := s.db.Query(`
		SELECT id, slug, status, COALESCE(seo_config, '{}'::jsonb)
		FROM variants WHERE status = 'active' ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Variant
	for rows.Next() {
		var v Variant
		var seoJSON []byte
		if err := rows.Scan(&v.ID, &v.Slug, &v.Status, &seoJSON); err != nil {
			return nil, err
		}
		if len(seoJSON) > 0 && string(seoJSON) != "{}" {
			raw := json.RawMessage(seoJSON)
			v.SEOConfig = &raw
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func decodeHeaderConfig(raw []byte, variantName, slug string) LandingHeaderConfig {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return defaultLandingHeaderConfig(variantName)
	}
	var cfg LandingHeaderConfig
	if err := json.Unmarshal(trimmed, &cfg); err != nil {
		log.Printf("variant: header config parse failed for %q: %v", slug, err)
		return defaultLandingHeaderConfig(variantName)
	}
	return normalizeLandingHeaderConfig(&cfg, variantName)
}

func requireRows(result sql.Result, notFoundMsg string) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New(notFoundMsg)
	}
	return nil
}
