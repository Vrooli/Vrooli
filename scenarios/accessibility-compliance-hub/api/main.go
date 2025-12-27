package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
)

// NOTE: HealthResponse type removed - using api-core/health for standardized responses

type Scan struct {
	ID         string    `json:"id"`
	URL        string    `json:"url"`
	Status     string    `json:"status"`
	Created    time.Time `json:"created"`
	Violations int       `json:"violations"`
}

type Violation struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Element     string `json:"element"`
}

type Report struct {
	ID     string      `json:"id"`
	ScanID string      `json:"scan_id"`
	Title  string      `json:"title"`
	Score  float64     `json:"score"`
	Issues []Violation `json:"issues"`
	Date   time.Time   `json:"date"`
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "accessibility-compliance-hub",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	r := mux.NewRouter()

	// Health check endpoint using api-core/health for standardized response format
	healthHandler := health.New().
		Version("1.0.0").
		Handler()
	r.HandleFunc("/health", healthHandler).Methods("GET")

	// API endpoints
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/scans", scansHandler).Methods("GET")
	api.HandleFunc("/violations", violationsHandler).Methods("GET")
	api.HandleFunc("/reports", reportsHandler).Methods("GET")

	// Start server
	logger.Info("accessibility compliance hub api starting")
	if err := server.Run(server.Config{
		Handler: r,
	}); err != nil {
		logger.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}

// NOTE: The old healthHandler function has been replaced by api-core/health for standardized responses.

func scansHandler(w http.ResponseWriter, r *http.Request) {
	// Mock data for demonstration - in production this would come from database
	// URLs are example values for testing API structure
	scans := []Scan{
		{
			ID:         "scan-001",
			URL:        "https://example.com",
			Status:     "completed",
			Created:    time.Now().Add(-24 * time.Hour),
			Violations: 5,
		},
		{
			ID:         "scan-002",
			URL:        "https://test-site.com", // Example test URL for mock data
			Status:     "in_progress",
			Created:    time.Now().Add(-2 * time.Hour),
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
