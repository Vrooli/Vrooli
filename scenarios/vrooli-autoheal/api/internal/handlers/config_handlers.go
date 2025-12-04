package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gorilla/mux"

	apierrors "vrooli-autoheal/internal/errors"
	"vrooli-autoheal/internal/userconfig"
)

// ConfigHandlers wraps handlers for configuration management
type ConfigHandlers struct {
	configMgr *userconfig.Manager
}

// NewConfigHandlers creates a new ConfigHandlers instance
func NewConfigHandlers(configMgr *userconfig.Manager) *ConfigHandlers {
	return &ConfigHandlers{
		configMgr: configMgr,
	}
}

// GetConfig returns the current configuration
// GET /api/v1/config
func (h *ConfigHandlers) GetConfig(w http.ResponseWriter, r *http.Request) {
	config := h.configMgr.Get()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(config); err != nil {
		apierrors.LogError("get_config", "encode_response", err)
	}
}

// UpdateConfig replaces the configuration
// PUT /api/v1/config
func (h *ConfigHandlers) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var config userconfig.Config
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("config", "parse request body", err))
		return
	}

	if err := h.configMgr.Update(&config); err != nil {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("config", "update configuration", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Configuration updated successfully",
		"config":  h.configMgr.Get(),
	})
}

// ValidateConfig validates a configuration without saving
// POST /api/v1/config/validate
func (h *ConfigHandlers) ValidateConfig(w http.ResponseWriter, r *http.Request) {
	var config userconfig.Config
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("config", "parse request body", err))
		return
	}

	result := h.configMgr.Validate(&config)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetSchema returns the JSON schema for configuration
// GET /api/v1/config/schema
func (h *ConfigHandlers) GetSchema(w http.ResponseWriter, r *http.Request) {
	schema, err := h.configMgr.GetSchema()
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewNotFoundError("config", "schema file", "config.schema.json"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(schema))
}

// ExportConfig exports the configuration as downloadable JSON
// GET /api/v1/config/export
func (h *ConfigHandlers) ExportConfig(w http.ResponseWriter, r *http.Request) {
	data, err := h.configMgr.Export()
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewInternalError("config", "export configuration", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=autoheal-config.json")
	w.Write(data)
}

// ImportConfig imports configuration from uploaded JSON
// POST /api/v1/config/import
func (h *ConfigHandlers) ImportConfig(w http.ResponseWriter, r *http.Request) {
	// Read request body (limit to 1MB)
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("config", "read request body", err))
		return
	}

	// Parse and validate
	config, err := h.configMgr.Import(data)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("config", "import configuration", err))
		return
	}

	// Apply the imported configuration
	if err := h.configMgr.Update(config); err != nil {
		apierrors.LogAndRespond(w, apierrors.NewInternalError("config", "apply imported configuration", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Configuration imported successfully",
		"config":  h.configMgr.Get(),
	})
}

// GetCheckConfig returns the effective configuration for a specific check
// GET /api/v1/config/checks/{checkId}
func (h *ConfigHandlers) GetCheckConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	checkID := vars["checkId"]

	checkConfig := h.configMgr.GetCheck(checkID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"checkId": checkID,
		"config":  checkConfig,
	})
}

// UpdateCheckEnabled enables or disables a check
// PUT /api/v1/config/checks/{checkId}/enabled
func (h *ConfigHandlers) UpdateCheckEnabled(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	checkID := vars["checkId"]

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("config", "parse request body", err))
		return
	}

	if err := h.configMgr.SetCheckEnabled(checkID, req.Enabled); err != nil {
		apierrors.LogAndRespond(w, apierrors.NewInternalError("config", "update check enabled state", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"checkId": checkID,
		"enabled": req.Enabled,
	})
}

// UpdateCheckAutoHeal enables or disables auto-heal for a check
// PUT /api/v1/config/checks/{checkId}/autoheal
func (h *ConfigHandlers) UpdateCheckAutoHeal(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	checkID := vars["checkId"]

	var req struct {
		AutoHeal bool `json:"autoHeal"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("config", "parse request body", err))
		return
	}

	if err := h.configMgr.SetCheckAutoHeal(checkID, req.AutoHeal); err != nil {
		apierrors.LogAndRespond(w, apierrors.NewInternalError("config", "update check auto-heal state", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"checkId":  checkID,
		"autoHeal": req.AutoHeal,
	})
}

// BulkUpdateChecks updates enabled/autoHeal for all checks
// PUT /api/v1/config/checks/bulk
func (h *ConfigHandlers) BulkUpdateChecks(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string `json:"action"` // "enableAll", "disableAll", "autoHealAll", "disableAutoHealAll"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("config", "parse request body", err))
		return
	}

	var err error
	switch req.Action {
	case "enableAll":
		err = h.configMgr.SetAllEnabled(true)
	case "disableAll":
		err = h.configMgr.SetAllEnabled(false)
	case "autoHealAll":
		err = h.configMgr.SetAllAutoHeal(true)
	case "disableAutoHealAll":
		err = h.configMgr.SetAllAutoHeal(false)
	default:
		apierrors.LogAndRespond(w, apierrors.NewValidationError("config", "invalid action", nil))
		return
	}

	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewInternalError("config", "bulk update checks", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"action":  req.Action,
		"config":  h.configMgr.Get(),
	})
}

// GetGlobalConfig returns the global configuration
// GET /api/v1/config/global
func (h *ConfigHandlers) GetGlobalConfig(w http.ResponseWriter, r *http.Request) {
	global := h.configMgr.GetGlobal()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(global)
}

// GetUIConfig returns the UI configuration
// GET /api/v1/config/ui
func (h *ConfigHandlers) GetUIConfig(w http.ResponseWriter, r *http.Request) {
	ui := h.configMgr.GetUI()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ui)
}

// GetDefaults returns the default check configurations
// GET /api/v1/config/defaults
func (h *ConfigHandlers) GetDefaults(w http.ResponseWriter, r *http.Request) {
	defaults := make(map[string]interface{})
	for checkID, def := range userconfig.KnownCheckDefaults {
		defaults[checkID] = map[string]interface{}{
			"enabled":         def.Enabled,
			"autoHeal":        def.AutoHeal,
			"intervalSeconds": def.IntervalSeconds,
			"thresholds":      def.Thresholds,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"global": userconfig.DefaultGlobal(),
		"ui":     userconfig.DefaultUI(),
		"checks": defaults,
	})
}
