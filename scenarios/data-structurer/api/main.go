package main

import (
	"context"
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

// HealthResponse represents the schema-compliant health check response
type HealthResponse struct {
	Status       string                 `json:"status"`
	Service      string                 `json:"service"`
	Timestamp    string                 `json:"timestamp"`
	Readiness    bool                   `json:"readiness"`
	Version      string                 `json:"version"`
	Dependencies map[string]interface{} `json:"dependencies"`
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
	DataStats    map[string]interface{} `json:"data_stats,omitempty"`
	Errors       []map[string]interface{} `json:"errors,omitempty"`
}

// handleHealthCheck implements comprehensive health checking
func handleHealthCheck(c *gin.Context, database *sql.DB) {
	start := time.Now()
	overallStatus := "healthy"
	var errors []map[string]interface{}
	readiness := true

	// Schema-compliant health response structure
	healthResponse := HealthResponse{
		Status:       overallStatus,
		Service:      "data-structurer-api",
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Readiness:    true,
		Version:      "1.0.0",
		Dependencies: make(map[string]interface{}),
	}

	// Check PostgreSQL connectivity
	dbHealth := checkDatabaseHealth(database)
	healthResponse.Dependencies["postgres"] = dbHealth
	if dbHealth["status"] != "healthy" {
		overallStatus = "degraded"
		if dbHealth["status"] == "unhealthy" {
			readiness = false
			overallStatus = "unhealthy"
		}
		if dbHealth["error"] != nil {
			errors = append(errors, dbHealth["error"].(map[string]interface{}))
		}
	}

	// Check Ollama AI service
	ollamaHealth := checkOllamaHealth()
	healthResponse.Dependencies["ollama"] = ollamaHealth
	if ollamaHealth["status"] != "healthy" {
		if overallStatus != "unhealthy" {
			overallStatus = "degraded"
		}
		if ollamaHealth["status"] == "unhealthy" {
			readiness = false
			overallStatus = "unhealthy"
		}
		if ollamaHealth["error"] != nil {
			errors = append(errors, ollamaHealth["error"].(map[string]interface{}))
		}
	}

	// Check Unstructured-io service
	unstructuredHealth := checkUnstructuredIOHealth()
	healthResponse.Dependencies["unstructured_io"] = unstructuredHealth
	if unstructuredHealth["status"] != "healthy" {
		if overallStatus != "unhealthy" {
			overallStatus = "degraded"
		}
		if unstructuredHealth["status"] == "unhealthy" {
			readiness = false
			overallStatus = "unhealthy"
		}
		if unstructuredHealth["error"] != nil {
			errors = append(errors, unstructuredHealth["error"].(map[string]interface{}))
		}
	}

	// Check N8N workflows
	n8nHealth := checkN8NHealth()
	healthResponse.Dependencies["n8n"] = n8nHealth
	if n8nHealth["status"] != "healthy" {
		if overallStatus != "unhealthy" {
			overallStatus = "degraded"
		}
		if n8nHealth["error"] != nil {
			errors = append(errors, n8nHealth["error"].(map[string]interface{}))
		}
	}

	// Check optional Qdrant vector database
	qdrantHealth := checkQdrantHealth()
	healthResponse.Dependencies["qdrant"] = qdrantHealth
	if qdrantHealth["status"] == "unhealthy" {
		// Qdrant is optional, so only degrade if configured but failing
		if overallStatus != "unhealthy" {
			overallStatus = "degraded"
		}
		if qdrantHealth["error"] != nil {
			errors = append(errors, qdrantHealth["error"].(map[string]interface{}))
		}
	}

	// Update final status
	healthResponse.Status = overallStatus
	healthResponse.Readiness = readiness

	// Add errors if any
	if len(errors) > 0 {
		healthResponse.Errors = errors
	}

	// Add metrics
	healthResponse.Metrics = map[string]interface{}{
		"total_dependencies":   5,
		"healthy_dependencies": countHealthyDependencies(healthResponse.Dependencies),
		"response_time_ms":     time.Since(start).Milliseconds(),
	}

	// Add data processing statistics
	dataStats := getDataStatistics(database)
	healthResponse.DataStats = dataStats

	// Return appropriate HTTP status
	statusCode := http.StatusOK
	if overallStatus == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, healthResponse)
}

