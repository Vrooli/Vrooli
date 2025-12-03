// Package health provides health check endpoints for the deployment manager.
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// DBPinger defines the interface for checking database connectivity.
type DBPinger interface {
	PingContext(ctx context.Context) error
}

// Handler handles health check requests.
type Handler struct {
	db DBPinger
}

// NewHandler creates a new health handler.
func NewHandler(db DBPinger) *Handler {
	return &Handler{db: db}
}

// Health returns basic runtime health status.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	status := "healthy"
	dbStatus := "connected"

	if err := h.db.PingContext(r.Context()); err != nil {
		status = "unhealthy"
		dbStatus = "disconnected"
	}

	response := map[string]interface{}{
		"status":    status,
		"service":   "Deployment Manager API",
		"version":   "1.0.0",
		"readiness": status == "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"dependencies": map[string]string{
			"database": dbStatus,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
