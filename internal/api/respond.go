package api

import (
	"encoding/json"
	"net/http"

	"github.com/vrooli/vrooli/internal/vroolierr"
)

type Response struct {
	Success   bool        `json:"success"`
	Error     string      `json:"error,omitempty"`
	ErrorCode string      `json:"error_code,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondSuccess(w http.ResponseWriter, status int, data any) {
	respondJSON(w, status, Response{Success: true, Data: data})
}

func respondError(w http.ResponseWriter, err error) {
	if err == nil {
		respondSuccess(w, http.StatusOK, nil)
		return
	}

	boundaryErr, _ := vroolierr.Ensure(
		err,
		"internal_error",
		"Internal",
		"",
		vroolierr.WithHTTPStatus(http.StatusInternalServerError),
	)
	respondJSON(w, vroolierr.HTTPStatus(boundaryErr, http.StatusInternalServerError), Response{
		Success:   false,
		Error:     boundaryErr.Error(),
		ErrorCode: boundaryErr.Code,
	})
}

func newAPIError(status int, code, message string, err error) error {
	return vroolierr.Wrap(err, code, "Runtime", message, vroolierr.WithHTTPStatus(status))
}
