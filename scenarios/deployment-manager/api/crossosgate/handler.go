package crossosgate

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Handler exposes the cross-OS gate as an HTTP endpoint deployment-manager's
// promotion flow (or an operator) can call. A nil gate means bridge is not
// configured; the handler then responds 503 so the route is inert until
// VROOLI_BRIDGE_URL is wired.
type Handler struct {
	gate *Gate
}

// NewHandler builds a Handler over an explicit Gate (used in tests with a fake
// Bridge). Production builds it from config via NewHTTPHandler.
func NewHandler(gate *Gate) *Handler { return &Handler{gate: gate} }

// Evaluate handles POST /api/v1/cross-os-gate/evaluate: decode a Request, run
// the cross-OS gate, and write deployment-manager's owned production-readiness
// Verdict. ProductionReady=false (with the per-OS ledger) is a normal 200 result
// — a withheld promotion is an answer, not an error.
func (h *Handler) Evaluate(w http.ResponseWriter, r *http.Request) {
	if h.gate == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "cross-OS deployment gate is not configured; set VROOLI_BRIDGE_URL to the vrooli-bridge control plane to enable it",
		})
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body: " + err.Error()})
		return
	}
	if strings.TrimSpace(req.Scenario) == "" || strings.TrimSpace(req.Revision) == "" || len(req.TargetOSes) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "scenario, revision, and at least one target OS are required",
		})
		return
	}

	verdict, err := h.gate.Evaluate(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "cross-OS gate failed: " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, verdict)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
