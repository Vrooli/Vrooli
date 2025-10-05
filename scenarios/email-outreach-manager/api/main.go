package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

var db *sql.DB

// Campaign represents an email campaign
type Campaign struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	TemplateID      *string   `json:"template_id,omitempty"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	SendSchedule    *time.Time `json:"send_schedule,omitempty"`
	TotalRecipients int       `json:"total_recipients"`
	SentCount       int       `json:"sent_count"`
}

// Template represents an email template
type Template struct {
	ID                     string    `json:"id"`
	Name                   string    `json:"name"`
	Subject                string    `json:"subject"`
	HTMLContent            string    `json:"html_content"`
	TextContent            string    `json:"text_content"`
	PersonalizationFields  []string  `json:"personalization_fields"`
	StyleCategory          string    `json:"style_category"`
	CreatedAt              time.Time `json:"created_at"`
}

// CreateCampaignRequest represents campaign creation request
type CreateCampaignRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	TemplateID  string `json:"template_id"`
}

// GenerateTemplateRequest represents template generation request
type GenerateTemplateRequest struct {
	Purpose   string   `json:"purpose" binding:"required"`
	Tone      string   `json:"tone"`
	Documents []string `json:"documents"`
}

func main() {
	// Get port from environment or use default
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "15001"
	}

	// Initialize database connection
	if err := initDB(); err != nil {
		log.Printf("Warning: Database initialization failed: %v", err)
		// Continue without DB for testing scenarios
	}

	// Create Gin router
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", healthCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Campaign endpoints
		v1.GET("/campaigns", listCampaigns)
		v1.POST("/campaigns", createCampaign)
		v1.GET("/campaigns/:id", getCampaign)
		v1.POST("/campaigns/:id/send", sendCampaign)
		v1.GET("/campaigns/:id/analytics", getCampaignAnalytics)

		// Template endpoints
		v1.POST("/templates/generate", generateTemplate)
		v1.GET("/templates", listTemplates)
	}

	// Start server
	addr := fmt.Sprintf(":%s", port)
	log.Printf("Email Outreach Manager API starting on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func initDB() error {
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "email_outreach"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Database connection established")
	return nil
}

func healthCheck(c *gin.Context) {
	health := gin.H{
		"status": "healthy",
		"service": "email-outreach-manager",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Check database connection
	if db != nil {
		if err := db.Ping(); err != nil {
			health["database"] = "unhealthy"
			health["database_error"] = err.Error()
			c.JSON(http.StatusServiceUnavailable, health)
			return
		}
		health["database"] = "healthy"
	} else {
		health["database"] = "not_configured"
	}

	c.JSON(http.StatusOK, health)
}

func listCampaigns(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not configured"})
		return
	}

	rows, err := db.Query(`
		SELECT id, name, description, template_id, status, created_at,
		       send_schedule, total_recipients, sent_count
		FROM campaigns
		ORDER BY created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to query campaigns: %v", err)})
		return
	}
	defer rows.Close()

	var campaigns []Campaign
	for rows.Next() {
		var campaign Campaign
		err := rows.Scan(
			&campaign.ID, &campaign.Name, &campaign.Description,
			&campaign.TemplateID, &campaign.Status, &campaign.CreatedAt,
			&campaign.SendSchedule, &campaign.TotalRecipients, &campaign.SentCount,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to scan campaign: %v", err)})
			return
		}
		campaigns = append(campaigns, campaign)
	}

	if campaigns == nil {
		campaigns = []Campaign{}
	}

	c.JSON(http.StatusOK, campaigns)
}

