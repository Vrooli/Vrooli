package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
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
	AvgConfidence    float64               `json:"avg_confidence,omitempty"`
}

type ProcessedData struct {
	ID              uuid.UUID              `json:"id" db:"id"`
	SchemaID        uuid.UUID              `json:"schema_id" db:"schema_id"`
	SourceFileName  string                 `json:"source_file_name" db:"source_file_name"`
	SourceFilePath  string                 `json:"source_file_path" db:"source_file_path"`
	SourceFileType  string                 `json:"source_file_type" db:"source_file_type"`
	SourceFileSize  int64                  `json:"source_file_size" db:"source_file_size"`
	RawContent      string                 `json:"raw_content" db:"raw_content"`
	StructuredData  map[string]interface{} `json:"structured_data" db:"structured_data"`
	ConfidenceScore *float64               `json:"confidence_score" db:"confidence_score"`
	ProcessingStatus string                `json:"processing_status" db:"processing_status"`
	ErrorMessage    string                 `json:"error_message" db:"error_message"`
	ProcessingTimeMs *int                  `json:"processing_time_ms" db:"processing_time_ms"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	ProcessedAt     *time.Time             `json:"processed_at" db:"processed_at"`
	Metadata        map[string]interface{} `json:"metadata" db:"metadata"`
}

type ProcessingRequest struct {
	SchemaID   uuid.UUID `json:"schema_id" binding:"required"`
	InputType  string    `json:"input_type" binding:"required,oneof=file text url"`
	InputData  string    `json:"input_data" binding:"required"`
	BatchMode  bool      `json:"batch_mode"`
}

type ProcessingResponse struct {
	ProcessingID    uuid.UUID              `json:"processing_id"`
	Status          string                 `json:"status"`
	StructuredData  map[string]interface{} `json:"structured_data,omitempty"`
	ConfidenceScore *float64               `json:"confidence_score,omitempty"`
	Errors          []string               `json:"errors,omitempty"`
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

func main() {
	// Load environment variables
	godotenv.Load()

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

	// Database configuration - support both DATABASE_URL and individual components
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Try to build from individual components
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")
		
		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Fatal("❌ Missing database configuration. Provide DATABASE_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}
		
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}
	
	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second
	
	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Printf("📆 Database URL configured")
	
	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			log.Printf("✅ Database connected successfully on attempt %d", attempt + 1)
			break
		}
		
		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay) * math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))
		
		// Add progressive jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))
		actualDelay := delay + jitter
		
		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt + 1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)
		
		// Provide detailed status every few attempts
		if attempt > 0 && attempt % 3 == 0 {
			log.Printf("📈 Retry progress:")
			log.Printf("   - Attempts made: %d/%d", attempt + 1, maxRetries)
			log.Printf("   - Total wait time: ~%v", time.Duration(attempt * 2) * baseDelay)
			log.Printf("   - Current delay: %v (with jitter: %v)", delay, jitter)
		}
		
		time.Sleep(actualDelay)
	}
	
	if pingErr != nil {
		log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}
	
	log.Println("🎉 Database connection pool established successfully!")

	// Initialize Gin router
	r := gin.Default()

	// CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"service": "data-structurer-api",
			"timestamp": time.Now().Unix(),
			"database": "connected",
		})
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
		"count": len(schemas),
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
		"id": id,
		"name": req.Name,
		"status": "created",
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
		"id": id,
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
		"id": id,
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
		"count": len(templates),
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
		"id": id,
		"name": req.Name,
		"status": "created",
		"created_at": createdAt,
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

	// Create processing job
	processingID := uuid.New()
	startTime := time.Now()

	// For now, we'll do synchronous processing for simple cases
	// In production, this should be async with a job queue
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
		"input_type": req.InputType,
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
		Status: status,
	}

	if status == "completed" {
		response.StructuredData = structuredData
		response.ConfidenceScore = confidence
	} else {
		response.Errors = []string{errorMsg}
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
	// This would integrate with unstructured-io
	// For now, just read plain text files
	cmd := exec.Command("cat", filePath)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func extractWithOllama(content, schema string) (map[string]interface{}, float64, error) {
	// This is a placeholder - in production this would call the shared ollama.json workflow
	// or use the ollama CLI directly to perform intelligent extraction

	// For demo purposes, return a simple extraction
	result := map[string]interface{}{
		"extracted_content": content,
		"extraction_method": "demo_mode",
		"timestamp": time.Now().Unix(),
	}

	// Mock confidence score
	confidence := 0.75

	return result, confidence, nil
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

	c.JSON(http.StatusOK, gin.H{
		"schema": gin.H{
			"id": schemaID,
			"name": schemaName,
			"description": schemaDesc,
		},
		"data": data,
		"pagination": gin.H{
			"limit": limit,
			"offset": offset,
			"total_count": totalCount,
			"has_more": offset + len(data) < totalCount,
		},
	})
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
		"jobs": jobs,
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