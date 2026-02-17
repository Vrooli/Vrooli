package httputil

import (
	"encoding/json"
	"log"
	"net/http"
)

func JSON(w http.ResponseWriter, data any) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(data)
}

func JSONWithStatus(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

func Error(w http.ResponseWriter, logPrefix, message string, code int) {
	if logPrefix != "" {
		log.Printf("%s: %s (status=%d)", logPrefix, message, code)
	}
	http.Error(w, message, code)
}

func BadRequest(w http.ResponseWriter, logPrefix, message string) {
	Error(w, logPrefix, message, http.StatusBadRequest)
}

func NotFound(w http.ResponseWriter, logPrefix, message string) {
	Error(w, logPrefix, message, http.StatusNotFound)
}

func InternalError(w http.ResponseWriter, logPrefix, message string) {
	Error(w, logPrefix, message, http.StatusInternalServerError)
}

func Conflict(w http.ResponseWriter, logPrefix, message string) {
	Error(w, logPrefix, message, http.StatusConflict)
}

func ServiceUnavailable(w http.ResponseWriter, logPrefix, message string) {
	Error(w, logPrefix, message, http.StatusServiceUnavailable)
}
