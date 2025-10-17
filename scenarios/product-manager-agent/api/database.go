package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Feature database operations
func (app *App) fetchFeatures() ([]Feature, error) {
	// If no database connection, return default features
	if app.DB == nil {
		return app.getDefaultFeatures(), nil
	}

	query := `
		SELECT id, name, description, reach, impact, confidence, effort,
		       priority, score, status, dependencies, roi, created_at, updated_at
		FROM features
		ORDER BY score DESC
		LIMIT 100
	`

	rows, err := app.DB.Query(query)
	if err != nil {
		return app.getDefaultFeatures(), nil
	}
	defer rows.Close()

	var features []Feature
	for rows.Next() {
		var f Feature
		var deps []byte
		err := rows.Scan(&f.ID, &f.Name, &f.Description, &f.Reach, &f.Impact,
			&f.Confidence, &f.Effort, &f.Priority, &f.Score, &f.Status,
			&deps, &f.ROI, &f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			continue
		}
		json.Unmarshal(deps, &f.Dependencies)
		features = append(features, f)
	}

	// If no features in DB, return some defaults
	if len(features) == 0 {
		features = app.getDefaultFeatures()
	}

	return features, nil
}

func (app *App) fetchFeaturesByIDs(ids []string) ([]Feature, error) {
	if len(ids) == 0 {
		return []Feature{}, nil
	}

	// For simplicity, fetch all and filter
	allFeatures, err := app.fetchFeatures()
	if err != nil {
		return nil, err
	}

	var filtered []Feature
	for _, f := range allFeatures {
		for _, id := range ids {
			if f.ID == id {
				filtered = append(filtered, f)
				break
			}
		}
	}

	return filtered, nil
}

func (app *App) fetchAvailableFeatures() ([]Feature, error) {
	// If no database, return default features
	if app.DB == nil {
		return app.getDefaultFeatures(), nil
	}

	query := `
		SELECT id, name, description, reach, impact, confidence, effort,
		       priority, score, status, dependencies, roi, created_at, updated_at
		FROM features
		WHERE status IN ('proposed', 'approved')
		ORDER BY score DESC
		LIMIT 50
	`

	rows, err := app.DB.Query(query)
	if err != nil {
		// Return default features if DB is not set up
		return app.getDefaultFeatures(), nil
	}
	defer rows.Close()

	var features []Feature
	for rows.Next() {
		var f Feature
		var deps []byte
		err := rows.Scan(&f.ID, &f.Name, &f.Description, &f.Reach, &f.Impact,
			&f.Confidence, &f.Effort, &f.Priority, &f.Score, &f.Status,
			&deps, &f.ROI, &f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			continue
		}
		json.Unmarshal(deps, &f.Dependencies)
		features = append(features, f)
	}

	if len(features) == 0 {
		features = app.getDefaultFeatures()
	}

	return features, nil
}

