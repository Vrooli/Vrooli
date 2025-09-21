package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

type Config struct {
	Port        string
	ChatPort    string
	PostgresURL string
	QdrantURL   string
	OllamaURL   string
	N8NBaseURL  string
	MinioURL    string
}

type Persona struct {
	ID                  string                 `json:"id" db:"id"`
	Name                string                 `json:"name" db:"name"`
	Description         string                 `json:"description" db:"description"`
	BaseModel           string                 `json:"base_model" db:"base_model"`
	FineTunedModelPath  *string                `json:"fine_tuned_model_path" db:"fine_tuned_model_path"`
	TrainingStatus      string                 `json:"training_status" db:"training_status"`
	PersonalityTraits   map[string]interface{} `json:"personality_traits" db:"personality_traits"`
	KnowledgeDomains    []string               `json:"knowledge_domains" db:"knowledge_domains"`
	ConversationStyle   map[string]interface{} `json:"conversation_style" db:"conversation_style"`
	DocumentCount       int                    `json:"document_count" db:"document_count"`
	TotalTokens         int                    `json:"total_tokens" db:"total_tokens"`
	LastTrained         *time.Time             `json:"last_trained" db:"last_trained"`
	CreatedAt           time.Time              `json:"created_at" db:"created_at"`
	LastUpdated         time.Time              `json:"last_updated" db:"last_updated"`
}

type DataSource struct {
	ID             string                 `json:"id" db:"id"`
	PersonaID      string                 `json:"persona_id" db:"persona_id"`
	SourceType     string                 `json:"source_type" db:"source_type"`
	SourceConfig   map[string]interface{} `json:"source_config" db:"source_config"`
	SyncFrequency  string                 `json:"sync_frequency" db:"sync_frequency"`
	LastSync       *time.Time             `json:"last_sync" db:"last_sync"`
	TotalDocuments int                    `json:"total_documents" db:"total_documents"`
	TotalSizeBytes int64                  `json:"total_size_bytes" db:"total_size_bytes"`
	Status         string                 `json:"status" db:"status"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
}

type TrainingJob struct {
	ID               string                 `json:"id" db:"id"`
	PersonaID        string                 `json:"persona_id" db:"persona_id"`
	JobType          string                 `json:"job_type" db:"job_type"`
	ModelName        string                 `json:"model_name" db:"model_name"`
	Technique        string                 `json:"technique" db:"technique"`
	TrainingConfig   map[string]interface{} `json:"training_config" db:"training_config"`
	DatasetPath      *string                `json:"dataset_path" db:"dataset_path"`
	CheckpointPath   *string                `json:"checkpoint_path" db:"checkpoint_path"`
	Metrics          map[string]interface{} `json:"metrics" db:"metrics"`
	Status           string                 `json:"status" db:"status"`
	StartedAt        *time.Time             `json:"started_at" db:"started_at"`
	CompletedAt      *time.Time             `json:"completed_at" db:"completed_at"`
	ErrorMessage     *string                `json:"error_message" db:"error_message"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
}

type APIToken struct {
	ID          string    `json:"id" db:"id"`
	PersonaID   string    `json:"persona_id" db:"persona_id"`
	TokenHash   string    `json:"token_hash" db:"token_hash"`
	Name        string    `json:"name" db:"name"`
	Permissions []string  `json:"permissions" db:"permissions"`
	LastUsed    *time.Time `json:"last_used" db:"last_used"`
	ExpiresAt   *time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type ChatRequest struct {
	PersonaID string `json:"persona_id" binding:"required"`
	Message   string `json:"message" binding:"required"`
	SessionID string `json:"session_id"`
}

type ChatResponse struct {
	Response    string    `json:"response"`
	SessionID   string    `json:"session_id"`
	PersonaName string    `json:"persona_name"`
	Timestamp   time.Time `json:"timestamp"`
	ContextUsed bool      `json:"context_used"`
}

type CreatePersonaRequest struct {
	Name              string                 `json:"name" binding:"required"`
	Description       string                 `json:"description"`
	PersonalityTraits map[string]interface{} `json:"personality_traits"`
}

type ConnectDataSourceRequest struct {
	PersonaID    string                 `json:"persona_id" binding:"required"`
	SourceType   string                 `json:"source_type" binding:"required"`
	SourceConfig map[string]interface{} `json:"source_config"`
}

type StartTrainingRequest struct {
	PersonaID string `json:"persona_id" binding:"required"`
	Model     string `json:"model" binding:"required"`
	Technique string `json:"technique" binding:"required"`
}

type CreateTokenRequest struct {
	PersonaID   string   `json:"persona_id" binding:"required"`
	Name        string   `json:"name" binding:"required"`
	Permissions []string `json:"permissions"`
}

type SearchRequest struct {
	PersonaID string `json:"persona_id" binding:"required"`
	Query     string `json:"query" binding:"required"`
	Limit     int    `json:"limit"`
}

type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Query   string         `json:"query"`
	Total   int            `json:"total"`
}

