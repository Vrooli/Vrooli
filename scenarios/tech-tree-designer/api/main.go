package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type TechTree struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Version     string    `json:"version" db:"version"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type Sector struct {
	ID                 string  `json:"id" db:"id"`
	TreeID             string  `json:"tree_id" db:"tree_id"`
	Name               string  `json:"name" db:"name"`
	Category           string  `json:"category" db:"category"`
	Description        string  `json:"description" db:"description"`
	ProgressPercentage float64 `json:"progress_percentage" db:"progress_percentage"`
	PositionX          float64 `json:"position_x" db:"position_x"`
	PositionY          float64 `json:"position_y" db:"position_y"`
	Color              string  `json:"color" db:"color"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
	Stages             []ProgressionStage `json:"stages,omitempty"`
}

type ProgressionStage struct {
	ID                 string                  `json:"id" db:"id"`
	SectorID           string                  `json:"sector_id" db:"sector_id"`
	StageType          string                  `json:"stage_type" db:"stage_type"`
	StageOrder         int                     `json:"stage_order" db:"stage_order"`
	Name               string                  `json:"name" db:"name"`
	Description        string                  `json:"description" db:"description"`
	ProgressPercentage float64                 `json:"progress_percentage" db:"progress_percentage"`
	Examples           json.RawMessage         `json:"examples" db:"examples"`
	PositionX          float64                 `json:"position_x" db:"position_x"`
	PositionY          float64                 `json:"position_y" db:"position_y"`
	CreatedAt          time.Time               `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time               `json:"updated_at" db:"updated_at"`
	ScenarioMappings   []ScenarioMapping       `json:"scenario_mappings,omitempty"`
	Dependencies       []StageDependency       `json:"dependencies,omitempty"`
}

type ScenarioMapping struct {
	ID                 string    `json:"id" db:"id"`
	ScenarioName       string    `json:"scenario_name" db:"scenario_name"`
	StageID            string    `json:"stage_id" db:"stage_id"`
	ContributionWeight float64   `json:"contribution_weight" db:"contribution_weight"`
	CompletionStatus   string    `json:"completion_status" db:"completion_status"`
	Priority           int       `json:"priority" db:"priority"`
	EstimatedImpact    float64   `json:"estimated_impact" db:"estimated_impact"`
	LastStatusCheck    time.Time `json:"last_status_check" db:"last_status_check"`
	Notes              string    `json:"notes" db:"notes"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

type StageDependency struct {
	ID                   string  `json:"id" db:"id"`
	DependentStageID     string  `json:"dependent_stage_id" db:"dependent_stage_id"`
	PrerequisiteStageID  string  `json:"prerequisite_stage_id" db:"prerequisite_stage_id"`
	DependencyType       string  `json:"dependency_type" db:"dependency_type"`
	DependencyStrength   float64 `json:"dependency_strength" db:"dependency_strength"`
	Description          string  `json:"description" db:"description"`
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
}

type StrategicMilestone struct {
	ID                      string          `json:"id" db:"id"`
	TreeID                  string          `json:"tree_id" db:"tree_id"`
	Name                    string          `json:"name" db:"name"`
	Description             string          `json:"description" db:"description"`
	MilestoneType           string          `json:"milestone_type" db:"milestone_type"`
	RequiredSectors         json.RawMessage `json:"required_sectors" db:"required_sectors"`
	RequiredStages          json.RawMessage `json:"required_stages" db:"required_stages"`
	CompletionPercentage    float64         `json:"completion_percentage" db:"completion_percentage"`
	EstimatedCompletionDate *time.Time      `json:"estimated_completion_date" db:"estimated_completion_date"`
	ConfidenceLevel         float64         `json:"confidence_level" db:"confidence_level"`
	BusinessValueEstimate   int64           `json:"business_value_estimate" db:"business_value_estimate"`
	CreatedAt               time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time       `json:"updated_at" db:"updated_at"`
}

type StrategicRecommendation struct {
	Scenario         string  `json:"scenario"`
	PriorityScore    float64 `json:"priority_score"`
	ImpactMultiplier float64 `json:"impact_multiplier"`
	Reasoning        string  `json:"reasoning"`
}

type AnalysisRequest struct {
	CurrentResources int      `json:"current_resources"`
	TimeHorizon      int      `json:"time_horizon"`
	PrioritySectors  []string `json:"priority_sectors"`
}

type AnalysisResponse struct {
	Recommendations     []StrategicRecommendation `json:"recommendations"`
	ProjectedTimeline   ProjectedTimeline         `json:"projected_timeline"`
	BottleneckAnalysis  []string                  `json:"bottleneck_analysis"`
	CrossSectorImpacts  []CrossSectorImpact       `json:"cross_sector_impacts"`
}

type ProjectedTimeline struct {
	Milestones []MilestoneProjection `json:"milestones"`
}

type MilestoneProjection struct {
	Name                string    `json:"name"`
	EstimatedCompletion time.Time `json:"estimated_completion"`
	Confidence          float64   `json:"confidence"`
}

type CrossSectorImpact struct {
	SourceSector string  `json:"source_sector"`
	TargetSector string  `json:"target_sector"`
	ImpactScore  float64 `json:"impact_score"`
	Description  string  `json:"description"`
}

var db *sql.DB

func initDB() error {
	_ = godotenv.Load()
	
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}
	
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "vrooli"
	}

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	var err error
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		return err
	}

	return db.Ping()
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start tech-tree-designer

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Initialize database
	if err := initDB(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Initialize Gin router
	r := gin.Default()

	// Add CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "tech-tree-designer"})
	})

	// API routes
	api := r.Group("/api/v1")
	{
		// Tech tree routes
		api.GET("/tech-tree", getTechTree)
		api.GET("/tech-tree/sectors", getSectors)
		api.GET("/tech-tree/sectors/:id", getSector)
		api.GET("/tech-tree/stages/:id", getStage)
		
		// Progress tracking routes
		api.GET("/progress/scenarios", getScenarioMappings)
		api.POST("/progress/scenarios", updateScenarioMapping)
		api.PUT("/progress/scenarios/:scenario", updateScenarioStatus)
		
		// Strategic analysis routes
		api.POST("/tech-tree/analyze", analyzeStrategicPath)
		api.GET("/milestones", getStrategicMilestones)
		api.GET("/recommendations", getRecommendations)
		
		// Dependencies and connections
		api.GET("/dependencies", getDependencies)
		api.GET("/connections", getCrossSectorConnections)
	}

	// Get port from environment or default
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Tech Tree Designer API starting on port %s", port)
	log.Printf("🌟 Strategic Intelligence System ready for superintelligence guidance")
	
	r.Run(":" + port)
}

