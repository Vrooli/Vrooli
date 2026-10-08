package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	apierrors "github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/errors"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/persistence"
)

func (h *Handlers) GetCheckActions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	checkID := vars["checkId"]

	// Get the check and verify it's healable
	healable, ok := h.registry.GetHealableCheck(checkID)
	if !ok {
		apierrors.LogAndRespond(w, apierrors.NewNotFoundError("actions", "healable check", checkID))
		return
	}

	// Get last result to determine available actions
	lastResult, _ := h.registry.GetResult(checkID)
	var lastResultPtr *checks.Result
	if lastResult.CheckID != "" {
		lastResultPtr = &lastResult
	}

	var actions []checks.RecoveryAction
	if contextAware, ok := healable.(checks.ContextAwareHealableCheck); ok {
		actions = contextAware.RecoveryActionsWithContext(r.Context(), lastResultPtr)
	} else {
		actions = healable.RecoveryActions(lastResultPtr)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"checkId": checkID,
		"actions": actions,
	}); err != nil {
		apierrors.LogError("get_check_actions", "encode_response", err)
	}
}

// ExecuteCheckAction executes a recovery action for a check
// [REQ:HEAL-ACTION-001]
func (h *Handlers) ExecuteCheckAction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	checkID := vars["checkId"]
	actionID := vars["actionId"]

	// Get the check and verify it's healable
	_, ok := h.registry.GetHealableCheck(checkID)
	if !ok {
		apierrors.LogAndRespond(w, apierrors.NewNotFoundError("actions", "healable check", checkID))
		return
	}

	// Execute the action with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	result := h.registry.ExecuteAction(ctx, checkID, actionID)

	// Log the action to the database
	if err := h.store.SaveActionLog(
		ctx,
		result.CheckID,
		result.ActionID,
		result.Success,
		result.TimedOut,
		result.Message,
		result.Output,
		result.Error,
		result.Duration.Milliseconds(),
	); err != nil {
		apierrors.LogError("execute_action", "save_action_log", err)
	}

	// Return the result
	w.Header().Set("Content-Type", "application/json")
	statusCode := http.StatusOK
	if result.Refusal != nil {
		statusCode = http.StatusConflict
	} else if !result.Success {
		statusCode = http.StatusInternalServerError
	}
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		apierrors.LogError("execute_action", "encode_response", err)
	}
}

// GetActionHistory returns the action log history
// [REQ:HEAL-ACTION-001]
func (h *Handlers) GetActionHistory(w http.ResponseWriter, r *http.Request) {
	// Parse optional checkId filter from query
	checkID := r.URL.Query().Get("checkId")
	limit := 50

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	var logs *persistence.ActionLogsResponse
	var err error

	if checkID != "" {
		logs, err = h.store.GetActionLogsForCheck(ctx, checkID, limit)
	} else {
		logs, err = h.store.GetActionLogs(ctx, limit)
	}

	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("action_history", "retrieve action logs", err))
		return
	}

	// Return empty array instead of null
	if logs.Logs == nil {
		logs.Logs = []persistence.ActionLog{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(logs); err != nil {
		apierrors.LogError("action_history", "encode_response", err)
	}
}
