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
	ProjectID   *string         `json:"project_id,omitempty"`
	Name        string          `json:"name"`
	Slug        string          `json:"slug"`
	Description string          `json:"description,omitempty"`
	Steps       []FunnelStep    `json:"steps"`
	Settings    json.RawMessage `json:"settings"`
	Status      string          `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type Project struct {
	ID          string    `json:"id"`
	TenantID    *string   `json:"tenant_id,omitempty"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Funnels     []Funnel  `json:"funnels"`
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
			return nil, fmt.Errorf("❌ Missing database configuration. Provide DATABASE_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}

		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 5

	// Implement exponential backoff for database connection
	db, err := connectWithBackoff(config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	s := &Server{
		db:     db,
		router: gin.Default(),
	}

	// Configure trusted proxies for production security
	s.router.SetTrustedProxies(nil)

	s.setupRoutes()
	return s, nil
}

func connectWithBackoff(config *pgxpool.Config) (*pgxpool.Pool, error) {
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	log.Println("🔄 Attempting database connection with exponential backoff...")

	var db *pgxpool.Pool
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		db, err = pgxpool.NewWithConfig(context.Background(), config)
		if err == nil {
			// Test the connection
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			pingErr := db.Ping(ctx)
			cancel()

			if pingErr == nil {
				log.Printf("✅ Database connected successfully on attempt %d", attempt+1)
				return db, nil
			}

			err = pingErr
			db.Close()
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

		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt+1, maxRetries, err)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)

		// Provide detailed status every few attempts
		if attempt > 0 && attempt%3 == 0 {
			log.Printf("📈 Retry progress:")
			log.Printf("   - Attempts made: %d/%d", attempt+1, maxRetries)
			log.Printf("   - Total wait time: ~%v", time.Duration(attempt*2)*baseDelay)
			log.Printf("   - Current delay: %v (with jitter: %v)", delay, jitter)
		}

		time.Sleep(actualDelay)
	}

	return nil, fmt.Errorf("❌ Database connection failed after %d attempts: %w", maxRetries, err)
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

	// Health endpoint at root for lifecycle system
	s.router.GET("/health", s.handleHealth)

	api := s.router.Group("/api/v1")
	{
		api.GET("/health", s.handleHealth)

		// Project endpoints
		api.GET("/projects", s.handleGetProjects)
		api.POST("/projects", s.handleCreateProject)
		api.PUT("/projects/:id", s.handleUpdateProject)

		// Funnel endpoints
		api.GET("/funnels", s.handleGetFunnels)
		api.GET("/funnels/:id", s.handleGetFunnel)
		api.POST("/funnels", s.handleCreateFunnel)
		api.PUT("/funnels/:id", s.handleUpdateFunnel)
		api.DELETE("/funnels/:id", s.handleDeleteFunnel)

		// Funnel execution endpoints (use different path to avoid conflict)
		api.GET("/execute/:slug", s.handleExecuteFunnel)
		api.POST("/execute/:slug/submit", s.handleSubmitStep)

		// Analytics endpoints
		api.GET("/funnels/:id/analytics", s.handleGetAnalytics)
		api.GET("/funnels/:id/leads", s.handleGetLeads)

		// Template endpoints
		api.GET("/templates", s.handleGetTemplates)
		api.GET("/templates/:slug", s.handleGetTemplate)
	}
}

func (s *Server) handleHealth(c *gin.Context) {
	// Check database connectivity
	dbConnected := true
	var dbLatency float64
	dbStart := time.Now()
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	err := s.db.Ping(ctx)
	if err != nil {
		dbConnected = false
		dbLatency = 0
	} else {
		dbLatency = float64(time.Since(dbStart).Milliseconds())
	}

	response := gin.H{
		"status":    "healthy",
		"service":   "funnel-builder-api",
		"timestamp": time.Now().Format(time.RFC3339),
		"readiness": dbConnected,
		"version":   "1.0.0",
		"dependencies": gin.H{
			"database": gin.H{
				"connected":  dbConnected,
				"latency_ms": dbLatency,
				"error":      nil,
			},
		},
	}

	if !dbConnected {
		response["status"] = "degraded"
		response["dependencies"].(gin.H)["database"].(gin.H)["error"] = gin.H{
			"code":      "CONNECTION_FAILED",
			"message":   err.Error(),
			"category":  "resource",
			"retryable": true,
		}
	}

	c.JSON(http.StatusOK, response)
}

