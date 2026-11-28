package agentaccess

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *HTTPHandler) accessibilityTree(w http.ResponseWriter, r *http.Request) {
	response, err := h.Broker.AccessibilityTree(r.Context(), chi.URLParam(r, "sessionID"))
	writeResponse(w, response, err)
}
