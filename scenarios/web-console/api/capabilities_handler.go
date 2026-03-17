package main

import (
	"context"
	"net/http"
	"time"
)

func (s *Server) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	writeJSON(w, http.StatusOK, map[string]any{
		"capabilities": s.capabilities.Resolve(ctx),
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	})
}

// handleCapabilitiesLiveness returns capability status using fast liveness-only
// checks (GET health endpoints only, no test transcription). Returns cached
// full-check results when fresh.
func (s *Server) handleCapabilitiesLiveness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	writeJSON(w, http.StatusOK, map[string]any{
		"capabilities": s.capabilities.ResolveLiveness(ctx),
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	})
}
