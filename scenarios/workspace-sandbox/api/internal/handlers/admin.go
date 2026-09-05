package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/driverpref"
	"workspace-sandbox/internal/sandbox"
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

// DriverInfo handles getting driver information.
func (h *Handlers) DriverInfo(w http.ResponseWriter, r *http.Request) {
	available, availErr := h.Driver().IsAvailable(r.Context())

	info := driver.Info{
		ID:          h.Driver().ID(),
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
	resp := driver.GetDriverOptions(r.Context(), h.Starter, h.Driver().ID(), h.InUserNamespace)
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
		"timestamp": h.Clock.Now().UTC().Format(time.RFC3339),
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
// Drivers that are available in the current process are hot-swapped immediately.
// Drivers that require a different outer launch mode persist the preference and
// return requiresRestart=true so the launcher can activate them on next boot.
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
	options := driver.GetDriverOptions(r.Context(), h.Starter, h.Driver().ID(), h.InUserNamespace)

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

	requestedID := driver.DriverID(req.DriverID)
	if requiresOuterLaunchRestart(requestedID, h.InUserNamespace) {
		if err := driverpref.Save(h.Config.Driver.BaseDir, requestedID); err != nil {
			h.JSONError(w, "failed to save driver preference: "+err.Error(), http.StatusInternalServerError)
			return
		}
		h.JSONSuccess(w, SelectDriverResponse{
			Success:         true,
			SelectedDriver:  req.DriverID,
			RequiresRestart: true,
			Message:         "Driver preference saved for " + req.DriverID + ". Restart workspace-sandbox so the launcher can activate it.",
		})
		return
	}

	// Hot-swap the driver via the slot. SwitchDriver does the
	// validate → store → persist sequence atomically.
	driverCfg := driver.Config{
		BaseDir:            h.Config.Driver.BaseDir,
		HomeOverlayBaseDir: h.Config.Driver.HomeOverlayBaseDir,
		MaxSandboxes:       h.Config.Limits.MaxSandboxes,
		MaxSizeMB:          h.Config.Limits.MaxSandboxSizeMB,
	}
	if err := driver.SwitchDriver(r.Context(), h.DriverSlot, driverCfg, driver.Deps{Clock: h.Clock, Mounter: h.Mounter, Starter: h.Starter}, requestedID); err != nil {
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

func requiresOuterLaunchRestart(id driver.DriverID, inUserNamespace bool) bool {
	return id == driver.DriverOverlayfsUserNS && !inUserNamespace
}

// GetDriverPreference handles getting the current driver preference.
func (h *Handlers) GetDriverPreference(w http.ResponseWriter, r *http.Request) {
	prefID, err := driverpref.Load(h.Config.Driver.BaseDir)
	pref := string(prefID)
	if errors.Is(err, driverpref.ErrNotFound) {
		// No preference set - return current driver
		pref = string(h.Driver().ID())
	} else if err != nil {
		h.JSONError(w, "failed to read driver preference: "+err.Error(), http.StatusInternalServerError)
		return
	}

	options := driver.GetDriverOptions(r.Context(), h.Starter, h.Driver().ID(), h.InUserNamespace)

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

// --- Execution Config Endpoints ---

// executionConfigResponse extends ExecutionConfig with policy settings
// that are conceptually "sandbox creation defaults" shown in the same UI tab.
type executionConfigResponse struct {
	config.ExecutionConfig
	DefaultNoLock bool `json:"defaultNoLock"`
}

// GetExecutionConfig returns the current execution configuration.
// GET /api/v1/config/execution
func (h *Handlers) GetExecutionConfig(w http.ResponseWriter, r *http.Request) {
	h.JSONSuccess(w, executionConfigResponse{
		ExecutionConfig: h.Config.Execution,
		DefaultNoLock:   h.Config.Policy.DefaultNoLock,
	})
}

// UpdateExecutionConfig updates execution configuration.
// PUT /api/v1/config/execution
// Note: This is an in-memory update. For persistence, the caller should
// also update environment variables or a config file.
func (h *Handlers) UpdateExecutionConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DefaultResourceLimits   config.ResourceLimitsConfig `json:"defaultResourceLimits"`
		MaxResourceLimits       config.ResourceLimitsConfig `json:"maxResourceLimits"`
		DefaultIsolationProfile string                      `json:"defaultIsolationProfile"`
		DefaultNoLock           bool                        `json:"defaultNoLock"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate limits (defaults don't exceed maximums)
	if err := validateResourceLimits(req.DefaultResourceLimits, req.MaxResourceLimits); err != nil {
		h.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate profile exists
	if h.ProfileStore != nil {
		if _, err := h.ProfileStore.Get(req.DefaultIsolationProfile); err != nil {
			h.JSONError(w, "invalid default profile: "+req.DefaultIsolationProfile, http.StatusBadRequest)
			return
		}
	}

	// Update in-memory config
	h.Config.Execution.DefaultResourceLimits = req.DefaultResourceLimits
	h.Config.Execution.MaxResourceLimits = req.MaxResourceLimits
	h.Config.Execution.DefaultIsolationProfile = req.DefaultIsolationProfile
	h.Config.Policy.DefaultNoLock = req.DefaultNoLock

	// Sync to service layer so new sandbox creates pick up the change
	if svc, ok := h.Service.(*sandbox.Service); ok {
		svc.SetDefaultNoLock(req.DefaultNoLock)
	}

	h.JSONSuccess(w, executionConfigResponse{
		ExecutionConfig: h.Config.Execution,
		DefaultNoLock:   h.Config.Policy.DefaultNoLock,
	})
}

// validateResourceLimits checks that defaults don't exceed maximums.
func validateResourceLimits(defaults, maxes config.ResourceLimitsConfig) error {
	if maxes.MemoryLimitMB > 0 && defaults.MemoryLimitMB > maxes.MemoryLimitMB {
		return fmt.Errorf("default memory limit (%d MB) exceeds maximum (%d MB)",
			defaults.MemoryLimitMB, maxes.MemoryLimitMB)
	}
	if maxes.CPUTimeSec > 0 && defaults.CPUTimeSec > maxes.CPUTimeSec {
		return fmt.Errorf("default CPU time (%d sec) exceeds maximum (%d sec)",
			defaults.CPUTimeSec, maxes.CPUTimeSec)
	}
	if maxes.MaxProcesses > 0 && defaults.MaxProcesses > maxes.MaxProcesses {
		return fmt.Errorf("default max processes (%d) exceeds maximum (%d)",
			defaults.MaxProcesses, maxes.MaxProcesses)
	}
	if maxes.MaxOpenFiles > 0 && defaults.MaxOpenFiles > maxes.MaxOpenFiles {
		return fmt.Errorf("default max open files (%d) exceeds maximum (%d)",
			defaults.MaxOpenFiles, maxes.MaxOpenFiles)
	}
	if maxes.TimeoutSec > 0 && defaults.TimeoutSec > maxes.TimeoutSec {
		return fmt.Errorf("default timeout (%d sec) exceeds maximum (%d sec)",
			defaults.TimeoutSec, maxes.TimeoutSec)
	}
	return nil
}

// --- Isolation Profile Endpoints ---

// ListProfiles returns all isolation profiles (builtin + custom).
// GET /api/v1/config/profiles
func (h *Handlers) ListProfiles(w http.ResponseWriter, r *http.Request) {
	if h.ProfileStore == nil {
		// Return just the defaults if no store configured
		h.JSONSuccess(w, config.DefaultProfiles())
		return
	}

	profiles, err := h.ProfileStore.List()
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.JSONSuccess(w, profiles)
}

// GetProfile returns a single profile by ID.
// GET /api/v1/config/profiles/{id}
func (h *Handlers) GetProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if h.ProfileStore == nil {
		// Check builtin profiles only
		for _, p := range config.DefaultProfiles() {
			if p.ID == id {
				h.JSONSuccess(w, p)
				return
			}
		}
		h.JSONError(w, "profile not found: "+id, http.StatusNotFound)
		return
	}

	profile, err := h.ProfileStore.Get(id)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusNotFound)
		return
	}
	h.JSONSuccess(w, profile)
}

// SaveProfile creates or updates a custom profile.
// PUT /api/v1/config/profiles/{id}
func (h *Handlers) SaveProfile(w http.ResponseWriter, r *http.Request) {
	if h.ProfileStore == nil {
		h.JSONError(w, "profile store not configured", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	var profile config.IsolationProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		h.JSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Ensure ID matches URL and can't be marked as builtin via API
	profile.ID = id
	profile.Builtin = false

	// Validate network access value
	validNetworkAccess := map[string]bool{"none": true, "localhost": true, "full": true}
	if !validNetworkAccess[profile.NetworkAccess] {
		h.JSONError(w, "networkAccess must be one of: none, localhost, full", http.StatusBadRequest)
		return
	}

	if err := h.ProfileStore.Save(profile); err != nil {
		h.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Refresh the resolver snapshot so future Resolves see the new
	// profile without requiring a server restart (Round 4 Phase 9).
	if err := h.RefreshProfileSnapshot(); err != nil {
		h.JSONError(w, "profile saved but snapshot refresh failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	h.JSONSuccess(w, profile)
}

// DeleteProfile removes a custom profile.
// DELETE /api/v1/config/profiles/{id}
func (h *Handlers) DeleteProfile(w http.ResponseWriter, r *http.Request) {
	if h.ProfileStore == nil {
		h.JSONError(w, "profile store not configured", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.ProfileStore.Delete(id); err != nil {
		h.JSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Mirror the resolver snapshot to the post-delete state so future
	// Resolves see the deletion immediately (Round 4 Phase 9).
	if err := h.RefreshProfileSnapshot(); err != nil {
		h.JSONError(w, "profile deleted but snapshot refresh failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
