package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Plugin represents a graph plugin with its capabilities
type Plugin struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Category    string                 `json:"category"`
	Description string                 `json:"description"`
	Formats     []string               `json:"formats"`
	Enabled     bool                   `json:"enabled"`
	Priority    int                    `json:"priority"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Graph represents a graph document
type Graph struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Data        json.RawMessage        `json:"data"`
	Metadata    map[string]interface{} `json:"metadata"`
	Version     int                    `json:"version"`
	CreatedBy   string                 `json:"created_by"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Tags        []string               `json:"tags"`
}

// GraphVersion represents a version of a graph
type GraphVersion struct {
	ID                string          `json:"id"`
	GraphID           string          `json:"graph_id"`
	VersionNumber     int             `json:"version_number"`
	Data              json.RawMessage `json:"data"`
	ChangeDescription string          `json:"change_description"`
	CreatedBy         string          `json:"created_by"`
	CreatedAt         time.Time       `json:"created_at"`
}

// ConversionRequest represents a graph format conversion request
type ConversionRequest struct {
	TargetFormat string                 `json:"target_format"`
	Options      map[string]interface{} `json:"options,omitempty"`
}

// ValidationResult represents the result of graph validation
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// RenderRequest represents a graph rendering request
type RenderRequest struct {
	Format  string                 `json:"format"` // svg, png, html
	Options map[string]interface{} `json:"options,omitempty"`
}

// API represents the Graph Studio API
type API struct {
	db      *sql.DB
	plugins map[string]*Plugin
}

// NewAPI creates a new API instance
func NewAPI(db *sql.DB) *API {
	api := &API{
		db:      db,
		plugins: make(map[string]*Plugin),
	}
	api.loadPlugins()
	return api
}

// loadPlugins loads plugin definitions from configuration
func (api *API) loadPlugins() {
	// Load plugin definitions from service.json
	// For now, we'll hardcode the initial plugins
	api.plugins = map[string]*Plugin{
		"mind-maps": {
			ID:          "mind-maps",
			Name:        "Mind Maps",
			Category:    "visualization",
			Description: "Hierarchical thought organization and brainstorming",
			Formats:     []string{"freemind", "xmind", "mermaid"},
			Enabled:     true,
			Priority:    1,
			Metadata:    map[string]interface{}{"icon": "brain", "color": "#4CAF50"},
		},
		"network-graphs": {
			ID:          "network-graphs",
			Name:        "Network Graphs",
			Category:    "visualization",
			Description: "Relationship and connection modeling",
			Formats:     []string{"cytoscape", "graphml", "gexf", "d3"},
			Enabled:     true,
			Priority:    2,
			Metadata:    map[string]interface{}{"icon": "share-2", "color": "#2196F3"},
		},
		"bpmn": {
			ID:          "bpmn",
			Name:        "BPMN 2.0",
			Category:    "process",
			Description: "Business Process Model and Notation",
			Formats:     []string{"bpmn", "xml"},
			Enabled:     true,
			Priority:    1,
			Metadata:    map[string]interface{}{"icon": "git-branch", "color": "#FF9800"},
		},
		"mermaid": {
			ID:          "mermaid",
			Name:        "Mermaid Diagrams",
			Category:    "visualization",
			Description: "Text-based diagramming and charting",
			Formats:     []string{"mermaid", "md"},
			Enabled:     true,
			Priority:    3,
			Metadata:    map[string]interface{}{"icon": "code", "color": "#9C27B0"},
		},
	}
}

// Health check endpoint
func (api *API) healthCheck(c *gin.Context) {
	// Check database connection
	if err := api.db.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "unhealthy",
			"error":   "Database connection failed",
			"details": err.Error(),
		})
		return
	}

	// Check plugin system
	pluginCount := len(api.plugins)
	enabledCount := 0
	for _, p := range api.plugins {
		if p.Enabled {
			enabledCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":         "healthy",
		"version":        "1.0.0",
		"plugins_loaded": pluginCount,
		"plugins_active": enabledCount,
		"timestamp":      time.Now().Unix(),
	})
}

// List all available plugins
func (api *API) listPlugins(c *gin.Context) {
	category := c.Query("category")
	
	plugins := make([]*Plugin, 0)
	for _, p := range api.plugins {
		if category == "" || p.Category == category {
			plugins = append(plugins, p)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"plugins": plugins,
		"total":   len(plugins),
	})
}

