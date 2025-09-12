package handlers

import (
	"encoding/json"
	"net/http"
	"system-monitor-api/internal/services"
)

// SettingsHandler handles settings-related API endpoints
type SettingsHandler struct {
	settingsManager *services.SettingsManager
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler(settingsManager *services.SettingsManager) *SettingsHandler {
	return &SettingsHandler{
		settingsManager: settingsManager,
	}
}

// SettingsResponse represents the API response for settings
type SettingsResponse struct {
	Success  bool                `json:"success"`
	Settings *services.Settings  `json:"settings,omitempty"`
	Error    string             `json:"error,omitempty"`
}

// MaintenanceStateRequest represents the request for maintenance state changes
type MaintenanceStateRequest struct {
	MaintenanceState string `json:"maintenanceState"`
}

// MaintenanceStateResponse represents the response for maintenance state operations
type MaintenanceStateResponse struct {
	Success          bool   `json:"success"`
	MaintenanceState string `json:"maintenanceState,omitempty"`
	Error           string `json:"error,omitempty"`
}

// GetSettings handles GET /api/settings
func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	settings := h.settingsManager.GetSettings()
	
	response := SettingsResponse{
		Success:  true,
		Settings: &settings,
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// UpdateSettings handles PUT /api/settings
func (h *SettingsHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var newSettings services.Settings
	if err := json.NewDecoder(r.Body).Decode(&newSettings); err != nil {
		response := SettingsResponse{
			Success: false,
			Error:   "Invalid JSON payload",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}
	
	// Validate settings
	if err := h.validateSettings(&newSettings); err != nil {
		response := SettingsResponse{
			Success: false,
			Error:   err.Error(),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}
	
	// Update settings
	if err := h.settingsManager.UpdateSettings(newSettings); err != nil {
		response := SettingsResponse{
			Success: false,
			Error:   "Failed to update settings: " + err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	
	// Return updated settings
	updatedSettings := h.settingsManager.GetSettings()
	response := SettingsResponse{
		Success:  true,
		Settings: &updatedSettings,
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ResetSettings handles POST /api/settings/reset
func (h *SettingsHandler) ResetSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if err := h.settingsManager.ResetSettings(); err != nil {
		response := SettingsResponse{
			Success: false,
			Error:   "Failed to reset settings: " + err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	
	// Return reset settings
	settings := h.settingsManager.GetSettings()
	response := SettingsResponse{
		Success:  true,
		Settings: &settings,
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetMaintenanceState handles GET /api/maintenance/state
func (h *SettingsHandler) GetMaintenanceState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	state := h.settingsManager.GetMaintenanceState()
	
	response := MaintenanceStateResponse{
		Success:          true,
		MaintenanceState: state,
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// SetMaintenanceState handles POST /api/maintenance/state
func (h *SettingsHandler) SetMaintenanceState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var request MaintenanceStateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response := MaintenanceStateResponse{
			Success: false,
			Error:   "Invalid JSON payload",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}
	
	// Validate maintenance state
	if request.MaintenanceState != "active" && request.MaintenanceState != "inactive" {
		response := MaintenanceStateResponse{
			Success: false,
			Error:   "Invalid maintenance state. Must be 'active' or 'inactive'",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}
	
	// Update maintenance state
	if err := h.settingsManager.SetMaintenanceState(request.MaintenanceState); err != nil {
		response := MaintenanceStateResponse{
			Success: false,
			Error:   "Failed to update maintenance state: " + err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	
	// Return updated state
	newState := h.settingsManager.GetMaintenanceState()
	response := MaintenanceStateResponse{
		Success:          true,
		MaintenanceState: newState,
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
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