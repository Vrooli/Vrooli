package main

import (
	"github.com/vrooli/api-core/preflight"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
)

// Configuration
type Config struct {
	Port   int
	DBHost string
	DBPort int
	DBUser string
	DBPass string
	DBName string

	// External service endpoints
	SessionAuthURL     string
	NotificationHubURL string
}

// Models
type ScenarioConfig struct {
	ID                   uuid.UUID              `json:"id" db:"id"`
	ScenarioName         string                 `json:"scenario_name" db:"scenario_name"`
	AuthRequired         bool                   `json:"auth_required" db:"auth_required"`
	AllowAnonymous       bool                   `json:"allow_anonymous" db:"allow_anonymous"`
	AllowRichMedia       bool                   `json:"allow_rich_media" db:"allow_rich_media"`
	ModerationLevel      string                 `json:"moderation_level" db:"moderation_level"`
	ThemeConfig          map[string]interface{} `json:"theme_config" db:"theme_config"`
	NotificationSettings map[string]interface{} `json:"notification_settings" db:"notification_settings"`
	CreatedAt            time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at" db:"updated_at"`
}

type Comment struct {
	ID           uuid.UUID              `json:"id" db:"id"`
	ScenarioName string                 `json:"scenario_name" db:"scenario_name"`
	ParentID     *uuid.UUID             `json:"parent_id" db:"parent_id"`
	AuthorID     *uuid.UUID             `json:"author_id" db:"author_id"`
	AuthorName   *string                `json:"author_name" db:"author_name"`
	Content      string                 `json:"content" db:"content"`
	ContentType  string                 `json:"content_type" db:"content_type"`
	Metadata     map[string]interface{} `json:"metadata" db:"metadata"`
	Status       string                 `json:"status" db:"status"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
	Version      int                    `json:"version" db:"version"`
	ThreadPath   *string                `json:"thread_path" db:"thread_path"`
	Depth        int                    `json:"depth" db:"depth"`
	ReplyCount   int                    `json:"reply_count" db:"reply_count"`

	// Computed fields
	RenderedContent string    `json:"rendered_content"`
	Children        []Comment `json:"children,omitempty"`
}

type CommentRequest struct {
	Content     string                 `json:"content" binding:"required"`
	ParentID    *uuid.UUID             `json:"parent_id"`
	ContentType string                 `json:"content_type"`
	AuthorToken string                 `json:"author_token"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type CommentResponse struct {
	Comments   []Comment `json:"comments"`
	TotalCount int       `json:"total_count"`
	HasMore    bool      `json:"has_more"`
}

type HealthResponse struct {
	Status       string            `json:"status"`
	Timestamp    time.Time         `json:"timestamp"`
	Version      string            `json:"version"`
	Database     string            `json:"database"`
	Dependencies map[string]string `json:"dependencies"`
}

// Database
type Database struct {
	conn *sql.DB
}

// Services
type SessionAuthService struct {
	baseURL string
	client  *http.Client
}

type NotificationService struct {
	baseURL string
	client  *http.Client
}

type UserInfo struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Name     string    `json:"name"`
}

