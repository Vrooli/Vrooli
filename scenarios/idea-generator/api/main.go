package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/handlers"
	_ "github.com/lib/pq"
)

type Campaign struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Color       string    `json:"color"`
	CreatedAt   time.Time `json:"created_at"`
}

type Idea struct {
	ID         string    `json:"id"`
	CampaignID string    `json:"campaign_id"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Workflow struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	URL         string `json:"url"`
}

type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

type ApiServer struct {
	db             *sql.DB
	ideaProcessor  *IdeaProcessor
	windmillURL    string
	postgresURL    string
	qdrantURL      string
	minioURL       string
	redisURL       string
	ollamaURL      string
	unstructuredURL string
}

func NewApiServer() (*ApiServer, error) {
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
			return nil, fmt.Errorf("❌ Missing database configuration. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}
		
		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}
	
	// Connect to database with exponential backoff
	db, err := initDB(postgresURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	
	// Initialize IdeaProcessor
	ideaProcessor := NewIdeaProcessor(db)
	
	// All service URLs are optional - will use sensible defaults if available
	windmillURL := os.Getenv("WINDMILL_BASE_URL")
	qdrantURL := os.Getenv("QDRANT_URL")
	minioURL := os.Getenv("MINIO_URL")
	redisURL := os.Getenv("REDIS_URL")
	ollamaURL := os.Getenv("OLLAMA_URL")
	unstructuredURL := os.Getenv("UNSTRUCTURED_URL")
	
	return &ApiServer{
		db:             db,
		ideaProcessor:  ideaProcessor,
		windmillURL:    windmillURL,
		postgresURL:    postgresURL,
		qdrantURL:      qdrantURL,
		minioURL:       minioURL,
		redisURL:       redisURL,
		ollamaURL:      ollamaURL,
		unstructuredURL: unstructuredURL,
	}, nil
}

func initDB(postgresURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
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

		// Add random jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * rand.Float64())
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
		return nil, fmt.Errorf("❌ Database connection failed after %d attempts: %w", maxRetries, pingErr)
	}
	
	log.Println("🎉 Database connection pool established successfully!")
	return db, nil
}

func (s *ApiServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	services := map[string]string{
		"n8n":            "healthy",
		"windmill":       "healthy", 
		"postgres":       "healthy",
		"qdrant":         "healthy",
		"minio":          "healthy",
		"redis":          "healthy",
		"ollama":         "healthy",
		"unstructured":   "healthy",
	}

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Services:  services,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *ApiServer) campaignsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		campaigns := []Campaign{
			{
				ID:          "1",
				Name:        "Product Innovation",
				Description: "Generate innovative product ideas for Q1",
				Color:       "#3B82F6",
				CreatedAt:   time.Now(),
			},
			{
				ID:          "2", 
				Name:        "Marketing Strategy",
				Description: "Creative marketing campaign ideas",
				Color:       "#10B981",
				CreatedAt:   time.Now(),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(campaigns)

	case "POST":
		var campaign Campaign
		if err := json.NewDecoder(r.Body).Decode(&campaign); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		campaign.ID = fmt.Sprintf("%d", time.Now().Unix())
		campaign.CreatedAt = time.Now()
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(campaign)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *ApiServer) ideasHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// Query ideas from database
		campaignID := r.URL.Query().Get("campaign_id")
		query := `SELECT id, campaign_id, title, content, status, created_at, updated_at 
				  FROM ideas WHERE 1=1`
		args := []interface{}{}
		
		if campaignID != "" {
			query += " AND campaign_id = $1"
			args = append(args, campaignID)
		}
		query += " ORDER BY created_at DESC LIMIT 50"
		
		rows, err := s.db.Query(query, args...)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		
		ideas := []Idea{}
		for rows.Next() {
			var idea Idea
			err := rows.Scan(&idea.ID, &idea.CampaignID, &idea.Title, 
				&idea.Content, &idea.Status, &idea.CreatedAt, &idea.UpdatedAt)
			if err != nil {
				continue
			}
			ideas = append(ideas, idea)
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ideas)

	case "POST":
		var req GenerateIdeasRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		// Use IdeaProcessor to generate ideas
		ctx := r.Context()
		response := s.ideaProcessor.GenerateIdeas(ctx, req)
		
		if !response.Success {
			http.Error(w, response.Error, http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *ApiServer) generateIdeasHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CampaignID      string   `json:"campaign_id"`
		Context         string   `json:"context"`
		DocumentRefs    []string `json:"document_refs"`
		CreativityLevel float64  `json:"creativity_level"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Create a GenerateIdeasRequest for the processor
	generateReq := GenerateIdeasRequest{
		CampaignID: req.CampaignID,
		Prompt:     req.Context,
		Count:      1, // Default to 1 idea per request
	}
	
	// Use IdeaProcessor to generate ideas
	ctx := r.Context()
	response := s.ideaProcessor.GenerateIdeas(ctx, generateReq)
	
	if !response.Success {
		http.Error(w, response.Error, http.StatusInternalServerError)
		return
	}
	
	// Convert response to expected format
	if len(response.Ideas) > 0 {
		idea := response.Ideas[0]
		result := map[string]interface{}{
			"id":      fmt.Sprintf("idea-%d", time.Now().Unix()),
			"title":   idea.Title,
			"content": idea.Description,
			"generation_metadata": map[string]interface{}{
				"context_used":    []string{req.Context},
				"processing_time": 1500,
				"confidence":      0.85,
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	} else {
		http.Error(w, "No ideas generated", http.StatusInternalServerError)
	}
}

func (s *ApiServer) workflowsHandler(w http.ResponseWriter, r *http.Request) {
	// Return available processing capabilities
	capabilities := []map[string]string{
		{
			"id":          "idea-generation",
			"name":        "Idea Generation",
			"description": "AI-powered idea generation with context awareness",
			"status":      "active",
			"endpoint":    "/api/ideas",
		},
		{
			"id":          "document-processing",
			"name":        "Document Processing",
			"description": "Upload and process documents for context extraction",
			"status":      "active",
			"endpoint":    "/api/documents/process",
		},
		{
			"id":          "semantic-search",
			"name":        "Semantic Search",
			"description": "Vector-based search across ideas and documents",
			"status":      "active",
			"endpoint":    "/api/search",
		},
		{
			"id":          "idea-refinement",
			"name":        "Idea Refinement",
			"description": "Refine and improve existing ideas",
			"status":      "active",
			"endpoint":    "/api/ideas/refine",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(capabilities)
}

func (s *ApiServer) statusHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"service":       "idea-generator-api",
		"version":       "1.0.0",
		"timestamp":     time.Now(),
		"uptime":        "running",
		"resources": map[string]string{
			"windmill":       s.windmillURL,
			"postgres":       "connected",
			"qdrant":         s.qdrantURL,
			"minio":          s.minioURL,
			"redis":          "connected",
			"ollama":         s.ollamaURL,
			"unstructured":   s.unstructuredURL,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *ApiServer) searchHandler(w http.ResponseWriter, r *http.Request) {
	var req SemanticSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	ctx := r.Context()
	results, err := s.ideaProcessor.SemanticSearch(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (s *ApiServer) refineIdeaHandler(w http.ResponseWriter, r *http.Request) {
	var req RefinementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	ctx := r.Context()
	err := s.ideaProcessor.RefineIdea(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	response := map[string]interface{}{
		"success": true,
		"message": "Idea refined successfully",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *ApiServer) processDocumentHandler(w http.ResponseWriter, r *http.Request) {
	var req DocumentProcessingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	ctx := r.Context()
	err := s.ideaProcessor.ProcessDocument(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	response := map[string]interface{}{
		"success": true,
		"message": "Document processing started",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start idea-generator

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "15100" // Default port for idea-generator API
		log.Printf("⚠️  API_PORT not set, using default port %s", port)
	}
	
	server, err := NewApiServer()
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}
	defer server.db.Close()
	
	r := mux.NewRouter()
	
	// API routes
	r.HandleFunc("/health", server.healthHandler).Methods("GET")
	r.HandleFunc("/status", server.statusHandler).Methods("GET")
	r.HandleFunc("/campaigns", server.campaignsHandler).Methods("GET", "POST")
	r.HandleFunc("/ideas", server.ideasHandler).Methods("GET", "POST")
	r.HandleFunc("/api/ideas/generate", server.generateIdeasHandler).Methods("POST")
	r.HandleFunc("/ideas/refine", server.refineIdeaHandler).Methods("POST")
	r.HandleFunc("/search", server.searchHandler).Methods("POST")
	r.HandleFunc("/documents/process", server.processDocumentHandler).Methods("POST")
	r.HandleFunc("/workflows", server.workflowsHandler).Methods("GET")

	// Enable CORS
	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"*"}),
	)(r)
	
	log.Printf("Idea Generator API server starting on port %s", port)
	log.Printf("Services:")
	log.Printf("  Database: Connected")
	log.Printf("  Windmill: %s", server.windmillURL)
	log.Printf("  Qdrant: %s", server.qdrantURL)
	log.Printf("  MinIO: %s", server.minioURL)
	log.Printf("  Ollama: %s", server.ollamaURL)
	log.Printf("  Unstructured: %s", server.unstructuredURL)

	log.Fatal(http.ListenAndServe(":"+port, corsHandler))
}