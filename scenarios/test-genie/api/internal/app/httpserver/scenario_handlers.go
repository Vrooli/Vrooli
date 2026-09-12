package httpserver

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	"test-genie/internal/shared"

	"github.com/gorilla/mux"
)

func (s *Server) handleListScenarios(w http.ResponseWriter, r *http.Request) {
	if s.scenarios == nil {
		s.writeError(w, http.StatusInternalServerError, "scenario directory service unavailable")
		return
	}
	summaries, err := s.scenarios.ListSummaries(r.Context())
	if err != nil {
		s.log("listing scenarios failed", map[string]interface{}{"error": err.Error()})
		s.writeError(w, http.StatusInternalServerError, "failed to load scenarios")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": summaries,
		"count": len(summaries),
	})
}

func (s *Server) handleGetScenario(w http.ResponseWriter, r *http.Request) {
	if s.scenarios == nil {
		s.writeError(w, http.StatusInternalServerError, "scenario directory service unavailable")
		return
	}
	params := mux.Vars(r)
	name := strings.TrimSpace(params["name"])
	if name == "" {
		s.writeError(w, http.StatusBadRequest, "scenario name is required")
		return
	}

	summary, err := s.scenarios.GetSummary(r.Context(), name)
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeError(w, http.StatusNotFound, "scenario not found")
			return
		}
		s.log("fetching scenario failed", map[string]interface{}{"error": err.Error()})
		s.writeError(w, http.StatusInternalServerError, "failed to load scenario")
		return
	}

	s.writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handleRunScenarioTests(w http.ResponseWriter, r *http.Request) {
	if s.scenarios == nil {
		s.writeError(w, http.StatusInternalServerError, "scenario directory service unavailable")
		return
	}
	params := mux.Vars(r)
	name := strings.TrimSpace(params["name"])
	if name == "" {
		s.writeError(w, http.StatusBadRequest, "scenario name is required")
		return
	}
	var payload struct {
		Type      string   `json:"type"`
		Paths     []string `json:"paths"`
		Playbooks []string `json:"playbooks"`
		Filter    string   `json:"filter"`
		// ScenarioPath overrides scenario directory resolution. Set by the CLI
		// when running inside a sandboxed agent. See cliutil/sandbox.go.
		ScenarioPath string `json:"scenarioPath"`
	}
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
			s.writeError(w, http.StatusBadRequest, "invalid JSON payload")
			return
		}
	}

	extraArgs := make([]string, 0, len(payload.Paths)+len(payload.Playbooks))
	for _, p := range payload.Paths {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			extraArgs = append(extraArgs, "--path", trimmed)
		}
		if len(extraArgs) >= 20 { // prevent unbounded arg growth
			break
		}
	}
	for _, pb := range payload.Playbooks {
		trimmed := strings.TrimSpace(pb)
		if trimmed != "" {
			extraArgs = append(extraArgs, "--playbook", trimmed)
		}
		if len(extraArgs) >= 40 {
			break
		}
	}
	if filter := strings.TrimSpace(payload.Filter); filter != "" {
		extraArgs = append(extraArgs, "--filter", filter)
	}

	cmd, result, err := s.scenarios.RunScenarioTests(r.Context(), name, payload.Type, extraArgs, strings.TrimSpace(payload.ScenarioPath))
	if err != nil {
		switch {
		case errors.Is(err, os.ErrNotExist):
			s.writeError(w, http.StatusNotFound, "scenario not found")
		case shared.IsValidationError(err):
			s.writeError(w, http.StatusBadRequest, err.Error())
		default:
			s.log("scenario test execution failed", map[string]interface{}{
				"error":    err.Error(),
				"scenario": name,
			})
			s.writeError(w, http.StatusInternalServerError, "scenario tests failed")
		}
		return
	}

	response := map[string]interface{}{
		"status":  "completed",
		"command": cmd,
		"type":    cmd.Type,
	}
	if result != nil && strings.TrimSpace(result.LogPath) != "" {
		response["logPath"] = result.LogPath
	}
	if result != nil {
		response["skipSummary"] = result.SkipSummary
	}
	s.writeJSON(w, http.StatusOK, response)
}
