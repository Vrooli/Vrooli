package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// Variant represents an A/B testing variant
type Variant struct {
	ID          int               `json:"id"`
	Slug        string            `json:"slug"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Weight      int               `json:"weight"`
	Status      string            `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	ArchivedAt  *time.Time        `json:"archived_at,omitempty"`
	Axes        map[string]string `json:"axes,omitempty"`
	HeaderConfig LandingHeaderConfig `json:"header_config"`
}

// VariantService handles A/B testing variant operations
type VariantService struct {
	db    *sql.DB
	space *VariantSpace
}

// NewVariantService creates a new variant service
func NewVariantService(db *sql.DB, space *VariantSpace) *VariantService {
	if space == nil {
		space = defaultVariantSpace
	}
	return &VariantService{db: db, space: space}
}

// SelectVariant implements weighted random variant selection (OT-P0-016: AB-API)
// Returns a variant based on weight distribution
func (vs *VariantService) SelectVariant() (*Variant, error) {
	// Fetch all active variants
	variants, err := vs.ListVariants("active")
	if err != nil {
		return nil, err
	}

	if len(variants) == 0 {
		return nil, errors.New("no active variants available")
	}

	// Calculate total weight
	totalWeight := 0
	for _, v := range variants {
		totalWeight += v.Weight
	}

	if totalWeight == 0 {
		// If all weights are 0, distribute evenly
		rand.Seed(time.Now().UnixNano())
		return &variants[rand.Intn(len(variants))], nil
	}

	// Weighted random selection
	rand.Seed(time.Now().UnixNano())
	randomWeight := rand.Intn(totalWeight)
	cumulativeWeight := 0

	for _, v := range variants {
		cumulativeWeight += v.Weight
		if randomWeight < cumulativeWeight {
			return &v, nil
		}
	}

	// Fallback to first variant (should never reach here)
	return &variants[0], nil
}

