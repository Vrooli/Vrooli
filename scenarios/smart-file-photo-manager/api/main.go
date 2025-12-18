package main

import (
	"github.com/vrooli/api-core/preflight"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

// File represents a file in the system
type File struct {
	ID              string                 `json:"id" db:"id"`
	OriginalName    string                 `json:"original_name" db:"original_name"`
	CurrentName     *string                `json:"current_name" db:"current_name"`
	FileHash        string                 `json:"file_hash" db:"file_hash"`
	SizeBytes       int64                  `json:"size_bytes" db:"size_bytes"`
	MimeType        *string                `json:"mime_type" db:"mime_type"`
	FileType        *string                `json:"file_type" db:"file_type"`
	Extension       *string                `json:"extension" db:"extension"`
	StoragePath     string                 `json:"storage_path" db:"storage_path"`
	ThumbnailPath   *string                `json:"thumbnail_path" db:"thumbnail_path"`
	Status          string                 `json:"status" db:"status"`
	ProcessingStage string                 `json:"processing_stage" db:"processing_stage"`
	Description     *string                `json:"description" db:"description"`
	OCRText         *string                `json:"ocr_text" db:"ocr_text"`
	DetectedObjects json.RawMessage        `json:"detected_objects" db:"detected_objects"`
	FolderPath      string                 `json:"folder_path" db:"folder_path"`
	Tags            []string               `json:"tags" db:"tags"`
	Categories      []string               `json:"categories" db:"categories"`
	CustomMetadata  map[string]interface{} `json:"custom_metadata" db:"custom_metadata"`
	FileCreatedAt   *time.Time             `json:"file_created_at" db:"file_created_at"`
	FileModifiedAt  *time.Time             `json:"file_modified_at" db:"file_modified_at"`
	UploadedAt      time.Time              `json:"uploaded_at" db:"uploaded_at"`
	ProcessedAt     *time.Time             `json:"processed_at" db:"processed_at"`
	LastAccessedAt  *time.Time             `json:"last_accessed_at" db:"last_accessed_at"`
	UploadedBy      *string                `json:"uploaded_by" db:"uploaded_by"`
	OwnerID         *string                `json:"owner_id" db:"owner_id"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`
}

// Suggestion represents an AI-generated suggestion
type Suggestion struct {
	ID               string     `json:"id" db:"id"`
	FileID           string     `json:"file_id" db:"file_id"`
	Type             string     `json:"type" db:"type"`
	Status           string     `json:"status" db:"status"`
	SuggestedValue   string     `json:"suggested_value" db:"suggested_value"`
	CurrentValue     *string    `json:"current_value" db:"current_value"`
	Reason           *string    `json:"reason" db:"reason"`
	Confidence       *float64   `json:"confidence" db:"confidence"`
	SimilarFileIDs   []string   `json:"similar_file_ids" db:"similar_file_ids"`
	SimilarityScores []float64  `json:"similarity_scores" db:"similarity_scores"`
	ReviewedBy       *string    `json:"reviewed_by" db:"reviewed_by"`
	ReviewedAt       *time.Time `json:"reviewed_at" db:"reviewed_at"`
	AppliedAt        *time.Time `json:"applied_at" db:"applied_at"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
}

// SearchRequest represents a search query
type SearchRequest struct {
	Query      string            `json:"query"`
	Type       string            `json:"type"` // text, semantic, visual
	Filters    map[string]string `json:"filters"`
	Limit      int               `json:"limit"`
	Offset     int               `json:"offset"`
	MinScore   float64           `json:"min_score"`
	FileTypes  []string          `json:"file_types"`
	FolderPath string            `json:"folder_path"`
}

// SearchResult represents search results
type SearchResult struct {
	Files        []File  `json:"files"`
	TotalResults int     `json:"total_results"`
	SearchTime   float64 `json:"search_time"`
	Suggestions  []string `json:"suggestions"`
}

// UploadRequest represents a file upload request
type UploadRequest struct {
	Filename    string            `json:"filename"`
	MimeType    string            `json:"mime_type"`
	SizeBytes   int64             `json:"size_bytes"`
	FileHash    string            `json:"file_hash"`
	StoragePath string            `json:"storage_path"`
	FolderPath  string            `json:"folder_path"`
	Metadata    map[string]interface{} `json:"metadata"`
}


// App represents the main application
type App struct {
	DB              *sql.DB
	RedisClient     *redis.Client
	QdrantURL       string
	MinioURL        string
	OllamaURL       string
	ProcessingQueue chan ProcessingJob
	WorkerPool      *WorkerPool
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "smart-file-photo-manager",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

	// Initialize database
	db, err := initDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Initialize Redis
	redisClient := initRedis()
	defer redisClient.Close()

	// Create processing queue and worker pool
	processingQueue := make(chan ProcessingJob, 1000)
	ctx, cancel := context.WithCancel(context.Background())
	workerPool := &WorkerPool{
		workers: 10,
		jobs:    processingQueue,
		ctx:     ctx,
		cancel:  cancel,
	}

	// External service URLs with defaults for local development
	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		qdrantPort := os.Getenv("QDRANT_PORT")
		if qdrantPort == "" {
			qdrantPort = "6333"
		}
		qdrantURL = fmt.Sprintf("http://localhost:%s", qdrantPort)
		log.Printf("📡 QDRANT_URL not set, using default: %s", qdrantURL)
	}
	
	minioURL := os.Getenv("MINIO_URL")
	if minioURL == "" {
		minioPort := os.Getenv("MINIO_PORT")
		if minioPort == "" {
			minioPort = "9000"
		}
		minioURL = fmt.Sprintf("localhost:%s", minioPort)
		log.Printf("📡 MINIO_URL not set, using default: %s", minioURL)
	}
	
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaPort := os.Getenv("OLLAMA_PORT")
		if ollamaPort == "" {
			ollamaPort = "11434"
		}
		ollamaURL = fmt.Sprintf("http://localhost:%s", ollamaPort)
		log.Printf("📡 OLLAMA_URL not set, using default: %s", ollamaURL)
	}

	// Create app instance
	app := &App{
		DB:              db,
		RedisClient:     redisClient,
		QdrantURL:       qdrantURL,
		MinioURL:        minioURL,
		OllamaURL:       ollamaURL,
		ProcessingQueue: processingQueue,
		WorkerPool:      workerPool,
	}

	// Start worker pool
	app.startWorkers()

	// Setup router
	router := setupRouter(app)

	log.Printf("Smart File Photo Manager API starting on port %s", port)
	log.Printf("Qdrant URL: %s", app.QdrantURL)
	log.Printf("MinIO URL: %s", app.MinioURL)
	log.Printf("Ollama URL: %s", app.OllamaURL)
	log.Printf("Processing workers: %d", app.WorkerPool.workers)
	
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func initDB() (*sql.DB, error) {
	// Database configuration - support both POSTGRES_URL and individual components
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		// Try to build from individual components
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")
		
		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			return nil, fmt.Errorf("❌ Missing database configuration. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}
		
		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to open database connection: %v", err)
	}
	
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
		
		// Add random jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(rand.Float64() * jitterRange)
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
		return nil, fmt.Errorf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}
	
	log.Println("🎉 Database connection pool established successfully!")
	return db, nil
}

func initRedis() *redis.Client {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisPort := os.Getenv("REDIS_PORT")
		if redisPort == "" {
			redisPort = "6379"
		}
		redisURL = fmt.Sprintf("redis://localhost:%s/0", redisPort)
		log.Printf("📡 REDIS_URL not set, using default: %s", redisURL)
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Printf("Failed to parse Redis URL, using defaults: %v", err)
		opt = &redis.Options{
			Addr: "localhost:6379",
		}
	}

	client := redis.NewClient(opt)
	
	// Test connection
	ctx := context.Background()
	_, err = client.Ping(ctx).Result()
	if err != nil {
		log.Printf("Failed to connect to Redis: %v", err)
	}

	return client
}

func setupRouter(app *App) *gin.Engine {
	router := gin.Default()

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
			"service":   "smart-file-photo-manager-api",
		})
	})

	// API routes
	api := router.Group("/api")
	{
		// Files endpoints
		api.GET("/files", app.getFiles)
		api.GET("/files/:id", app.getFile)
		api.POST("/files", app.uploadFile)
		api.PUT("/files/:id", app.updateFile)
		api.DELETE("/files/:id", app.deleteFile)
		api.GET("/files/:id/download", app.downloadFile)
		api.GET("/files/:id/preview", app.previewFile)

		// Search endpoints
		api.POST("/search", app.searchFiles)
		api.GET("/search", app.searchFilesGET)

		// Suggestions endpoints
		api.GET("/suggestions", app.getSuggestions)
		api.PUT("/suggestions/:id", app.updateSuggestion)
		api.POST("/suggestions/:id/apply", app.applySuggestion)

		// Organization endpoints
		api.POST("/organize", app.organizeFiles)
		api.POST("/batch-organize", app.batchOrganize)
		api.POST("/find-duplicates", app.findDuplicates)

		// Folders endpoints
		api.GET("/folders", app.getFolders)
		api.POST("/folders", app.createFolder)
		api.PUT("/folders/:path", app.updateFolder)
		api.DELETE("/folders/:path", app.deleteFolder)

		// Processing endpoints
		api.POST("/process/:id", app.processFile)
		api.GET("/processing-status/:id", app.getProcessingStatus)

		// Statistics endpoints
		api.GET("/stats", app.getStats)
		api.GET("/stats/files", app.getFileStats)
		api.GET("/stats/processing", app.getProcessingStats)
	}

	return router
}

// File handlers
func (app *App) getFiles(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	folderPath := c.DefaultQuery("folder", "/")
	fileType := c.Query("type")
	status := c.Query("status")

	query := `
		SELECT id, original_name, current_name, file_hash, size_bytes, mime_type, 
		       file_type, extension, storage_path, thumbnail_path, status, 
		       processing_stage, description, ocr_text, detected_objects, 
		       folder_path, tags, categories, custom_metadata, file_created_at, 
		       file_modified_at, uploaded_at, processed_at, last_accessed_at, 
		       uploaded_by, owner_id, created_at, updated_at
		FROM files 
		WHERE folder_path = $1
	`
	args := []interface{}{folderPath}
	argCount := 1

	if fileType != "" {
		argCount++
		query += fmt.Sprintf(" AND file_type = $%d", argCount)
		args = append(args, fileType)
	}

	if status != "" {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
	}

	query += " ORDER BY uploaded_at DESC LIMIT $" + strconv.Itoa(argCount+1) + " OFFSET $" + strconv.Itoa(argCount+2)
	args = append(args, limit, offset)

	rows, err := app.DB.Query(query, args...)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to query files: " + err.Error()})
		return
	}
	defer rows.Close()

	var files []File
	for rows.Next() {
		var file File
		err := rows.Scan(
			&file.ID, &file.OriginalName, &file.CurrentName, &file.FileHash,
			&file.SizeBytes, &file.MimeType, &file.FileType, &file.Extension,
			&file.StoragePath, &file.ThumbnailPath, &file.Status, &file.ProcessingStage,
			&file.Description, &file.OCRText, &file.DetectedObjects, &file.FolderPath,
			&file.Tags, &file.Categories, &file.CustomMetadata, &file.FileCreatedAt,
			&file.FileModifiedAt, &file.UploadedAt, &file.ProcessedAt, &file.LastAccessedAt,
			&file.UploadedBy, &file.OwnerID, &file.CreatedAt, &file.UpdatedAt,
		)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to scan file: " + err.Error()})
			return
		}
		files = append(files, file)
	}

	c.JSON(200, gin.H{
		"files":  files,
		"total":  len(files),
		"limit":  limit,
		"offset": offset,
	})
}

func (app *App) getFile(c *gin.Context) {
	id := c.Param("id")

	var file File
	query := `
		SELECT id, original_name, current_name, file_hash, size_bytes, mime_type, 
		       file_type, extension, storage_path, thumbnail_path, status, 
		       processing_stage, description, ocr_text, detected_objects, 
		       folder_path, tags, categories, custom_metadata, file_created_at, 
		       file_modified_at, uploaded_at, processed_at, last_accessed_at, 
		       uploaded_by, owner_id, created_at, updated_at
		FROM files WHERE id = $1
	`

	err := app.DB.QueryRow(query, id).Scan(
		&file.ID, &file.OriginalName, &file.CurrentName, &file.FileHash,
		&file.SizeBytes, &file.MimeType, &file.FileType, &file.Extension,
		&file.StoragePath, &file.ThumbnailPath, &file.Status, &file.ProcessingStage,
		&file.Description, &file.OCRText, &file.DetectedObjects, &file.FolderPath,
		&file.Tags, &file.Categories, &file.CustomMetadata, &file.FileCreatedAt,
		&file.FileModifiedAt, &file.UploadedAt, &file.ProcessedAt, &file.LastAccessedAt,
		&file.UploadedBy, &file.OwnerID, &file.CreatedAt, &file.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(404, gin.H{"error": "File not found"})
		} else {
			c.JSON(500, gin.H{"error": "Failed to get file: " + err.Error()})
		}
		return
	}

	c.JSON(200, file)
}

func (app *App) uploadFile(c *gin.Context) {
	var req UploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Insert file record (with dynamic schema detection for compatibility)
	// Check which columns exist
	var hasUserID, hasFilename, hasOriginalName bool
	app.DB.QueryRow(`
		SELECT
			EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'files' AND column_name = 'user_id'),
			EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'files' AND column_name = 'filename'),
			EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'files' AND column_name = 'original_name')
	`).Scan(&hasUserID, &hasFilename, &hasOriginalName)

	// Build query based on actual schema - handle both columns if they exist
	var query string
	var args []interface{}

	if hasFilename && hasOriginalName {
		// Both columns exist - populate both with the same value
		if hasUserID {
			query = `
				INSERT INTO files (filename, original_name, file_hash, size_bytes, mime_type,
				                  storage_path, folder_path, status, processing_stage,
				                  custom_metadata, user_id, uploaded_at, created_at, updated_at)
				VALUES ($1, $1, $2, $3, $4, $5, $6, 'pending', 'uploaded', $7, 'default', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
				RETURNING id`
		} else {
			query = `
				INSERT INTO files (filename, original_name, file_hash, size_bytes, mime_type,
				                  storage_path, folder_path, status, processing_stage,
				                  custom_metadata, uploaded_at, created_at, updated_at)
				VALUES ($1, $1, $2, $3, $4, $5, $6, 'pending', 'uploaded', $7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
				RETURNING id`
		}
	} else if hasFilename {
		// Only filename column exists
		if hasUserID {
			query = `
				INSERT INTO files (filename, file_hash, size_bytes, mime_type,
				                  storage_path, folder_path, status, processing_stage,
				                  custom_metadata, user_id, uploaded_at, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, 'pending', 'uploaded', $7, 'default', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
				RETURNING id`
		} else {
			query = `
				INSERT INTO files (filename, file_hash, size_bytes, mime_type,
				                  storage_path, folder_path, status, processing_stage,
				                  custom_metadata, uploaded_at, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, 'pending', 'uploaded', $7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
				RETURNING id`
		}
	} else {
		// Only original_name column exists (or neither, default to original_name)
		if hasUserID {
			query = `
				INSERT INTO files (original_name, file_hash, size_bytes, mime_type,
				                  storage_path, folder_path, status, processing_stage,
				                  custom_metadata, user_id, uploaded_at, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, 'pending', 'uploaded', $7, 'default', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
				RETURNING id`
		} else {
			query = `
				INSERT INTO files (original_name, file_hash, size_bytes, mime_type,
				                  storage_path, folder_path, status, processing_stage,
				                  custom_metadata, uploaded_at, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, 'pending', 'uploaded', $7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
				RETURNING id`
		}
	}

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(req.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	args = []interface{}{req.Filename, req.FileHash, req.SizeBytes,
		req.MimeType, req.StoragePath, req.FolderPath, metadataJSON}

	var fileID string
	err = app.DB.QueryRow(query, args...).Scan(&fileID)

	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to create file record: " + err.Error()})
		return
	}

	// Queue file for processing
	app.queueFileProcessing(fileID, req)

	c.JSON(201, gin.H{
		"file_id": fileID,
		"status":  "uploaded",
		"message": "File uploaded successfully and queued for processing",
	})
}


func (app *App) updateFile(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

func (app *App) deleteFile(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

func (app *App) downloadFile(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

func (app *App) previewFile(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

// Search handlers
func (app *App) searchFiles(c *gin.Context) {
	var req SearchRequest
	
	// Check if request came from GET endpoint
	if searchReq, exists := c.Get("search_request"); exists {
		req = searchReq.(SearchRequest)
	} else {
		// Handle POST request
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request: " + err.Error()})
			return
		}
	}

	startTime := time.Now()

	// For now, implement basic text search
	// In a full implementation, this would integrate with Qdrant for semantic search
	var files []File
	query := `
		SELECT id, original_name, current_name, file_hash, size_bytes, mime_type, 
		       file_type, extension, storage_path, thumbnail_path, status, 
		       processing_stage, description, ocr_text, detected_objects, 
		       folder_path, tags, categories, custom_metadata, file_created_at, 
		       file_modified_at, uploaded_at, processed_at, last_accessed_at, 
		       uploaded_by, owner_id, created_at, updated_at
		FROM files 
		WHERE (original_name ILIKE $1 OR description ILIKE $1 OR ocr_text ILIKE $1)
		ORDER BY uploaded_at DESC 
		LIMIT $2 OFFSET $3
	`

	searchTerm := "%" + req.Query + "%"
	limit := req.Limit
	if limit == 0 {
		limit = 20
	}

	rows, err := app.DB.Query(query, searchTerm, limit, req.Offset)
	if err != nil {
		c.JSON(500, gin.H{"error": "Search failed: " + err.Error()})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var file File
		err := rows.Scan(
			&file.ID, &file.OriginalName, &file.CurrentName, &file.FileHash,
			&file.SizeBytes, &file.MimeType, &file.FileType, &file.Extension,
			&file.StoragePath, &file.ThumbnailPath, &file.Status, &file.ProcessingStage,
			&file.Description, &file.OCRText, &file.DetectedObjects, &file.FolderPath,
			&file.Tags, &file.Categories, &file.CustomMetadata, &file.FileCreatedAt,
			&file.FileModifiedAt, &file.UploadedAt, &file.ProcessedAt, &file.LastAccessedAt,
			&file.UploadedBy, &file.OwnerID, &file.CreatedAt, &file.UpdatedAt,
		)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to scan file: " + err.Error()})
			return
		}
		files = append(files, file)
	}

	searchTime := time.Since(startTime).Seconds()

	result := SearchResult{
		Files:        files,
		TotalResults: len(files),
		SearchTime:   searchTime,
		Suggestions:  []string{}, // TODO: Implement search suggestions
	}

	c.JSON(200, result)
}

func (app *App) searchFilesGET(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(400, gin.H{"error": "Query parameter 'q' is required"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	req := SearchRequest{
		Query:  query,
		Type:   c.DefaultQuery("type", "text"),
		Limit:  limit,
		Offset: offset,
	}

	// Reuse the POST search logic
	c.Set("search_request", req)
	app.searchFiles(c)
}

// Placeholder handlers for other endpoints
func (app *App) getSuggestions(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

func (app *App) updateSuggestion(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

func (app *App) applySuggestion(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

func (app *App) organizeFiles(c *gin.Context) {
	var req struct {
		FileIDs []string `json:"file_ids"`
		Strategy string  `json:"strategy"` // by_type, by_date, by_content, smart
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}
	
	if len(req.FileIDs) == 0 {
		// Organize all unorganized files
		query := `SELECT id FROM files WHERE folder_path = '/' OR folder_path IS NULL`
		rows, _ := app.DB.Query(query)
		defer rows.Close()
		
		for rows.Next() {
			var id string
			rows.Scan(&id)
			req.FileIDs = append(req.FileIDs, id)
		}
	}
	
	// Queue organization jobs
	for _, fileID := range req.FileIDs {
		job := ProcessingJob{
			FileID:  fileID,
			JobType: "organize",
			Payload: req.Strategy,
		}
		app.queueJob(job)
	}
	
	c.JSON(200, gin.H{
		"message": fmt.Sprintf("Queued %d files for organization", len(req.FileIDs)),
		"file_ids": req.FileIDs,
	})
}

func (app *App) batchOrganize(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

func (app *App) findDuplicates(c *gin.Context) {
	// Find all duplicate files in the system
	query := `
		SELECT file_hash, array_agg(id) as file_ids, COUNT(*) as count
		FROM files
		GROUP BY file_hash
		HAVING COUNT(*) > 1
		ORDER BY count DESC
	`
	
	rows, err := app.DB.Query(query)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to find duplicates: " + err.Error()})
		return
	}
	defer rows.Close()
	
	type DuplicateGroup struct {
		FileHash string   `json:"file_hash"`
		FileIDs  []string `json:"file_ids"`
		Count    int      `json:"count"`
	}
	
	var duplicates []DuplicateGroup
	for rows.Next() {
		var group DuplicateGroup
		var fileIDsJSON []byte
		err := rows.Scan(&group.FileHash, &fileIDsJSON, &group.Count)
		if err != nil {
			continue
		}
		json.Unmarshal(fileIDsJSON, &group.FileIDs)
		duplicates = append(duplicates, group)
	}
	
	c.JSON(200, gin.H{
		"duplicate_groups": duplicates,
		"total_groups":     len(duplicates),
	})
}

func (app *App) getFolders(c *gin.Context) {
	// Get list of folders from database
	query := `
		SELECT id, path, parent_path, name, description, metadata, created_at, updated_at
		FROM folders
		ORDER BY path ASC
	`

	rows, err := app.DB.Query(query)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to query folders: " + err.Error()})
		return
	}
	defer rows.Close()

	type Folder struct {
		ID          string                 `json:"id"`
		Path        string                 `json:"path"`
		ParentPath  *string                `json:"parent_path"`
		Name        string                 `json:"name"`
		Description *string                `json:"description"`
		Metadata    map[string]interface{} `json:"metadata"`
		CreatedAt   time.Time              `json:"created_at"`
		UpdatedAt   time.Time              `json:"updated_at"`
	}

	var folders []Folder
	for rows.Next() {
		var folder Folder
		var metadataJSON []byte

		err := rows.Scan(
			&folder.ID, &folder.Path, &folder.ParentPath, &folder.Name,
			&folder.Description, &metadataJSON, &folder.CreatedAt, &folder.UpdatedAt,
		)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to scan folder: " + err.Error()})
			return
		}

		// Parse metadata JSON
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &folder.Metadata)
		}

		folders = append(folders, folder)
	}

	c.JSON(200, gin.H{
		"folders": folders,
		"total":   len(folders),
	})
}

func (app *App) createFolder(c *gin.Context) {
	var req struct {
		Path        string                 `json:"path" binding:"required"`
		ParentPath  string                 `json:"parent_path"`
		Name        string                 `json:"name" binding:"required"`
		Description string                 `json:"description"`
		Metadata    map[string]interface{} `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(req.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	// Insert folder
	query := `
		INSERT INTO folders (path, parent_path, name, description, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, created_at
	`

	var folderID string
	var createdAt time.Time
	err = app.DB.QueryRow(query, req.Path, req.ParentPath, req.Name, req.Description, metadataJSON).Scan(&folderID, &createdAt)

	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to create folder: " + err.Error()})
		return
	}

	c.JSON(201, gin.H{
		"folder_id":  folderID,
		"path":       req.Path,
		"name":       req.Name,
		"created_at": createdAt,
		"message":    "Folder created successfully",
	})
}

func (app *App) updateFolder(c *gin.Context) {
	folderPath := c.Param("path")

	var req struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Metadata    map[string]interface{} `json:"metadata"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(req.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	// Update folder
	query := `
		UPDATE folders
		SET name = COALESCE(NULLIF($1, ''), name),
		    description = COALESCE(NULLIF($2, ''), description),
		    metadata = COALESCE($3, metadata),
		    updated_at = CURRENT_TIMESTAMP
		WHERE path = $4
		RETURNING id, updated_at
	`

	var folderID string
	var updatedAt time.Time
	err = app.DB.QueryRow(query, req.Name, req.Description, metadataJSON, folderPath).Scan(&folderID, &updatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(404, gin.H{"error": "Folder not found"})
		} else {
			c.JSON(500, gin.H{"error": "Failed to update folder: " + err.Error()})
		}
		return
	}

	c.JSON(200, gin.H{
		"folder_id":  folderID,
		"path":       folderPath,
		"updated_at": updatedAt,
		"message":    "Folder updated successfully",
	})
}

func (app *App) deleteFolder(c *gin.Context) {
	folderPath := c.Param("path")

	// Check if folder has files
	var fileCount int
	err := app.DB.QueryRow(`SELECT COUNT(*) FROM files WHERE folder_path = $1`, folderPath).Scan(&fileCount)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to check folder contents: " + err.Error()})
		return
	}

	if fileCount > 0 {
		c.JSON(400, gin.H{
			"error":      "Folder is not empty",
			"file_count": fileCount,
			"message":    "Cannot delete folder with files. Move or delete files first.",
		})
		return
	}

	// Delete folder
	result, err := app.DB.Exec(`DELETE FROM folders WHERE path = $1`, folderPath)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to delete folder: " + err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(404, gin.H{"error": "Folder not found"})
		return
	}

	c.JSON(200, gin.H{
		"message": "Folder deleted successfully",
		"path":    folderPath,
	})
}

func (app *App) processFile(c *gin.Context) {
	fileID := c.Param("id")
	
	// Get file info
	var req UploadRequest
	query := `
		SELECT original_name, mime_type, size_bytes, file_hash, storage_path, folder_path
		FROM files WHERE id = $1
	`
	err := app.DB.QueryRow(query, fileID).Scan(
		&req.Filename, &req.MimeType, &req.SizeBytes, 
		&req.FileHash, &req.StoragePath, &req.FolderPath,
	)
	
	if err != nil {
		c.JSON(404, gin.H{"error": "File not found"})
		return
	}
	
	// Queue for reprocessing
	app.queueFileProcessing(fileID, req)
	
	c.JSON(200, gin.H{
		"message": "File queued for processing",
		"file_id": fileID,
	})
}

func (app *App) getProcessingStatus(c *gin.Context) {
	fileID := c.Param("id")
	
	var status, stage string
	var processedAt *time.Time
	query := `SELECT status, processing_stage, processed_at FROM files WHERE id = $1`
	err := app.DB.QueryRow(query, fileID).Scan(&status, &stage, &processedAt)
	
	if err != nil {
		c.JSON(404, gin.H{"error": "File not found"})
		return
	}
	
	c.JSON(200, gin.H{
		"file_id":          fileID,
		"status":           status,
		"processing_stage": stage,
		"processed_at":     processedAt,
	})
}

func (app *App) getStats(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

func (app *App) getFileStats(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

func (app *App) getProcessingStats(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