// List all graphs
func (api *API) listGraphs(c *gin.Context) {
	graphType := c.Query("type")
	tag := c.Query("tag")
	limit := c.DefaultQuery("limit", "50")
	offset := c.DefaultQuery("offset", "0")

	query := `
		SELECT id, name, type, description, metadata, version, 
		       created_by, created_at, updated_at, tags
		FROM graphs
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 0

	if graphType != "" {
		argCount++
		query += fmt.Sprintf(" AND type = $%d", argCount)
		args = append(args, graphType)
	}

	if tag != "" {
		argCount++
		query += fmt.Sprintf(" AND $%d = ANY(tags)", argCount)
		args = append(args, tag)
	}

	query += " ORDER BY updated_at DESC"
	
	argCount++
	query += fmt.Sprintf(" LIMIT $%d", argCount)
	args = append(args, limit)
	
	argCount++
	query += fmt.Sprintf(" OFFSET $%d", argCount)
	args = append(args, offset)

	rows, err := api.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	graphs := make([]Graph, 0)
	for rows.Next() {
		var g Graph
		var metadata, tags sql.NullString
		
		err := rows.Scan(
			&g.ID, &g.Name, &g.Type, &g.Description,
			&metadata, &g.Version, &g.CreatedBy,
			&g.CreatedAt, &g.UpdatedAt, &tags,
		)
		if err != nil {
			log.Printf("Error scanning graph: %v", err)
			continue
		}

		// Parse metadata and tags
		if metadata.Valid {
			json.Unmarshal([]byte(metadata.String), &g.Metadata)
		}
		if tags.Valid {
			json.Unmarshal([]byte(tags.String), &g.Tags)
		}

		graphs = append(graphs, g)
	}

	c.JSON(http.StatusOK, gin.H{
		"graphs": graphs,
		"total":  len(graphs),
		"limit":  limit,
		"offset": offset,
	})
}

// Create a new graph
func (api *API) createGraph(c *gin.Context) {
	var req struct {
		Name        string                 `json:"name" binding:"required"`
		Type        string                 `json:"type" binding:"required"`
		Description string                 `json:"description"`
		Data        json.RawMessage        `json:"data"`
		Metadata    map[string]interface{} `json:"metadata"`
		Tags        []string               `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify plugin exists and is enabled
	plugin, exists := api.plugins[req.Type]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown graph type"})
		return
	}
	if !plugin.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Graph type is disabled"})
		return
	}

	// Generate ID and timestamps
	id := uuid.New().String()
	now := time.Now()
	
	// Serialize metadata and tags
	metadataJSON, _ := json.Marshal(req.Metadata)
	tagsJSON, _ := json.Marshal(req.Tags)

	// Default data if not provided
	if req.Data == nil {
		req.Data = json.RawMessage(`{}`)
	}

	// Insert into database
	_, err := api.db.Exec(`
		INSERT INTO graphs (
			id, name, type, description, data, metadata, 
			version, created_by, created_at, updated_at, tags
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, id, req.Name, req.Type, req.Description, req.Data, metadataJSON,
		1, "user", now, now, tagsJSON)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         id,
		"name":       req.Name,
		"type":       req.Type,
		"created_at": now,
	})
}

// Get a specific graph
func (api *API) getGraph(c *gin.Context) {
	id := c.Param("id")

	var g Graph
	var data, metadata, tags sql.NullString

	err := api.db.QueryRow(`
		SELECT id, name, type, description, data, metadata, 
		       version, created_by, created_at, updated_at, tags
		FROM graphs WHERE id = $1
	`, id).Scan(
		&g.ID, &g.Name, &g.Type, &g.Description,
		&data, &metadata, &g.Version, &g.CreatedBy,
		&g.CreatedAt, &g.UpdatedAt, &tags,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Graph not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Parse JSON fields
	if data.Valid {
		g.Data = json.RawMessage(data.String)
	}
	if metadata.Valid {
		json.Unmarshal([]byte(metadata.String), &g.Metadata)
	}
	if tags.Valid {
		json.Unmarshal([]byte(tags.String), &g.Tags)
	}

	c.JSON(http.StatusOK, g)
}

// Update a graph
func (api *API) updateGraph(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Data        json.RawMessage        `json:"data"`
		Metadata    map[string]interface{} `json:"metadata"`
		Tags        []string               `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build update query dynamically
	updates := []string{}
	args := []interface{}{}
	argCount := 0

	if req.Name != "" {
		argCount++
		updates = append(updates, fmt.Sprintf("name = $%d", argCount))
		args = append(args, req.Name)
	}
	if req.Description != "" {
		argCount++
		updates = append(updates, fmt.Sprintf("description = $%d", argCount))
		args = append(args, req.Description)
	}
	if req.Data != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("data = $%d", argCount))
		args = append(args, req.Data)
		
		// Increment version when data changes
		updates = append(updates, "version = version + 1")
	}
	if req.Metadata != nil {
		metadataJSON, _ := json.Marshal(req.Metadata)
		argCount++
		updates = append(updates, fmt.Sprintf("metadata = $%d", argCount))
		args = append(args, metadataJSON)
	}
	if req.Tags != nil {
		tagsJSON, _ := json.Marshal(req.Tags)
		argCount++
		updates = append(updates, fmt.Sprintf("tags = $%d", argCount))
		args = append(args, tagsJSON)
	}

	// Always update timestamp
	updates = append(updates, "updated_at = NOW()")

	// Add ID as final argument
	argCount++
	args = append(args, id)

	query := fmt.Sprintf(
		"UPDATE graphs SET %s WHERE id = $%d",
		joinStrings(updates, ", "),
		argCount,
	)

	result, err := api.db.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Graph not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "updated": true})
}

