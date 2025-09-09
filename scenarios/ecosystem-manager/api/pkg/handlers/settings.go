package handlers

import (
	"encoding/json"
	"net/http"
	"log"

	"github.com/ecosystem-manager/api/pkg/queue"
	"github.com/ecosystem-manager/api/pkg/websocket"
	"github.com/ecosystem-manager/api/pkg/settings"
)

// SettingsHandlers contains handlers for settings-related endpoints
type SettingsHandlers struct {
	processor *queue.Processor
	wsManager *websocket.Manager
}


// NewSettingsHandlers creates a new settings handlers instance
func NewSettingsHandlers(processor *queue.Processor, wsManager *websocket.Manager) *SettingsHandlers {
	return &SettingsHandlers{
		processor: processor,
		wsManager: wsManager,
	}
}

// GetSettingsHandler returns current settings
func (h *SettingsHandlers) GetSettingsHandler(w http.ResponseWriter, r *http.Request) {
	currentSettings := settings.GetSettings()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"settings": currentSettings,
	})
}

// UpdateSettingsHandler updates settings and applies them
func (h *SettingsHandlers) UpdateSettingsHandler(w http.ResponseWriter, r *http.Request) {
	var newSettings settings.Settings
	if err := json.NewDecoder(r.Body).Decode(&newSettings); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	
	// Validate settings
	if newSettings.Slots < 1 || newSettings.Slots > 5 {
		http.Error(w, "Slots must be between 1 and 5", http.StatusBadRequest)
		return
	}
	if newSettings.RefreshInterval < 5 || newSettings.RefreshInterval > 300 {
		http.Error(w, "Refresh interval must be between 5 and 300 seconds", http.StatusBadRequest)
		return
	}
	if newSettings.MaxTurns < 5 || newSettings.MaxTurns > 80 {
		http.Error(w, "Max turns must be between 5 and 80", http.StatusBadRequest)
		return
	}
	if newSettings.TaskTimeout < 5 || newSettings.TaskTimeout > 240 {
		http.Error(w, "Task timeout must be between 5 and 240 minutes", http.StatusBadRequest)
		return
	}
	
	// Store old active state for comparison
	oldSettings := settings.GetSettings()
	wasActive := oldSettings.Active
	
	// Update settings
	settings.UpdateSettings(newSettings)
	
	// Apply processor settings
	if wasActive != newSettings.Active {
		if newSettings.Active {
			h.processor.Resume() // Remove from maintenance mode
			log.Println("Queue processor activated via settings")
		} else {
			h.processor.Pause() // Put in maintenance mode
			log.Println("Queue processor paused via settings")
		}
	}
	
	// Broadcast settings change via WebSocket
	h.wsManager.BroadcastUpdate("settings_updated", newSettings)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"settings": newSettings,
		"message":  "Settings updated successfully",
	})
	
	log.Printf("Settings updated: slots=%d, refresh=%ds, active=%v, max_turns=%d, timeout=%dm", 
		newSettings.Slots, newSettings.RefreshInterval, newSettings.Active, newSettings.MaxTurns, newSettings.TaskTimeout)
}

// ResetSettingsHandler resets settings to defaults
func (h *SettingsHandlers) ResetSettingsHandler(w http.ResponseWriter, r *http.Request) {
	// Pause processor if it was active
	if settings.IsActive() {
		h.processor.Pause()
		log.Println("Queue processor paused during settings reset")
	}
	
	// Reset to defaults
	newSettings := settings.ResetSettings()
	
	// Broadcast settings change via WebSocket
	h.wsManager.BroadcastUpdate("settings_reset", newSettings)
	
	w.Header().Set("Content-Type", "application/json") 
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"settings": newSettings,
		"message":  "Settings reset to defaults",
	})
	
	log.Println("Settings reset to defaults")
}