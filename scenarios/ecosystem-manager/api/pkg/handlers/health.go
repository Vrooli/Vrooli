package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ecosystem-manager/api/pkg/queue"
	"github.com/ecosystem-manager/api/pkg/recycler"
)

// HealthHandlers contains handlers for health check endpoints
type HealthHandlers struct {
	processor *queue.Processor
	recycler  *recycler.Recycler
	startTime time.Time
}

// NewHealthHandlers creates a new health handlers instance
func NewHealthHandlers(processor *queue.Processor, recycler *recycler.Recycler) *HealthHandlers {
	return &HealthHandlers{
		processor: processor,
		recycler:  recycler,
		startTime: time.Now(),
	}
}

// HealthCheckHandler returns service health status
func (h *HealthHandlers) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check queue directory health
	queuePath := filepath.Join("/home", os.Getenv("USER"), "Vrooli", "scenarios", "ecosystem-manager", "queue")
	storageHealthy := true
	var storageError map[string]any

	// Test if we can read the queue directory
	if _, err := os.Stat(queuePath); err != nil {
		storageHealthy = false
		storageError = map[string]any{
			"code":      "STORAGE_ACCESS_ERROR",
			"message":   fmt.Sprintf("Cannot access queue directory: %v", err),
			"category":  "resource",
			"retryable": true,
		}
	}

	// Get queue status
	queueStatus := h.processor.GetQueueStatus()

	// Get memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Overall service status
	status := "healthy"
	if !storageHealthy {
		status = "degraded"
	}

	healthResponse := map[string]any{
		"status":    status,
		"service":   "ecosystem-manager-api",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"readiness": true, // Service is ready to accept requests
		"version":   "2.0.0",
		"dependencies": map[string]any{
			"storage": map[string]any{
				"connected": storageHealthy,
				"type":      "yaml-files",
				"path":      queuePath,
			},
			"queue_processor": map[string]any{
				"connected": true,
				"active":    queueStatus["processor_active"],
				"state":     queueStatus["maintenance_state"],
			},
		},
		"metrics": map[string]any{
			"uptime_seconds":  time.Since(h.startTime).Seconds(),
			"goroutines":      runtime.NumGoroutine(),
			"memory_mb":       memStats.Alloc / 1024 / 1024,
			"gc_cycles":       memStats.NumGC,
			"executing_tasks": queueStatus["executing_count"],
			"available_slots": queueStatus["available_slots"],
			"pending_count":   queueStatus["pending_count"],
			"completed_count": queueStatus["completed_count"],
			"failed_count":    queueStatus["failed_count"],
		},
	}

	// Recycler stats (if available)
	if h.recycler != nil {
		stats := h.recycler.Stats()
		healthResponse["dependencies"].(map[string]any)["recycler"] = map[string]any{
			"connected": true,
			"enqueued":  stats.Enqueued,
			"processed": stats.Processed,
			"dropped":   stats.Dropped,
			"requeued":  stats.Requeued,
		}
	}

	// Add storage error if present
	if storageError != nil {
		healthResponse["dependencies"].(map[string]any)["storage"].(map[string]any)["error"] = storageError
	} else {
		healthResponse["dependencies"].(map[string]any)["storage"].(map[string]any)["error"] = nil
	}

	writeJSON(w, healthResponse, http.StatusOK)
}
