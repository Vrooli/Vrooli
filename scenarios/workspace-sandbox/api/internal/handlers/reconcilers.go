package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
)

// ListReconcilers returns the names + last-run metrics of every
// reconciler registered in the Runner. Useful for the UI / operator
// CLI to know which names POST /admin/reconcilers/{name} accepts.
func (h *Handlers) ListReconcilers(w http.ResponseWriter, r *http.Request) {
	if h.Reconcilers == nil {
		h.JSONError(w, "reconcilers not configured", http.StatusServiceUnavailable)
		return
	}
	h.JSONSuccess(w, map[string]any{
		"names":   h.Reconcilers.Names(),
		"metrics": h.Reconcilers.Metrics(),
	})
}

// RunReconciler fires a single named reconciler synchronously and
// returns its ReconcileReport. The default Runner registers:
// lifecycle, heal, orphan, daemon-reaper, and (when configured)
// manual-review-expiry.
func (h *Handlers) RunReconciler(w http.ResponseWriter, r *http.Request) {
	if h.Reconcilers == nil {
		h.JSONError(w, "reconcilers not configured", http.StatusServiceUnavailable)
		return
	}
	name := mux.Vars(r)["name"]
	if name == "" {
		h.JSONError(w, "reconciler name required", http.StatusBadRequest)
		return
	}
	report, err := h.Reconcilers.RunOne(r.Context(), name)
	if err != nil {
		h.JSONError(w, err.Error(), http.StatusNotFound)
		return
	}
	h.JSONSuccess(w, map[string]any{
		"name":           name,
		"itemsProcessed": report.ItemsProcessed,
		"itemsFailed":    report.ItemsFailed,
		"durationMs":     report.Duration.Milliseconds(),
		"lastError":      report.LastError,
		"details":        report.Details,
	})
}
