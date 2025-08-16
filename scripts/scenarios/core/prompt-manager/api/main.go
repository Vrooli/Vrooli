package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Domain models
type Campaign struct {
	ID           string    `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	Description  *string   `json:"description" db:"description"`
	Color        string    `json:"color" db:"color"`
	Icon         string    `json:"icon" db:"icon"`
	ParentID     *string   `json:"parent_id" db:"parent_id"`
	SortOrder    int       `json:"sort_order" db:"sort_order"`
	IsFavorite   bool      `json:"is_favorite" db:"is_favorite"`
	PromptCount  int       `json:"prompt_count" db:"prompt_count"`
	LastUsed     *time.Time `json:"last_used" db:"last_used"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type Prompt struct {
	ID                   string    `json:"id" db:"id"`
	CampaignID           string    `json:"campaign_id" db:"campaign_id"`
	Title                string    `json:"title" db:"title"`
	Content              string    `json:"content" db:"content"`
	Description          *string   `json:"description" db:"description"`
	Variables            []string  `json:"variables" db:"variables"`
	UsageCount           int       `json:"usage_count" db:"usage_count"`
	LastUsed             *time.Time `json:"last_used" db:"last_used"`
	IsFavorite           bool      `json:"is_favorite" db:"is_favorite"`
	IsArchived           bool      `json:"is_archived" db:"is_archived"`
	QuickAccessKey       *string   `json:"quick_access_key" db:"quick_access_key"`
	Version              int       `json:"version" db:"version"`
	ParentVersionID      *string   `json:"parent_version_id" db:"parent_version_id"`
	WordCount            *int      `json:"word_count" db:"word_count"`
	EstimatedTokens      *int      `json:"estimated_tokens" db:"estimated_tokens"`
	EffectivenessRating  *int      `json:"effectiveness_rating" db:"effectiveness_rating"`
	Notes                *string   `json:"notes" db:"notes"`
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time `json:"updated_at" db:"updated_at"`
	// Joined fields
	CampaignName         *string   `json:"campaign_name,omitempty"`
	Tags                 []string  `json:"tags,omitempty"`
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
	ID            string    `json:"id" db:"id"`
	PromptID      string    `json:"prompt_id" db:"prompt_id"`
	Model         string    `json:"model" db:"model"`
	InputVars     *string   `json:"input_variables" db:"input_variables"`
	Response      *string   `json:"response" db:"response"`
	ResponseTime  *float64  `json:"response_time" db:"response_time"`
	TokenCount    *int      `json:"token_count" db:"token_count"`
	Rating        *int      `json:"rating" db:"rating"`
	Notes         *string   `json:"notes" db:"notes"`
	TestedAt      time.Time `json:"tested_at" db:"tested_at"`
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
	Model         string            `json:"model"`
	Variables     map[string]string `json:"variables"`
	MaxTokens     *int              `json:"max_tokens"`
	Temperature   *float64          `json:"temperature"`
}

// API Server
type APIServer struct {
	db        *sql.DB
	qdrantURL string
	ollamaURL string
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8085"
	}

	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		postgresURL = "postgres://postgres:postgres@localhost:5433/prompt_manager?sslmode=disable"
	}

	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		qdrantURL = "http://localhost:6333"
	}

	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	// Connect to database
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

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

	// Health check
	router.HandleFunc("/health", server.healthCheck).Methods("GET")

	// API routes
	api := router.PathPrefix("/api").Subrouter()

	// Campaign endpoints
	api.HandleFunc("/campaigns", server.getCampaigns).Methods("GET")
	api.HandleFunc("/campaigns", server.createCampaign).Methods("POST")
	api.HandleFunc("/campaigns/{id}", server.getCampaign).Methods("GET")
	api.HandleFunc("/campaigns/{id}", server.updateCampaign).Methods("PUT")
	api.HandleFunc("/campaigns/{id}", server.deleteCampaign).Methods("DELETE")
	api.HandleFunc("/campaigns/{id}/prompts", server.getCampaignPrompts).Methods("GET")

	// Prompt endpoints
	api.HandleFunc("/prompts", server.getPrompts).Methods("GET")
	api.HandleFunc("/prompts", server.createPrompt).Methods("POST")
	api.HandleFunc("/prompts/{id}", server.getPrompt).Methods("GET")
	api.HandleFunc("/prompts/{id}", server.updatePrompt).Methods("PUT")
	api.HandleFunc("/prompts/{id}", server.deletePrompt).Methods("DELETE")
	api.HandleFunc("/prompts/{id}/use", server.recordPromptUsage).Methods("POST")
	
	// Search endpoints
	api.HandleFunc("/search/prompts", server.searchPrompts).Methods("GET")
	api.HandleFunc("/prompts/semantic", server.semanticSearch).Methods("POST")
	
	// Quick access
	api.HandleFunc("/prompts/quick/{key}", server.getPromptByQuickKey).Methods("GET")
	api.HandleFunc("/prompts/recent", server.getRecentPrompts).Methods("GET")
	api.HandleFunc("/prompts/favorites", server.getFavoritePrompts).Methods("GET")

	// Testing
	api.HandleFunc("/prompts/{id}/test", server.testPrompt).Methods("POST")
	api.HandleFunc("/prompts/{id}/test-history", server.getTestHistory).Methods("GET")

	// Tags
	api.HandleFunc("/tags", server.getTags).Methods("GET")
	api.HandleFunc("/tags", server.createTag).Methods("POST")

	// Templates
	api.HandleFunc("/templates", server.getTemplates).Methods("GET")
	api.HandleFunc("/templates/{id}", server.getTemplate).Methods("GET")
	
	// N8N Workflow Integration endpoints
	api.HandleFunc("/enhance", server.enhancePrompt).Methods("POST")
	api.HandleFunc("/campaigns/manage", server.manageCampaignViaWorkflow).Methods("POST")

	log.Printf("🚀 Prompt Manager API starting on port %s", port)
	log.Printf("🗄️  Database: %s", postgresURL)
	log.Printf("🔍 Qdrant: %s", qdrantURL)
	log.Printf("🧠 Ollama: %s", ollamaURL)

	handler := corsHandler(router)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

