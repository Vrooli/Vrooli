package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"gopkg.in/yaml.v3"
)

type Schema struct {
	ID               uuid.UUID              `json:"id" db:"id"`
	Name             string                 `json:"name" db:"name"`
	Description      string                 `json:"description" db:"description"`
	SchemaDefinition map[string]interface{} `json:"schema_definition" db:"schema_definition"`
	ExampleData      map[string]interface{} `json:"example_data" db:"example_data"`
	Version          int                    `json:"version" db:"version"`
	IsActive         bool                   `json:"is_active" db:"is_active"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
	CreatedBy        string                 `json:"created_by" db:"created_by"`
	UsageCount       int                    `json:"usage_count,omitempty"`
	AvgConfidence    float64                `json:"avg_confidence,omitempty"`
}

type ProcessedData struct {
	ID               uuid.UUID              `json:"id" db:"id"`
	SchemaID         uuid.UUID              `json:"schema_id" db:"schema_id"`
	SourceFileName   string                 `json:"source_file_name" db:"source_file_name"`
	SourceFilePath   string                 `json:"source_file_path" db:"source_file_path"`
	SourceFileType   string                 `json:"source_file_type" db:"source_file_type"`
	SourceFileSize   int64                  `json:"source_file_size" db:"source_file_size"`
	RawContent       string                 `json:"raw_content" db:"raw_content"`
	StructuredData   map[string]interface{} `json:"structured_data" db:"structured_data"`
	ConfidenceScore  *float64               `json:"confidence_score" db:"confidence_score"`
	ProcessingStatus string                 `json:"processing_status" db:"processing_status"`
	ErrorMessage     string                 `json:"error_message" db:"error_message"`
	ProcessingTimeMs *int                   `json:"processing_time_ms" db:"processing_time_ms"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	ProcessedAt      *time.Time             `json:"processed_at" db:"processed_at"`
	Metadata         map[string]interface{} `json:"metadata" db:"metadata"`
}

type ProcessingRequest struct {
	SchemaID   uuid.UUID `json:"schema_id" binding:"required"`
	InputType  string    `json:"input_type" binding:"required,oneof=file text url"`
	InputData  string    `json:"input_data" binding:"required"`
	BatchMode  bool      `json:"batch_mode"`
	BatchItems []string  `json:"batch_items,omitempty"` // For batch mode: array of inputs
}

type ProcessingResponse struct {
	ProcessingID    uuid.UUID              `json:"processing_id"`
	Status          string                 `json:"status"`
	StructuredData  map[string]interface{} `json:"structured_data,omitempty"`
	ConfidenceScore *float64               `json:"confidence_score,omitempty"`
	Errors          []string               `json:"errors,omitempty"`
}

type BatchProcessingResponse struct {
	BatchID          uuid.UUID          `json:"batch_id"`
	Status           string             `json:"status"`
	TotalItems       int                `json:"total_items"`
	Completed        int                `json:"completed"`
	Failed           int                `json:"failed"`
	Results          []ProcessingResult `json:"results,omitempty"`
	AvgConfidence    *float64           `json:"avg_confidence,omitempty"`
	ProcessingTimeMs int                `json:"processing_time_ms"`
}

type ProcessingResult struct {
	ProcessingID    uuid.UUID              `json:"processing_id"`
	Status          string                 `json:"status"`
	StructuredData  map[string]interface{} `json:"structured_data,omitempty"`
	ConfidenceScore *float64               `json:"confidence_score,omitempty"`
	Error           string                 `json:"error,omitempty"`
}

