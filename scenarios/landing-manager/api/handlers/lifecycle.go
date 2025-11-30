package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"

	"landing-manager/errors"
	"landing-manager/util"
	"landing-manager/validation"
)

// HandleScenarioStart starts a generated scenario
func (h *Handler) HandleScenarioStart(w http.ResponseWriter, r *http.Request) {
	scenarioID := mux.Vars(r)["scenario_id"]
	if err := validation.ValidateScenarioID(scenarioID); err != nil {
		h.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	loc := util.ResolveScenarioPath(scenarioID)
	if !loc.Found {
		h.respondNotFound(w, scenarioID, "not found in staging or production")
		return
	}

	args := buildLifecycleArgs("start", scenarioID, loc)
	result := h.executor().Execute("vrooli", args...)

	if result.Err != nil {
		h.handleLifecycleError(w, "start", scenarioID, result.Output, result.Err)
		return
	}

	h.Log("scenario_started", map[string]interface{}{"scenario_id": scenarioID})
	h.respondLifecycleSuccess(w, scenarioID, "started", result.Output)
}

// HandleScenarioStop stops a generated scenario
func (h *Handler) HandleScenarioStop(w http.ResponseWriter, r *http.Request) {
	scenarioID := mux.Vars(r)["scenario_id"]
	if err := validation.ValidateScenarioID(scenarioID); err != nil {
		h.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	loc := util.ResolveScenarioPath(scenarioID)
	var args []string
	if loc.Found {
		args = buildLifecycleArgs("stop", scenarioID, loc)
	} else {
		args = []string{"scenario", "stop", scenarioID}
	}

	result := h.executor().Execute("vrooli", args...)

	if result.Err != nil {
		if util.IsScenarioNotFound(result.Output) || strings.Contains(result.Output, "Cannot find Vrooli utilities") {
			h.RespondJSON(w, http.StatusOK, map[string]interface{}{
				"success": true,
				"message": fmt.Sprintf("Scenario '%s' already stopped or not found", scenarioID),
				"output":  result.Output,
			})
			return
		}

		h.Log("scenario_stop_failed", map[string]interface{}{
			"scenario_id": scenarioID,
			"error":       result.Err.Error(),
			"output":      result.Output,
		})
		h.RespondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to stop scenario: %v", result.Err),
			"output":  result.Output,
		})
		return
	}

	h.Log("scenario_stopped", map[string]interface{}{"scenario_id": scenarioID})
	h.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"message":     "Scenario stopped successfully",
		"scenario_id": scenarioID,
		"output":      result.Output,
	})
}

// HandleScenarioRestart restarts a generated scenario
func (h *Handler) HandleScenarioRestart(w http.ResponseWriter, r *http.Request) {
	scenarioID := mux.Vars(r)["scenario_id"]
	if err := validation.ValidateScenarioID(scenarioID); err != nil {
		h.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	loc := util.ResolveScenarioPath(scenarioID)
	if !loc.Found {
		h.respondNotFound(w, scenarioID, "not found in staging or production")
		return
	}

	args := buildLifecycleArgs("restart", scenarioID, loc)
	result := h.executor().Execute("vrooli", args...)

	if result.Err != nil {
		h.handleLifecycleError(w, "restart", scenarioID, result.Output, result.Err)
		return
	}

	h.Log("scenario_restarted", map[string]interface{}{"scenario_id": scenarioID})
	h.respondLifecycleSuccess(w, scenarioID, "restarted", result.Output)
}

// HandleScenarioStatus returns the status of a generated scenario.
// Decision: Staging and production scenarios use different status detection methods:
//   - Staging: Check PID files directly (faster, doesn't require CLI)
//   - Production: Query vrooli CLI for full status (more detailed)
func (h *Handler) HandleScenarioStatus(w http.ResponseWriter, r *http.Request) {
	scenarioID := mux.Vars(r)["scenario_id"]
	if err := validation.ValidateScenarioID(scenarioID); err != nil {
		h.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	loc := util.ResolveScenarioPath(scenarioID)

	// Decision: Staging scenarios use direct PID-based status check
	if loc.IsStaging() {
		running, activeCount := h.checkStagingProcesses(scenarioID)
		statusText := buildStagingStatusText(scenarioID, running, activeCount)

		h.RespondJSON(w, http.StatusOK, map[string]interface{}{
			"success":     true,
			"scenario_id": scenarioID,
			"running":     running,
			"status_text": statusText,
			"location":    util.LocationStaging,
		})
		return
	}

	// Decision: Production scenarios use CLI status check
	if loc.IsProduction() {
		h.handleProductionStatus(w, scenarioID)
		return
	}

	// Scenario not found in either location
	h.RespondJSON(w, http.StatusNotFound, map[string]interface{}{
		"success": false,
		"message": fmt.Sprintf("Scenario '%s' not found", scenarioID),
	})
}

// checkStagingProcesses counts active processes for a staging scenario.
// Decision: A process is considered active if its PID responds to kill -0.
func (h *Handler) checkStagingProcesses(scenarioID string) (bool, int) {
	processDir := filepath.Join(os.Getenv("HOME"), ".vrooli", "processes", "scenarios", scenarioID)
	entries, err := os.ReadDir(processDir)
	if err != nil || len(entries) == 0 {
		return false, 0
	}

	activeCount := 0
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".pid" {
			pidFile := filepath.Join(processDir, entry.Name())
			if pidBytes, err := os.ReadFile(pidFile); err == nil {
				pidStr := strings.TrimSpace(string(pidBytes))
				// Decision: Use kill -0 to check if process is alive without signaling it
				checkResult := h.executor().Execute("kill", "-0", pidStr)
				if checkResult.Err == nil {
					activeCount++
				}
			}
		}
	}

	return activeCount > 0, activeCount
}

// buildStagingStatusText constructs a human-readable status for staging scenarios.
func buildStagingStatusText(scenarioID string, running bool, activeCount int) string {
	if running {
		return fmt.Sprintf("Scenario '%s' is running in staging area (%d active process(es))", scenarioID, activeCount)
	}
	return fmt.Sprintf("Scenario '%s' is in staging area (generated/)", scenarioID)
}

// handleProductionStatus handles status check for production scenarios via CLI.
func (h *Handler) handleProductionStatus(w http.ResponseWriter, scenarioID string) {
	result := h.executor().Execute("vrooli", "scenario", "status", scenarioID)

	if result.Err != nil {
		if util.IsScenarioNotFound(result.Output) {
			h.RespondJSON(w, http.StatusNotFound, map[string]interface{}{
				"success": false,
				"message": fmt.Sprintf("Scenario '%s' not found", scenarioID),
			})
			return
		}

		h.Log("scenario_status_failed", map[string]interface{}{
			"scenario_id": scenarioID,
			"error":       result.Err.Error(),
		})
		h.RespondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to get scenario status: %v", result.Err),
		})
		return
	}

	running := isScenarioRunning(result.Output)

	h.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"scenario_id": scenarioID,
		"running":     running,
		"status_text": result.Output,
		"location":    util.LocationProduction,
	})
}

