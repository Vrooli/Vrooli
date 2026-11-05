package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/system"
	apierrors "github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/errors"
)

const stormDecisionLimit = 50

// StormStatus reports the storm authority's state: the scopes it froze and
// has not seen thawed, and its decision rows from the runtime registry.
// [REQ:STORM-002]
func (h *Handlers) StormStatus(w http.ResponseWriter, r *http.Request) {
	response := map[string]any{"contained": []system.StormOutcome{}, "decisions": []any{}, "mode": ""}
	if check, ok := h.registry.GetCheck(system.EmergencyWatchdogReportCheckID); ok {
		if report, ok := check.(*system.EmergencyWatchdogReportCheck); ok {
			response["contained"] = report.ContainedScopes()
			if last, ok := h.registry.GetResult(system.EmergencyWatchdogReportCheckID); ok {
				response["actions"] = report.RecoveryActions(&last)
			}
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if home, err := os.UserHomeDir(); err == nil {
		decisions, err := system.ListStormDecisions(ctx, home, stormDecisionLimit)
		if err != nil {
			response["decisions_error"] = err.Error()
		} else {
			response["decisions"] = decisions
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		apierrors.LogError("storm", "encode_response", err)
	}
}
