package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// Global structured logger
var logger *slog.Logger

// =============================================================================
// DATA STRUCTURES
// =============================================================================

// Profile represents a tenant in the multi-tenant system
type Profile struct {
	ID           uuid.UUID              `json:"id" db:"id"`
	Name         string                 `json:"name" db:"name"`
	Slug         string                 `json:"slug" db:"slug"`
	APIKeyPrefix string                 `json:"api_key_prefix" db:"api_key_prefix"`
	Settings     map[string]interface{} `json:"settings" db:"settings"`
	Plan         string                 `json:"plan" db:"plan"`
	Status       string                 `json:"status" db:"status"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
}

// Contact represents a notification recipient
type Contact struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	ProfileID   uuid.UUID              `json:"profile_id" db:"profile_id"`
	ExternalID  *string                `json:"external_id" db:"external_id"`
	Identifier  string                 `json:"identifier" db:"identifier"`
	FirstName   *string                `json:"first_name" db:"first_name"`
	LastName    *string                `json:"last_name" db:"last_name"`
	Timezone    string                 `json:"timezone" db:"timezone"`
	Locale      string                 `json:"locale" db:"locale"`
	Preferences map[string]interface{} `json:"preferences" db:"preferences"`
	Status      string                 `json:"status" db:"status"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

// Template represents a notification template
type Template struct {
	ID       uuid.UUID              `json:"id" db:"id"`
	Name     string                 `json:"name" db:"name"`
	Slug     string                 `json:"slug" db:"slug"`
	Channels []string               `json:"channels" db:"channels"`
	Category string                 `json:"category" db:"category"`
	Subject  *string                `json:"subject" db:"subject"`
	Content  map[string]interface{} `json:"content" db:"content"`
	Status   string                 `json:"status" db:"status"`
}

// Notification represents a notification instance
type Notification struct {
	ID                uuid.UUID              `json:"id" db:"id"`
	ProfileID         uuid.UUID              `json:"profile_id" db:"profile_id"`
	ContactID         uuid.UUID              `json:"contact_id" db:"contact_id"`
	TemplateID        *uuid.UUID             `json:"template_id" db:"template_id"`
	Subject           *string                `json:"subject" db:"subject"`
	Content           map[string]interface{} `json:"content" db:"content"`
	Variables         map[string]interface{} `json:"variables" db:"variables"`
	ChannelsRequested []string               `json:"channels_requested" db:"channels_requested"`
	ChannelsAttempted []string               `json:"channels_attempted" db:"channels_attempted"`
	Priority          string                 `json:"priority" db:"priority"`
	ScheduledAt       time.Time              `json:"scheduled_at" db:"scheduled_at"`
	Status            string                 `json:"status" db:"status"`
	CreatedAt         time.Time              `json:"created_at" db:"created_at"`
}

// NotificationRequest represents API request to send notification
type NotificationRequest struct {
	TemplateID  *string                `json:"template_id"`
	ContactID   *string                `json:"contact_id"`
	Recipients  []RecipientRequest     `json:"recipients"`
	Subject     *string                `json:"subject"`
	Content     map[string]interface{} `json:"content"`
	Variables   map[string]interface{} `json:"variables"`
	Channels    []string               `json:"channels"`
	Priority    string                 `json:"priority"`
	ScheduledAt *time.Time             `json:"scheduled_at"`
	ExternalID  *string                `json:"external_id"`
}

// RecipientRequest represents a recipient in a notification request
type RecipientRequest struct {
	ContactID  string                 `json:"contact_id"`
	Identifier *string                `json:"identifier"`
	Variables  map[string]interface{} `json:"variables"`
}

// Server represents the HTTP server
type Server struct {
	db        *sql.DB
	redis     *redis.Client
	router    *gin.Engine
	port      string
	processor *NotificationProcessor
}

// =============================================================================
// SERVER INITIALIZATION
// =============================================================================

var startTime time.Time

func main() {
	// Lifecycle validation: Ensure binary is run through Vrooli (not directly)
	// This environment variable is set by the lifecycle system and is intentionally optional
	// (will not exist if running binary directly, which is what we're preventing)
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start notification-hub

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Initialize structured logger
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	startTime = time.Now()
	server, err := NewServer()
	if err != nil {
		logger.Error("Failed to create server", "error", err)
		os.Exit(1)
	}

	logger.Info("Notification Hub API starting", "port", server.port)
	logger.Info("Notification processor initialized", "workers", 5)

	// Start background notification processing
	go server.startBackgroundProcessing()

	if err := server.router.Run(":" + server.port); err != nil {
		logger.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}

func NewServer() (*Server, error) {
	// Environment variables - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		return nil, fmt.Errorf("API_PORT environment variable is required")
	}
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return nil, fmt.Errorf("REDIS_URL environment variable is required")
	}

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
			return nil, fmt.Errorf("missing database configuration. Provide POSTGRES_URL or all required database connection parameters")
		}

		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	// Initialize database
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

	logger.Info("Attempting database connection with exponential backoff")
	logger.Info("Database URL configured")

	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			logger.Info("Database connected successfully", "attempt", attempt+1)
			break
		}

		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay)*math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))

		// Add progressive jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))
		actualDelay := delay + jitter

		logger.Warn("Connection attempt failed", "attempt", attempt+1, "max_retries", maxRetries, "error", pingErr)
		logger.Info("Waiting before next attempt", "delay", actualDelay)

		// Provide detailed status every few attempts
		if attempt > 0 && attempt%3 == 0 {
			logger.Info("Retry progress",
				"attempts_made", attempt+1,
				"max_retries", maxRetries,
				"total_wait_time", time.Duration(attempt*2)*baseDelay,
				"current_delay", delay,
				"jitter", jitter)
		}

		time.Sleep(actualDelay)
	}

	if pingErr != nil {
		return nil, fmt.Errorf("Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}

	logger.Info("Database connection pool established successfully")

	// Initialize Redis
	redisOpts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	rdb := redis.NewClient(redisOpts)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Create notification processor
	processor := NewNotificationProcessor(db, rdb)

	server := &Server{
		db:        db,
		redis:     rdb,
		port:      port,
		processor: processor,
	}

	server.setupRoutes()
	return server, nil
}

func (s *Server) setupRoutes() {
	gin.SetMode(gin.ReleaseMode)
	s.router = gin.Default()

	// Add CORS middleware with explicit origin control
	s.router.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Build allowed origins from environment
		// UI_PORT is required since UI is a core component
		uiPort := os.Getenv("UI_PORT")
		if uiPort == "" {
			logger.Error("UI_PORT environment variable is required for CORS configuration")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Server configuration error"})
			return
		}

		// N8N_URL is optional - only add if n8n resource is enabled
		// Default origins include localhost and 127.0.0.1 for local development
		allowedOrigins := []string{
			"http://localhost:" + uiPort,
			"http://127.0.0.1:" + uiPort, // Required for health checks and local testing
		}

		// N8N_URL is optional: only used when n8n workflow integration is enabled
		// Gracefully handles absence by only adding to CORS if present
		n8nURL := os.Getenv("N8N_URL")
		if n8nURL != "" {
			allowedOrigins = append(allowedOrigins, n8nURL)
		}

		// Check if origin is in allowed list
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				c.Header("Access-Control-Allow-Origin", origin)
				break
			}
		}

		// If no origin header (e.g., CLI/Postman), allow but don't set CORS header
		if origin == "" {
			allowed = true
		}

		if allowed {
			c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization,X-API-Key")
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == "OPTIONS" {
			if allowed {
				c.AbortWithStatus(204)
			} else {
				c.AbortWithStatus(403)
			}
			return
		}
		c.Next()
	})

	// Health check
	s.router.GET("/health", s.healthCheck)
	s.router.GET("/docs", s.apiDocs)

	// Public endpoints
	public := s.router.Group("/api/v1")
	public.POST("/webhooks/unsubscribe", s.handleUnsubscribe)

	// Profile management (admin endpoints)
	admin := s.router.Group("/api/v1/admin")
	admin.POST("/profiles", s.createProfile)
	admin.GET("/profiles", s.listProfiles)
	admin.GET("/profiles/:id", s.getProfile)
	admin.PUT("/profiles/:id", s.updateProfile)

	// Profile-scoped endpoints (require API key authentication)
	api := s.router.Group("/api/v1/profiles/:profile_id")
	api.Use(s.authenticateAPIKey())

	// Notification endpoints
	api.POST("/notifications/send", s.sendNotification)
	api.GET("/notifications", s.listNotifications)
	api.GET("/notifications/:id", s.getNotification)
	api.GET("/notifications/:id/status", s.getNotificationStatus)

	// Contact management
	api.POST("/contacts", s.createContact)
	api.GET("/contacts", s.listContacts)
	api.GET("/contacts/:id", s.getContact)
	api.PUT("/contacts/:id", s.updateContact)
	api.PUT("/contacts/:id/preferences", s.updateContactPreferences)

	// Template management
	api.POST("/templates", s.createTemplate)
	api.GET("/templates", s.listTemplates)
	api.GET("/templates/:id", s.getTemplate)
	api.PUT("/templates/:id", s.updateTemplate)

	// Analytics
	api.GET("/analytics/delivery-stats", s.getDeliveryStats)
	api.GET("/analytics/daily-stats", s.getDailyStats)
}