// GetVariantBySlug retrieves a variant by slug (OT-P0-014: AB-URL)
func (vs *VariantService) GetVariantBySlug(slug string) (*Variant, error) {
	query := `
		SELECT id, slug, name, description, weight, status, created_at, updated_at, archived_at, header_config
		FROM variants
		WHERE slug = $1
	`

	var v Variant
	var archivedAt sql.NullTime
	var headerJSON []byte
	err := vs.db.QueryRow(query, slug).Scan(
		&v.ID, &v.Slug, &v.Name, &v.Description, &v.Weight,
		&v.Status, &v.CreatedAt, &v.UpdatedAt, &archivedAt, &headerJSON,
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

	axes, err := vs.getVariantAxes(v.ID)
	if err != nil {
		return nil, err
	}
	v.Axes = axes

	return &v, nil
}

// ListVariants returns all variants, optionally filtered by status (OT-P0-017: AB-CRUD)
func (vs *VariantService) ListVariants(statusFilter string) ([]Variant, error) {
	query := `
		SELECT id, slug, name, description, weight, status, created_at, updated_at, archived_at, header_config
		FROM variants
		WHERE status != 'deleted'
	`

	args := []interface{}{}
	if statusFilter != "" {
		query += " AND status = $1"
		args = append(args, statusFilter)
	}

	query += " ORDER BY created_at ASC"

	rows, err := vs.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	variants := []Variant{}
	for rows.Next() {
		var v Variant
		var archivedAt sql.NullTime
		var headerJSON []byte
		err := rows.Scan(
			&v.ID, &v.Slug, &v.Name, &v.Description, &v.Weight,
			&v.Status, &v.CreatedAt, &v.UpdatedAt, &archivedAt, &headerJSON,
		)
		if err != nil {
			return nil, err
		}

		if archivedAt.Valid {
			v.ArchivedAt = &archivedAt.Time
		}

		v.HeaderConfig = decodeHeaderConfig(headerJSON, v.Name, v.Slug)

		variants = append(variants, v)
	}

	for i := range variants {
		axes, err := vs.getVariantAxes(variants[i].ID)
		if err != nil {
			return nil, err
		}
		variants[i].Axes = axes
	}

	return variants, nil
}

// CreateVariant creates a new variant (OT-P0-017: AB-CRUD)
func (vs *VariantService) CreateVariant(slug, name, description string, weight int, axes map[string]string) (*Variant, error) {
	if weight < 0 || weight > 100 {
		return nil, errors.New("weight must be between 0 and 100")
	}
	if len(axes) == 0 {
		return nil, errors.New("axes selection is required")
	}
	if err := vs.space.ValidateSelection(axes); err != nil {
		return nil, err
	}

	headerCfg := defaultLandingHeaderConfig(name)
	headerJSON, err := marshalHeaderConfig(headerCfg)
	if err != nil {
		return nil, fmt.Errorf("marshal header config: %w", err)
	}

	tx, err := vs.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO variants (slug, name, description, weight, status, header_config)
		VALUES ($1, $2, $3, $4, 'active', $5)
		RETURNING id, slug, name, description, weight, status, created_at, updated_at
	`

	var v Variant
	err = tx.QueryRow(query, slug, name, description, weight, headerJSON).Scan(
		&v.ID, &v.Slug, &v.Name, &v.Description, &v.Weight,
		&v.Status, &v.CreatedAt, &v.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if err := vs.saveVariantAxesTx(tx, v.ID, axes); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	v.Axes = axes
	v.HeaderConfig = headerCfg

	return &v, nil
}

// UpdateVariant updates variant values (OT-P0-017: AB-CRUD)
func (vs *VariantService) UpdateVariant(slug string, name *string, description *string, weight *int, axes map[string]string, headerConfig *LandingHeaderConfig) (*Variant, error) {
	if weight != nil && (*weight < 0 || *weight > 100) {
		return nil, errors.New("weight must be between 0 and 100")
	}

	var headerJSON []byte
	if headerConfig != nil {
		targetName := ""
		if name != nil && strings.TrimSpace(*name) != "" {
			targetName = strings.TrimSpace(*name)
		} else {
			if err := vs.db.QueryRow(`SELECT name FROM variants WHERE slug = $1`, slug).Scan(&targetName); err != nil {
				return nil, fmt.Errorf("fetch variant for header config: %w", err)
			}
		}
		norm := normalizeLandingHeaderConfig(headerConfig, targetName)
		var err error
		headerJSON, err = marshalHeaderConfig(norm)
		if err != nil {
			return nil, fmt.Errorf("marshal header config: %w", err)
		}
	}

	// Build dynamic update query
	query := "UPDATE variants SET updated_at = NOW()"
	args := []interface{}{}
	argIndex := 1

	if name != nil {
		query += fmt.Sprintf(", name = $%d", argIndex)
		args = append(args, *name)
		argIndex++
	}

	if description != nil {
		query += fmt.Sprintf(", description = $%d", argIndex)
		args = append(args, *description)
		argIndex++
	}

	if weight != nil {
		query += fmt.Sprintf(", weight = $%d", argIndex)
		args = append(args, *weight)
		argIndex++
	}

	if headerJSON != nil {
		query += fmt.Sprintf(", header_config = $%d", argIndex)
		args = append(args, headerJSON)
		argIndex++
	}

	query += fmt.Sprintf(" WHERE slug = $%d", argIndex)
	args = append(args, slug)

	query += " RETURNING id, slug, name, description, weight, status, created_at, updated_at, archived_at, header_config"

	var v Variant
	var archivedAt sql.NullTime
	tx, err := vs.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

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
		if err := vs.saveVariantAxesTx(tx, v.ID, axes); err != nil {
			return nil, err
		}
		v.Axes = axes
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	if axes == nil {
		axesMap, err := vs.getVariantAxes(v.ID)
		if err != nil {
			return nil, err
		}
		v.Axes = axesMap
	}

	return &v, nil
}

// ArchiveVariant archives a variant (OT-P0-017, OT-P0-018: AB-CRUD, AB-ARCHIVE)
// Archived variants remain queryable but won't be selected randomly
func (vs *VariantService) ArchiveVariant(slug string) error {
	query := `
		UPDATE variants
		SET status = 'archived', archived_at = NOW(), updated_at = NOW()
		WHERE slug = $1 AND status != 'deleted'
	`

	result, err := vs.db.Exec(query, slug)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return errors.New("variant not found or already deleted")
	}

	return nil
}

// DeleteVariant soft-deletes a variant (OT-P0-017: AB-CRUD)
func (vs *VariantService) DeleteVariant(slug string) error {
	query := `
		UPDATE variants
		SET status = 'deleted', updated_at = NOW()
		WHERE slug = $1
	`

	result, err := vs.db.Exec(query, slug)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return errors.New("variant not found")
	}

	return nil
}

func (vs *VariantService) getVariantAxes(variantID int) (map[string]string, error) {
	rows, err := vs.db.Query(`SELECT axis_id, variant_value FROM variant_axes WHERE variant_id = $1`, variantID)
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

	return axes, nil
}

func (vs *VariantService) saveVariantAxesTx(tx *sql.Tx, variantID int, axes map[string]string) error {
	if len(axes) == 0 {
		return errors.New("axes selection is required")
	}
	if err := vs.space.ValidateSelection(axes); err != nil {
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
			DO UPDATE SET variant_value = EXCLUDED.variant_value, updated_at = NOW()
		`, variantID, axisID, value); err != nil {
			return fmt.Errorf("insert variant axis %s: %w", axisID, err)
		}
	}

	return nil
}

func decodeHeaderConfig(raw []byte, variantName, slug string) LandingHeaderConfig {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return defaultLandingHeaderConfig(variantName)
	}

	var cfg LandingHeaderConfig
	if err := json.Unmarshal(trimmed, &cfg); err != nil {
		logStructuredError("variant_header_parse_failed", map[string]interface{}{
			"variant_slug": slug,
			"error":        err.Error(),
		})
		return defaultLandingHeaderConfig(variantName)
	}

	return normalizeLandingHeaderConfig(&cfg, variantName)
}