// isScenarioRunning determines if a scenario is running based on CLI output.
// Decision: A scenario is running if the output contains the running indicator.
func isScenarioRunning(output string) bool {
	return strings.Contains(output, "🟢 RUNNING") || strings.Contains(output, "Status:        🟢 RUNNING")
}

// HandleScenarioLogs returns logs from a generated scenario
func (h *Handler) HandleScenarioLogs(w http.ResponseWriter, r *http.Request) {
	scenarioID := mux.Vars(r)["scenario_id"]
	if err := validation.ValidateScenarioID(scenarioID); err != nil {
		h.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	tail, _ := validation.ValidateTailParam(r.URL.Query().Get("tail"))
	loc := util.ResolveScenarioPath(scenarioID)
	if !loc.Found {
		h.respondNotFound(w, scenarioID, "not found")
		return
	}

	args := buildLifecycleArgs("logs", scenarioID, loc, "--tail", tail)
	result := h.executor().Execute("vrooli", args...)

	if result.Err != nil {
		if util.IsScenarioNotFound(result.Output) {
			h.respondNotFound(w, scenarioID, "not found")
			return
		}

		h.Log("scenario_logs_failed", map[string]interface{}{
			"scenario_id": scenarioID,
			"error":       result.Err.Error(),
		})
		h.RespondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Failed to get scenario logs: %v", result.Err),
		})
		return
	}

	h.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"scenario_id": scenarioID,
		"logs":        result.Output,
	})
}

// HandleScenarioPromote moves a scenario from staging to production.
// Decision: Only staging scenarios can be promoted. Production scenarios already exist.
func (h *Handler) HandleScenarioPromote(w http.ResponseWriter, r *http.Request) {
	scenarioID := mux.Vars(r)["scenario_id"]
	if err := validation.ValidateScenarioID(scenarioID); err != nil {
		h.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	stagingPath := util.StagingPath(scenarioID)
	productionPath := util.ProductionPath(scenarioID)

	// Decision: Scenario must exist in staging to be promoted
	if _, err := os.Stat(stagingPath); os.IsNotExist(err) {
		appErr := errors.NewNotFoundError("scenario", scenarioID)
		appErr.Details = "Not found in staging area (generated/)"
		appErr.Suggestion = "Generate the scenario first, or check if it was already promoted"
		h.RespondAppError(w, appErr)
		return
	}

	// Decision: Cannot overwrite existing production scenario
	if _, err := os.Stat(productionPath); err == nil {
		appErr := errors.NewConflictError(scenarioID, fmt.Sprintf("Scenario '%s' already exists in production", scenarioID))
		appErr.Suggestion = "Delete or rename the existing production scenario first, then try promoting again"
		h.RespondAppError(w, appErr)
		return
	}

	// Decision: Stop scenario before moving to prevent file lock issues
	stopResult := h.executor().Execute("vrooli", "scenario", "stop", scenarioID)
	if stopResult.Err != nil {
		h.Log("promote_stop_warning", map[string]interface{}{
			"scenario_id": scenarioID,
			"output":      stopResult.Output,
			"error":       stopResult.Err.Error(),
		})
	}

	// Move directory from staging to production
	if err := os.Rename(stagingPath, productionPath); err != nil {
		appErr := errors.NewFileSystemError("move to production", stagingPath, err)
		appErr.Suggestion = "Check file permissions and ensure the target directory is writable"
		h.RespondAppError(w, appErr)
		return
	}

	h.Log("scenario_promoted", map[string]interface{}{
		"scenario_id": scenarioID,
		"from":        stagingPath,
		"to":          productionPath,
	})

	h.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success":         true,
		"message":         "Scenario promoted to production successfully",
		"scenario_id":     scenarioID,
		"production_path": productionPath,
	})
}

