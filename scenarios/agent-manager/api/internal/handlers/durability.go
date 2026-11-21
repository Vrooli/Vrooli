package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// GetDurability returns the evidence-bounded durability projection for one
// run. The response states its epoch and lane coverage so an agent can act on
// the boundary instead of guessing what an empty result means.
func (h *Handler) GetDurability(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid run id")
		return
	}
	verdict, err := h.svc.Durability(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, verdict)
}
