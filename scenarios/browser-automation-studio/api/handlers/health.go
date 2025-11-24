package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/vrooli/browser-automation-studio/constants"
)

// Health handles GET /health and GET /api/v1/health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	// Check database health
	var databaseHealthy bool
	var databaseLatency float64
	var databaseError map[string]any

	start := time.Now()
	if h.repo != nil {
		// Try a simple database operation to check health
		_, err := h.repo.ListProjects(ctx, 1, 0)
		databaseLatency = float64(time.Since(start).Nanoseconds()) / 1e6 // Convert to milliseconds

		if err != nil {
			databaseHealthy = false
			databaseError = map[string]any{
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
		databaseError = map[string]any{
			"code":      "REPOSITORY_NOT_INITIALIZED",
			"message":   "Database repository not initialized",
			"category":  "internal",
			"retryable": false,
		}
	}

	// Check browserless health
	var automationHealthy bool
	var automationError map[string]any

	if h.workflowService != nil {
		ok, err := h.workflowService.CheckAutomationHealth(ctx)
		if err != nil {
			automationError = map[string]any{
				"code":      "AUTOMATION_ENGINE_ERROR",
				"message":   fmt.Sprintf("Automation engine health check failed: %v", err),
				"category":  "resource",
				"retryable": true,
			}
		}
		automationHealthy = ok
	} else {
		automationError = map[string]any{
			"code":      "AUTOMATION_ENGINE_NOT_INITIALIZED",
			"message":   "Automation workflow service not initialized",
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
	} else if !automationHealthy {
		status = "degraded"
		// Keep readiness true for degraded state - we can still serve some requests
	}

	// Build dependencies map
	dependencies := map[string]any{
		"database": map[string]any{
			"connected":  databaseHealthy,
			"latency_ms": databaseLatency,
			"error":      nil,
		},
		"external_services": []map[string]any{
			{
				"name":      "automation_engine",
				"connected": automationHealthy,
				"error":     nil,
			},
		},
	}

	// Add errors if present
	if databaseError != nil {
		dependencies["database"].(map[string]any)["error"] = databaseError
	}
	if automationError != nil {
		dependencies["external_services"].([]map[string]any)[0]["error"] = automationError
	}

	response := HealthResponse{
		Status:       status,
		Service:      "browser-automation-studio-api",
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Readiness:    readiness,
		Version:      "1.0.0",
		Dependencies: dependencies,
		Metrics: map[string]any{
			"goroutines": 0, // Could be populated with runtime.NumGoroutine() if needed
		},
	}

	// Set appropriate HTTP status code
	statusCode := http.StatusOK
	if status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.log.WithError(err).Error("Failed to encode health response")
	}
}