type SearchResult struct {
	ID        string    `json:"id"`
	Score     float64   `json:"score"`
	Text      string    `json:"text"`
	Source    string    `json:"source"`
	Timestamp time.Time `json:"timestamp"`
}

var db *sql.DB
var config Config

func loadConfig() Config {
	// All configuration is REQUIRED - no defaults
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		// Try to build from individual components
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")
		
		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Fatal("❌ Database configuration missing. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}
		
		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}
	
	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}
	
	chatPort := os.Getenv("CHAT_PORT")
	if chatPort == "" {
		log.Fatal("❌ CHAT_PORT environment variable is required")
	}
	
	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		log.Fatal("❌ QDRANT_URL environment variable is required")
	}
	
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		log.Fatal("❌ OLLAMA_URL environment variable is required")
	}
	
	n8nURL := os.Getenv("N8N_BASE_URL")
	if n8nURL == "" {
		log.Fatal("❌ N8N_BASE_URL environment variable is required")
	}
	
	minioURL := os.Getenv("MINIO_URL")
	if minioURL == "" {
		log.Fatal("❌ MINIO_URL environment variable is required")
	}
	
	return Config{
		Port:        port,
		ChatPort:    chatPort,
		PostgresURL: postgresURL,
		QdrantURL:   qdrantURL,
		OllamaURL:   ollamaURL,
		N8NBaseURL:  n8nURL,
		MinioURL:    minioURL,
	}
}

// getEnv removed to prevent hardcoded defaults

func initDatabase() error {
	var err error
	db, err = sql.Open("postgres", config.PostgresURL)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
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
		return fmt.Errorf("❌ Database connection failed after %d attempts: %w", maxRetries, pingErr)
	}
	
	log.Println("🎉 Database connection pool established successfully!")
	return nil
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start personal-digital-twin

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	config = loadConfig()

	if err := initDatabase(); err != nil {
		log.Fatal("Database initialization failed:", err)
	}
	defer db.Close()

	// Start both API servers
	go startChatServer()
	startMainServer()
}

func startMainServer() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})
	r.Use(func(ctx *gin.Context) {
		c.HandlerFunc(ctx.Writer, ctx.Request)
		ctx.Next()
	})

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "personal-digital-twin-api",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// Persona management endpoints
	r.POST("/api/persona/create", createPersona)
	r.GET("/api/persona/:id", getPersona)
	r.GET("/api/personas", listPersonas)

	// Data source endpoints
	r.POST("/api/datasource/connect", connectDataSource)
	r.GET("/api/datasources/:persona_id", getDataSources)

	// Search endpoints
	r.POST("/api/search", searchDocuments)

	// Training endpoints
	r.POST("/api/train/start", startTraining)
	r.GET("/api/training/jobs/:persona_id", getTrainingJobs)

	// Token management endpoints
	r.POST("/api/tokens/create", createAPIToken)
	r.GET("/api/tokens/:persona_id", getAPITokens)

	log.Printf("Main API server starting on port %s", config.Port)
	log.Fatal(r.Run(":" + config.Port))
}

func startChatServer() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})
	r.Use(func(ctx *gin.Context) {
		c.HandlerFunc(ctx.Writer, ctx.Request)
		ctx.Next()
	})

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "personal-digital-twin-chat",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// Chat endpoints
	r.POST("/api/chat", handleChat)
	r.GET("/api/chat/history/:session_id", getChatHistory)

	log.Printf("Chat API server starting on port %s", config.ChatPort)
	log.Fatal(r.Run(":" + config.ChatPort))
}