type SchemaTemplate struct {
	ID               uuid.UUID              `json:"id" db:"id"`
	Name             string                 `json:"name" db:"name"`
	Category         string                 `json:"category" db:"category"`
	Description      string                 `json:"description" db:"description"`
	SchemaDefinition map[string]interface{} `json:"schema_definition" db:"schema_definition"`
	ExampleData      map[string]interface{} `json:"example_data" db:"example_data"`
	UsageCount       int                    `json:"usage_count" db:"usage_count"`
	IsPublic         bool                   `json:"is_public" db:"is_public"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
	Tags             []string               `json:"tags" db:"tags"`
}

var db *sql.DB

func configureUIServing(router *gin.Engine) string {
	distDir := filepath.Join("..", "ui", "dist")
	distIndex := filepath.Join(distDir, "index.html")

	if _, err := os.Stat(distIndex); err == nil {
		assetsDir := filepath.Join(distDir, "assets")
		if info, err := os.Stat(assetsDir); err == nil && info.IsDir() {
			router.StaticFS("/assets", gin.Dir(assetsDir, true))
		}
		if _, err := os.Stat(filepath.Join(distDir, "favicon.svg")); err == nil {
			router.StaticFile("/favicon.svg", filepath.Join(distDir, "favicon.svg"))
		}
		router.GET("/", func(c *gin.Context) {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.File(distIndex)
		})
		router.GET("/index.html", func(c *gin.Context) {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.File(distIndex)
		})
		return distIndex
	}

	fallback := filepath.Join("..", "ui", "index.html")
	if _, err := os.Stat(fallback); err == nil {
		router.GET("/", func(c *gin.Context) {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.File(fallback)
		})
		if _, err := os.Stat(filepath.Join("..", "ui", "favicon.svg")); err == nil {
			router.StaticFile("/favicon.svg", filepath.Join("..", "ui", "favicon.svg"))
		}
		return fallback
	}

	log.Printf("⚠️  UI bundle not found at %s or %s", distIndex, fallback)
	return ""
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "data-structurer",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Load environment variables
	godotenv.Load()

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

	// Connect to database using api-core with automatic retry
	var err error
	db, err = database.Connect(context.Background(), database.Config{
		Driver: "postgres",
	})
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}
	defer db.Close()

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("🎉 Database connection pool established successfully!")

	// Initialize Gin router
	r := gin.Default()

	// Configure trusted proxies (localhost only for development)
	if err := r.SetTrustedProxies([]string{"127.0.0.1", "::1"}); err != nil {
		log.Printf("⚠️  Warning: Failed to set trusted proxies: %v", err)
	}

	// CORS middleware - restricted to localhost origins for security
	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		// Only allow localhost origins
		allowedOrigins := []string{
			"http://localhost",
			"https://localhost",
			"http://127.0.0.1",
			"https://127.0.0.1",
		}

		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if strings.HasPrefix(origin, allowedOrigin) {
				allowed = true
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}

		if !allowed && origin != "" {
			// For non-localhost origins, don't set CORS headers
			log.Printf("⚠️  Blocked CORS request from non-localhost origin: %s", origin)
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	uiIndexPath := configureUIServing(r)

	// Health check endpoint - using standardized api-core/health
	healthHandler := health.New().Version("1.0.0").Check(health.DB(db), health.Critical).Handler()
	r.GET("/health", gin.WrapF(healthHandler))

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/") || path == "/health" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
			return
		}
		if uiIndexPath != "" {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.File(uiIndexPath)
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
	})

	// API routes
	api := r.Group("/api/v1")

	// Schema endpoints
	api.GET("/schemas", getSchemas)
	api.POST("/schemas", createSchema)
	api.GET("/schemas/:id", getSchema)
	api.PUT("/schemas/:id", updateSchema)
	api.DELETE("/schemas/:id", deleteSchema)

	// Schema template endpoints
	api.GET("/schema-templates", getSchemaTemplates)
	api.GET("/schema-templates/:id", getSchemaTemplate)
	api.POST("/schemas/from-template/:template_id", createSchemaFromTemplate)

	// Data processing endpoints
	api.POST("/process", processData)
	api.GET("/process/:id", getProcessingResult)
	api.GET("/data/:schema_id", getProcessedData)

	// Processing jobs endpoints
	api.GET("/jobs", getProcessingJobs)
	api.GET("/jobs/:id", getProcessingJob)

	log.Printf("Data Structurer API starting on port %s", port)
	log.Fatal(r.Run(":" + port))
}

// Schema CRUD operations
func getSchemas(c *gin.Context) {
	query := `
		SELECT s.id, s.name, s.description, s.schema_definition, s.example_data, 
		       s.version, s.is_active, s.created_at, s.updated_at, s.created_by,
		       COALESCE(COUNT(pd.id), 0) as usage_count,
		       COALESCE(AVG(pd.confidence_score), 0) as avg_confidence
		FROM schemas s
		LEFT JOIN processed_data pd ON s.id = pd.schema_id
		WHERE s.is_active = true
		GROUP BY s.id, s.name, s.description, s.schema_definition, s.example_data, 
		         s.version, s.is_active, s.created_at, s.updated_at, s.created_by
		ORDER BY s.created_at DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch schemas"})
		return
	}
	defer rows.Close()

	var schemas []Schema
	for rows.Next() {
		var s Schema
		var schemaDefBytes, exampleDataBytes []byte

		err := rows.Scan(
			&s.ID, &s.Name, &s.Description, &schemaDefBytes, &exampleDataBytes,
			&s.Version, &s.IsActive, &s.CreatedAt, &s.UpdatedAt, &s.CreatedBy,
			&s.UsageCount, &s.AvgConfidence,
		)
		if err != nil {
			continue
		}

		json.Unmarshal(schemaDefBytes, &s.SchemaDefinition)
		json.Unmarshal(exampleDataBytes, &s.ExampleData)
		schemas = append(schemas, s)
	}

	c.JSON(http.StatusOK, gin.H{
		"schemas": schemas,
		"count":   len(schemas),
	})
}