// Application
type App struct {
	config        *Config
	db            *Database
	sessionAuth   *SessionAuthService
	notifications *NotificationService
	sanitizer     *bluemonday.Policy
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "comment-system",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Command line flags
	var initDB = flag.Bool("init-db", false, "Initialize database schema")
	flag.Parse()

	// Load environment
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment")
	}

	// Configuration
	config := &Config{
		Port:               getEnvInt("API_PORT", 8080),
		DBHost:             getEnv("POSTGRES_HOST", "localhost"),
		DBPort:             getEnvInt("POSTGRES_PORT", 5432),
		DBUser:             getEnv("POSTGRES_USER", "postgres"),
		DBPass:             getEnv("POSTGRES_PASSWORD", "postgres"),
		DBName:             getEnv("POSTGRES_DB", "vrooli"),
		SessionAuthURL:     getEnv("SESSION_AUTH_URL", "http://localhost:8001"),
		NotificationHubURL: getEnv("NOTIFICATION_HUB_URL", "http://localhost:8002"),
	}

	// Initialize database connection
	db, err := NewDatabase(config)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Handle database initialization
	if *initDB {
		if err := db.InitSchema(); err != nil {
			log.Fatalf("Failed to initialize database schema: %v", err)
		}
		log.Println("Database schema initialized successfully")
		return
	}

	// Initialize services
	sessionAuth := &SessionAuthService{
		baseURL: config.SessionAuthURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}

	notifications := &NotificationService{
		baseURL: config.NotificationHubURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}

	// HTML sanitization policy for Markdown
	sanitizer := bluemonday.UGCPolicy()

	// Initialize app
	app := &App{
		config:        config,
		db:            db,
		sessionAuth:   sessionAuth,
		notifications: notifications,
		sanitizer:     sanitizer,
	}

	// Setup router
	router := app.setupRouter()

	// Start server
	log.Printf("Comment System API starting on port %d", config.Port)
	log.Printf("Health check: http://localhost:%d/health", config.Port)
	log.Printf("API docs: http://localhost:%d/docs", config.Port)

	if err := router.Run(fmt.Sprintf(":%d", config.Port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func (app *App) setupRouter() *gin.Engine {
	router := gin.Default()

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Requested-With")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health endpoints
	router.GET("/health", app.healthCheck)
	router.GET("/health/postgres", app.postgresHealthCheck)

	// API routes
	api := router.Group("/api/v1")
	{
		// Comment endpoints
		api.GET("/comments/:scenario", app.getComments)
		api.POST("/comments/:scenario", app.createComment)
		api.PUT("/comments/:id", app.updateComment)
		api.DELETE("/comments/:id", app.deleteComment)

		// Configuration endpoints
		api.GET("/config/:scenario", app.getConfig)
		api.POST("/config/:scenario", app.updateConfig)

		// Admin endpoints
		api.GET("/admin/stats", app.getStats)
		api.POST("/admin/moderate/:id", app.moderateComment)
	}

	// Documentation endpoint
	router.GET("/docs", app.serveDocs)

	return router
}

// Database operations
func NewDatabase(config *Config) (*Database, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPass, config.DBName)

	conn, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	// Configure connection pool
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(10 * time.Minute)

	return &Database{conn: conn}, nil
}

func (db *Database) Close() error {
	return db.conn.Close()
}

func (db *Database) InitSchema() error {
	// Read schema file
	schemaSQL, err := os.ReadFile("../initialization/storage/postgres/schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema file: %v", err)
	}

	// Execute schema
	if _, err := db.conn.Exec(string(schemaSQL)); err != nil {
		return fmt.Errorf("failed to execute schema: %v", err)
	}

	return nil
}