// Get the main tech tree
func getTechTree(c *gin.Context) {
	var tree TechTree
	err := db.QueryRow(`
		SELECT id, name, description, version, is_active, created_at, updated_at 
		FROM tech_trees WHERE is_active = true ORDER BY created_at DESC LIMIT 1
	`).Scan(&tree.ID, &tree.Name, &tree.Description, &tree.Version, &tree.IsActive, 
		&tree.CreatedAt, &tree.UpdatedAt)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tech tree"})
		return
	}
	
	c.JSON(http.StatusOK, tree)
}

// Get all sectors with their progress
func getSectors(c *gin.Context) {
	rows, err := db.Query(`
		SELECT id, tree_id, name, category, description, progress_percentage,
			   position_x, position_y, color, created_at, updated_at
		FROM sectors 
		ORDER BY category, name
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sectors"})
		return
	}
	defer rows.Close()

	var sectors []Sector
	for rows.Next() {
		var sector Sector
		err := rows.Scan(&sector.ID, &sector.TreeID, &sector.Name, &sector.Category,
			&sector.Description, &sector.ProgressPercentage, &sector.PositionX,
			&sector.PositionY, &sector.Color, &sector.CreatedAt, &sector.UpdatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan sector"})
			return
		}
		
		// Load stages for each sector
		stages, err := getStagesForSector(sector.ID)
		if err == nil {
			sector.Stages = stages
		}
		
		sectors = append(sectors, sector)
	}

	c.JSON(http.StatusOK, gin.H{"sectors": sectors})
}

// Get specific sector with detailed stages
func getSector(c *gin.Context) {
	sectorID := c.Param("id")

	var sector Sector
	err := db.QueryRow(`
		SELECT id, tree_id, name, category, description, progress_percentage,
			   position_x, position_y, color, created_at, updated_at
		FROM sectors WHERE id = $1
	`, sectorID).Scan(&sector.ID, &sector.TreeID, &sector.Name, &sector.Category,
		&sector.Description, &sector.ProgressPercentage, &sector.PositionX,
		&sector.PositionY, &sector.Color, &sector.CreatedAt, &sector.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Sector not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sector"})
		}
		return
	}

	// Load stages with scenario mappings
	stages, err := getStagesForSector(sectorID)
	if err == nil {
		sector.Stages = stages
	}

	c.JSON(http.StatusOK, sector)
}

// Get specific stage with detailed info
func getStage(c *gin.Context) {
	stageID := c.Param("id")

	var stage ProgressionStage
	err := db.QueryRow(`
		SELECT id, sector_id, stage_type, stage_order, name, description,
			   progress_percentage, examples, position_x, position_y, created_at, updated_at
		FROM progression_stages WHERE id = $1
	`, stageID).Scan(&stage.ID, &stage.SectorID, &stage.StageType, &stage.StageOrder,
		&stage.Name, &stage.Description, &stage.ProgressPercentage, &stage.Examples,
		&stage.PositionX, &stage.PositionY, &stage.CreatedAt, &stage.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Stage not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stage"})
		}
		return
	}

	// Load scenario mappings and dependencies
	mappings, err := getScenarioMappingsForStage(stageID)
	if err == nil {
		stage.ScenarioMappings = mappings
	}

	c.JSON(http.StatusOK, stage)
}

// Helper function to get stages for a sector
func getStagesForSector(sectorID string) ([]ProgressionStage, error) {
	rows, err := db.Query(`
		SELECT id, sector_id, stage_type, stage_order, name, description,
			   progress_percentage, examples, position_x, position_y, created_at, updated_at
		FROM progression_stages 
		WHERE sector_id = $1
		ORDER BY stage_order
	`, sectorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stages []ProgressionStage
	for rows.Next() {
		var stage ProgressionStage
		err := rows.Scan(&stage.ID, &stage.SectorID, &stage.StageType, &stage.StageOrder,
			&stage.Name, &stage.Description, &stage.ProgressPercentage, &stage.Examples,
			&stage.PositionX, &stage.PositionY, &stage.CreatedAt, &stage.UpdatedAt)
		if err != nil {
			return nil, err
		}
		
		// Load scenario mappings for each stage
		mappings, err := getScenarioMappingsForStage(stage.ID)
		if err == nil {
			stage.ScenarioMappings = mappings
		}
		
		stages = append(stages, stage)
	}
	
	return stages, nil
}

// Helper function to get scenario mappings for a stage
func getScenarioMappingsForStage(stageID string) ([]ScenarioMapping, error) {
	rows, err := db.Query(`
		SELECT id, scenario_name, stage_id, contribution_weight, completion_status,
			   priority, estimated_impact, last_status_check, notes, created_at, updated_at
		FROM scenario_mappings
		WHERE stage_id = $1
		ORDER BY priority, estimated_impact DESC
	`, stageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mappings []ScenarioMapping
	for rows.Next() {
		var mapping ScenarioMapping
		err := rows.Scan(&mapping.ID, &mapping.ScenarioName, &mapping.StageID,
			&mapping.ContributionWeight, &mapping.CompletionStatus, &mapping.Priority,
			&mapping.EstimatedImpact, &mapping.LastStatusCheck, &mapping.Notes,
			&mapping.CreatedAt, &mapping.UpdatedAt)
		if err != nil {
			return nil, err
		}
		mappings = append(mappings, mapping)
	}
	
	return mappings, nil
}

// Get all scenario mappings
func getScenarioMappings(c *gin.Context) {
	rows, err := db.Query(`
		SELECT sm.id, sm.scenario_name, sm.stage_id, sm.contribution_weight,
			   sm.completion_status, sm.priority, sm.estimated_impact,
			   sm.last_status_check, sm.notes, sm.created_at, sm.updated_at,
			   ps.name as stage_name, s.name as sector_name
		FROM scenario_mappings sm
		JOIN progression_stages ps ON sm.stage_id = ps.id
		JOIN sectors s ON ps.sector_id = s.id
		ORDER BY sm.priority, sm.estimated_impact DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch scenario mappings"})
		return
	}
	defer rows.Close()

	var mappings []gin.H
	for rows.Next() {
		var mapping ScenarioMapping
		var stageName, sectorName string
		err := rows.Scan(&mapping.ID, &mapping.ScenarioName, &mapping.StageID,
			&mapping.ContributionWeight, &mapping.CompletionStatus, &mapping.Priority,
			&mapping.EstimatedImpact, &mapping.LastStatusCheck, &mapping.Notes,
			&mapping.CreatedAt, &mapping.UpdatedAt, &stageName, &sectorName)
		if err != nil {
			continue
		}
		
		mappings = append(mappings, gin.H{
			"mapping":     mapping,
			"stage_name":  stageName,
			"sector_name": sectorName,
		})
	}

	c.JSON(http.StatusOK, gin.H{"scenario_mappings": mappings})
}

