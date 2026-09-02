package main

import (
	"net/http"
	"strings"

	monetization "github.com/vrooli/vrooli/packages/monetization-go"
)

// journeyHandler is the Web Console observation surface consumed by the
// shared JourneyProbe. It is intentionally loopback-only and reports routing
// observations, never credentials or lease material.
func (s *Server) journeyHandler(w http.ResponseWriter, r *http.Request) {
	if !webConsoleSameOrigin(w, r) {
		return
	}
	operation := monetization.JourneyOperation(strings.TrimSpace(r.URL.Query().Get("operation")))
	if operation == "" {
		http.Error(w, "operation is required", http.StatusBadRequest)
		return
	}
	writeCredentialJSON(w, http.StatusOK, map[string]string{
		"operation":  string(operation),
		"observed":   "web-console-route-observed",
		"route":      "/api/v1/internal/monetization/journey",
		"app_key":    "web-console",
		"bundle_key": "business_suite",
	})
}
