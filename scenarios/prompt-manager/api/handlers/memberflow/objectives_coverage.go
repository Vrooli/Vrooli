package memberflow

import (
	"encoding/json"
	"net/http"

	objectivesdomain "prompt-manager/internal/objectives"
)

// serveObjectiveCoverage serves GET /objectives from the objective authority.
// It replaces the retired parser + team.json join: the authority owns current
// objective state, and the retained declarations appear only as declared-by
// context and a drift report.
func (h *connectHandler) serveObjectiveCoverage(w http.ResponseWriter, r *http.Request) {
	cov, err := objectivesdomain.BuildCoverage(r.Context(), h.objectives, h.repoRoot, h.storeDir)
	if err != nil {
		writeBridgeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(cov); err != nil {
		writeBridgeError(w, http.StatusInternalServerError, err.Error())
	}
}

func writeBridgeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