func createSchema(c *gin.Context) {
	var req struct {
		Name             string                 `json:"name" binding:"required"`
		Description      string                 `json:"description"`
		SchemaDefinition map[string]interface{} `json:"schema_definition" binding:"required"`
		ExampleData      map[string]interface{} `json:"example_data"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := uuid.New()
	schemaDefBytes, _ := json.Marshal(req.SchemaDefinition)
	exampleDataBytes, _ := json.Marshal(req.ExampleData)

	query := `
		INSERT INTO schemas (id, name, description, schema_definition, example_data, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	var createdAt time.Time
	err := db.QueryRow(query, id, req.Name, req.Description, schemaDefBytes, exampleDataBytes, "api").Scan(&id, &createdAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			c.JSON(http.StatusConflict, gin.H{"error": "Schema name already exists"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create schema"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         id,
		"name":       req.Name,
		"status":     "created",
		"created_at": createdAt,
	})
}

func getSchema(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schema ID"})
		return
	}

	query := `
		SELECT s.id, s.name, s.description, s.schema_definition, s.example_data, 
		       s.version, s.is_active, s.created_at, s.updated_at, s.created_by,
		       COALESCE(COUNT(pd.id), 0) as usage_count,
		       COALESCE(AVG(pd.confidence_score), 0) as avg_confidence
		FROM schemas s
		LEFT JOIN processed_data pd ON s.id = pd.schema_id
		WHERE s.id = $1
		GROUP BY s.id, s.name, s.description, s.schema_definition, s.example_data, 
		         s.version, s.is_active, s.created_at, s.updated_at, s.created_by
	`

	var s Schema
	var schemaDefBytes, exampleDataBytes []byte

	err = db.QueryRow(query, id).Scan(
		&s.ID, &s.Name, &s.Description, &schemaDefBytes, &exampleDataBytes,
		&s.Version, &s.IsActive, &s.CreatedAt, &s.UpdatedAt, &s.CreatedBy,
		&s.UsageCount, &s.AvgConfidence,
	)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schema not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch schema"})
		return
	}

	json.Unmarshal(schemaDefBytes, &s.SchemaDefinition)
	json.Unmarshal(exampleDataBytes, &s.ExampleData)

	c.JSON(http.StatusOK, s)
}

func updateSchema(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schema ID"})
		return
	}

	var req struct {
		Description      string                 `json:"description"`
		SchemaDefinition map[string]interface{} `json:"schema_definition"`
		ExampleData      map[string]interface{} `json:"example_data"`
		IsActive         *bool                  `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Description != "" {
		updates = append(updates, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, req.Description)
		argIndex++
	}
	if req.SchemaDefinition != nil {
		schemaDefBytes, _ := json.Marshal(req.SchemaDefinition)
		updates = append(updates, fmt.Sprintf("schema_definition = $%d", argIndex))
		args = append(args, schemaDefBytes)
		argIndex++
	}
	if req.ExampleData != nil {
		exampleDataBytes, _ := json.Marshal(req.ExampleData)
		updates = append(updates, fmt.Sprintf("example_data = $%d", argIndex))
		args = append(args, exampleDataBytes)
		argIndex++
	}
	if req.IsActive != nil {
		updates = append(updates, fmt.Sprintf("is_active = $%d", argIndex))
		args = append(args, *req.IsActive)
		argIndex++
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	updates = append(updates, fmt.Sprintf("updated_at = CURRENT_TIMESTAMP, version = version + 1"))
	args = append(args, id)

	query := fmt.Sprintf("UPDATE schemas SET %s WHERE id = $%d", strings.Join(updates, ", "), argIndex)

	result, err := db.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update schema"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schema not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":     id,
		"status": "updated",
	})
}

func deleteSchema(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schema ID"})
		return
	}

	// Soft delete by setting is_active to false
	query := "UPDATE schemas SET is_active = false, updated_at = CURRENT_TIMESTAMP WHERE id = $1"
	result, err := db.Exec(query, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete schema"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schema not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":     id,
		"status": "deleted",
	})
}

// Schema Templates
func getSchemaTemplates(c *gin.Context) {
	category := c.Query("category")
	query := `
		SELECT id, name, category, description, schema_definition, example_data, 
		       usage_count, is_public, created_at, updated_at, tags
		FROM schema_templates 
		WHERE is_public = true
	`
	args := []interface{}{}

	if category != "" {
		query += " AND category = $1"
		args = append(args, category)
	}

	query += " ORDER BY usage_count DESC, created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch templates"})
		return
	}
	defer rows.Close()

	var templates []SchemaTemplate
	for rows.Next() {
		var t SchemaTemplate
		var schemaDefBytes, exampleDataBytes []byte
		var tagsStr string

		err := rows.Scan(
			&t.ID, &t.Name, &t.Category, &t.Description, &schemaDefBytes, &exampleDataBytes,
			&t.UsageCount, &t.IsPublic, &t.CreatedAt, &t.UpdatedAt, &tagsStr,
		)
		if err != nil {
			continue
		}

		json.Unmarshal(schemaDefBytes, &t.SchemaDefinition)
		json.Unmarshal(exampleDataBytes, &t.ExampleData)

		// Parse tags array from PostgreSQL format
		if tagsStr != "" {
			t.Tags = strings.Split(strings.Trim(tagsStr, "{}"), ",")
		}

		templates = append(templates, t)
	}

	c.JSON(http.StatusOK, gin.H{
		"templates": templates,
		"count":     len(templates),
	})
}

func getSchemaTemplate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	query := `
		SELECT id, name, category, description, schema_definition, example_data, 
		       usage_count, is_public, created_at, updated_at, tags
		FROM schema_templates 
		WHERE id = $1 AND is_public = true
	`

	var t SchemaTemplate
	var schemaDefBytes, exampleDataBytes []byte
	var tagsStr string

	err = db.QueryRow(query, id).Scan(
		&t.ID, &t.Name, &t.Category, &t.Description, &schemaDefBytes, &exampleDataBytes,
		&t.UsageCount, &t.IsPublic, &t.CreatedAt, &t.UpdatedAt, &tagsStr,
	)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch template"})
		return
	}

	json.Unmarshal(schemaDefBytes, &t.SchemaDefinition)
	json.Unmarshal(exampleDataBytes, &t.ExampleData)

	if tagsStr != "" {
		t.Tags = strings.Split(strings.Trim(tagsStr, "{}"), ",")
	}

	c.JSON(http.StatusOK, t)
}

func createSchemaFromTemplate(c *gin.Context) {
	templateIDStr := c.Param("template_id")
	templateID, err := uuid.Parse(templateIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid template ID"})
		return
	}

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get template
	var schemaDefBytes, exampleDataBytes []byte
	templateQuery := `
		SELECT schema_definition, example_data FROM schema_templates 
		WHERE id = $1 AND is_public = true
	`
	err = db.QueryRow(templateQuery, templateID).Scan(&schemaDefBytes, &exampleDataBytes)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch template"})
		return
	}

	// Create schema from template
	id := uuid.New()
	query := `
		INSERT INTO schemas (id, name, description, schema_definition, example_data, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at
	`

	var createdAt time.Time
	err = db.QueryRow(query, id, req.Name, req.Description, schemaDefBytes, exampleDataBytes, "api").Scan(&createdAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			c.JSON(http.StatusConflict, gin.H{"error": "Schema name already exists"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create schema"})
		}
		return
	}

	// Update template usage count
	db.Exec("UPDATE schema_templates SET usage_count = usage_count + 1 WHERE id = $1", templateID)

	c.JSON(http.StatusCreated, gin.H{
		"id":          id,
		"name":        req.Name,
		"status":      "created",
		"created_at":  createdAt,
		"template_id": templateID,
	})
}

// Data Processing
func processData(c *gin.Context) {
	var req ProcessingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate schema exists
	var schemaName string
	schemaQuery := "SELECT name FROM schemas WHERE id = $1 AND is_active = true"
	err := db.QueryRow(schemaQuery, req.SchemaID).Scan(&schemaName)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schema not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate schema"})
		return
	}

	// Handle batch mode
	if req.BatchMode {
		processBatchData(c, req)
		return
	}

	// Single item processing
	processingID := uuid.New()
	startTime := time.Now()

	structuredData, confidence, err := performDataProcessing(req.InputType, req.InputData, req.SchemaID)
	processingTime := int(time.Since(startTime).Milliseconds())

	status := "completed"
	var errorMsg string
	if err != nil {
		status = "failed"
		errorMsg = err.Error()
	}

	// Store processing result
	structuredDataBytes, _ := json.Marshal(structuredData)
	metadata := map[string]interface{}{
		"input_type":         req.InputType,
		"processing_time_ms": processingTime,
	}
	metadataBytes, _ := json.Marshal(metadata)

	insertQuery := `
		INSERT INTO processed_data
		(id, schema_id, source_file_name, raw_content, structured_data, confidence_score,
		 processing_status, error_message, processing_time_ms, processed_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err = db.Exec(insertQuery,
		processingID, req.SchemaID, "api_input", req.InputData, structuredDataBytes,
		confidence, status, errorMsg, processingTime, time.Now(), metadataBytes)
	if err != nil {
		log.Printf("Failed to store processing result: %v", err)
	}

	response := ProcessingResponse{
		ProcessingID: processingID,
		Status:       status,
	}

	if status == "completed" {
		response.StructuredData = structuredData
		response.ConfidenceScore = confidence
	} else {
		response.Errors = []string{errorMsg}
	}

	c.JSON(http.StatusOK, response)
}

func processBatchData(c *gin.Context, req ProcessingRequest) {
	if len(req.BatchItems) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "batch_items required for batch mode"})
		return
	}

	batchID := uuid.New()
	startTime := time.Now()

	var results []ProcessingResult
	completedCount := 0
	failedCount := 0
	var confidenceSum float64
	var confidenceCount int

	// Process each item in the batch
	for _, inputData := range req.BatchItems {
		processingID := uuid.New()
		itemStartTime := time.Now()

		structuredData, confidence, err := performDataProcessing(req.InputType, inputData, req.SchemaID)
		itemProcessingTime := int(time.Since(itemStartTime).Milliseconds())

		result := ProcessingResult{
			ProcessingID: processingID,
		}

		if err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			failedCount++
		} else {
			result.Status = "completed"
			result.StructuredData = structuredData
			result.ConfidenceScore = confidence
			completedCount++

			if confidence != nil {
				confidenceSum += *confidence
				confidenceCount++
			}
		}

		results = append(results, result)

		// Store individual result in database
		status := result.Status
		var errorMsg string
		if result.Error != "" {
			errorMsg = result.Error
		}

		structuredDataBytes, _ := json.Marshal(structuredData)
		metadata := map[string]interface{}{
			"input_type":         req.InputType,
			"processing_time_ms": itemProcessingTime,
			"batch_id":           batchID.String(),
		}
		metadataBytes, _ := json.Marshal(metadata)

		insertQuery := `
			INSERT INTO processed_data
			(id, schema_id, source_file_name, raw_content, structured_data, confidence_score,
			 processing_status, error_message, processing_time_ms, processed_at, metadata)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`

		_, dbErr := db.Exec(insertQuery,
			processingID, req.SchemaID, "batch_input", inputData, structuredDataBytes,
			confidence, status, errorMsg, itemProcessingTime, time.Now(), metadataBytes)
		if dbErr != nil {
			log.Printf("Failed to store batch processing result: %v", dbErr)
		}
	}

	totalProcessingTime := int(time.Since(startTime).Milliseconds())

	// Calculate average confidence
	var avgConfidence *float64
	if confidenceCount > 0 {
		avg := confidenceSum / float64(confidenceCount)
		avgConfidence = &avg
	}

	// Determine overall batch status
	batchStatus := "completed"
	if failedCount == len(req.BatchItems) {
		batchStatus = "failed"
	} else if failedCount > 0 {
		batchStatus = "partial"
	}

	response := BatchProcessingResponse{
		BatchID:          batchID,
		Status:           batchStatus,
		TotalItems:       len(req.BatchItems),
		Completed:        completedCount,
		Failed:           failedCount,
		Results:          results,
		AvgConfidence:    avgConfidence,
		ProcessingTimeMs: totalProcessingTime,
	}

	c.JSON(http.StatusOK, response)
}

