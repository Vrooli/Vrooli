package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/internal/httpjson"
)

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	return httpjson.Decode(w, r, dst)
}

func decodeJSONBodyAllowEmpty(w http.ResponseWriter, r *http.Request, dst any) error {
	return httpjson.DecodeAllowEmpty(w, r, dst)
}

// parseUUIDParam extracts a UUID from the URL parameter with the given name.
// If parsing fails, it writes the provided error to the response and returns false.
// On success, it returns the parsed UUID and true.
func (h *Handler) parseUUIDParam(w http.ResponseWriter, r *http.Request, paramName string, errResponse *APIError) (uuid.UUID, bool) {
	idStr := chi.URLParam(r, paramName)
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, errResponse)
		return uuid.UUID{}, false
	}
	return id, true
}
