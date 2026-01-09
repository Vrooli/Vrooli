package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/ecosystem-manager/api/pkg/queue"
	"github.com/ecosystem-manager/api/pkg/recycler"
)

// HealthHandlers contains handlers for health check endpoints
type HealthHandlers struct {
	processor *queue.Processor
	recycler  *recycler.Recycler
	queueDir  string
	db        *sql.DB
	version   string
	startTime time.Time
}

// NewHealthHandlers creates a new health handlers instance
func NewHealthHandlers(processor *queue.Processor, recycler *recycler.Recycler, queueDir string, db *sql.DB, version string) *HealthHandlers {
	return &HealthHandlers{
		processor: processor,
		recycler:  recycler,
		queueDir:  queueDir,
		db:        db,
		version:   version,
		startTime: time.Now(),
	}
}

// HealthCheckHandler returns service health status
func (h *HealthHandlers) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check queue directory health
	queuePath := h.queueDir
	storageHealthy := true
	var storageError map[string]any

	if strings.TrimSpace(queuePath) == "" {
		storageHealthy = false
		storageError = map[string]any{
			"code":      "STORAGE_PATH_MISSING",
			"message":   "Queue directory not configured",
			"category":  "resource",
			"retryable": false,
		}
	} else if _, err := os.Stat(queuePath); err != nil { // Test if we can read the queue directory
		storageHealthy = false
		storageError = map[string]any{
			"code":      "STORAGE_ACCESS_ERROR",
			"message":   fmt.Sprintf("Cannot access queue directory: %v", err),
			"category":  "resource",
			"retryable": true,
		}
	}

	// Check database health (if configured)
	dbHealthy := true
	dbChecked := false
	var dbError map[string]any
	if h.db != nil {
		dbChecked = true
		ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
		defer cancel()
		if err := h.db.PingContext(ctx); err != nil {
			dbHealthy = false
			dbError = map[string]any{
				"code":      "DB_PING_FAILED",
				"message":   fmt.Sprintf("Database ping failed: %v", err),
				"category":  "database",
				"retryable": true,
			}
		}
	}

	// Get queue status
	queueStatus := map[string]any{}
	if h.processor != nil {
		queueStatus = h.processor.GetQueueStatus()
	}

	// Get memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Overall service status
	status := "healthy"
	if !storageHealthy || (dbChecked && !dbHealthy) {
		status = "degraded"
	}

	healthResponse := map[string]any{
		"status":    status,
		"service":   "ecosystem-manager-api",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"readiness": true, // Service is ready to accept requests
		"version":   h.version,
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
			"database": map[string]any{
				"connected": dbHealthy && dbChecked,
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

	// Add database error if present
	if dbChecked {
		if dbError != nil {
			healthResponse["dependencies"].(map[string]any)["database"].(map[string]any)["error"] = dbError
		} else {
			healthResponse["dependencies"].(map[string]any)["database"].(map[string]any)["error"] = nil
		}
	} else {
		healthResponse["dependencies"].(map[string]any)["database"].(map[string]any)["error"] = map[string]any{
			"code":      "DB_NOT_CONFIGURED",
			"message":   "Database not configured for health checks",
			"category":  "database",
			"retryable": false,
		}
	}

	writeJSON(w, healthResponse, http.StatusOK)
}