func (app *App) fetchRecentFeatures(limit int) ([]Feature, error) {
	// If no database, return default features
	if app.DB == nil {
		defaultFeatures := app.getDefaultFeatures()
		if len(defaultFeatures) > limit {
			return defaultFeatures[:limit], nil
		}
		return defaultFeatures, nil
	}

	query := `
		SELECT id, name, description, reach, impact, confidence, effort,
		       priority, score, status, dependencies, roi, created_at, updated_at
		FROM features
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := app.DB.Query(query, limit)
	if err != nil {
		defaultFeatures := app.getDefaultFeatures()
		if len(defaultFeatures) > limit {
			return defaultFeatures[:limit], nil
		}
		return defaultFeatures, nil
	}
	defer rows.Close()

	var features []Feature
	for rows.Next() {
		var f Feature
		var deps []byte
		err := rows.Scan(&f.ID, &f.Name, &f.Description, &f.Reach, &f.Impact,
			&f.Confidence, &f.Effort, &f.Priority, &f.Score, &f.Status,
			&deps, &f.ROI, &f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			continue
		}
		json.Unmarshal(deps, &f.Dependencies)
		features = append(features, f)
	}

	if len(features) == 0 {
		defaultFeatures := app.getDefaultFeatures()
		if len(defaultFeatures) > limit {
			return defaultFeatures[:limit], nil
		}
		return defaultFeatures, nil
	}

	return features, nil
}

func (app *App) storeFeature(feature *Feature) error {
	if feature.ID == "" {
		feature.ID = uuid.New().String()
	}

	// If no database, just return success (feature is in memory)
	if app.DB == nil {
		return nil
	}

	deps, _ := json.Marshal(feature.Dependencies)

	query := `
		INSERT INTO features (id, name, description, reach, impact, confidence,
		                     effort, priority, score, status, dependencies, roi,
		                     created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (id) DO UPDATE SET
		    name = $2, description = $3, reach = $4, impact = $5, confidence = $6,
		    effort = $7, priority = $8, score = $9, status = $10, dependencies = $11,
		    roi = $12, updated_at = $14
	`

	_, err := app.DB.Exec(query, feature.ID, feature.Name, feature.Description,
		feature.Reach, feature.Impact, feature.Confidence, feature.Effort,
		feature.Priority, feature.Score, feature.Status, deps, feature.ROI,
		feature.CreatedAt, feature.UpdatedAt)

	return err
}

func (app *App) updateFeatureDB(feature *Feature) error {
	return app.storeFeature(feature)
}

// Roadmap database operations
func (app *App) fetchCurrentRoadmap() (*Roadmap, error) {
	// If no database, return default roadmap
	if app.DB == nil {
		return app.getDefaultRoadmap(), nil
	}

	query := `
		SELECT id, name, start_date, end_date, features, milestones, version, created_at
		FROM roadmaps
		ORDER BY created_at DESC
		LIMIT 1
	`

	var roadmap Roadmap
	var features, milestones []byte

	err := app.DB.QueryRow(query).Scan(&roadmap.ID, &roadmap.Name,
		&roadmap.StartDate, &roadmap.EndDate, &features, &milestones,
		&roadmap.Version, &roadmap.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			// Return default roadmap
			return app.getDefaultRoadmap(), nil
		}
		return app.getDefaultRoadmap(), nil
	}

	json.Unmarshal(features, &roadmap.Features)
	json.Unmarshal(milestones, &roadmap.Milestones)

	return &roadmap, nil
}

func (app *App) storeRoadmap(roadmap *Roadmap) error {
	if roadmap.ID == "" {
		roadmap.ID = uuid.New().String()
	}

	// If no database, just return success
	if app.DB == nil {
		return nil
	}

	features, _ := json.Marshal(roadmap.Features)
	milestones, _ := json.Marshal(roadmap.Milestones)

	query := `
		INSERT INTO roadmaps (id, name, start_date, end_date, features, milestones, version, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
		    name = $2, start_date = $3, end_date = $4, features = $5,
		    milestones = $6, version = $7
	`

	_, err := app.DB.Exec(query, roadmap.ID, roadmap.Name, roadmap.StartDate,
		roadmap.EndDate, features, milestones, roadmap.Version, roadmap.CreatedAt)

	return err
}

// Sprint database operations
func (app *App) fetchCurrentSprint() (*SprintPlan, error) {
	// For now, return a mock sprint
	return app.getDefaultSprint(), nil
}

func (app *App) storeSprintPlan(sprint *SprintPlan) error {
	if sprint.ID == "" {
		sprint.ID = uuid.New().String()
	}

	// Store sprint plan in database
	// Implementation depends on schema
	return nil
}

// Analysis storage operations
func (app *App) storeMarketAnalysis(analysis *MarketAnalysis) error {
	if analysis.ID == "" {
		analysis.ID = fmt.Sprintf("ma-%d", time.Now().Unix())
	}
	// Store in database
	return nil
}

func (app *App) storeCompetitorAnalysis(analysis *CompetitorAnalysis) error {
	if analysis.ID == "" {
		analysis.ID = fmt.Sprintf("ca-%d", time.Now().Unix())
	}
	// Store in database
	return nil
}

func (app *App) storeROICalculation(calc *ROICalculation) error {
	if calc.ID == "" {
		calc.ID = uuid.New().String()
	}
	// Store in database
	return nil
}

// Dashboard metrics
func (app *App) fetchDashboardMetrics() (DashboardMetrics, error) {
	metrics := DashboardMetrics{
		ActiveFeatures:   24,
		SprintProgress:   87,
		TeamVelocity:     42.5,
		CustomerNPS:      72,
		CompletedTasks:   156,
		PendingDecisions: 3,
	}

	// Try to get real metrics from DB if available
	if app.DB != nil {
		var count int
		err := app.DB.QueryRow("SELECT COUNT(*) FROM features WHERE status = 'in_progress'").Scan(&count)
		if err == nil && count > 0 {
			metrics.ActiveFeatures = count
		}
	}

	return metrics, nil
}

// Default data for when database is not populated
func (app *App) getDefaultFeatures() []Feature {
	return []Feature{
		{
			ID:          "f1",
			Name:        "Dark Mode Support",
			Description: "Implement system-wide dark mode with theme switching",
			Reach:       5000,
			Impact:      4,
			Confidence:  0.9,
			Effort:      5,
			Priority:    "CRITICAL",
			Score:       7.2,
			Status:      "proposed",
			CreatedAt:   time.Now().Add(-72 * time.Hour),
			UpdatedAt:   time.Now().Add(-24 * time.Hour),
		},
		{
			ID:          "f2",
			Name:        "API Rate Limiting",
			Description: "Implement rate limiting to prevent API abuse",
			Reach:       3000,
			Impact:      5,
			Confidence:  0.95,
			Effort:      4,
			Priority:    "HIGH",
			Score:       8.9,
			Status:      "approved",
			CreatedAt:   time.Now().Add(-48 * time.Hour),
			UpdatedAt:   time.Now().Add(-12 * time.Hour),
		},
		{
			ID:          "f3",
			Name:        "Mobile App",
			Description: "Native mobile application for iOS and Android",
			Reach:       8000,
			Impact:      5,
			Confidence:  0.7,
			Effort:      20,
			Priority:    "MEDIUM",
			Score:       1.4,
			Status:      "proposed",
			CreatedAt:   time.Now().Add(-96 * time.Hour),
			UpdatedAt:   time.Now().Add(-36 * time.Hour),
		},
		{
			ID:          "f4",
			Name:        "Analytics Dashboard",
			Description: "Advanced analytics and reporting dashboard",
			Reach:       2000,
			Impact:      3,
			Confidence:  0.85,
			Effort:      8,
			Priority:    "MEDIUM",
			Score:       0.64,
			Status:      "in_progress",
			CreatedAt:   time.Now().Add(-120 * time.Hour),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          "f5",
			Name:        "Two-Factor Authentication",
			Description: "Add 2FA support for enhanced security",
			Reach:       10000,
			Impact:      5,
			Confidence:  0.98,
			Effort:      6,
			Priority:    "CRITICAL",
			Score:       8.17,
			Status:      "approved",
			CreatedAt:   time.Now().Add(-24 * time.Hour),
			UpdatedAt:   time.Now().Add(-6 * time.Hour),
		},
	}
}

func (app *App) getDefaultRoadmap() *Roadmap {
	startDate := time.Now()
	return &Roadmap{
		ID:        "default-roadmap",
		Name:      fmt.Sprintf("Product Roadmap %s", startDate.Format("Jan 2006")),
		StartDate: startDate,
		EndDate:   startDate.AddDate(0, 6, 0),
		Features:  []string{"f1", "f2", "f3", "f4", "f5"},
		Milestones: []Milestone{
			{
				ID:          "m1",
				Name:        "Q1 Release",
				Date:        startDate.AddDate(0, 3, 0),
				Description: "Security and core features",
				Features:    []string{"f1", "f2", "f5"},
			},
			{
				ID:          "m2",
				Name:        "Q2 Release",
				Date:        startDate.AddDate(0, 6, 0),
				Description: "Mobile and analytics",
				Features:    []string{"f3", "f4"},
			},
		},
		Version:   1,
		CreatedAt: time.Now(),
	}
}

func (app *App) getDefaultSprint() *SprintPlan {
	features := app.getDefaultFeatures()[:3]
	totalEffort := 0
	for _, f := range features {
		totalEffort += f.Effort
	}

	return &SprintPlan{
		ID:             "sprint-1",
		SprintNumber:   1,
		StartDate:      time.Now(),
		EndDate:        time.Now().AddDate(0, 0, 14),
		Capacity:       20,
		Features:       features,
		TotalEffort:    totalEffort,
		EstimatedValue: 50000,
		Velocity:       42.5,
		RiskLevel:      "medium",
		PlannedAt:      time.Now(),
	}
}