func performDataProcessing(inputType, inputData string, schemaID uuid.UUID) (map[string]interface{}, *float64, error) {
	// This is a simplified implementation
	// In production, this would use unstructured-io for document parsing
	// and ollama for intelligent extraction based on the schema

	var rawContent string
	var err error

	switch inputType {
	case "text":
		rawContent = inputData
	case "file":
		// For file processing, we would use unstructured-io
		rawContent, err = extractFileContent(inputData)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to extract file content: %v", err)
		}
	case "url":
		// For URL processing, we would fetch and extract content
		return nil, nil, fmt.Errorf("URL processing not yet implemented")
	default:
		return nil, nil, fmt.Errorf("unsupported input type: %s", inputType)
	}

	// Get schema definition for context
	var schemaDefBytes []byte
	schemaQuery := "SELECT schema_definition FROM schemas WHERE id = $1"
	err = db.QueryRow(schemaQuery, schemaID).Scan(&schemaDefBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get schema: %v", err)
	}

	// Use Ollama for intelligent extraction
	structuredData, confidence, err := extractWithOllama(rawContent, string(schemaDefBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract with AI: %v", err)
	}

	return structuredData, &confidence, nil
}

func extractFileContent(filePath string) (string, error) {
	// Use unstructured-io for document parsing
	unstructuredURL := os.Getenv("UNSTRUCTURED_IO_URL")
	if unstructuredURL == "" {
		unstructuredURL = "http://localhost:11450"
	}

	// For now, handle text files directly
	// TODO: Implement full unstructured-io integration for PDFs, DOCX, images
	if strings.HasSuffix(filePath, ".txt") {
		cmd := exec.Command("cat", filePath)
		output, err := cmd.Output()
		if err != nil {
			return "", err
		}
		return string(output), nil
	}

	// For other file types, use unstructured-io API
	// This is a placeholder for the full implementation
	return fmt.Sprintf("Content extraction from %s pending unstructured-io integration", filePath), nil
}

