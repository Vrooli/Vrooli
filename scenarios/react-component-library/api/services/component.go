package services

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/vrooli/scenarios/react-component-library/models"
)

var (
	ErrComponentNotFound = errors.New("component not found")
	ErrInvalidCode       = errors.New("invalid component code")
)

type ComponentService struct {
	db *sql.DB
}

func NewComponentService(db *sql.DB) *ComponentService {
	return &ComponentService{
		db: db,
	}
}

// CreateComponent creates a new component in the database
func (s *ComponentService) CreateComponent(component *models.Component) error {
	query := `
		INSERT INTO components (
			id, name, category, description, code, props_schema,
			created_at, updated_at, version, author, usage_count,
			tags, is_active, dependencies, example_usage
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)`

	_, err := s.db.Exec(
		query,
		component.ID,
		component.Name,
		component.Category,
		component.Description,
		component.Code,
		component.PropsSchema,
		component.CreatedAt,
		component.UpdatedAt,
		component.Version,
		component.Author,
		component.UsageCount,
		strings.Join(component.Tags, ","),
		component.IsActive,
		strings.Join(component.Dependencies, ","),
		component.ExampleUsage,
	)

	return err
}

// GetComponent retrieves a component by ID
func (s *ComponentService) GetComponent(id uuid.UUID) (*models.Component, error) {
	query := `
		SELECT 
			id, name, category, description, code, props_schema,
			created_at, updated_at, version, author, usage_count,
			accessibility_score, performance_metrics, tags, is_active,
			dependencies, screenshots, example_usage
		FROM components 
		WHERE id = $1 AND is_active = true`

	row := s.db.QueryRow(query, id)

	var component models.Component
	var tagsStr, dependenciesStr, screenshotsStr string
	
	err := row.Scan(
		&component.ID,
		&component.Name,
		&component.Category,
		&component.Description,
		&component.Code,
		&component.PropsSchema,
		&component.CreatedAt,
		&component.UpdatedAt,
		&component.Version,
		&component.Author,
		&component.UsageCount,
		&component.AccessibilityScore,
		&component.PerformanceMetrics,
		&tagsStr,
		&component.IsActive,
		&dependenciesStr,
		&screenshotsStr,
		&component.ExampleUsage,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrComponentNotFound
		}
		return nil, err
	}

	// Parse comma-separated strings back to slices
	if tagsStr != "" {
		component.Tags = strings.Split(tagsStr, ",")
	}
	if dependenciesStr != "" {
		component.Dependencies = strings.Split(dependenciesStr, ",")
	}
	if screenshotsStr != "" {
		component.Screenshots = strings.Split(screenshotsStr, ",")
	}

	return &component, nil
}