// Health check endpoint
func (s *APIServer) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	status := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"services": map[string]interface{}{
			"database": s.checkDatabase(),
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
	resp, err := http.Get(s.qdrantURL + "/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		return "unavailable"
	}
	return "healthy"
}

func (s *APIServer) checkOllama() string {
	resp, err := http.Get(s.ollamaURL + "/api/tags")
	if err != nil || resp.StatusCode != http.StatusOK {
		return "unavailable"
	}
	return "healthy"
}

// Campaign endpoints
func (s *APIServer) getCampaigns(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, name, description, color, icon, parent_id, sort_order, 
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
		err := rows.Scan(
			&campaign.ID, &campaign.Name, &campaign.Description, &campaign.Color,
			&campaign.Icon, &campaign.ParentID, &campaign.SortOrder,
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
		ID:         uuid.New().String(),
		Name:       req.Name,
		Description: req.Description,
		Color:      "#6366f1", // Default color
		Icon:       "folder",  // Default icon
		SortOrder:  0,
		IsFavorite: false,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
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
		INSERT INTO campaigns (id, name, description, color, icon, parent_id, 
		                      sort_order, is_favorite, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING prompt_count, last_used`

	err := s.db.QueryRow(query,
		campaign.ID, campaign.Name, campaign.Description, campaign.Color,
		campaign.Icon, req.ParentID, campaign.SortOrder, campaign.IsFavorite,
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
		SELECT id, name, description, color, icon, parent_id, sort_order,
		       is_favorite, prompt_count, last_used, created_at, updated_at
		FROM campaigns 
		WHERE id = $1`

	var campaign Campaign
	err := s.db.QueryRow(query, campaignID).Scan(
		&campaign.ID, &campaign.Name, &campaign.Description, &campaign.Color,
		&campaign.Icon, &campaign.ParentID, &campaign.SortOrder,
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
		SELECT p.id, p.campaign_id, p.title, p.content, p.description, 
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

	var prompts []Prompt
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
		SELECT p.id, p.campaign_id, p.title, p.content, p.description, 
		       p.variables, p.usage_count, p.last_used, p.is_favorite, 
		       p.is_archived, p.quick_access_key, p.version, p.parent_version_id,
		       p.word_count, p.estimated_tokens, p.effectiveness_rating, 
		       p.notes, p.created_at, p.updated_at, c.name as campaign_name
		FROM prompts p
		LEFT JOIN campaigns c ON p.campaign_id = c.id
		WHERE p.is_archived = false AND (
			to_tsvector('english', p.title) @@ plainto_tsquery('english', $1) OR
			to_tsvector('english', p.content) @@ plainto_tsquery('english', $1) OR
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
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *APIServer) deleteCampaign(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *APIServer) getCampaignPrompts(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID := vars["id"]

	query := `
		SELECT p.id, p.campaign_id, p.title, p.content, p.description, 
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
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *APIServer) deletePrompt(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
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

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "usage recorded"})
}

func (s *APIServer) semanticSearch(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
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
	w.WriteHeader(http.StatusNotImplemented)
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

// N8N Workflow Integration Handlers
func (s *APIServer) enhancePrompt(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Prompt           string `json:"prompt"`
		EnhancementType  string `json:"enhancement_type"`
		Context          string `json:"context"`
		TargetAudience   string `json:"target_audience"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Call n8n webhook for prompt enhancement
	n8nURL := os.Getenv("N8N_URL")
	if n8nURL == "" {
		n8nURL = "http://localhost:5678"
	}

	webhookURL := fmt.Sprintf("%s/webhook/prompt-enhancer-webhook", n8nURL)
	
	payload, _ := json.Marshal(req)
	resp, err := http.Post(webhookURL, "application/json", strings.NewReader(string(payload)))
	if err != nil {
		http.Error(w, "Failed to enhance prompt", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *APIServer) manageCampaignViaWorkflow(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action       string   `json:"action"`
		CampaignName string   `json:"campaign_name"`
		Prompts      []string `json:"prompts"`
		SearchQuery  string   `json:"search_query"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Call n8n webhook for campaign management
	n8nURL := os.Getenv("N8N_URL")
	if n8nURL == "" {
		n8nURL = "http://localhost:5678"
	}

	webhookURL := fmt.Sprintf("%s/webhook/campaign-manager-webhook", n8nURL)
	
	payload, _ := json.Marshal(req)
	resp, err := http.Post(webhookURL, "application/json", strings.NewReader(string(payload)))
	if err != nil {
		http.Error(w, "Failed to manage campaign", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}