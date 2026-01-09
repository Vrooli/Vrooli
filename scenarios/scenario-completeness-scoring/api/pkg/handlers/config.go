package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	"scenario-completeness-scoring/pkg/config"
	apierrors "scenario-completeness-scoring/pkg/errors"
)

// HandleGetConfig returns the current global scoring configuration.
// [REQ:SCS-CFG-001] Component toggle API
func (ctx *Context) HandleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := ctx.ConfigLoader.LoadGlobal()
	if err != nil {
		defaultCfg := config.DefaultConfig()
		cfg = &defaultCfg
	}

	response := map[string]interface{}{
		"config":            cfg,
		"effective_weights": cfg.Weights.Normalize(cfg.Components),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// HandleGetConfigSchema returns a UI-friendly schema that explains the config.
func (ctx *Context) HandleGetConfigSchema(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"schema": config.Schema(),
	})
}

// HandleResetConfig resets the global configuration to defaults.
func (ctx *Context) HandleResetConfig(w http.ResponseWriter, r *http.Request) {
	cfg := config.DefaultConfig()
	if err := ctx.ConfigLoader.SaveGlobal(&cfg); err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeInternalError,
			"Failed to reset configuration",
			apierrors.CategoryFileSystem,
		).WithDetails(err.Error()).AsRecoverable().WithNextSteps(
			"Verify write permissions to config directory",
			"Try again in a few moments",
		), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Global configuration reset to defaults",
		"config":  cfg,
	})
}

// HandleUpdateConfig updates the global scoring configuration.
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

	if err := config.ValidateConfig(&cfg); err != nil {
		writeAPIError(w, apierrors.NewAPIError(
			apierrors.ErrCodeConfigInvalid,
			"Invalid configuration values",
			apierrors.CategoryConfig,
		).WithDetails(err.Error()).WithNextSteps(
			"Verify weights sum to 100",
			"Ensure at least one scoring dimension is enabled",
			"Ensure each enabled dimension has at least one sub-metric enabled",
		), http.StatusBadRequest)
		return
	}

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

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":          true,
		"message":          "Global configuration updated",
		"config":           cfg,
		"effective_weights": cfg.Weights.Normalize(cfg.Components),
	})
}

