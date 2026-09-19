package httpserver

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"test-genie/internal/playbooksclaims"

	"github.com/gorilla/mux"
)

// claimDTO is the JSON projection of a playbooks claim. Field names mirror
// the busy-response shape documented in the concurrency-guard plan §9.
type claimDTO struct {
	ScenarioName string    `json:"scenario_name"`
	RunID        string    `json:"run_id"`
	Mode         string    `json:"mode"`
	StartedBy    string    `json:"started_by"`
	AcquiredAt   time.Time `json:"acquired_at"`
	HeartbeatAt  time.Time `json:"heartbeat_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	Alive        bool      `json:"alive"`
}

func toClaimDTO(c playbooksclaims.Claim, now time.Time) claimDTO {
	return claimDTO{
		ScenarioName: c.ScenarioName,
		RunID:        c.RunID,
		Mode:         string(c.Mode),
		StartedBy:    c.StartedBy,
		AcquiredAt:   c.AcquiredAt,
		HeartbeatAt:  c.HeartbeatAt,
		ExpiresAt:    c.ExpiresAt,
		Alive:        c.Alive(now),
	}
}

func (s *Server) handleListPlaybooksClaims(w http.ResponseWriter, r *http.Request) {
	if s.playbooksClaims == nil {
		s.writeError(w, http.StatusInternalServerError, "playbooks claims service unavailable")
		return
	}
	claims, err := s.playbooksClaims.List(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	now := s.playbooksClaims.Now()
	out := make([]claimDTO, 0, len(claims))
	for _, c := range claims {
		out = append(out, toClaimDTO(c, now))
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"claims": out})
}

func (s *Server) handleGetPlaybooksClaim(w http.ResponseWriter, r *http.Request) {
	if s.playbooksClaims == nil {
		s.writeError(w, http.StatusInternalServerError, "playbooks claims service unavailable")
		return
	}
	name := strings.TrimSpace(mux.Vars(r)["scenario"])
	if name == "" {
		s.writeError(w, http.StatusBadRequest, "scenario name is required")
		return
	}
	claim, err := s.playbooksClaims.Get(r.Context(), name)
	if err != nil {
		if errors.Is(err, playbooksclaims.ErrNotFound) {
			s.writeJSON(w, http.StatusOK, map[string]any{"claim": nil})
			return
		}
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	now := s.playbooksClaims.Now()
	dto := toClaimDTO(claim, now)
	s.writeJSON(w, http.StatusOK, map[string]any{"claim": dto})
}

func (s *Server) handleReleasePlaybooksClaim(w http.ResponseWriter, r *http.Request) {
	if s.playbooksClaims == nil {
		s.writeError(w, http.StatusInternalServerError, "playbooks claims service unavailable")
		return
	}
	name := strings.TrimSpace(mux.Vars(r)["scenario"])
	if name == "" {
		s.writeError(w, http.StatusBadRequest, "scenario name is required")
		return
	}
	actor := strings.TrimSpace(r.Header.Get("X-Vrooli-Actor"))
	if actor == "" {
		actor = "anonymous"
	}
	broken, err := s.playbooksClaims.ForceBreak(r.Context(), name)
	if err != nil {
		if errors.Is(err, playbooksclaims.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "no active claim for scenario")
			return
		}
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	now := s.playbooksClaims.Now()
	age := now.Sub(broken.HeartbeatAt)
	s.log("playbooks claim force-released", map[string]interface{}{
		"actor":           actor,
		"scenario":        name,
		"broken_run_id":   broken.RunID,
		"heartbeat_age_s": age.Seconds(),
	})
	s.writeJSON(w, http.StatusOK, map[string]any{"released": toClaimDTO(broken, now)})
}