// ListComponents retrieves components with optional filtering
func (s *ComponentService) ListComponents(category string, limit, offset int) ([]models.Component, int, error) {
	var components []models.Component
	var total int

	// Build WHERE clause
	whereClause := "WHERE is_active = true"
	args := []interface{}{}
	argIndex := 1

	if category != "" {
		whereClause += fmt.Sprintf(" AND category = $%d", argIndex)
		args = append(args, category)
		argIndex++
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM components %s", whereClause)
	err := s.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get components
	query := fmt.Sprintf(`
		SELECT 
			id, name, category, description, code, props_schema,
			created_at, updated_at, version, author, usage_count,
			accessibility_score, performance_metrics, tags, is_active,
			dependencies, screenshots, example_usage
		FROM components 
		%s 
		ORDER BY created_at DESC 
		LIMIT $%d OFFSET $%d`,
		whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var component models.Component
		var tagsStr, dependenciesStr, screenshotsStr string

		err := rows.Scan(
			&component.ID,
			&component.Name,
			&component.Category,
			&component.Description,
			&component.Code,
			&component.PropsSchema,
			&component.CreatedAt,
			&component.UpdatedAt,
			&component.Version,
			&component.Author,
			&component.UsageCount,
			&component.AccessibilityScore,
			&component.PerformanceMetrics,
			&tagsStr,
			&component.IsActive,
			&dependenciesStr,
			&screenshotsStr,
			&component.ExampleUsage,
		)
		if err != nil {
			return nil, 0, err
		}

		// Parse comma-separated strings
		if tagsStr != "" {
			component.Tags = strings.Split(tagsStr, ",")
		}
		if dependenciesStr != "" {
			component.Dependencies = strings.Split(dependenciesStr, ",")
		}
		if screenshotsStr != "" {
			component.Screenshots = strings.Split(screenshotsStr, ",")
		}

		components = append(components, component)
	}

	return components, total, nil
}

// UpdateComponent updates an existing component
func (s *ComponentService) UpdateComponent(component *models.Component) (*models.Component, error) {
	query := `
		UPDATE components SET 
			name = $2, category = $3, description = $4, code = $5,
			props_schema = $6, updated_at = $7, tags = $8,
			dependencies = $9, example_usage = $10
		WHERE id = $1 AND is_active = true
		RETURNING 
			id, name, category, description, code, props_schema,
			created_at, updated_at, version, author, usage_count,
			accessibility_score, performance_metrics, tags, is_active,
			dependencies, screenshots, example_usage`

	row := s.db.QueryRow(
		query,
		component.ID,
		component.Name,
		component.Category,
		component.Description,
		component.Code,
		component.PropsSchema,
		component.UpdatedAt,
		strings.Join(component.Tags, ","),
		strings.Join(component.Dependencies, ","),
		component.ExampleUsage,
	)

	var updatedComponent models.Component
	var tagsStr, dependenciesStr, screenshotsStr string

	err := row.Scan(
		&updatedComponent.ID,
		&updatedComponent.Name,
		&updatedComponent.Category,
		&updatedComponent.Description,
		&updatedComponent.Code,
		&updatedComponent.PropsSchema,
		&updatedComponent.CreatedAt,
		&updatedComponent.UpdatedAt,
		&updatedComponent.Version,
		&updatedComponent.Author,
		&updatedComponent.UsageCount,
		&updatedComponent.AccessibilityScore,
		&updatedComponent.PerformanceMetrics,
		&tagsStr,
		&updatedComponent.IsActive,
		&dependenciesStr,
		&screenshotsStr,
		&updatedComponent.ExampleUsage,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrComponentNotFound
		}
		return nil, err
	}

	// Parse comma-separated strings
	if tagsStr != "" {
		updatedComponent.Tags = strings.Split(tagsStr, ",")
	}
	if dependenciesStr != "" {
		updatedComponent.Dependencies = strings.Split(dependenciesStr, ",")
	}
	if screenshotsStr != "" {
		updatedComponent.Screenshots = strings.Split(screenshotsStr, ",")
	}

	return &updatedComponent, nil
}

// DeleteComponent soft-deletes a component
func (s *ComponentService) DeleteComponent(id uuid.UUID) error {
	query := `UPDATE components SET is_active = false WHERE id = $1 AND is_active = true`
	result, err := s.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrComponentNotFound
	}

	return nil
}

// ValidateComponentCode performs basic validation on React component code
func (s *ComponentService) ValidateComponentCode(code string) error {
	// Basic validation - check for React component patterns
	code = strings.TrimSpace(code)
	
	if code == "" {
		return ErrInvalidCode
	}

	// Check for basic React patterns
	validPatterns := []string{
		"const",
		"function",
		"export",
		"return",
	}

	foundPatterns := 0
	for _, pattern := range validPatterns {
		if strings.Contains(code, pattern) {
			foundPatterns++
		}
	}

	if foundPatterns < 2 {
		return ErrInvalidCode
	}

	return nil
}

// RecordUsage records component usage for analytics
func (s *ComponentService) RecordUsage(componentID uuid.UUID, userAgent, context string) error {
	query := `
		INSERT INTO usage_analytics (id, component_id, scenario, context, used_at, user_agent)
		VALUES ($1, $2, $3, $4, NOW(), $5)`

	_, err := s.db.Exec(query, uuid.New(), componentID, "api", context, userAgent)
	if err != nil {
		// Don't fail the main operation if analytics fails
		fmt.Printf("Warning: Failed to record usage analytics: %v\n", err)
	}

	// Update usage count
	updateQuery := `UPDATE components SET usage_count = usage_count + 1 WHERE id = $1`
	s.db.Exec(updateQuery, componentID)

	return nil
}

// ExportComponent exports a component in the specified format
func (s *ComponentService) ExportComponent(id uuid.UUID, req models.ComponentExportRequest) (*models.ComponentExportResponse, error) {
	component, err := s.GetComponent(id)
	if err != nil {
		return nil, err
	}

	response := &models.ComponentExportResponse{
		ExportType:       req.Format,
		Dependencies:     component.Dependencies,
		UsageInstructions: generateUsageInstructions(component, req.Format),
	}

	switch req.Format {
	case models.ExportFormatRawCode:
		response.Code = component.Code
	case models.ExportFormatNPMPackage:
		// Generate npm package structure
		response.Code = generateNPMPackage(component)
		response.UsageInstructions = "npm install && import { " + component.Name + " } from './" + component.Name + "';"
	case models.ExportFormatCDN:
		// Generate CDN-ready format (would upload to MinIO in real implementation)
		response.ExportURL = fmt.Sprintf("https://cdn.vrooli.com/components/%s/%s.js", component.ID, component.Name)
		response.UsageInstructions = `<script src="` + response.ExportURL + `"></script>`
	case models.ExportFormatZip:
		// Generate ZIP package (would create actual ZIP in real implementation)
		response.ExportURL = fmt.Sprintf("/api/v1/components/%s/download?format=zip", id)
		response.UsageInstructions = "Download and extract the ZIP file, then import the component."
	}

	return response, nil
}

