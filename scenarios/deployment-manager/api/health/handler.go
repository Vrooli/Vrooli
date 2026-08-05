// Package health provides health check endpoints for the deployment manager.
package health

import (
	"context"
	"errors"
	"net/http"

	"github.com/vrooli/api-core/health"
)

// Handler handles health check requests.
type Handler struct {
	healthFunc http.HandlerFunc
}

// NewHandler creates a new health handler using the standardized api-core/health package.
func NewHandler(db interface{ PingContext(context.Context) error }) *Handler {
		healthHandler := health.New().Version("1.0.0").Check(health.Func("database", func(ctx context.Context) error {
		if db == nil {
			return errors.New("database is not configured")
		}
		return db.PingContext(ctx)
	}), health.Critical).Handler()
	return &Handler{healthFunc: healthHandler}
}

// Health returns basic runtime health status using the standardized api-core/health package.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	h.healthFunc(w, r)
}
