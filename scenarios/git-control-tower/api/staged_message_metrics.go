package main

import (
	"encoding/json"
	"net/http"
	"time"
)

func (s *Server) handleStagedMessageMetrics(w http.ResponseWriter, r *http.Request) {
	if s.stagedMessages == nil {
		http.Error(w, "staged-message metrics unavailable", http.StatusServiceUnavailable)
		return
	}
	metrics, err := s.stagedMessages.Metrics(r.Context(), time.Now(), 5*time.Minute)
	if err != nil {
		http.Error(w, "failed to read staged-message metrics", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		s.logf("ERROR: staged-message metrics response failed: %v", err)
	}
}
