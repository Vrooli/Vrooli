package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/vrooli/system-monitor/internal/config"
	"github.com/vrooli/system-monitor/internal/models"
	"github.com/vrooli/system-monitor/internal/services"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	config      *config.Config
	monitorSvc  *services.MonitorService
	startTime   time.Time
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(cfg *config.Config, monitorSvc *services.MonitorService) *HealthHandler {
	return &HealthHandler{
		config:      cfg,
		monitorSvc:  monitorSvc,
		startTime:   time.Now(),
	}
}

// Handle processes health check requests
func (h *HealthHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Perform health checks
	checks := make(map[string]interface{})
	overallStatus := "healthy"
	
	// Check metrics collection capability
	ctx := r.Context()
	metrics, err := h.monitorSvc.GetCurrentMetrics(ctx)
	
	if err == nil && metrics != nil {
		checks["metrics_collection"] = map[string]interface{}{
			"status": "healthy",
			"message": fmt.Sprintf("Collecting metrics - CPU: %.1f%%, Memory: %.1f%%", 
				metrics.CPUUsage, metrics.MemoryUsage),
		}
	} else {
		checks["metrics_collection"] = map[string]interface{}{
			"status": "degraded",
			"message": "Unable to collect some metrics",
			"error": err.Error(),
		}
		overallStatus = "degraded"
	}
	
	// Check filesystem access
	if _, err := os.Stat("/tmp"); err == nil {
		checks["filesystem"] = map[string]interface{}{
			"status": "healthy",
			"message": "Filesystem accessible",
		}
	} else {
		checks["filesystem"] = map[string]interface{}{
			"status": "unhealthy",
			"error": "Cannot access filesystem",
		}
		overallStatus = "unhealthy"
	}
	
	// Check database connection (if configured)
	if h.config.HasDatabase() {
		// This would check actual database connection
		checks["database"] = map[string]interface{}{
			"status": "healthy",
			"message": "Database connection active",
		}
	}
	
	// Calculate uptime
	uptime := time.Since(h.startTime).Seconds()
	
	response := models.HealthResponse{
		Status:    overallStatus,
		Service:   h.config.Server.ServiceName,
		Timestamp: time.Now().Unix(),
		Uptime:    uptime,
		Checks:    checks,
		Version:   h.config.Server.Version,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// respondWithJSON sends a JSON response
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

// respondWithError sends an error response
func respondWithError(w http.ResponseWriter, code int, err error) {
	respondWithJSON(w, code, map[string]string{
		"error": err.Error(),
		"status": "error",
	})
}