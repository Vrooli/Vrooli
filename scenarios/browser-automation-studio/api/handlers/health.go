package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/vrooli/browser-automation-studio/constants"
)

// startTime records when the handler package was initialized for uptime calculation.
var startTime = time.Now()

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

	// Check automation engine health
	var automationHealthy bool
	var automationError map[string]any

	if h.executionService != nil {
		ok, err := h.executionService.CheckAutomationHealth(ctx)
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

	// Check storage health (MinIO)
	var storageHealthy bool
	var storageLatency float64
	var storageError map[string]any

	storageStart := time.Now()
	if h.storage != nil {
		if err := h.storage.HealthCheck(ctx); err != nil {
			storageLatency = float64(time.Since(storageStart).Nanoseconds()) / 1e6
			storageHealthy = false
			storageError = map[string]any{
				"code":      "STORAGE_CONNECTION_ERROR",
				"message":   fmt.Sprintf("Storage health check failed: %v", err),
				"category":  "resource",
				"retryable": true,
			}
		} else {
			storageLatency = float64(time.Since(storageStart).Nanoseconds()) / 1e6
			storageHealthy = true
		}
	} else {
		storageHealthy = false
		storageError = map[string]any{
			"code":      "STORAGE_NOT_INITIALIZED",
			"message":   "Storage client not initialized - screenshot/artifact storage unavailable",
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
	} else if !automationHealthy || !storageHealthy {
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
		"storage": map[string]any{
			"connected":  storageHealthy,
			"latency_ms": storageLatency,
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
	if storageError != nil {
		dependencies["storage"].(map[string]any)["error"] = storageError
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
		Metrics:      buildHealthMetrics(),
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

// buildHealthMetrics returns runtime metrics for the health endpoint.
// These metrics help operators understand system resource usage at a glance.
func buildHealthMetrics() map[string]any {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	uptime := time.Since(startTime)

	return map[string]any{
		"goroutines":      runtime.NumGoroutine(),
		"uptime_seconds":  int64(uptime.Seconds()),
		"heap_alloc_mb":   float64(memStats.HeapAlloc) / (1024 * 1024),
		"heap_inuse_mb":   float64(memStats.HeapInuse) / (1024 * 1024),
		"gc_pause_total_ms": float64(memStats.PauseTotalNs) / 1e6,
	}
}
