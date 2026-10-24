package deployments

import (
	"net/http"

	"github.com/gorilla/mux"
)

// Handler owns the governance-plane deployment endpoints. Actual artifact
// production is delegated to a ramp; this surface refuses unimplemented
// generic dispatch instead of fabricating a deployment record.
type Handler struct {
	log func(string, map[string]interface{})
}

func NewHandler(log func(string, map[string]interface{})) *Handler { return &Handler{log: log} }

func (h *Handler) Deploy(w http.ResponseWriter, r *http.Request) {
	profileID := mux.Vars(r)["profile_id"]
	if validation := ValidateProfile(profileID); !validation.Valid {
		status, response := FormatValidationError(validation)
		writeJSON(w, status, response)
		return
	}
	writeJSON(w, http.StatusNotImplemented, map[string]interface{}{
		"error": "deployment dispatcher is not implemented", "reason": "dispatcher_not_implemented", "profile_id": profileID,
	})
}

func (h *Handler) Status(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusNotFound, map[string]interface{}{"error": "deployment not found", "reason": "deployment_not_found"})
}
