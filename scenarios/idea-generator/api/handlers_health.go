package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// statusHandler handles service status requests
func (s *ApiServer) statusHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"service":   "idea-generator-api",
		"version":   "1.0.0",
		"timestamp": time.Now(),
		"uptime":    "running",
		"resources": map[string]string{
			"postgres":     "connected",
			"qdrant":       s.qdrantURL,
			"minio":        s.minioURL,
			"redis":        "connected",
			"ollama":       s.ollamaURL,
			"unstructured": s.unstructuredURL,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.Printf("Failed to encode status response: %v", err)
	}
}

// workflowsHandler handles workflow capability listing
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
	if err := json.NewEncoder(w).Encode(capabilities); err != nil {
		log.Printf("Failed to encode capabilities response: %v", err)
	}
}
