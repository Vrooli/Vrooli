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

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

// Data models
type API struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Provider         string    `json:"provider"`
	Description      string    `json:"description"`
	BaseURL          string    `json:"base_url"`
	DocumentationURL string    `json:"documentation_url"`
	PricingURL       string    `json:"pricing_url"`
	Category         string    `json:"category"`
	Status           string    `json:"status"`
	AuthType         string    `json:"auth_type"`
	Tags             []string  `json:"tags"`
	Capabilities     []string  `json:"capabilities"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	LastRefreshed    time.Time `json:"last_refreshed"`
	SourceURL        string    `json:"source_url"`
}

type Note struct {
	ID        string    `json:"id"`
	APIID     string    `json:"api_id"`
	Content   string    `json:"content"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"`
}

type PricingTier struct {
	ID               string     `json:"id"`
	APIID            string     `json:"api_id"`
	Name             string     `json:"name"`
	PricePerRequest  float64    `json:"price_per_request"`
	PricePerMB       float64    `json:"price_per_mb"`
	MonthlyC         ***float64 `json:"monthly_cost"`
	FreeTierRequests int        `json:"free_tier_requests"`
}

type SearchRequest struct {
	Query      string   `json:"query"`
	Limit      int      `json:"limit"`
	Configured bool     `json:"configured,omitempty"`
	MaxPrice   float64  `json:"max_price,omitempty"`
	Categories []string `json:"categories,omitempty"`
}

type SearchResult struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Provider       string  `json:"provider"`
	Description    string  `json:"description"`
	RelevanceScore float64 `json:"relevance_score"`
	Configured     bool    `json:"configured"`
	PricingSummary string  `json:"pricing_summary"`
	Category       string  `json:"category"`
}

type ResearchRequest struct {
	Capability   string                 `json:"capability"`
	Requirements map[string]interface{} `json:"requirements"`
}

type ResearchResponse struct {
	ResearchID    string `json:"research_id"`
	Status        string `json:"status"`
	EstimatedTime int    `json:"estimated_time"`
}

var db *sql.DB

