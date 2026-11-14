package control

import (
	"encoding/json"
	"net/http"

	internal "device-control/internal/control"
	internalflows "device-control/internal/flows"
	"github.com/gorilla/mux"
)

func (h *handler) listAgents(w http.ResponseWriter, _ *http.Request) {
	write(w, http.StatusOK, map[string]any{"agents": h.service.ListAgents()})
}

func (h *handler) startAgent(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Goal                    string `json:"goal"`
		DeviceID                string `json:"device_id"`
		Actor                   string `json:"actor"`
		SkillAvailable          bool   `json:"skill_available"`
		DryRun                  bool   `json:"dry_run"`
		Confirmed               bool   `json:"confirmed"`
		ExpectSelected          string `json:"expect_selected"`
		ExpectForegroundPackage string `json:"expect_foreground_package"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	a, err := h.service.StartAgentWithAssertions(r.Context(), in.Goal, in.DeviceID, in.Actor, in.SkillAvailable, in.DryRun, internalflows.DefaultAgentPolicy(), in.Confirmed, internal.AgentOutcomeAssertions{SelectedControl: in.ExpectSelected, ForegroundPackage: in.ExpectForegroundPackage})
	if err != nil {
		writeError(w, http.StatusPreconditionFailed, "agent_unavailable", err.Error())
		return
	}
	write(w, http.StatusAccepted, map[string]any{"agent": a})
}

func (h *handler) abortAgent(w http.ResponseWriter, r *http.Request) {
	a, err := h.service.AbortAgent(mux.Vars(r)["id"], "operator requested abort")
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"agent": a})
}

func (h *handler) promoteAgent(w http.ResponseWriter, r *http.Request) {
	a, err := h.service.PromoteAgent(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusConflict, "promotion_refused", err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"agent": a})
}
