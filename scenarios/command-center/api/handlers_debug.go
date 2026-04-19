package main

import (
	"encoding/json"
	"net/http"
)

func (s *Server) registerDebugRoutes() {
	s.router.HandleFunc("/api/v1/debug/r3f-stats", s.handleR3FStats).Methods("GET", "POST")
}

func (s *Server) handleR3FStats(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"events": s.stats.Snapshot(),
		})
	case http.MethodPost:
		var evt R3FEvent
		if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body",
				"expected R3F event JSON", map[string]string{"details": err.Error()})
			return
		}
		s.stats.Append(evt)
		writeJSON(w, http.StatusAccepted, map[string]any{"accepted": true})
	}
}