// =============================================================================
// MIDDLEWARE
// =============================================================================

func (s *Server) authenticateAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		profileID := c.Param("profile_id")
		apiKey := c.GetHeader("X-API-Key")

		if apiKey == "" {
			apiKey = c.GetHeader("Authorization")
			if strings.HasPrefix(apiKey, "Bearer ") {
				apiKey = strings.TrimPrefix(apiKey, "Bearer ")
			}
		}

		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			c.Abort()
			return
		}

		// Verify API key and profile
		var profile Profile
		var apiKeyHash string
		var settingsJSON []byte
		query := `
			SELECT id, name, slug, api_key_prefix, api_key_hash, settings, plan, status, created_at, updated_at
			FROM profiles
			WHERE id = $1 AND status = 'active'
		`

		logger.Info("Authenticating API request", "profile_id", profileID, "api_key_prefix", apiKey[:min(len(apiKey), 6)])

		err := s.db.QueryRow(query, profileID).Scan(
			&profile.ID, &profile.Name, &profile.Slug, &profile.APIKeyPrefix,
			&apiKeyHash, &settingsJSON, &profile.Plan, &profile.Status,
			&profile.CreatedAt, &profile.UpdatedAt,
		)

		if err != nil {
			logger.Error("Profile lookup failed", "error", err, "profile_id", profileID)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key or profile"})
			c.Abort()
			return
		}

		// Parse settings JSON
		if len(settingsJSON) > 0 {
			if err := json.Unmarshal(settingsJSON, &profile.Settings); err != nil {
				logger.Error("Failed to parse settings", "error", err)
				profile.Settings = make(map[string]interface{})
			}
		} else {
			profile.Settings = make(map[string]interface{})
		}

		// Compare API key with bcrypt hash
		if err := bcrypt.CompareHashAndPassword([]byte(apiKeyHash), []byte(apiKey)); err != nil {
			logger.Error("API key verification failed", "error", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key or profile"})
			c.Abort()
			return
		}

		logger.Info("Authentication successful", "profile_id", profileID, "profile_name", profile.Name)

		c.Set("profile", profile)
		c.Next()
	}
}

