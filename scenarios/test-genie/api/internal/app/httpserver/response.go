package httpserver

import (
	"encoding/json"
	"net/http"
)

func (s *Server) writeJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	// Security headers (defense-in-depth; also applied by the router's
	// securityHeadersMiddleware) plus the permissive CORS origin used API-wide.
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Strict-Transport-Security", "max-age=31536000")
	w.Header().Set("Access-Control-Allow-Origin", "*")
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
