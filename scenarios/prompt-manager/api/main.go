package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Domain models
type Campaign struct {
	ID          string     `json:"id" db:"id"`
	Name        string     `json:"name" db:"name"`
	Description *string    `json:"description" db:"description"`
	Color       string     `json:"color" db:"color"`
	Icon        string     `json:"icon" db:"icon"`
	ParentID    *string    `json:"parent_id" db:"parent_id"`
	SortOrder   int        `json:"sort_order" db:"sort_order"`
	IsFavorite  bool       `json:"is_favorite" db:"is_favorite"`
	PromptCount int        `json:"prompt_count" db:"prompt_count"`
	LastUsed    *time.Time `json:"last_used" db:"last_used"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

type Prompt struct {
	ID                  string     `json:"id" db:"id"`
	CampaignID          string     `json:"campaign_id" db:"campaign_id"`
	Title               string     `json:"title" db:"title"`
	Content             string     `json:"content" db:"content"`
	Description         *string    `json:"description" db:"description"`
	Variables           []string   `json:"variables" db:"variables"`
	UsageCount          int        `json:"usage_count" db:"usage_count"`
	LastUsed            *time.Time `json:"last_used" db:"last_used"`
	IsFavorite          bool       `json:"is_favorite" db:"is_favorite"`
	IsArchived          bool       `json:"is_archived" db:"is_archived"`
	QuickAccessKey      *string    `json:"quick_access_key" db:"quick_access_key"`
	Version             int        `json:"version" db:"version"`
	ParentVersionID     *string    `json:"parent_version_id" db:"parent_version_id"`
	WordCount           *int       `json:"word_count" db:"word_count"`
	EstimatedTokens     *int       `json:"estimated_tokens" db:"estimated_tokens"`
	EffectivenessRating *int       `json:"effectiveness_rating" db:"effectiveness_rating"`
	Notes               *string    `json:"notes" db:"notes"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at" db:"updated_at"`
	// Joined fields
	CampaignName *string  `json:"campaign_name,omitempty"`
	Tags         []string `json:"tags,omitempty"`
}

type Tag struct {
	ID          string  `json:"id" db:"id"`
	Name        string  `json:"name" db:"name"`
	Color       *string `json:"color" db:"color"`
	Description *string `json:"description" db:"description"`
}

type Template struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description" db:"description"`
	Content     string    `json:"content" db:"content"`
	Variables   []string  `json:"variables" db:"variables"`
	Category    *string   `json:"category" db:"category"`
	UsageCount  int       `json:"usage_count" db:"usage_count"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type TestResult struct {
	ID           string    `json:"id" db:"id"`
	PromptID     string    `json:"prompt_id" db:"prompt_id"`
	Model        string    `json:"model" db:"model"`
	InputVars    *string   `json:"input_variables" db:"input_variables"`
	Response     *string   `json:"response" db:"response"`
	ResponseTime *float64  `json:"response_time" db:"response_time"`
	TokenCount   *int      `json:"token_count" db:"token_count"`
	Rating       *int      `json:"rating" db:"rating"`
	Notes        *string   `json:"notes" db:"notes"`
	TestedAt     time.Time `json:"tested_at" db:"tested_at"`
}

type PromptVersion struct {
	ID            string     `json:"id" db:"id"`
	PromptID      string     `json:"prompt_id" db:"prompt_id"`
	VersionNumber int        `json:"version_number" db:"version_number"`
	FilePath      string     `json:"file_path" db:"file_path"`
	ContentCache  *string    `json:"content_cache" db:"content_cache"`
	Variables     []string   `json:"variables" db:"variables"`
	ChangeSummary *string    `json:"change_summary" db:"change_summary"`
	CreatedBy     *string    `json:"created_by" db:"created_by"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
}

// Request/Response types
type CreatePromptRequest struct {
	CampaignID          string   `json:"campaign_id"`
	Title               string   `json:"title"`
	Content             string   `json:"content"`
	Description         *string  `json:"description"`
	Variables           []string `json:"variables"`
	QuickAccessKey      *string  `json:"quick_access_key"`
	EffectivenessRating *int     `json:"effectiveness_rating"`
	Notes               *string  `json:"notes"`
	Tags                []string `json:"tags"`
}

type UpdatePromptRequest struct {
	Title               *string  `json:"title"`
	Content             *string  `json:"content"`
	Description         *string  `json:"description"`
	Variables           []string `json:"variables"`
	IsFavorite          *bool    `json:"is_favorite"`
	IsArchived          *bool    `json:"is_archived"`
	QuickAccessKey      *string  `json:"quick_access_key"`
	EffectivenessRating *int     `json:"effectiveness_rating"`
	Notes               *string  `json:"notes"`
	Tags                []string `json:"tags"`
	ChangeSummary       *string  `json:"change_summary"`
}

type CreateCampaignRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Color       *string `json:"color"`
	Icon        *string `json:"icon"`
	ParentID    *string `json:"parent_id"`
	IsFavorite  *bool   `json:"is_favorite"`
}

type TestPromptRequest struct {
	Model       string            `json:"model"`
	Variables   map[string]string `json:"variables"`
	MaxTokens   *int              `json:"max_tokens"`
	Temperature *float64          `json:"temperature"`
}

// API Server
type APIServer struct {
	db        *sql.DB
	qdrantURL string
	ollamaURL string
}

