package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"workspace-sandbox/internal/driver"
)

// APIInfo handles requests to the root and /api paths with helpful API documentation.
// This improves discoverability for developers who don't know the versioned API path.
func (h *Handlers) APIInfo(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"service":     "Workspace Sandbox API",
		"version":     Version,
		"description": "Copy-on-write workspaces for safe code modification",
		"apiVersion":  "v1",
		"basePath":    "/api/v1",
		"endpoints": map[string]string{
			"health":    "/api/v1/health",
			"sandboxes": "/api/v1/sandboxes",
			"driver":    "/api/v1/driver/info",
			"stats":     "/api/v1/stats",
			"metrics":   "/api/v1/metrics",
		},
		"documentation": "https://github.com/vrooli/workspace-sandbox",
	}
	h.JSONSuccess(w, response)
}

// Health handles health check requests.
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	status := "healthy"
	dbStatus := "connected"

	if err := h.DB.PingContext(r.Context()); err != nil {
		status = "unhealthy"
		dbStatus = "disconnected"
	}

	// Check driver availability using injected driver
	driverAvailable, _ := h.Driver().IsAvailable(r.Context())
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
	available, availErr := h.Driver().IsAvailable(r.Context())

	info := driver.Info{
		Type:        h.Driver().Type(),
		Version:     h.Driver().Version(),
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
	resp := driver.GetDriverOptions(r.Context(), h.Driver().Type(), h.InUserNamespace)
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

// SelectDriverRequest is the request body for selecting a driver.
type SelectDriverRequest struct {
	// DriverID is the ID of the driver to select (e.g., "fuse-overlayfs", "overlayfs-userns")
	DriverID string `json:"driverId"`
}

// SelectDriverResponse is the response from selecting a driver.
type SelectDriverResponse struct {
	Success         bool   `json:"success"`
	SelectedDriver  string `json:"selectedDriver"`
	RequiresRestart bool   `json:"requiresRestart"`
	Message         string `json:"message"`
}

// SelectDriver handles setting the preferred driver.
// This endpoint allows users to select which driver to use.
// The driver is hot-swapped immediately without requiring an API restart.
// In-flight operations continue with the old driver; new operations use the new driver.
func (h *Handlers) SelectDriver(w http.ResponseWriter, r *http.Request) {
	var req SelectDriverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.DriverID == "" {
		h.JSONError(w, "driverId is required", http.StatusBadRequest)
		return
	}

	// Get available options to validate the request
	options := driver.GetDriverOptions(r.Context(), h.Driver().Type(), h.InUserNamespace)

	// Find the requested driver option
	var selectedOption *driver.DriverOption
	for i := range options.Options {
		if string(options.Options[i].ID) == req.DriverID {
			selectedOption = &options.Options[i]
			break
		}
	}

	if selectedOption == nil {
		h.JSONError(w, "unknown driver: "+req.DriverID, http.StatusBadRequest)
		return
	}

	if !selectedOption.Available {
		// Build helpful error message
		var unmetReqs []string
		for _, r := range selectedOption.Requirements {
			if !r.Met {
				unmetReqs = append(unmetReqs, r.Name)
			}
		}
		h.JSONError(w, "driver not available - unmet requirements: "+join(unmetReqs, ", "), http.StatusBadRequest)
		return
	}

	// Check if this is a change from current
	currentDriverID := string(options.CurrentDriver)
	isChange := currentDriverID != req.DriverID

	if !isChange {
		// Already using this driver
		h.JSONSuccess(w, SelectDriverResponse{
			Success:         true,
			SelectedDriver:  req.DriverID,
			RequiresRestart: false,
			Message:         "Already using " + req.DriverID + ".",
		})
		return
	}

	// Hot-swap the driver using the manager
	// This atomically switches to the new driver and saves the preference
	if err := h.DriverManager.Switch(r.Context(), driver.DriverOptionID(req.DriverID)); err != nil {
		h.JSONError(w, "failed to switch driver: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.JSONSuccess(w, SelectDriverResponse{
		Success:         true,
		SelectedDriver:  req.DriverID,
		RequiresRestart: false,
		Message:         "Driver switched to " + req.DriverID + ". Change is active immediately.",
	})
}

// GetDriverPreference handles getting the current driver preference.
func (h *Handlers) GetDriverPreference(w http.ResponseWriter, r *http.Request) {
	pref, err := driver.LoadDriverPreference(h.Config.Driver.BaseDir)
	if err != nil {
		// No preference set - return current driver
		pref = string(h.Driver().Type())
	}

	options := driver.GetDriverOptions(r.Context(), h.Driver().Type(), h.InUserNamespace)

	response := map[string]interface{}{
		"preference":    pref,
		"currentDriver": string(options.CurrentDriver),
		"isActive":      pref == string(options.CurrentDriver) || pref == "",
	}

	h.JSONSuccess(w, response)
}

// join is a simple helper to join strings with a separator.
func join(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