// HandleScenarioDelete removes a scenario from staging.
// Decision: Only staging scenarios can be deleted via API. Production requires manual deletion
// for safety (prevents accidental deletion of promoted, possibly live scenarios).
func (h *Handler) HandleScenarioDelete(w http.ResponseWriter, r *http.Request) {
	scenarioID := mux.Vars(r)["scenario_id"]
	if err := validation.ValidateScenarioID(scenarioID); err != nil {
		h.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	stagingPath := util.StagingPath(scenarioID)

	// Decision: Only allow deletion of staging scenarios for safety
	if _, err := os.Stat(stagingPath); os.IsNotExist(err) {
		appErr := errors.NewNotFoundError("scenario", scenarioID)
		appErr.Details = "Not found in staging area (generated/)"
		appErr.Suggestion = "Only scenarios in the staging area can be deleted through this endpoint. Production scenarios require manual deletion."
		h.RespondAppError(w, appErr)
		return
	}

	// Decision: Stop scenario first to release file locks and clean up processes
	stopResult := h.executor().Execute("vrooli", "scenario", "stop", scenarioID, "--path", stagingPath)
	if stopResult.Err != nil {
		h.Log("delete_stop_warning", map[string]interface{}{
			"scenario_id": scenarioID,
			"output":      stopResult.Output,
			"error":       stopResult.Err.Error(),
		})
	}

	// Remove directory
	if err := os.RemoveAll(stagingPath); err != nil {
		appErr := errors.NewFileSystemError("delete scenario", stagingPath, err)
		appErr.Suggestion = "Check file permissions. If the scenario is running, stop it first."
		h.RespondAppError(w, appErr)
		return
	}

	h.Log("scenario_deleted", map[string]interface{}{
		"scenario_id": scenarioID,
		"path":        stagingPath,
	})

	h.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"message":     "Scenario deleted successfully",
		"scenario_id": scenarioID,
	})
}

// Helper functions

func (h *Handler) respondNotFound(w http.ResponseWriter, scenarioID, reason string) {
	appErr := errors.NewNotFoundError("scenario", scenarioID)
	appErr.Details = reason
	appErr.Suggestion = "Check that the scenario exists in staging (generated/) or production (/scenarios/)"
	h.RespondAppError(w, appErr)
}

func (h *Handler) handleLifecycleError(w http.ResponseWriter, operation, scenarioID, output string, err error) {
	if util.IsScenarioNotFound(output) {
		h.respondNotFound(w, scenarioID, "not found")
		return
	}

	// Create structured lifecycle error with helpful context
	appErr := errors.NewLifecycleError(operation, scenarioID, err)
	appErr.Details = util.SanitizeCommandOutput(output)

	// Provide operation-specific recovery suggestions
	switch operation {
	case "start":
		if strings.Contains(output, "port") || strings.Contains(output, "EADDRINUSE") {
			appErr.Suggestion = "Another process may be using the required port. Try stopping any conflicting services."
		} else if strings.Contains(output, "database") || strings.Contains(output, "postgres") {
			appErr.Suggestion = "Check that PostgreSQL is running: vrooli resource status postgres"
		} else {
			appErr.Suggestion = "Check the scenario configuration and logs for more details."
		}
	case "restart":
		appErr.Suggestion = "Try stopping the scenario first, then start it again."
	case "logs":
		appErr.Suggestion = "The scenario may not have started yet. Check the status first."
	}

	h.RespondAppError(w, appErr)
}

func (h *Handler) respondLifecycleSuccess(w http.ResponseWriter, scenarioID, operation, output string) {
	h.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"success":     true,
		"message":     fmt.Sprintf("Scenario %s successfully", operation),
		"scenario_id": scenarioID,
		"output":      util.SanitizeCommandOutput(output),
	})
}

// buildLifecycleArgs constructs the argument list for vrooli lifecycle commands.
// Decision: Staging scenarios need --path argument; production scenarios don't.
// This is a helper that returns args instead of an exec.Cmd, allowing the caller
// to use the CommandExecutor seam.
func buildLifecycleArgs(operation, scenarioID string, loc util.ScenarioLocation, extraArgs ...string) []string {
	args := []string{"scenario", operation, scenarioID}
	// Decision: Use the RequiresPathArg() method from ScenarioLocation
	if loc.RequiresPathArg() {
		args = append(args, "--path", loc.Path)
	}
	args = append(args, extraArgs...)
	return args
}
