package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"scenario-completeness-scoring/pkg/config"
	apierrors "scenario-completeness-scoring/pkg/errors"

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
// [REQ:SCS-CORE-003] Structured error responses with actionable guidance
func (ctx *Context) HandleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeValidationFailed,
			"Failed to read request body",
			apierrors.CategoryValidation,
		).WithDetails(err.Error()).WithNextSteps(
			"Ensure the request body is valid",
			"Check that Content-Type is application/json",
		), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var cfg config.ScoringConfig
	if err := json.Unmarshal(body, &cfg); err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeValidationFailed,
			"Invalid JSON in request body",
			apierrors.CategoryValidation,
		).WithDetails(err.Error()).WithNextSteps(
			"Verify the JSON syntax is correct",
			"Use a JSON validator to check the payload",
		), http.StatusBadRequest)
		return
	}

	// Validate configuration
	if err := config.ValidateConfig(&cfg); err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeConfigInvalid,
			"Invalid configuration values",
			apierrors.CategoryConfig,
		).WithDetails(err.Error()).WithNextSteps(
			"Check that weight values are between 0 and 100",
			"Ensure at least one scoring component is enabled",
			"Verify weights sum to 100",
		), http.StatusBadRequest)
		return
	}

	// Save to disk
	if err := ctx.ConfigLoader.SaveGlobal(&cfg); err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeInternalError,
			"Failed to save configuration",
			apierrors.CategoryFileSystem,
		).WithDetails(err.Error()).AsRecoverable().WithNextSteps(
			"Check disk space availability",
			"Verify write permissions to config directory",
			"Try again in a few moments",
		), http.StatusInternalServerError)
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
// [REQ:SCS-CORE-003] Structured error responses with actionable guidance
func (ctx *Context) HandleGetScenarioConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	// Validate scenario name
	// ASSUMPTION: Scenario names are user-controlled input
	// HARDENED: Explicit validation prevents path traversal and injection
	if errMsg := ValidateScenarioName(scenarioName); errMsg != "" {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeValidationFailed,
			"Invalid scenario name",
			apierrors.CategoryValidation,
		).WithDetails(errMsg).WithNextSteps(
			"Scenario names must start with a letter or number",
			"Use only letters, numbers, hyphens, and underscores",
			"Maximum length is 64 characters",
		), http.StatusBadRequest)
		return
	}

	// Get effective config (merged global + scenario override)
	effectiveCfg, err := ctx.ConfigLoader.GetEffectiveConfig(scenarioName)
	if err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeInternalError,
			fmt.Sprintf("Failed to load configuration for scenario '%s'", scenarioName),
			apierrors.CategoryConfig,
		).WithDetails(err.Error()).AsRecoverable().WithNextSteps(
			"Check that global configuration exists",
			"Verify the scenario name is correct",
			"Try reloading the page",
		), http.StatusInternalServerError)
		return
	}

	// Get scenario-specific override
	override, err := ctx.ConfigLoader.LoadScenarioOverride(scenarioName)
	if err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeInternalError,
			fmt.Sprintf("Failed to load override for scenario '%s'", scenarioName),
			apierrors.CategoryConfig,
		).WithDetails(err.Error()).AsRecoverable().WithNextSteps(
			"The scenario may not have a custom override yet",
			"Try creating an override first via PUT /config/scenarios/{scenario}",
		), http.StatusInternalServerError)
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
// [REQ:SCS-CORE-003] Structured error responses with actionable guidance
func (ctx *Context) HandleUpdateScenarioConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	// Validate scenario name
	// ASSUMPTION: Scenario names are user-controlled input
	// HARDENED: Explicit validation prevents path traversal and injection
	if errMsg := ValidateScenarioName(scenarioName); errMsg != "" {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeValidationFailed,
			"Invalid scenario name",
			apierrors.CategoryValidation,
		).WithDetails(errMsg).WithNextSteps(
			"Scenario names must start with a letter or number",
			"Use only letters, numbers, hyphens, and underscores",
			"Maximum length is 64 characters",
		), http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeValidationFailed,
			"Failed to read request body",
			apierrors.CategoryValidation,
		).WithDetails(err.Error()), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var override config.ScenarioOverride
	if err := json.Unmarshal(body, &override); err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeValidationFailed,
			"Invalid JSON in request body",
			apierrors.CategoryValidation,
		).WithDetails(err.Error()).WithNextSteps(
			"Verify the JSON syntax is correct",
			"Example: {\"enabled\": true, \"preset\": \"skip-e2e\"}",
		), http.StatusBadRequest)
		return
	}

	override.Scenario = scenarioName

	// Validate if overrides are provided
	if override.Overrides != nil {
		if err := config.ValidateConfig(override.Overrides); err != nil {
			writeAPIError(w, apierrors.NewAPIError(
				apierrors.ErrCodeConfigInvalid,
				fmt.Sprintf("Invalid configuration overrides for scenario '%s'", scenarioName),
				apierrors.CategoryConfig,
			).WithDetails(err.Error()).WithNextSteps(
				"Check that weight values are between 0 and 100",
				"Ensure at least one scoring component is enabled",
			), http.StatusBadRequest)
			return
		}
	}

	// Save override
	if err := ctx.ConfigLoader.SaveScenarioOverride(&override); err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeInternalError,
			fmt.Sprintf("Failed to save configuration for scenario '%s'", scenarioName),
			apierrors.CategoryFileSystem,
		).WithDetails(err.Error()).AsRecoverable().WithNextSteps(
			"Check disk space availability",
			"Verify write permissions",
			"Try again in a few moments",
		), http.StatusInternalServerError)
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
// [REQ:SCS-CORE-003] Structured error responses with actionable guidance
func (ctx *Context) HandleDeleteScenarioConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["scenario"]

	// Validate scenario name
	// ASSUMPTION: Scenario names are user-controlled input
	// HARDENED: Explicit validation prevents path traversal and injection
	if errMsg := ValidateScenarioName(scenarioName); errMsg != "" {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeValidationFailed,
			"Invalid scenario name",
			apierrors.CategoryValidation,
		).WithDetails(errMsg).WithNextSteps(
			"Scenario names must start with a letter or number",
			"Use only letters, numbers, hyphens, and underscores",
			"Maximum length is 64 characters",
		), http.StatusBadRequest)
		return
	}

	if err := ctx.ConfigLoader.DeleteScenarioOverride(scenarioName); err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeInternalError,
			fmt.Sprintf("Failed to delete configuration for scenario '%s'", scenarioName),
			apierrors.CategoryFileSystem,
		).WithDetails(err.Error()).AsRecoverable().WithNextSteps(
			"The scenario may not have an existing override",
			"Check file system permissions",
			"Try again in a few moments",
		), http.StatusInternalServerError)
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
// [REQ:SCS-CORE-003] Structured error responses with actionable guidance
func (ctx *Context) HandleApplyPreset(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	presetName := vars["name"]

	preset := config.GetPreset(presetName)
	if preset == nil {
		availablePresets := config.ListPresetInfo()
		presetNames := make([]string, len(availablePresets))
		for i, p := range availablePresets {
			presetNames[i] = p.Name
		}
		writeAPIError(w, apierrors.NewAPIError(
			"PRESET_NOT_FOUND",
			fmt.Sprintf("Unknown preset: '%s'", presetName),
			apierrors.CategoryValidation,
		).WithNextSteps(
			fmt.Sprintf("Available presets: %v", presetNames),
			"Use GET /api/v1/config/presets to see all available presets",
		), http.StatusNotFound)
		return
	}

	// Save preset config as global config
	if err := ctx.ConfigLoader.SaveGlobal(&preset.Config); err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeInternalError,
			fmt.Sprintf("Failed to apply preset '%s'", presetName),
			apierrors.CategoryFileSystem,
		).WithDetails(err.Error()).AsRecoverable().WithNextSteps(
			"Check disk space availability",
			"Verify write permissions",
			"Try again in a few moments",
		), http.StatusInternalServerError)
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
