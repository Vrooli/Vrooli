// Package health provides health check endpoints for the deployment manager.
package health

import (
	"net/http"

	"github.com/vrooli/api-core/health"
)

// DBPinger defines the interface for checking database connectivity.
type DBPinger interface {
	Ping() error
}

// Handler handles health check requests.
type Handler struct {
	healthFunc http.HandlerFunc
}

// NewHandler creates a new health handler using the standardized api-core/health package.
func NewHandler(db DBPinger) *Handler {
	healthHandler := health.New().Version("1.0.0").Check(health.DB(db), health.Critical).Handler()
	return &Handler{healthFunc: healthHandler}
}

// Health returns basic runtime health status using the standardized api-core/health package.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	h.healthFunc(w, r)
}
