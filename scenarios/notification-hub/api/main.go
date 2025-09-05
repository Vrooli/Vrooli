package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

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
	ID           uuid.UUID              `json:"id" db:"id"`
	ProfileID    uuid.UUID              `json:"profile_id" db:"profile_id"`
	ExternalID   *string                `json:"external_id" db:"external_id"`
	Identifier   string                 `json:"identifier" db:"identifier"`
	FirstName    *string                `json:"first_name" db:"first_name"`
	LastName     *string                `json:"last_name" db:"last_name"`
	Timezone     string                 `json:"timezone" db:"timezone"`
	Locale       string                 `json:"locale" db:"locale"`
	Preferences  map[string]interface{} `json:"preferences" db:"preferences"`
	Status       string                 `json:"status" db:"status"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
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
	TemplateID        *string                `json:"template_id"`
	ContactID         *string                `json:"contact_id"`
	Recipients        []RecipientRequest     `json:"recipients"`
	Subject           *string                `json:"subject"`
	Content           map[string]interface{} `json:"content"`
	Variables         map[string]interface{} `json:"variables"`
	Channels          []string               `json:"channels"`
	Priority          string                 `json:"priority"`
	ScheduledAt       *time.Time             `json:"scheduled_at"`
	ExternalID        *string                `json:"external_id"`
}

// RecipientRequest represents a recipient in a notification request
type RecipientRequest struct {
	ContactID   string                 `json:"contact_id"`
	Identifier  *string                `json:"identifier"`
	Variables   map[string]interface{} `json:"variables"`
}

// Server represents the HTTP server
type Server struct {
	db     *sql.DB
	redis  *redis.Client
	router *gin.Engine
	port   string
	n8nURL string
}

// =============================================================================
// SERVER INITIALIZATION
// =============================================================================

func main() {
	server, err := NewServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	log.Printf("🔔 Notification Hub API starting on port %s", server.port)
	log.Printf("🔗 n8n workflows available at %s", server.n8nURL)

	if err := server.router.Run(":" + server.port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func NewServer() (*Server, error) {
	// Environment variables
	port := getEnv("API_PORT", getEnv("PORT", ""))
	postgresURL := getEnv("POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/notification_hub?sslmode=disable")
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379")
	n8nURL := getEnv("N8N_BASE_URL", "http://localhost:5678")

	// Initialize database
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize Redis
	redisOpts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	rdb := redis.NewClient(redisOpts)
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	server := &Server{
		db:     db,
		redis:  rdb,
		port:   port,
		n8nURL: n8nURL,
	}

	server.setupRoutes()
	return server, nil
}

func (s *Server) setupRoutes() {
	gin.SetMode(gin.ReleaseMode)
	s.router = gin.Default()

	// Add CORS middleware
	s.router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization,X-API-Key")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
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
			c.JSON(401, gin.H{"error": "API key required"})
			c.Abort()
			return
		}

		// Verify API key and profile
		var profile Profile
		query := `
			SELECT id, name, slug, api_key_prefix, settings, plan, status, created_at, updated_at
			FROM profiles 
			WHERE id = $1 AND api_key_hash = crypt($2, api_key_hash) AND status = 'active'
		`
		
		err := s.db.QueryRow(query, profileID, apiKey).Scan(
			&profile.ID, &profile.Name, &profile.Slug, &profile.APIKeyPrefix,
			&profile.Settings, &profile.Plan, &profile.Status,
			&profile.CreatedAt, &profile.UpdatedAt,
		)

		if err != nil {
			c.JSON(401, gin.H{"error": "Invalid API key or profile"})
			c.Abort()
			return
		}

		c.Set("profile", profile)
		c.Next()
	}
}

// =============================================================================
// API ENDPOINTS
// =============================================================================

func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "healthy",
		"service": "notification-hub-api",
		"version": "1.0.0",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (s *Server) apiDocs(c *gin.Context) {
	docs := `
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
curl -X POST http://localhost:28100/api/v1/profiles/your-profile-id/notifications/send \
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
	`
	c.Header("Content-Type", "text/html")
	c.String(200, docs)
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
		c.JSON(400, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Generate slug if not provided
	if req.Slug == "" {
		req.Slug = strings.ToLower(strings.ReplaceAll(req.Name, " ", "-"))
	}

	// Generate API key
	apiKey, apiKeyHash, apiKeyPrefix, err := s.generateAPIKey(req.Slug)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to generate API key"})
		return
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
		req.Settings, req.Plan).Scan(&createdAt, &updatedAt)

	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to create profile: " + err.Error()})
		return
	}

	c.JSON(201, gin.H{
		"id": profileID,
		"name": req.Name,
		"slug": req.Slug,
		"api_key": apiKey, // Only returned on creation
		"api_key_prefix": apiKeyPrefix,
		"plan": req.Plan,
		"status": "active",
		"created_at": createdAt,
		"updated_at": updatedAt,
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
		c.JSON(500, gin.H{"error": "Failed to fetch profiles"})
		return
	}
	defer rows.Close()

	var profiles []Profile
	for rows.Next() {
		var p Profile
		err := rows.Scan(&p.ID, &p.Name, &p.Slug, &p.APIKeyPrefix,
			&p.Settings, &p.Plan, &p.Status, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			continue
		}
		profiles = append(profiles, p)
	}

	c.JSON(200, gin.H{"profiles": profiles})
}

func (s *Server) getProfile(c *gin.Context) {
	profileID := c.Param("id")
	
	var profile Profile
	query := `
		SELECT id, name, slug, api_key_prefix, settings, plan, status, created_at, updated_at
		FROM profiles
		WHERE id = $1
	`
	
	err := s.db.QueryRow(query, profileID).Scan(
		&profile.ID, &profile.Name, &profile.Slug, &profile.APIKeyPrefix,
		&profile.Settings, &profile.Plan, &profile.Status,
		&profile.CreatedAt, &profile.UpdatedAt,
	)

	if err != nil {
		c.JSON(404, gin.H{"error": "Profile not found"})
		return
	}

	c.JSON(200, profile)
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
		c.JSON(400, gin.H{"error": "Invalid request: " + err.Error()})
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
		c.JSON(500, gin.H{"error": "Failed to update profile"})
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
		c.JSON(400, gin.H{"error": "Invalid request: " + err.Error()})
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
			c.JSON(400, gin.H{"error": "Invalid contact_id: " + recipient.ContactID})
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

		_, err = s.db.Exec(query, notificationID, profile.ID, contactUUID, templateUUID,
			req.Subject, req.Content, mergedVariables, req.Channels, req.Priority,
			scheduledAt, req.ExternalID)

		if err != nil {
			log.Printf("Failed to create notification: %v", err)
			c.JSON(500, gin.H{"error": "Failed to create notification"})
			return
		}

		notifications = append(notifications, notificationID)

		// Trigger n8n workflow for immediate processing
		go s.triggerNotificationWorkflow(notificationID, profile.ID, contactUUID, templateUUID)
	}

	c.JSON(201, gin.H{
		"notifications": notifications,
		"message": fmt.Sprintf("Created %d notifications", len(notifications)),
	})
}

func (s *Server) triggerNotificationWorkflow(notificationID, profileID, contactID uuid.UUID, templateID *uuid.UUID) {
	payload := map[string]interface{}{
		"notification_id": notificationID,
		"profile_id":      profileID,
		"contact_id":      contactID,
		"template_id":     templateID,
	}

	// Call n8n webhook
	client := &http.Client{Timeout: 30 * time.Second}
	payloadBytes, _ := json.Marshal(payload)
	
	_, err := client.Post(s.n8nURL+"/webhook/notification-router", "application/json", strings.NewReader(string(payloadBytes)))
	if err != nil {
		log.Printf("Failed to trigger n8n workflow: %v", err)
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

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Additional endpoint stubs (implement as needed)
func (s *Server) listNotifications(c *gin.Context) {
	c.JSON(200, gin.H{"notifications": []Notification{}})
}

func (s *Server) getNotification(c *gin.Context) {
	c.JSON(404, gin.H{"error": "Not implemented"})
}

func (s *Server) getNotificationStatus(c *gin.Context) {
	c.JSON(404, gin.H{"error": "Not implemented"})
}

func (s *Server) createContact(c *gin.Context) {
	c.JSON(404, gin.H{"error": "Not implemented"})
}

func (s *Server) listContacts(c *gin.Context) {
	c.JSON(200, gin.H{"contacts": []Contact{}})
}

func (s *Server) getContact(c *gin.Context) {
	c.JSON(404, gin.H{"error": "Not implemented"})
}

func (s *Server) updateContact(c *gin.Context) {
	c.JSON(404, gin.H{"error": "Not implemented"})
}

func (s *Server) updateContactPreferences(c *gin.Context) {
	c.JSON(404, gin.H{"error": "Not implemented"})
}

func (s *Server) createTemplate(c *gin.Context) {
	c.JSON(404, gin.H{"error": "Not implemented"})
}

func (s *Server) listTemplates(c *gin.Context) {
	c.JSON(200, gin.H{"templates": []Template{}})
}

func (s *Server) getTemplate(c *gin.Context) {
	c.JSON(404, gin.H{"error": "Not implemented"})
}

func (s *Server) updateTemplate(c *gin.Context) {
	c.JSON(404, gin.H{"error": "Not implemented"})
}

func (s *Server) getDeliveryStats(c *gin.Context) {
	c.JSON(200, gin.H{"stats": map[string]interface{}{}})
}

func (s *Server) getDailyStats(c *gin.Context) {
	c.JSON(200, gin.H{"daily_stats": []interface{}{}})
}

func (s *Server) handleUnsubscribe(c *gin.Context) {
	c.JSON(200, gin.H{"message": "Unsubscribe processed"})
}