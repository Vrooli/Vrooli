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

func (s *Server) handleGaps(w http.ResponseWriter, r *http.Request) {
	dashboards := map[string][]MetricEntry{}
	for _, room := range s.registry.Rooms {
		readings, _ := s.readings(r.Context(), s.registry.Dashboard(room.ID))
		for _, m := range readings {
			if m.Coverage != CoverageNow {
				dashboards[room.ID] = append(dashboards[room.ID], m)
			}
		}
	}
	if len(dashboards) == 0 {
		dashboards = s.registry.GapsByDashboard()
	}
	writeJSON(w, http.StatusOK, gapsResponse{
		GeneratedAt: time.Now().UTC(),
		Dashboards:  dashboards,
	})
}