// =============================================================================
// API ENDPOINTS
// =============================================================================

func (s *Server) healthCheck(c *gin.Context) {
	overallStatus := "healthy"
	var errors []map[string]interface{}
	readiness := true

	// Schema-compliant health response
	healthResponse := map[string]interface{}{
		"status":       overallStatus,
		"service":      "notification-hub-api",
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
		"readiness":    true,
		"dependencies": map[string]interface{}{},
	}

	// Check database connectivity and operations
	dbHealth := s.checkDatabaseHealth()
	healthResponse["dependencies"].(map[string]interface{})["database"] = dbHealth
	if dbHealth["status"] != "healthy" {
		overallStatus = "degraded"
		if dbHealth["status"] == "unhealthy" {
			readiness = false
		}
		if dbHealth["error"] != nil {
			errors = append(errors, dbHealth["error"].(map[string]interface{}))
		}
	}

	// Check Redis connectivity and cache operations
	redisHealth := s.checkRedisHealth()
	healthResponse["dependencies"].(map[string]interface{})["redis"] = redisHealth
	if redisHealth["status"] != "healthy" {
		overallStatus = "degraded"
		if redisHealth["status"] == "unhealthy" {
			readiness = false
		}
		if redisHealth["error"] != nil {
			errors = append(errors, redisHealth["error"].(map[string]interface{}))
		}
	}

	// Check notification processor system
	processorHealth := s.checkNotificationProcessor()
	healthResponse["dependencies"].(map[string]interface{})["notification_processor"] = processorHealth
	if processorHealth["status"] != "healthy" {
		if overallStatus != "unhealthy" {
			overallStatus = "degraded"
		}
		if processorHealth["error"] != nil {
			errors = append(errors, processorHealth["error"].(map[string]interface{}))
		}
	}

	// Check profile system functionality
	profileHealth := s.checkProfileSystem()
	healthResponse["dependencies"].(map[string]interface{})["profile_system"] = profileHealth
	if profileHealth["status"] != "healthy" {
		if overallStatus != "unhealthy" {
			overallStatus = "degraded"
		}
		if profileHealth["error"] != nil {
			errors = append(errors, profileHealth["error"].(map[string]interface{}))
		}
	}

	// Check template management system
	templateHealth := s.checkTemplateSystem()
	healthResponse["dependencies"].(map[string]interface{})["template_system"] = templateHealth
	if templateHealth["status"] != "healthy" {
		if overallStatus != "unhealthy" {
			overallStatus = "degraded"
		}
		if templateHealth["error"] != nil {
			errors = append(errors, templateHealth["error"].(map[string]interface{}))
		}
	}

	// Update final status
	healthResponse["status"] = overallStatus
	healthResponse["readiness"] = readiness

	// Add errors if any
	if len(errors) > 0 {
		healthResponse["errors"] = errors
	}

	// Add metrics
	healthResponse["metrics"] = map[string]interface{}{
		"total_dependencies":   5,
		"healthy_dependencies": s.countHealthyDependencies(healthResponse["dependencies"].(map[string]interface{})),
		"uptime_seconds":       time.Now().Unix() - startTime.Unix(),
	}

	// Return appropriate HTTP status
	statusCode := 200
	if overallStatus == "unhealthy" {
		statusCode = 503
	} else if overallStatus == "degraded" {
		statusCode = 200 // Still operational but with warnings
	}

	c.JSON(statusCode, healthResponse)
}