func (s *Server) handleGetProjects(c *gin.Context) {
	tenantID := c.Query("tenant_id")

	var tenantUUID *uuid.UUID
	if tenantID != "" {
		parsed, err := uuid.Parse(tenantID)
		if err == nil {
			tenantUUID = &parsed
		}
	}

	query := `
		SELECT id, tenant_id, name, COALESCE(description, ''), created_at, updated_at
		FROM funnel_builder.projects
		WHERE ($1::uuid IS NULL OR tenant_id = $1)
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(context.Background(), query, tenantUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			log.Printf("failed to scan project row: %v", err)
			continue
		}

		funnels, funnelErr := s.getProjectFunnels(&p.ID)
		if funnelErr != nil {
			log.Printf("failed to load funnels for project %s: %v", p.ID, funnelErr)
		}
		p.Funnels = funnels
		projects = append(projects, p)
	}

	c.JSON(http.StatusOK, projects)
}

func (s *Server) handleCreateProject(c *gin.Context) {
	var input struct {
		Name        string  `json:"name" binding:"required"`
		Description string  `json:"description"`
		TenantID    *string `json:"tenant_id"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	description := strings.TrimSpace(input.Description)

	query := `
		INSERT INTO funnel_builder.projects (name, description, tenant_id)
		VALUES ($1, NULLIF($2, ''), $3)
		RETURNING id, tenant_id, name, COALESCE(description, ''), created_at, updated_at
	`

	var project Project
	err := s.db.QueryRow(context.Background(), query, name, description, input.TenantID).Scan(
		&project.ID,
		&project.TenantID,
		&project.Name,
		&project.Description,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	project.Funnels = []Funnel{}

	c.JSON(http.StatusCreated, project)
}

func (s *Server) handleUpdateProject(c *gin.Context) {
	id := c.Param("id")

	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	name := strings.TrimSpace(input.Name)
	description := strings.TrimSpace(input.Description)

	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	query := `
		UPDATE funnel_builder.projects
		SET name = $2, description = NULLIF($3, ''), updated_at = NOW()
		WHERE id = $1
		RETURNING id, tenant_id, name, COALESCE(description, ''), created_at, updated_at
	`

	var project Project
	err := s.db.QueryRow(context.Background(), query, id, name, description).Scan(
		&project.ID,
		&project.TenantID,
		&project.Name,
		&project.Description,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	funnels, funnelErr := s.getProjectFunnels(&project.ID)
	if funnelErr != nil {
		log.Printf("failed to load funnels for project %s: %v", project.ID, funnelErr)
	}
	project.Funnels = funnels

	c.JSON(http.StatusOK, project)
}

func (s *Server) handleGetFunnels(c *gin.Context) {
	tenantID := c.Query("tenant_id")

	query := `
		SELECT id, tenant_id, project_id, name, slug, description, settings, status, created_at, updated_at
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
		err := rows.Scan(&f.ID, &f.TenantID, &f.ProjectID, &f.Name, &f.Slug, &f.Description, &f.Settings, &f.Status, &f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			continue
		}

		steps, stepErr := s.getFunnelSteps(f.ID)
		if stepErr != nil {
			log.Printf("failed to load steps for funnel %s: %v", f.ID, stepErr)
		}
		f.Steps = steps
		funnels = append(funnels, f)
	}

	c.JSON(http.StatusOK, funnels)
}

func (s *Server) handleGetFunnel(c *gin.Context) {
	id := c.Param("id")

	var f Funnel
	query := `
		SELECT id, tenant_id, project_id, name, slug, description, settings, status, created_at, updated_at
		FROM funnel_builder.funnels
		WHERE id = $1
	`

	err := s.db.QueryRow(context.Background(), query, id).Scan(
		&f.ID,
		&f.TenantID,
		&f.ProjectID,
		&f.Name,
		&f.Slug,
		&f.Description,
		&f.Settings,
		&f.Status,
		&f.CreatedAt,
		&f.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Funnel not found"})
		return
	}

	steps, stepErr := s.getFunnelSteps(f.ID)
	if stepErr != nil {
		log.Printf("failed to load steps for funnel %s: %v", f.ID, stepErr)
	}
	f.Steps = steps

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
		ProjectID   *string         `json:"project_id"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.ProjectID == nil || *input.ProjectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id is required"})
		return
	}

	projectID := strings.TrimSpace(*input.ProjectID)
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id is required"})
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
		INSERT INTO funnel_builder.funnels (name, slug, description, settings, tenant_id, project_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	err = tx.QueryRow(context.Background(), query, input.Name, input.Slug, input.Description, input.Settings, input.TenantID, projectID).Scan(&funnelID)
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
		ProjectID   *string         `json:"project_id"`
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

	args := []interface{}{id, input.Name, input.Description, input.Settings, input.Status}

	if input.ProjectID != nil {
		trimmed := strings.TrimSpace(*input.ProjectID)
		var project interface{}
		if trimmed == "" {
			project = nil
		} else {
			project = trimmed
		}
		query = `
			UPDATE funnel_builder.funnels
			SET name = $2, description = $3, settings = $4, status = $5, project_id = $6, updated_at = NOW()
			WHERE id = $1
		`
		args = append(args, project)
	}

	_, err := s.db.Exec(context.Background(), query, args...)
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
		clientIP := c.ClientIP()
		if clientIP == "" {
			clientIP = "127.0.0.1" // Default for tests and local development
		}

		insertQuery := `
			INSERT INTO funnel_builder.leads (funnel_id, session_id, ip_address, user_agent)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`
		err = s.db.QueryRow(context.Background(), insertQuery, funnelID, sessionID, clientIP, c.Request.UserAgent()).Scan(&leadID)
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
		result, err := s.db.Exec(context.Background(), completeQuery, leadID)
		if err != nil {
			log.Printf("ERROR: Failed to mark lead %s as completed: %v", leadID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete funnel"})
			return
		}
		rowsAffected := result.RowsAffected()
		if rowsAffected == 0 {
			log.Printf("WARNING: UPDATE affected 0 rows for lead %s", leadID)
		} else {
			log.Printf("Successfully marked lead %s as completed (%d rows)", leadID, rowsAffected)
		}
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

	ctx := context.Background()

	// Count total views based on recorded analytics events
	var totalViews int
	viewQuery := `
		SELECT COUNT(*)
		FROM funnel_builder.analytics_events
		WHERE funnel_id = $1 AND event_type IN ('view', 'start')
	`
	if err := s.db.QueryRow(ctx, viewQuery, funnelID).Scan(&totalViews); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get overall lead metrics
	metricsQuery := `
		SELECT 
			COUNT(DISTINCT l.id) as total_leads,
			COUNT(DISTINCT CASE WHEN l.completed THEN l.id END) as completed_leads,
			COUNT(DISTINCT CASE 
				WHEN COALESCE(NULLIF(l.email, ''), NULLIF(l.phone, ''), NULLIF(l.name, '')) IS NOT NULL
					OR (l.data IS NOT NULL AND l.data <> '{}'::jsonb)
				THEN l.id
				ELSE NULL
			END) as captured_leads,
			AVG(CASE WHEN l.completed THEN EXTRACT(EPOCH FROM (l.completed_at - l.created_at)) END) as avg_completion_time
		FROM funnel_builder.leads l
		WHERE l.funnel_id = $1
	`

	var totalLeads, completedLeads, capturedLeads int
	var avgTime sql.NullFloat64
	if err := s.db.QueryRow(ctx, metricsQuery, funnelID).Scan(&totalLeads, &completedLeads, &capturedLeads, &avgTime); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	conversionRate := float64(0)
	if totalLeads > 0 {
		conversionRate = float64(completedLeads) / float64(totalLeads) * 100
	}

	if totalViews < totalLeads {
		totalViews = totalLeads
	}

	captureRate := float64(0)
	if totalLeads > 0 {
		captureRate = float64(capturedLeads) / float64(totalLeads) * 100
	}

	// Get step drop-off data
	dropOffQuery := `
		SELECT fs.id, fs.title, fs.position,
			COUNT(DISTINCT sr.lead_id) as responses,
			COUNT(DISTINCT CASE WHEN ae.event_type IN ('view', 'start') THEN ae.lead_id END) as visitors,
			AVG(sr.duration_ms) as avg_duration
		FROM funnel_builder.funnel_steps fs
		LEFT JOIN funnel_builder.step_responses sr ON fs.id = sr.step_id
		LEFT JOIN funnel_builder.analytics_events ae ON fs.id = ae.step_id
		WHERE fs.funnel_id = $1
		GROUP BY fs.id, fs.title, fs.position
		ORDER BY fs.position
	`

	rows, err := s.db.Query(ctx, dropOffQuery, funnelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	dropOffPoints := make([]map[string]interface{}, 0)
	for rows.Next() {
		var stepID, stepTitle string
		var position, responses, visitors int
		var avgDuration sql.NullFloat64

		if err := rows.Scan(&stepID, &stepTitle, &position, &responses, &visitors, &avgDuration); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		dropOffRate := float64(0)
		if visitors > 0 && responses < visitors {
			dropOffRate = float64(visitors-responses) / float64(visitors) * 100
		}

		dropOffPoints = append(dropOffPoints, map[string]interface{}{
			"stepId":      stepID,
			"stepTitle":   stepTitle,
			"position":    position,
			"dropOffRate": dropOffRate,
			"responses":   responses,
			"visitors":    visitors,
			"avgDuration": avgDuration.Float64,
		})
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Build daily stats for the past two weeks
	dailyStatsQuery := `
		WITH date_series AS (
			SELECT generate_series(CURRENT_DATE - INTERVAL '13 days', CURRENT_DATE, INTERVAL '1 day')::date AS date
		), lead_counts AS (
			SELECT created_at::date AS day,
				COUNT(*) AS sessions,
				COUNT(*) FILTER (
					WHERE COALESCE(NULLIF(email, ''), NULLIF(phone, ''), NULLIF(name, '')) IS NOT NULL
						OR (data IS NOT NULL AND data <> '{}'::jsonb)
				) AS captured,
				COUNT(*) FILTER (WHERE completed AND completed_at IS NOT NULL) AS conversions
			FROM funnel_builder.leads
			WHERE funnel_id = $1
			GROUP BY created_at::date
		)
		SELECT d.date,
			COALESCE(l.sessions, 0) AS views,
			COALESCE(l.captured, 0) AS leads,
			COALESCE(l.conversions, 0) AS conversions
		FROM date_series d
		LEFT JOIN lead_counts l ON l.day = d.date
		ORDER BY d.date
	`

	dailyRows, err := s.db.Query(ctx, dailyStatsQuery, funnelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer dailyRows.Close()

	dailyStats := make([]map[string]interface{}, 0)
	for dailyRows.Next() {
		var day time.Time
		var views, leads, conversions int

		if err := dailyRows.Scan(&day, &views, &leads, &conversions); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		dailyStats = append(dailyStats, map[string]interface{}{
			"date":        day.Format("2006-01-02"),
			"views":       views,
			"leads":       leads,
			"conversions": conversions,
		})
	}

	if err := dailyRows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	trafficQuery := `
		SELECT COALESCE(NULLIF(l.source, ''), 'Direct') as source, COUNT(*) as count
		FROM funnel_builder.leads l
		WHERE l.funnel_id = $1
		GROUP BY source
		ORDER BY count DESC
	`

	trafficRows, err := s.db.Query(ctx, trafficQuery, funnelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer trafficRows.Close()

	trafficSources := make([]map[string]interface{}, 0)
	for trafficRows.Next() {
		var source string
		var count int

		if err := trafficRows.Scan(&source, &count); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		percentage := float64(0)
		if totalLeads > 0 {
			percentage = float64(count) / float64(totalLeads) * 100
		}

		trafficSources = append(trafficSources, map[string]interface{}{
			"source":     source,
			"count":      count,
			"percentage": percentage,
		})
	}

	if err := trafficRows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	averageTimeSeconds := float64(0)
	if avgTime.Valid {
		averageTimeSeconds = avgTime.Float64
	}

	c.JSON(http.StatusOK, gin.H{
		"funnelId":       funnelID,
		"totalViews":     totalViews,
		"totalLeads":     totalLeads,
		"completedLeads": completedLeads,
		"capturedLeads":  capturedLeads,
		"conversionRate": conversionRate,
		"captureRate":    captureRate,
		"averageTime":    averageTimeSeconds,
		"dropOffPoints":  dropOffPoints,
		"dailyStats":     dailyStats,
		"trafficSources": trafficSources,
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
		var email, phone, name, source sql.NullString
		var data []byte

		if err := rows.Scan(&l.ID, &email, &phone, &name, &data, &source, &l.Completed, &l.CreatedAt); err != nil {
			log.Printf("Error scanning lead row: %v", err)
			continue
		}

		// Map nullable fields
		if email.Valid {
			l.Email = email.String
		}
		if phone.Valid {
			l.Phone = phone.String
		}
		if name.Valid {
			l.Name = name.String
		}
		if source.Valid {
			l.Source = source.String
		}
		if len(data) > 0 {
			l.Data = json.RawMessage(data)
		}

		leads = append(leads, l)
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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

func (s *Server) getProjectFunnels(projectID *string) ([]Funnel, error) {
	var (
		query string
		args  []interface{}
	)

	if projectID == nil {
		query = `
			SELECT id, tenant_id, project_id, name, slug, description, settings, status, created_at, updated_at
			FROM funnel_builder.funnels
			WHERE project_id IS NULL
			ORDER BY created_at DESC
		`
	} else {
		query = `
			SELECT id, tenant_id, project_id, name, slug, description, settings, status, created_at, updated_at
			FROM funnel_builder.funnels
			WHERE project_id = $1
			ORDER BY created_at DESC
		`
		args = append(args, *projectID)
	}

	rows, err := s.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var funnels []Funnel
	for rows.Next() {
		var f Funnel
		err := rows.Scan(&f.ID, &f.TenantID, &f.ProjectID, &f.Name, &f.Slug, &f.Description, &f.Settings, &f.Status, &f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			log.Printf("failed to scan funnel for project: %v", err)
			continue
		}

		steps, stepErr := s.getFunnelSteps(f.ID)
		if stepErr != nil {
			log.Printf("failed to load steps for funnel %s: %v", f.ID, stepErr)
		}
		f.Steps = steps
		funnels = append(funnels, f)
	}

	if funnels == nil {
		return []Funnel{}, nil
	}

	return funnels, nil
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

	if steps == nil {
		return []FunnelStep{}, nil
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
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start funnel-builder

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

	server, err := NewServer()
	if err != nil {
		log.Fatal("Failed to create server:", err)
	}
	defer server.db.Close()

	log.Printf("Funnel Builder API starting on port %s", port)
	if err := server.router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
