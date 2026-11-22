package frozenreplay

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

func (s *Service) Handler() http.Handler {
	router := http.NewServeMux()
	router.HandleFunc("POST /", s.handleCreate)
	router.HandleFunc("GET /{receiptID}", s.handleGet)
	return router
}

func (s *Service) handleCreate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request CreateRequest
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "request must contain a valid replay receipt")
		return
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid_request", "request must contain exactly one JSON value")
		return
	}
	result, err := s.Create(r.Context(), request)
	if err != nil {
		switch {
		case errors.Is(err, ErrIdempotencyKey):
			writeError(w, http.StatusConflict, "idempotency_conflict", "idempotency key is already bound to different replay evidence")
		case errors.Is(err, ErrInvalidReceipt):
			writeError(w, http.StatusUnprocessableEntity, "unverified_evidence", err.Error())
		default:
			writeError(w, http.StatusServiceUnavailable, "receipt_store_unavailable", "Test Genie could not persist the replay receipt")
		}
		return
	}
	status := http.StatusCreated
	if result.Replayed {
		status = http.StatusOK
	}
	writeJSON(w, status, result)
}

func (s *Service) handleGet(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("receiptID"))
	receipt, err := s.Get(r.Context(), id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "frozen replay receipt was not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "integrity_failure", "stored receipt is unreadable or failed its integrity check")
		return
	}
	writeJSON(w, http.StatusOK, receipt)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"code": code, "error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