// Health check helper methods
func (s *Server) checkDatabaseHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}

	// Test database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.db.PingContext(ctx); err != nil {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code":      "DATABASE_CONNECTION_FAILED",
			"message":   "Database connection failed: " + err.Error(),
			"category":  "resource",
			"retryable": true,
		}
		return health
	}
	health["checks"].(map[string]interface{})["ping"] = "ok"

	// Test profiles table query
	var profileCount int
	query := "SELECT COUNT(*) FROM profiles WHERE status = 'active'"
	if err := s.db.QueryRowContext(ctx, query).Scan(&profileCount); err != nil {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code":      "DATABASE_QUERY_FAILED",
			"message":   "Failed to query profiles table: " + err.Error(),
			"category":  "resource",
			"retryable": true,
		}
	} else {
		health["checks"].(map[string]interface{})["profiles_query"] = "ok"
		health["checks"].(map[string]interface{})["active_profiles"] = profileCount
	}

	// Test notifications table query
	var notificationCount int
	recentQuery := "SELECT COUNT(*) FROM notifications WHERE created_at > NOW() - INTERVAL '1 hour'"
	if err := s.db.QueryRowContext(ctx, recentQuery).Scan(&notificationCount); err != nil {
		if health["status"] == "healthy" {
			health["status"] = "degraded"
		}
		if health["error"] == nil {
			health["error"] = map[string]interface{}{
				"code":      "DATABASE_QUERY_FAILED",
				"message":   "Failed to query notifications table: " + err.Error(),
				"category":  "resource",
				"retryable": true,
			}
		}
	} else {
		health["checks"].(map[string]interface{})["notifications_query"] = "ok"
		health["checks"].(map[string]interface{})["recent_notifications"] = notificationCount
	}

	return health
}