func extractWithOllama(content, schema string) (map[string]interface{}, float64, error) {
	// Use Ollama for intelligent content extraction based on schema
	ollamaURL := os.Getenv("OLLAMA_API_BASE")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	// Create extraction prompt
	prompt := fmt.Sprintf(`Extract structured data from the following content according to the schema.
Return ONLY valid JSON that matches the schema structure.

Schema:
%s

Content:
%s

Extracted JSON:`, schema, content)

	// Call Ollama API
	requestBody := map[string]interface{}{
		"model":  "llama3.2",
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"temperature": 0.1,
			"top_k":       10,
			"top_p":       0.1,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", ollamaURL+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to call Ollama: %v", err)
	}
	defer resp.Body.Close()

	var ollamaResponse struct {
		Response string `json:"response"`
		Done     bool   `json:"done"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ollamaResponse); err != nil {
		return nil, 0, fmt.Errorf("failed to decode Ollama response: %v", err)
	}

	// Try to parse the extracted JSON
	var extractedData map[string]interface{}
	responseText := strings.TrimSpace(ollamaResponse.Response)

	// Find JSON in the response
	startIdx := strings.Index(responseText, "{")
	endIdx := strings.LastIndex(responseText, "}")
	if startIdx >= 0 && endIdx >= 0 && endIdx >= startIdx {
		jsonStr := responseText[startIdx : endIdx+1]
		if err := json.Unmarshal([]byte(jsonStr), &extractedData); err != nil {
			// Fallback to simpler extraction
			extractedData = map[string]interface{}{
				"raw_extraction":    content,
				"extraction_method": "ollama_fallback",
				"error":             err.Error(),
			}
		}
	} else {
		// Fallback for non-JSON response
		extractedData = map[string]interface{}{
			"raw_extraction":    content,
			"extraction_method": "ollama_text",
			"ollama_response":   responseText,
		}
	}

	// Calculate confidence based on extraction success
	confidence := 0.95
	if _, hasError := extractedData["error"]; hasError {
		confidence = 0.5
	}

	return extractedData, confidence, nil
}

func getProcessingResult(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid processing ID"})
		return
	}

	query := `
		SELECT id, schema_id, source_file_name, source_file_path, source_file_type,
		       source_file_size, raw_content, structured_data, confidence_score,
		       processing_status, error_message, processing_time_ms, created_at,
		       processed_at, metadata
		FROM processed_data 
		WHERE id = $1
	`

	var pd ProcessedData
	var structuredDataBytes, metadataBytes []byte

	err = db.QueryRow(query, id).Scan(
		&pd.ID, &pd.SchemaID, &pd.SourceFileName, &pd.SourceFilePath, &pd.SourceFileType,
		&pd.SourceFileSize, &pd.RawContent, &structuredDataBytes, &pd.ConfidenceScore,
		&pd.ProcessingStatus, &pd.ErrorMessage, &pd.ProcessingTimeMs, &pd.CreatedAt,
		&pd.ProcessedAt, &metadataBytes,
	)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Processing result not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch processing result"})
		return
	}

	json.Unmarshal(structuredDataBytes, &pd.StructuredData)
	json.Unmarshal(metadataBytes, &pd.Metadata)

	c.JSON(http.StatusOK, pd)
}

func getProcessedData(c *gin.Context) {
	schemaIDStr := c.Param("schema_id")
	schemaID, err := uuid.Parse(schemaIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schema ID"})
		return
	}

	limit := 100
	offset := 0
	format := c.Query("format") // json, csv, yaml

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Get schema info
	var schemaName, schemaDesc string
	schemaQuery := "SELECT name, description FROM schemas WHERE id = $1"
	err = db.QueryRow(schemaQuery, schemaID).Scan(&schemaName, &schemaDesc)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schema not found"})
		return
	}

	// Get processed data
	query := `
		SELECT id, source_file_name, structured_data, confidence_score,
		       processing_status, created_at, processed_at, metadata
		FROM processed_data
		WHERE schema_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.Query(query, schemaID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch processed data"})
		return
	}
	defer rows.Close()

	var data []map[string]interface{}
	for rows.Next() {
		var item map[string]interface{} = make(map[string]interface{})
		var structuredDataBytes, metadataBytes []byte
		var id uuid.UUID
		var fileName, status string
		var confidence *float64
		var createdAt time.Time
		var processedAt *time.Time

		err := rows.Scan(&id, &fileName, &structuredDataBytes, &confidence,
			&status, &createdAt, &processedAt, &metadataBytes)
		if err != nil {
			continue
		}

		var structuredData, metadata map[string]interface{}
		json.Unmarshal(structuredDataBytes, &structuredData)
		json.Unmarshal(metadataBytes, &metadata)

		item["id"] = id
		item["source_file_name"] = fileName
		item["structured_data"] = structuredData
		item["confidence_score"] = confidence
		item["processing_status"] = status
		item["created_at"] = createdAt
		item["processed_at"] = processedAt
		item["metadata"] = metadata

		data = append(data, item)
	}

	// Get total count
	var totalCount int
	countQuery := "SELECT COUNT(*) FROM processed_data WHERE schema_id = $1"
	db.QueryRow(countQuery, schemaID).Scan(&totalCount)

	// Handle different export formats
	switch format {
	case "csv":
		exportAsCSV(c, schemaName, data)
	case "yaml":
		exportAsYAML(c, schemaName, data)
	default:
		// Default JSON response
		c.JSON(http.StatusOK, gin.H{
			"schema": gin.H{
				"id":          schemaID,
				"name":        schemaName,
				"description": schemaDesc,
			},
			"data": data,
			"pagination": gin.H{
				"limit":       limit,
				"offset":      offset,
				"total_count": totalCount,
				"has_more":    offset+len(data) < totalCount,
			},
		})
	}
}

