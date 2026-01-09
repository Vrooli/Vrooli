package handlers

import (
	"net/http"

	"github.com/gorilla/mux"

	"landing-manager/validation"
)

// HandlePersonaList returns all available personas
func (h *Handler) HandlePersonaList(w http.ResponseWriter, r *http.Request) {
	personas, err := h.PersonaService.GetPersonas()
	if err != nil {
		h.Log("failed to list personas", map[string]interface{}{"error": err.Error()})
		http.Error(w, "Failed to list personas", http.StatusInternalServerError)
		return
	}

	h.RespondJSON(w, http.StatusOK, personas)
}

// HandlePersonaShow returns a specific persona by ID
func (h *Handler) HandlePersonaShow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := validation.ValidatePersonaID(id); err != nil {
		h.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	persona, err := h.PersonaService.GetPersona(id)
	if err != nil {
		h.Log("failed to get persona", map[string]interface{}{"id": id, "error": err.Error()})
		h.RespondError(w, http.StatusNotFound, "Persona not found")
		return
	}

	h.RespondJSON(w, http.StatusOK, persona)
}
