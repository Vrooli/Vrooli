package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// API handles all graph studio operations
type API struct {
	plugins         map[string]*Plugin
	conversionEngine *ConversionEngine
	validator       *GraphValidator
	metrics         *MetricsCollector
}

// NewAPI creates a new API instance
func NewAPI() *API {
	api := &API{
		plugins:         make(map[string]*Plugin),
		conversionEngine: NewConversionEngine(),
		metrics:         NewMetricsCollector(),
	}
	api.loadPlugins()
	api.validator = NewGraphValidator(api.plugins)
	return api
}

// loadPlugins loads plugin definitions from database
func (api *API) loadPlugins() {
	// Initialize plugins map
	api.plugins = make(map[string]*Plugin)
	
	// Note: Database connection is not available during API initialization
	// Plugins will be loaded on first request in getPluginsFromDB()
}

// getPluginsFromDB loads plugins from database with caching
func (api *API) getPluginsFromDB(db *sql.DB) error {
	// If plugins already loaded, return immediately
	if len(api.plugins) > 0 {
		return nil
	}
	
	// Query plugins from database
	rows, err := db.Query(`
		SELECT id, name, category, description, formats, enabled, priority, metadata
		FROM plugins 
		ORDER BY priority ASC, name ASC
	`)
	if err != nil {
		return fmt.Errorf("failed to query plugins: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var plugin Plugin
		var formatsJSON, metadataJSON []byte
		
		err := rows.Scan(
			&plugin.ID, &plugin.Name, &plugin.Category, &plugin.Description,
			&formatsJSON, &plugin.Enabled, &plugin.Priority, &metadataJSON,
		)
		if err != nil {
			log.Printf("Error scanning plugin: %v", err)
			continue
		}
		
		// Parse JSON fields
		if err := json.Unmarshal(formatsJSON, &plugin.Formats); err != nil {
			log.Printf("Error parsing formats for plugin %s: %v", plugin.ID, err)
			plugin.Formats = []string{}
		}
		
		if err := json.Unmarshal(metadataJSON, &plugin.Metadata); err != nil {
			log.Printf("Error parsing metadata for plugin %s: %v", plugin.ID, err)
			plugin.Metadata = make(map[string]interface{})
		}
		
		api.plugins[plugin.ID] = &plugin
	}
	
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating plugins: %w", err)
	}
	
	log.Printf("Loaded %d plugins from database", len(api.plugins))
	return nil
}

// HealthCheck handles health check endpoint
func (api *API) HealthCheck(c *gin.Context) {
	db := getDB(c)
	
	// Check database connection
	if err := db.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, HealthResponse{
			Status:  "unhealthy",
			Error:   "Database connection failed",
			Details: err.Error(),
		})
		return
	}
	
	// Count plugins
	pluginCount := len(api.plugins)
	enabledCount := 0
	for _, p := range api.plugins {
		if p.Enabled {
			enabledCount++
		}
	}
	
	c.JSON(http.StatusOK, HealthResponse{
		Status:        "healthy",
		Version:       "1.0.0",
		PluginsLoaded: pluginCount,
		PluginsActive: enabledCount,
		Timestamp:     time.Now().Unix(),
	})
}

// ListPlugins lists all available plugins
func (api *API) ListPlugins(c *gin.Context) {
	db := getDB(c)
	
	// Load plugins from database if not already loaded
	if err := api.getPluginsFromDB(db); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to load plugins",
			Details: err.Error(),
		})
		return
	}
	
	category := c.Query("category")
	
	plugins := make([]*Plugin, 0)
	for _, p := range api.plugins {
		if category == "" || p.Category == category {
			plugins = append(plugins, p)
		}
	}
	
	c.JSON(http.StatusOK, ListResponse{
		Data:  plugins,
		Total: len(plugins),
	})
}

// ListGraphs lists all graphs with pagination
func (api *API) ListGraphs(c *gin.Context) {
	db := getDB(c)
	
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
	
	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
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
	
	c.JSON(http.StatusOK, ListResponse{
		Data:   graphs,
		Total:  len(graphs),
		Limit:  limit,
		Offset: offset,
	})
}