func (db *Database) GetComments(scenarioName string, parentID *uuid.UUID, limit, offset int, sort string) ([]Comment, int, error) {
	var baseQuery strings.Builder
	var countQuery strings.Builder
	var args []interface{}
	argIndex := 1

	// Base WHERE clause
	baseQuery.WriteString("FROM comments WHERE scenario_name = $")
	baseQuery.WriteString(strconv.Itoa(argIndex))
	baseQuery.WriteString(" AND status = 'active'")
	args = append(args, scenarioName)
	argIndex++

	// Parent filter
	if parentID != nil {
		baseQuery.WriteString(" AND parent_id = $")
		baseQuery.WriteString(strconv.Itoa(argIndex))
		args = append(args, *parentID)
		argIndex++
	} else {
		baseQuery.WriteString(" AND parent_id IS NULL")
	}

	// Count query
	countQuery.WriteString("SELECT COUNT(*) ")
	countQuery.WriteString(baseQuery.String())

	var totalCount int
	if err := db.conn.QueryRow(countQuery.String(), args...).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	// Main query with ordering and pagination
	var mainQuery strings.Builder
	mainQuery.WriteString("SELECT id, scenario_name, parent_id, author_id, author_name, content, content_type, ")
	mainQuery.WriteString("metadata, status, created_at, updated_at, version, thread_path, depth, reply_count ")
	mainQuery.WriteString(baseQuery.String())

	// Ordering
	switch sort {
	case "oldest":
		mainQuery.WriteString(" ORDER BY created_at ASC")
	case "threaded":
		mainQuery.WriteString(" ORDER BY thread_path ASC")
	default: // newest
		mainQuery.WriteString(" ORDER BY created_at DESC")
	}

	// Pagination
	mainQuery.WriteString(" LIMIT $")
	mainQuery.WriteString(strconv.Itoa(argIndex))
	mainQuery.WriteString(" OFFSET $")
	mainQuery.WriteString(strconv.Itoa(argIndex + 1))
	args = append(args, limit, offset)

	rows, err := db.conn.Query(mainQuery.String(), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var c Comment
		var metadataJSON []byte

		err := rows.Scan(
			&c.ID, &c.ScenarioName, &c.ParentID, &c.AuthorID, &c.AuthorName,
			&c.Content, &c.ContentType, &metadataJSON, &c.Status,
			&c.CreatedAt, &c.UpdatedAt, &c.Version, &c.ThreadPath, &c.Depth, &c.ReplyCount,
		)
		if err != nil {
			return nil, 0, err
		}

		// Parse metadata JSON
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &c.Metadata); err != nil {
				c.Metadata = make(map[string]interface{})
			}
		} else {
			c.Metadata = make(map[string]interface{})
		}

		// Render content if markdown
		if c.ContentType == "markdown" {
			rendered := blackfriday.Run([]byte(c.Content))
			c.RenderedContent = string(rendered)
		} else {
			c.RenderedContent = c.Content
		}

		comments = append(comments, c)
	}

	return comments, totalCount, nil
}

func (db *Database) CreateComment(comment *Comment) error {
	metadataJSON, _ := json.Marshal(comment.Metadata)

	query := `
		INSERT INTO comments (scenario_name, parent_id, author_id, author_name, content, content_type, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at, version, thread_path, depth, reply_count
	`

	return db.conn.QueryRow(
		query,
		comment.ScenarioName, comment.ParentID, comment.AuthorID, comment.AuthorName,
		comment.Content, comment.ContentType, metadataJSON,
	).Scan(&comment.ID, &comment.CreatedAt, &comment.UpdatedAt, &comment.Version, &comment.ThreadPath, &comment.Depth, &comment.ReplyCount)
}

func (db *Database) GetScenarioConfig(scenarioName string) (*ScenarioConfig, error) {
	config := &ScenarioConfig{}
	var themeJSON, notifJSON []byte

	query := `
		SELECT id, scenario_name, auth_required, allow_anonymous, allow_rich_media, 
		       moderation_level, theme_config, notification_settings, created_at, updated_at
		FROM scenario_configs WHERE scenario_name = $1
	`

	err := db.conn.QueryRow(query, scenarioName).Scan(
		&config.ID, &config.ScenarioName, &config.AuthRequired, &config.AllowAnonymous,
		&config.AllowRichMedia, &config.ModerationLevel, &themeJSON, &notifJSON,
		&config.CreatedAt, &config.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Create default configuration
			return db.CreateDefaultConfig(scenarioName)
		}
		return nil, err
	}

	// Parse JSON fields
	json.Unmarshal(themeJSON, &config.ThemeConfig)
	json.Unmarshal(notifJSON, &config.NotificationSettings)

	return config, nil
}

