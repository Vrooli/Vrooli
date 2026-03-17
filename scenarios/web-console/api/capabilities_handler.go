package main

import (
	"context"
	"log"
	"net/http"
	"time"
)

func (s *Server) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	caps := s.capabilities.Resolve(ctx)
	log.Printf("capabilities: full resolve took %dms", time.Since(start).Milliseconds())
	writeJSON(w, http.StatusOK, map[string]any{
		"capabilities": caps,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	})
}

// handleCapabilitiesLiveness returns capability status using fast liveness-only
// checks (GET health endpoints only, no test transcription). Returns cached
// full-check results when fresh.
func (s *Server) handleCapabilitiesLiveness(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	caps := s.capabilities.ResolveLiveness(ctx)
	log.Printf("capabilities: liveness resolve took %dms", time.Since(start).Milliseconds())
	writeJSON(w, http.StatusOK, map[string]any{
		"capabilities": caps,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	})
}