func createPersona(c *gin.Context) {
	var req CreatePersonaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	personaID := uuid.New().String()
	personalityTraitsJSON, _ := json.Marshal(req.PersonalityTraits)

	query := `
		INSERT INTO personas (id, name, description, personality_traits)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at`

	var createdAt time.Time
	err := db.QueryRow(query, personaID, req.Name, req.Description, personalityTraitsJSON).Scan(&createdAt)
	if err != nil {
		log.Printf("Error creating persona: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create persona"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         personaID,
		"name":       req.Name,
		"created_at": createdAt,
		"message":    "Persona created successfully",
	})
}

func getPersona(c *gin.Context) {
	personaID := c.Param("id")

	var persona Persona
	var personalityTraitsJSON, knowledgeDomainsJSON, conversationStyleJSON []byte

	query := `
		SELECT id, name, description, base_model, fine_tuned_model_path,
		       training_status, personality_traits, knowledge_domains,
		       conversation_style, document_count, total_tokens,
		       last_trained, created_at, last_updated
		FROM personas WHERE id = $1`

	err := db.QueryRow(query, personaID).Scan(
		&persona.ID, &persona.Name, &persona.Description, &persona.BaseModel,
		&persona.FineTunedModelPath, &persona.TrainingStatus,
		&personalityTraitsJSON, &knowledgeDomainsJSON, &conversationStyleJSON,
		&persona.DocumentCount, &persona.TotalTokens, &persona.LastTrained,
		&persona.CreatedAt, &persona.LastUpdated,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Persona not found"})
			return
		}
		log.Printf("Error fetching persona: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch persona"})
		return
	}

	// Parse JSON fields
	json.Unmarshal(personalityTraitsJSON, &persona.PersonalityTraits)
	json.Unmarshal(knowledgeDomainsJSON, &persona.KnowledgeDomains)
	json.Unmarshal(conversationStyleJSON, &persona.ConversationStyle)

	c.JSON(http.StatusOK, persona)
}

func listPersonas(c *gin.Context) {
	query := `
		SELECT id, name, description, base_model, training_status,
		       document_count, total_tokens, created_at, last_updated
		FROM personas ORDER BY created_at DESC`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Error listing personas: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list personas"})
		return
	}
	defer rows.Close()

	var personas []gin.H
	for rows.Next() {
		var p struct {
			ID             string    `db:"id"`
			Name           string    `db:"name"`
			Description    string    `db:"description"`
			BaseModel      string    `db:"base_model"`
			TrainingStatus string    `db:"training_status"`
			DocumentCount  int       `db:"document_count"`
			TotalTokens    int       `db:"total_tokens"`
			CreatedAt      time.Time `db:"created_at"`
			LastUpdated    time.Time `db:"last_updated"`
		}

		err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.BaseModel,
			&p.TrainingStatus, &p.DocumentCount, &p.TotalTokens,
			&p.CreatedAt, &p.LastUpdated)
		if err != nil {
			continue
		}

		personas = append(personas, gin.H{
			"id":              p.ID,
			"name":            p.Name,
			"description":     p.Description,
			"base_model":      p.BaseModel,
			"training_status": p.TrainingStatus,
			"document_count":  p.DocumentCount,
			"total_tokens":    p.TotalTokens,
			"created_at":      p.CreatedAt,
			"last_updated":    p.LastUpdated,
		})
	}

	c.JSON(http.StatusOK, gin.H{"personas": personas})
}

