package handlers

import (
	"net/http"
	"strings"
	"time"

	"agent-manager/internal/toolfriction"
)

const runtimeFrictionMaxWindow = 30 * 24 * time.Hour

// RuntimeFriction serves a bounded, read-only aggregate of observed run time.
// Imported transcript runs are included by the orchestration source.
func (h *Handler) RuntimeFriction(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	until := now
	since := now.Add(-7 * 24 * time.Hour)
	query := r.URL.Query()
	if raw := strings.TrimSpace(query.Get("until")); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeSimpleError(w, r, "until", "until must be RFC 3339")
			return
		}
		until = parsed.UTC()
	}
	if raw := strings.TrimSpace(query.Get("since")); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeSimpleError(w, r, "since", "since must be RFC 3339")
			return
		}
		since = parsed.UTC()
	} else {
		since = until.Add(-7 * 24 * time.Hour)
	}
	if !until.After(since) || until.After(now) || until.Sub(since) > runtimeFrictionMaxWindow {
		writeSimpleError(w, r, "window", "since and until must define a past window no longer than 30 days")
		return
	}
	window := toolfriction.RuntimeWindow{Since: since, Until: until}
	if h.svc.RuntimeFrictionService == nil {
		writeJSON(w, http.StatusServiceUnavailable, unavailableRuntimeFriction(window, "runtime friction source is not configured"))
		return
	}
	report, err := h.svc.RuntimeFrictionReport(r.Context(), since, until)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, unavailableRuntimeFriction(window, "runtime friction data is unavailable"))
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func unavailableRuntimeFriction(window toolfriction.RuntimeWindow, reason string) toolfriction.RuntimeReport {
	return toolfriction.RuntimeReport{
		Window: window,
		Coverage: toolfriction.RuntimeCoverage{
			State:                "unavailable",
			Reason:               reason,
			ImportedRunsIncluded: true,
		},
		Categories: []toolfriction.RuntimeLoss{
			{Category: toolfriction.RuntimeStoppedByCheck},
			{Category: toolfriction.RuntimeWaitingOnValidation},
			{Category: toolfriction.RuntimeAgentManagerFailure},
			{Category: toolfriction.RuntimeRework},
			{Category: toolfriction.RuntimeSandboxEnvironment},
			{Category: toolfriction.RuntimeBuilding},
		},
	}
}