func exportAsCSV(c *gin.Context, schemaName string, data []map[string]interface{}) {
	if len(data) == 0 {
		c.String(http.StatusOK, "")
		return
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Collect all unique keys from structured_data
	keySet := make(map[string]bool)
	for _, item := range data {
		if structuredData, ok := item["structured_data"].(map[string]interface{}); ok {
			for key := range structuredData {
				keySet[key] = true
			}
		}
	}

	// Build header
	headers := []string{"id", "source_file_name", "confidence_score", "processing_status", "created_at"}
	for key := range keySet {
		headers = append(headers, key)
	}
	writer.Write(headers)

	// Write rows
	for _, item := range data {
		// Format confidence score properly (handle nil pointer)
		confidenceStr := ""
		if confidence, ok := item["confidence_score"].(*float64); ok && confidence != nil {
			confidenceStr = fmt.Sprintf("%.2f", *confidence)
		}

		row := []string{
			fmt.Sprintf("%v", item["id"]),
			fmt.Sprintf("%v", item["source_file_name"]),
			confidenceStr,
			fmt.Sprintf("%v", item["processing_status"]),
			fmt.Sprintf("%v", item["created_at"]),
		}

		if structuredData, ok := item["structured_data"].(map[string]interface{}); ok {
			for key := range keySet {
				if val, exists := structuredData[key]; exists {
					row = append(row, fmt.Sprintf("%v", val))
				} else {
					row = append(row, "")
				}
			}
		}

		writer.Write(row)
	}

	writer.Flush()

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.csv", schemaName))
	c.String(http.StatusOK, buf.String())
}

func exportAsYAML(c *gin.Context, schemaName string, data []map[string]interface{}) {
	// Convert data to JSON-serializable format first
	jsonData, err := json.Marshal(data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare data for YAML"})
		return
	}

	// Parse back to generic interface for YAML
	var yamlReady interface{}
	if err := json.Unmarshal(jsonData, &yamlReady); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare data for YAML"})
		return
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(yamlReady)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate YAML"})
		return
	}

	c.Header("Content-Type", "application/x-yaml")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.yaml", schemaName))
	c.String(http.StatusOK, string(yamlData))
}

