package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type HealthHandlers struct {
	db *sql.DB
}

func NewHealthHandlers(db *sql.DB) *HealthHandlers {
	return &HealthHandlers{db: db}
}

// RegisterRoutes mounts the health endpoints for both base and API prefixed routers.
func (h *HealthHandlers) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/health", h.Health).Methods("GET")
}

func (h *HealthHandlers) Health(w http.ResponseWriter, r *http.Request) {
	// Check database connectivity
	dbConnected := false
	var dbLatency float64
	var dbError map[string]interface{}

	if h.db != nil {
		start := time.Now()
		err := h.db.Ping()
		dbLatency = float64(time.Since(start).Milliseconds())

		if err == nil {
			dbConnected = true
		} else {
			dbError = map[string]interface{}{
				"code":      "CONNECTION_FAILED",
				"message":   err.Error(),
				"category":  "resource",
				"retryable": true,
			}
		}
	} else {
		dbError = map[string]interface{}{
			"code":      "NOT_INITIALIZED",
			"message":   "Database connection not initialized",
			"category":  "configuration",
			"retryable": false,
		}
	}

	// Determine overall status
	status := "healthy"
	readiness := true
	var statusNotes []string

	if !dbConnected {
		status = "degraded"
		statusNotes = append(statusNotes, "Database connectivity issues")
	}

	// Build response compliant with health-api.schema.json
	response := map[string]interface{}{
		"status":    status,
		"service":   "secrets-manager-api",
		"timestamp": time.Now().Format(time.RFC3339),
		"readiness": readiness,
		"version":   "1.0.0",
		"dependencies": map[string]interface{}{
			"database": map[string]interface{}{
				"connected":  dbConnected,
				"latency_ms": dbLatency,
				"error":      dbError,
			},
		},
	}

	if len(statusNotes) > 0 {
		response["status_notes"] = statusNotes
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
