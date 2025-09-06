package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type Server struct {
	db     *pgxpool.Pool
	router *gin.Engine
}

type Funnel struct {
	ID          string          `json:"id"`
	TenantID    *string         `json:"tenant_id,omitempty"`
	Name        string          `json:"name"`
	Slug        string          `json:"slug"`
	Description string          `json:"description,omitempty"`
	Steps       []FunnelStep    `json:"steps"`
	Settings    json.RawMessage `json:"settings"`
	Status      string          `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type FunnelStep struct {
	ID             string          `json:"id"`
	FunnelID       string          `json:"funnel_id"`
	Type           string          `json:"type"`
	Position       int             `json:"position"`
	Title          string          `json:"title"`
	Content        json.RawMessage `json:"content"`
	BranchingRules json.RawMessage `json:"branching_rules,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type Lead struct {
	ID          string          `json:"id"`
	FunnelID    string          `json:"funnel_id"`
	TenantID    *string         `json:"tenant_id,omitempty"`
	SessionID   string          `json:"session_id"`
	Email       string          `json:"email,omitempty"`
	Phone       string          `json:"phone,omitempty"`
	Name        string          `json:"name,omitempty"`
	Data        json.RawMessage `json:"data"`
	Source      string          `json:"source,omitempty"`
	IPAddress   string          `json:"ip_address,omitempty"`
	UserAgent   string          `json:"user_agent,omitempty"`
	Completed   bool            `json:"completed"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type StepResponse struct {
	ID         string          `json:"id"`
	LeadID     string          `json:"lead_id"`
	StepID     string          `json:"step_id"`
	Response   json.RawMessage `json:"response"`
	DurationMs int             `json:"duration_ms,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

func NewServer() (*Server, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/vrooli?sslmode=disable"
	}

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 5

	db, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	s := &Server{
		db:     db,
		router: gin.Default(),
	}

	s.setupRoutes()
	return s, nil
}

func (s *Server) setupRoutes() {
	s.router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	api := s.router.Group("/api/v1")
	{
		api.GET("/health", s.handleHealth)
		
		// Funnel endpoints
		api.GET("/funnels", s.handleGetFunnels)
		api.GET("/funnels/:id", s.handleGetFunnel)
		api.POST("/funnels", s.handleCreateFunnel)
		api.PUT("/funnels/:id", s.handleUpdateFunnel)
		api.DELETE("/funnels/:id", s.handleDeleteFunnel)
		
		// Funnel execution endpoints
		api.GET("/funnels/:slug/execute", s.handleExecuteFunnel)
		api.POST("/funnels/:slug/submit", s.handleSubmitStep)
		
		// Analytics endpoints
		api.GET("/funnels/:id/analytics", s.handleGetAnalytics)
		api.GET("/funnels/:id/leads", s.handleGetLeads)
		
		// Template endpoints
		api.GET("/templates", s.handleGetTemplates)
		api.GET("/templates/:slug", s.handleGetTemplate)
	}
}

func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"time":   time.Now().Unix(),
	})
}

