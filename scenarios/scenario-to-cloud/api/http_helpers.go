package main

import (
	"encoding/json"
	"net/http"
)

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

type APIErrorEnvelope struct {
	Error APIError `json:"error"`
}

func writeAPIError(w http.ResponseWriter, status int, apiErr APIError) {
	writeJSON(w, status, APIErrorEnvelope{Error: apiErr})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