func (s *Server) checkRedisHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Test Redis connection
	if err := s.redis.Ping(ctx).Err(); err != nil {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code":      "REDIS_CONNECTION_FAILED",
			"message":   "Redis connection failed: " + err.Error(),
			"category":  "resource",
			"retryable": true,
		}
		return health
	}
	health["checks"].(map[string]interface{})["ping"] = "ok"

	// Test cache operations
	testKey := "health_check:" + fmt.Sprintf("%d", time.Now().UnixNano())
	testValue := "test_value"

	if err := s.redis.Set(ctx, testKey, testValue, time.Minute).Err(); err != nil {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code":      "REDIS_WRITE_FAILED",
			"message":   "Redis write operation failed: " + err.Error(),
			"category":  "resource",
			"retryable": true,
		}
	} else {
		health["checks"].(map[string]interface{})["write"] = "ok"

		// Test read operation
		if val, err := s.redis.Get(ctx, testKey).Result(); err != nil || val != testValue {
			health["status"] = "degraded"
			if health["error"] == nil {
				health["error"] = map[string]interface{}{
					"code":      "REDIS_READ_FAILED",
					"message":   "Redis read operation failed",
					"category":  "resource",
					"retryable": true,
				}
			}
		} else {
			health["checks"].(map[string]interface{})["read"] = "ok"
		}

		// Clean up test key
		s.redis.Del(ctx, testKey)
	}

	return health
}

func (s *Server) checkNotificationProcessor() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}

	if s.processor == nil {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code":      "PROCESSOR_NOT_INITIALIZED",
			"message":   "Notification processor not initialized",
			"category":  "internal",
			"retryable": false,
		}
		return health
	}

	health["checks"].(map[string]interface{})["initialized"] = "ok"

	// Check for pending notifications (this gives insight into processor workload)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var pendingCount int
	query := "SELECT COUNT(*) FROM notifications WHERE status = 'pending'"
	if err := s.db.QueryRowContext(ctx, query).Scan(&pendingCount); err != nil {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code":      "PROCESSOR_STATUS_CHECK_FAILED",
			"message":   "Failed to check processor queue: " + err.Error(),
			"category":  "resource",
			"retryable": true,
		}
	} else {
		health["checks"].(map[string]interface{})["queue_check"] = "ok"
		health["checks"].(map[string]interface{})["pending_notifications"] = pendingCount

		// Warning if too many pending notifications
		if pendingCount > 100 {
			health["status"] = "degraded"
			health["error"] = map[string]interface{}{
				"code":      "HIGH_QUEUE_BACKLOG",
				"message":   fmt.Sprintf("High number of pending notifications: %d", pendingCount),
				"category":  "resource",
				"retryable": true,
			}
		}
	}

	return health
}

func (s *Server) checkProfileSystem() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Check profiles table structure and basic operations
	var totalProfiles, activeProfiles int

	// Count total profiles
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM profiles").Scan(&totalProfiles); err != nil {
		health["status"] = "unhealthy"
		health["error"] = map[string]interface{}{
			"code":      "PROFILE_TABLE_ACCESS_FAILED",
			"message":   "Cannot access profiles table: " + err.Error(),
			"category":  "resource",
			"retryable": true,
		}
		return health
	}

	// Count active profiles
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM profiles WHERE status = 'active'").Scan(&activeProfiles); err != nil {
		health["status"] = "degraded"
		health["error"] = map[string]interface{}{
			"code":      "PROFILE_QUERY_FAILED",
			"message":   "Failed to query active profiles: " + err.Error(),
			"category":  "resource",
			"retryable": true,
		}
	} else {
		health["checks"].(map[string]interface{})["table_access"] = "ok"
		health["checks"].(map[string]interface{})["total_profiles"] = totalProfiles
		health["checks"].(map[string]interface{})["active_profiles"] = activeProfiles
	}

	return health
}

func (s *Server) checkTemplateSystem() map[string]interface{} {
	health := map[string]interface{}{
		"status": "healthy",
		"checks": map[string]interface{}{},
	}

	// For now, this is a basic check since templates aren't fully implemented
	// In a full implementation, this would check template storage, validation, etc.
	health["checks"].(map[string]interface{})["placeholder"] = "ok"
	health["checks"].(map[string]interface{})["note"] = "Template system checks not fully implemented"

	return health
}

func (s *Server) countHealthyDependencies(deps map[string]interface{}) int {
	count := 0
	for _, dep := range deps {
		if depMap, ok := dep.(map[string]interface{}); ok {
			if status, exists := depMap["status"]; exists && status == "healthy" {
				count++
			}
		}
	}
	return count
}