func (db *Database) CreateDefaultConfig(scenarioName string) (*ScenarioConfig, error) {
	config := &ScenarioConfig{
		ID:                   uuid.New(),
		ScenarioName:         scenarioName,
		AuthRequired:         true,
		AllowAnonymous:       false,
		AllowRichMedia:       false,
		ModerationLevel:      "manual",
		ThemeConfig:          map[string]interface{}{"theme": "default"},
		NotificationSettings: map[string]interface{}{"mentions": true, "replies": true, "new_comments": false},
	}

	themeJSON, _ := json.Marshal(config.ThemeConfig)
	notifJSON, _ := json.Marshal(config.NotificationSettings)

	query := `
		INSERT INTO scenario_configs (id, scenario_name, auth_required, allow_anonymous, allow_rich_media, moderation_level, theme_config, notification_settings)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at
	`

	err := db.conn.QueryRow(
		query,
		config.ID, config.ScenarioName, config.AuthRequired, config.AllowAnonymous,
		config.AllowRichMedia, config.ModerationLevel, themeJSON, notifJSON,
	).Scan(&config.CreatedAt, &config.UpdatedAt)

	return config, err
}

// HTTP Handlers
func (app *App) healthCheck(c *gin.Context) {
	// Test database connection
	dbStatus := "healthy"
	if err := app.db.conn.Ping(); err != nil {
		dbStatus = "unhealthy: " + err.Error()
	}

	// Test external dependencies
	dependencies := map[string]string{
		"session_authenticator": app.testSessionAuth(),
		"notification_hub":      app.testNotificationHub(),
	}

	response := HealthResponse{
		Status:       "healthy",
		Timestamp:    time.Now(),
		Version:      "1.0.0",
		Database:     dbStatus,
		Dependencies: dependencies,
	}

	if dbStatus != "healthy" {
		response.Status = "unhealthy"
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}

	c.JSON(http.StatusOK, response)
}

func (app *App) postgresHealthCheck(c *gin.Context) {
	if err := app.db.conn.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
	})
}

func (app *App) getComments(c *gin.Context) {
	scenarioName := c.Param("scenario")

	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")
	sort := c.DefaultQuery("sort", "newest")
	parentIDStr := c.Query("parent_id")

	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	var parentID *uuid.UUID
	if parentIDStr != "" {
		if parsed, err := uuid.Parse(parentIDStr); err == nil {
			parentID = &parsed
		}
	}

	// Validate limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	comments, totalCount, err := app.db.GetComments(scenarioName, parentID, limit, offset, sort)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := CommentResponse{
		Comments:   comments,
		TotalCount: totalCount,
		HasMore:    offset+len(comments) < totalCount,
	}

	c.JSON(http.StatusOK, response)
}

func (app *App) createComment(c *gin.Context) {
	scenarioName := c.Param("scenario")

	var req CommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get scenario configuration
	config, err := app.db.GetScenarioConfig(scenarioName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get scenario configuration"})
		return
	}

	// Validate authentication requirements
	var userInfo *UserInfo
	if config.AuthRequired || req.AuthorToken != "" {
		userInfo, err = app.sessionAuth.ValidateToken(req.AuthorToken)
		if err != nil {
			if config.AuthRequired {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
				return
			}
			// Allow anonymous if not required
			userInfo = nil
		}
	}

	// Create comment
	comment := &Comment{
		ScenarioName: scenarioName,
		ParentID:     req.ParentID,
		Content:      strings.TrimSpace(req.Content),
		ContentType:  req.ContentType,
		Metadata:     req.Metadata,
		Status:       "active",
	}

	if comment.ContentType == "" {
		comment.ContentType = "markdown"
	}

	if userInfo != nil {
		comment.AuthorID = &userInfo.ID
		comment.AuthorName = &userInfo.Name
	}

	if comment.Metadata == nil {
		comment.Metadata = make(map[string]interface{})
	}

	// Validate content
	if len(comment.Content) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Comment content cannot be empty"})
		return
	}

	if len(comment.Content) > 10000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Comment content too long (max 10000 characters)"})
		return
	}

	// Create comment in database
	if err := app.db.CreateComment(comment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create comment"})
		return
	}

	// Render content
	if comment.ContentType == "markdown" {
		rendered := blackfriday.Run([]byte(comment.Content))
		comment.RenderedContent = app.sanitizer.Sanitize(string(rendered))
	} else {
		comment.RenderedContent = comment.Content
	}

	// Send notifications (async)
	go app.sendCommentNotifications(comment, config)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"comment": comment,
	})
}

