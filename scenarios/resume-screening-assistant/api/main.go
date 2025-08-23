package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type Config struct {
	Port          string
	N8NBaseURL    string
	WindmillURL   string
	PostgresURL   string
	QdrantURL     string
}

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Version string `json:"version"`
	Time    string `json:"time"`
}

type JobsResponse struct {
	Success bool          `json:"success"`
	Jobs    []interface{} `json:"jobs"`
	Count   int           `json:"count"`
}

type CandidatesResponse struct {
	Success    bool          `json:"success"`
	Candidates []interface{} `json:"candidates"`
	Count      int           `json:"count"`
}

type SearchResponse struct {
	Success bool          `json:"success"`
	Results []interface{} `json:"results"`
	Count   int           `json:"count"`
	Query   string        `json:"query"`
}

func loadConfig() *Config {
	return &Config{
		Port:          getEnv("PORT", "8090"),
		N8NBaseURL:    getEnv("N8N_BASE_URL", "http://localhost:5678"),
		WindmillURL:   getEnv("WINDMILL_BASE_URL", "http://localhost:8000"),
		PostgresURL:   getEnv("POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/resume_screening"),
		QdrantURL:     getEnv("QDRANT_URL", "http://localhost:6333"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:  "healthy",
		Service: "resume-screening-assistant-api",
		Version: "1.0.0",
		Time:    time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func jobsHandler(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, this would query the database or call n8n
	// For now, return mock data to demonstrate the endpoint
	mockJobs := []interface{}{
		map[string]interface{}{
			"id":          1,
			"job_title":   "Senior Software Engineer",
			"company_name": "Tech Innovations Inc",
			"location":    "San Francisco, CA",
			"experience_required": 5,
			"candidate_count": 12,
			"status": "active",
		},
		map[string]interface{}{
			"id":          2,
			"job_title":   "Data Scientist",
			"company_name": "AI Solutions Corp",
			"location":    "New York, NY",
			"experience_required": 3,
			"candidate_count": 8,
			"status": "active",
		},
		map[string]interface{}{
			"id":          3,
			"job_title":   "Product Manager",
			"company_name": "StartupX",
			"location":    "Remote",
			"experience_required": 4,
			"candidate_count": 15,
			"status": "active",
		},
	}

	response := JobsResponse{
		Success: true,
		Jobs:    mockJobs,
		Count:   len(mockJobs),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func candidatesHandler(w http.ResponseWriter, r *http.Request) {
	// Parse job_id query parameter if provided
	jobID := r.URL.Query().Get("job_id")
	
	// Mock candidates data
	mockCandidates := []interface{}{
		map[string]interface{}{
			"id":             1,
			"candidate_name": "Alice Johnson",
			"email":          "alice.johnson@email.com",
			"score":          0.92,
			"experience_years": 6,
			"parsed_skills":  []string{"Python", "Machine Learning", "PostgreSQL", "Docker"},
			"status":         "reviewed",
			"job_id":         1,
		},
		map[string]interface{}{
			"id":             2,
			"candidate_name": "Bob Smith",
			"email":          "bob.smith@email.com",
			"score":          0.85,
			"experience_years": 4,
			"parsed_skills":  []string{"JavaScript", "React", "Node.js", "AWS"},
			"status":         "pending",
			"job_id":         1,
		},
		map[string]interface{}{
			"id":             3,
			"candidate_name": "Carol Davis",
			"email":          "carol.davis@email.com",
			"score":          0.88,
			"experience_years": 5,
			"parsed_skills":  []string{"Python", "Data Science", "TensorFlow", "SQL"},
			"status":         "interviewed",
			"job_id":         2,
		},
	}

	// Filter by job_id if provided
	if jobID != "" {
		if id, err := strconv.Atoi(jobID); err == nil {
			filteredCandidates := make([]interface{}, 0)
			for _, candidate := range mockCandidates {
				if candidateMap, ok := candidate.(map[string]interface{}); ok {
					if candidateJobID, exists := candidateMap["job_id"]; exists {
						if candidateJobID == id {
							filteredCandidates = append(filteredCandidates, candidate)
						}
					}
				}
			}
			mockCandidates = filteredCandidates
		}
	}

	response := CandidatesResponse{
		Success:    true,
		Candidates: mockCandidates,
		Count:      len(mockCandidates),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	searchType := r.URL.Query().Get("type")
	if searchType == "" {
		searchType = "both"
	}

	// Mock search results
	mockResults := []interface{}{}
	
	if query != "" {
		// Simulate semantic search results
		if searchType == "both" || searchType == "candidates" {
			mockResults = append(mockResults, map[string]interface{}{
				"type":           "candidate",
				"id":             1,
				"candidate_name": "Alice Johnson",
				"score":          0.95,
				"skills":         []string{"Python", "Machine Learning", "PostgreSQL"},
				"experience_years": 6,
				"relevance":      "High match for technical skills",
			})
		}
		
		if searchType == "both" || searchType == "jobs" {
			mockResults = append(mockResults, map[string]interface{}{
				"type":         "job",
				"id":           1,
				"job_title":    "Senior Software Engineer",
				"company_name": "Tech Innovations Inc",
				"score":        0.87,
				"location":     "San Francisco, CA",
				"relevance":    "Strong technical requirements match",
			})
		}
	}

	response := SearchResponse{
		Success: true,
		Results: mockResults,
		Count:   len(mockResults),
		Query:   query,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func setupRoutes(config *Config) *mux.Router {
	r := mux.NewRouter()

	// Health endpoint
	r.HandleFunc("/health", healthHandler).Methods("GET")

	// API endpoints
	r.HandleFunc("/api/jobs", jobsHandler).Methods("GET")
	r.HandleFunc("/api/candidates", candidatesHandler).Methods("GET")
	r.HandleFunc("/api/search", searchHandler).Methods("GET")

	// Info endpoint
	r.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		info := map[string]interface{}{
			"service":     "resume-screening-assistant-api",
			"version":     "1.0.0",
			"description": "Resume Screening Assistant coordination API",
			"endpoints": map[string]string{
				"health":     "/health",
				"jobs":       "/api/jobs",
				"candidates": "/api/candidates",
				"search":     "/api/search",
			},
			"resources": map[string]string{
				"n8n":       config.N8NBaseURL,
				"windmill":  config.WindmillURL,
				"postgres":  "Connected",
				"qdrant":    config.QdrantURL,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	}).Methods("GET")

	return r
}

func main() {
	config := loadConfig()
	
	log.Printf("Starting Resume Screening Assistant API server...")
	log.Printf("Port: %s", config.Port)
	log.Printf("n8n URL: %s", config.N8NBaseURL)
	log.Printf("Windmill URL: %s", config.WindmillURL)
	log.Printf("Qdrant URL: %s", config.QdrantURL)

	router := setupRoutes(config)

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(router)

	log.Printf("Server starting on port %s", config.Port)
	log.Printf("Health check: http://localhost:%s/health", config.Port)
	log.Printf("API endpoints: http://localhost:%s/api/", config.Port)

	if err := http.ListenAndServe(":"+config.Port, handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}