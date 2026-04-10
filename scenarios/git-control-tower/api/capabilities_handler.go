package main

import (
	"context"
	"net/http"
	"time"
)

func (s *Server) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	resp := NewResponse(w)
	resp.OK(map[string]any{
		"capabilities": s.capabilities.Resolve(ctx),
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	})
}