func main() {
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start prompt-manager

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Port configuration - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		log.Fatal("❌ API_PORT or PORT environment variable is required")
	}

	// Database configuration - support both POSTGRES_URL and individual components
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		// Try to build from individual components - REQUIRED, no defaults
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")

		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Fatal("❌ Database configuration missing. Provide POSTGRES_URL or all required database connection parameters")
		}

		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	// Optional resource URLs - will be empty if not available
	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		log.Println("⚠️  QDRANT_URL not provided - semantic search will be disabled")
	}

	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		log.Println("⚠️  OLLAMA_URL not provided - prompt testing will be disabled")
	}

	// Connect to database
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Set search path to use public schema (tables are in public, not prompt_mgr)
	_, err = db.Exec("SET search_path TO public")
	if err != nil {
		log.Fatal("Failed to set search_path:", err)
	}

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Printf("📝 Database URL configured")

	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			log.Printf("✅ Database connected successfully on attempt %d", attempt+1)
			break
		}

		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay)*math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))

		// Add random jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * rand.Float64())
		actualDelay := delay + jitter

		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt+1, maxRetries, pingErr)
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

	if pingErr != nil {
		log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}

	log.Println("🎉 Database connection pool established successfully!")

	server := &APIServer{
		db:        db,
		qdrantURL: qdrantURL,
		ollamaURL: ollamaURL,
	}

	router := mux.NewRouter()

	// CORS middleware
	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)

	// Health check (outside versioning for simplicity)
	router.HandleFunc("/health", server.healthCheck).Methods("GET")

	// API v1 routes
	v1 := router.PathPrefix("/api/v1").Subrouter()

	// Campaign endpoints
	v1.HandleFunc("/campaigns", server.getCampaigns).Methods("GET")
	v1.HandleFunc("/campaigns", server.createCampaign).Methods("POST")
	v1.HandleFunc("/campaigns/{id}", server.getCampaign).Methods("GET")
	v1.HandleFunc("/campaigns/{id}", server.updateCampaign).Methods("PUT")
	v1.HandleFunc("/campaigns/{id}", server.deleteCampaign).Methods("DELETE")
	v1.HandleFunc("/campaigns/{id}/prompts", server.getCampaignPrompts).Methods("GET")

	// Prompt endpoints
	v1.HandleFunc("/prompts", server.getPrompts).Methods("GET")
	v1.HandleFunc("/prompts", server.createPrompt).Methods("POST")
	v1.HandleFunc("/prompts/{id}", server.getPrompt).Methods("GET")
	v1.HandleFunc("/prompts/{id}", server.updatePrompt).Methods("PUT")
	v1.HandleFunc("/prompts/{id}", server.deletePrompt).Methods("DELETE")
	v1.HandleFunc("/prompts/{id}/use", server.recordPromptUsage).Methods("POST")

	// Search endpoints
	v1.HandleFunc("/search/prompts", server.searchPrompts).Methods("GET")
	v1.HandleFunc("/prompts/semantic", server.semanticSearch).Methods("POST")

	// Quick access
	v1.HandleFunc("/prompts/quick/{key}", server.getPromptByQuickKey).Methods("GET")
	v1.HandleFunc("/prompts/recent", server.getRecentPrompts).Methods("GET")
	v1.HandleFunc("/prompts/favorites", server.getFavoritePrompts).Methods("GET")

	// Testing
	v1.HandleFunc("/prompts/{id}/test", server.testPrompt).Methods("POST")
	v1.HandleFunc("/prompts/{id}/test-history", server.getTestHistory).Methods("GET")

	// Tags
	v1.HandleFunc("/tags", server.getTags).Methods("GET")
	v1.HandleFunc("/tags", server.createTag).Methods("POST")

	// Templates
	v1.HandleFunc("/templates", server.getTemplates).Methods("GET")
	v1.HandleFunc("/templates/{id}", server.getTemplate).Methods("GET")

	// Export/Import endpoints
	v1.HandleFunc("/export", server.exportData).Methods("GET")
	v1.HandleFunc("/import", server.importData).Methods("POST")

	// Prompt Version History
	v1.HandleFunc("/prompts/{id}/versions", server.getPromptVersions).Methods("GET")
	v1.HandleFunc("/prompts/{id}/revert/{version}", server.revertPromptVersion).Methods("POST")

	log.Printf("🚀 Prompt Manager API starting on port %s", port)
	log.Printf("🗄️  Database: Connected")
	if qdrantURL != "" {
		log.Printf("🔍 Qdrant: %s", qdrantURL)
	} else {
		log.Printf("🔍 Qdrant: Not available (semantic search disabled)")
	}
	if ollamaURL != "" {
		log.Printf("🧠 Ollama: %s", ollamaURL)
	} else {
		log.Printf("🧠 Ollama: Not available (prompt testing disabled)")
	}

	handler := corsHandler(router)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

// Health check endpoint
func (s *APIServer) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check database connectivity
	dbHealthy := s.checkDatabase()
	overallStatus := "healthy"
	if dbHealthy != "healthy" {
		overallStatus = "degraded"
	}

	status := map[string]interface{}{
		"status":    overallStatus,
		"service":   "prompt-manager-api",
		"timestamp": time.Now().Format(time.RFC3339),
		"readiness": dbHealthy == "healthy", // Ready only if database is healthy
		"dependencies": map[string]interface{}{
			"database": map[string]interface{}{
				"connected": dbHealthy == "healthy",
				"error":     nil,
			},
		},
		// Legacy field for backward compatibility
		"services": map[string]interface{}{
			"database": dbHealthy,
			"qdrant":   s.checkQdrant(),
			"ollama":   s.checkOllama(),
		},
	}

	json.NewEncoder(w).Encode(status)
}

func (s *APIServer) checkDatabase() string {
	if err := s.db.Ping(); err != nil {
		return "unhealthy"
	}
	return "healthy"
}

func (s *APIServer) checkQdrant() string {
	if s.qdrantURL == "" {
		return "not_configured"
	}
	resp, err := http.Get(s.qdrantURL + "/health")
	if err != nil {
		return "unavailable"
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) // Drain body to allow connection reuse

	if resp.StatusCode != http.StatusOK {
		return "unavailable"
	}
	return "healthy"
}

func (s *APIServer) checkOllama() string {
	if s.ollamaURL == "" {
		return "not_configured"
	}
	resp, err := http.Get(s.ollamaURL + "/api/tags")
	if err != nil {
		return "unavailable"
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) // Drain body to allow connection reuse

	if resp.StatusCode != http.StatusOK {
		return "unavailable"
	}
	return "healthy"
}

