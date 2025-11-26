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
	"time"

	"github.com/ecosystem-manager/api/pkg/queue"
	"github.com/ecosystem-manager/api/pkg/recycler"
	"github.com/ecosystem-manager/api/pkg/settings"
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

const settingsCommandTimeout = 10 * time.Second

// NewSettingsHandlers creates a new settings handlers instance
func NewSettingsHandlers(processor *queue.Processor, wsManager *websocket.Manager, recycler *recycler.Recycler) *SettingsHandlers {
	return &SettingsHandlers{
		processor: processor,
		wsManager: wsManager,
		recycler:  recycler,
	}
}

// GetSettingsHandler returns current settings with validation constraints
func (h *SettingsHandlers) GetSettingsHandler(w http.ResponseWriter, r *http.Request) {
	currentSettings := settings.GetSettings()

	writeJSON(w, map[string]any{
		"success":  true,
		"settings": currentSettings,
		"constraints": map[string]any{
			"slots": map[string]int{
				"min": settings.MinSlots,
				"max": settings.MaxSlots,
			},
			"cooldown_seconds": map[string]int{
				"min": settings.MinCooldownSeconds,
				"max": settings.MaxCooldownSeconds,
			},
			"max_turns": map[string]int{
				"min": settings.MinMaxTurns,
				"max": settings.MaxMaxTurns,
			},
			"task_timeout": map[string]int{
				"min": settings.MinTaskTimeout,
				"max": settings.MaxTaskTimeout,
			},
			"idle_timeout_cap": map[string]int{
				"min": settings.MinIdleTimeoutCap,
				"max": settings.MaxIdleTimeoutCap,
			},
			"recycler": map[string]any{
				"interval_seconds": map[string]int{
					"min": settings.MinRecyclerInterval,
					"max": settings.MaxRecyclerInterval,
				},
				"max_retries": map[string]int{
					"min": settings.MinRecyclerMaxRetries,
					"max": settings.MaxRecyclerMaxRetries,
				},
				"retry_delay_seconds": map[string]int{
					"min": settings.MinRecyclerRetryDelaySecs,
					"max": settings.MaxRecyclerRetryDelaySecs,
				},
				"completion_threshold": map[string]int{
					"min": settings.MinRecyclerCompletionThreshold,
					"max": settings.MaxRecyclerCompletionThreshold,
				},
				"failure_threshold": map[string]int{
					"min": settings.MinRecyclerFailureThreshold,
					"max": settings.MaxRecyclerFailureThreshold,
				},
			},
		},
	}, http.StatusOK)
}

// UpdateSettingsHandler updates settings and applies them
func (h *SettingsHandlers) UpdateSettingsHandler(w http.ResponseWriter, r *http.Request) {
	newSettingsPtr, ok := decodeJSONBody[settings.Settings](w, r)
	if !ok {
		return
	}
	oldSettings := settings.GetSettings()
	validated, err := settings.ValidateAndNormalize(*newSettingsPtr, oldSettings)
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Store old active state for comparison
	wasActive := oldSettings.Active

	// Update settings
	settings.UpdateSettings(validated)
	if h.recycler != nil {
		h.recycler.OnSettingsUpdated(oldSettings, validated)
		h.recycler.Wake()
	}

	var resumeSummary *queue.ResumeResetSummary
	// Apply processor settings
	if wasActive != validated.Active {
		if validated.Active {
			summary := h.processor.ResumeWithReset() // Remove from maintenance mode with cleanup
			resumeSummary = &summary
			log.Println("Queue processor activated via settings")
		} else {
			h.processor.Pause() // Put in maintenance mode
			log.Println("Queue processor paused via settings")
		}
	}

	if err := settings.SaveToDisk(); err != nil {
		log.Printf("Failed to persist settings: %v", err)
		writeError(w, "Failed to persist settings", http.StatusInternalServerError)
		return
	}

	// Broadcast settings change via WebSocket
	h.wsManager.BroadcastUpdate("settings_updated", validated)

	response := map[string]any{
		"success":  true,
		"settings": validated,
		"message":  "Settings updated successfully",
	}
	if resumeSummary != nil {
		response["resume_reset_summary"] = resumeSummary
	}

	writeJSON(w, response, http.StatusOK)

	log.Printf("Settings updated: slots=%d, cooldown=%ds, active=%v, max_turns=%d, timeout=%dm, idle_cap=%dm",
		validated.Slots, validated.CooldownSeconds, validated.Active, validated.MaxTurns, validated.TaskTimeout, validated.IdleTimeoutCap)
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

// ResetSettingsHandler resets settings to defaults
func (h *SettingsHandlers) ResetSettingsHandler(w http.ResponseWriter, r *http.Request) {
	// Pause processor if it was active
	if settings.IsActive() {
		h.processor.Pause()
		log.Println("Queue processor paused during settings reset")
	}

	// Reset to defaults
	newSettings := settings.ResetSettings()

	if err := settings.SaveToDisk(); err != nil {
		log.Printf("Failed to persist settings: %v", err)
		writeError(w, "Failed to persist settings", http.StatusInternalServerError)
		return
	}

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
	output, err := runSettingsCommand("resource-ollama", "content", "list")
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
	output, err := runSettingsCommand("resource-openrouter", "content", "models", "--limit", "200", "--json")
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

func runSettingsCommand(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), settingsCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return output, context.DeadlineExceeded
	}

	return output, err
}