// Update scenario status
func updateScenarioStatus(c *gin.Context) {
	scenarioName := c.Param("scenario")
	
	var request struct {
		CompletionStatus string  `json:"completion_status"`
		Notes           string  `json:"notes"`
		EstimatedImpact float64 `json:"estimated_impact,omitempty"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Update scenario mapping
	_, err := db.Exec(`
		UPDATE scenario_mappings 
		SET completion_status = $1, notes = $2, last_status_check = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP,
		    estimated_impact = CASE WHEN $3 > 0 THEN $3 ELSE estimated_impact END
		WHERE scenario_name = $1
	`, request.CompletionStatus, request.Notes, request.EstimatedImpact, scenarioName)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update scenario status"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Scenario status updated successfully",
		"scenario": scenarioName,
		"status": request.CompletionStatus,
	})
}

// Strategic analysis endpoint
func analyzeStrategicPath(c *gin.Context) {
	var request AnalysisRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Generate strategic recommendations based on current state
	recommendations := generateStrategicRecommendations(request)
	
	// Calculate projected timeline
	timeline := calculateProjectedTimeline(request)
	
	// Identify bottlenecks
	bottlenecks := identifyBottlenecks()
	
	// Analyze cross-sector impacts
	impacts := analyzeCrossSectorImpacts()
	
	response := AnalysisResponse{
		Recommendations:    recommendations,
		ProjectedTimeline:  timeline,
		BottleneckAnalysis: bottlenecks,
		CrossSectorImpacts: impacts,
	}
	
	c.JSON(http.StatusOK, response)
}

// Generate strategic recommendations
func generateStrategicRecommendations(request AnalysisRequest) []StrategicRecommendation {
	// Simple heuristic-based recommendations (in production, this would use AI)
	recommendations := []StrategicRecommendation{
		{
			Scenario:         "graph-studio",
			PriorityScore:    9.5,
			ImpactMultiplier: 3.2,
			Reasoning:        "Graph visualization capabilities are fundamental for all tech tree analysis and strategic planning interfaces",
		},
		{
			Scenario:         "research-assistant",
			PriorityScore:    9.0,
			ImpactMultiplier: 2.8,
			Reasoning:        "Research capabilities accelerate discovery of new technology pathways and sector connections",
		},
		{
			Scenario:         "data-structurer",
			PriorityScore:    8.5,
			ImpactMultiplier: 2.5,
			Reasoning:        "Data organization capabilities are required for all sector foundation stages",
		},
	}
	
	return recommendations
}

// Calculate projected timeline for milestones
func calculateProjectedTimeline(request AnalysisRequest) ProjectedTimeline {
	milestones := []MilestoneProjection{
		{
			Name:                "Personal Productivity Mastery",
			EstimatedCompletion: time.Now().AddDate(0, 6, 0),
			Confidence:          0.8,
		},
		{
			Name:                "Core Sector Digital Twins",
			EstimatedCompletion: time.Now().AddDate(2, 0, 0),
			Confidence:          0.6,
		},
		{
			Name:                "Civilization Digital Twin",
			EstimatedCompletion: time.Now().AddDate(5, 0, 0),
			Confidence:          0.4,
		},
	}
	
	return ProjectedTimeline{Milestones: milestones}
}

// Identify bottlenecks in the tech tree
func identifyBottlenecks() []string {
	return []string{
		"Software engineering foundation stages need more scenario coverage",
		"Cross-sector integration patterns are underdeveloped",
		"AI analysis capabilities require more sophisticated models",
		"Real-time progress tracking needs automated scenario status detection",
	}
}

// Analyze cross-sector impacts
func analyzeCrossSectorImpacts() []CrossSectorImpact {
	return []CrossSectorImpact{
		{
			SourceSector: "Software Engineering",
			TargetSector: "Manufacturing",
			ImpactScore:  0.9,
			Description:  "Software development capabilities directly enable all manufacturing system development",
		},
		{
			SourceSector: "Healthcare",
			TargetSector: "Manufacturing",
			ImpactScore:  0.6,
			Description:  "Healthcare insights drive medical device and biotech manufacturing requirements",
		},
	}
}

// Get strategic milestones
func getStrategicMilestones(c *gin.Context) {
	rows, err := db.Query(`
		SELECT id, tree_id, name, description, milestone_type, required_sectors,
			   required_stages, completion_percentage, estimated_completion_date,
			   confidence_level, business_value_estimate, created_at, updated_at
		FROM strategic_milestones
		ORDER BY business_value_estimate DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch milestones"})
		return
	}
	defer rows.Close()

	var milestones []StrategicMilestone
	for rows.Next() {
		var milestone StrategicMilestone
		err := rows.Scan(&milestone.ID, &milestone.TreeID, &milestone.Name,
			&milestone.Description, &milestone.MilestoneType, &milestone.RequiredSectors,
			&milestone.RequiredStages, &milestone.CompletionPercentage,
			&milestone.EstimatedCompletionDate, &milestone.ConfidenceLevel,
			&milestone.BusinessValueEstimate, &milestone.CreatedAt, &milestone.UpdatedAt)
		if err != nil {
			continue
		}
		milestones = append(milestones, milestone)
	}

	c.JSON(http.StatusOK, gin.H{"milestones": milestones})
}