// Campaign endpoints
func (s *APIServer) getCampaigns(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, name, description, color, parent_id, sort_order, 
		       is_favorite, prompt_count, last_used, created_at, updated_at
		FROM campaigns 
		ORDER BY sort_order, name`

	rows, err := s.db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var campaigns []Campaign
	for rows.Next() {
		var campaign Campaign
		campaign.Icon = "folder" // Default icon
		err := rows.Scan(
			&campaign.ID, &campaign.Name, &campaign.Description, &campaign.Color,
			&campaign.ParentID, &campaign.SortOrder,
			&campaign.IsFavorite, &campaign.PromptCount, &campaign.LastUsed,
			&campaign.CreatedAt, &campaign.UpdatedAt,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		campaigns = append(campaigns, campaign)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(campaigns)
}

func (s *APIServer) createCampaign(w http.ResponseWriter, r *http.Request) {
	var req CreateCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	campaign := Campaign{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Color:       "#6366f1", // Default color
		Icon:        "folder",  // Default icon
		SortOrder:   0,
		IsFavorite:  false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if req.Color != nil {
		campaign.Color = *req.Color
	}
	if req.Icon != nil {
		campaign.Icon = *req.Icon
	}
	if req.IsFavorite != nil {
		campaign.IsFavorite = *req.IsFavorite
	}

	// Get next sort order
	var maxOrder int
	s.db.QueryRow("SELECT COALESCE(MAX(sort_order), 0) FROM campaigns").Scan(&maxOrder)
	campaign.SortOrder = maxOrder + 1

	query := `
		INSERT INTO campaigns (id, name, description, color, parent_id, 
		                      sort_order, is_favorite, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING prompt_count, last_used`

	err := s.db.QueryRow(query,
		campaign.ID, campaign.Name, campaign.Description, campaign.Color,
		req.ParentID, campaign.SortOrder, campaign.IsFavorite,
		campaign.CreatedAt, campaign.UpdatedAt,
	).Scan(&campaign.PromptCount, &campaign.LastUsed)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	campaign.ParentID = req.ParentID

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(campaign)
}

func (s *APIServer) getCampaign(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID := vars["id"]

	query := `
		SELECT id, name, description, color, parent_id, sort_order,
		       is_favorite, prompt_count, last_used, created_at, updated_at
		FROM campaigns 
		WHERE id = $1`

	var campaign Campaign
	campaign.Icon = "folder" // Default icon
	err := s.db.QueryRow(query, campaignID).Scan(
		&campaign.ID, &campaign.Name, &campaign.Description, &campaign.Color,
		&campaign.ParentID, &campaign.SortOrder,
		&campaign.IsFavorite, &campaign.PromptCount, &campaign.LastUsed,
		&campaign.CreatedAt, &campaign.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Campaign not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(campaign)
}

// Prompt endpoints
func (s *APIServer) getPrompts(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	archived := r.URL.Query().Get("archived") == "true"
	favorites := r.URL.Query().Get("favorites") == "true"

	query := `
		SELECT p.id, p.campaign_id, p.title, p.content_cache, p.description,
		       p.variables, p.usage_count, p.last_used, p.is_favorite,
		       p.is_archived, p.quick_access_key, p.version, p.parent_version_id,
		       p.word_count, p.estimated_tokens, p.effectiveness_rating,
		       p.notes, p.created_at, p.updated_at, c.name as campaign_name
		FROM prompts p
		LEFT JOIN campaigns c ON p.campaign_id = c.id
		WHERE p.is_archived = $1`

	args := []interface{}{archived}
	if favorites {
		query += " AND p.is_favorite = true"
	}

	query += " ORDER BY p.last_used DESC NULLS LAST, p.created_at DESC LIMIT $2"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	prompts := make([]Prompt, 0)
	for rows.Next() {
		prompt, err := s.scanPrompt(rows)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		prompts = append(prompts, prompt)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prompts)
}

func (s *APIServer) createPrompt(w http.ResponseWriter, r *http.Request) {
	var req CreatePromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	prompt := Prompt{
		ID:                  uuid.New().String(),
		CampaignID:          req.CampaignID,
		Title:               req.Title,
		Content:             req.Content,
		Description:         req.Description,
		Variables:           req.Variables,
		Version:             1,
		WordCount:           calculateWordCount(&req.Content),
		EstimatedTokens:     calculateTokenCount(&req.Content),
		EffectivenessRating: req.EffectivenessRating,
		Notes:               req.Notes,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	if req.QuickAccessKey != nil && *req.QuickAccessKey != "" {
		prompt.QuickAccessKey = req.QuickAccessKey
	}

	variablesJSON, _ := json.Marshal(prompt.Variables)

	query := `
		INSERT INTO prompts (id, campaign_id, title, content, description, variables,
		                    quick_access_key, word_count, estimated_tokens, 
		                    effectiveness_rating, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING usage_count, last_used, is_favorite, is_archived, version, parent_version_id`

	err := s.db.QueryRow(query,
		prompt.ID, prompt.CampaignID, prompt.Title, prompt.Content,
		prompt.Description, variablesJSON, prompt.QuickAccessKey,
		prompt.WordCount, prompt.EstimatedTokens, prompt.EffectivenessRating,
		prompt.Notes, prompt.CreatedAt, prompt.UpdatedAt,
	).Scan(&prompt.UsageCount, &prompt.LastUsed, &prompt.IsFavorite,
		&prompt.IsArchived, &prompt.Version, &prompt.ParentVersionID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add tags if provided
	if len(req.Tags) > 0 {
		s.addPromptTags(prompt.ID, req.Tags)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(prompt)
}

func (s *APIServer) getPrompt(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	promptID := vars["id"]

	query := `
		SELECT p.id, p.campaign_id, p.title, p.content, p.description, 
		       p.variables, p.usage_count, p.last_used, p.is_favorite, 
		       p.is_archived, p.quick_access_key, p.version, p.parent_version_id,
		       p.word_count, p.estimated_tokens, p.effectiveness_rating, 
		       p.notes, p.created_at, p.updated_at, c.name as campaign_name
		FROM prompts p
		LEFT JOIN campaigns c ON p.campaign_id = c.id
		WHERE p.id = $1`

	row := s.db.QueryRow(query, promptID)
	prompt, err := s.scanPrompt(row)

	if err == sql.ErrNoRows {
		http.Error(w, "Prompt not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get tags
	prompt.Tags = s.getPromptTags(promptID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prompt)
}

// Search endpoints
func (s *APIServer) searchPrompts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Search query required", http.StatusBadRequest)
		return
	}

	// Use PostgreSQL full-text search
	sqlQuery := `
		SELECT p.id, p.campaign_id, p.title, p.content_cache, p.description,
		       p.variables, p.usage_count, p.last_used, p.is_favorite,
		       p.is_archived, p.quick_access_key, p.version, p.parent_version_id,
		       p.word_count, p.estimated_tokens, p.effectiveness_rating,
		       p.notes, p.created_at, p.updated_at, c.name as campaign_name
		FROM prompts p
		LEFT JOIN campaigns c ON p.campaign_id = c.id
		WHERE p.is_archived = false AND (
			to_tsvector('english', p.title) @@ plainto_tsquery('english', $1) OR
			to_tsvector('english', p.content_cache) @@ plainto_tsquery('english', $1) OR
			to_tsvector('english', COALESCE(p.description, '')) @@ plainto_tsquery('english', $1)
		)
		ORDER BY
			ts_rank(to_tsvector('english', p.title), plainto_tsquery('english', $1)) DESC,
			p.usage_count DESC,
			p.created_at DESC
		LIMIT 20`

	rows, err := s.db.Query(sqlQuery, query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var prompts []Prompt
	for rows.Next() {
		prompt, err := s.scanPrompt(rows)
		if err != nil {
			continue // Skip invalid rows
		}
		prompts = append(prompts, prompt)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prompts)
}

// Helper functions
func (s *APIServer) scanPrompt(row interface{}) (Prompt, error) {
	var prompt Prompt
	var variablesJSON []byte

	// Scanner interface for both QueryRow and Rows
	var scanner interface{ Scan(...interface{}) error }
	switch r := row.(type) {
	case *sql.Row:
		scanner = r
	case *sql.Rows:
		scanner = r
	default:
		return prompt, fmt.Errorf("unsupported row type")
	}

	err := scanner.Scan(
		&prompt.ID, &prompt.CampaignID, &prompt.Title, &prompt.Content,
		&prompt.Description, &variablesJSON, &prompt.UsageCount, &prompt.LastUsed,
		&prompt.IsFavorite, &prompt.IsArchived, &prompt.QuickAccessKey,
		&prompt.Version, &prompt.ParentVersionID, &prompt.WordCount,
		&prompt.EstimatedTokens, &prompt.EffectivenessRating, &prompt.Notes,
		&prompt.CreatedAt, &prompt.UpdatedAt, &prompt.CampaignName,
	)

	if err != nil {
		return prompt, err
	}

	// Parse variables JSON
	if len(variablesJSON) > 0 {
		json.Unmarshal(variablesJSON, &prompt.Variables)
	}

	return prompt, nil
}

func (s *APIServer) getPromptTags(promptID string) []string {
	query := `
		SELECT t.name 
		FROM tags t
		JOIN prompt_tags pt ON t.id = pt.tag_id
		WHERE pt.prompt_id = $1
		ORDER BY t.name`

	rows, err := s.db.Query(query, promptID)
	if err != nil {
		return []string{}
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if rows.Scan(&tag) == nil {
			tags = append(tags, tag)
		}
	}

	return tags
}

func (s *APIServer) addPromptTags(promptID string, tagNames []string) {
	for _, tagName := range tagNames {
		// Insert tag if it doesn't exist
		s.db.Exec("INSERT INTO tags (name) VALUES ($1) ON CONFLICT (name) DO NOTHING", tagName)

		// Link prompt to tag
		s.db.Exec(`
			INSERT INTO prompt_tags (prompt_id, tag_id)
			SELECT $1, id FROM tags WHERE name = $2
			ON CONFLICT DO NOTHING`,
			promptID, tagName)
	}
}

func calculateWordCount(content *string) *int {
	if content == nil {
		return nil
	}
	words := strings.Fields(*content)
	count := len(words)
	return &count
}

func calculateTokenCount(content *string) *int {
	if content == nil {
		return nil
	}
	// Rough estimation: ~0.75 tokens per word for English text
	words := len(strings.Fields(*content))
	tokens := int(float64(words) * 0.75)
	return &tokens
}

// Stub implementations for remaining endpoints
func (s *APIServer) updateCampaign(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID := vars["id"]

	var req CreateCampaignRequest // Reuse the create request structure
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Build dynamic update query
	updates := []string{"updated_at = CURRENT_TIMESTAMP"}
	args := []interface{}{}
	argIndex := 1

	if req.Name != "" {
		updates = append(updates, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, req.Name)
		argIndex++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, req.Description)
		argIndex++
	}
	if req.Color != nil {
		updates = append(updates, fmt.Sprintf("color = $%d", argIndex))
		args = append(args, *req.Color)
		argIndex++
	}
	if req.Icon != nil {
		updates = append(updates, fmt.Sprintf("icon = $%d", argIndex))
		args = append(args, *req.Icon)
		argIndex++
	}
	if req.IsFavorite != nil {
		updates = append(updates, fmt.Sprintf("is_favorite = $%d", argIndex))
		args = append(args, *req.IsFavorite)
		argIndex++
	}
	if req.ParentID != nil {
		updates = append(updates, fmt.Sprintf("parent_id = $%d", argIndex))
		args = append(args, req.ParentID)
		argIndex++
	}

	// Add campaign ID as last argument
	args = append(args, campaignID)

	query := fmt.Sprintf(
		"UPDATE campaigns SET %s WHERE id = $%d",
		strings.Join(updates, ", "),
		argIndex,
	)

	result, err := s.db.Exec(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Campaign not found", http.StatusNotFound)
		return
	}

	// Return updated campaign
	s.getCampaign(w, r)
}

func (s *APIServer) deleteCampaign(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID := vars["id"]

	// Check if campaign has prompts
	var promptCount int
	err := s.db.QueryRow("SELECT COUNT(*) FROM prompts WHERE campaign_id = $1", campaignID).Scan(&promptCount)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if promptCount > 0 {
		http.Error(w, "Campaign has prompts. Delete or move prompts first.", http.StatusConflict)
		return
	}

	// Delete the campaign
	result, err := s.db.Exec("DELETE FROM campaigns WHERE id = $1", campaignID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Campaign not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *APIServer) getCampaignPrompts(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID := vars["id"]

	query := `
		SELECT p.id, p.campaign_id, p.title, p.content_cache, p.description,
		       p.variables, p.usage_count, p.last_used, p.is_favorite,
		       p.is_archived, p.quick_access_key, p.version, p.parent_version_id,
		       p.word_count, p.estimated_tokens, p.effectiveness_rating,
		       p.notes, p.created_at, p.updated_at, c.name as campaign_name
		FROM prompts p
		LEFT JOIN campaigns c ON p.campaign_id = c.id
		WHERE p.campaign_id = $1 AND p.is_archived = false
		ORDER BY p.last_used DESC NULLS LAST, p.created_at DESC`

	rows, err := s.db.Query(query, campaignID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var prompts []Prompt
	for rows.Next() {
		prompt, err := s.scanPrompt(rows)
		if err != nil {
			continue
		}
		prompts = append(prompts, prompt)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prompts)
}

func (s *APIServer) updatePrompt(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	promptID := vars["id"]

	var req UpdatePromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create version snapshot if content is being updated
	if req.Content != nil || req.Title != nil {
		_, err := s.db.Exec(`
			INSERT INTO prompt_mgr.prompt_versions (prompt_id, version_number, file_path, content_cache, variables, change_summary)
			SELECT id, version, 'snapshot', content_cache, variables::jsonb,
			       CASE
			           WHEN $2 THEN 'Manual update'
			           ELSE NULL
			       END
			FROM prompt_mgr.prompts
			WHERE id = $1
		`, promptID, req.ChangeSummary != nil)
		if err != nil {
			log.Printf("Warning: Failed to create version snapshot: %v", err)
		}
	}

	// Build dynamic update query
	updates := []string{"updated_at = CURRENT_TIMESTAMP", "version = version + 1"}
	args := []interface{}{}
	argIndex := 1

	if req.Title != nil {
		updates = append(updates, fmt.Sprintf("title = $%d", argIndex))
		args = append(args, *req.Title)
		argIndex++
	}
	if req.Content != nil {
		updates = append(updates, fmt.Sprintf("content = $%d", argIndex))
		args = append(args, *req.Content)
		argIndex++
		// Update word count and token count
		updates = append(updates, fmt.Sprintf("word_count = $%d", argIndex))
		args = append(args, calculateWordCount(req.Content))
		argIndex++
		updates = append(updates, fmt.Sprintf("estimated_tokens = $%d", argIndex))
		args = append(args, calculateTokenCount(req.Content))
		argIndex++
	}
	if req.Description != nil {
		updates = append(updates, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, req.Description)
		argIndex++
	}
	if req.IsFavorite != nil {
		updates = append(updates, fmt.Sprintf("is_favorite = $%d", argIndex))
		args = append(args, *req.IsFavorite)
		argIndex++
	}
	if req.IsArchived != nil {
		updates = append(updates, fmt.Sprintf("is_archived = $%d", argIndex))
		args = append(args, *req.IsArchived)
		argIndex++
	}
	if req.QuickAccessKey != nil {
		updates = append(updates, fmt.Sprintf("quick_access_key = $%d", argIndex))
		args = append(args, req.QuickAccessKey)
		argIndex++
	}
	if req.EffectivenessRating != nil {
		updates = append(updates, fmt.Sprintf("effectiveness_rating = $%d", argIndex))
		args = append(args, req.EffectivenessRating)
		argIndex++
	}
	if req.Notes != nil {
		updates = append(updates, fmt.Sprintf("notes = $%d", argIndex))
		args = append(args, req.Notes)
		argIndex++
	}
	if len(req.Variables) > 0 {
		variablesJSON, _ := json.Marshal(req.Variables)
		updates = append(updates, fmt.Sprintf("variables = $%d", argIndex))
		args = append(args, variablesJSON)
		argIndex++
	}

	// Add prompt ID as last argument
	args = append(args, promptID)

	query := fmt.Sprintf(
		"UPDATE prompts SET %s WHERE id = $%d",
		strings.Join(updates, ", "),
		argIndex,
	)

	_, err := s.db.Exec(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update tags if provided
	if len(req.Tags) > 0 {
		// Remove existing tags
		s.db.Exec("DELETE FROM prompt_tags WHERE prompt_id = $1", promptID)
		// Add new tags
		s.addPromptTags(promptID, req.Tags)
	}

	// Return updated prompt
	s.getPrompt(w, r)
}

func (s *APIServer) deletePrompt(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	promptID := vars["id"]

	// Delete associated tags first (cascade)
	_, err := s.db.Exec("DELETE FROM prompt_tags WHERE prompt_id = $1", promptID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Delete the prompt
	result, err := s.db.Exec("DELETE FROM prompts WHERE id = $1", promptID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Prompt not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *APIServer) recordPromptUsage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	promptID := vars["id"]

	// Update usage count and last used timestamp
	query := `UPDATE prompts SET usage_count = usage_count + 1, last_used = CURRENT_TIMESTAMP WHERE id = $1`
	_, err := s.db.Exec(query, promptID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update campaign last used
	s.db.Exec(`UPDATE campaigns SET last_used = CURRENT_TIMESTAMP WHERE id = (SELECT campaign_id FROM prompts WHERE id = $1)`, promptID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "usage recorded"})
}

func (s *APIServer) semanticSearch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query  string   `json:"query"`
		Limit  int      `json:"limit"`
		Filter []string `json:"filter"` // Optional campaign IDs to filter
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}

	// If Qdrant is available, use vector search
	if s.qdrantURL != "" {
		// Generate embedding for query using Ollama (if available)
		if s.ollamaURL != "" {
			embedding, err := s.generateEmbedding(req.Query)
			if err == nil && embedding != nil {
				// Search Qdrant with the embedding
				results, err := s.searchQdrant(embedding, req.Limit, req.Filter)
				if err == nil {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(results)
					return
				}
				log.Printf("Qdrant search failed, falling back to PostgreSQL: %v", err)
			}
		}
	}

	// Fallback to PostgreSQL full-text search
	s.searchPrompts(w, r)
}

// generateEmbedding creates an embedding vector using Ollama
func (s *APIServer) generateEmbedding(text string) ([]float32, error) {
	if s.ollamaURL == "" {
		return nil, fmt.Errorf("Ollama not configured")
	}

	payload := map[string]interface{}{
		"model":  "nomic-embed-text",
		"prompt": text,
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(s.ollamaURL+"/api/embeddings", "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding generation failed: %d", resp.StatusCode)
	}

	var result struct {
		Embedding []float32 `json:"embedding"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Embedding, nil
}

// searchQdrant searches the vector database for similar prompts
func (s *APIServer) searchQdrant(embedding []float32, limit int, campaignFilter []string) ([]Prompt, error) {
	payload := map[string]interface{}{
		"vector":       embedding,
		"limit":        limit,
		"with_payload": true,
	}

	// Add campaign filter if provided
	if len(campaignFilter) > 0 {
		payload["filter"] = map[string]interface{}{
			"must": []map[string]interface{}{
				{
					"key": "campaign_id",
					"match": map[string]interface{}{
						"any": campaignFilter,
					},
				},
			},
		}
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(s.qdrantURL+"/collections/prompts/points/search", "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("qdrant search failed: %d - %s", resp.StatusCode, string(body))
	}

	var searchResult struct {
		Result []struct {
			ID      string                 `json:"id"`
			Score   float32                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		return nil, err
	}

	// Convert Qdrant results to prompts
	promptIDs := make([]string, 0, len(searchResult.Result))
	for _, result := range searchResult.Result {
		if id, ok := result.Payload["prompt_id"].(string); ok {
			promptIDs = append(promptIDs, id)
		}
	}

	if len(promptIDs) == 0 {
		return []Prompt{}, nil
	}

	// Fetch full prompt details from PostgreSQL
	query := `
		SELECT p.id, p.campaign_id, p.title, p.content_cache, p.description,
		       p.variables, p.usage_count, p.last_used, p.is_favorite,
		       p.is_archived, p.quick_access_key, p.version, p.parent_version_id,
		       p.word_count, p.estimated_tokens, p.effectiveness_rating,
		       p.notes, p.created_at, p.updated_at, c.name as campaign_name
		FROM prompts p
		LEFT JOIN campaigns c ON p.campaign_id = c.id
		WHERE p.id = ANY($1::uuid[])
		ORDER BY array_position($1::uuid[], p.id::uuid)`

	rows, err := s.db.Query(query, "{"+strings.Join(promptIDs, ",")+"}")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prompts []Prompt
	for rows.Next() {
		prompt, err := s.scanPrompt(rows)
		if err != nil {
			continue
		}
		prompts = append(prompts, prompt)
	}

	return prompts, nil
}

func (s *APIServer) getPromptByQuickKey(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *APIServer) getRecentPrompts(w http.ResponseWriter, r *http.Request) {
	s.getPrompts(w, r) // Use existing logic with default ordering
}

func (s *APIServer) getFavoritePrompts(w http.ResponseWriter, r *http.Request) {
	// Modify query to add favorites=true
	r.URL.RawQuery = "favorites=true"
	s.getPrompts(w, r)
}

func (s *APIServer) testPrompt(w http.ResponseWriter, r *http.Request) {
	if s.ollamaURL == "" {
		http.Error(w, "Prompt testing is not available (Ollama not configured)", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	promptID := vars["id"]

	// Fetch the prompt
	query := `
		SELECT p.id, p.campaign_id, p.title, p.content_cache, p.description,
		       p.variables, p.usage_count, p.last_used, p.is_favorite,
		       p.is_archived, p.quick_access_key, p.version, p.parent_version_id,
		       p.word_count, p.estimated_tokens, p.effectiveness_rating,
		       p.notes, p.created_at, p.updated_at, c.name as campaign_name
		FROM prompts p
		LEFT JOIN campaigns c ON p.campaign_id = c.id
		WHERE p.id = $1`

	row := s.db.QueryRow(query, promptID)
	prompt, err := s.scanPrompt(row)
	if err == sql.ErrNoRows {
		http.Error(w, "Prompt not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse test request
	var req TestPromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Default model and parameters
	if req.Model == "" {
		req.Model = "llama3.2"
	}
	if req.MaxTokens == nil {
		maxTokens := 1000
		req.MaxTokens = &maxTokens
	}
	if req.Temperature == nil {
		temp := 0.7
		req.Temperature = &temp
	}

	// Replace variables in prompt content
	finalContent := prompt.Content
	for key, value := range req.Variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		finalContent = strings.ReplaceAll(finalContent, placeholder, value)
	}

	// Call Ollama API
	startTime := time.Now()

	payload := map[string]interface{}{
		"model":  req.Model,
		"prompt": finalContent,
		"stream": false,
		"options": map[string]interface{}{
			"num_predict": *req.MaxTokens,
			"temperature": *req.Temperature,
		},
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(s.ollamaURL+"/api/generate", "application/json", bytes.NewReader(jsonData))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to call Ollama: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		http.Error(w, fmt.Sprintf("Ollama error: %s", string(body)), http.StatusInternalServerError)
		return
	}

	// Parse Ollama response
	var ollamaResp struct {
		Model           string `json:"model"`
		Response        string `json:"response"`
		Done            bool   `json:"done"`
		TotalDuration   int64  `json:"total_duration"`
		LoadDuration    int64  `json:"load_duration"`
		PromptEvalCount int    `json:"prompt_eval_count"`
		EvalCount       int    `json:"eval_count"`
		EvalDuration    int64  `json:"eval_duration"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse Ollama response: %v", err), http.StatusInternalServerError)
		return
	}

	responseTime := time.Since(startTime).Milliseconds()

	// Store test result in database
	testResult := TestResult{
		ID:           uuid.New().String(),
		PromptID:     promptID,
		Model:        req.Model,
		Response:     &ollamaResp.Response,
		ResponseTime: ptrFloat64(float64(responseTime)),
		TokenCount:   &ollamaResp.EvalCount,
		TestedAt:     time.Now(),
	}

	if len(req.Variables) > 0 {
		varsJSON, _ := json.Marshal(req.Variables)
		varsStr := string(varsJSON)
		testResult.InputVars = &varsStr
	}

	// Insert test result
	insertQuery := `
		INSERT INTO test_results (id, prompt_id, model, input_variables, response, 
		                         response_time, token_count, tested_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err = s.db.Exec(insertQuery,
		testResult.ID, testResult.PromptID, testResult.Model, testResult.InputVars,
		testResult.Response, testResult.ResponseTime, testResult.TokenCount, testResult.TestedAt,
	)

	if err != nil {
		log.Printf("Failed to save test result: %v", err)
	}

	// Return test result
	response := map[string]interface{}{
		"test_id":       testResult.ID,
		"model":         testResult.Model,
		"response":      ollamaResp.Response,
		"response_time": responseTime,
		"token_count":   ollamaResp.EvalCount,
		"prompt_tokens": ollamaResp.PromptEvalCount,
		"variables":     req.Variables,
		"tested_at":     testResult.TestedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func ptrFloat64(f float64) *float64 {
	return &f
}

func (s *APIServer) getTestHistory(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *APIServer) getTags(w http.ResponseWriter, r *http.Request) {
	query := `SELECT id, name, color, description FROM tags ORDER BY name`

	rows, err := s.db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var tag Tag
		err := rows.Scan(&tag.ID, &tag.Name, &tag.Color, &tag.Description)
		if err != nil {
			continue
		}
		tags = append(tags, tag)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tags)
}

func (s *APIServer) createTag(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *APIServer) getTemplates(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *APIServer) getTemplate(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

// Export/Import functionality
type ExportData struct {
	Version    string     `json:"version"`
	ExportedAt time.Time  `json:"exported_at"`
	Campaigns  []Campaign `json:"campaigns"`
	Prompts    []Prompt   `json:"prompts"`
	Tags       []Tag      `json:"tags"`
	Templates  []Template `json:"templates,omitempty"`
}

func (s *APIServer) exportData(w http.ResponseWriter, r *http.Request) {
	// Get filter parameters
	campaignID := r.URL.Query().Get("campaign_id")
	includeArchived := r.URL.Query().Get("include_archived") == "true"

	export := ExportData{
		Version:    "1.0",
		ExportedAt: time.Now(),
		Campaigns:  []Campaign{},
		Prompts:    []Prompt{},
		Tags:       []Tag{},
		Templates:  []Template{},
	}

	// Export campaigns
	// Note: icon column may not exist in older databases, so we handle it gracefully
	campaignQuery := `SELECT id, name, description, color, parent_id, sort_order, 
		is_favorite, prompt_count, last_used, created_at, updated_at FROM campaigns`
	args := []interface{}{}

	if campaignID != "" {
		campaignQuery += " WHERE id = $1"
		args = append(args, campaignID)
	}

	rows, err := s.db.Query(campaignQuery, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var c Campaign
		c.Icon = "folder" // Default icon value
		err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.Color,
			&c.ParentID, &c.SortOrder, &c.IsFavorite, &c.PromptCount,
			&c.LastUsed, &c.CreatedAt, &c.UpdatedAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		export.Campaigns = append(export.Campaigns, c)
	}

	// Export prompts
	promptQuery := `SELECT id, campaign_id, title, content_cache, description, variables,
		usage_count, last_used, is_favorite, is_archived, quick_access_key,
		version, parent_version_id, word_count, estimated_tokens,
		effectiveness_rating, notes, created_at, updated_at
		FROM prompts WHERE 1=1`

	if campaignID != "" {
		promptQuery += " AND campaign_id = $1"
		args = []interface{}{campaignID}
	} else {
		args = []interface{}{}
	}

	if !includeArchived {
		promptQuery += " AND is_archived = false"
	}

	rows, err = s.db.Query(promptQuery, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var p Prompt
		var varsJSON string
		err := rows.Scan(&p.ID, &p.CampaignID, &p.Title, &p.Content, &p.Description,
			&varsJSON, &p.UsageCount, &p.LastUsed, &p.IsFavorite, &p.IsArchived,
			&p.QuickAccessKey, &p.Version, &p.ParentVersionID, &p.WordCount,
			&p.EstimatedTokens, &p.EffectivenessRating, &p.Notes, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Parse variables JSON
		if varsJSON != "" {
			json.Unmarshal([]byte(varsJSON), &p.Variables)
		}

		// Get tags for this prompt
		tagRows, err := s.db.Query(`
			SELECT t.name FROM tags t 
			JOIN prompt_tags pt ON t.id = pt.tag_id 
			WHERE pt.prompt_id = $1`, p.ID)
		if err == nil {
			defer tagRows.Close()
			p.Tags = []string{}
			for tagRows.Next() {
				var tagName string
				if err := tagRows.Scan(&tagName); err == nil {
					p.Tags = append(p.Tags, tagName)
				}
			}
		}

		export.Prompts = append(export.Prompts, p)
	}

	// Export all tags
	tagRows, err := s.db.Query(`SELECT id, name, color, description FROM tags`)
	if err == nil {
		defer tagRows.Close()
		for tagRows.Next() {
			var t Tag
			if err := tagRows.Scan(&t.ID, &t.Name, &t.Color, &t.Description); err == nil {
				export.Tags = append(export.Tags, t)
			}
		}
	}

	// Set response headers for file download
	fileName := fmt.Sprintf("prompt-manager-export-%s.json", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))

	json.NewEncoder(w).Encode(export)
}

func (s *APIServer) importData(w http.ResponseWriter, r *http.Request) {
	var importData ExportData
	if err := json.NewDecoder(r.Body).Decode(&importData); err != nil {
		http.Error(w, "Invalid import data format", http.StatusBadRequest)
		return
	}

	// Track what was imported
	result := map[string]interface{}{
		"campaigns_imported": 0,
		"prompts_imported":   0,
		"tags_imported":      0,
		"errors":             []string{},
	}

	// Start a transaction
	tx, err := s.db.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Import campaigns (maintaining parent-child relationships)
	campaignIDMap := make(map[string]string) // old ID -> new ID mapping

	for _, campaign := range importData.Campaigns {
		newID := uuid.New().String()
		campaignIDMap[campaign.ID] = newID

		// Handle parent_id mapping
		var parentID *string
		if campaign.ParentID != nil && *campaign.ParentID != "" {
			if newParentID, exists := campaignIDMap[*campaign.ParentID]; exists {
				parentID = &newParentID
			}
		}

		_, err := tx.Exec(`
			INSERT INTO campaigns (id, name, description, color, parent_id, 
				sort_order, is_favorite, created_at, updated_at) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (id) DO NOTHING`,
			newID, campaign.Name, campaign.Description, campaign.Color,
			parentID, campaign.SortOrder, campaign.IsFavorite,
			time.Now(), time.Now())

		if err != nil {
			result["errors"] = append(result["errors"].([]string),
				fmt.Sprintf("Failed to import campaign %s: %v", campaign.Name, err))
			continue
		}
		result["campaigns_imported"] = result["campaigns_imported"].(int) + 1
	}

	// Import tags
	tagIDMap := make(map[string]string)
	for _, tag := range importData.Tags {
		newID := uuid.New().String()
		tagIDMap[tag.ID] = newID

		_, err := tx.Exec(`
			INSERT INTO tags (id, name, color, description) 
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (name) DO NOTHING`,
			newID, tag.Name, tag.Color, tag.Description)

		if err != nil {
			result["errors"] = append(result["errors"].([]string),
				fmt.Sprintf("Failed to import tag %s: %v", tag.Name, err))
			continue
		}
		result["tags_imported"] = result["tags_imported"].(int) + 1
	}

	// Import prompts
	for _, prompt := range importData.Prompts {
		newID := uuid.New().String()

		// Map campaign ID
		campaignID := prompt.CampaignID
		if newCampaignID, exists := campaignIDMap[prompt.CampaignID]; exists {
			campaignID = newCampaignID
		}

		varsJSON, _ := json.Marshal(prompt.Variables)

		_, err := tx.Exec(`
			INSERT INTO prompts (id, campaign_id, title, content, description, 
				variables, usage_count, last_used, is_favorite, is_archived, 
				quick_access_key, version, word_count, estimated_tokens, 
				effectiveness_rating, notes, created_at, updated_at) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`,
			newID, campaignID, prompt.Title, prompt.Content, prompt.Description,
			string(varsJSON), 0, nil, prompt.IsFavorite, prompt.IsArchived,
			nil, 1, prompt.WordCount, prompt.EstimatedTokens,
			prompt.EffectivenessRating, prompt.Notes, time.Now(), time.Now())

		if err != nil {
			result["errors"] = append(result["errors"].([]string),
				fmt.Sprintf("Failed to import prompt %s: %v", prompt.Title, err))
			continue
		}

		// Link tags to prompt
		for _, tagName := range prompt.Tags {
			var tagID string
			err := tx.QueryRow("SELECT id FROM tags WHERE name = $1", tagName).Scan(&tagID)
			if err == nil {
				tx.Exec("INSERT INTO prompt_tags (prompt_id, tag_id) VALUES ($1, $2)", newID, tagID)
			}
		}

		result["prompts_imported"] = result["prompts_imported"].(int) + 1
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to commit import: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Version control stub implementations
func (s *APIServer) getPromptVersions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	promptID := vars["id"]

	// Validate prompt exists
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM prompt_mgr.prompts WHERE id = $1)", promptID).Scan(&exists)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
		return
	}
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Prompt not found"})
		return
	}

	// Retrieve version history
	query := `
		SELECT id, prompt_id, version_number, file_path, content_cache,
		       variables, change_summary, created_by, created_at
		FROM prompt_mgr.prompt_versions
		WHERE prompt_id = $1
		ORDER BY version_number DESC
	`

	rows, err := s.db.Query(query, promptID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to retrieve versions"})
		return
	}
	defer rows.Close()

	versions := make([]PromptVersion, 0)
	for rows.Next() {
		var v PromptVersion
		var variables []byte
		err := rows.Scan(&v.ID, &v.PromptID, &v.VersionNumber, &v.FilePath, &v.ContentCache,
			&variables, &v.ChangeSummary, &v.CreatedBy, &v.CreatedAt)
		if err != nil {
			continue
		}
		if len(variables) > 0 {
			json.Unmarshal(variables, &v.Variables)
		}
		versions = append(versions, v)
	}

	json.NewEncoder(w).Encode(versions)
}

func (s *APIServer) revertPromptVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	promptID := vars["id"]
	versionNum := vars["version"]

	// Validate version exists
	var version PromptVersion
	var variables []byte
	query := `
		SELECT id, prompt_id, version_number, file_path, content_cache,
		       variables, change_summary, created_by, created_at
		FROM prompt_mgr.prompt_versions
		WHERE prompt_id = $1 AND version_number = $2
	`
	err := s.db.QueryRow(query, promptID, versionNum).Scan(
		&version.ID, &version.PromptID, &version.VersionNumber, &version.FilePath,
		&version.ContentCache, &variables, &version.ChangeSummary, &version.CreatedBy, &version.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Version not found"})
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Database error"})
		}
		return
	}
	if len(variables) > 0 {
		json.Unmarshal(variables, &version.Variables)
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Get current prompt version
	var currentVersion int
	err = tx.QueryRow("SELECT version FROM prompt_mgr.prompts WHERE id = $1", promptID).Scan(&currentVersion)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to get current version"})
		return
	}

	// Create a version entry for the current state before reverting
	_, err = tx.Exec(`
		INSERT INTO prompt_mgr.prompt_versions (prompt_id, version_number, file_path, content_cache, variables, change_summary)
		SELECT id, version, 'reverted', content_cache, variables::jsonb, 'Snapshot before revert to version ' || $2
		FROM prompt_mgr.prompts
		WHERE id = $1
	`, promptID, versionNum)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create snapshot"})
		return
	}

	// Update prompt with version content
	variablesJSON, _ := json.Marshal(version.Variables)
	_, err = tx.Exec(`
		UPDATE prompt_mgr.prompts
		SET content_cache = $1,
		    variables = $2::jsonb,
		    version = version + 1,
		    parent_version_id = $3,
		    updated_at = NOW()
		WHERE id = $4
	`, version.ContentCache, variablesJSON, version.ID, promptID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to revert prompt"})
		return
	}

	if err = tx.Commit(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to commit revert"})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Reverted to version %s", versionNum),
		"new_version": currentVersion + 1,
	})
}
