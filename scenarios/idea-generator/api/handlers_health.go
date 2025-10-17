package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// healthHandler handles health check requests
func (s *ApiServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	// Check database connectivity
	dbConnected := true
	var dbLatency *float64
	if s.db != nil {
		start := time.Now()
		err := s.db.Ping()
		if err == nil {
			latency := float64(time.Since(start).Milliseconds())
			dbLatency = &latency
		} else {
			dbConnected = false
		}
	}

	dependencies := map[string]interface{}{
		"database": map[string]interface{}{
			"connected":  dbConnected,
			"latency_ms": dbLatency,
			"error":      nil,
		},
	}

	response := HealthResponse{
		Status:       "healthy",
		Service:      "idea-generator-api",
		Timestamp:    time.Now(),
		Readiness:    dbConnected,
		Version:      "1.0.0",
		Dependencies: dependencies,
	}

	if !dbConnected {
		response.Status = "degraded"
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Can't use http.Error after headers sent, just log
		// Note: logging package imported as "log"
		log.Printf("Failed to encode health response: %v", err)
	}
}

// statusHandler handles service status requests
func (s *ApiServer) statusHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"service":   "idea-generator-api",
		"version":   "1.0.0",
		"timestamp": time.Now(),
		"uptime":    "running",
		"resources": map[string]string{
			"windmill":     s.windmillURL,
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