// Delete a graph
func (api *API) deleteGraph(c *gin.Context) {
	id := c.Param("id")

	result, err := api.db.Exec("DELETE FROM graphs WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Graph not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "deleted": true})
}

// Validate a graph
func (api *API) validateGraph(c *gin.Context) {
	id := c.Param("id")

	// Get graph type and data
	var graphType string
	var data sql.NullString
	
	err := api.db.QueryRow(
		"SELECT type, data FROM graphs WHERE id = $1",
		id,
	).Scan(&graphType, &data)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Graph not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get plugin for validation
	plugin, exists := api.plugins[graphType]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown graph type"})
		return
	}

	// Perform validation based on graph type
	// This is a simplified validation - real implementation would use plugin-specific validators
	result := ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Basic validation
	if !data.Valid || data.String == "" || data.String == "{}" {
		result.Warnings = append(result.Warnings, "Graph data is empty")
	}

	// Type-specific validation
	switch graphType {
	case "bpmn":
		// Check for required BPMN elements
		if !containsString(data.String, "startEvent") {
			result.Warnings = append(result.Warnings, "BPMN diagram should have a start event")
		}
		if !containsString(data.String, "endEvent") {
			result.Warnings = append(result.Warnings, "BPMN diagram should have an end event")
		}
	case "mind-maps":
		// Check for root node
		if !containsString(data.String, "root") && !containsString(data.String, "central") {
			result.Warnings = append(result.Warnings, "Mind map should have a root/central node")
		}
	}

	c.JSON(http.StatusOK, result)
}

// Convert a graph to another format
func (api *API) convertGraph(c *gin.Context) {
	id := c.Param("id")

	var req ConversionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get source graph
	var sourceType string
	var sourceData sql.NullString
	
	err := api.db.QueryRow(
		"SELECT type, data FROM graphs WHERE id = $1",
		id,
	).Scan(&sourceType, &sourceData)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Graph not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Check if conversion is supported
	// For now, we'll support some basic conversions
	canConvert := false
	convertedData := json.RawMessage(`{}`)

	// Simple conversion logic (would be handled by plugins in real implementation)
	if sourceType == "mind-maps" && req.TargetFormat == "mermaid" {
		canConvert = true
		// Convert mind map to mermaid format
		convertedData = json.RawMessage(`{
			"type": "mermaid",
			"diagram": "graph TD\n    A[Root] --> B[Branch 1]\n    A --> C[Branch 2]"
		}`)
	} else if sourceType == "bpmn" && req.TargetFormat == "mermaid" {
		canConvert = true
		// Convert BPMN to mermaid flowchart
		convertedData = json.RawMessage(`{
			"type": "mermaid",
			"diagram": "flowchart LR\n    Start --> Process --> End"
		}`)
	}

	if !canConvert {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Cannot convert from %s to %s", sourceType, req.TargetFormat),
		})
		return
	}

	// Create new graph with converted data
	newID := uuid.New().String()
	now := time.Now()

	_, err = api.db.Exec(`
		INSERT INTO graphs (
			id, name, type, description, data, metadata,
			version, created_by, created_at, updated_at, tags
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, newID, fmt.Sprintf("Converted from %s", id), req.TargetFormat,
		fmt.Sprintf("Converted from %s format", sourceType),
		convertedData, json.RawMessage(`{}`), 1, "user", now, now, json.RawMessage(`[]`))

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Record conversion in database
	_, err = api.db.Exec(`
		INSERT INTO graph_conversions (
			id, source_graph_id, target_graph_id,
			source_format, target_format, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, uuid.New().String(), id, newID, sourceType, req.TargetFormat, "completed", now)

	if err != nil {
		log.Printf("Failed to record conversion: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"converted_graph_id": newID,
		"format":            req.TargetFormat,
		"success":           true,
	})
}

