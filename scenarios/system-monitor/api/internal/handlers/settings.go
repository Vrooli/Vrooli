package handlers

// DOC: docs/reference/api-endpoints.md#settings

import (
	"log/slog"
	"net/http"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/api"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/apierrors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/convert"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/httputil"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services"
)

// SettingsHandler handles settings-related API endpoints
type SettingsHandler struct {
	log             *slog.Logger
	settingsManager SettingsProvider
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(settingsManager SettingsProvider, log *slog.Logger) *SettingsHandler {
	return &SettingsHandler{
		log:             log,
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
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// UpdateSettings handles PUT /api/settings
func (h *SettingsHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var reqPb apipb.UpdateSettingsRequest
	if err := httputil.DecodeProtoJSON(r, &reqPb); err != nil {
		httputil.HandleError(w, h.log, r, apierrors.Validation("body", "Invalid JSON payload"))
		return
	}

	newSettings := convert.ProtoToSettings(reqPb.Settings)
	if newSettings == nil {
		httputil.HandleError(w, h.log, r, apierrors.Validation("body", "Settings are required"))
		return
	}

	// Validate settings
	if err := h.validateSettings(newSettings); err != nil {
		httputil.HandleError(w, h.log, r, err)
		return
	}

	// Update settings
	if err := h.settingsManager.UpdateSettings(*newSettings); err != nil {
		httputil.HandleError(w, h.log, r, apierrors.Internal("Failed to update settings", err))
		return
	}

	// Return updated settings
	updatedSettings := h.settingsManager.GetSettings()
	resp := &apipb.UpdateSettingsResponse{
		Success:  true,
		Settings: convert.SettingsToProto(&updatedSettings),
	}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// ResetSettings handles POST /api/settings/reset
func (h *SettingsHandler) ResetSettings(w http.ResponseWriter, r *http.Request) {
	if err := h.settingsManager.ResetSettings(); err != nil {
		httputil.HandleError(w, h.log, r, apierrors.Internal("Failed to reset settings", err))
		return
	}

	// Return reset settings
	settings := h.settingsManager.GetSettings()
	resp := &apipb.ResetSettingsResponse{
		Success:  true,
		Settings: convert.SettingsToProto(&settings),
	}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// GetMaintenanceState handles GET /api/maintenance/state
func (h *SettingsHandler) GetMaintenanceState(w http.ResponseWriter, r *http.Request) {
	state := h.settingsManager.GetMaintenanceState()

	resp := &apipb.GetMaintenanceStateResponse{
		Success:          true,
		MaintenanceState: state,
	}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// SetMaintenanceState handles POST /api/maintenance/state
func (h *SettingsHandler) SetMaintenanceState(w http.ResponseWriter, r *http.Request) {
	var reqPb apipb.SetMaintenanceStateRequest
	if err := httputil.DecodeProtoJSON(r, &reqPb); err != nil {
		httputil.HandleError(w, h.log, r, apierrors.Validation("body", "Invalid JSON payload"))
		return
	}

	// Validate maintenance state
	if reqPb.MaintenanceState != "active" && reqPb.MaintenanceState != "inactive" {
		httputil.HandleError(w, h.log, r, apierrors.Validation("maintenance_state", "Must be 'active' or 'inactive'"))
		return
	}

	// Update maintenance state
	if err := h.settingsManager.SetMaintenanceState(reqPb.MaintenanceState); err != nil {
		httputil.HandleError(w, h.log, r, apierrors.Internal("Failed to update maintenance state", err))
		return
	}

	// Return updated state
	newState := h.settingsManager.GetMaintenanceState()
	resp := &apipb.SetMaintenanceStateResponse{
		Success:          true,
		MaintenanceState: newState,
	}
	httputil.SafeProtoJSON(w, h.log, r, resp)
}

// validateSettings validates the settings before applying them
func (h *SettingsHandler) validateSettings(settings *services.Settings) error {
	// Validate intervals (must be positive)
	if settings.MetricCollectionInterval <= 0 {
		return apierrors.Validation("metric_collection_interval", "must be greater than 0")
	}
	if settings.AnomalyDetectionInterval <= 0 {
		return apierrors.Validation("anomaly_detection_interval", "must be greater than 0")
	}
	if settings.ThresholdCheckInterval <= 0 {
		return apierrors.Validation("threshold_check_interval", "must be greater than 0")
	}
	if settings.CooldownPeriodSeconds < 0 {
		return apierrors.Validation("cooldown_period_seconds", "must be greater than or equal to 0")
	}

	// Validate thresholds (must be between 0 and 100)
	if settings.CPUThreshold < 0 || settings.CPUThreshold > 100 {
		return apierrors.Validation("cpu_threshold", "must be between 0 and 100")
	}
	if settings.MemoryThreshold < 0 || settings.MemoryThreshold > 100 {
		return apierrors.Validation("memory_threshold", "must be between 0 and 100")
	}
	if settings.DiskThreshold < 0 || settings.DiskThreshold > 100 {
		return apierrors.Validation("disk_threshold", "must be between 0 and 100")
	}

	// Validate reasonable ranges
	if settings.MetricCollectionInterval > 3600 { // Max 1 hour
		return apierrors.Validation("metric_collection_interval", "must be less than or equal to 3600 seconds")
	}
	if settings.AnomalyDetectionInterval > 7200 { // Max 2 hours
		return apierrors.Validation("anomaly_detection_interval", "must be less than or equal to 7200 seconds")
	}
	if settings.ThresholdCheckInterval > 1800 { // Max 30 minutes
		return apierrors.Validation("threshold_check_interval", "must be less than or equal to 1800 seconds")
	}
	if settings.CooldownPeriodSeconds > 86400 { // Max 24 hours
		return apierrors.Validation("cooldown_period_seconds", "must be less than or equal to 86400 seconds")
	}

	// Validate metrics lifecycle settings.
	if settings.MetricsRetentionDays <= 0 {
		return apierrors.Validation("metrics_retention_days", "must be greater than 0")
	}
	if settings.MetricsRetentionDays > 3650 { // Max ~10 years
		return apierrors.Validation("metrics_retention_days", "must be less than or equal to 3650 days")
	}
	if settings.RetentionCheckIntervalSeconds < 60 {
		return apierrors.Validation("retention_check_interval_seconds", "must be greater than or equal to 60 seconds")
	}
	if settings.RetentionCheckIntervalSeconds > 604800 { // Max 7 days
		return apierrors.Validation("retention_check_interval_seconds", "must be less than or equal to 604800 seconds")
	}

	return nil
}
