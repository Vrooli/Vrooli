package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
)

// DatabaseService handles all database operations
type DatabaseService struct {
	db *sql.DB
}

func NewDatabaseService(database *sql.DB) *DatabaseService {
	return &DatabaseService{db: database}
}

// unmarshalJSONOrDefault attempts to unmarshal JSON into target, returning empty map on failure
// This provides consistent error handling for JSON fields throughout the database layer
func unmarshalJSONOrDefault(jsonStr string, target *map[string]interface{}, context string) {
	*target = make(map[string]interface{})
	if jsonStr == "" || jsonStr == "{}" {
		return
	}
	if err := json.Unmarshal([]byte(jsonStr), target); err != nil {
		log.Printf("Warning: Failed to unmarshal %s: %v (using empty map)", context, err)
	}
}

func (ds *DatabaseService) CreateSaaSScenario(scenario *SaaSScenario) error {
	query := `
		INSERT INTO saas_scenarios (id, scenario_name, display_name, description, saas_type,
			industry, revenue_potential, has_landing_page, landing_page_url, last_scan,
			confidence_score, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (scenario_name) DO UPDATE SET
			display_name = $3, description = $4, saas_type = $5, industry = $6,
			revenue_potential = $7, confidence_score = $11, metadata = $12, last_scan = $10
	`

	metadataJSON, err := json.Marshal(scenario.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal scenario metadata: %w", err)
	}

	_, err = ds.db.Exec(query, scenario.ID, scenario.ScenarioName, scenario.DisplayName,
		scenario.Description, scenario.SaaSType, scenario.Industry, scenario.RevenuePotential,
		scenario.HasLandingPage, scenario.LandingPageURL, scenario.LastScan,
		scenario.ConfidenceScore, string(metadataJSON))

	return err
}

func (ds *DatabaseService) GetSaaSScenarios() ([]SaaSScenario, error) {
	query := `
		SELECT id, scenario_name, display_name, description, saas_type, industry,
			revenue_potential, has_landing_page, landing_page_url, last_scan,
			confidence_score, metadata
		FROM saas_scenarios
		ORDER BY confidence_score DESC, last_scan DESC
	`

	rows, err := ds.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scenarios []SaaSScenario
	for rows.Next() {
		var s SaaSScenario
		var metadataJSON string

		err := rows.Scan(&s.ID, &s.ScenarioName, &s.DisplayName, &s.Description,
			&s.SaaSType, &s.Industry, &s.RevenuePotential, &s.HasLandingPage,
			&s.LandingPageURL, &s.LastScan, &s.ConfidenceScore, &metadataJSON)
		if err != nil {
			log.Printf("Warning: Failed to scan scenario row: %v (skipping row)", err)
			continue
		}

		unmarshalJSONOrDefault(metadataJSON, &s.Metadata, fmt.Sprintf("metadata for scenario %s", s.ID))
		scenarios = append(scenarios, s)
	}

	return scenarios, nil
}

func (ds *DatabaseService) CreateLandingPage(page *LandingPage) error {
	query := `
		INSERT INTO landing_pages (id, scenario_id, template_id, variant, title, description,
			content, seo_metadata, performance_metrics, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	contentJSON, err := json.Marshal(page.Content)
	if err != nil {
		return fmt.Errorf("failed to marshal landing page content: %w", err)
	}

	seoJSON, err := json.Marshal(page.SEOMetadata)
	if err != nil {
		return fmt.Errorf("failed to marshal SEO metadata: %w", err)
	}

	metricsJSON, err := json.Marshal(page.PerformanceMetrics)
	if err != nil {
		return fmt.Errorf("failed to marshal performance metrics: %w", err)
	}

	_, err = ds.db.Exec(query, page.ID, page.ScenarioID, page.TemplateID, page.Variant,
		page.Title, page.Description, string(contentJSON), string(seoJSON),
		string(metricsJSON), page.Status, page.CreatedAt, page.UpdatedAt)

	return err
}

func (ds *DatabaseService) GetLandingPageByID(id string) (*LandingPage, error) {
	query := `
		SELECT id, scenario_id, template_id, variant, title, description,
			content, seo_metadata, performance_metrics, status, created_at, updated_at
		FROM landing_pages
		WHERE id = $1
	`

	var page LandingPage
	var contentJSON, seoJSON, metricsJSON string

	err := ds.db.QueryRow(query, id).Scan(
		&page.ID, &page.ScenarioID, &page.TemplateID, &page.Variant,
		&page.Title, &page.Description, &contentJSON, &seoJSON,
		&metricsJSON, &page.Status, &page.CreatedAt, &page.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON fields with consistent error handling
	unmarshalJSONOrDefault(contentJSON, &page.Content, fmt.Sprintf("content for landing page %s", id))
	unmarshalJSONOrDefault(seoJSON, &page.SEOMetadata, fmt.Sprintf("SEO metadata for landing page %s", id))
	unmarshalJSONOrDefault(metricsJSON, &page.PerformanceMetrics, fmt.Sprintf("performance metrics for landing page %s", id))

	return &page, nil
}

func (ds *DatabaseService) GetTemplates(category, saasType string) ([]Template, error) {
	query := `
		SELECT id, name, category, saas_type, industry, html_content, css_content,
			js_content, config_schema, preview_url, usage_count, rating, created_at
		FROM templates
		WHERE ($1 = '' OR category = $1) AND ($2 = '' OR saas_type = $2)
		ORDER BY usage_count DESC, rating DESC
	`

	rows, err := ds.db.Query(query, category, saasType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []Template
	for rows.Next() {
		var t Template
		var configJSON string

		err := rows.Scan(&t.ID, &t.Name, &t.Category, &t.SaaSType, &t.Industry,
			&t.HTMLContent, &t.CSSContent, &t.JSContent, &configJSON,
			&t.PreviewURL, &t.UsageCount, &t.Rating, &t.CreatedAt)
		if err != nil {
			log.Printf("Warning: Failed to scan template row: %v (skipping row)", err)
			continue
		}

		unmarshalJSONOrDefault(configJSON, &t.ConfigSchema, fmt.Sprintf("config schema for template %s", t.ID))
		templates = append(templates, t)
	}

	return templates, nil
}

func (ds *DatabaseService) GetActiveABTestsCount() (int, error) {
	query := `
		SELECT COUNT(*)
		FROM (
			SELECT scenario_id
			FROM landing_pages
			WHERE status = 'active'
			GROUP BY scenario_id
			HAVING COUNT(*) > 1
		) AS active_tests
	`
	var count sql.NullInt64
	err := ds.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, err
	}
	if count.Valid {
		return int(count.Int64), nil
	}
	return 0, nil
}

func (ds *DatabaseService) GetAverageConversionRate() (float64, error) {
	query := `
		SELECT AVG(conversion_rate)
		FROM analytics_summary
		WHERE date_range_end >= NOW() - INTERVAL '30 days'
	`
	var avgRate sql.NullFloat64
	err := ds.db.QueryRow(query).Scan(&avgRate)
	if err != nil {
		return 0.0, err
	}
	if avgRate.Valid {
		return avgRate.Float64, nil
	}
	return 0.0, nil
}