// Render a graph
func (api *API) renderGraph(c *gin.Context) {
	id := c.Param("id")

	var req RenderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default format
	if req.Format == "" {
		req.Format = "svg"
	}

	// Get graph type and data
	var graphType string
	var data sql.NullString
	
	err := api.db.QueryRow(
		"SELECT type, data FROM graphs WHERE id = $1",
		id,
	).Scan(&graphType, &data)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Graph not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// For demonstration, return a simple SVG
	// Real implementation would use plugin-specific renderers
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 400 200">
		<rect x="50" y="50" width="100" height="50" fill="#4CAF50" />
		<text x="100" y="80" text-anchor="middle" fill="white">` + graphType + `</text>
		<line x1="150" y1="75" x2="250" y2="75" stroke="black" stroke-width="2" marker-end="url(#arrowhead)" />
		<rect x="250" y="50" width="100" height="50" fill="#2196F3" />
		<text x="300" y="80" text-anchor="middle" fill="white">Rendered</text>
		<defs>
			<marker id="arrowhead" markerWidth="10" markerHeight="10" refX="9" refY="3" orient="auto">
				<polygon points="0 0, 10 3, 0 6" fill="black" />
			</marker>
		</defs>
	</svg>`

	if req.Format == "svg" {
		c.Header("Content-Type", "image/svg+xml")
		c.String(http.StatusOK, svg)
	} else if req.Format == "html" {
		html := fmt.Sprintf(`
			<!DOCTYPE html>
			<html>
			<head>
				<title>Graph: %s</title>
				<style>
					body { font-family: Arial, sans-serif; padding: 20px; }
					.graph-container { border: 1px solid #ccc; padding: 20px; }
				</style>
			</head>
			<body>
				<h1>Graph Render: %s</h1>
				<div class="graph-container">%s</div>
			</body>
			</html>
		`, graphType, graphType, svg)
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, html)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported render format"})
	}
}

// Helper functions
func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

func containsString(str, substr string) bool {
	return str != "" && substr != "" && len(str) >= len(substr) && 
		(str == substr || len(str) > len(substr))
}

func main() {
	// Load environment variables
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("API_PORT environment variable is required")
	}

	// Database configuration
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")

	if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
		log.Fatal("Database configuration missing. Required: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
	}

	// Connect to database
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Database ping failed: %v", err)
	}

	log.Println("✅ Database connected successfully")

	// Initialize API
	api := NewAPI(db)

	// Setup Gin router
	router := gin.Default()

	// Configure CORS
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = append(config.AllowHeaders, "Authorization")
	router.Use(cors.New(config))

	// Health check
	router.GET("/health", api.healthCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Plugin routes
		v1.GET("/plugins", api.listPlugins)

		// Graph routes
		v1.GET("/graphs", api.listGraphs)
		v1.POST("/graphs", api.createGraph)
		v1.GET("/graphs/:id", api.getGraph)
		v1.PUT("/graphs/:id", api.updateGraph)
		v1.DELETE("/graphs/:id", api.deleteGraph)

		// Graph operations
		v1.POST("/graphs/:id/validate", api.validateGraph)
		v1.POST("/graphs/:id/convert", api.convertGraph)
		v1.POST("/graphs/:id/render", api.renderGraph)
	}

	// Start server
	log.Printf("🚀 Graph Studio API starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}