// Health check helper methods
func checkDatabaseHealth(database *sql.DB) map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}

	if database == nil {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code":      "DATABASE_NOT_INITIALIZED",
			"message":   "Database connection not initialized",
			"category":  "configuration",
			"retryable": false,
		}
		return health
	}

	// Test database connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := database.PingContext(ctx); err != nil {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code":      "DATABASE_CONNECTION_FAILED",
			"message":   "Failed to ping database: " + err.Error(),
			"category":  "resource",
			"retryable": true,
		}
		return health
	}
	health["checks"].(map[string]interface{})["ping"] = "ok"

	// Check data structurer tables exist
	var tableCount int
	tableQuery := `SELECT COUNT(*) FROM information_schema.tables 
		WHERE table_schema = 'public' AND table_name IN ('schemas', 'processed_data', 'schema_templates')`
	if err := database.QueryRowContext(ctx, tableQuery).Scan(&tableCount); err != nil {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code":      "DATABASE_SCHEMA_CHECK_FAILED",
			"message":   "Failed to verify data structurer tables: " + err.Error(),
			"category":  "resource",
			"retryable": true,
		}
	} else {
		health["checks"].(map[string]interface{})["structurer_tables"] = tableCount
		if tableCount < 3 {
			health["status"] = "degraded"
			health["error"] = map[string]interface{}{
				"code":      "DATABASE_SCHEMA_INCOMPLETE",
				"message":   fmt.Sprintf("Missing data structurer tables. Found %d of 3 required tables", tableCount),
				"category":  "configuration",
				"retryable": false,
			}
		}
	}

	// Check connection pool
	stats := database.Stats()
	health["checks"].(map[string]interface{})["open_connections"] = stats.OpenConnections
	health["checks"].(map[string]interface{})["in_use"] = stats.InUse
	health["checks"].(map[string]interface{})["idle"] = stats.Idle

	return health
}

func checkOllamaHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}

	ollamaURL := os.Getenv("OLLAMA_BASE_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	// Test Ollama connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", ollamaURL+"/api/tags", nil)
	if err != nil {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code":      "OLLAMA_REQUEST_FAILED",
			"message":   "Failed to create request to Ollama: " + err.Error(),
			"category":  "internal",
			"retryable": false,
		}
		return health
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code":      "OLLAMA_CONNECTION_FAILED",
			"message":   "Failed to connect to Ollama: " + err.Error(),
			"category":  "network",
			"retryable": true,
		}
		return health
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		health["checks"].(map[string]interface{})["connectivity"] = "ok"
		
		// Parse response to check available models
		var response struct {
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&response); err == nil {
			health["checks"].(map[string]interface{})["available_models"] = len(response.Models)
			
			// Check for required models
			requiredModels := []string{"llama3.2", "mistral", "nomic-embed-text"}
			availableModels := make(map[string]bool)
			for _, model := range response.Models {
				availableModels[model.Name] = true
			}
			
			missingModels := []string{}
			for _, required := range requiredModels {
				if !availableModels[required] {
					missingModels = append(missingModels, required)
				}
			}
			
			if len(missingModels) > 0 {
				health["status"] = "degraded"
				health["error"] = map[string]interface{}{
					"code":      "OLLAMA_MODELS_MISSING",
					"message":   fmt.Sprintf("Missing required models: %v", missingModels),
					"category":  "configuration",
					"retryable": false,
				}
			} else {
				health["checks"].(map[string]interface{})["required_models"] = "available"
			}
		}
	} else {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code":      "OLLAMA_UNHEALTHY",
			"message":   fmt.Sprintf("Ollama returned status %d", resp.StatusCode),
			"category":  "resource",
			"retryable": true,
		}
	}

	return health
}

func checkUnstructuredIOHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}

	unstructuredURL := os.Getenv("UNSTRUCTURED_IO_URL")
	if unstructuredURL == "" {
		unstructuredURL = "http://localhost:8000"
	}

	// Test Unstructured-io connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", unstructuredURL+"/general/v0/general", nil)
	if err != nil {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code":      "UNSTRUCTURED_REQUEST_FAILED",
			"message":   "Failed to create request to Unstructured-io: " + err.Error(),
			"category":  "internal",
			"retryable": false,
		}
		return health
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code":      "UNSTRUCTURED_CONNECTION_FAILED",
			"message":   "Failed to connect to Unstructured-io: " + err.Error(),
			"category":  "network",
			"retryable": true,
		}
		return health
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnprocessableEntity {
		// 422 is expected when calling without a file
		health["checks"].(map[string]interface{})["connectivity"] = "ok"
		health["checks"].(map[string]interface{})["document_processing"] = "available"
	} else {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code":      "UNSTRUCTURED_UNHEALTHY",
			"message":   fmt.Sprintf("Unstructured-io returned status %d", resp.StatusCode),
			"category":  "resource",
			"retryable": true,
		}
	}

	return health
}

