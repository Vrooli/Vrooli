package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"knowledge-observatory/internal/services/dochealth"
)

func respondWithError(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func respondDocHealthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, dochealth.ErrScenarioNameInvalid):
		respondWithError(w, http.StatusBadRequest, err)
	case errors.Is(err, dochealth.ErrScenarioNotFound):
		respondWithError(w, http.StatusNotFound, err)
	case errors.Is(err, dochealth.ErrScenarioRootInvalid):
		respondWithError(w, http.StatusServiceUnavailable, err)
	default:
		respondWithError(w, http.StatusInternalServerError, err)
	}
}