func (s *Server) apiDocs(c *gin.Context) {
	// Get the actual API port from environment (required by lifecycle system)
	apiPort := os.Getenv("API_PORT")
	// API_PORT is always set by the Vrooli lifecycle system, so no fallback needed
	baseURL := fmt.Sprintf("http://localhost:%s", apiPort)

	docs := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Notification Hub API Documentation</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 1200px; margin: 0 auto; padding: 20px; }
        .endpoint { background: #f5f5f5; padding: 15px; margin: 10px 0; border-radius: 5px; }
        .method { font-weight: bold; color: #007cba; }
        .url { font-family: monospace; background: #333; color: #fff; padding: 2px 6px; border-radius: 3px; }
    </style>
</head>
<body>
    <h1>🔔 Notification Hub API</h1>
    <p>Multi-tenant notification management system for email, SMS, and push notifications.</p>

    <h2>Authentication</h2>
    <p>Use profile-scoped API keys in the <code>X-API-Key</code> header or <code>Authorization: Bearer</code> header.</p>

    <h2>Core Endpoints</h2>

    <div class="endpoint">
        <span class="method">POST</span> <span class="url">/api/v1/profiles/{profile_id}/notifications/send</span>
        <p>Send notifications via multiple channels. Supports templates, variables, and channel preferences.</p>
    </div>

    <div class="endpoint">
        <span class="method">GET</span> <span class="url">/api/v1/profiles/{profile_id}/notifications</span>
        <p>List notifications with filtering and pagination.</p>
    </div>

    <div class="endpoint">
        <span class="method">POST</span> <span class="url">/api/v1/profiles/{profile_id}/contacts</span>
        <p>Create notification recipients with channel preferences.</p>
    </div>

    <div class="endpoint">
        <span class="method">GET</span> <span class="url">/api/v1/profiles/{profile_id}/analytics/delivery-stats</span>
        <p>Get delivery statistics and performance metrics.</p>
    </div>

    <h2>Example Request</h2>
    <pre>
curl -X POST %s/api/v1/profiles/your-profile-id/notifications/send \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "template_id": "welcome-email",
    "recipients": [{
      "contact_id": "contact-uuid",
      "variables": {"name": "John", "company": "Acme Corp"}
    }],
    "channels": ["email", "push"],
    "priority": "normal"
  }'
    </pre>

    <p>For detailed API documentation, see the <a href="/health">health endpoint</a> for service status.</p>
</body>
</html>
	`, baseURL)
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, docs)
}

// Profile Management
func (s *Server) createProfile(c *gin.Context) {
	var req struct {
		Name     string                 `json:"name" binding:"required"`
		Slug     string                 `json:"slug"`
		Settings map[string]interface{} `json:"settings"`
		Plan     string                 `json:"plan"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Generate slug if not provided
	if req.Slug == "" {
		req.Slug = strings.ToLower(strings.ReplaceAll(req.Name, " ", "-"))
	}

	// Generate API key
	apiKey, apiKeyHash, apiKeyPrefix, err := s.generateAPIKey(req.Slug)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate API key"})
		return
	}

	// Convert settings to JSON for JSONB column
	settingsJSON := []byte("{}")
	if req.Settings != nil {
		settingsJSON, err = json.Marshal(req.Settings)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid settings format"})
			return
		}
	}

	// Create profile
	profileID := uuid.New()
	query := `
		INSERT INTO profiles (id, name, slug, api_key_hash, api_key_prefix, settings, plan, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'active')
		RETURNING created_at, updated_at
	`

	var createdAt, updatedAt time.Time
	err = s.db.QueryRow(query, profileID, req.Name, req.Slug, apiKeyHash, apiKeyPrefix,
		settingsJSON, req.Plan).Scan(&createdAt, &updatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create profile: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":             profileID,
		"name":           req.Name,
		"slug":           req.Slug,
		"api_key":        apiKey, // Only returned on creation
		"api_key_prefix": apiKeyPrefix,
		"plan":           req.Plan,
		"status":         "active",
		"created_at":     createdAt,
		"updated_at":     updatedAt,
	})
}

