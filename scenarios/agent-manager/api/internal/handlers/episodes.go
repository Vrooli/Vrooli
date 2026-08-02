package handlers

import (
	"net/http"

	"agent-manager/internal/runsignal"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func (h *Handler) GetEpisodesHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid UUID format for run ID")
		return
	}
	if _, err := h.svc.GetRun(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}
	episodes, err := h.svc.Episodes(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, struct {
		ClassifierVersion string                      `json:"classifierVersion"`
		Episodes          []runsignal.FrictionEpisode `json:"episodes"`
	}{runsignal.EpisodeClassifierVersion, episodes})
}
