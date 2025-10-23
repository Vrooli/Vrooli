package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
)

type ErrorResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, payload interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	if status > 0 {
		w.WriteHeader(status)
	}
	return json.NewEncoder(w).Encode(payload)
}

func WriteError(w http.ResponseWriter, status int, message string, details ...interface{}) {
	resp := ErrorResponse{Success: false, Message: message}
	if len(details) > 0 {
		resp.Details = details[0]
	}
	_ = WriteJSON(w, status, resp)
}

// WriteDecodeError normalizes request-decoding failures into API responses.
func WriteDecodeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrRequestTooLarge):
		WriteError(w, http.StatusRequestEntityTooLarge, "Request body exceeds allowed size")
	case errors.Is(err, ErrUnknownFields):
		WriteError(w, http.StatusBadRequest, "Payload contains unknown fields")
	case errors.Is(err, ErrUnexpectedPayload):
		WriteError(w, http.StatusBadRequest, "Unexpected data after JSON payload")
	case errors.Is(err, ErrInvalidJSON):
		WriteError(w, http.StatusBadRequest, "Invalid JSON payload")
	default:
		WriteError(w, http.StatusBadRequest, "Unable to decode request body")
	}
}
