package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/orchestrator/phases/isolation"
	"test-genie/internal/orchestrator/targetruntime"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
)

var applyPlaybooksSeed = phases.ApplyPlaybooksSeed

// resolveSeedTargetAPI reuses the same process-record/runtime URL resolver as
// Test Genie suite execution. Seed apply requires a live target API because it
// installs a routed storage lease on that service; it must not guess a URL or
// start a separately named shadow target.
var resolveSeedTargetAPI = func(ctx context.Context, scenarioName, scenarioDir string) (string, error) {
	urls := targetruntime.New(scenarioName, scenarioDir).LiveSurfaces(ctx)
	if strings.TrimSpace(urls.API) == "" {
		return "", fmt.Errorf("scenario %s has no live API URL", scenarioName)
	}
	return urls.API, nil
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
		Retain       bool   `json:"retain"`
		ScenarioPath string `json:"scenarioPath"`
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

	token, record, err := s.applySeedSession(r.Context(), name, strings.TrimSpace(payload.ScenarioPath), "", payload.Retain || isolation.ShouldRetainFromEnv())
	if err != nil {
		status := http.StatusInternalServerError
		if connect.CodeOf(err) == connect.CodeAlreadyExists {
			status = http.StatusConflict
		}
		if connect.CodeOf(err) == connect.CodeInvalidArgument {
			status = http.StatusBadRequest
		}
		if connect.CodeOf(err) == connect.CodeUnavailable {
			status = http.StatusServiceUnavailable
		}
		if connect.CodeOf(err) == connect.CodeResourceExhausted {
			status = http.StatusTooManyRequests
		}
		s.writeError(w, status, err.Error())
		return
	}
	session := record.Session

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

	session, err := s.clearSeedSession(r.Context(), name, token)
	if err != nil {
		status := http.StatusInternalServerError
		if connect.CodeOf(err) == connect.CodeNotFound {
			status = http.StatusNotFound
		}
		if connect.CodeOf(err) == connect.CodeFailedPrecondition {
			status = http.StatusConflict
		}
		s.writeError(w, status, err.Error())
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
	s.seedSessionsMu.Unlock()
	session, err := s.clearSeedSession(r.Context(), name, token)
	if err != nil {
		status := http.StatusInternalServerError
		if token == "" || connect.CodeOf(err) == connect.CodeNotFound {
			status = http.StatusNotFound
		}
		s.writeError(w, status, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"status":   "cleaned",
		"scenario": name,
		"run_id":   session.Session.RunID,
	})
}
