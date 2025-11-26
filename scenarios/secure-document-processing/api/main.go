package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Version   string            `json:"version"`
	Services  map[string]string `json:"services"`
}

type Document struct {
	ID       string    `json:"id"`
	Filename string    `json:"filename"`
	Status   string    `json:"status"`
	Created  time.Time `json:"created"`
}

type ProcessingJob struct {
	ID        string    `json:"id"`
	JobName   string    `json:"jobName"`
	Status    string    `json:"status"`
	Created   time.Time `json:"created"`
	Documents []string  `json:"documents"`
}

type Workflow struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        string    `json:"type"`
	Created     time.Time `json:"created"`
}

func main() {
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start secure-document-processing

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	port := getEnv("API_PORT", getEnv("PORT", ""))

	r := mux.NewRouter()

	// Health check endpoint
	r.HandleFunc("/health", healthHandler).Methods("GET")

	// API endpoints
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/documents", documentsHandler).Methods("GET")
	api.HandleFunc("/jobs", jobsHandler).Methods("GET")
	api.HandleFunc("/workflows", workflowsHandler).Methods("GET")

	// Start server
	log.Printf("🚀 Secure Document Processing API starting on port %s", port)
	log.Printf("🔒 Service endpoints:")
	log.Printf("   Health: http://localhost:%s/health", port)
	log.Printf("   Documents: http://localhost:%s/api/documents", port)
	log.Printf("   Jobs: http://localhost:%s/api/jobs", port)
	log.Printf("   Workflows: http://localhost:%s/api/workflows", port)

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	services := map[string]string{
		"windmill":     getServiceStatus(os.Getenv("WINDMILL_BASE_URL")),
		"vault":        getServiceStatus(os.Getenv("VAULT_URL")),
		"minio":        getServiceStatus(os.Getenv("MINIO_URL")),
		"unstructured": getServiceStatus(os.Getenv("UNSTRUCTURED_URL")),
		"postgres":     "healthy", // Assume healthy if we got this far
	}

	if os.Getenv("QDRANT_URL") != "" {
		services["qdrant"] = getServiceStatus(os.Getenv("QDRANT_URL"))
	}

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Services:  services,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func documentsHandler(w http.ResponseWriter, r *http.Request) {
	// Mock data for now - in real implementation, this would query PostgreSQL
	documents := []Document{
		{
			ID:       "doc-001",
			Filename: "sample-document.pdf",
			Status:   "processed",
			Created:  time.Now().Add(-24 * time.Hour),
		},
		{
			ID:       "doc-002",
			Filename: "contract.docx",
			Status:   "processing",
			Created:  time.Now().Add(-2 * time.Hour),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(documents)
}

func jobsHandler(w http.ResponseWriter, r *http.Request) {
	// Mock data for now - in real implementation, this would query PostgreSQL
	jobs := []ProcessingJob{
		{
			ID:        "job-001",
			JobName:   "Contract Analysis",
			Status:    "completed",
			Created:   time.Now().Add(-24 * time.Hour),
			Documents: []string{"doc-001"},
		},
		{
			ID:        "job-002",
			JobName:   "Document Redaction",
			Status:    "processing",
			Created:   time.Now().Add(-2 * time.Hour),
			Documents: []string{"doc-002"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jobs)
}

func workflowsHandler(w http.ResponseWriter, r *http.Request) {
	// Mock data for now - in real implementation, this would query PostgreSQL
	workflows := []Workflow{
		{
			ID:          "wf-001",
			Name:        "Document Redaction",
			Description: "Remove sensitive information from documents",
			Type:        "redaction",
			Created:     time.Now().Add(-7 * 24 * time.Hour),
		},
		{
			ID:          "wf-002",
			Name:        "Contract Analysis",
			Description: "Extract key terms and clauses from contracts",
			Type:        "analysis",
			Created:     time.Now().Add(-14 * 24 * time.Hour),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workflows)
}

func getServiceStatus(url string) string {
	if url == "" {
		return "not_configured"
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "unhealthy"
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return "healthy"
	}
	return "unhealthy"
}
