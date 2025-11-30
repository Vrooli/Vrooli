package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

// Variant represents an A/B testing variant
type Variant struct {
	ID          int       `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Weight      int       `json:"weight"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ArchivedAt  *time.Time `json:"archived_at,omitempty"`
	Axes        map[string]string `json:"axes,omitempty"`
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
		SELECT id, slug, name, description, weight, status, created_at, updated_at, archived_at
		FROM variants
		WHERE slug = $1
	`

	var v Variant
	var archivedAt sql.NullTime
	err := vs.db.QueryRow(query, slug).Scan(
		&v.ID, &v.Slug, &v.Name, &v.Description, &v.Weight,
		&v.Status, &v.CreatedAt, &v.UpdatedAt, &archivedAt,
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
		SELECT id, slug, name, description, weight, status, created_at, updated_at, archived_at
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
		err := rows.Scan(
			&v.ID, &v.Slug, &v.Name, &v.Description, &v.Weight,
			&v.Status, &v.CreatedAt, &v.UpdatedAt, &archivedAt,
		)
		if err != nil {
			return nil, err
		}

		if archivedAt.Valid {
			v.ArchivedAt = &archivedAt.Time
		}

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

	tx, err := vs.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO variants (slug, name, description, weight, status)
		VALUES ($1, $2, $3, $4, 'active')
		RETURNING id, slug, name, description, weight, status, created_at, updated_at
	`

	var v Variant
	err = tx.QueryRow(query, slug, name, description, weight).Scan(
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

	return &v, nil
}

// UpdateVariant updates variant weight and/or name (OT-P0-017: AB-CRUD)
func (vs *VariantService) UpdateVariant(slug string, name *string, description *string, weight *int, axes map[string]string) (*Variant, error) {
	if weight != nil && (*weight < 0 || *weight > 100) {
		return nil, errors.New("weight must be between 0 and 100")
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

	query += fmt.Sprintf(" WHERE slug = $%d", argIndex)
	args = append(args, slug)

	query += " RETURNING id, slug, name, description, weight, status, created_at, updated_at, archived_at"

	var v Variant
	var archivedAt sql.NullTime
	tx, err := vs.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	err = tx.QueryRow(query, args...).Scan(
		&v.ID, &v.Slug, &v.Name, &v.Description, &v.Weight,
		&v.Status, &v.CreatedAt, &v.UpdatedAt, &archivedAt,
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

// HTTP Handlers

// handleVariantSelect handles GET /api/v1/variants/select (OT-P0-016: AB-API)
// Returns the full variant object for frontend compatibility
func handleVariantSelect(vs *VariantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		variant, err := vs.SelectVariant()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(variant)
	}
}

// handlePublicVariantBySlug handles GET /api/v1/public/variants/{slug} (no auth required)
// Used by the public landing page for URL-based variant selection
func handlePublicVariantBySlug(vs *VariantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		slug := r.URL.Path[len("/api/v1/public/variants/"):]
		if slug == "" {
			http.Error(w, "Variant slug required", http.StatusBadRequest)
			return
		}

		variant, err := vs.GetVariantBySlug(slug)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		// Only return active variants for public access
		if variant.Status != "active" {
			http.Error(w, "Variant not available", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(variant)
	}
}

// handleVariantBySlug handles GET /api/v1/variants/{slug} (OT-P0-014: AB-URL)
func handleVariantBySlug(vs *VariantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		slug := r.URL.Path[len("/api/v1/variants/"):]
		if slug == "" || slug == "select" {
			http.Error(w, "Variant slug required", http.StatusBadRequest)
			return
		}

		variant, err := vs.GetVariantBySlug(slug)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(variant)
	}
}

// handleVariantsList handles GET /api/v1/variants (OT-P0-017: AB-CRUD)
func handleVariantsList(vs *VariantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		statusFilter := r.URL.Query().Get("status")

		variants, err := vs.ListVariants(statusFilter)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"variants": variants,
		})
	}
}

// handleVariantCreate handles POST /api/v1/variants (OT-P0-017: AB-CRUD)
// DEPRECATED: Use handleVariantCreateWithSections instead
func handleVariantCreate(vs *VariantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Slug        string            `json:"slug"`
			Name        string            `json:"name"`
			Description string            `json:"description"`
			Weight      int               `json:"weight"`
			Axes        map[string]string `json:"axes"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Slug == "" || req.Name == "" {
			http.Error(w, "slug and name are required", http.StatusBadRequest)
			return
		}

		if len(req.Axes) == 0 {
			http.Error(w, "axes selection is required", http.StatusBadRequest)
			return
		}

		variant, err := vs.CreateVariant(req.Slug, req.Name, req.Description, req.Weight, req.Axes)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(variant)
	}
}

// handleVariantCreateWithSections creates a variant and copies sections from Control
func handleVariantCreateWithSections(vs *VariantService, cs *ContentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Slug        string            `json:"slug"`
			Name        string            `json:"name"`
			Description string            `json:"description"`
			Weight      int               `json:"weight"`
			Axes        map[string]string `json:"axes"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Slug == "" || req.Name == "" {
			http.Error(w, "slug and name are required", http.StatusBadRequest)
			return
		}

		if len(req.Axes) == 0 {
			http.Error(w, "axes selection is required", http.StatusBadRequest)
			return
		}

		variant, err := vs.CreateVariant(req.Slug, req.Name, req.Description, req.Weight, req.Axes)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Copy sections from Control variant (id=1) to give new variant default content
		if err := cs.CopySectionsFromVariant(1, int64(variant.ID)); err != nil {
			// Log error but don't fail - variant was created successfully
			logStructuredError("Failed to copy sections to new variant", map[string]interface{}{
				"variant_id":   variant.ID,
				"variant_slug": variant.Slug,
				"error":        err.Error(),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(variant)
	}
}

// handleVariantUpdate handles PATCH /api/v1/variants/{slug} (OT-P0-017: AB-CRUD)
func handleVariantUpdate(vs *VariantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		slug := r.URL.Path[len("/api/v1/variants/"):]
		if slug == "" {
			http.Error(w, "Variant slug required", http.StatusBadRequest)
			return
		}

		var req struct {
			Name        *string           `json:"name,omitempty"`
			Description *string           `json:"description,omitempty"`
			Weight      *int              `json:"weight,omitempty"`
			Axes        map[string]string `json:"axes,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		variant, err := vs.UpdateVariant(slug, req.Name, req.Description, req.Weight, req.Axes)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(variant)
	}
}

// handleVariantArchive handles POST /api/v1/variants/{slug}/archive (OT-P0-018: AB-ARCHIVE)
func handleVariantArchive(vs *VariantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		slug := r.URL.Path[len("/api/v1/variants/"):]
		slug = slug[:len(slug)-len("/archive")]

		if slug == "" {
			http.Error(w, "Variant slug required", http.StatusBadRequest)
			return
		}

		if err := vs.ArchiveVariant(slug); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Variant archived successfully",
			"slug":    slug,
		})
	}
}

// handleVariantDelete handles DELETE /api/v1/variants/{slug} (OT-P0-017: AB-CRUD)
func handleVariantDelete(vs *VariantService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		slug := r.URL.Path[len("/api/v1/variants/"):]
		if slug == "" {
			http.Error(w, "Variant slug required", http.StatusBadRequest)
			return
		}

		if err := vs.DeleteVariant(slug); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Variant deleted successfully",
			"slug":    slug,
		})
	}
}