func (s *Server) listProfiles(c *gin.Context) {
	query := `
		SELECT id, name, slug, api_key_prefix, settings, plan, status, created_at, updated_at
		FROM profiles
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch profiles"})
		return
	}
	defer rows.Close()

	var profiles []Profile
	for rows.Next() {
		var p Profile
		var settingsJSON []byte
		err := rows.Scan(&p.ID, &p.Name, &p.Slug, &p.APIKeyPrefix,
			&settingsJSON, &p.Plan, &p.Status, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			logger.Error("Failed to scan profile row", "error", err)
			continue
		}

		// Unmarshal settings JSON
		if len(settingsJSON) > 0 {
			if err := json.Unmarshal(settingsJSON, &p.Settings); err != nil {
				logger.Error("Failed to unmarshal settings", "error", err, "profile_id", p.ID)
				p.Settings = make(map[string]interface{})
			}
		} else {
			p.Settings = make(map[string]interface{})
		}

		profiles = append(profiles, p)
	}

	c.JSON(http.StatusOK, gin.H{"profiles": profiles})
}

func (s *Server) getProfile(c *gin.Context) {
	profileID := c.Param("id")

	var profile Profile
	var settingsJSON []byte
	query := `
		SELECT id, name, slug, api_key_prefix, settings, plan, status, created_at, updated_at
		FROM profiles
		WHERE id = $1
	`

	err := s.db.QueryRow(query, profileID).Scan(
		&profile.ID, &profile.Name, &profile.Slug, &profile.APIKeyPrefix,
		&settingsJSON, &profile.Plan, &profile.Status,
		&profile.CreatedAt, &profile.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
		return
	}

	// Unmarshal settings JSON
	if len(settingsJSON) > 0 {
		if err := json.Unmarshal(settingsJSON, &profile.Settings); err != nil {
			logger.Error("Failed to unmarshal settings", "error", err, "profile_id", profile.ID)
			profile.Settings = make(map[string]interface{})
		}
	} else {
		profile.Settings = make(map[string]interface{})
	}

	c.JSON(http.StatusOK, profile)
}

func (s *Server) updateProfile(c *gin.Context) {
	profileID := c.Param("id")

	var req struct {
		Name     *string                `json:"name"`
		Settings map[string]interface{} `json:"settings"`
		Plan     *string                `json:"plan"`
		Status   *string                `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Build dynamic update query
	setParts := []string{"updated_at = NOW()"}
	args := []interface{}{profileID}
	argIndex := 2

	if req.Name != nil {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *req.Name)
		argIndex++
	}
	if req.Settings != nil {
		setParts = append(setParts, fmt.Sprintf("settings = $%d", argIndex))
		args = append(args, req.Settings)
		argIndex++
	}
	if req.Plan != nil {
		setParts = append(setParts, fmt.Sprintf("plan = $%d", argIndex))
		args = append(args, *req.Plan)
		argIndex++
	}
	if req.Status != nil {
		setParts = append(setParts, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *req.Status)
		argIndex++
	}

	query := fmt.Sprintf("UPDATE profiles SET %s WHERE id = $1", strings.Join(setParts, ", "))

	_, err := s.db.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	// Return updated profile
	s.getProfile(c)
}

// Notification Management
func (s *Server) sendNotification(c *gin.Context) {
	profile := c.MustGet("profile").(Profile)

	var req NotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Set defaults
	if req.Priority == "" {
		req.Priority = "normal"
	}
	if req.Channels == nil || len(req.Channels) == 0 {
		req.Channels = []string{"email"}
	}
	if req.Variables == nil {
		req.Variables = make(map[string]interface{})
	}

	notifications := []uuid.UUID{}

	// Process each recipient
	for _, recipient := range req.Recipients {
		notificationID := uuid.New()

		// Create notification record
		query := `
			INSERT INTO notifications (
				id, profile_id, contact_id, template_id, subject, content, variables,
				channels_requested, priority, scheduled_at, status, external_id
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'pending', $11)
		`

		scheduledAt := time.Now()
		if req.ScheduledAt != nil {
			scheduledAt = *req.ScheduledAt
		}

		var templateUUID *uuid.UUID
		if req.TemplateID != nil {
			parsed, err := uuid.Parse(*req.TemplateID)
			if err == nil {
				templateUUID = &parsed
			}
		}

		contactUUID, err := uuid.Parse(recipient.ContactID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contact_id: " + recipient.ContactID})
			return
		}

		// Merge recipient variables with request variables
		mergedVariables := make(map[string]interface{})
		for k, v := range req.Variables {
			mergedVariables[k] = v
		}
		for k, v := range recipient.Variables {
			mergedVariables[k] = v
		}

		// Convert maps to JSON for JSONB columns
		contentJSON := []byte("{}")
		if req.Content != nil {
			contentJSON, err = json.Marshal(req.Content)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid content format"})
				return
			}
		}

		variablesJSON, err := json.Marshal(mergedVariables)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid variables format"})
			return
		}

		_, err = s.db.Exec(query, notificationID, profile.ID, contactUUID, templateUUID,
			req.Subject, contentJSON, variablesJSON, pq.Array(req.Channels), req.Priority,
			scheduledAt, req.ExternalID)

		if err != nil {
			logger.Error("Failed to create notification", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create notification"})
			return
		}

		notifications = append(notifications, notificationID)
	}

	// Process notifications immediately if not scheduled for future
	if req.ScheduledAt == nil || req.ScheduledAt.Before(time.Now().Add(1*time.Minute)) {
		go s.processor.ProcessPendingNotifications()
	}

	c.JSON(http.StatusCreated, gin.H{
		"notifications": notifications,
		"message":       fmt.Sprintf("Created %d notifications", len(notifications)),
	})
}

