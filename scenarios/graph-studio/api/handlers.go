package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// API handles all graph studio operations
type API struct {
	plugins          map[string]*Plugin
	conversionEngine *ConversionEngine
	validator        *GraphValidator
	metrics          *MetricsCollector
}

// NewAPI creates a new API instance
func NewAPI() *API {
	api := &API{
		plugins:          make(map[string]*Plugin),
		conversionEngine: NewConversionEngine(),
		metrics:          NewMetricsCollector(),
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
	// Clear existing plugins to allow fresh load (without breaking validator reference)
	for k := range api.plugins {
		delete(api.plugins, k)
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
		plugin := &Plugin{}
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
		} else if plugin.Metadata == nil {
			plugin.Metadata = make(map[string]interface{})
		}

		log.Printf("Loaded plugin: id=%s, name=%s, enabled=%v", plugin.ID, plugin.Name, plugin.Enabled)
		api.plugins[plugin.ID] = plugin
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

	// Ensure plugin registry is loaded so restart checks reflect real state
	if len(api.plugins) == 0 {
		if err := api.getPluginsFromDB(db); err != nil {
			c.JSON(http.StatusServiceUnavailable, HealthResponse{
				Status:    "degraded",
				Version:   "1.0.0",
				Timestamp: time.Now().Unix(),
				Error:     "Failed to load plugins",
				Details:   err.Error(),
			})
			return
		}
	}

	// Count plugins
	pluginCount := len(api.plugins)
	enabledCount := 0
	for _, p := range api.plugins {
		if p.Enabled {
			enabledCount++
		}
	}

	status := "healthy"
	if pluginCount == 0 || enabledCount == 0 {
		status = "degraded"
	}

	c.JSON(http.StatusOK, HealthResponse{
		Status:        status,
		Version:       "1.0.0",
		PluginsLoaded: pluginCount,
		PluginsActive: enabledCount,
		Timestamp:     time.Now().Unix(),
	})
}

// GetStats returns dashboard statistics
func (api *API) GetStats(c *gin.Context) {
	db := getDB(c)

	// Get total graphs count
	var totalGraphs int
	if err := db.QueryRow("SELECT COUNT(*) FROM graphs").Scan(&totalGraphs); err != nil {
		log.Printf("Error getting total graphs: %v", err)
		totalGraphs = 0
	}

	// Get conversions today count
	var conversionsToday int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM graph_conversions
		WHERE created_at >= CURRENT_DATE
	`).Scan(&conversionsToday); err != nil {
		log.Printf("Error getting conversions today: %v", err)
		conversionsToday = 0
	}

	// Get active users (last 7 days)
	var activeUsers int
	if err := db.QueryRow(`
		SELECT COUNT(DISTINCT created_by) FROM graphs
		WHERE created_at > NOW() - INTERVAL '7 days'
	`).Scan(&activeUsers); err != nil {
		log.Printf("Error getting active users: %v", err)
		activeUsers = 1
	}

	c.JSON(http.StatusOK, DashboardStatsResponse{
		TotalGraphs:      totalGraphs,
		ConversionsToday: conversionsToday,
		ActiveUsers:      activeUsers,
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

// ListGraphs lists all graphs with pagination and search
func (api *API) ListGraphs(c *gin.Context) {
	db := getDB(c)

	graphType := c.Query("type")
	tag := c.Query("tag")
	search := c.Query("search")
	limitParam := c.DefaultQuery("limit", "50")
	offsetParam := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitParam)
	if err != nil || limit <= 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid limit parameter"})
		return
	}
	if limit > 200 {
		limit = 200
	}

	offset, err := strconv.Atoi(offsetParam)
	if err != nil || offset < 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid offset parameter"})
		return
	}

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

	// Full-text search across name, description, and tags
	if search != "" {
		argCount++
		query += fmt.Sprintf(` AND (
			LOWER(name) LIKE LOWER($%d) OR
			LOWER(COALESCE(description, '')) LIKE LOWER($%d) OR
			(jsonb_typeof(tags) = 'array' AND EXISTS (
				SELECT 1 FROM jsonb_array_elements_text(tags) tag
				WHERE LOWER(tag) LIKE LOWER($%d)
			))
		)`, argCount, argCount, argCount)
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern)
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
		Limit:  strconv.Itoa(limit),
		Offset: strconv.Itoa(offset),
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

	// Load plugins from database first (needed for validation)
	if err := api.getPluginsFromDB(db); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to load plugins",
			Details: err.Error(),
		})
		return
	}

	// Sanitize input
	req.Name = SanitizeGraphName(req.Name)
	req.Description = SanitizeDescription(req.Description)
	req.Tags = SanitizeTags(req.Tags)

	// Validate request (after plugins are loaded)
	if validationErrors := api.validator.ValidateCreateGraphRequest(&req); len(validationErrors) > 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Validation failed",
			Details: validationErrors.Error(),
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
	userID := getUserID(c)
	id := c.Param("id")

	// Validate graph ID format
	if !IsValidGraphID(id) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid graph ID format"})
		return
	}

	// Check read permission
	hasPermission, err := CheckGraphPermission(db, id, userID, PermissionRead)
	if err != nil {
		if err.Error() == "graph not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Graph not found"})
		} else {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		}
		return
	}
	if !hasPermission {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Access denied",
			Details: "You do not have permission to view this graph",
		})
		return
	}

	var g Graph
	var data, metadata, tags, permissions sql.NullString

	err = db.QueryRow(`
		SELECT id, name, type, description, data, metadata,
		       version, created_by, created_at, updated_at, tags, permissions
		FROM graphs WHERE id = $1
	`, id).Scan(
		&g.ID, &g.Name, &g.Type, &g.Description,
		&data, &metadata, &g.Version, &g.CreatedBy,
		&g.CreatedAt, &g.UpdatedAt, &tags, &permissions,
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
	if permissions.Valid && permissions.String != "" {
		var perms GraphPermissions
		if err := json.Unmarshal([]byte(permissions.String), &perms); err == nil {
			g.Permissions = &perms
		}
	}

	c.JSON(http.StatusOK, g)
}

// UpdateGraph updates an existing graph
func (api *API) UpdateGraph(c *gin.Context) {
	db := getDB(c)
	userID := getUserID(c)
	id := c.Param("id")

	// Validate graph ID format
	if !IsValidGraphID(id) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid graph ID format"})
		return
	}

	// Check write permission
	hasPermission, err := CheckGraphPermission(db, id, userID, PermissionWrite)
	if err != nil {
		if err.Error() == "graph not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Graph not found"})
		} else {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		}
		return
	}
	if !hasPermission {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Access denied",
			Details: "You do not have permission to modify this graph",
		})
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
	userID := getUserID(c)
	id := c.Param("id")

	// Validate graph ID format
	if !IsValidGraphID(id) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid graph ID format"})
		return
	}

	// Check write permission (only owner or editors can delete)
	hasPermission, err := CheckGraphPermission(db, id, userID, PermissionWrite)
	if err != nil {
		if err.Error() == "graph not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Graph not found"})
		} else {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		}
		return
	}
	if !hasPermission {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Access denied",
			Details: "You do not have permission to delete this graph",
		})
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
	userID := getUserID(c)

	// Validate graph ID format
	if !IsValidGraphID(id) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid graph ID format"})
		return
	}

	// Check read permission (need to read the graph to validate it)
	hasPermission, err := CheckGraphPermission(db, id, userID, PermissionRead)
	if err != nil {
		if err.Error() == "graph not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Graph not found"})
		} else {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		}
		return
	}
	if !hasPermission {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Access denied",
			Details: "You do not have permission to validate this graph",
		})
		return
	}

	// Get graph type and data
	var graphType string
	var data sql.NullString

	err = db.QueryRow(
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
	plugin, exists := api.plugins[graphType]
	if !exists {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Unknown graph type"})
		return
	}
	if !plugin.Enabled {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Graph type is disabled"})
		return
	}

	result := api.computeValidationResult(graphType, data)

	c.JSON(http.StatusOK, result)
}

// computeValidationResult evaluates stored graph data and returns a validation report.
func (api *API) computeValidationResult(graphType string, data sql.NullString) ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	if !data.Valid {
		result.Errors = append(result.Errors, "Graph has no data to validate")
		result.Valid = false
		return result
	}

	trimmed := strings.TrimSpace(data.String)
	if trimmed == "" || trimmed == "{}" {
		result.Errors = append(result.Errors, "Graph data is empty")
		result.Valid = false
		return result
	}

	validationErr := api.validator.validateGraphData(graphType, json.RawMessage(data.String))
	if validationErr != nil {
		result.Errors = append(result.Errors, formatValidationError(validationErr))
		result.Valid = false
		return result
	}

	api.appendValidationWarnings(graphType, trimmed, &result)
	return result
}

func formatValidationError(err *ValidationError) string {
	if err == nil {
		return ""
	}

	message := err.Message
	if err.Field != "" {
		message = fmt.Sprintf("%s: %s", err.Field, err.Message)
	}
	if err.Value != "" {
		message = fmt.Sprintf("%s (%s)", message, err.Value)
	}
	return message
}

func (api *API) appendValidationWarnings(graphType, data string, result *ValidationResult) {
	switch graphType {
	case "bpmn":
		lower := strings.ToLower(data)
		if !strings.Contains(lower, "startevent") {
			result.Warnings = append(result.Warnings, "BPMN diagram should have a start event")
		}
		if !strings.Contains(lower, "endevent") {
			result.Warnings = append(result.Warnings, "BPMN diagram should have an end event")
		}
	case "mind-maps":
		lower := strings.ToLower(data)
		if !strings.Contains(lower, "\"root\"") && !strings.Contains(lower, "\"central\"") {
			result.Warnings = append(result.Warnings, "Mind map should have a root/central node")
		}
	}
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

	// Check read permission (need to read the graph to convert it)
	hasPermission, err := CheckGraphPermission(db, id, userID, PermissionRead)
	if err != nil {
		if err.Error() == "graph not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Graph not found"})
		} else {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		}
		return
	}
	if !hasPermission {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Access denied",
			Details: "You do not have permission to convert this graph",
		})
		return
	}

	var req ConversionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Load plugins from database (needed for validation)
	if err := api.getPluginsFromDB(db); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to load plugins",
			Details: err.Error(),
		})
		return
	}

	// Validate request (after plugins are loaded)
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

	err = db.QueryRow(
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
		"format":             req.TargetFormat,
		"success":            true,
	})
}

// RenderGraph renders a graph in various formats
func (api *API) RenderGraph(c *gin.Context) {
	db := getDB(c)
	id := c.Param("id")
	userID := getUserID(c)

	// Validate graph ID format
	if !IsValidGraphID(id) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid graph ID format"})
		return
	}

	// Check read permission (need to read the graph to render it)
	hasPermission, err := CheckGraphPermission(db, id, userID, PermissionRead)
	if err != nil {
		if err.Error() == "graph not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Graph not found"})
		} else {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		}
		return
	}
	if !hasPermission {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Access denied",
			Details: "You do not have permission to render this graph",
		})
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

	err = db.QueryRow(
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

// ExportGraph exports a graph to various formats (GraphML, GEXF, JSON)
func (api *API) ExportGraph(c *gin.Context) {
	db := getDB(c)
	id := c.Param("id")
	userID := getUserID(c)

	// Validate graph ID format
	if !IsValidGraphID(id) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid graph ID format"})
		return
	}

	// Check read permission
	hasPermission, err := CheckGraphPermission(db, id, userID, PermissionRead)
	if err != nil {
		if err.Error() == "graph not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Graph not found"})
		} else {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		}
		return
	}
	if !hasPermission {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Access denied",
			Details: "You do not have permission to export this graph",
		})
		return
	}

	var req ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// Fetch graph from database
	var graph Graph
	var dataBytes []byte
	var tagsJSON []byte
	var permissionsJSON []byte

	err = db.QueryRow(`
		SELECT id, name, type, description, data, version,
		       created_by, created_at, updated_at, tags, permissions
		FROM graphs
		WHERE id = $1
	`, id).Scan(
		&graph.ID, &graph.Name, &graph.Type, &graph.Description,
		&dataBytes, &graph.Version, &graph.CreatedBy,
		&graph.CreatedAt, &graph.UpdatedAt, &tagsJSON, &permissionsJSON,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Graph not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	graph.Data = json.RawMessage(dataBytes)
	if err := json.Unmarshal(tagsJSON, &graph.Tags); err != nil {
		graph.Tags = []string{}
	}

	var content string
	var mimeType string
	var filename string

	// Export based on requested format
	switch strings.ToLower(req.Format) {
	case "graphml":
		exporter := &GraphMLExporter{}
		content, err = exporter.Export(&graph)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Export failed",
				Details: err.Error(),
			})
			return
		}
		mimeType = "application/xml"
		filename = fmt.Sprintf("%s.graphml", sanitizeFilename(graph.Name))

	case "gexf":
		exporter := &GEXFExporter{}
		content, err = exporter.Export(&graph)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Export failed",
				Details: err.Error(),
			})
			return
		}
		mimeType = "application/xml"
		filename = fmt.Sprintf("%s.gexf", sanitizeFilename(graph.Name))

	case "json":
		// Export as JSON
		jsonData, err := json.MarshalIndent(graph, "", "  ")
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Export failed",
				Details: err.Error(),
			})
			return
		}
		content = string(jsonData)
		mimeType = "application/json"
		filename = fmt.Sprintf("%s.json", sanitizeFilename(graph.Name))

	default:
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Unsupported export format",
			Details: "Supported formats: graphml, gexf, json",
		})
		return
	}

	// Log the export
	api.metrics.LogEvent(db, id, graph.Type, userID, "exported", map[string]interface{}{
		"format": req.Format,
	}, 0)

	// Return export response
	c.JSON(http.StatusOK, ExportResponse{
		Format:   req.Format,
		Filename: filename,
		Content:  content,
		MimeType: mimeType,
	})
}

// sanitizeFilename removes unsafe characters from filenames
func sanitizeFilename(name string) string {
	// Replace unsafe characters with underscores
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	sanitized := replacer.Replace(name)

	// Limit length
	if len(sanitized) > 50 {
		sanitized = sanitized[:50]
	}

	return sanitized
}