func connectDataSource(c *gin.Context) {
	var req ConnectDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sourceID := uuid.New().String()
	sourceConfigJSON, _ := json.Marshal(req.SourceConfig)

	query := `
		INSERT INTO data_sources (id, persona_id, source_type, source_config)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	var returnedID string
	err := db.QueryRow(query, sourceID, req.PersonaID, req.SourceType, sourceConfigJSON).Scan(&returnedID)
	if err != nil {
		log.Printf("Error connecting data source: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect data source"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"source_id": returnedID,
		"message":   "Data source connected successfully",
	})
}

func getDataSources(c *gin.Context) {
	personaID := c.Param("persona_id")

	query := `
		SELECT id, source_type, source_config, sync_frequency, last_sync,
		       total_documents, total_size_bytes, status, created_at
		FROM data_sources WHERE persona_id = $1 ORDER BY created_at DESC`

	rows, err := db.Query(query, personaID)
	if err != nil {
		log.Printf("Error fetching data sources: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch data sources"})
		return
	}
	defer rows.Close()

	var sources []DataSource
	for rows.Next() {
		var source DataSource
		var sourceConfigJSON []byte

		err := rows.Scan(&source.ID, &source.SourceType, &sourceConfigJSON,
			&source.SyncFrequency, &source.LastSync, &source.TotalDocuments,
			&source.TotalSizeBytes, &source.Status, &source.CreatedAt)
		if err != nil {
			continue
		}

		json.Unmarshal(sourceConfigJSON, &source.SourceConfig)
		source.PersonaID = personaID
		sources = append(sources, source)
	}

	c.JSON(http.StatusOK, gin.H{"data_sources": sources})
}

func searchDocuments(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Limit <= 0 {
		req.Limit = 10
	}

	// Mock implementation - in real scenario, this would integrate with Qdrant
	mockResults := []SearchResult{
		{
			ID:        uuid.New().String(),
			Score:     0.95,
			Text:      "This is a sample search result for query: " + req.Query,
			Source:    "mock_document.txt",
			Timestamp: time.Now(),
		},
	}

	response := SearchResponse{
		Results: mockResults,
		Query:   req.Query,
		Total:   len(mockResults),
	}

	c.JSON(http.StatusOK, response)
}

func startTraining(c *gin.Context) {
	var req StartTrainingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	jobID := uuid.New().String()

	query := `
		INSERT INTO training_jobs (id, persona_id, model_name, technique, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	var returnedID string
	err := db.QueryRow(query, jobID, req.PersonaID, req.Model, req.Technique, "queued").Scan(&returnedID)
	if err != nil {
		log.Printf("Error starting training: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start training"})
		return
	}

	// Trigger training via n8n webhook (async)
	go func() {
		payload := map[string]interface{}{
			"persona_id": req.PersonaID,
			"model":      req.Model,
			"technique":  req.Technique,
			"job_id":     returnedID,
		}
		// Here you would make HTTP request to n8n webhook
		log.Printf("Would trigger training webhook for job %s", returnedID)
		_ = payload
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"job_id":  returnedID,
		"status":  "queued",
		"message": "Training job started",
	})
}

func getTrainingJobs(c *gin.Context) {
	personaID := c.Param("persona_id")

	query := `
		SELECT id, job_type, model_name, technique, status, started_at,
		       completed_at, error_message, created_at
		FROM training_jobs WHERE persona_id = $1 ORDER BY created_at DESC`

	rows, err := db.Query(query, personaID)
	if err != nil {
		log.Printf("Error fetching training jobs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch training jobs"})
		return
	}
	defer rows.Close()

	var jobs []TrainingJob
	for rows.Next() {
		var job TrainingJob
		err := rows.Scan(&job.ID, &job.JobType, &job.ModelName, &job.Technique,
			&job.Status, &job.StartedAt, &job.CompletedAt, &job.ErrorMessage,
			&job.CreatedAt)
		if err != nil {
			continue
		}

		job.PersonaID = personaID
		jobs = append(jobs, job)
	}

	c.JSON(http.StatusOK, gin.H{"training_jobs": jobs})
}

