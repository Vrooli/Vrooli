package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"scenario-completeness-scoring/pkg/config"

	"github.com/gorilla/mux"
)

// HandleGetConfig returns the current scoring configuration
// [REQ:SCS-CFG-001] Component toggle API
func (ctx *Context) HandleGetConfig(w http.ResponseWriter, r *http.Request) {
	// Load actual config from disk
	cfg, err := ctx.ConfigLoader.LoadGlobal()
	if err != nil {
		// Fall back to default config if load fails
		defaultCfg := config.DefaultConfig()
		cfg = &defaultCfg
	}

	response := map[string]interface{}{
		"version": cfg.Version,
		"scoring": map[string]interface{}{
			"weights": map[string]int{
				"quality":  cfg.Weights.Quality,
				"coverage": cfg.Weights.Coverage,
				"quantity": cfg.Weights.Quantity,
				"ui":       cfg.Weights.UI,
			},
			"quality_breakdown": map[string]int{
				"requirement_pass_rate": 20,
				"target_pass_rate":      15,
				"test_pass_rate":        15,
			},
			"coverage_breakdown": map[string]int{
				"test_coverage_ratio": 8,
				"depth_score":         7,
			},
			"quantity_breakdown": map[string]int{
				"requirements": 4,
				"targets":      3,
				"tests":        3,
			},
			"ui_breakdown": map[string]float64{
				"template_check":       10,
				"component_complexity": 5,
				"api_integration":      6,
				"routing":              1.5,
				"code_volume":          2.5,
			},
		},
		"components": cfg.Components,
		"penalties":  cfg.Penalties,
		"classifications": map[string]interface{}{
			"production_ready":      map[string]int{"min": 96, "max": 100},
			"nearly_ready":          map[string]int{"min": 81, "max": 95},
			"mostly_complete":       map[string]int{"min": 61, "max": 80},
			"functional_incomplete": map[string]int{"min": 41, "max": 60},
			"foundation_laid":       map[string]int{"min": 21, "max": 40},
			"early_stage":           map[string]int{"min": 0, "max": 20},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleUpdateConfig updates the global scoring configuration
// [REQ:SCS-CFG-004] Configuration persistence
func (ctx *Context) HandleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var cfg config.ScoringConfig
	if err := json.Unmarshal(body, &cfg); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Validate configuration
	if err := config.ValidateConfig(&cfg); err != nil {
		http.Error(w, fmt.Sprintf("Invalid configuration: %v", err), http.StatusBadRequest)
		return
	}

	// Save to disk
	if err := ctx.ConfigLoader.SaveGlobal(&cfg); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save configuration: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Global configuration updated",
		"config":  cfg,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleGetScenarioConfig returns configuration for a specific scenario
// [REQ:SCS-CFG-002] Per-scenario overrides
func (ctx *Context) HandleGetScenarioConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	// Get effective config (merged global + scenario override)
	effectiveCfg, err := ctx.ConfigLoader.GetEffectiveConfig(scenarioName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load configuration: %v", err), http.StatusInternalServerError)
		return
	}

	// Get scenario-specific override
	override, err := ctx.ConfigLoader.LoadScenarioOverride(scenarioName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load override: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"scenario":  scenarioName,
		"effective": effectiveCfg,
		"override":  override,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleUpdateScenarioConfig updates configuration for a specific scenario
// [REQ:SCS-CFG-002] Per-scenario overrides
func (ctx *Context) HandleUpdateScenarioConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var override config.ScenarioOverride
	if err := json.Unmarshal(body, &override); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	override.Scenario = scenarioName

	// Validate if overrides are provided
	if override.Overrides != nil {
		if err := config.ValidateConfig(override.Overrides); err != nil {
			http.Error(w, fmt.Sprintf("Invalid configuration: %v", err), http.StatusBadRequest)
			return
		}
	}

	// Save override
	if err := ctx.ConfigLoader.SaveScenarioOverride(&override); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save configuration: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":  true,
		"message":  fmt.Sprintf("Configuration for %s updated", scenarioName),
		"override": override,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleDeleteScenarioConfig deletes configuration override for a specific scenario
// [REQ:SCS-CFG-002] Per-scenario overrides
func (ctx *Context) HandleDeleteScenarioConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	if err := ctx.ConfigLoader.DeleteScenarioOverride(scenarioName); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete configuration: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Configuration override for %s deleted", scenarioName),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleListPresets returns all available configuration presets
// [REQ:SCS-CFG-003] Configuration presets system
func (ctx *Context) HandleListPresets(w http.ResponseWriter, r *http.Request) {
	presets := config.ListPresetInfo()

	response := map[string]interface{}{
		"presets": presets,
		"total":   len(presets),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleApplyPreset applies a preset to global configuration
// [REQ:SCS-CFG-003] Configuration presets system
func (ctx *Context) HandleApplyPreset(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	presetName := vars["name"]

	preset := config.GetPreset(presetName)
	if preset == nil {
		http.Error(w, fmt.Sprintf("Unknown preset: %s", presetName), http.StatusNotFound)
		return
	}

	// Save preset config as global config
	if err := ctx.ConfigLoader.SaveGlobal(&preset.Config); err != nil {
		http.Error(w, fmt.Sprintf("Failed to apply preset: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Applied preset '%s'", presetName),
		"preset":  preset.Name,
		"config":  preset.Config,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