func (app *App) updateComment(c *gin.Context) {
	commentIDStr := c.Param("id")
	_, err := uuid.Parse(commentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
		return
	}

	var req CommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate user authentication and ownership
	_, err = app.sessionAuth.ValidateToken(req.AuthorToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// TODO: Implement comment update logic
	// This would involve checking ownership, updating content, saving history, etc.

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Comment update not yet implemented",
	})
}

func (app *App) deleteComment(c *gin.Context) {
	commentIDStr := c.Param("id")
	_, err := uuid.Parse(commentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
		return
	}

	// TODO: Implement comment deletion (soft delete)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Comment deletion not yet implemented",
	})
}

func (app *App) getConfig(c *gin.Context) {
	scenarioName := c.Param("scenario")

	config, err := app.db.GetScenarioConfig(scenarioName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get configuration"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"config": config})
}

func (app *App) updateConfig(c *gin.Context) {
	scenarioName := c.Param("scenario")
	_ = scenarioName

	// TODO: Implement configuration update
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Configuration update not yet implemented",
	})
}

func (app *App) getStats(c *gin.Context) {
	// TODO: Implement admin statistics
	c.JSON(http.StatusOK, gin.H{
		"message": "Admin statistics not yet implemented",
	})
}

func (app *App) moderateComment(c *gin.Context) {
	commentIDStr := c.Param("id")
	_, err := uuid.Parse(commentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
		return
	}

	// TODO: Implement comment moderation

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Comment moderation not yet implemented",
	})
}

func (app *App) serveDocs(c *gin.Context) {
	docs := `
# Comment System API Documentation

## Endpoints

### GET /health
Health check endpoint

### GET /api/v1/comments/:scenario
Get comments for a scenario
- Query params: limit, offset, sort (newest|oldest|threaded), parent_id

### POST /api/v1/comments/:scenario
Create new comment
- Body: {content, parent_id?, content_type?, author_token, metadata?}

### GET /api/v1/config/:scenario
Get scenario configuration

## Integration

See the JavaScript SDK at /sdk/vrooli-comments.js for easy integration.
`

	c.Header("Content-Type", "text/plain")
	c.String(http.StatusOK, docs)
}

// Service integrations
func (app *App) testSessionAuth() string {
	resp, err := app.sessionAuth.client.Get(app.sessionAuth.baseURL + "/health")
	if err != nil {
		return "disconnected: " + err.Error()
	}
	resp.Body.Close()

	if resp.StatusCode == 200 {
		return "connected"
	}
	return "error: status " + resp.Status
}

func (app *App) testNotificationHub() string {
	resp, err := app.notifications.client.Get(app.notifications.baseURL + "/health")
	if err != nil {
		return "disconnected: " + err.Error()
	}
	resp.Body.Close()

	if resp.StatusCode == 200 {
		return "connected"
	}
	return "error: status " + resp.Status
}

func (auth *SessionAuthService) ValidateToken(token string) (*UserInfo, error) {
	if token == "" {
		return nil, fmt.Errorf("no token provided")
	}

	// TODO: Implement actual session-authenticator integration
	// For now, mock a response
	mockUser := &UserInfo{
		ID:       uuid.New(),
		Username: "testuser",
		Name:     "Test User",
	}

	return mockUser, nil
}

func (app *App) sendCommentNotifications(comment *Comment, config *ScenarioConfig) {
	// TODO: Implement notification sending via notification-hub
	// This would send notifications for:
	// - New replies to comments
	// - Mentions in comments
	// - New comments (if configured)

	log.Printf("Would send notifications for comment %s in scenario %s", comment.ID, comment.ScenarioName)
}

// Utility functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
