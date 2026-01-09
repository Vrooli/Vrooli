package server

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// LoggingMiddleware logs HTTP requests in structured format.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		LogStructured("request", map[string]interface{}{
			"method":   r.Method,
			"path":     r.RequestURI,
			"duration": time.Since(start).String(),
		})
	})
}

// LogStructured outputs logs in a structured JSON-like format for better observability.
func LogStructured(msg string, fields map[string]interface{}) {
	if len(fields) == 0 {
		log.Printf(`{"level":"info","message":"%s"}`, msg)
		return
	}
	fieldsJSON, _ := json.Marshal(fields)
	log.Printf(`{"level":"info","message":"%s","fields":%s}`, msg, string(fieldsJSON))
}