// GetComponentVersions retrieves all versions of a component
func (s *ComponentService) GetComponentVersions(componentID uuid.UUID) ([]models.ComponentVersion, error) {
	query := `
		SELECT id, component_id, version, code, changelog, breaking_changes, deprecated, created_at
		FROM component_versions 
		WHERE component_id = $1 
		ORDER BY created_at DESC`

	rows, err := s.db.Query(query, componentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []models.ComponentVersion
	for rows.Next() {
		var version models.ComponentVersion
		var breakingChangesStr string

		err := rows.Scan(
			&version.ID,
			&version.ComponentID,
			&version.Version,
			&version.Code,
			&version.Changelog,
			&breakingChangesStr,
			&version.Deprecated,
			&version.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if breakingChangesStr != "" {
			version.BreakingChanges = strings.Split(breakingChangesStr, ",")
		}

		versions = append(versions, version)
	}

	return versions, nil
}

// GetUsageAnalytics retrieves usage analytics for components
func (s *ComponentService) GetUsageAnalytics(componentID, startDate, endDate string) (interface{}, error) {
	// Simplified implementation - would have more complex analytics in production
	query := `
		SELECT COUNT(*) as usage_count, DATE(used_at) as date
		FROM usage_analytics 
		WHERE 1=1`
	
	args := []interface{}{}
	argIndex := 1

	if componentID != "" {
		query += fmt.Sprintf(" AND component_id = $%d", argIndex)
		id, _ := uuid.Parse(componentID)
		args = append(args, id)
		argIndex++
	}

	if startDate != "" {
		query += fmt.Sprintf(" AND used_at >= $%d", argIndex)
		args = append(args, startDate)
		argIndex++
	}

	if endDate != "" {
		query += fmt.Sprintf(" AND used_at <= $%d", argIndex)
		args = append(args, endDate)
		argIndex++
	}

	query += " GROUP BY DATE(used_at) ORDER BY date DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	analytics := make([]map[string]interface{}, 0)
	for rows.Next() {
		var count int
		var date string
		err := rows.Scan(&count, &date)
		if err != nil {
			return nil, err
		}
		analytics = append(analytics, map[string]interface{}{
			"date":  date,
			"count": count,
		})
	}

	return analytics, nil
}

// GetPopularComponents retrieves the most popular components
func (s *ComponentService) GetPopularComponents(limit int, timeframe string) ([]models.Component, error) {
	query := `
		SELECT c.id, c.name, c.category, c.description, c.usage_count,
			   c.accessibility_score, c.created_at
		FROM components c
		WHERE c.is_active = true
		ORDER BY c.usage_count DESC
		LIMIT $1`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var components []models.Component
	for rows.Next() {
		var component models.Component
		err := rows.Scan(
			&component.ID,
			&component.Name,
			&component.Category,
			&component.Description,
			&component.UsageCount,
			&component.AccessibilityScore,
			&component.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		components = append(components, component)
	}

	return components, nil
}

// Helper functions

func generateUsageInstructions(component *models.Component, format models.ExportFormat) string {
	switch format {
	case models.ExportFormatRawCode:
		return fmt.Sprintf("Copy and paste the code into your project. Usage: <%s />", component.Name)
	case models.ExportFormatNPMPackage:
		return fmt.Sprintf("Install the package and import: import { %s } from './%s';", component.Name, component.Name)
	case models.ExportFormatCDN:
		return "Include the script tag in your HTML, then use the component in your React application."
	default:
		return "Please refer to the component documentation for usage instructions."
	}
}

func generateNPMPackage(component *models.Component) string {
	return fmt.Sprintf(`{
  "name": "%s",
  "version": "%s", 
  "description": "%s",
  "main": "index.js",
  "dependencies": %s
}

// index.js
%s

export default %s;`,
		strings.ToLower(component.Name),
		component.Version,
		component.Description,
		`{}`, // Would parse actual dependencies
		component.Code,
		component.Name,
	)
}