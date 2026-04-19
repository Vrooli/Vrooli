package main

import (
	"net/http"
	"time"
)

type gapsResponse struct {
	GeneratedAt time.Time                `json:"generated_at"`
	Dashboards  map[string][]MetricEntry `json:"dashboards"`
}

func (s *Server) registerGapRoutes() {
	s.router.HandleFunc("/api/v1/gaps", s.handleGaps).Methods("GET")
}

func (s *Server) handleGaps(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, gapsResponse{
		GeneratedAt: time.Now().UTC(),
		Dashboards:  s.registry.GapsByDashboard(),
	})
}
