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
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Services  map[string]string `json:"services"`
}

type Scan struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Status    string    `json:"status"`
	Created   time.Time `json:"created"`
	Violations int      `json:"violations"`
}

type Violation struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Element     string `json:"element"`
}

type Report struct {
	ID      string     `json:"id"`
	ScanID  string     `json:"scan_id"`
	Title   string     `json:"title"`
	Score   float64    `json:"score"`
	Issues  []Violation `json:"issues"`
	Date    time.Time  `json:"date"`
}

func main() {
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start accessibility-compliance-hub

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	port := getEnv("API_PORT", getEnv("PORT", "8080"))

	r := mux.NewRouter()

	// Health check endpoint
	r.HandleFunc("/health", healthHandler).Methods("GET")

	// API endpoints
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/scans", scansHandler).Methods("GET")
	api.HandleFunc("/violations", violationsHandler).Methods("GET")
	api.HandleFunc("/reports", reportsHandler).Methods("GET")

	// Start server
	log.Printf("🚀 Accessibility Compliance Hub API starting on port %s", port)
	log.Printf("🔍 Service endpoints:")
	log.Printf("   Health: http://localhost:%s/health", port)
	log.Printf("   Scans: http://localhost:%s/api/scans", port)
	log.Printf("   Violations: http://localhost:%s/api/violations", port)
	log.Printf("   Reports: http://localhost:%s/api/reports", port)

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
		"axe-core":      "healthy",
		"wave":          "healthy",
		"pa11y":         "healthy",
		"postgres":      "healthy",
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

func scansHandler(w http.ResponseWriter, r *http.Request) {
	// Mock data
	scans := []Scan{
		{
			ID:        "scan-001",
			URL:       "https://example.com",
			Status:    "completed",
			Created:   time.Now().Add(-24 * time.Hour),
			Violations: 5,
		},
		{
			ID:        "scan-002",
			URL:       "https://test-site.com",
			Status:    "in_progress",
			Created:   time.Now().Add(-2 * time.Hour),
			Violations: 0,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scans)
}

func violationsHandler(w http.ResponseWriter, r *http.Request) {
	// Mock data
	violations := []Violation{
		{
			ID:          "viol-001",
			Type:        "color-contrast",
			Description: "Insufficient color contrast",
			Severity:    "high",
			Element:     "button",
		},
		{
			ID:          "viol-002",
			Type:        "alt-text",
			Description: "Missing alt text on image",
			Severity:    "medium",
			Element:     "img",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(violations)
}

func reportsHandler(w http.ResponseWriter, r *http.Request) {
	// Mock data
	reports := []Report{
		{
			ID:     "report-001",
			ScanID: "scan-001",
			Title:  "Accessibility Audit Report",
			Score:  85.5,
			Issues: []Violation{
				{
					ID:          "viol-001",
					Type:        "color-contrast",
					Description: "Insufficient color contrast",
					Severity:    "high",
					Element:     "button",
				},
			},
			Date: time.Now().Add(-24 * time.Hour),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reports)
}