func checkN8NHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}

	n8nURL := os.Getenv("N8N_BASE_URL")
	if n8nURL == "" {
		n8nURL = "http://localhost:5678"
	}

	// Test N8N connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", n8nURL+"/healthz", nil)
	if err != nil {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code":      "N8N_REQUEST_FAILED",
			"message":   "Failed to create request to N8N: " + err.Error(),
			"category":  "internal",
			"retryable": false,
		}
		return health
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code":      "N8N_CONNECTION_FAILED",
			"message":   "Failed to connect to N8N: " + err.Error(),
			"category":  "network",
			"retryable": true,
		}
		return health
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		health["checks"].(map[string]interface{})["connectivity"] = "ok"
		health["checks"].(map[string]interface{})["workflows"] = "available"
	} else {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code":      "N8N_UNHEALTHY",
			"message":   fmt.Sprintf("N8N returned status %d", resp.StatusCode),
			"category":  "resource",
			"retryable": true,
		}
	}

	return health
}

func checkQdrantHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status": "not_configured",
		"checks": map[string]interface{}{},
	}

	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		qdrantURL = "http://localhost:6333"
	}

	// Test Qdrant connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", qdrantURL+"/health", nil)
	if err != nil {
		health["status"] = "not_configured"
		health["checks"].(map[string]interface{})["vector_search"] = "disabled"
		return health
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		health["status"] = "not_configured"
		health["checks"].(map[string]interface{})["vector_search"] = "disabled"
		return health
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		health["status"] = "healthy"
		health["checks"].(map[string]interface{})["connectivity"] = "ok"
		health["checks"].(map[string]interface{})["vector_search"] = "available"
	} else {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code":      "QDRANT_UNHEALTHY",
			"message":   fmt.Sprintf("Qdrant returned status %d", resp.StatusCode),
			"category":  "resource",
			"retryable": true,
		}
	}

	return health
}

func countHealthyDependencies(deps map[string]interface{}) int {
	count := 0
	for _, dep := range deps {
		if depMap, ok := dep.(map[string]interface{}); ok {
			if status, exists := depMap["status"]; exists && (status == "healthy" || status == "not_configured") {
				count++
			}
		}
	}
	return count
}

func getDataStatistics(database *sql.DB) map[string]interface{} {
	stats := map[string]interface{}{
		"total_schemas":           0,
		"active_schemas":          0,
		"total_processed_items":   0,
		"successful_processings":  0,
		"failed_processings":      0,
		"avg_confidence_score":    0.0,
		"avg_processing_time_ms":  0,
		"schema_templates":        0,
	}

	if database == nil {
		return stats
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Get schema counts
	var totalSchemas, activeSchemas, schemaTemplates int
	database.QueryRowContext(ctx, "SELECT COUNT(*) FROM schemas").Scan(&totalSchemas)
	database.QueryRowContext(ctx, "SELECT COUNT(*) FROM schemas WHERE is_active = true").Scan(&activeSchemas)
	database.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_templates").Scan(&schemaTemplates)
	stats["total_schemas"] = totalSchemas
	stats["active_schemas"] = activeSchemas
	stats["schema_templates"] = schemaTemplates

	// Get processing statistics
	var totalProcessed, successfulProcessed, failedProcessed int
	database.QueryRowContext(ctx, "SELECT COUNT(*) FROM processed_data").Scan(&totalProcessed)
	database.QueryRowContext(ctx, "SELECT COUNT(*) FROM processed_data WHERE processing_status = 'completed'").Scan(&successfulProcessed)
	database.QueryRowContext(ctx, "SELECT COUNT(*) FROM processed_data WHERE processing_status = 'failed'").Scan(&failedProcessed)
	stats["total_processed_items"] = totalProcessed
	stats["successful_processings"] = successfulProcessed
	stats["failed_processings"] = failedProcessed

	// Get average confidence score
	var avgConfidence sql.NullFloat64
	if err := database.QueryRowContext(ctx, "SELECT AVG(confidence_score) FROM processed_data WHERE confidence_score IS NOT NULL").Scan(&avgConfidence); err == nil && avgConfidence.Valid {
		stats["avg_confidence_score"] = math.Round(avgConfidence.Float64*100) / 100
	}

	// Get average processing time
	var avgProcessingTime sql.NullFloat64
	if err := database.QueryRowContext(ctx, "SELECT AVG(processing_time_ms) FROM processed_data WHERE processing_time_ms IS NOT NULL").Scan(&avgProcessingTime); err == nil && avgProcessingTime.Valid {
		stats["avg_processing_time_ms"] = int(avgProcessingTime.Float64)
	}

	return stats
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start data-structurer

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

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
		handleHealthCheck(c, db)
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