// Get recommendations (simplified version)
func getRecommendations(c *gin.Context) {
	resources := 5 // Default resource level
	if r := c.Query("resources"); r != "" {
		if parsed, err := strconv.Atoi(r); err == nil {
			resources = parsed
		}
	}
	
	request := AnalysisRequest{
		CurrentResources: resources,
		TimeHorizon:      12, // 12 months
		PrioritySectors:  []string{"software", "manufacturing", "healthcare"},
	}
	
	recommendations := generateStrategicRecommendations(request)
	c.JSON(http.StatusOK, gin.H{"recommendations": recommendations})
}

// Get dependencies
func getDependencies(c *gin.Context) {
	rows, err := db.Query(`
		SELECT sd.id, sd.dependent_stage_id, sd.prerequisite_stage_id,
			   sd.dependency_type, sd.dependency_strength, sd.description,
			   ps1.name as dependent_stage_name, ps2.name as prerequisite_stage_name
		FROM stage_dependencies sd
		JOIN progression_stages ps1 ON sd.dependent_stage_id = ps1.id
		JOIN progression_stages ps2 ON sd.prerequisite_stage_id = ps2.id
		ORDER BY sd.dependency_strength DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch dependencies"})
		return
	}
	defer rows.Close()

	var dependencies []gin.H
	for rows.Next() {
		var dep StageDependency
		var dependentName, prerequisiteName string
		err := rows.Scan(&dep.ID, &dep.DependentStageID, &dep.PrerequisiteStageID,
			&dep.DependencyType, &dep.DependencyStrength, &dep.Description,
			&dependentName, &prerequisiteName)
		if err != nil {
			continue
		}
		
		dependencies = append(dependencies, gin.H{
			"dependency":         dep,
			"dependent_name":     dependentName,
			"prerequisite_name":  prerequisiteName,
		})
	}

	c.JSON(http.StatusOK, gin.H{"dependencies": dependencies})
}

// Get cross-sector connections
func getCrossSectorConnections(c *gin.Context) {
	rows, err := db.Query(`
		SELECT sc.id, sc.source_sector_id, sc.target_sector_id,
			   sc.connection_type, sc.strength, sc.description, sc.examples,
			   s1.name as source_name, s2.name as target_name
		FROM sector_connections sc
		JOIN sectors s1 ON sc.source_sector_id = s1.id
		JOIN sectors s2 ON sc.target_sector_id = s2.id
		ORDER BY sc.strength DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch connections"})
		return
	}
	defer rows.Close()

	var connections []gin.H
	for rows.Next() {
		var conn struct {
			ID               string          `json:"id"`
			SourceSectorID   string          `json:"source_sector_id"`
			TargetSectorID   string          `json:"target_sector_id"`
			ConnectionType   string          `json:"connection_type"`
			Strength         float64         `json:"strength"`
			Description      string          `json:"description"`
			Examples         json.RawMessage `json:"examples"`
		}
		var sourceName, targetName string
		
		err := rows.Scan(&conn.ID, &conn.SourceSectorID, &conn.TargetSectorID,
			&conn.ConnectionType, &conn.Strength, &conn.Description, &conn.Examples,
			&sourceName, &targetName)
		if err != nil {
			continue
		}
		
		connections = append(connections, gin.H{
			"connection":   conn,
			"source_name":  sourceName,
			"target_name":  targetName,
		})
	}

	c.JSON(http.StatusOK, gin.H{"connections": connections})
}

// Add new scenario mapping
func updateScenarioMapping(c *gin.Context) {
	var mapping ScenarioMapping
	if err := c.ShouldBindJSON(&mapping); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Set ID and timestamps
	mapping.ID = uuid.New().String()
	mapping.CreatedAt = time.Now()
	mapping.UpdatedAt = time.Now()
	mapping.LastStatusCheck = time.Now()
	
	_, err := db.Exec(`
		INSERT INTO scenario_mappings (id, scenario_name, stage_id, contribution_weight,
			completion_status, priority, estimated_impact, last_status_check, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (scenario_name, stage_id) DO UPDATE SET
			contribution_weight = $4, completion_status = $5, priority = $6,
			estimated_impact = $7, last_status_check = $8, notes = $9, updated_at = $11
	`, mapping.ID, mapping.ScenarioName, mapping.StageID, mapping.ContributionWeight,
		mapping.CompletionStatus, mapping.Priority, mapping.EstimatedImpact,
		mapping.LastStatusCheck, mapping.Notes, mapping.CreatedAt, mapping.UpdatedAt)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update scenario mapping"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Scenario mapping updated successfully",
		"mapping": mapping,
	})
}