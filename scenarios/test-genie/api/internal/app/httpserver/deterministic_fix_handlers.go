package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"test-genie/internal/deterministicfix"

	"github.com/gorilla/mux"
)

// deterministicFixRequest is the body for the deterministic (provider-driven)
// fix path. Unlike the agent-based path it is dry-run by default; callers must
// set apply=true to write changes.
type deterministicFixRequest struct {
	Apply     bool     `json:"apply,omitempty"`
	RuleIDs   []string `json:"ruleIds,omitempty"`
	Providers []string `json:"providers,omitempty"`
}

// handleDeterministicFix aggregates each delegated provider's shared Fix RPC for
// the target scenario and returns a unified candidate report.
// POST /api/v1/scenarios/{name}/fix/deterministic
func (s *Server) handleDeterministicFix(w http.ResponseWriter, r *http.Request) {
	scenarioName := mux.Vars(r)["name"]
	if scenarioName == "" {
		s.writeError(w, http.StatusBadRequest, "scenario name is required")
		return
	}

	var req deterministicFixRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
			s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
			return
		}
	}

	runner := deterministicfix.NewRunner()
	if len(req.Providers) > 0 {
		runner.Providers = req.Providers
	}
	report := runner.Run(r.Context(), scenarioName, req.Apply, req.RuleIDs)
	s.writeJSON(w, http.StatusOK, report)
}
