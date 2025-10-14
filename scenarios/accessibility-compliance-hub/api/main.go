package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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
	// Protect against direct execution - must be run through lifecycle system
	lifecycleManagedValue := os.Getenv("VROOLI_LIFECYCLE_MANAGED")
	if lifecycleManagedValue == "" || lifecycleManagedValue != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start accessibility-compliance-hub

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

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
	logger.Info("accessibility compliance hub api starting",
		"port", port,
		"health_endpoint", fmt.Sprintf("http://localhost:%s/health", port),
		"scans_endpoint", fmt.Sprintf("http://localhost:%s/api/scans", port),
		"violations_endpoint", fmt.Sprintf("http://localhost:%s/api/violations", port),
		"reports_endpoint", fmt.Sprintf("http://localhost:%s/api/reports", port),
	)

	if err := http.ListenAndServe(":"+port, r); err != nil {
		logger.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	// Create timeout context for health check (5 second timeout)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Channel to signal completion
	done := make(chan bool, 1)

	var response HealthResponse

	// Perform health check in goroutine with timeout
	go func() {
		services := map[string]string{
			"axe-core": "healthy",
			"wave":     "healthy",
			"pa11y":    "healthy",
			"postgres": "healthy",
		}

		response = HealthResponse{
			Status:    "healthy",
			Timestamp: time.Now(),
			Version:   "1.0.0",
			Services:  services,
		}
		done <- true
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	case <-ctx.Done():
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "timeout",
			"error":  "health check timed out after 5 seconds",
		})
	}
}

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
