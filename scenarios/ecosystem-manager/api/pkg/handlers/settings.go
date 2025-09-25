package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"sort"
	"strings"

	"github.com/ecosystem-manager/api/pkg/queue"
	"github.com/ecosystem-manager/api/pkg/recycler"
	"github.com/ecosystem-manager/api/pkg/settings"
	"github.com/ecosystem-manager/api/pkg/summarizer"
	"github.com/ecosystem-manager/api/pkg/tasks"
	"github.com/ecosystem-manager/api/pkg/websocket"
)

// SettingsHandlers contains handlers for settings-related endpoints
type SettingsHandlers struct {
	processor *queue.Processor
	wsManager *websocket.Manager
	recycler  *recycler.Recycler
}

type modelOption struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Provider string `json:"provider"`
}

// NewSettingsHandlers creates a new settings handlers instance
func NewSettingsHandlers(processor *queue.Processor, wsManager *websocket.Manager, recycler *recycler.Recycler) *SettingsHandlers {
	return &SettingsHandlers{
		processor: processor,
		wsManager: wsManager,
		recycler:  recycler,
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

	oldSettings := settings.GetSettings()
	recycler := newSettings.Recycler
	if recycler.EnabledFor == "" {
		recycler.EnabledFor = oldSettings.Recycler.EnabledFor
	}
	recycler.EnabledFor = strings.ToLower(strings.TrimSpace(recycler.EnabledFor))
	switch recycler.EnabledFor {
	case "", "off", "resources", "scenarios", "both":
		if recycler.EnabledFor == "" {
			recycler.EnabledFor = "off"
		}
	default:
		http.Error(w, "Recycler enabled_for must be one of off, resources, scenarios, both", http.StatusBadRequest)
		return
	}

	if recycler.IntervalSeconds == 0 {
		recycler.IntervalSeconds = oldSettings.Recycler.IntervalSeconds
	}
	if recycler.IntervalSeconds < 30 || recycler.IntervalSeconds > 1800 {
		http.Error(w, "Recycler interval must be between 30 and 1800 seconds", http.StatusBadRequest)
		return
	}

	if recycler.ModelProvider == "" {
		recycler.ModelProvider = oldSettings.Recycler.ModelProvider
	}
	recycler.ModelProvider = strings.ToLower(strings.TrimSpace(recycler.ModelProvider))
	if recycler.ModelProvider != "ollama" && recycler.ModelProvider != "openrouter" {
		http.Error(w, "Recycler model_provider must be 'ollama' or 'openrouter'", http.StatusBadRequest)
		return
	}

	if recycler.ModelName == "" {
		recycler.ModelName = oldSettings.Recycler.ModelName
	}

	if recycler.CompletionThreshold == 0 {
		recycler.CompletionThreshold = oldSettings.Recycler.CompletionThreshold
	}
	if recycler.CompletionThreshold < 1 || recycler.CompletionThreshold > 10 {
		http.Error(w, "Recycler completion_threshold must be between 1 and 10", http.StatusBadRequest)
		return
	}

	if recycler.FailureThreshold == 0 {
		recycler.FailureThreshold = oldSettings.Recycler.FailureThreshold
	}
	if recycler.FailureThreshold < 1 || recycler.FailureThreshold > 10 {
		http.Error(w, "Recycler failure_threshold must be between 1 and 10", http.StatusBadRequest)
		return
	}

	newSettings.Recycler = recycler

	// Store old active state for comparison
	wasActive := oldSettings.Active

	// Update settings
	settings.UpdateSettings(newSettings)
	if h.recycler != nil {
		h.recycler.Wake()
	}

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

// GetRecyclerModelsHandler returns available models for a given provider.
func (h *SettingsHandlers) GetRecyclerModelsHandler(w http.ResponseWriter, r *http.Request) {
	provider := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("provider")))
	if provider == "" {
		provider = "ollama"
	}

	var (
		models []modelOption
		err    error
	)

	switch provider {
	case "ollama":
		models, err = listOllamaModels()
	case "openrouter":
		models, err = listOpenRouterModels()
	default:
		http.Error(w, "unsupported provider", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("failed to list %s models: %v", provider, err)
	}

	response := map[string]interface{}{
		"success":  err == nil,
		"provider": provider,
		"models":   models,
	}
	if err != nil {
		response["error"] = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// TestRecyclerHandler generates a recycler summary for a mock task using current settings.
func (h *SettingsHandlers) TestRecyclerHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OutputText       string `json:"output_text"`
		PreviousNote     string `json:"previous_note"`
		TaskType         string `json:"task_type"`
		Operation        string `json:"operation"`
		CompletionStreak int    `json:"completion_streak"`
		FailureStreak    int    `json:"failure_streak"`
		CompletionCount  int    `json:"completion_count"`
		Title            string `json:"title"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	output := strings.TrimSpace(req.OutputText)
	if output == "" {
		http.Error(w, "output_text is required", http.StatusBadRequest)
		return
	}

	settingsSnapshot := settings.GetSettings()
	config := settingsSnapshot.Recycler

	taskType := strings.TrimSpace(req.TaskType)
	if taskType == "" {
		taskType = "scenario"
	}
	operation := strings.TrimSpace(req.Operation)
	if operation == "" {
		operation = "improver"
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "Recycler Test Task"
	}

	mockTask := tasks.TaskItem{
		ID:                          "recycler-test",
		Title:                       title,
		Type:                        taskType,
		Operation:                   operation,
		CompletionCount:             req.CompletionCount,
		ConsecutiveCompletionClaims: req.CompletionStreak,
		ConsecutiveFailures:         req.FailureStreak,
		Notes:                       req.PreviousNote,
	}

	result, err := summarizer.GenerateNote(
		context.Background(),
		summarizer.Config{Provider: config.ModelProvider, Model: config.ModelName},
		summarizer.Input{Task: mockTask, Output: output, PreviousNote: req.PreviousNote},
	)

	response := map[string]interface{}{
		"success":          err == nil,
		"result":           result,
		"provider":         config.ModelProvider,
		"model":            config.ModelName,
		"completion_count": mockTask.CompletionCount,
	}

	if err != nil {
		log.Printf("Recycler test summarizer error: %v", err)
		fallback := summarizer.DefaultResult()
		response["result"] = fallback
		response["error"] = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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

func listOllamaModels() ([]modelOption, error) {
	cmd := exec.Command("resource-ollama", "content", "list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return defaultOllamaModels(), fmt.Errorf("resource-ollama content list failed: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}

	lines := strings.Split(strings.ReplaceAll(string(output), "\r\n", "\n"), "\n")
	models := make([]modelOption, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) == 0 {
			continue
		}
		if strings.EqualFold(fields[0], "name") {
			continue
		}
		name := fields[0]
		models = append(models, modelOption{
			ID:       name,
			Label:    name,
			Provider: "ollama",
		})
	}

	if len(models) == 0 {
		return defaultOllamaModels(), fmt.Errorf("resource-ollama returned no models")
	}

	sort.SliceStable(models, func(i, j int) bool {
		return strings.ToLower(models[i].Label) < strings.ToLower(models[j].Label)
	})

	return models, nil
}

func listOpenRouterModels() ([]modelOption, error) {
	cmd := exec.Command("resource-openrouter", "content", "models", "--limit", "200", "--json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("resource-openrouter content models failed: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}

	var payload struct {
		DefaultModel string `json:"default_model"`
		Models       []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			DisplayName string `json:"display_name"`
			Provider    string `json:"provider"`
		} `json:"models"`
	}

	if err := json.Unmarshal(output, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse openrouter model list: %w", err)
	}

	models := make([]modelOption, 0, len(payload.Models))
	for _, m := range payload.Models {
		label := strings.TrimSpace(m.DisplayName)
		if label == "" {
			label = strings.TrimSpace(m.Name)
		}
		if label == "" {
			label = strings.TrimSpace(m.ID)
		}
		if label == "" {
			continue
		}
		models = append(models, modelOption{
			ID:       strings.TrimSpace(m.ID),
			Label:    label,
			Provider: "openrouter",
		})
	}

	if len(models) == 0 && payload.DefaultModel != "" {
		models = append(models, modelOption{
			ID:       payload.DefaultModel,
			Label:    payload.DefaultModel,
			Provider: "openrouter",
		})
	}

	sort.SliceStable(models, func(i, j int) bool {
		return strings.ToLower(models[i].Label) < strings.ToLower(models[j].Label)
	})

	return models, nil
}

func defaultOllamaModels() []modelOption {
	return []modelOption{
		{ID: "llama3.1:8b", Label: "llama3.1:8b", Provider: "ollama"},
		{ID: "llama3.2:3b", Label: "llama3.2:3b", Provider: "ollama"},
	}
}