// Processing Jobs (simplified for now)
func getProcessingJobs(c *gin.Context) {
	query := `
		SELECT id, schema_id, input_type, status, priority, total_items,
		       processed_items, failed_items, created_at, started_at, completed_at
		FROM processing_jobs 
		ORDER BY priority DESC, created_at ASC
		LIMIT 50
	`

	rows, err := db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch jobs"})
		return
	}
	defer rows.Close()

	var jobs []map[string]interface{}
	for rows.Next() {
		var job map[string]interface{} = make(map[string]interface{})
		var id, schemaID uuid.UUID
		var inputType, status string
		var priority, totalItems, processedItems, failedItems int
		var createdAt time.Time
		var startedAt, completedAt *time.Time

		err := rows.Scan(&id, &schemaID, &inputType, &status, &priority, &totalItems,
			&processedItems, &failedItems, &createdAt, &startedAt, &completedAt)
		if err != nil {
			continue
		}

		job["id"] = id
		job["schema_id"] = schemaID
		job["input_type"] = inputType
		job["status"] = status
		job["priority"] = priority
		job["total_items"] = totalItems
		job["processed_items"] = processedItems
		job["failed_items"] = failedItems
		job["created_at"] = createdAt
		job["started_at"] = startedAt
		job["completed_at"] = completedAt

		jobs = append(jobs, job)
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":  jobs,
		"count": len(jobs),
	})
}