func createAPIToken(c *gin.Context) {
	var req CreateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokenID := uuid.New().String()
	token := uuid.New().String() // In real implementation, use proper token generation
	tokenHash := token           // In real implementation, hash the token

	if req.Permissions == nil {
		req.Permissions = []string{"read"}
	}

	permissionsJSON, _ := json.Marshal(req.Permissions)

	query := `
		INSERT INTO api_tokens (id, persona_id, token_hash, name, permissions)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	var returnedID string
	err := db.QueryRow(query, tokenID, req.PersonaID, tokenHash, req.Name, permissionsJSON).Scan(&returnedID)
	if err != nil {
		log.Printf("Error creating API token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"token":       token,
		"token_id":    returnedID,
		"name":        req.Name,
		"permissions": req.Permissions,
		"message":     "API token created successfully",
	})
}

func getAPITokens(c *gin.Context) {
	personaID := c.Param("persona_id")

	query := `
		SELECT id, name, permissions, last_used, expires_at, created_at
		FROM api_tokens WHERE persona_id = $1 ORDER BY created_at DESC`

	rows, err := db.Query(query, personaID)
	if err != nil {
		log.Printf("Error fetching API tokens: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch API tokens"})
		return
	}
	defer rows.Close()

	var tokens []gin.H
	for rows.Next() {
		var token struct {
			ID        string     `db:"id"`
			Name      string     `db:"name"`
			LastUsed  *time.Time `db:"last_used"`
			ExpiresAt *time.Time `db:"expires_at"`
			CreatedAt time.Time  `db:"created_at"`
		}
		var permissionsJSON []byte

		err := rows.Scan(&token.ID, &token.Name, &permissionsJSON,
			&token.LastUsed, &token.ExpiresAt, &token.CreatedAt)
		if err != nil {
			continue
		}

		var permissions []string
		json.Unmarshal(permissionsJSON, &permissions)

		tokens = append(tokens, gin.H{
			"id":          token.ID,
			"name":        token.Name,
			"permissions": permissions,
			"last_used":   token.LastUsed,
			"expires_at":  token.ExpiresAt,
			"created_at":  token.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"tokens": tokens})
}

func handleChat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get persona information
	var persona Persona
	var personalityTraitsJSON []byte

	query := `SELECT id, name, description, personality_traits FROM personas WHERE id = $1`
	err := db.QueryRow(query, req.PersonaID).Scan(&persona.ID, &persona.Name,
		&persona.Description, &personalityTraitsJSON)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Persona not found"})
			return
		}
		log.Printf("Error fetching persona: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch persona"})
		return
	}

	json.Unmarshal(personalityTraitsJSON, &persona.PersonalityTraits)

	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	// Mock response generation - in real implementation, integrate with Ollama
	response := fmt.Sprintf("Hello! I'm %s. %s How can I help you today?",
		persona.Name, persona.Description)

	// Store conversation in database
	conversationData := map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"role":      "user",
				"content":   req.Message,
				"timestamp": time.Now().Format(time.RFC3339),
			},
			{
				"role":      "assistant",
				"content":   response,
				"timestamp": time.Now().Format(time.RFC3339),
			},
		},
	}

	conversationJSON, _ := json.Marshal(conversationData["messages"])

	// Insert or update conversation
	_, err = db.Exec(`
		INSERT INTO conversations (id, persona_id, session_id, messages, total_tokens_used)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (session_id, persona_id) DO UPDATE SET
		messages = $4, total_tokens_used = total_tokens_used + $5`,
		uuid.New().String(), req.PersonaID, sessionID, conversationJSON,
		len(req.Message)+len(response))

	if err != nil {
		log.Printf("Warning: Failed to store conversation: %v", err)
	}

	chatResponse := ChatResponse{
		Response:    response,
		SessionID:   sessionID,
		PersonaName: persona.Name,
		Timestamp:   time.Now(),
		ContextUsed: false, // Mock value
	}

	c.JSON(http.StatusOK, chatResponse)
}

func getChatHistory(c *gin.Context) {
	sessionID := c.Param("session_id")
	personaID := c.Query("persona_id")

	if personaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "persona_id query parameter is required"})
		return
	}

	query := `
		SELECT messages, created_at FROM conversations
		WHERE session_id = $1 AND persona_id = $2`

	var messagesJSON []byte
	var createdAt time.Time

	err := db.QueryRow(query, sessionID, personaID).Scan(&messagesJSON, &createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusOK, gin.H{
				"messages":   []interface{}{},
				"session_id": sessionID,
			})
			return
		}
		log.Printf("Error fetching chat history: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chat history"})
		return
	}

	var messages []interface{}
	json.Unmarshal(messagesJSON, &messages)

	c.JSON(http.StatusOK, gin.H{
		"messages":   messages,
		"session_id": sessionID,
		"created_at": createdAt,
	})
}