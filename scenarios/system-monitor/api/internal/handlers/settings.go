package handlers

import (
	"net/http"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/api"
	"system-monitor-api/internal/convert"
	"system-monitor-api/internal/httputil"
	"system-monitor-api/internal/services"
)

// SettingsHandler handles settings-related API endpoints
type SettingsHandler struct {
	settingsManager SettingsProvider
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(settingsManager SettingsProvider) *SettingsHandler {
	return &SettingsHandler{
		settingsManager: settingsManager,
	}
}

// GetSettings handles GET /api/settings
func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	settings := h.settingsManager.GetSettings()

	resp := &apipb.GetSettingsResponse{
		Success:  true,
		Settings: convert.SettingsToProto(&settings),
	}
	httputil.ProtoJSON(w, resp) //nolint:errcheck
}

// UpdateSettings handles PUT /api/settings
func (h *SettingsHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var reqPb apipb.UpdateSettingsRequest
	if err := httputil.DecodeProtoJSON(r, &reqPb); err != nil {
		httputil.ProtoJSONWithStatus(w, http.StatusBadRequest, &apipb.UpdateSettingsResponse{ //nolint:errcheck
			Error: "Invalid JSON payload",
		})
		return
	}

	newSettings := convert.ProtoToSettings(reqPb.Settings)
	if newSettings == nil {
		httputil.ProtoJSONWithStatus(w, http.StatusBadRequest, &apipb.UpdateSettingsResponse{ //nolint:errcheck
			Error: "Settings are required",
		})
		return
	}

	// Validate settings
	if err := h.validateSettings(newSettings); err != nil {
		httputil.ProtoJSONWithStatus(w, http.StatusBadRequest, &apipb.UpdateSettingsResponse{ //nolint:errcheck
			Error: err.Error(),
		})
		return
	}

	// Update settings
	if err := h.settingsManager.UpdateSettings(*newSettings); err != nil {
		httputil.ProtoJSONWithStatus(w, http.StatusInternalServerError, &apipb.UpdateSettingsResponse{ //nolint:errcheck
			Error: "Failed to update settings: " + err.Error(),
		})
		return
	}

	// Return updated settings
	updatedSettings := h.settingsManager.GetSettings()
	resp := &apipb.UpdateSettingsResponse{
		Success:  true,
		Settings: convert.SettingsToProto(&updatedSettings),
	}
	httputil.ProtoJSON(w, resp) //nolint:errcheck
}

// ResetSettings handles POST /api/settings/reset
func (h *SettingsHandler) ResetSettings(w http.ResponseWriter, r *http.Request) {
	if err := h.settingsManager.ResetSettings(); err != nil {
		httputil.ProtoJSONWithStatus(w, http.StatusInternalServerError, &apipb.ResetSettingsResponse{ //nolint:errcheck
			Error: "Failed to reset settings: " + err.Error(),
		})
		return
	}

	// Return reset settings
	settings := h.settingsManager.GetSettings()
	resp := &apipb.ResetSettingsResponse{
		Success:  true,
		Settings: convert.SettingsToProto(&settings),
	}
	httputil.ProtoJSON(w, resp) //nolint:errcheck
}

// GetMaintenanceState handles GET /api/maintenance/state
func (h *SettingsHandler) GetMaintenanceState(w http.ResponseWriter, r *http.Request) {
	state := h.settingsManager.GetMaintenanceState()

	resp := &apipb.GetMaintenanceStateResponse{
		Success:          true,
		MaintenanceState: state,
	}
	httputil.ProtoJSON(w, resp) //nolint:errcheck
}

// SetMaintenanceState handles POST /api/maintenance/state
func (h *SettingsHandler) SetMaintenanceState(w http.ResponseWriter, r *http.Request) {
	var reqPb apipb.SetMaintenanceStateRequest
	if err := httputil.DecodeProtoJSON(r, &reqPb); err != nil {
		httputil.ProtoJSONWithStatus(w, http.StatusBadRequest, &apipb.SetMaintenanceStateResponse{ //nolint:errcheck
			Error: "Invalid JSON payload",
		})
		return
	}

	// Validate maintenance state
	if reqPb.MaintenanceState != "active" && reqPb.MaintenanceState != "inactive" {
		httputil.ProtoJSONWithStatus(w, http.StatusBadRequest, &apipb.SetMaintenanceStateResponse{ //nolint:errcheck
			Error: "Invalid maintenance state. Must be 'active' or 'inactive'",
		})
		return
	}

	// Update maintenance state
	if err := h.settingsManager.SetMaintenanceState(reqPb.MaintenanceState); err != nil {
		httputil.ProtoJSONWithStatus(w, http.StatusInternalServerError, &apipb.SetMaintenanceStateResponse{ //nolint:errcheck
			Error: "Failed to update maintenance state: " + err.Error(),
		})
		return
	}

	// Return updated state
	newState := h.settingsManager.GetMaintenanceState()
	resp := &apipb.SetMaintenanceStateResponse{
		Success:          true,
		MaintenanceState: newState,
	}
	httputil.ProtoJSON(w, resp) //nolint:errcheck
}

// validateSettings validates the settings before applying them
func (h *SettingsHandler) validateSettings(settings *services.Settings) error {
	// Validate intervals (must be positive)
	if settings.MetricCollectionInterval <= 0 {
		return &ValidationError{Field: "metric_collection_interval", Message: "must be greater than 0"}
	}
	if settings.AnomalyDetectionInterval <= 0 {
		return &ValidationError{Field: "anomaly_detection_interval", Message: "must be greater than 0"}
	}
	if settings.ThresholdCheckInterval <= 0 {
		return &ValidationError{Field: "threshold_check_interval", Message: "must be greater than 0"}
	}
	if settings.CooldownPeriodSeconds < 0 {
		return &ValidationError{Field: "cooldown_period_seconds", Message: "must be greater than or equal to 0"}
	}

	// Validate thresholds (must be between 0 and 100)
	if settings.CPUThreshold < 0 || settings.CPUThreshold > 100 {
		return &ValidationError{Field: "cpu_threshold", Message: "must be between 0 and 100"}
	}
	if settings.MemoryThreshold < 0 || settings.MemoryThreshold > 100 {
		return &ValidationError{Field: "memory_threshold", Message: "must be between 0 and 100"}
	}
	if settings.DiskThreshold < 0 || settings.DiskThreshold > 100 {
		return &ValidationError{Field: "disk_threshold", Message: "must be between 0 and 100"}
	}

	// Validate reasonable ranges
	if settings.MetricCollectionInterval > 3600 { // Max 1 hour
		return &ValidationError{Field: "metric_collection_interval", Message: "must be less than or equal to 3600 seconds"}
	}
	if settings.AnomalyDetectionInterval > 7200 { // Max 2 hours
		return &ValidationError{Field: "anomaly_detection_interval", Message: "must be less than or equal to 7200 seconds"}
	}
	if settings.ThresholdCheckInterval > 1800 { // Max 30 minutes
		return &ValidationError{Field: "threshold_check_interval", Message: "must be less than or equal to 1800 seconds"}
	}
	if settings.CooldownPeriodSeconds > 86400 { // Max 24 hours
		return &ValidationError{Field: "cooldown_period_seconds", Message: "must be less than or equal to 86400 seconds"}
	}

	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
