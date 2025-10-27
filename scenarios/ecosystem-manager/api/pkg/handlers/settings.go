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

	writeJSON(w, map[string]any{
		"success":  true,
		"settings": currentSettings,
	}, http.StatusOK)
}

// UpdateSettingsHandler updates settings and applies them
func (h *SettingsHandlers) UpdateSettingsHandler(w http.ResponseWriter, r *http.Request) {
	newSettingsPtr, ok := decodeJSONBody[settings.Settings](w, r)
	if !ok {
		return
	}
	newSettings := *newSettingsPtr

	// Validate settings
	if newSettings.Slots < settings.MinSlots || newSettings.Slots > settings.MaxSlots {
		writeError(w, fmt.Sprintf("Slots must be between %d and %d", settings.MinSlots, settings.MaxSlots), http.StatusBadRequest)
		return
	}
	if newSettings.RefreshInterval < settings.MinRefreshInterval || newSettings.RefreshInterval > settings.MaxRefreshInterval {
		writeError(w, fmt.Sprintf("Refresh interval must be between %d and %d seconds", settings.MinRefreshInterval, settings.MaxRefreshInterval), http.StatusBadRequest)
		return
	}
	if newSettings.MaxTurns < settings.MinMaxTurns || newSettings.MaxTurns > settings.MaxMaxTurns {
		writeError(w, fmt.Sprintf("Max turns must be between %d and %d", settings.MinMaxTurns, settings.MaxMaxTurns), http.StatusBadRequest)
		return
	}
	if newSettings.TaskTimeout < settings.MinTaskTimeout || newSettings.TaskTimeout > settings.MaxTaskTimeout {
		writeError(w, fmt.Sprintf("Task timeout must be between %d and %d minutes", settings.MinTaskTimeout, settings.MaxTaskTimeout), http.StatusBadRequest)
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
		writeError(w, "Recycler enabled_for must be one of off, resources, scenarios, both", http.StatusBadRequest)
		return
	}

	if recycler.IntervalSeconds == 0 {
		recycler.IntervalSeconds = oldSettings.Recycler.IntervalSeconds
	}
	if recycler.IntervalSeconds < settings.MinRecyclerInterval || recycler.IntervalSeconds > settings.MaxRecyclerInterval {
		writeError(w, fmt.Sprintf("Recycler interval must be between %d and %d seconds", settings.MinRecyclerInterval, settings.MaxRecyclerInterval), http.StatusBadRequest)
		return
	}

	if recycler.ModelProvider == "" {
		recycler.ModelProvider = oldSettings.Recycler.ModelProvider
	}
	recycler.ModelProvider = strings.ToLower(strings.TrimSpace(recycler.ModelProvider))
	if recycler.ModelProvider != "ollama" && recycler.ModelProvider != "openrouter" {
		writeError(w, "Recycler model_provider must be 'ollama' or 'openrouter'", http.StatusBadRequest)
		return
	}

	if recycler.ModelName == "" {
		recycler.ModelName = oldSettings.Recycler.ModelName
	}

	if recycler.CompletionThreshold == 0 {
		recycler.CompletionThreshold = oldSettings.Recycler.CompletionThreshold
	}
	if recycler.CompletionThreshold < settings.MinRecyclerCompletionThreshold || recycler.CompletionThreshold > settings.MaxRecyclerCompletionThreshold {
		writeError(w, fmt.Sprintf("Recycler completion_threshold must be between %d and %d", settings.MinRecyclerCompletionThreshold, settings.MaxRecyclerCompletionThreshold), http.StatusBadRequest)
		return
	}

	if recycler.FailureThreshold == 0 {
		recycler.FailureThreshold = oldSettings.Recycler.FailureThreshold
	}
	if recycler.FailureThreshold < settings.MinRecyclerFailureThreshold || recycler.FailureThreshold > settings.MaxRecyclerFailureThreshold {
		writeError(w, fmt.Sprintf("Recycler failure_threshold must be between %d and %d", settings.MinRecyclerFailureThreshold, settings.MaxRecyclerFailureThreshold), http.StatusBadRequest)
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

	var resumeSummary *queue.ResumeResetSummary
	// Apply processor settings
	if wasActive != newSettings.Active {
		if newSettings.Active {
			summary := h.processor.ResumeWithReset() // Remove from maintenance mode with cleanup
			resumeSummary = &summary
			log.Println("Queue processor activated via settings")
		} else {
			h.processor.Pause() // Put in maintenance mode
			log.Println("Queue processor paused via settings")
		}
	}

	// Broadcast settings change via WebSocket
	h.wsManager.BroadcastUpdate("settings_updated", newSettings)

	response := map[string]any{
		"success":  true,
		"settings": newSettings,
		"message":  "Settings updated successfully",
	}
	if resumeSummary != nil {
		response["resume_reset_summary"] = resumeSummary
	}

	writeJSON(w, response, http.StatusOK)

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
		writeError(w, "unsupported provider", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("failed to list %s models: %v", provider, err)
	}

	response := map[string]any{
		"success":  err == nil,
		"provider": provider,
		"models":   models,
	}
	if err != nil {
		response["error"] = err.Error()
	}

	writeJSON(w, response, http.StatusOK)
}

// TestRecyclerHandler generates a recycler summary for a mock task using current settings.
func (h *SettingsHandlers) TestRecyclerHandler(w http.ResponseWriter, r *http.Request) {
	type testRequest struct {
		OutputText     string `json:"output_text"`
		ModelProvider  string `json:"model_provider"`
		ModelName      string `json:"model_name"`
		PromptOverride string `json:"prompt_override"`
	}

	req, ok := decodeJSONBody[testRequest](w, r)
	if !ok {
		return
	}

	output := strings.TrimSpace(req.OutputText)
	if output == "" {
		writeError(w, "output_text is required", http.StatusBadRequest)
		return
	}

	settingsSnapshot := settings.GetSettings()
	config := settingsSnapshot.Recycler

	overrideProvider := strings.ToLower(strings.TrimSpace(req.ModelProvider))
	overrideModel := strings.TrimSpace(req.ModelName)

	selectedProvider := config.ModelProvider
	selectedModel := config.ModelName

	if overrideProvider != "" {
		switch overrideProvider {
		case "ollama", "openrouter":
			selectedProvider = overrideProvider
		default:
			writeError(w, "model_provider must be 'ollama' or 'openrouter'", http.StatusBadRequest)
			return
		}
		if overrideModel == "" {
			writeError(w, "model_name is required when overriding model_provider", http.StatusBadRequest)
			return
		}
	}

	if overrideModel != "" {
		selectedModel = overrideModel
	}

	input := summarizer.Input{Output: output, PromptOverride: req.PromptOverride}

	result, err := summarizer.GenerateNote(
		context.Background(),
		summarizer.Config{Provider: selectedProvider, Model: selectedModel},
		input,
	)

	promptUsed := strings.TrimSpace(req.PromptOverride)
	if promptUsed == "" {
		promptUsed = summarizer.BuildPrompt(output)
	}

	response := map[string]any{
		"success":  err == nil,
		"result":   result,
		"provider": selectedProvider,
		"model":    selectedModel,
		"prompt":   promptUsed,
	}

	if err != nil {
		log.Printf("Recycler test summarizer error: %v", err)
		fallback := summarizer.DefaultResult()
		response["result"] = fallback
		response["error"] = err.Error()
	}

	writeJSON(w, response, http.StatusOK)
}

// PreviewRecyclerPromptHandler builds the recycler LLM prompt for mock output without executing the model.
func (h *SettingsHandlers) PreviewRecyclerPromptHandler(w http.ResponseWriter, r *http.Request) {
	type previewRequest struct {
		OutputText string `json:"output_text"`
	}

	req, ok := decodeJSONBody[previewRequest](w, r)
	if !ok {
		return
	}

	output := strings.TrimSpace(req.OutputText)
	if output == "" {
		writeError(w, "output_text is required", http.StatusBadRequest)
		return
	}

	prompt := summarizer.BuildPrompt(output)

	writeJSON(w, map[string]string{"prompt": prompt}, http.StatusOK)
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

	writeJSON(w, map[string]any{
		"success":  true,
		"settings": newSettings,
		"message":  "Settings reset to defaults",
	}, http.StatusOK)

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
