package httpserver

import (
	"encoding/json"
	"net/http"

	"test-genie/internal/agents"
)

func (s *Server) writeJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		s.log("failed to encode JSON response", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

func (s *Server) writeError(w http.ResponseWriter, statusCode int, message string) {
	s.writeJSON(w, statusCode, map[string]string{"error": message})
}

// writeFailure writes a structured FailureInfo response.
// It logs the failure with internal details and writes a user-safe response.
// This is the preferred method for returning errors from handlers.
func (s *Server) writeFailure(w http.ResponseWriter, failure agents.FailureInfo) {
	// Log the full failure details including internal information
	s.log("request failed", failure.ToLogFields())

	// Write user-safe response (excludes internal details)
	statusCode := failure.ToHTTPStatus()
	response := failure.ToMap()
	response["error"] = failure.Message // Keep backward compatibility with "error" field
	s.writeJSON(w, statusCode, response)
}

// writeFailureWithContext writes a structured failure with additional context for logging.
// Use this when you have extra context that should be logged but not returned to the user.
func (s *Server) writeFailureWithContext(w http.ResponseWriter, failure agents.FailureInfo, logContext map[string]interface{}) {
	// Merge failure fields with additional context
	fields := failure.ToLogFields()
	for k, v := range logContext {
		fields[k] = v
	}
	s.log("request failed", fields)

	// Write user-safe response
	statusCode := failure.ToHTTPStatus()
	response := failure.ToMap()
	response["error"] = failure.Message
	s.writeJSON(w, statusCode, response)
}
