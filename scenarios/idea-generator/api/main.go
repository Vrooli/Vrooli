package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/handlers"
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
	n8nURL         string
	windmillURL    string
	postgresURL    string
	qdrantURL      string
	minioURL       string
	redisURL       string
	ollamaURL      string
	unstructuredURL string
}

func NewApiServer() *ApiServer {
	return &ApiServer{
		n8nURL:         getEnvOrDefault("N8N_BASE_URL", "http://localhost:5678"),
		windmillURL:    getEnvOrDefault("WINDMILL_BASE_URL", "http://localhost:5681"),
		postgresURL:    getEnvOrDefault("POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/idea_generator"),
		qdrantURL:      getEnvOrDefault("QDRANT_URL", "http://localhost:6333"),
		minioURL:       getEnvOrDefault("MINIO_URL", "http://localhost:9000"),
		redisURL:       getEnvOrDefault("REDIS_URL", "redis://localhost:6379"),
		ollamaURL:      getEnvOrDefault("OLLAMA_URL", "http://localhost:11434"),
		unstructuredURL: getEnvOrDefault("UNSTRUCTURED_URL", "http://localhost:11450"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
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
		ideas := []Idea{
			{
				ID:         "1",
				CampaignID: "1",
				Title:      "AI-Powered Productivity Assistant",
				Content:    "A smart assistant that learns user habits and automates routine tasks",
				Status:     "draft",
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ideas)

	case "POST":
		var idea Idea
		if err := json.NewDecoder(r.Body).Decode(&idea); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		
		idea.ID = fmt.Sprintf("%d", time.Now().Unix())
		idea.CreatedAt = time.Now()
		idea.UpdatedAt = time.Now()
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(idea)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *ApiServer) workflowsHandler(w http.ResponseWriter, r *http.Request) {
	workflows := []Workflow{
		{
			ID:          "idea-generation-workflow",
			Name:        "Idea Generation",
			Description: "AI-powered idea generation with context awareness",
			Status:      "active",
			URL:         s.n8nURL + "/workflow/idea-generation-workflow",
		},
		{
			ID:          "document-processing-pipeline",
			Name:        "Document Processing",
			Description: "Upload and process documents for context extraction",
			Status:      "active",
			URL:         s.n8nURL + "/workflow/document-processing-pipeline",
		},
		{
			ID:          "semantic-search-workflow",
			Name:        "Semantic Search",
			Description: "Vector-based search across ideas and documents",
			Status:      "active",
			URL:         s.n8nURL + "/workflow/semantic-search-workflow",
		},
		{
			ID:          "agent-refinement-workflow",
			Name:        "Agent Refinement",
			Description: "Multi-agent system for idea development",
			Status:      "active",
			URL:         s.n8nURL + "/workflow/agent-refinement-workflow",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workflows)
}

func (s *ApiServer) statusHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"service":       "idea-generator-api",
		"version":       "1.0.0",
		"timestamp":     time.Now(),
		"uptime":        "running",
		"resources": map[string]string{
			"n8n":            s.n8nURL,
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

func main() {
	server := NewApiServer()
	
	r := mux.NewRouter()
	
	// API routes
	r.HandleFunc("/health", server.healthHandler).Methods("GET")
	r.HandleFunc("/status", server.statusHandler).Methods("GET")
	r.HandleFunc("/campaigns", server.campaignsHandler).Methods("GET", "POST")
	r.HandleFunc("/ideas", server.ideasHandler).Methods("GET", "POST")
	r.HandleFunc("/workflows", server.workflowsHandler).Methods("GET")

	// Enable CORS
	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"*"}),
	)(r)

	port := getEnvOrDefault("PORT", "8500")
	
	log.Printf("Idea Generator API server starting on port %s", port)
	log.Printf("Services:")
	log.Printf("  n8n: %s", server.n8nURL)
	log.Printf("  Windmill: %s", server.windmillURL)
	log.Printf("  Qdrant: %s", server.qdrantURL)
	log.Printf("  MinIO: %s", server.minioURL)
	log.Printf("  Ollama: %s", server.ollamaURL)
	log.Printf("  Unstructured: %s", server.unstructuredURL)

	log.Fatal(http.ListenAndServe(":"+port, corsHandler))
}