func getProcessingJob(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	query := `
		SELECT id, schema_id, input_type, input_data, batch_mode, total_items,
		       processed_items, failed_items, status, priority, created_at,
		       started_at, completed_at, error_details, result_summary
		FROM processing_jobs 
		WHERE id = $1
	`

	var job map[string]interface{} = make(map[string]interface{})
	var jobID, schemaID uuid.UUID
	var inputType, inputData, status string
	var batchMode bool
	var totalItems, processedItems, failedItems, priority int
	var createdAt time.Time
	var startedAt, completedAt *time.Time
	var errorDetailsBytes, resultSummaryBytes []byte

	err = db.QueryRow(query, id).Scan(
		&jobID, &schemaID, &inputType, &inputData, &batchMode, &totalItems,
		&processedItems, &failedItems, &status, &priority, &createdAt,
		&startedAt, &completedAt, &errorDetailsBytes, &resultSummaryBytes,
	)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch job"})
		return
	}

	var errorDetails, resultSummary map[string]interface{}
	json.Unmarshal(errorDetailsBytes, &errorDetails)
	json.Unmarshal(resultSummaryBytes, &resultSummary)

	job["id"] = jobID
	job["schema_id"] = schemaID
	job["input_type"] = inputType
	job["input_data"] = inputData
	job["batch_mode"] = batchMode
	job["total_items"] = totalItems
	job["processed_items"] = processedItems
	job["failed_items"] = failedItems
	job["status"] = status
	job["priority"] = priority
	job["created_at"] = createdAt
	job["started_at"] = startedAt
	job["completed_at"] = completedAt
	job["error_details"] = errorDetails
	job["result_summary"] = resultSummary

	c.JSON(http.StatusOK, job)
}
