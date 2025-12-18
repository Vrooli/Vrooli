package handlers

import (
	"net/http"
	"time"

	"workspace-sandbox/internal/driver"
)

// Health handles health check requests.
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	status := "healthy"
	dbStatus := "connected"

	if err := h.DB.PingContext(r.Context()); err != nil {
		status = "unhealthy"
		dbStatus = "disconnected"
	}

	// Check driver availability using injected driver
	driverAvailable, _ := h.Driver.IsAvailable(r.Context())
	driverStatus := "available"
	if !driverAvailable {
		driverStatus = "unavailable"
	}

	response := map[string]interface{}{
		"status":    status,
		"service":   "Workspace Sandbox API",
		"version":   Version,
		"readiness": status == "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"dependencies": map[string]string{
			"database": dbStatus,
			"driver":   driverStatus,
		},
		"config": map[string]interface{}{
			"projectRoot": h.Config.Driver.ProjectRoot,
		},
	}

	h.JSONSuccess(w, response)
}

// DriverInfo handles getting driver information.
func (h *Handlers) DriverInfo(w http.ResponseWriter, r *http.Request) {
	available, availErr := h.Driver.IsAvailable(r.Context())

	info := driver.Info{
		Type:        h.Driver.Type(),
		Version:     h.Driver.Version(),
		Description: "Linux overlayfs driver for copy-on-write sandboxes",
		Available:   available,
	}

	response := map[string]interface{}{
		"driver":    info,
		"available": available,
	}
	if availErr != nil {
		response["error"] = availErr.Error()
	}

	h.JSONSuccess(w, response)
}

// DriverOptions handles getting all available driver options with their requirements.
// This endpoint is used by the UI settings dialog to show driver configuration options.
func (h *Handlers) DriverOptions(w http.ResponseWriter, r *http.Request) {
	resp := driver.GetDriverOptions(r.Context(), h.Driver.Type(), h.InUserNamespace)
	h.JSONSuccess(w, resp)
}

// Stats handles getting aggregate sandbox statistics.
// This endpoint supports dashboard metrics and monitoring.
func (h *Handlers) Stats(w http.ResponseWriter, r *http.Request) {
	if h.StatsGetter == nil {
		h.JSONError(w, "stats not available", http.StatusServiceUnavailable)
		return
	}

	stats, err := h.StatsGetter.GetStats(r.Context())
	if h.HandleDomainError(w, err) {
		return
	}

	// Include timestamp for cache invalidation hints
	response := map[string]interface{}{
		"stats":     stats,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	h.JSONSuccess(w, response)
}