func createCampaign(c *gin.Context) {
	var req CreateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not configured"})
		return
	}

	// Create new campaign
	campaign := Campaign{
		ID:              uuid.New().String(),
		Name:            req.Name,
		Description:     req.Description,
		Status:          "draft",
		CreatedAt:       time.Now(),
		TotalRecipients: 0,
		SentCount:       0,
	}

	if req.TemplateID != "" {
		campaign.TemplateID = &req.TemplateID
	}

	_, err := db.Exec(`
		INSERT INTO campaigns (id, name, description, template_id, status, created_at, total_recipients, sent_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, campaign.ID, campaign.Name, campaign.Description, campaign.TemplateID,
		campaign.Status, campaign.CreatedAt, campaign.TotalRecipients, campaign.SentCount)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create campaign: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, campaign)
}

func getCampaign(c *gin.Context) {
	id := c.Param("id")

	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not configured"})
		return
	}

	var campaign Campaign
	err := db.QueryRow(`
		SELECT id, name, description, template_id, status, created_at,
		       send_schedule, total_recipients, sent_count
		FROM campaigns
		WHERE id = $1
	`, id).Scan(
		&campaign.ID, &campaign.Name, &campaign.Description,
		&campaign.TemplateID, &campaign.Status, &campaign.CreatedAt,
		&campaign.SendSchedule, &campaign.TotalRecipients, &campaign.SentCount,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Campaign not found"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get campaign: %v", err)})
		return
	}

	c.JSON(http.StatusOK, campaign)
}

func sendCampaign(c *gin.Context) {
	id := c.Param("id")

	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not configured"})
		return
	}

	// Update campaign status to sending
	result, err := db.Exec(`
		UPDATE campaigns
		SET status = 'sending', send_schedule = $1
		WHERE id = $2 AND status = 'draft'
	`, time.Now(), id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to update campaign: %v", err)})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Campaign not found or not in draft status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"send_job_id":          uuid.New().String(),
		"status":               "queued",
		"estimated_completion": time.Now().Add(10 * time.Minute).Format(time.RFC3339),
	})
}

func getCampaignAnalytics(c *gin.Context) {
	id := c.Param("id")

	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not configured"})
		return
	}

	// Check if campaign exists
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM campaigns WHERE id = $1)", id).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to check campaign: %v", err)})
		return
	}

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Campaign not found"})
		return
	}

	// Return analytics (mock data for now)
	c.JSON(http.StatusOK, gin.H{
		"campaign_id": id,
		"metrics": gin.H{
			"sent":       0,
			"opened":     0,
			"clicked":    0,
			"bounced":    0,
			"open_rate":  0.0,
			"click_rate": 0.0,
		},
		"recipient_breakdown": gin.H{
			"full_personalization": gin.H{
				"count":     0,
				"open_rate": 0.0,
			},
			"partial_personalization": gin.H{
				"count":     0,
				"open_rate": 0.0,
			},
			"template_only": gin.H{
				"count":     0,
				"open_rate": 0.0,
			},
		},
	})
}

func generateTemplate(c *gin.Context) {
	var req GenerateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	// Validate tone
	validTones := map[string]bool{
		"professional": true,
		"friendly":     true,
		"casual":       true,
	}

	tone := req.Tone
	if tone == "" {
		tone = "professional"
	}

	if !validTones[tone] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tone. Must be professional, friendly, or casual"})
		return
	}

	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not configured"})
		return
	}

	// Create template (simplified - in real implementation would call Ollama)
	template := Template{
		ID:                    uuid.New().String(),
		Name:                  fmt.Sprintf("Template for %s", req.Purpose),
		Subject:               fmt.Sprintf("About %s", req.Purpose),
		HTMLContent:           fmt.Sprintf("<html><body><h1>%s</h1><p>This is a %s email about %s.</p></body></html>", req.Purpose, tone, req.Purpose),
		TextContent:           fmt.Sprintf("%s\n\nThis is a %s email about %s.", req.Purpose, tone, req.Purpose),
		PersonalizationFields: []string{"name", "email"},
		StyleCategory:         tone,
		CreatedAt:             time.Now(),
	}

	_, err := db.Exec(`
		INSERT INTO templates (id, name, subject, html_content, text_content, style_category, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, template.ID, template.Name, template.Subject, template.HTMLContent,
		template.TextContent, template.StyleCategory, template.CreatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create template: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, template)
}

func listTemplates(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database not configured"})
		return
	}

	rows, err := db.Query(`
		SELECT id, name, subject, html_content, text_content, style_category, created_at
		FROM templates
		ORDER BY created_at DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to query templates: %v", err)})
		return
	}
	defer rows.Close()

	var templates []Template
	for rows.Next() {
		var template Template
		err := rows.Scan(
			&template.ID, &template.Name, &template.Subject,
			&template.HTMLContent, &template.TextContent,
			&template.StyleCategory, &template.CreatedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to scan template: %v", err)})
			return
		}
		templates = append(templates, template)
	}

	if templates == nil {
		templates = []Template{}
	}

	c.JSON(http.StatusOK, templates)
}
