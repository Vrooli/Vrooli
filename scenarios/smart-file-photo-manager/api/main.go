package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
	DB          *sql.DB
	RedisClient *redis.Client
	N8NBaseURL  string
	WindmillURL string
	QdrantURL   string
	MinioURL    string
	OllamaURL   string
}

func main() {
	// Load environment variables
	port := getEnv("API_PORT", getEnv("PORT", ""))

	// Initialize database
	db, err := initDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Initialize Redis
	redisClient := initRedis()
	defer redisClient.Close()

	// Create app instance
	app := &App{
		DB:          db,
		RedisClient: redisClient,
		N8NBaseURL:  os.Getenv("N8N_BASE_URL"),
		WindmillURL: os.Getenv("WINDMILL_BASE_URL"),
		QdrantURL:   os.Getenv("QDRANT_URL"),
		MinioURL:    os.Getenv("MINIO_URL"),
		OllamaURL:   os.Getenv("OLLAMA_URL"),
	}

	// Setup router
	router := setupRouter(app)

	log.Printf("Smart File Photo Manager API starting on port %s", port)
	log.Printf("N8N Base URL: %s", app.N8NBaseURL)
	log.Printf("Windmill URL: %s", app.WindmillURL)
	
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func initDB() (*sql.DB, error) {
	dbURL := os.Getenv("POSTGRES_URL")
	if dbURL == "" {
		log.Fatal("POSTGRES_URL environment variable is required")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func initRedis() *redis.Client {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
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

	// Insert file record
	query := `
		INSERT INTO files (original_name, file_hash, size_bytes, mime_type, 
		                  storage_path, folder_path, status, processing_stage,
		                  custom_metadata, uploaded_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending', 'uploaded', $7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`

	var fileID string
	err := app.DB.QueryRow(query, req.Filename, req.FileHash, req.SizeBytes,
		req.MimeType, req.StoragePath, req.FolderPath, req.Metadata).Scan(&fileID)

	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to create file record: " + err.Error()})
		return
	}

	// Trigger processing via n8n webhook
	go app.triggerProcessing(fileID, req)

	c.JSON(201, gin.H{
		"file_id": fileID,
		"status":  "uploaded",
		"message": "File uploaded successfully and queued for processing",
	})
}

func (app *App) triggerProcessing(fileID string, req UploadRequest) {
	if app.N8NBaseURL == "" {
		log.Printf("N8N Base URL not configured, skipping processing trigger for file %s", fileID)
		return
	}

	payload := map[string]interface{}{
		"file_id":      fileID,
		"filename":     req.Filename,
		"mime_type":    req.MimeType,
		"size_bytes":   req.SizeBytes,
		"storage_path": req.StoragePath,
		"folder_path":  req.FolderPath,
		"metadata":     req.Metadata,
	}

	jsonData, _ := json.Marshal(payload)
	webhookURL := fmt.Sprintf("%s/webhook/file-upload", app.N8NBaseURL)

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to trigger processing for file %s: %v", fileID, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("Processing trigger failed for file %s: HTTP %d", fileID, resp.StatusCode)
		return
	}

	log.Printf("Processing triggered for file %s", fileID)
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
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request: " + err.Error()})
		return
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
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

func (app *App) batchOrganize(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

func (app *App) findDuplicates(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

func (app *App) getFolders(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

func (app *App) createFolder(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

func (app *App) updateFolder(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

func (app *App) deleteFolder(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

func (app *App) processFile(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
}

func (app *App) getProcessingStatus(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Not implemented yet"})
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

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
