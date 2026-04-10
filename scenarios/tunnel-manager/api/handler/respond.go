package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"tunnel-manager/domain"

	"github.com/gorilla/mux"
)

const (
	maxQueryHours = 720   // 30 days
	maxQueryLimit = 10000 // max rows for history endpoints
)

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError maps a domain error to the correct HTTP status and writes a structured response.
// Unknown errors become 500 internal_error.
func writeError(w http.ResponseWriter, err error) {
	var de *domain.DomainError
	if errors.As(err, &de) {
		writeJSONErrorWithCode(w, codeToStatus(de.Code), de.Message, de.Code)
		return
	}
	writeJSONErrorWithCode(w, http.StatusInternalServerError, "internal error", domain.ErrCodeInternal)
}

func codeToStatus(code string) int {
	switch code {
	case domain.ErrCodeValidation:
		return http.StatusBadRequest
	case domain.ErrCodeNotFound:
		return http.StatusNotFound
	case domain.ErrCodeConflict:
		return http.StatusConflict
	case domain.ErrCodeUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// writeJSONErrorWithCode writes a structured error response with an error code.
func writeJSONErrorWithCode(w http.ResponseWriter, status int, message, code string) {
	writeJSON(w, status, domain.ApiErrorResponse{
		Error:     message,
		ErrorCode: code,
	})
}

// decodeJSON reads and decodes a JSON request body into dest.
// Returns true on success. On failure, writes a 400 error and returns false.
func decodeJSON(w http.ResponseWriter, r *http.Request, dest any) bool {
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		writeJSONErrorWithCode(w, http.StatusBadRequest, "invalid JSON body", domain.ErrCodeValidation)
		return false
	}
	return true
}

// parsePathID extracts and validates the {id} path parameter.
// Returns the parsed ID, or writes a 400 error and returns 0, false.
func parsePathID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		writeJSONErrorWithCode(w, http.StatusBadRequest, "invalid route id", domain.ErrCodeValidation)
		return 0, false
	}
	return id, true
}