func (s *Server) handleGetFunnels(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	
	query := `
		SELECT id, tenant_id, name, slug, description, settings, status, created_at, updated_at
		FROM funnel_builder.funnels
		WHERE ($1::uuid IS NULL OR tenant_id = $1)
		ORDER BY created_at DESC
	`
	
	var tenantUUID *uuid.UUID
	if tenantID != "" {
		parsed, _ := uuid.Parse(tenantID)
		tenantUUID = &parsed
	}
	
	rows, err := s.db.Query(context.Background(), query, tenantUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	
	var funnels []Funnel
	for rows.Next() {
		var f Funnel
		err := rows.Scan(&f.ID, &f.TenantID, &f.Name, &f.Slug, &f.Description, &f.Settings, &f.Status, &f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			continue
		}
		
		// Load steps for each funnel
		f.Steps, _ = s.getFunnelSteps(f.ID)
		funnels = append(funnels, f)
	}
	
	c.JSON(http.StatusOK, funnels)
}

func (s *Server) handleGetFunnel(c *gin.Context) {
	id := c.Param("id")
	
	var f Funnel
	query := `
		SELECT id, tenant_id, name, slug, description, settings, status, created_at, updated_at
		FROM funnel_builder.funnels
		WHERE id = $1
	`
	
	err := s.db.QueryRow(context.Background(), query, id).Scan(
		&f.ID, &f.TenantID, &f.Name, &f.Slug, &f.Description, &f.Settings, &f.Status, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Funnel not found"})
		return
	}
	
	f.Steps, _ = s.getFunnelSteps(f.ID)
	
	c.JSON(http.StatusOK, f)
}

func (s *Server) handleCreateFunnel(c *gin.Context) {
	var input struct {
		Name        string          `json:"name" binding:"required"`
		Slug        string          `json:"slug"`
		Description string          `json:"description"`
		Steps       []FunnelStep    `json:"steps"`
		Settings    json.RawMessage `json:"settings"`
		TenantID    *string         `json:"tenant_id"`
	}
	
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if input.Slug == "" {
		input.Slug = generateSlug(input.Name)
	}
	
	if input.Settings == nil {
		input.Settings = json.RawMessage(`{"theme":{"primaryColor":"#0ea5e9"},"progressBar":true}`)
	}
	
	tx, err := s.db.Begin(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback(context.Background())
	
	var funnelID string
	query := `
		INSERT INTO funnel_builder.funnels (name, slug, description, settings, tenant_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	
	err = tx.QueryRow(context.Background(), query, input.Name, input.Slug, input.Description, input.Settings, input.TenantID).Scan(&funnelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	// Insert steps if provided
	for i, step := range input.Steps {
		stepQuery := `
			INSERT INTO funnel_builder.funnel_steps (funnel_id, type, position, title, content, branching_rules)
			VALUES ($1, $2, $3, $4, $5, $6)
		`
		_, err = tx.Exec(context.Background(), stepQuery, funnelID, step.Type, i, step.Title, step.Content, step.BranchingRules)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	
	if err = tx.Commit(context.Background()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"id":          funnelID,
		"slug":        input.Slug,
		"preview_url": fmt.Sprintf("/preview/%s", funnelID),
	})
}

func (s *Server) handleUpdateFunnel(c *gin.Context) {
	id := c.Param("id")
	
	var input struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Steps       []FunnelStep    `json:"steps"`
		Settings    json.RawMessage `json:"settings"`
		Status      string          `json:"status"`
	}
	
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	query := `
		UPDATE funnel_builder.funnels
		SET name = $2, description = $3, settings = $4, status = $5, updated_at = NOW()
		WHERE id = $1
	`
	
	_, err := s.db.Exec(context.Background(), query, id, input.Name, input.Description, input.Settings, input.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Funnel updated"})
}

func (s *Server) handleDeleteFunnel(c *gin.Context) {
	id := c.Param("id")
	
	query := `DELETE FROM funnel_builder.funnels WHERE id = $1`
	_, err := s.db.Exec(context.Background(), query, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Funnel deleted"})
}

func (s *Server) handleExecuteFunnel(c *gin.Context) {
	slug := c.Param("slug")
	sessionID := c.Query("session_id")
	
	if sessionID == "" {
		sessionID = uuid.New().String()
	}
	
	// Get funnel by slug
	var funnelID string
	var settings json.RawMessage
	query := `SELECT id, settings FROM funnel_builder.funnels WHERE slug = $1 AND status = 'active'`
	err := s.db.QueryRow(context.Background(), query, slug).Scan(&funnelID, &settings)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Funnel not found"})
		return
	}
	
	// Get or create lead
	var leadID string
	leadQuery := `SELECT id FROM funnel_builder.leads WHERE funnel_id = $1 AND session_id = $2`
	err = s.db.QueryRow(context.Background(), leadQuery, funnelID, sessionID).Scan(&leadID)
	if err != nil {
		// Create new lead
		insertQuery := `
			INSERT INTO funnel_builder.leads (funnel_id, session_id, ip_address, user_agent)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`
		err = s.db.QueryRow(context.Background(), insertQuery, funnelID, sessionID, c.ClientIP(), c.Request.UserAgent()).Scan(&leadID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		
		// Record analytics event
		s.recordEvent(funnelID, leadID, "start", nil)
	}
	
	// Get current step for lead
	currentStep := s.getCurrentStep(funnelID, leadID)
	
	// Record view event
	if currentStep != nil {
		stepID := currentStep["id"].(string)
		s.recordEvent(funnelID, leadID, "view", &stepID)
	}
	
	progress := s.calculateProgress(funnelID, leadID)
	
	c.JSON(http.StatusOK, gin.H{
		"step":       currentStep,
		"progress":   progress,
		"session_id": sessionID,
		"settings":   settings,
	})
}

func (s *Server) handleSubmitStep(c *gin.Context) {
	slug := c.Param("slug")
	
	var input struct {
		SessionID string          `json:"session_id" binding:"required"`
		StepID    string          `json:"step_id" binding:"required"`
		Response  json.RawMessage `json:"response" binding:"required"`
		Duration  int             `json:"duration_ms"`
	}
	
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Get funnel and lead
	var funnelID, leadID string
	query := `
		SELECT f.id, l.id
		FROM funnel_builder.funnels f
		JOIN funnel_builder.leads l ON f.id = l.funnel_id
		WHERE f.slug = $1 AND l.session_id = $2
	`
	err := s.db.QueryRow(context.Background(), query, slug, input.SessionID).Scan(&funnelID, &leadID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}
	
	// Record step response
	responseQuery := `
		INSERT INTO funnel_builder.step_responses (lead_id, step_id, response, duration_ms)
		VALUES ($1, $2, $3, $4)
	`
	_, err = s.db.Exec(context.Background(), responseQuery, leadID, input.StepID, input.Response, input.Duration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	// Update lead data
	s.updateLeadData(leadID, input.Response)
	
	// Record analytics event
	s.recordEvent(funnelID, leadID, "step_complete", &input.StepID)
	
	// Get next step
	nextStep := s.getNextStep(funnelID, input.StepID)
	if nextStep == nil {
		// Mark lead as completed
		completeQuery := `UPDATE funnel_builder.leads SET completed = true, completed_at = NOW() WHERE id = $1`
		s.db.Exec(context.Background(), completeQuery, leadID)
		s.recordEvent(funnelID, leadID, "complete", nil)
		
		c.JSON(http.StatusOK, gin.H{
			"completed": true,
			"message":   "Funnel completed",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"next_step": nextStep,
		"progress":  s.calculateProgress(funnelID, leadID),
	})
}

func (s *Server) handleGetAnalytics(c *gin.Context) {
	funnelID := c.Param("id")
	
	// Get overall metrics
	metricsQuery := `
		SELECT 
			COUNT(DISTINCT l.id) as total_leads,
			COUNT(DISTINCT CASE WHEN l.completed THEN l.id END) as completed_leads,
			AVG(CASE WHEN l.completed THEN EXTRACT(EPOCH FROM (l.completed_at - l.created_at)) END) as avg_completion_time
		FROM funnel_builder.leads l
		WHERE l.funnel_id = $1
	`
	
	var totalLeads, completedLeads int
	var avgTime sql.NullFloat64
	err := s.db.QueryRow(context.Background(), metricsQuery, funnelID).Scan(&totalLeads, &completedLeads, &avgTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	conversionRate := float64(0)
	if totalLeads > 0 {
		conversionRate = float64(completedLeads) / float64(totalLeads) * 100
	}
	
	// Get step drop-off data
	dropOffQuery := `
		SELECT fs.id, fs.title, fs.position,
			COUNT(DISTINCT sr.lead_id) as responses,
			AVG(sr.duration_ms) as avg_duration
		FROM funnel_builder.funnel_steps fs
		LEFT JOIN funnel_builder.step_responses sr ON fs.id = sr.step_id
		WHERE fs.funnel_id = $1
		GROUP BY fs.id, fs.title, fs.position
		ORDER BY fs.position
	`
	
	rows, _ := s.db.Query(context.Background(), dropOffQuery, funnelID)
	defer rows.Close()
	
	var dropOffPoints []map[string]interface{}
	for rows.Next() {
		var stepID, stepTitle string
		var position, responses int
		var avgDuration sql.NullFloat64
		
		rows.Scan(&stepID, &stepTitle, &position, &responses, &avgDuration)
		
		dropOffRate := float64(0)
		if totalLeads > 0 && responses < totalLeads {
			dropOffRate = float64(totalLeads-responses) / float64(totalLeads) * 100
		}
		
		dropOffPoints = append(dropOffPoints, map[string]interface{}{
			"stepId":      stepID,
			"stepTitle":   stepTitle,
			"dropOffRate": dropOffRate,
			"responses":   responses,
			"avgDuration": avgDuration.Float64,
		})
	}
	
	c.JSON(http.StatusOK, gin.H{
		"funnelId":       funnelID,
		"totalViews":     totalLeads,
		"totalLeads":     completedLeads,
		"conversionRate": conversionRate,
		"averageTime":    avgTime.Float64,
		"dropOffPoints":  dropOffPoints,
	})
}

func (s *Server) handleGetLeads(c *gin.Context) {
	funnelID := c.Param("id")
	format := c.Query("format")
	
	query := `
		SELECT id, email, phone, name, data, source, completed, created_at
		FROM funnel_builder.leads
		WHERE funnel_id = $1
		ORDER BY created_at DESC
	`
	
	rows, err := s.db.Query(context.Background(), query, funnelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	
	var leads []Lead
	for rows.Next() {
		var l Lead
		rows.Scan(&l.ID, &l.Email, &l.Phone, &l.Name, &l.Data, &l.Source, &l.Completed, &l.CreatedAt)
		leads = append(leads, l)
	}
	
	if format == "csv" {
		// Convert to CSV format
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=leads.csv")
		c.String(http.StatusOK, convertToCSV(leads))
		return
	}
	
	c.JSON(http.StatusOK, leads)
}

func (s *Server) handleGetTemplates(c *gin.Context) {
	query := `
		SELECT id, name, slug, description, category, template_data, metrics
		FROM funnel_builder.funnel_templates
		WHERE is_public = true
		ORDER BY name
	`
	
	rows, err := s.db.Query(context.Background(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	
	var templates []map[string]interface{}
	for rows.Next() {
		var id, name, slug, description, category string
		var templateData, metrics json.RawMessage
		
		rows.Scan(&id, &name, &slug, &description, &category, &templateData, &metrics)
		
		templates = append(templates, map[string]interface{}{
			"id":           id,
			"name":         name,
			"slug":         slug,
			"description":  description,
			"category":     category,
			"templateData": templateData,
			"metrics":      metrics,
		})
	}
	
	c.JSON(http.StatusOK, templates)
}

func (s *Server) handleGetTemplate(c *gin.Context) {
	slug := c.Param("slug")
	
	var template map[string]interface{}
	query := `
		SELECT id, name, slug, description, category, template_data, metrics
		FROM funnel_builder.funnel_templates
		WHERE slug = $1 AND is_public = true
	`
	
	var id, name, slugResult, description, category string
	var templateData, metrics json.RawMessage
	
	err := s.db.QueryRow(context.Background(), query, slug).Scan(&id, &name, &slugResult, &description, &category, &templateData, &metrics)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template not found"})
		return
	}
	
	template = map[string]interface{}{
		"id":           id,
		"name":         name,
		"slug":         slugResult,
		"description":  description,
		"category":     category,
		"templateData": templateData,
		"metrics":      metrics,
	}
	
	c.JSON(http.StatusOK, template)
}

// Helper functions
func (s *Server) getFunnelSteps(funnelID string) ([]FunnelStep, error) {
	query := `
		SELECT id, funnel_id, type, position, title, content, branching_rules, created_at, updated_at
		FROM funnel_builder.funnel_steps
		WHERE funnel_id = $1
		ORDER BY position
	`
	
	rows, err := s.db.Query(context.Background(), query, funnelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var steps []FunnelStep
	for rows.Next() {
		var step FunnelStep
		err := rows.Scan(&step.ID, &step.FunnelID, &step.Type, &step.Position, &step.Title, &step.Content, &step.BranchingRules, &step.CreatedAt, &step.UpdatedAt)
		if err != nil {
			continue
		}
		steps = append(steps, step)
	}
	
	return steps, nil
}

func (s *Server) getCurrentStep(funnelID, leadID string) map[string]interface{} {
	// Get the last completed step
	query := `
		SELECT MAX(fs.position)
		FROM funnel_builder.step_responses sr
		JOIN funnel_builder.funnel_steps fs ON sr.step_id = fs.id
		WHERE sr.lead_id = $1 AND fs.funnel_id = $2
	`
	
	var lastPosition sql.NullInt64
	s.db.QueryRow(context.Background(), query, leadID, funnelID).Scan(&lastPosition)
	
	nextPosition := 0
	if lastPosition.Valid {
		nextPosition = int(lastPosition.Int64) + 1
	}
	
	// Get the next step
	stepQuery := `
		SELECT id, type, position, title, content
		FROM funnel_builder.funnel_steps
		WHERE funnel_id = $1 AND position = $2
	`
	
	var stepID, stepType, title string
	var position int
	var content json.RawMessage
	
	err := s.db.QueryRow(context.Background(), stepQuery, funnelID, nextPosition).Scan(&stepID, &stepType, &position, &title, &content)
	if err != nil {
		return nil
	}
	
	return map[string]interface{}{
		"id":       stepID,
		"type":     stepType,
		"position": position,
		"title":    title,
		"content":  content,
	}
}

func (s *Server) getNextStep(funnelID, currentStepID string) map[string]interface{} {
	// Get current step position
	var currentPosition int
	query := `SELECT position FROM funnel_builder.funnel_steps WHERE id = $1`
	s.db.QueryRow(context.Background(), query, currentStepID).Scan(&currentPosition)
	
	// Get next step
	nextQuery := `
		SELECT id, type, position, title, content
		FROM funnel_builder.funnel_steps
		WHERE funnel_id = $1 AND position = $2
	`
	
	var stepID, stepType, title string
	var position int
	var content json.RawMessage
	
	err := s.db.QueryRow(context.Background(), nextQuery, funnelID, currentPosition+1).Scan(&stepID, &stepType, &position, &title, &content)
	if err != nil {
		return nil
	}
	
	return map[string]interface{}{
		"id":       stepID,
		"type":     stepType,
		"position": position,
		"title":    title,
		"content":  content,
	}
}

func (s *Server) calculateProgress(funnelID, leadID string) float64 {
	// Get total steps
	var totalSteps int
	s.db.QueryRow(context.Background(), 
		`SELECT COUNT(*) FROM funnel_builder.funnel_steps WHERE funnel_id = $1`, 
		funnelID).Scan(&totalSteps)
	
	// Get completed steps
	var completedSteps int
	s.db.QueryRow(context.Background(),
		`SELECT COUNT(DISTINCT sr.step_id) 
		FROM funnel_builder.step_responses sr
		JOIN funnel_builder.funnel_steps fs ON sr.step_id = fs.id
		WHERE sr.lead_id = $1 AND fs.funnel_id = $2`,
		leadID, funnelID).Scan(&completedSteps)
	
	if totalSteps == 0 {
		return 0
	}
	
	return float64(completedSteps) / float64(totalSteps) * 100
}

func (s *Server) updateLeadData(leadID string, response json.RawMessage) {
	// Parse response and update lead fields if they contain email, name, etc.
	var data map[string]interface{}
	if err := json.Unmarshal(response, &data); err != nil {
		return
	}
	
	email, _ := data["email"].(string)
	name, _ := data["name"].(string)
	phone, _ := data["phone"].(string)
	
	if email != "" || name != "" || phone != "" {
		query := `
			UPDATE funnel_builder.leads
			SET email = COALESCE(NULLIF($2, ''), email),
				name = COALESCE(NULLIF($3, ''), name),
				phone = COALESCE(NULLIF($4, ''), phone),
				data = data || $5,
				updated_at = NOW()
			WHERE id = $1
		`
		s.db.Exec(context.Background(), query, leadID, email, name, phone, response)
	}
}

func (s *Server) recordEvent(funnelID, leadID, eventType string, stepID *string) {
	query := `
		INSERT INTO funnel_builder.analytics_events (funnel_id, lead_id, event_type, step_id)
		VALUES ($1, $2, $3, $4)
	`
	s.db.Exec(context.Background(), query, funnelID, leadID, eventType, stepID)
}

func generateSlug(name string) string {
	// Simple slug generation - in production, use a proper slugify library
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}

func convertToCSV(leads []Lead) string {
	// Simple CSV conversion - in production, use a proper CSV library
	csv := "ID,Email,Name,Phone,Source,Completed,Created At\n"
	for _, lead := range leads {
		csv += fmt.Sprintf("%s,%s,%s,%s,%s,%t,%s\n",
			lead.ID, lead.Email, lead.Name, lead.Phone, lead.Source, lead.Completed, lead.CreatedAt.Format(time.RFC3339))
	}
	return csv
}

func main() {
	server, err := NewServer()
	if err != nil {
		log.Fatal("Failed to create server:", err)
	}
	defer server.db.Close()
	
	port := os.Getenv("FUNNEL_API_PORT")
	if port == "" {
		port = "15000"
	}
	
	log.Printf("Funnel Builder API starting on port %s", port)
	if err := server.router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}