// CreateGraph creates a new graph
func (api *API) CreateGraph(c *gin.Context) {
	db := getDB(c)
	userID := getUserID(c)
	
	var req CreateGraphRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	
	// Sanitize input
	req.Name = SanitizeGraphName(req.Name)
	req.Description = SanitizeDescription(req.Description)
	req.Tags = SanitizeTags(req.Tags)
	
	// Validate request
	if validationErrors := api.validator.ValidateCreateGraphRequest(&req); len(validationErrors) > 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Validation failed",
			Details: validationErrors.Error(),
		})
		return
	}
	
	// Load plugins from database if not already loaded
	if err := api.getPluginsFromDB(db); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to load plugins",
			Details: err.Error(),
		})
		return
	}
	
	// Verify plugin exists and is enabled
	plugin, exists := api.plugins[req.Type]
	if !exists {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Unknown graph type"})
		return
	}
	if !plugin.Enabled {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Graph type is disabled"})
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
	_, err := db.Exec(`
		INSERT INTO graphs (
			id, name, type, description, data, metadata, 
			version, created_by, created_at, updated_at, tags
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, id, req.Name, req.Type, req.Description, req.Data, metadataJSON,
		1, userID, now, now, tagsJSON)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	
	// Log the operation
	api.metrics.LogGraphOperation(db, "created", userID, id, req.Type, time.Since(now), true, map[string]interface{}{
		"name": req.Name,
		"tags": req.Tags,
	})
	
	c.JSON(http.StatusCreated, map[string]interface{}{
		"id":         id,
		"name":       req.Name,
		"type":       req.Type,
		"created_at": now,
	})
}

// GetGraph retrieves a specific graph
func (api *API) GetGraph(c *gin.Context) {
	db := getDB(c)
	id := c.Param("id")
	
	// Validate graph ID format
	if !IsValidGraphID(id) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid graph ID format"})
		return
	}
	
	var g Graph
	var data, metadata, tags sql.NullString
	
	err := db.QueryRow(`
		SELECT id, name, type, description, data, metadata, 
		       version, created_by, created_at, updated_at, tags
		FROM graphs WHERE id = $1
	`, id).Scan(
		&g.ID, &g.Name, &g.Type, &g.Description,
		&data, &metadata, &g.Version, &g.CreatedBy,
		&g.CreatedAt, &g.UpdatedAt, &tags,
	)
	
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Graph not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
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

// UpdateGraph updates an existing graph
func (api *API) UpdateGraph(c *gin.Context) {
	db := getDB(c)
	id := c.Param("id")
	
	// Validate graph ID format
	if !IsValidGraphID(id) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid graph ID format"})
		return
	}
	
	var req UpdateGraphRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	
	// Sanitize input
	if req.Name != "" {
		req.Name = SanitizeGraphName(req.Name)
	}
	if req.Description != "" {
		req.Description = SanitizeDescription(req.Description)
	}
	if req.Tags != nil {
		req.Tags = SanitizeTags(req.Tags)
	}
	
	// Validate request
	if validationErrors := api.validator.ValidateUpdateGraphRequest(&req); len(validationErrors) > 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Validation failed",
			Details: validationErrors.Error(),
		})
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
	
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "No fields to update"})
		return
	}
	
	// Always update timestamp
	updates = append(updates, "updated_at = NOW()")
	
	// Add ID as final argument
	argCount++
	args = append(args, id)
	
	query := fmt.Sprintf(
		"UPDATE graphs SET %s WHERE id = $%d",
		strings.Join(updates, ", "),
		argCount,
	)
	
	result, err := db.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Graph not found"})
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: "Graph updated"})
}

// DeleteGraph deletes a graph
func (api *API) DeleteGraph(c *gin.Context) {
	db := getDB(c)
	id := c.Param("id")
	
	// Validate graph ID format
	if !IsValidGraphID(id) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid graph ID format"})
		return
	}
	
	result, err := db.Exec("DELETE FROM graphs WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Graph not found"})
		return
	}
	
	c.JSON(http.StatusOK, SuccessResponse{Success: true, Message: "Graph deleted"})
}

// ValidateGraph validates a graph against its schema
func (api *API) ValidateGraph(c *gin.Context) {
	db := getDB(c)
	id := c.Param("id")
	
	// Validate graph ID format
	if !IsValidGraphID(id) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid graph ID format"})
		return
	}
	
	// Get graph type and data
	var graphType string
	var data sql.NullString
	
	err := db.QueryRow(
		"SELECT type, data FROM graphs WHERE id = $1",
		id,
	).Scan(&graphType, &data)
	
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Graph not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	
	// Load plugins from database if not already loaded
	if err := api.getPluginsFromDB(db); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to load plugins",
			Details: err.Error(),
		})
		return
	}
	
	// Get plugin for validation
	_, exists := api.plugins[graphType]
	if !exists {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Unknown graph type"})
		return
	}
	
	// Perform validation based on graph type
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
		if !strings.Contains(data.String, "startEvent") {
			result.Warnings = append(result.Warnings, "BPMN diagram should have a start event")
		}
		if !strings.Contains(data.String, "endEvent") {
			result.Warnings = append(result.Warnings, "BPMN diagram should have an end event")
		}
	case "mind-maps":
		if !strings.Contains(data.String, "root") && !strings.Contains(data.String, "central") {
			result.Warnings = append(result.Warnings, "Mind map should have a root/central node")
		}
	}
	
	c.JSON(http.StatusOK, result)
}

// ConvertGraph converts a graph to another format
func (api *API) ConvertGraph(c *gin.Context) {
	db := getDB(c)
	id := c.Param("id")
	userID := getUserID(c)
	
	// Validate graph ID format
	if !IsValidGraphID(id) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid graph ID format"})
		return
	}
	
	var req ConversionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	
	// Validate request
	if validationErrors := api.validator.ValidateConversionRequest(&req); len(validationErrors) > 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Validation failed",
			Details: validationErrors.Error(),
		})
		return
	}
	
	// Get source graph
	var sourceType string
	var sourceData sql.NullString
	
	err := db.QueryRow(
		"SELECT type, data FROM graphs WHERE id = $1",
		id,
	).Scan(&sourceType, &sourceData)
	
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Graph not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	
	// Check if conversion is supported using the conversion engine
	if !api.conversionEngine.CanConvert(sourceType, req.TargetFormat) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: fmt.Sprintf("Cannot convert from %s to %s", sourceType, req.TargetFormat),
		})
		return
	}
	
	// Convert source data to json.RawMessage
	var sourceJSON json.RawMessage
	if sourceData.Valid && sourceData.String != "" {
		sourceJSON = json.RawMessage(sourceData.String)
	} else {
		sourceJSON = json.RawMessage(`{}`)
	}
	
	// Perform the conversion
	convertedData, err := api.conversionEngine.Convert(sourceType, req.TargetFormat, sourceJSON, req.Options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Conversion failed",
			Details: err.Error(),
		})
		return
	}
	
	// Create new graph with converted data
	newID := uuid.New().String()
	now := time.Now()
	
	_, err = db.Exec(`
		INSERT INTO graphs (
			id, name, type, description, data, metadata,
			version, created_by, created_at, updated_at, tags
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, newID, fmt.Sprintf("Converted from %s", id), req.TargetFormat,
		fmt.Sprintf("Converted from %s format", sourceType),
		convertedData, json.RawMessage(`{}`), 1, userID, now, now, json.RawMessage(`[]`))
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	
	// Record conversion in database
	_, _ = db.Exec(`
		INSERT INTO graph_conversions (
			id, source_graph_id, target_graph_id,
			source_format, target_format, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, uuid.New().String(), id, newID, sourceType, req.TargetFormat, "completed", now)
	
	// Log the conversion
	api.metrics.LogConversion(db, userID, id, newID, sourceType, req.TargetFormat, time.Since(now), true, "")
	
	c.JSON(http.StatusOK, map[string]interface{}{
		"converted_graph_id": newID,
		"format":            req.TargetFormat,
		"success":           true,
	})
}

// RenderGraph renders a graph in various formats
func (api *API) RenderGraph(c *gin.Context) {
	db := getDB(c)
	id := c.Param("id")
	
	// Validate graph ID format
	if !IsValidGraphID(id) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid graph ID format"})
		return
	}
	
	var req RenderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	
	// Default format
	if req.Format == "" {
		req.Format = "svg"
	}
	
	// Get graph type and data
	var graphType string
	var data sql.NullString
	
	err := db.QueryRow(
		"SELECT type, data FROM graphs WHERE id = $1",
		id,
	).Scan(&graphType, &data)
	
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Graph not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}
	
	// For demonstration, return a simple SVG
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
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Unsupported render format"})
	}
}

// ListConversions lists all supported conversion paths
func (api *API) ListConversions(c *gin.Context) {
	conversions := api.conversionEngine.GetSupportedConversions()
	
	// Transform to a more detailed format
	result := make(map[string][]map[string]interface{})
	for from, targets := range conversions {
		for _, to := range targets {
			if result[from] == nil {
				result[from] = make([]map[string]interface{}, 0)
			}
			
			metadata, err := api.conversionEngine.GetConversionMetadata(from, to)
			if err != nil {
				continue
			}
			
			result[from] = append(result[from], map[string]interface{}{
				"target":      to,
				"name":        metadata.Name,
				"description": metadata.Description,
				"data_loss":   metadata.DataLoss,
				"quality":     metadata.Quality,
				"features":    metadata.Features,
			})
		}
	}
	
	c.JSON(http.StatusOK, map[string]interface{}{
		"conversions": result,
		"total_paths": len(conversions),
	})
}

// GetConversionMetadata returns detailed metadata for a specific conversion
func (api *API) GetConversionMetadata(c *gin.Context) {
	from := c.Param("from")
	to := c.Param("to")
	
	if !api.conversionEngine.CanConvert(from, to) {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: fmt.Sprintf("Conversion from %s to %s is not supported", from, to),
		})
		return
	}
	
	metadata, err := api.conversionEngine.GetConversionMetadata(from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get conversion metadata",
			Details: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, map[string]interface{}{
		"from":        from,
		"to":          to,
		"name":        metadata.Name,
		"description": metadata.Description,
		"data_loss":   metadata.DataLoss,
		"quality":     metadata.Quality,
		"features":    metadata.Features,
		"supported":   true,
	})
}

// GetSystemMetrics returns system metrics and analytics
func (api *API) GetSystemMetrics(c *gin.Context) {
	db := getDB(c)
	
	metrics := api.metrics.GetSystemMetrics(db)
	
	c.JSON(http.StatusOK, map[string]interface{}{
		"metrics":   metrics,
		"timestamp": time.Now().Unix(),
	})
}

// GetDetailedHealth returns detailed health status
func (api *API) GetDetailedHealth(c *gin.Context) {
	db := getDB(c)
	
	health := api.metrics.GetHealthStatus(db)
	
	c.JSON(http.StatusOK, health)
}