// startBackgroundProcessing runs periodic notification processing
func (s *Server) startBackgroundProcessing() {
	ticker := time.NewTicker(10 * time.Second) // Process every 10 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.processor.ProcessPendingNotifications(); err != nil {
				logger.Error("Error processing notifications", "error", err)
			}
		}
	}
}

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

func (s *Server) generateAPIKey(slug string) (string, string, string, error) {
	// Generate random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", "", err
	}

	// Create API key with prefix
	prefix := slug[:min(len(slug), 4)] + "_"
	apiKey := prefix + hex.EncodeToString(bytes)

	// Hash the API key
	hash, err := bcrypt.GenerateFromPassword([]byte(apiKey), bcrypt.DefaultCost)
	if err != nil {
		return "", "", "", err
	}

	return apiKey, string(hash), prefix, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Additional endpoint stubs (implement as needed)
func (s *Server) listNotifications(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"notifications": []Notification{}})
}

func (s *Server) getNotification(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "Not implemented"})
}

func (s *Server) getNotificationStatus(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "Not implemented"})
}

func (s *Server) createContact(c *gin.Context) {
	profile := c.MustGet("profile").(Profile)

	var req struct {
		ExternalID  *string                `json:"external_id"`
		Identifier  string                 `json:"identifier" binding:"required"`
		FirstName   *string                `json:"first_name"`
		LastName    *string                `json:"last_name"`
		Timezone    string                 `json:"timezone"`
		Locale      string                 `json:"locale"`
		Preferences map[string]interface{} `json:"preferences"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Set defaults
	if req.Timezone == "" {
		req.Timezone = "UTC"
	}
	if req.Locale == "" {
		req.Locale = "en-US"
	}

	// Convert preferences to JSON
	preferencesJSON := []byte("{}")
	if req.Preferences != nil {
		var err error
		preferencesJSON, err = json.Marshal(req.Preferences)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid preferences format"})
			return
		}
	}

	// Create contact
	contactID := uuid.New()
	query := `
		INSERT INTO contacts (
			id, profile_id, external_id, identifier, first_name, last_name,
			timezone, locale, preferences, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'active')
		RETURNING created_at, updated_at
	`

	var createdAt, updatedAt time.Time
	err := s.db.QueryRow(query, contactID, profile.ID, req.ExternalID, req.Identifier,
		req.FirstName, req.LastName, req.Timezone, req.Locale, preferencesJSON).
		Scan(&createdAt, &updatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create contact: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":          contactID,
		"profile_id":  profile.ID,
		"external_id": req.ExternalID,
		"identifier":  req.Identifier,
		"first_name":  req.FirstName,
		"last_name":   req.LastName,
		"timezone":    req.Timezone,
		"locale":      req.Locale,
		"status":      "active",
		"created_at":  createdAt,
		"updated_at":  updatedAt,
	})
}

func (s *Server) listContacts(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"contacts": []Contact{}})
}

func (s *Server) getContact(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "Not implemented"})
}

func (s *Server) updateContact(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "Not implemented"})
}

func (s *Server) updateContactPreferences(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "Not implemented"})
}

func (s *Server) createTemplate(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "Not implemented"})
}

func (s *Server) listTemplates(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"templates": []Template{}})
}

func (s *Server) getTemplate(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "Not implemented"})
}

func (s *Server) updateTemplate(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "Not implemented"})
}

func (s *Server) getDeliveryStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"stats": map[string]interface{}{}})
}

func (s *Server) getDailyStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"daily_stats": []interface{}{}})
}

func (s *Server) handleUnsubscribe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Unsubscribe processed"})
}
