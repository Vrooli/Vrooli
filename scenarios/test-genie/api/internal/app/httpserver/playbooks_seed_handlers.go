package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/playbooks/isolation"
)

var applyPlaybooksSeed = phases.ApplyPlaybooksSeed

type seedSession struct {
	Scenario string
	Created  time.Time
	Session  *phases.PlaybooksSeedSession
}

func (s *Server) handlePlaybooksSeedApply(w http.ResponseWriter, r *http.Request) {
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
		Retain bool `json:"retain"`
	}
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
			s.writeError(w, http.StatusBadRequest, "invalid JSON payload")
			return
		}
	}

	if root := strings.TrimSpace(s.scenarios.ScenarioRoot()); root == "" {
		s.writeError(w, http.StatusInternalServerError, "scenarios root is not configured")
		return
	}

	s.seedSessionsMu.Lock()
	if token, ok := s.seedSessionsByScenario[name]; ok && token != "" {
		s.seedSessionsMu.Unlock()
		s.writeError(w, http.StatusConflict, "seed session already active for scenario")
		return
	}
	s.seedSessionsMu.Unlock()

	scenariosRoot := strings.TrimSpace(s.scenarios.ScenarioRoot())
	scenarioDir := filepath.Join(scenariosRoot, name)
	ws, err := workspace.New(scenariosRoot, name)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	env := ws.Environment()
	env.ScenarioDir = scenarioDir

	retain := payload.Retain || isolation.ShouldRetainFromEnv()

	session, err := applyPlaybooksSeed(r.Context(), env, nil, retain)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	token := uuid.NewString()
	session.CleanupRef = token

	s.seedSessionsMu.Lock()
	s.seedSessions[token] = &seedSession{
		Scenario: name,
		Created:  time.Now().UTC(),
		Session:  session,
	}
	s.seedSessionsByScenario[name] = token
	s.seedSessionsMu.Unlock()

	s.writeJSON(w, http.StatusOK, map[string]any{
		"status":        "seeded",
		"scenario":      name,
		"run_id":        session.RunID,
		"seed_state":    session.SeedState,
		"cleanup_token": token,
		"resources":     session.Resources,
	})
}

func (s *Server) handlePlaybooksSeedCleanup(w http.ResponseWriter, r *http.Request) {
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
		CleanupToken string `json:"cleanup_token"`
	}
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
			s.writeError(w, http.StatusBadRequest, "invalid JSON payload")
			return
		}
	}
	token := strings.TrimSpace(payload.CleanupToken)
	if token == "" {
		s.writeError(w, http.StatusBadRequest, "cleanup_token is required")
		return
	}

	s.seedSessionsMu.Lock()
	session, ok := s.seedSessions[token]
	if ok && session != nil && session.Scenario != name {
		s.seedSessionsMu.Unlock()
		s.writeError(w, http.StatusConflict, "cleanup token does not match scenario")
		return
	}
	delete(s.seedSessions, token)
	delete(s.seedSessionsByScenario, name)
	s.seedSessionsMu.Unlock()

	if !ok || session == nil || session.Session == nil {
		s.writeError(w, http.StatusNotFound, "seed session not found")
		return
	}

	if err := session.Session.Cleanup(r.Context()); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"status":   "cleaned",
		"scenario": name,
		"run_id":   session.Session.RunID,
	})
}

func (s *Server) handlePlaybooksSeedCleanupForce(w http.ResponseWriter, r *http.Request) {
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
	if strings.ToLower(strings.TrimSpace(r.URL.Query().Get("force"))) != "true" {
		s.writeError(w, http.StatusBadRequest, "force=true is required to cleanup without a token")
		return
	}

	s.seedSessionsMu.Lock()
	token := s.seedSessionsByScenario[name]
	session, ok := s.seedSessions[token]
	delete(s.seedSessions, token)
	delete(s.seedSessionsByScenario, name)
	s.seedSessionsMu.Unlock()

	if !ok || session == nil || session.Session == nil {
		s.writeError(w, http.StatusNotFound, "seed session not found")
		return
	}

	if err := session.Session.Cleanup(r.Context()); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"status":   "cleaned",
		"scenario": name,
		"run_id":   session.Session.RunID,
	})
}