func main() {
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start api-library

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	initDB()
	router := setupRouter()

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(router)

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

	log.Printf("API Library service starting on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func initDB() {
	// Database configuration - support both POSTGRES_URL and individual components
	psqlInfo := os.Getenv("POSTGRES_URL")
	if psqlInfo == "" {
		// Try to build from individual components - REQUIRED, no defaults
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")

		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Fatal("❌ Database configuration missing. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}

		psqlInfo = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			dbHost, dbPort, dbUser, dbPassword, dbName)
	}

	var err error
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
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
			log.Printf("✅ Database connected successfully on attempt %d", attempt+1)
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
}

func setupRouter() *mux.Router {
	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", healthHandler).Methods("GET")

	// API v1 routes
	v1 := router.PathPrefix("/api/v1").Subrouter()

	// Search endpoints
	v1.HandleFunc("/search", searchAPIsHandler).Methods("GET", "POST")

	// API CRUD
	v1.HandleFunc("/apis", listAPIsHandler).Methods("GET")
	v1.HandleFunc("/apis", createAPIHandler).Methods("POST")
	v1.HandleFunc("/apis/{id}", getAPIHandler).Methods("GET")
	v1.HandleFunc("/apis/{id}", updateAPIHandler).Methods("PUT")
	v1.HandleFunc("/apis/{id}", deleteAPIHandler).Methods("DELETE")

	// Notes
	v1.HandleFunc("/apis/{id}/notes", getNotesHandler).Methods("GET")
	v1.HandleFunc("/apis/{id}/notes", addNoteHandler).Methods("POST")

	// Configuration tracking
	v1.HandleFunc("/configured", getConfiguredAPIsHandler).Methods("GET")
	v1.HandleFunc("/apis/{id}/configure", markConfiguredHandler).Methods("POST")

	// Research integration
	v1.HandleFunc("/request-research", requestResearchHandler).Methods("POST")

	return router
}

// Handlers

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "api-library",
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func searchAPIsHandler(w http.ResponseWriter, r *http.Request) {
	var searchReq SearchRequest

	if r.Method == "POST" {
		if err := json.NewDecoder(r.Body).Decode(&searchReq); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
	} else {
		searchReq.Query = r.URL.Query().Get("query")
		searchReq.Limit = 10 // default
	}

	if searchReq.Query == "" {
		http.Error(w, "Query parameter is required", http.StatusBadRequest)
		return
	}

	if searchReq.Limit == 0 {
		searchReq.Limit = 10
	}

	// Perform text search using PostgreSQL's full-text search
	query := `
		SELECT 
			a.id, a.name, a.provider, a.description, a.category,
			ts_rank(a.search_vector, plainto_tsquery('english', $1)) as relevance,
			COALESCE(c.is_configured, false) as configured,
			COALESCE(p.min_price::text, 'Pricing not available') as pricing
		FROM apis a
		LEFT JOIN api_credentials c ON a.id = c.api_id AND c.environment = 'development'
		LEFT JOIN (
			SELECT api_id, MIN(COALESCE(price_per_request, monthly_cost)) as min_price 
			FROM pricing_tiers 
			GROUP BY api_id
		) p ON a.id = p.api_id
		WHERE a.search_vector @@ plainto_tsquery('english', $1)
		ORDER BY relevance DESC
		LIMIT $2
	`

	rows, err := db.Query(query, searchReq.Query, searchReq.Limit)
	if err != nil {
		log.Printf("Search query failed: %v", err)
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var result SearchResult
		err := rows.Scan(&result.ID, &result.Name, &result.Provider,
			&result.Description, &result.Category, &result.RelevanceScore,
			&result.Configured, &result.PricingSummary)
		if err != nil {
			log.Printf("Failed to scan row: %v", err)
			continue
		}
		results = append(results, result)
	}

	response := map[string]interface{}{
		"results": results,
		"count":   len(results),
		"query":   searchReq.Query,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func listAPIsHandler(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	status := r.URL.Query().Get("status")

	query := `SELECT id, name, provider, description, category, status FROM apis WHERE 1=1`
	var args []interface{}
	argCount := 0

	if category != "" {
		argCount++
		query += fmt.Sprintf(" AND category = $%d", argCount)
		args = append(args, category)
	}

	if status != "" {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
	}

	query += " ORDER BY name"

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to fetch APIs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var apis []API
	for rows.Next() {
		var api API
		err := rows.Scan(&api.ID, &api.Name, &api.Provider, &api.Description, &api.Category, &api.Status)
		if err != nil {
			continue
		}
		apis = append(apis, api)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apis)
}

func createAPIHandler(w http.ResponseWriter, r *http.Request) {
	var api API
	if err := json.NewDecoder(r.Body).Decode(&api); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	api.ID = uuid.New().String()
	api.CreatedAt = time.Now()
	api.UpdatedAt = time.Now()
	api.LastRefreshed = time.Now()

	query := `
		INSERT INTO apis (id, name, provider, description, base_url, documentation_url, 
			pricing_url, category, status, auth_type, tags, capabilities, source_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING created_at, updated_at
	`

	err := db.QueryRow(query, api.ID, api.Name, api.Provider, api.Description,
		api.BaseURL, api.DocumentationURL, api.PricingURL, api.Category,
		api.Status, api.AuthType, pq.Array(api.Tags), pq.Array(api.Capabilities),
		api.SourceURL).Scan(&api.CreatedAt, &api.UpdatedAt)

	if err != nil {
		log.Printf("Failed to create API: %v", err)
		http.Error(w, "Failed to create API", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(api)
}

func getAPIHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]

	var api API
	query := `
		SELECT id, name, provider, description, base_url, documentation_url,
			pricing_url, category, status, auth_type, tags, capabilities,
			created_at, updated_at, last_refreshed, source_url
		FROM apis WHERE id = $1
	`

	err := db.QueryRow(query, apiID).Scan(
		&api.ID, &api.Name, &api.Provider, &api.Description,
		&api.BaseURL, &api.DocumentationURL, &api.PricingURL,
		&api.Category, &api.Status, &api.AuthType,
		pq.Array(&api.Tags), pq.Array(&api.Capabilities),
		&api.CreatedAt, &api.UpdatedAt, &api.LastRefreshed, &api.SourceURL)

	if err == sql.ErrNoRows {
		http.Error(w, "API not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to fetch API", http.StatusInternalServerError)
		return
	}

	// Fetch notes
	notesQuery := `SELECT id, content, type, created_at, created_by FROM notes WHERE api_id = $1`
	rows, _ := db.Query(notesQuery, apiID)
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		rows.Scan(&note.ID, &note.Content, &note.Type, &note.CreatedAt, &note.CreatedBy)
		notes = append(notes, note)
	}

	response := map[string]interface{}{
		"api":   api,
		"notes": notes,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func updateAPIHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]

	var api API
	if err := json.NewDecoder(r.Body).Decode(&api); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	query := `
		UPDATE apis 
		SET name = $2, provider = $3, description = $4, base_url = $5,
			documentation_url = $6, pricing_url = $7, category = $8,
			status = $9, auth_type = $10, tags = $11, capabilities = $12,
			source_url = $13, updated_at = NOW()
		WHERE id = $1
	`

	_, err := db.Exec(query, apiID, api.Name, api.Provider, api.Description,
		api.BaseURL, api.DocumentationURL, api.PricingURL, api.Category,
		api.Status, api.AuthType, pq.Array(api.Tags), pq.Array(api.Capabilities),
		api.SourceURL)

	if err != nil {
		http.Error(w, "Failed to update API", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func deleteAPIHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]

	_, err := db.Exec("DELETE FROM apis WHERE id = $1", apiID)
	if err != nil {
		http.Error(w, "Failed to delete API", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func getNotesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]

	query := `SELECT id, content, type, created_at, created_by FROM notes WHERE api_id = $1 ORDER BY created_at DESC`
	rows, err := db.Query(query, apiID)
	if err != nil {
		http.Error(w, "Failed to fetch notes", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var notes []Note
	for rows.Next() {
		var note Note
		err := rows.Scan(&note.ID, &note.Content, &note.Type, &note.CreatedAt, &note.CreatedBy)
		if err != nil {
			continue
		}
		note.APIID = apiID
		notes = append(notes, note)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notes)
}

func addNoteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]

	var note Note
	if err := json.NewDecoder(r.Body).Decode(&note); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	note.ID = uuid.New().String()
	note.APIID = apiID
	note.CreatedAt = time.Now()
	if note.CreatedBy == "" {
		note.CreatedBy = "user"
	}

	query := `
		INSERT INTO notes (id, api_id, content, type, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at
	`

	err := db.QueryRow(query, note.ID, note.APIID, note.Content, note.Type, note.CreatedBy).Scan(&note.CreatedAt)
	if err != nil {
		http.Error(w, "Failed to add note", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(note)
}

func getConfiguredAPIsHandler(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT a.id, a.name, a.provider, a.description, a.category, c.environment, c.configuration_date
		FROM apis a
		JOIN api_credentials c ON a.id = c.api_id
		WHERE c.is_configured = true
		ORDER BY a.name
	`

	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, "Failed to fetch configured APIs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id, name, provider, description, category, environment string
		var configDate time.Time

		err := rows.Scan(&id, &name, &provider, &description, &category, &environment, &configDate)
		if err != nil {
			continue
		}

		results = append(results, map[string]interface{}{
			"id":                 id,
			"name":               name,
			"provider":           provider,
			"description":        description,
			"category":           category,
			"environment":        environment,
			"configuration_date": configDate,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func markConfiguredHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	apiID := vars["id"]

	var config struct {
		Environment string `json:"environment"`
		Notes       string `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if config.Environment == "" {
		config.Environment = "development"
	}

	query := `
		INSERT INTO api_credentials (api_id, is_configured, environment, configuration_notes, configuration_date)
		VALUES ($1, true, $2, $3, NOW())
		ON CONFLICT (api_id, environment) 
		DO UPDATE SET is_configured = true, configuration_notes = $3, configuration_date = NOW()
	`

	_, err := db.Exec(query, apiID, config.Environment, config.Notes)
	if err != nil {
		http.Error(w, "Failed to mark API as configured", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func requestResearchHandler(w http.ResponseWriter, r *http.Request) {
	var req ResearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Capability == "" {
		http.Error(w, "Capability is required", http.StatusBadRequest)
		return
	}

	// Create research request in database
	researchID := uuid.New().String()
	query := `
		INSERT INTO research_requests (id, capability, requirements, status)
		VALUES ($1, $2, $3, 'queued')
	`

	reqJSON, _ := json.Marshal(req.Requirements)
	_, err := db.Exec(query, researchID, req.Capability, reqJSON)
	if err != nil {
		http.Error(w, "Failed to create research request", http.StatusInternalServerError)
		return
	}

	// TODO: Trigger research-assistant scenario
	// For now, just return the queued status

	response := ResearchResponse{
		ResearchID:    researchID,
		Status:        "queued",
		EstimatedTime: 300, // 5 minutes estimate
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// getEnv removed to prevent hardcoded defaults
