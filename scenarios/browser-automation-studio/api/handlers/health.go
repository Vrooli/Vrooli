package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Health handles GET /health and GET /api/v1/health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Check database health
	var databaseHealthy bool
	var databaseLatency float64
	var databaseError map[string]interface{}

	start := time.Now()
	if h.repo != nil {
		// Try a simple database operation to check health
		_, err := h.repo.ListProjects(ctx, 1, 0)
		databaseLatency = float64(time.Since(start).Nanoseconds()) / 1e6 // Convert to milliseconds

		if err != nil {
			databaseHealthy = false
			databaseError = map[string]interface{}{
				"code":      "DATABASE_CONNECTION_ERROR",
				"message":   fmt.Sprintf("Database health check failed: %v", err),
				"category":  "resource",
				"retryable": true,
			}
		} else {
			databaseHealthy = true
		}
	} else {
		databaseHealthy = false
		databaseError = map[string]interface{}{
			"code":      "REPOSITORY_NOT_INITIALIZED",
			"message":   "Database repository not initialized",
			"category":  "internal",
			"retryable": false,
		}
	}

	// Check browserless health
	var browserlessHealthy bool
	var browserlessError map[string]interface{}

	if h.browserless != nil {
		if err := h.browserless.CheckBrowserlessHealth(); err != nil {
			browserlessHealthy = false
			browserlessError = map[string]interface{}{
				"code":      "BROWSERLESS_CONNECTION_ERROR",
				"message":   fmt.Sprintf("Browserless health check failed: %v", err),
				"category":  "resource",
				"retryable": true,
			}
		} else {
			browserlessHealthy = true
		}
	} else {
		browserlessHealthy = false
		browserlessError = map[string]interface{}{
			"code":      "BROWSERLESS_NOT_INITIALIZED",
			"message":   "Browserless client not initialized",
			"category":  "internal",
			"retryable": false,
		}
	}

	// Overall service status
	status := "healthy"
	readiness := true

	if !databaseHealthy {
		status = "unhealthy"
		readiness = false
	} else if !browserlessHealthy {
		status = "degraded"
		// Keep readiness true for degraded state - we can still serve some requests
	}

	// Build dependencies map
	dependencies := map[string]interface{}{
		"database": map[string]interface{}{
			"connected":  databaseHealthy,
			"latency_ms": databaseLatency,
			"error":      nil,
		},
		"external_services": []map[string]interface{}{
			{
				"name":      "browserless",
				"connected": browserlessHealthy,
				"error":     nil,
			},
		},
	}

	// Add errors if present
	if databaseError != nil {
		dependencies["database"].(map[string]interface{})["error"] = databaseError
	}
	if browserlessError != nil {
		dependencies["external_services"].([]map[string]interface{})[0]["error"] = browserlessError
	}

	response := HealthResponse{
		Status:       status,
		Service:      "browser-automation-studio-api",
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Readiness:    readiness,
		Version:      "1.0.0",
		Dependencies: dependencies,
		Metrics: map[string]interface{}{
			"goroutines": 0, // Could be populated with runtime.NumGoroutine() if needed
		},
	}

	// Set appropriate HTTP status code
	statusCode := http.StatusOK
	if status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
