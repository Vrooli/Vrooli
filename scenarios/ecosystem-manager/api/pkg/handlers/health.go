package handlers

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/ecosystem-manager/api/pkg/queue"
)

// HealthHandlers contains handlers for health check endpoints
type HealthHandlers struct {
	processor *queue.Processor
	startTime time.Time
}

// NewHealthHandlers creates a new health handlers instance
func NewHealthHandlers(processor *queue.Processor) *HealthHandlers {
	return &HealthHandlers{
		processor: processor,
		startTime: time.Now(),
	}
}

// HealthCheckHandler returns service health status
func (h *HealthHandlers) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(h.startTime)

	// Get queue status
	queueStatus := h.processor.GetQueueStatus()

	// Get memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	status := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"uptime":    uptime.String(),
		"version":   "1.0.0",
		"service":   "ecosystem-manager-api",
		"queue": map[string]interface{}{
			"processor_active":  queueStatus["processor_active"],
			"maintenance_state": queueStatus["maintenance_state"],
			"executing_count":   queueStatus["executing_count"],
			"available_slots":   queueStatus["available_slots"],
		},
		"system": map[string]interface{}{
			"goroutines": runtime.NumGoroutine(),
			"memory_mb":  memStats.Alloc / 1024 / 1024,
			"gc_cycles":  memStats.NumGC,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
